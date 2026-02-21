#!/usr/bin/env bash
set -euo pipefail

TFVARS_PATH="terraform/infra/terraform.tfvars"
TERRAFORM_DIR="terraform/infra"
PROJECT_ID=""
REGION=""
API_URL=""
ADMIN_URL=""
EXPECT_ADMIN_AUTH=""

usage() {
  cat <<'USAGE'
Usage: ./scripts/gcp_verify_deployment.sh [options]

Options:
  --tfvars=<path>              Default: terraform/infra/terraform.tfvars
  --terraform-dir=<path>       Default: terraform/infra
  --project-id=<id>            Override project_id (else read from tfvars)
  --region=<region>            Override region (else read from tfvars or us-central1)
  --api-url=<url>              Override API base URL (else terraform output api_base_url or tfvars api_domain)
  --admin-url=<url>            Override admin base URL (else terraform output admin_base_url or tfvars admin_domain)
  --expect-admin-auth=<bool>   Force admin auth expectation (true/false). Defaults from tfvars enable_admin_iap or true.
  -h, --help                   Show this help

Checks:
1) required local tools + tfvars validation
2) terraform outputs available (LB IP + SMTP egress IP)
3) Cloud Run api/admin services exist and have ready revisions
4) API /healthz returns HTTP 200
5) admin URL enforces auth challenge when IAP/auth is expected
6) IAP backend is enabled when admin auth is expected
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
}

pass() {
  echo "PASS: $*"
}

tfvars_read_string() {
  local key="$1"
  local file="$2"
  awk -F= -v key="$key" '
    $1 ~ "^[[:space:]]*" key "[[:space:]]*$" {
      value = $2
      sub(/^[[:space:]]*/, "", value)
      sub(/[[:space:]]*#.*/, "", value)
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      print value
      exit
    }
  ' "$file"
}

normalize_bool() {
  local raw="$1"
  raw="$(echo "$raw" | tr '[:upper:]' '[:lower:]')"
  case "$raw" in
    true|1|yes|y) echo "true" ;;
    false|0|no|n) echo "false" ;;
    *) echo "" ;;
  esac
}

terraform_output_raw() {
  local key="$1"
  terraform -chdir="$TERRAFORM_DIR" output -raw "$key" 2>/dev/null
}

for arg in "$@"; do
  case "$arg" in
    --tfvars=*) TFVARS_PATH="${arg#*=}" ;;
    --terraform-dir=*) TERRAFORM_DIR="${arg#*=}" ;;
    --project-id=*) PROJECT_ID="${arg#*=}" ;;
    --region=*) REGION="${arg#*=}" ;;
    --api-url=*) API_URL="${arg#*=}" ;;
    --admin-url=*) ADMIN_URL="${arg#*=}" ;;
    --expect-admin-auth=*) EXPECT_ADMIN_AUTH="${arg#*=}" ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument '$arg'"
      ;;
  esac
done

for cmd in terraform gcloud curl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "required command not found: $cmd"
  fi
done

if [ ! -f "$TFVARS_PATH" ]; then
  fail "tfvars file not found: $TFVARS_PATH"
fi

if [ ! -d "$TERRAFORM_DIR" ]; then
  fail "terraform directory not found: $TERRAFORM_DIR"
fi

./scripts/validate_tfvars_required.sh \
  "$TFVARS_PATH" \
  project_id \
  api_domain \
  admin_domain

if [ -z "$PROJECT_ID" ]; then
  PROJECT_ID="$(tfvars_read_string "project_id" "$TFVARS_PATH")"
fi
[ -n "$PROJECT_ID" ] || fail "project_id could not be resolved"

if [ -z "$REGION" ]; then
  REGION="$(tfvars_read_string "region" "$TFVARS_PATH")"
fi
REGION="${REGION:-us-central1}"

if [ -z "$API_URL" ]; then
  API_URL="$(terraform_output_raw "api_base_url" || true)"
fi
if [ -z "$API_URL" ]; then
  API_URL="https://$(tfvars_read_string "api_domain" "$TFVARS_PATH")"
fi
[ -n "$API_URL" ] || fail "api url could not be resolved"

if [ -z "$ADMIN_URL" ]; then
  ADMIN_URL="$(terraform_output_raw "admin_base_url" || true)"
fi
if [ -z "$ADMIN_URL" ]; then
  ADMIN_URL="https://$(tfvars_read_string "admin_domain" "$TFVARS_PATH")"
