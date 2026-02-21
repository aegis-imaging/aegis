#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-}"
REGION="${REGION:-us-central1}"
TAG="${TAG:-latest}"
PROJECT_TFVARS="${PROJECT_TFVARS:-terraform/project/terraform.tfvars}"
INFRA_TFVARS="${INFRA_TFVARS:-terraform/infra/terraform.tfvars}"
SKIP_BOOTSTRAP=0
SKIP_IMAGES=0
SKIP_APPLY=0

for arg in "$@"; do
  case "$arg" in
    --project-id=*) PROJECT_ID="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --project-tfvars=*) PROJECT_TFVARS="${arg#*=}" ;;
    --infra-tfvars=*) INFRA_TFVARS="${arg#*=}" ;;
    --skip-bootstrap) SKIP_BOOTSTRAP=1 ;;
    --skip-images) SKIP_IMAGES=1 ;;
    --skip-apply) SKIP_APPLY=1 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_install_poc.sh --project-id=<id> [options]

Options:
  --region=<region>                 Default: us-central1
  --tag=<tag>                       Default: latest
  --project-tfvars=<path>           Default: terraform/project/terraform.tfvars
  --infra-tfvars=<path>             Default: terraform/infra/terraform.tfvars
  --skip-bootstrap                  Skip terraform/project apply
  --skip-images                     Skip docker build/push step
  --skip-apply                      Stop after plan steps (no terraform apply)

This orchestrates:
1) gcp preflight checks
2) terraform/project bootstrap apply
3) terraform/infra Artifact Registry bootstrap
4) image build/push
5) terraform/infra full apply
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

if [ -z "$PROJECT_ID" ]; then
  echo "error: --project-id is required (or env PROJECT_ID)" >&2
  exit 1
fi

if [ ! -f "$PROJECT_TFVARS" ]; then
  echo "error: missing project tfvars file: $PROJECT_TFVARS" >&2
  exit 1
fi

if [ ! -f "$INFRA_TFVARS" ]; then
  echo "error: missing infra tfvars file: $INFRA_TFVARS" >&2
  exit 1
fi

./scripts/validate_tfvars_required.sh \
  "$PROJECT_TFVARS" \
  project_id \
  billing_account

./scripts/validate_tfvars_required.sh \
  "$INFRA_TFVARS" \
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

./scripts/gcp_preflight.sh

if [ "$SKIP_BOOTSTRAP" -eq 0 ]; then
  if [ "$SKIP_APPLY" -eq 1 ]; then
    ./scripts/gcp_bootstrap_project.sh --tfvars="$PROJECT_TFVARS"
  else
    ./scripts/gcp_bootstrap_project.sh --tfvars="$PROJECT_TFVARS" --apply
  fi
fi

if [ "$SKIP_APPLY" -eq 1 ]; then
  ./scripts/gcp_apply_infra.sh --tfvars="$INFRA_TFVARS"
else
  ./scripts/gcp_apply_infra.sh --tfvars="$INFRA_TFVARS" --target-artifact-repo
fi

if [ "$SKIP_IMAGES" -eq 0 ]; then
  ./scripts/gcp_build_push_images.sh --project-id="$PROJECT_ID" --region="$REGION" --tag="$TAG"
fi

if [ "$SKIP_APPLY" -eq 0 ]; then
  ./scripts/gcp_apply_infra.sh --tfvars="$INFRA_TFVARS" --apply
fi
