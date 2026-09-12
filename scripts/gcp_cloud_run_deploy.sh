#!/usr/bin/env bash
# gcp_cloud_run_deploy.sh <service> <image> [extra `gcloud run deploy` flags...]
#
# Deploys one Cloud Run service. When the new revision fails to start, prints
# its recent logs so the cause is visible in the workflow run instead of
# behind a console link. Requires PROJECT_ID and REGION in the environment.
set -euo pipefail

SERVICE="${1:?service name required}"
IMAGE="${2:?image required}"
shift 2
: "${PROJECT_ID:?PROJECT_ID required}" "${REGION:?REGION required}"

if gcloud run deploy "$SERVICE" --image="$IMAGE" --region="$REGION" --project="$PROJECT_ID" --quiet "$@"; then
  exit 0
fi

SINCE="$(date -u -d '-15 minutes' +%Y-%m-%dT%H:%M:%SZ)"
echo "::group::Recent logs for $SERVICE (severity >= WARNING, last 15 minutes)"
gcloud logging read \
  "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"$SERVICE\" AND severity>=WARNING AND timestamp>=\"$SINCE\"" \
  --project="$PROJECT_ID" --limit=80 --order=asc \
  --format='value(timestamp,severity,textPayload,jsonPayload.msg,jsonPayload.message,jsonPayload.error)' || true
echo "::endgroup::"
echo "::error::Cloud Run deploy of $SERVICE failed — see the log excerpt above"
exit 1
