#!/usr/bin/env bash
set -euo pipefail

REGION="${REGION:-us-east-1}"
PROJECT_NAME="${PROJECT_NAME:-aegis}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
SERVICES="${SERVICES:-}"
PUSH=1

for arg in "$@"; do
  case "$arg" in
    --region=*) REGION="${arg#*=}" ;;
    --project-name=*) PROJECT_NAME="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --platform=*) PLATFORM="${arg#*=}" ;;
    --services=*) SERVICES="${arg#*=}" ;;
    --no-push) PUSH=0 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/aws_build_push_images.sh [options]

Options:
  --region=<region>           Default: us-east-1
  --project-name=<name>       Default: aegis
  --tag=<tag>                 Default: latest
  --platform=<platform>       Default: linux/amd64
  --services=<a,b,c>          Subset to build (default: api admin-dashboard defacing
                              phi-detection qc-service bids-service classification-service
                              protocol-service dimse-receiver). Also accepts upload-portal,
                              dwv, mcp-server, synth-service, analytics-service, sct-service.
                              Minimal footprint: --services=api,admin-dashboard
  --no-push                   Build locally with --load only

Environment variables supported: REGION, PROJECT_NAME, TAG, PLATFORM, SERVICES,
DWV_BASE_URL and OHIF_BASE_URL (baked into the admin dashboard bundle; optional).
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

# shellcheck source=scripts/image_build_common.sh
. "$(dirname "$0")/image_build_common.sh"

for cmd in aws docker; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
done

SERVICES="$(resolve_image_services "$SERVICES")"

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text 2>/dev/null || true)"
if [ -z "$ACCOUNT_ID" ] || [ "$ACCOUNT_ID" = "None" ]; then
  echo "error: unable to determine AWS account id. Configure aws credentials first." >&2
  exit 1
fi

REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"

if [ "$PUSH" -eq 1 ]; then
  aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY"
fi

for name in $SERVICES; do
  build_image "${REGISTRY}/${PROJECT_NAME}/${name}:${TAG}" "$name" "$PLATFORM" "$PUSH"
done
