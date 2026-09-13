#!/usr/bin/env bash
# azure_bootstrap_ci.sh — one-time Azure bootstrap for the GitHub Actions pipeline.
#
# Run in Azure Cloud Shell (bash) from a clone of this repository, signed in as
# an Owner of the subscription who can also assign Entra directory roles:
#
#   git clone -b develop https://github.com/aegis-imaging/aegis.git && cd aegis
#   ./scripts/azure_bootstrap_ci.sh                  # bootstrap (safe to re-run)
#   ./scripts/azure_bootstrap_ci.sh --post-apply     # after Terraform Azure + Deploy to Azure ran
#   ./scripts/azure_bootstrap_ci.sh --bind-domains   # after the DNS records resolve and the
#                                                    # domain tfvars were applied
#
# Bootstrap:
#   1. Reports whether the subscription is fresh or already carries an AEGIS deployment.
#   2. Creates the terraform state resource group, storage account and container
#      named in terraform/azure/backend.tf if they are missing.
#   3. Runs scripts/azure_bootstrap_github_oidc.sh (app registration, federated
#      credentials, role assignments).
#   4. Writes terraform/azure/terraform.tfvars with the minimal-footprint profile
#      and prints the GitHub secrets and variables to set.
# Post-apply:
#   Prints the Cloudflare records for the API and admin hostnames.
# Bind-domains:
#   Requests the free managed certificates for both hostnames.
#
# Overrides: LOCATION, API_HOST, ADMIN_HOST, ADMIN_EMAIL, PROJECT_NAME, ENVIRONMENT.
set -euo pipefail

LOCATION="${LOCATION:-eastus2}"
API_HOST="${API_HOST:-azure.api.aegisimaging.ai}"
ADMIN_HOST="${ADMIN_HOST:-azure.admin.aegisimaging.ai}"
ADMIN_EMAIL="${ADMIN_EMAIL:-ops@aegisimaging.ai}"
PROJECT_NAME="${PROJECT_NAME:-aegis}"
ENVIRONMENT="${ENVIRONMENT:-prod}"
TF_DIR="terraform/azure"
MODE="${1:-bootstrap}"

cd "$(dirname "$0")/.."
command -v az >/dev/null 2>&1 || { echo "error: az CLI required (az login first)" >&2; exit 1; }
command -v terraform >/dev/null 2>&1 || { echo "error: terraform required (preinstalled in Azure Cloud Shell)" >&2; exit 1; }

STATE_RG="$(awk -F'"' '/resource_group_name/{print $2; exit}' "$TF_DIR/backend.tf")"
STATE_SA="$(awk -F'"' '/storage_account_name/{print $2; exit}' "$TF_DIR/backend.tf")"
STATE_CONTAINER="$(awk -F'"' '/container_name/{print $2; exit}' "$TF_DIR/backend.tf")"
STATE_KEY="$(awk -F'"' '/^[[:space:]]+key[[:space:]]+=/{print $2; exit}' "$TF_DIR/backend.tf")"
RG="$PROJECT_NAME-$ENVIRONMENT"
ACR="$(echo "${PROJECT_NAME}${ENVIRONMENT}acr" | tr -d '-')"

SUB="$(az account show --query id -o tsv)"
TENANT="$(az account show --query tenantId -o tsv)"
echo "==> subscription $SUB, tenant $TENANT"

tf() { terraform -chdir="$TF_DIR" "$@"; }

# ── post-apply / bind-domains ────────────────────────────────────────────────
if [ "$MODE" = "--post-apply" ]; then
  tf init -input=false >/dev/null
  cat <<EOF

Cloudflare DNS records (DNS only, grey cloud):
  CNAME  ${API_HOST%%.aegisimaging.ai}          ->  $(tf output -raw api_fqdn)
  TXT    asuid.${API_HOST%%.aegisimaging.ai}    ->  $(tf output -raw custom_domain_verification_id)
  CNAME  ${ADMIN_HOST%%.aegisimaging.ai}        ->  $(tf output -raw admin_dashboard_fqdn)
  TXT    asuid.${ADMIN_HOST%%.aegisimaging.ai}  ->  $(tf output -raw custom_domain_verification_id)

Then add these two lines to the AZURE_TERRAFORM_TFVARS secret, run Terraform Azure with APPLY,
approve azure-prod, and come back with:  ./scripts/azure_bootstrap_ci.sh --bind-domains
  api_domain   = "$API_HOST"
  admin_domain = "$ADMIN_HOST"
EOF
  exit 0
fi
if [ "$MODE" = "--bind-domains" ]; then
  tf init -input=false >/dev/null
  ENV_NAME="$(tf output -raw container_app_environment_name)"
  for pair in "$RG-api:$API_HOST" "$RG-admin-dashboard:$ADMIN_HOST"; do
    app="${pair%%:*}"; host="${pair##*:}"
    echo "==> binding $host to $app with a managed certificate"
    az containerapp hostname bind -g "$RG" -n "$app" --hostname "$host" --environment "$ENV_NAME" --validation-method CNAME -o none
  done
  echo "==> certificates take 5–15 minutes; then:  curl -s https://$API_HOST/healthz"
  exit 0
fi

