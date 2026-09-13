#!/usr/bin/env bash
# One-time bootstrap: the Entra ID identity GitHub Actions uses to reach Azure
# via OIDC (no client secrets). Creates an app registration + service
# principal, federated credentials for this repository, and the role
# assignments terraform/azure needs. Safe to re-run.
#
# Usage: ./scripts/azure_bootstrap_github_oidc.sh [--repo=owner/name] [--app-name=<name>] [--subscription=<id>]
set -euo pipefail

REPO="${REPO:-aegis-imaging/aegis}"
APP_NAME="${APP_NAME:-aegis-github-actions}"
SUBSCRIPTION="${SUBSCRIPTION:-}"

for arg in "$@"; do
  case "$arg" in
    --repo=*) REPO="${arg#*=}" ;;
    --app-name=*) APP_NAME="${arg#*=}" ;;
    --subscription=*) SUBSCRIPTION="${arg#*=}" ;;
    -h|--help)
      sed -n '2,8p' "$0"
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

command -v az >/dev/null 2>&1 || { echo "error: az CLI required (az login first)" >&2; exit 1; }
[ -n "$SUBSCRIPTION" ] || SUBSCRIPTION="$(az account show --query id -o tsv)"
TENANT="$(az account show --query tenantId -o tsv)"

APP_ID="$(az ad app list --display-name "$APP_NAME" --query '[0].appId' -o tsv)"
if [ -z "$APP_ID" ]; then
  APP_ID="$(az ad app create --display-name "$APP_NAME" --query appId -o tsv)"
  echo "==> created app registration $APP_NAME ($APP_ID)"
else
  echo "==> using existing app registration $APP_NAME ($APP_ID)"
fi
APP_OBJECT_ID="$(az ad app show --id "$APP_ID" --query id -o tsv)"

if ! az ad sp show --id "$APP_ID" >/dev/null 2>&1; then
  az ad sp create --id "$APP_ID" >/dev/null
  echo "==> created service principal"
fi
SP_OBJECT_ID="$(az ad sp show --id "$APP_ID" --query id -o tsv)"

federated_credential() {
  local name="$1" subject="$2"
  if [ -n "$(az ad app federated-credential list --id "$APP_OBJECT_ID" --query "[?name=='$name'].name" -o tsv)" ]; then
    echo "    federated credential $name already exists"
    return
  fi
  az ad app federated-credential create --id "$APP_OBJECT_ID" --parameters "{
    \"name\": \"$name\",
    \"issuer\": \"https://token.actions.githubusercontent.com\",
    \"subject\": \"$subject\",
    \"audiences\": [\"api://AzureADTokenExchange\"]
  }" >/dev/null
  echo "    created federated credential $name -> $subject"
}
echo "==> federated credentials for $REPO"
federated_credential github-main "repo:${REPO}:ref:refs/heads/main"
federated_credential github-develop "repo:${REPO}:ref:refs/heads/develop"
federated_credential github-pull-request "repo:${REPO}:pull_request"
federated_credential github-env-azure-prod "repo:${REPO}:environment:azure-prod"

# Contributor manages the resources; User Access Administrator is needed
# because terraform/azure assigns roles to the Container Apps managed identity.
echo "==> role assignments on /subscriptions/$SUBSCRIPTION"
for role in "Contributor" "User Access Administrator"; do
  if az role assignment create --assignee-object-id "$SP_OBJECT_ID" --assignee-principal-type ServicePrincipal \
       --role "$role" --scope "/subscriptions/${SUBSCRIPTION}" >/dev/null 2>&1; then
    echo "    $role: assigned"
  else
    echo "    $role: already assigned (or you lack permission to assign it)"
  fi
done

# terraform/azure creates the Easy Auth app registration (auth.tf), so the
# identity must be allowed to manage applications in the tenant.
if az rest --method POST \
     --uri "https://graph.microsoft.com/v1.0/directoryRoles/roleTemplateId=9b895d92-2cd3-44c7-9d02-a6ac2d5ea5c3/members/\$ref" \
     --body "{\"@odata.id\":\"https://graph.microsoft.com/v1.0/directoryObjects/${SP_OBJECT_ID}\"}" >/dev/null 2>&1; then
  echo "==> added to the Application Administrator directory role"
else
  echo "==> Application Administrator role: already a member, or your account cannot assign directory roles."
  echo "    Without it, terraform apply fails on azuread_application.admin_easyauth — add the role in Entra admin center."
fi

cat <<EOF

GitHub repository secrets (Settings -> Secrets and variables -> Actions):
  AZURE_CLIENT_ID        = $APP_ID
  AZURE_TENANT_ID        = $TENANT
  AZURE_SUBSCRIPTION_ID  = $SUBSCRIPTION
  AZURE_TERRAFORM_TFVARS = <full contents of terraform/azure/terraform.tfvars>

GitHub repository variables:
  AZURE_CI_ENABLED       = true
  AZURE_RESOURCE_GROUP, AZURE_ACR_LOGIN_SERVER, AZURE_API_URL, AZURE_ADMIN_URL
  (see docs/runbooks/ci-cd.md for the full list)

GitHub environment: create "azure-prod" with yourself as required reviewer.
EOF
