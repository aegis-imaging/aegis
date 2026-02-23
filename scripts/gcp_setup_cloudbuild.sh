#!/usr/bin/env bash
# gcp_setup_cloudbuild.sh — One-time setup for Cloud Build CI/CD
#
# Run this once to:
#   1. Grant the Cloud Build service account the IAM roles it needs to push
#      images to Artifact Registry and deploy to Cloud Run.
#   2. Link the GitHub repository to the Cloud Build connection.
#   3. Create a Cloud Build trigger that auto-deploys on push to `develop`.
#
# PREREQUISITE (manual, console only — do this ONCE before running the script):
#   Connect Cloud Build to GitHub via the Cloud Build GitHub App.
#   1. Open https://console.cloud.google.com/cloud-build/triggers/connect
#   2. Select "GitHub (Cloud Build GitHub App)" and click Continue
#   3. Authenticate with GitHub, select the aegis-imaging org + aegis repo
#   4. Note the connection name created (default: "aegis") — pass as
#      --connection-name=<name> if it differs
#
# After the GitHub App connection exists, this script:
#   - Links the aegis repo to the connection (gcloud builds repositories create)
#   - Creates the push-to-develop trigger using the 2nd-gen --repository flag
#   - Requires --service-account (2nd-gen triggers mandate this)
#
# Usage:
#   ./scripts/gcp_setup_cloudbuild.sh [--project-id=<id>] [--region=<region>]
#
# All flags have sensible production defaults; pass overrides when targeting a
# different environment.
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-aegis-prod-488120}"
REGION="${REGION:-us-central1}"
CONNECTION_NAME="${CONNECTION_NAME:-aegis}"
REPO_NAME="${REPO_NAME:-aegis}"
REPO_URI="${REPO_URI:-https://github.com/aegis-imaging/aegis.git}"
TRIGGER_NAME="${TRIGGER_NAME:-deploy-on-develop}"
# Service account that Cloud Build runs as (must have run.admin + artifactregistry.writer)
CB_BYOSA="${CB_BYOSA:-aegis-cloud-build@${PROJECT_ID}.iam.gserviceaccount.com}"

for arg in "$@"; do
  case "$arg" in
    --project-id=*) PROJECT_ID="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --connection-name=*) CONNECTION_NAME="${arg#*=}" ;;
    --repo-name=*) REPO_NAME="${arg#*=}" ;;
    --repo-uri=*) REPO_URI="${arg#*=}" ;;
    --trigger-name=*) TRIGGER_NAME="${arg#*=}" ;;
    --service-account=*) CB_BYOSA="${arg#*=}" ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_setup_cloudbuild.sh [options]

Options:
  --project-id=<id>         GCP project ID (default: aegis-prod-488120)
  --region=<region>         Cloud Build region (default: us-central1)
  --connection-name=<name>  Cloud Build connection name (default: aegis)
  --repo-name=<name>        Repository resource name within connection (default: aegis)
  --repo-uri=<uri>          GitHub remote URI (default: https://github.com/aegis-imaging/aegis.git)
  --trigger-name=<name>     Trigger name (default: deploy-on-develop)
  --service-account=<email> SA email for trigger runs (default: aegis-cloud-build@PROJECT.iam.gserviceaccount.com)

Environment variables: PROJECT_ID, REGION, CONNECTION_NAME, REPO_NAME, REPO_URI, TRIGGER_NAME, CB_BYOSA
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

REPO_RESOURCE="projects/${PROJECT_ID}/locations/${REGION}/connections/${CONNECTION_NAME}/repositories/${REPO_NAME}"
SA_RESOURCE="projects/${PROJECT_ID}/serviceAccounts/${CB_BYOSA}"

echo "==> AEGIS Cloud Build setup"
echo "    Project    : $PROJECT_ID"
echo "    Region     : $REGION"
echo "    Connection : $CONNECTION_NAME"
echo "    Repository : $REPO_RESOURCE"
echo "    Trigger    : $TRIGGER_NAME"
echo "    Run SA     : $CB_BYOSA"
echo ""

# Resolve the default Cloud Build SA for IAM grants
echo "==> Resolving Cloud Build service account..."
PROJECT_NUMBER=$("$GCLOUD" projects describe "$PROJECT_ID" \
  --format="value(projectNumber)" 2>/dev/null)

if [ -z "$PROJECT_NUMBER" ]; then
  echo "error: could not resolve project number for '$PROJECT_ID'" >&2
  echo "       Make sure you are authenticated: gcloud auth login" >&2
  exit 1
fi

CB_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"
echo "    Default SA : $CB_SA"
echo ""

# Grant IAM roles to BOTH the default SA and the BYOSA (idempotent — safe to re-run)
ROLES=(
  "roles/artifactregistry.writer"   # push images to Artifact Registry
  "roles/run.admin"                 # deploy / update Cloud Run services
  "roles/iam.serviceAccountUser"    # impersonate Cloud Run SA during deploy
)

for sa in "$CB_SA" "$CB_BYOSA"; do
  for role in "${ROLES[@]}"; do
    echo "==> Granting $role to $sa..."
    "$GCLOUD" projects add-iam-policy-binding "$PROJECT_ID" \
      --member="serviceAccount:${sa}" \
      --role="$role" \
      --condition=None \
      --quiet
  done
done

echo ""

# Link the repository to the connection (idempotent — skip if already linked)
echo "==> Linking repository to connection '$CONNECTION_NAME'..."
if "$GCLOUD" builds repositories describe "$REPO_NAME" \
    --connection="$CONNECTION_NAME" \
    --region="$REGION" \
    --project="$PROJECT_ID" \
    &>/dev/null 2>&1; then
  echo "    Repository already linked — skipping."
else
  "$GCLOUD" builds repositories create "$REPO_NAME" \
    --connection="$CONNECTION_NAME" \
    --remote-uri="$REPO_URI" \
    --region="$REGION" \
    --project="$PROJECT_ID"
  echo "    Repository linked."
fi

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

# Note: --service-account is required for 2nd-gen (--repository) triggers.
# Using the existing aegis-cloud-build SA which already has the needed roles.
"$GCLOUD" builds triggers create github \
  --project="$PROJECT_ID" \
  --name="$TRIGGER_NAME" \
  --description="Auto-deploy all services on merge to develop" \
  --repository="$REPO_RESOURCE" \
  --branch-pattern="^develop$" \
  --build-config="cloudbuild.yaml" \
  --service-account="$SA_RESOURCE" \
  --region="$REGION"

echo ""
echo "==> Done. Cloud Build trigger '$TRIGGER_NAME' is active."
echo ""
echo "    Next steps:"
echo "      1. Push a commit to the 'develop' branch to trigger a build."
echo "      2. Monitor: gcloud builds list --project=$PROJECT_ID --limit=5"
echo "      3. Smoke test after deploy: curl https://api.aegisimaging.ai/healthz | jq ."
echo ""
