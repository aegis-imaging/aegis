#!/usr/bin/env bash
set -euo pipefail

TFVARS_PATH="terraform/infra/terraform.tfvars"
RUN_APPLY=0
AUTO_APPROVE=0
TARGET_ARTIFACT_REPO=0

for arg in "$@"; do
  case "$arg" in
    --tfvars=*) TFVARS_PATH="${arg#*=}" ;;
    --apply) RUN_APPLY=1 ;;
    --auto-approve) AUTO_APPROVE=1 ;;
    --target-artifact-repo) TARGET_ARTIFACT_REPO=1 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_apply_infra.sh [--tfvars=path] [--target-artifact-repo] [--apply] [--auto-approve]

Runs the AEGIS terraform/infra workflow:
1) local preflight checks
2) terraform init
3) optional targeted apply for Artifact Registry
4) terraform plan
5) optional terraform apply
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
  echo "hint: copy terraform/infra/terraform.tfvars.example and set required fields" >&2
  exit 1
fi

./scripts/validate_tfvars_required.sh \
  "$TFVARS_PATH" \
  project_id \
  api_domain \
  admin_domain \
  iap_oauth_client_id \
  iap_oauth_client_secret \
  api_image \
  admin_dashboard_image \
  defacing_image \
  phi_detection_image \
  qc_service_image \
  bids_service_image \
  classification_service_image \
  protocol_service_image

if [ -x "./scripts/gcp_preflight.sh" ]; then
  ./scripts/gcp_preflight.sh --skip-docker
fi

echo "==> terraform/infra init"
terraform -chdir=terraform/infra init

if [ "$TARGET_ARTIFACT_REPO" -eq 1 ]; then
  echo "==> terraform/infra apply target=google_artifact_registry_repository.services"
  terraform -chdir=terraform/infra apply -target=google_artifact_registry_repository.services -var-file="../../${TFVARS_PATH}"
fi

echo "==> terraform/infra plan"
terraform -chdir=terraform/infra plan -var-file="../../${TFVARS_PATH}"

if [ "$RUN_APPLY" -eq 1 ]; then
  echo "==> terraform/infra apply"
  if [ "$AUTO_APPROVE" -eq 1 ]; then
    terraform -chdir=terraform/infra apply -auto-approve -var-file="../../${TFVARS_PATH}"
  else
    terraform -chdir=terraform/infra apply -var-file="../../${TFVARS_PATH}"
  fi
fi
