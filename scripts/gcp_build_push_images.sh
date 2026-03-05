#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-}"
REGION="${REGION:-us-central1}"
REPOSITORY="${REPOSITORY:-aegis-services}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH=1
# DWV_BASE_URL: set to DWV Cloud Run URL to bake it into the admin-dashboard build.
# Example: DWV_BASE_URL=https://dwv-abc123-uc.a.run.app
# Leave empty to use localhost:3005 fallback (local dev).
DWV_BASE_URL="${DWV_BASE_URL:-}"

for arg in "$@"; do
  case "$arg" in
    --project-id=*) PROJECT_ID="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --repository=*) REPOSITORY="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --platform=*) PLATFORM="${arg#*=}" ;;
    --dwv-base-url=*) DWV_BASE_URL="${arg#*=}" ;;
    --no-push) PUSH=0 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_build_push_images.sh --project-id=<id> [options]

Options:
  --region=<region>         Default: us-central1
  --repository=<repo>       Default: aegis-services
  --tag=<tag>               Default: latest
  --platform=<platform>     Default: linux/amd64
  --dwv-base-url=<url>      DWV viewer URL baked into admin-dashboard build (optional)
  --no-push                 Build locally (uses --load) instead of pushing

Environment variables supported: PROJECT_ID, REGION, REPOSITORY, TAG, PLATFORM, DWV_BASE_URL
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

# admin-dashboard is built separately because it takes an optional DWV_BASE_URL build arg
ADMIN_IMAGE="${REPO_BASE}/admin-dashboard:${TAG}"
echo "==> Building ${ADMIN_IMAGE} from frontend/admin-dashboard"
ADMIN_BUILD_ARGS="--build-arg VITE_DWV_BASE_URL=${DWV_BASE_URL}"
if [ "$PUSH" -eq 1 ]; then
  # shellcheck disable=SC2086
  docker buildx build --platform "$PLATFORM" $ADMIN_BUILD_ARGS -t "$ADMIN_IMAGE" --push "frontend/admin-dashboard"
else
  # shellcheck disable=SC2086
  docker buildx build --platform "$PLATFORM" $ADMIN_BUILD_ARGS -t "$ADMIN_IMAGE" --load "frontend/admin-dashboard"
fi
