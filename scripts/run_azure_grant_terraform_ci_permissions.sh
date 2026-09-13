#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/scripts/azure_terraform_ci_permissions.env"
DRY_RUN=""

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/run_azure_grant_terraform_ci_permissions.sh [--env-file=path] [--dry-run]

Defaults:
  --env-file=./scripts/azure_terraform_ci_permissions.env

Steps:
  1) Copy scripts/azure_terraform_ci_permissions.env.example to scripts/azure_terraform_ci_permissions.env
  2) Fill in AZURE_CLIENT_ID (subscription + tenant auto-detect from az account if left blank)
  3) Run this wrapper script
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --env-file=*) ENV_FILE="${arg#*=}" ;;
    --dry-run) DRY_RUN="--dry-run" ;;
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

if [ ! -f "$ENV_FILE" ]; then
  echo "error: env file not found: $ENV_FILE" >&2
  echo "hint: cp scripts/azure_terraform_ci_permissions.env.example scripts/azure_terraform_ci_permissions.env" >&2
  exit 1
fi

set -a
source "$ENV_FILE"
set +a

if [ -z "${AZURE_SUBSCRIPTION_ID:-}" ] || [ -z "${AZURE_TENANT_ID:-}" ]; then
  if command -v az >/dev/null 2>&1; then
    AZURE_SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID:-$(az account show --query id -o tsv 2>/dev/null || true)}"
    AZURE_TENANT_ID="${AZURE_TENANT_ID:-$(az account show --query tenantId -o tsv 2>/dev/null || true)}"
  fi
fi

if [ -z "${AZURE_KEYVAULT_NAME:-}" ]; then
  AZURE_KEYVAULT_NAME="aegis-prod-kv-d2r8el"
fi

required_vars=(
  AZURE_SUBSCRIPTION_ID
  AZURE_TENANT_ID
  AZURE_CLIENT_ID
  AZURE_KEYVAULT_NAME
)

for var in "${required_vars[@]}"; do
  if [ -z "${!var:-}" ]; then
    echo "error: $var is empty in $ENV_FILE" >&2
    exit 1
  fi
done

"$ROOT_DIR/scripts/azure_grant_terraform_ci_permissions.sh" \
  --subscription-id="$AZURE_SUBSCRIPTION_ID" \
  --tenant-id="$AZURE_TENANT_ID" \
  --client-id="$AZURE_CLIENT_ID" \
  --keyvault-name="$AZURE_KEYVAULT_NAME" \
  $DRY_RUN
