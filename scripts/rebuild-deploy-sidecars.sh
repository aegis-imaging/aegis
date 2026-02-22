#!/usr/bin/env bash
# Rebuild sidecar images (+ API) and redeploy all Cloud Run services.
# Usage: ./scripts/rebuild-deploy-sidecars.sh [TAG]
# Default TAG: latest

set -euo pipefail

TAG="${1:-latest}"
PROJECT="aegis-prod-488120"
REGION="us-central1"
REGISTRY="${REGION}-docker.pkg.dev/${PROJECT}/aegis-services"
PLATFORM="linux/amd64"

sidecars=(
  "defacing"
  "phi-detection"
  "qc-service"
  "bids-service"
  "classification-service"
  "protocol-service"
)

echo "==> Rebuilding AEGIS sidecar images + API (tag: ${TAG})"
echo "    Registry: ${REGISTRY}"
echo ""

gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet

for svc in "${sidecars[@]}"; do
  echo "==> [${svc}] Building..."
  docker buildx build --platform "${PLATFORM}" \
    -t "${REGISTRY}/${svc}:${TAG}" \
    --push "./${svc}"
  echo "==> [${svc}] Pushed."
  echo ""
done

echo "==> [api] Building..."
docker buildx build --platform "${PLATFORM}" \
  -t "${REGISTRY}/api:${TAG}" \
  --push "./api"
echo "==> [api] Pushed."
echo ""

echo "==> Deploying all services..."
./scripts/deploy-services.sh "${TAG}"
echo ""

echo "==> Applying Terraform liveness probe update..."
(cd terraform/infra && terraform apply -target=google_cloud_run_v2_service.sidecars -auto-approve)
echo ""

echo "==> Done. Waiting 35s for health loop to re-probe..."
sleep 35
curl -s https://api.aegisimaging.ai/healthz | python3 -m json.tool