fi
[ -n "$ADMIN_URL" ] || fail "admin url could not be resolved"

if [ -z "$EXPECT_ADMIN_AUTH" ]; then
  EXPECT_ADMIN_AUTH="$(tfvars_read_string "enable_admin_iap" "$TFVARS_PATH")"
fi
EXPECT_ADMIN_AUTH="$(normalize_bool "${EXPECT_ADMIN_AUTH:-true}")"
EXPECT_ADMIN_AUTH="${EXPECT_ADMIN_AUTH:-true}"

if ! gcloud projects describe "$PROJECT_ID" >/dev/null 2>&1; then
  fail "unable to access project '$PROJECT_ID' with active gcloud credentials"
fi
pass "gcloud can access project '$PROJECT_ID'"

lb_ip="$(terraform_output_raw "load_balancer_ip" || true)"
[ -n "$lb_ip" ] || fail "terraform output load_balancer_ip unavailable (run terraform init/apply in $TERRAFORM_DIR)"
pass "terraform load balancer IP = $lb_ip"

smtp_egress_ip="$(terraform_output_raw "smtp_egress_ip" || true)"
[ -n "$smtp_egress_ip" ] || fail "terraform output smtp_egress_ip unavailable"
pass "terraform SMTP egress IP = $smtp_egress_ip"

api_service_name="$(terraform_output_raw "api_service_name" || true)"
api_service_name="${api_service_name:-aegis-api}"
admin_service_name="$(terraform_output_raw "admin_service_name" || true)"
admin_service_name="${admin_service_name:-aegis-admin-dashboard}"

api_ready_revision="$(
  gcloud run services describe "$api_service_name" \
    --region "$REGION" \
    --project "$PROJECT_ID" \
    --format='value(status.latestReadyRevisionName)' \
    2>/dev/null || true
)"
[ -n "$api_ready_revision" ] || fail "Cloud Run service '$api_service_name' has no ready revision"
pass "Cloud Run API service ready revision = $api_ready_revision"

admin_ready_revision="$(
  gcloud run services describe "$admin_service_name" \
    --region "$REGION" \
    --project "$PROJECT_ID" \
    --format='value(status.latestReadyRevisionName)' \
    2>/dev/null || true
)"
[ -n "$admin_ready_revision" ] || fail "Cloud Run service '$admin_service_name' has no ready revision"
pass "Cloud Run admin service ready revision = $admin_ready_revision"

api_health_code="$(curl -sS -o /dev/null -w '%{http_code}' "${API_URL%/}/healthz")"
[ "$api_health_code" = "200" ] || fail "API health check failed for ${API_URL%/}/healthz (HTTP $api_health_code)"
pass "API /healthz returned HTTP 200 (${API_URL%/}/healthz)"

admin_code="$(curl -sS -o /dev/null -w '%{http_code}' "${ADMIN_URL%/}/")"
if [ "$EXPECT_ADMIN_AUTH" = "true" ]; then
  case "$admin_code" in
    302|401|403) pass "admin URL enforces auth challenge (HTTP $admin_code)" ;;
    *) fail "expected admin auth challenge (302/401/403), got HTTP $admin_code from ${ADMIN_URL%/}/" ;;
  esac
else
  case "$admin_code" in
    2*|3*|4*) pass "admin URL reachable without strict auth requirement (HTTP $admin_code)" ;;
    *) fail "admin URL returned unexpected HTTP $admin_code from ${ADMIN_URL%/}/" ;;
  esac
fi

if [ "$EXPECT_ADMIN_AUTH" = "true" ]; then
  iap_backend_service_name="$(terraform_output_raw "iap_backend_service_name" || true)"
  [ -n "$iap_backend_service_name" ] || fail "terraform output iap_backend_service_name unavailable"

  iap_enabled="$(
    gcloud compute backend-services describe "$iap_backend_service_name" \
      --project "$PROJECT_ID" \
      --global \
      --format='value(iap.enabled)' \
      2>/dev/null || true
  )"
  iap_enabled="$(normalize_bool "$iap_enabled")"
  [ "$iap_enabled" = "true" ] || fail "IAP not enabled on backend service '$iap_backend_service_name'"
  pass "IAP enabled on backend service '$iap_backend_service_name'"
fi

echo "ok: gcp deployment verification checks passed"
