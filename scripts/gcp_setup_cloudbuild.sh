#!/usr/bin/env bash
# gcp_setup_cloudbuild.sh — One-time setup for Cloud Build CI/CD
#
# Run this once to:
#   1. Grant the Cloud Build service account the IAM roles it needs to push
#      images to Artifact Registry and deploy to Cloud Run.
#   2. Create a Cloud Build trigger that auto-deploys on push to `develop`.
#
# PREREQUISITE (manual, console only):
#   Connect Cloud Build to the GitHub repository before running this script.
#   1. Open https://console.cloud.google.com/cloud-build/triggers
#   2. Click "Connect repository"
#   3. Choose "GitHub (Cloud Build GitHub App)"
#   4. Authenticate and select the aegis-imaging/aegis repository
#   5. Then run this script.
#
# Usage:
#   ./scripts/gcp_setup_cloudbuild.sh [--project-id=<id>] [--region=<region>]
#
# All flags have sensible production defaults; pass overrides when targeting a
# different environment.
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-aegis-prod-488120}"
REGION="${REGION:-us-central1}"
REPO_OWNER="${REPO_OWNER:-aegis-imaging}"
REPO_NAME="${REPO_NAME:-aegis}"
TRIGGER_NAME="${TRIGGER_NAME:-deploy-on-develop}"

for arg in "$@"; do
  case "$arg" in
    --project-id=*) PROJECT_ID="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --repo-owner=*) REPO_OWNER="${arg#*=}" ;;
    --repo-name=*) REPO_NAME="${arg#*=}" ;;
    --trigger-name=*) TRIGGER_NAME="${arg#*=}" ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_setup_cloudbuild.sh [options]

Options:
  --project-id=<id>       GCP project ID (default: aegis-prod-488120)
  --region=<region>       Cloud Build trigger region (default: us-central1)
  --repo-owner=<owner>    GitHub org/user (default: aegis-imaging)
  --repo-name=<name>      GitHub repo name (default: aegis)
  --trigger-name=<name>   Trigger name (default: deploy-on-develop)

Environment variables: PROJECT_ID, REGION, REPO_OWNER, REPO_NAME, TRIGGER_NAME
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

GCLOUD="${GCLOUD:-gcloud}"
if ! command -v "$GCLOUD" &>/dev/null; then
  # Try the well-known install location on this machine
  GCLOUD="/Users/aegis/google-cloud-sdk/bin/gcloud"
fi

echo "==> AEGIS Cloud Build setup"
echo "    Project : $PROJECT_ID"
echo "    Region  : $REGION"
echo "    Repo    : $REPO_OWNER/$REPO_NAME"
echo "    Trigger : $TRIGGER_NAME"
echo ""

# Resolve Cloud Build default service account: [PROJECT_NUMBER]@cloudbuild.gserviceaccount.com
echo "==> Resolving Cloud Build service account..."
PROJECT_NUMBER=$("$GCLOUD" projects describe "$PROJECT_ID" \
  --format="value(projectNumber)" 2>/dev/null)

if [ -z "$PROJECT_NUMBER" ]; then
  echo "error: could not resolve project number for '$PROJECT_ID'" >&2
  echo "       Make sure you are authenticated: gcloud auth login" >&2
  exit 1
fi

CB_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"
echo "    SA      : $CB_SA"
echo ""

# Grant IAM roles (idempotent — safe to re-run)
ROLES=(
  "roles/artifactregistry.writer"   # push images to Artifact Registry
  "roles/run.admin"                 # deploy / update Cloud Run services
  "roles/iam.serviceAccountUser"    # impersonate Cloud Run SA during deploy
)

for role in "${ROLES[@]}"; do
  echo "==> Granting $role to $CB_SA..."
  "$GCLOUD" projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:${CB_SA}" \
    --role="$role" \
    --condition=None \
    --quiet
done

echo ""

# Create (or update) the trigger
echo "==> Creating trigger '$TRIGGER_NAME'..."

# Check if the trigger already exists
if "$GCLOUD" builds triggers describe "$TRIGGER_NAME" \
    --project="$PROJECT_ID" \
    --region="$REGION" \
    &>/dev/null 2>&1; then
  echo "    Trigger already exists — deleting and recreating to apply latest config..."
  "$GCLOUD" builds triggers delete "$TRIGGER_NAME" \
    --project="$PROJECT_ID" \
    --region="$REGION" \
    --quiet
fi

"$GCLOUD" builds triggers create github \
  --project="$PROJECT_ID" \
  --name="$TRIGGER_NAME" \
  --description="Auto-deploy all services on merge to develop" \
  --repo-name="$REPO_NAME" \
  --repo-owner="$REPO_OWNER" \
  --branch-pattern="^develop$" \
  --build-config="cloudbuild.yaml" \
  --region="$REGION"

echo ""
echo "==> Done. Cloud Build trigger '$TRIGGER_NAME' is active."
echo ""
echo "    Next steps:"
echo "      1. Push a commit to the 'develop' branch to trigger a build."
echo "      2. Monitor: gcloud builds list --project=$PROJECT_ID --limit=5"
echo "      3. Smoke test after deploy: curl https://api.aegisimaging.ai/healthz | jq ."
echo ""
