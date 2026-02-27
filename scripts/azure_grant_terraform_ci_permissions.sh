#!/usr/bin/env bash
set -euo pipefail

SUBSCRIPTION_ID=""
TENANT_ID=""
CLIENT_ID=""
KEYVAULT_NAME=""
DRY_RUN=0

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/azure_grant_terraform_ci_permissions.sh \
    --subscription-id <SUBSCRIPTION_ID> \
    --tenant-id <TENANT_ID> \
    --client-id <AZURE_CLIENT_ID> \
    --keyvault-name <KEYVAULT_NAME> \
    [--dry-run]

What it does:
  1) Grants Key Vault secret read permissions to the service principal:
     - RBAC vaults: "Key Vault Secrets User" role assignment
     - Access-policy vaults: key vault policy with secret permissions get,list
  2) Ensures the service principal is a member of the Entra directory role "Application Administrator"
     (required for azuread_application operations in Terraform).

Notes:
  - Requires az CLI logged in as a user with sufficient permissions:
      * Subscription RBAC rights to create role assignments on the Key Vault scope
      * Entra role assignment rights (Privileged Role Administrator or Global Administrator)
  - Safe to re-run (idempotent checks included).
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --subscription-id=*) SUBSCRIPTION_ID="${arg#*=}" ;;
    --tenant-id=*) TENANT_ID="${arg#*=}" ;;
    --client-id=*) CLIENT_ID="${arg#*=}" ;;
    --keyvault-name=*) KEYVAULT_NAME="${arg#*=}" ;;
    --dry-run) DRY_RUN=1 ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -z "$SUBSCRIPTION_ID" ] || [ -z "$TENANT_ID" ] || [ -z "$CLIENT_ID" ] || [ -z "$KEYVAULT_NAME" ]; then
  echo "error: missing required arguments" >&2
  usage >&2
  exit 1
fi

if ! command -v az >/dev/null 2>&1; then
  echo "error: az CLI is not installed or not in PATH" >&2
  exit 1
fi

run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] $*"
  else
    "$@"
  fi
}

echo "==> Setting subscription context"
run az login --tenant "$TENANT_ID" --only-show-errors >/dev/null
run az account set --subscription "$SUBSCRIPTION_ID"

echo "==> Resolving service principal object ID"
SP_OBJECT_ID="$(az ad sp show --id "$CLIENT_ID" --query id -o tsv)"
if [ -z "$SP_OBJECT_ID" ]; then
  echo "error: could not resolve service principal object id for client id '$CLIENT_ID'" >&2
  exit 1
fi
echo "    service principal object id: $SP_OBJECT_ID"

echo "==> Resolving Key Vault resource ID"
KEYVAULT_ID="$(az keyvault show -n "$KEYVAULT_NAME" --query id -o tsv)"
if [ -z "$KEYVAULT_ID" ]; then
  echo "error: could not resolve key vault id for '$KEYVAULT_NAME'" >&2
  exit 1
fi
echo "    key vault id: $KEYVAULT_ID"

KV_RBAC_ENABLED="$(az keyvault show -n "$KEYVAULT_NAME" --query properties.enableRbacAuthorization -o tsv)"

if [ "$KV_RBAC_ENABLED" = "true" ]; then
  echo "==> Key Vault uses RBAC: ensuring Key Vault Secrets User role assignment exists"
  EXISTING_KV_ASSIGNMENT_COUNT="$(az role assignment list \
    --assignee-object-id "$SP_OBJECT_ID" \
    --scope "$KEYVAULT_ID" \
    --query "[?roleDefinitionName=='Key Vault Secrets User'] | length(@)" \
    -o tsv)"

  if [ "$EXISTING_KV_ASSIGNMENT_COUNT" = "0" ]; then
    run az role assignment create \
      --assignee-object-id "$SP_OBJECT_ID" \
      --assignee-principal-type ServicePrincipal \
      --role "Key Vault Secrets User" \
      --scope "$KEYVAULT_ID" >/dev/null
    if [ "$DRY_RUN" -eq 1 ]; then
      echo "    would create Key Vault Secrets User assignment"
    else
      echo "    created Key Vault Secrets User assignment"
    fi
  else
    echo "    assignment already present"
  fi
