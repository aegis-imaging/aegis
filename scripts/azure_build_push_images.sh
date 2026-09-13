#!/usr/bin/env bash
set -euo pipefail

REGISTRY="${REGISTRY:-}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
SERVICES="${SERVICES:-}"
PUSH=1

for arg in "$@"; do
  case "$arg" in
    --registry=*) REGISTRY="${arg#*=}" ;;
    --tag=*) TAG="${arg#*=}" ;;
    --platform=*) PLATFORM="${arg#*=}" ;;
    --services=*) SERVICES="${arg#*=}" ;;
    --no-push) PUSH=0 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/azure_build_push_images.sh [options]

Options:
  --registry=<login-server>   ACR login server, e.g. aegisprodacr.azurecr.io.
                              Default: terraform -chdir=terraform/azure output -raw acr_login_server
  --tag=<tag>                 Default: latest
  --platform=<platform>       Default: linux/amd64
  --services=<a,b,c>          Subset to build (default: api admin-dashboard defacing
                              phi-detection qc-service bids-service classification-service
                              protocol-service dimse-receiver). Also accepts upload-portal,
                              dwv, mcp-server, synth-service, analytics-service, sct-service.
                              Minimal footprint: --services=api,admin-dashboard
  --no-push                   Build locally with --load only

Images are pushed as <registry>/<service>:<tag>, which is what terraform/azure
expects. Terraform ignores image changes after the first apply, so roll a new
image with: az containerapp update -g <rg> -n aegis-prod-<service> --image <ref>

Environment variables supported: REGISTRY, TAG, PLATFORM, SERVICES,
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

for cmd in az docker; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
done

SERVICES="$(resolve_image_services "$SERVICES")"

if [ -z "$REGISTRY" ]; then
  REGISTRY="$(terraform -chdir=terraform/azure output -raw acr_login_server 2>/dev/null || true)"
fi
if [ -z "$REGISTRY" ]; then
  echo "error: ACR login server unknown. Pass --registry=<name>.azurecr.io or apply terraform/azure first." >&2
  exit 1
fi

if [ "$PUSH" -eq 1 ]; then
  az acr login --name "${REGISTRY%%.*}"
fi

for name in $SERVICES; do
  build_image "${REGISTRY}/${name}:${TAG}" "$name" "$PLATFORM" "$PUSH"
done
