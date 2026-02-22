#!/usr/bin/env bash
# Force-deploy all AEGIS Cloud Run services to pick up new :latest images.
# Usage: ./scripts/deploy-services.sh [TAG]
# Default TAG: latest

set -euo pipefail

TAG="${1:-latest}"
PROJECT="aegis-prod-488120"
REGION="us-central1"
REGISTRY="us-central1-docker.pkg.dev/${PROJECT}/aegis-services"

sidecars=(
  "defacing"
  "phi-detection"
  "qc-service"
  "bids-service"
  "classification-service"
  "protocol-service"
)

echo "==> Deploying AEGIS Cloud Run services (tag: ${TAG})"
echo "    Project: ${PROJECT} / Region: ${REGION}"
echo ""

for svc in "${sidecars[@]}"; do
  echo "==> [${svc}] Deploying..."
  gcloud run services update "${svc}" \
    --region="${REGION}" \
    --project="${PROJECT}" \
    --image="${REGISTRY}/${svc}:${TAG}"
  echo "==> [${svc}] Done."
  echo ""
done

echo "==> [api] Deploying..."
gcloud run services update aegis-api \
  --region="${REGION}" \
  --project="${PROJECT}" \
  --image="${REGISTRY}/api:${TAG}"
echo "==> [api] Done."
echo ""

echo "==> [admin-dashboard] Deploying..."
gcloud run services update aegis-admin-dashboard \
  --region="${REGION}" \
  --project="${PROJECT}" \
  --image="${REGISTRY}/admin-dashboard:${TAG}"
echo "==> [admin-dashboard] Done."
echo ""

echo "==> All services deployed."
