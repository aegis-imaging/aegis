#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-}"
REGION="${REGION:-us-central1}"
REPOSITORY="${REPOSITORY:-aegis-services}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH=1

for arg in "$@"; do
  case "$arg" in
    --project-id=*) PROJECT_ID="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --repository=*) REPOSITORY="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --platform=*) PLATFORM="${arg#*=}" ;;
    --no-push) PUSH=0 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_build_push_images.sh --project-id=<id> [options]

Options:
  --region=<region>         Default: us-central1
  --repository=<repo>       Default: aegis-services
  --tag=<tag>               Default: latest
  --platform=<platform>     Default: linux/amd64
  --no-push                 Build locally (uses --load) instead of pushing

Environment variables supported: PROJECT_ID, REGION, REPOSITORY, TAG, PLATFORM
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
  echo "error: PROJECT_ID is required (use --project-id or env PROJECT_ID)" >&2
  exit 1
fi

if [ ! -x "./scripts/gcp_preflight.sh" ]; then
  echo "error: missing preflight script: ./scripts/gcp_preflight.sh" >&2
  exit 1
fi

if [ "$PUSH" -eq 1 ]; then
  ./scripts/gcp_preflight.sh
else
  ./scripts/gcp_preflight.sh --skip-adc
fi

REGISTRY_HOST="${REGION}-docker.pkg.dev"
REPO_BASE="${REGISTRY_HOST}/${PROJECT_ID}/${REPOSITORY}"

if [ "$PUSH" -eq 1 ]; then
  gcloud auth configure-docker "$REGISTRY_HOST" --quiet
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
  image="${REPO_BASE}/${name}:${TAG}"

  echo "==> Building ${image} from ${context}"
  if [ "$PUSH" -eq 1 ]; then
    docker buildx build --platform "$PLATFORM" -t "$image" --push "$context"
  else
    docker buildx build --platform "$PLATFORM" -t "$image" --load "$context"
  fi
done
