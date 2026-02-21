#!/usr/bin/env bash
set -euo pipefail

REGION="${REGION:-us-east-1}"
PROJECT_NAME="${PROJECT_NAME:-aegis}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH=1

for arg in "$@"; do
  case "$arg" in
    --region=*) REGION="${arg#*=}" ;;
    --project-name=*) PROJECT_NAME="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --platform=*) PLATFORM="${arg#*=}" ;;
    --no-push) PUSH=0 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/aws_build_push_images.sh [options]

Options:
  --region=<region>           Default: us-east-1
  --project-name=<name>       Default: aegis
  --tag=<tag>                 Default: latest
  --platform=<platform>       Default: linux/amd64
  --no-push                   Build locally with --load only

Environment variables supported: REGION, PROJECT_NAME, TAG, PLATFORM
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

for cmd in aws docker; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
done

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text 2>/dev/null || true)"
if [ -z "$ACCOUNT_ID" ] || [ "$ACCOUNT_ID" = "None" ]; then
  echo "error: unable to determine AWS account id. Configure aws credentials first." >&2
  exit 1
fi

REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"

if [ "$PUSH" -eq 1 ]; then
  aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY"
fi

SERVICES=(
  "api:api"
  "admin-dashboard:frontend/admin-dashboard"
  "defacing:defacing"
  "phi-detection:phi-detection"
  "qc-service:qc-service"
  "bids-service:bids-service"
  "classification-service:classification-service"
  "protocol-service:protocol-service"
  "dimse-receiver:dimse-receiver"
)

for service in "${SERVICES[@]}"; do
  name="${service%%:*}"
  context="${service#*:}"
  image="${REGISTRY}/${PROJECT_NAME}/${name}:${TAG}"

  echo "==> Building ${image} from ${context}"
  if [ "$PUSH" -eq 1 ]; then
    docker buildx build --platform "$PLATFORM" -t "$image" --push "$context"
  else
    docker buildx build --platform "$PLATFORM" -t "$image" --load "$context"
  fi
done