# ── 1. discovery ─────────────────────────────────────────────────────────────
have_rg="$(az group exists -n "$RG")"
have_state=false
if az storage account show -n "$STATE_SA" -g "$STATE_RG" >/dev/null 2>&1; then
  have_state="$(az storage blob exists --account-name "$STATE_SA" --container-name "$STATE_CONTAINER" \
                  --name "$STATE_KEY" --auth-mode key --query exists -o tsv 2>/dev/null || echo false)"
fi
if [ "$have_state" = true ]; then
  echo "==> existing terraform state found: the CI apply will trim the deployment in place"
elif [ "$have_rg" = true ]; then
  echo "error: resource group $RG exists but there is no state blob." >&2
  echo "       Recover the state (storage account $STATE_SA, blob versions) before continuing." >&2
  exit 1
else
  echo "==> fresh subscription: no AEGIS state or resource group found"
fi

# ── 2. state backend ─────────────────────────────────────────────────────────
az group create -n "$STATE_RG" -l "$LOCATION" -o none
if ! az storage account show -n "$STATE_SA" -g "$STATE_RG" >/dev/null 2>&1; then
  if [ "$(az storage account check-name -n "$STATE_SA" --query nameAvailable -o tsv)" != "true" ]; then
    echo "error: storage account name $STATE_SA is taken outside this subscription." >&2
    echo "       Change storage_account_name in $TF_DIR/backend.tf to a new unique name and re-run." >&2
    exit 1
  fi
  az storage account create -n "$STATE_SA" -g "$STATE_RG" -l "$LOCATION" --sku Standard_LRS \
    --kind StorageV2 --allow-blob-public-access false --min-tls-version TLS1_2 -o none
  echo "==> created state storage account $STATE_SA"
fi
az storage container create -n "$STATE_CONTAINER" --account-name "$STATE_SA" --auth-mode key -o none
echo "==> terraform backend ready ($STATE_SA/$STATE_CONTAINER)"

# ── 3. GitHub identity ───────────────────────────────────────────────────────
./scripts/azure_bootstrap_github_oidc.sh --subscription="$SUB" | grep -vE '^(GitHub|  AZURE_|  \(see|Environment:)' || true
APP_ID="$(az ad app list --display-name aegis-github-actions --query '[0].appId' -o tsv)"

# ── 4. tfvars ────────────────────────────────────────────────────────────────
if [ "$have_state" = true ]; then
  DB_PASSWORD="REPLACE_WITH_THE_EXISTING_db_admin_password"
  echo "==> state exists: keep db_admin_password from the current tfvars secret (or reset it with"
  echo "    az postgres flexible-server update --admin-password) and put it in the file below"
else
  DB_PASSWORD="$(openssl rand -hex 14)Zq1!"
fi
TFVARS="$TF_DIR/terraform.tfvars"
cat > "$TFVARS" <<EOF
azure_region       = "$LOCATION"
environment        = "$ENVIRONMENT"
project_name       = "$PROJECT_NAME"
azure_ad_tenant_id = "$TENANT"

# Custom hostnames are added after the first apply (--post-apply prints the DNS records).
api_domain          = ""
admin_domain        = ""
admin_dashboard_url = "https://$ADMIN_HOST"
landing_base_url    = "https://aegisimaging.ai"

db_admin_username = "aegis"
db_admin_password = "$DB_PASSWORD"
db_sku_name       = "B_Standard_B1ms"
db_storage_mb     = 32768

alert_email       = "$ADMIN_EMAIL"
first_admin_email = "$ADMIN_EMAIL"
contact_email     = "contact@aegisimaging.ai"

# Minimal footprint (docs/runbooks/minimal-footprint.md)
enable_sidecars      = false
enable_dwv           = false
enable_mcp_server    = false
enable_upload_portal = false
api_cpu              = 0.5
api_memory           = "1Gi"
dimse_receiver_image = ""
EOF
echo "==> wrote $TFVARS"

cat <<EOF

================ GitHub → Settings → Secrets and variables → Actions ================

Secrets:
  AZURE_CLIENT_ID        = $APP_ID
  AZURE_TENANT_ID        = $TENANT
  AZURE_SUBSCRIPTION_ID  = $SUB
  AZURE_TERRAFORM_TFVARS = the whole file printed below

Variables:
  AZURE_CI_ENABLED       = true
  AZURE_BUILD_SERVICES   = api admin-dashboard
  AZURE_RESOURCE_GROUP   = $RG
  AZURE_ACR_LOGIN_SERVER = $ACR.azurecr.io
  AZURE_API_URL          = https://$API_HOST
  AZURE_ADMIN_URL        = https://$ADMIN_HOST
  AZURE_PLAN_ARCHIVE     = https://$STATE_SA.blob.core.windows.net/$STATE_CONTAINER/plans

Environment: azure-prod with yourself as required reviewer (Settings → Environments).

Then: Actions → Terraform Azure → Run workflow (main, confirm_apply = APPLY), approve azure-prod;
      Actions → Deploy to Azure → Run workflow (main);
      back here: ./scripts/azure_bootstrap_ci.sh --post-apply

---------------- AZURE_TERRAFORM_TFVARS (copy everything between the lines) ----------------
$(cat "$TFVARS")
--------------------------------------------------------------------------------------------
EOF
