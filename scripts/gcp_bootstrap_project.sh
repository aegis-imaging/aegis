#!/usr/bin/env bash
set -euo pipefail

TFVARS_PATH="terraform/project/terraform.tfvars"
RUN_APPLY=0
AUTO_APPROVE=0

for arg in "$@"; do
  case "$arg" in
    --apply) RUN_APPLY=1 ;;
    --auto-approve) AUTO_APPROVE=1 ;;
    --tfvars=*) TFVARS_PATH="${arg#*=}" ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_bootstrap_project.sh [--tfvars=path] [--apply] [--auto-approve]

Runs the AEGIS terraform/project bootstrap workflow:
1) local preflight checks
2) terraform init
3) terraform plan
4) optional terraform apply
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

if [ ! -f "$TFVARS_PATH" ]; then
  echo "error: tfvars file not found: $TFVARS_PATH" >&2
  echo "hint: copy terraform/project/terraform.tfvars.example and set project_id/region/billing_account" >&2
  exit 1
fi

./scripts/validate_tfvars_required.sh \
  "$TFVARS_PATH" \
  project_id \
  billing_account

if [ -x "./scripts/gcp_preflight.sh" ]; then
  ./scripts/gcp_preflight.sh --skip-docker
fi

echo "==> terraform/project init"
terraform -chdir=terraform/project init

echo "==> terraform/project plan"
terraform -chdir=terraform/project plan -var-file="../../${TFVARS_PATH}"

if [ "$RUN_APPLY" -eq 1 ]; then
  echo "==> terraform/project apply"
  if [ "$AUTO_APPROVE" -eq 1 ]; then
    terraform -chdir=terraform/project apply -auto-approve -var-file="../../${TFVARS_PATH}"
  else
    terraform -chdir=terraform/project apply -var-file="../../${TFVARS_PATH}"
  fi
fi