else
  echo "==> Key Vault uses Access Policies: ensuring get/list secret policy exists"
  HAS_GET="$(az keyvault show -n "$KEYVAULT_NAME" --query "contains(properties.accessPolicies[?objectId=='$SP_OBJECT_ID'].permissions.secrets[], 'get')" -o tsv)"
  HAS_LIST="$(az keyvault show -n "$KEYVAULT_NAME" --query "contains(properties.accessPolicies[?objectId=='$SP_OBJECT_ID'].permissions.secrets[], 'list')" -o tsv)"

  if [ "$HAS_GET" = "true" ] && [ "$HAS_LIST" = "true" ]; then
    echo "    access policy already present"
  else
    run az keyvault set-policy \
      -n "$KEYVAULT_NAME" \
      --object-id "$SP_OBJECT_ID" \
      --secret-permissions get list >/dev/null
    if [ "$DRY_RUN" -eq 1 ]; then
      echo "    would create key vault access policy (get,list)"
    else
      echo "    created key vault access policy (get,list)"
    fi
  fi
fi

echo "==> Ensuring Entra directory role 'Application Administrator' is active"
APP_ADMIN_ROLE_ID="$(az rest --method GET \
  --url "https://graph.microsoft.com/v1.0/directoryRoles?\$filter=displayName eq 'Application Administrator'" \
  --query 'value[0].id' -o tsv)"

if [ -z "$APP_ADMIN_ROLE_ID" ] || [ "$APP_ADMIN_ROLE_ID" = "null" ]; then
  TEMPLATE_ID="$(az rest --method GET \
    --url "https://graph.microsoft.com/v1.0/directoryRoleTemplates?\$filter=displayName eq 'Application Administrator'" \
    --query 'value[0].id' -o tsv)"

  if [ -z "$TEMPLATE_ID" ] || [ "$TEMPLATE_ID" = "null" ]; then
    echo "error: could not resolve directory role template for Application Administrator" >&2
    exit 1
  fi

  run az rest --method POST \
    --url "https://graph.microsoft.com/v1.0/directoryRoles" \
    --headers "Content-Type=application/json" \
    --body "{\"roleTemplateId\":\"$TEMPLATE_ID\"}" >/dev/null || true

  APP_ADMIN_ROLE_ID="$(az rest --method GET \
    --url "https://graph.microsoft.com/v1.0/directoryRoles?\$filter=displayName eq 'Application Administrator'" \
    --query 'value[0].id' -o tsv)"
fi

if [ -z "$APP_ADMIN_ROLE_ID" ] || [ "$APP_ADMIN_ROLE_ID" = "null" ]; then
  echo "error: failed to resolve directory role id for Application Administrator" >&2
  exit 1
fi
echo "    directory role id: $APP_ADMIN_ROLE_ID"

echo "==> Ensuring service principal is member of Application Administrator"
MEMBER_COUNT="$(az rest --method GET \
  --url "https://graph.microsoft.com/v1.0/directoryRoles/$APP_ADMIN_ROLE_ID/members" \
  --query "contains(value[].id, '$SP_OBJECT_ID')" -o tsv)"

if [ "$MEMBER_COUNT" = "false" ]; then
  run az rest --method POST \
    --url "https://graph.microsoft.com/v1.0/directoryRoles/${APP_ADMIN_ROLE_ID}/members/\$ref" \
    --headers "Content-Type=application/json" \
    --body "{\"@odata.id\":\"https://graph.microsoft.com/v1.0/directoryObjects/${SP_OBJECT_ID}\"}" >/dev/null
  echo "    added service principal to Application Administrator"
else
  echo "    membership already present"
fi

echo "==> Done"
echo "You can now re-run Terraform Azure workflow with confirm_apply=APPLY."
