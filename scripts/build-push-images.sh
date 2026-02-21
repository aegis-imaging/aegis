#!/usr/bin/env bash
# Build and push all AEGIS service images to Google Artifact Registry.
# Usage: ./scripts/build-push-images.sh [TAG]
# Default TAG: latest

set -euo pipefail

TAG="${1:-latest}"
REGISTRY="us-central1-docker.pkg.dev/aegis-prod-488120/aegis-services"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Building and pushing AEGIS images (tag: $TAG)"
echo "    Registry: $REGISTRY"
echo ""

services=(
  "api"
  "defacing"
  "phi-detection"
  "qc-service"
  "bids-service"
  "classification-service"
  "protocol-service"
)

for svc in "${services[@]}"; do
  echo "==> [$svc] Building..."
  docker build --platform linux/amd64 -t "$REGISTRY/$svc:$TAG" "$REPO_ROOT/$svc"
  echo "==> [$svc] Pushing..."
  docker push "$REGISTRY/$svc:$TAG"
  echo "==> [$svc] Done."
  echo ""
done

echo "==> [admin-dashboard] Building..."
docker build --platform linux/amd64 -t "$REGISTRY/admin-dashboard:$TAG" "$REPO_ROOT/frontend/admin-dashboard"
echo "==> [admin-dashboard] Pushing..."
docker push "$REGISTRY/admin-dashboard:$TAG"
echo "==> [admin-dashboard] Done."
echo ""

echo "==> All images pushed successfully."
echo ""
echo "Images:"
for svc in "${services[@]}" admin-dashboard; do
  echo "    $REGISTRY/$svc:$TAG"
done
