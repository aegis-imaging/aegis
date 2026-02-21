#!/usr/bin/env bash
set -euo pipefail

PROVIDER="gcp"
TERRAFORM_DIR=""
BASE_URL=""
PROJECT_SLUG="default"
TIMEOUT_SECONDS="30"
PIPELINE_TIMEOUT_SECONDS="45"
REQUIRE_AUTH_CONTEXT="true"
IAP_EMAIL=""
ADMIN_HEADERS=()

usage() {
  cat <<'USAGE'
Usage: ./scripts/cloud_smoke_from_terraform.sh [options]

Options:
  --provider=<gcp|aws>             Default: gcp
  --terraform-dir=<path>           Default: terraform/infra (gcp) or terraform/aws (aws)
  --base-url=<url>                 Override resolved base URL
  --project-slug=<slug>            Default: default
  --timeout=<seconds>              Default: 30
  --pipeline-timeout=<seconds>     Default: 45
  --admin-header=<header:value>    Repeatable; forwarded to cloud_smoke_test.py
  --iap-email=<email>              Convenience for GCP IAP auth header
  --require-auth-context=<bool>    Default: true
  -h, --help                       Show this help

Behavior:
- GCP base URL resolves from terraform output api_base_url.
- AWS base URL resolves from terraform output alb_dns (converted to https://<alb_dns>).
- Fails fast when require-auth-context=true and no admin auth context is provided.
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
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

for arg in "$@"; do
  case "$arg" in
    --provider=*) PROVIDER="${arg#*=}" ;;
    --terraform-dir=*) TERRAFORM_DIR="${arg#*=}" ;;
    --base-url=*) BASE_URL="${arg#*=}" ;;
    --project-slug=*) PROJECT_SLUG="${arg#*=}" ;;
    --timeout=*) TIMEOUT_SECONDS="${arg#*=}" ;;
    --pipeline-timeout=*) PIPELINE_TIMEOUT_SECONDS="${arg#*=}" ;;
    --admin-header=*) ADMIN_HEADERS+=("${arg#*=}") ;;
    --iap-email=*) IAP_EMAIL="${arg#*=}" ;;
    --require-auth-context=*) REQUIRE_AUTH_CONTEXT="${arg#*=}" ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument '$arg'"
      ;;
  esac
done

if [ -z "$TERRAFORM_DIR" ]; then
  case "$PROVIDER" in
    gcp) TERRAFORM_DIR="terraform/infra" ;;
    aws) TERRAFORM_DIR="terraform/aws" ;;
    *) fail "invalid --provider '$PROVIDER' (expected gcp or aws)" ;;
  esac
fi

for cmd in terraform python3; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "required command not found: $cmd"
  fi
done

if [ ! -d "$TERRAFORM_DIR" ]; then
  fail "terraform directory not found: $TERRAFORM_DIR"
fi

if [ -z "$BASE_URL" ]; then
  case "$PROVIDER" in
    gcp)
      BASE_URL="$(terraform -chdir="$TERRAFORM_DIR" output -raw api_base_url 2>/dev/null || true)"
      ;;
    aws)
      alb_dns="$(terraform -chdir="$TERRAFORM_DIR" output -raw alb_dns 2>/dev/null || true)"
      if [ -n "$alb_dns" ]; then
        BASE_URL="https://${alb_dns}"
      fi
      ;;
    *)
      fail "invalid --provider '$PROVIDER' (expected gcp or aws)"
      ;;
  esac
fi

[ -n "$BASE_URL" ] || fail "unable to resolve base URL from terraform outputs (use --base-url to override)"

REQUIRE_AUTH_CONTEXT="$(normalize_bool "$REQUIRE_AUTH_CONTEXT")"
REQUIRE_AUTH_CONTEXT="${REQUIRE_AUTH_CONTEXT:-true}"

if [ "$REQUIRE_AUTH_CONTEXT" = "true" ] && [ "${#ADMIN_HEADERS[@]}" -eq 0 ] && [ -z "$IAP_EMAIL" ]; then
  fail "provide --admin-header or --iap-email (or set --require-auth-context=false)"
fi

cmd=(
  python3
  scripts/cloud_smoke_test.py
  --base-url "$BASE_URL"
  --project-slug "$PROJECT_SLUG"
  --timeout "$TIMEOUT_SECONDS"
  --pipeline-timeout "$PIPELINE_TIMEOUT_SECONDS"
)

for header in "${ADMIN_HEADERS[@]}"; do
  cmd+=(--admin-header "$header")
done

if [ -n "$IAP_EMAIL" ]; then
  cmd+=(--iap-email "$IAP_EMAIL")
fi

echo "==> cloud smoke provider=${PROVIDER} base_url=${BASE_URL}"
"${cmd[@]}"
