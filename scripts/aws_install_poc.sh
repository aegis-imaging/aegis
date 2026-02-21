#!/usr/bin/env bash
set -euo pipefail

TFVARS="${TFVARS:-terraform/aws/terraform.tfvars}"
REGION="${REGION:-us-east-1}"
PROJECT_NAME="${PROJECT_NAME:-aegis}"
TAG="${TAG:-latest}"
SKIP_FIRST_APPLY=0
SKIP_IMAGES=0
SKIP_SECOND_APPLY=0

for arg in "$@"; do
  case "$arg" in
    --tfvars=*) TFVARS="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --project-name=*) PROJECT_NAME="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --skip-first-apply) SKIP_FIRST_APPLY=1 ;;
    --skip-images) SKIP_IMAGES=1 ;;
    --skip-second-apply) SKIP_SECOND_APPLY=1 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/aws_install_poc.sh [options]

Options:
  --tfvars=<path>               Default: terraform/aws/terraform.tfvars
  --region=<region>             Default: us-east-1
  --project-name=<name>         Default: aegis
  --tag=<tag>                   Default: latest
  --skip-first-apply            Skip initial infra apply (repo/bootstrap)
  --skip-images                 Skip image build/push
  --skip-second-apply           Skip final apply (roll new image tags)

This orchestrates:
1) initial terraform/aws apply
2) ECR image build/push
3) final terraform/aws apply
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

if [ ! -f "$TFVARS" ]; then
  echo "error: missing tfvars file: $TFVARS" >&2
  exit 1
fi

./scripts/validate_tfvars_required.sh \
  "$TFVARS" \
  acm_certificate_arn \
  cognito_domain_prefix

if [ "$SKIP_FIRST_APPLY" -eq 0 ]; then
  ./scripts/aws_apply_infra.sh --tfvars="$TFVARS" --apply
fi

if [ "$SKIP_IMAGES" -eq 0 ]; then
  ./scripts/aws_build_push_images.sh --region="$REGION" --project-name="$PROJECT_NAME" --tag="$TAG"
fi

if [ "$SKIP_SECOND_APPLY" -eq 0 ]; then
  ./scripts/aws_apply_infra.sh --tfvars="$TFVARS" --apply
fi
