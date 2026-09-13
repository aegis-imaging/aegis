#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/scripts/azure_terraform_ci_permissions.env"
DRY_RUN=0
NO_WATCH=0
REF="main"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/run_azure_permissions_and_trigger_apply.sh [--env-file=path] [--dry-run] [--no-watch] [--ref=main]

What it does:
  1) Runs scripts/run_azure_grant_terraform_ci_permissions.sh
  2) Triggers GitHub Actions workflow "Terraform Azure" with confirm_apply=APPLY

Options:
  --env-file=path   Path to env file (default: ./scripts/azure_terraform_ci_permissions.env)
  --dry-run         Passes --dry-run to permission grant step and stops before workflow trigger
  --no-watch        Trigger workflow without waiting for completion
  --ref=branch      Git ref for workflow dispatch (default: main)
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --env-file=*) ENV_FILE="${arg#*=}" ;;
    --dry-run) DRY_RUN=1 ;;
    --no-watch) NO_WATCH=1 ;;
    --ref=*) REF="${arg#*=}" ;;
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

echo "==> Step 1/2: grant Azure permissions for Terraform CI principal"
if [ "$DRY_RUN" -eq 1 ]; then
  "$ROOT_DIR/scripts/run_azure_grant_terraform_ci_permissions.sh" --env-file="$ENV_FILE" --dry-run
  echo "==> Dry-run mode enabled; skipping workflow trigger step"
  exit 0
fi

"$ROOT_DIR/scripts/run_azure_grant_terraform_ci_permissions.sh" --env-file="$ENV_FILE"

echo "==> Step 2/2: trigger Terraform Azure apply workflow"
if [ "$NO_WATCH" -eq 1 ]; then
  "$ROOT_DIR/scripts/trigger_terraform_azure_apply.sh" --ref="$REF" --confirm=APPLY --no-watch
else
  "$ROOT_DIR/scripts/trigger_terraform_azure_apply.sh" --ref="$REF" --confirm=APPLY
fi
