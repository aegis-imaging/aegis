#!/usr/bin/env bash
set -euo pipefail

TFVARS_PATH="terraform/aws/terraform.tfvars"
RUN_APPLY=0
AUTO_APPROVE=0

for arg in "$@"; do
  case "$arg" in
    --tfvars=*) TFVARS_PATH="${arg#*=}" ;;
    --apply) RUN_APPLY=1 ;;
    --auto-approve) AUTO_APPROVE=1 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/aws_apply_infra.sh [--tfvars=path] [--apply] [--auto-approve]

Runs the AEGIS terraform/aws workflow:
1) local CLI preflight checks
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
  echo "hint: copy terraform/aws/terraform.tfvars.example and set required inputs" >&2
  exit 1
fi

./scripts/validate_tfvars_required.sh \
  "$TFVARS_PATH" \
  acm_certificate_arn \
  cognito_domain_prefix

for cmd in aws terraform; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
done

if ! aws sts get-caller-identity >/dev/null 2>&1; then
  echo "error: AWS credentials not configured. Run: aws configure (or set env credentials)" >&2
  exit 1
fi

echo "==> terraform/aws init"
terraform -chdir=terraform/aws init

echo "==> terraform/aws plan"
terraform -chdir=terraform/aws plan -var-file="../../${TFVARS_PATH}"

if [ "$RUN_APPLY" -eq 1 ]; then
  echo "==> terraform/aws apply"
  if [ "$AUTO_APPROVE" -eq 1 ]; then
    terraform -chdir=terraform/aws apply -auto-approve -var-file="../../${TFVARS_PATH}"
  else
    terraform -chdir=terraform/aws apply -var-file="../../${TFVARS_PATH}"
  fi
fi
