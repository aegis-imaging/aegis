#!/usr/bin/env bash
set -euo pipefail

SKIP_DOCKER=0
SKIP_ADC=0

for arg in "$@"; do
  case "$arg" in
    --skip-docker) SKIP_DOCKER=1 ;;
    --skip-adc) SKIP_ADC=1 ;;
    -h|--help)
      cat <<'USAGE'
Usage: ./scripts/gcp_preflight.sh [--skip-docker] [--skip-adc]

Checks local prerequisites for AEGIS GCP installation:
- gcloud CLI installed and authenticated
- terraform CLI installed
- docker CLI installed (unless --skip-docker)
- active gcloud project configured
- application-default credentials available (unless --skip-adc)
USAGE
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      exit 1
      ;;
  esac
done

require_cmd() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    echo "error: required command not found: $name" >&2
    exit 1
  fi
}

require_cmd gcloud
require_cmd terraform
if [ "$SKIP_DOCKER" -eq 0 ]; then
  require_cmd docker
fi

active_account="$(gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null || true)"
if [ -z "$active_account" ]; then
  echo "error: no active gcloud account. Run: gcloud auth login" >&2
  exit 1
fi

project_id="$(gcloud config get-value project 2>/dev/null || true)"
if [ -z "$project_id" ] || [ "$project_id" = "(unset)" ]; then
  echo "error: no active gcloud project. Run: gcloud config set project <PROJECT_ID>" >&2
  exit 1
fi

if [ "$SKIP_ADC" -eq 0 ]; then
  if ! gcloud auth application-default print-access-token >/dev/null 2>&1; then
    echo "error: application-default credentials not configured. Run: gcloud auth application-default login" >&2
    exit 1
  fi
fi

gcloud_version="$(gcloud --version | head -n 1)"
terraform_version="$(terraform version | head -n 1)"

if [ "$SKIP_DOCKER" -eq 0 ]; then
  docker_version="$(docker --version)"
else
  docker_version="skipped"
fi

echo "ok: gcp preflight checks passed"
echo "active_account=${active_account}"
echo "project_id=${project_id}"
echo "${gcloud_version}"
echo "${terraform_version}"
echo "${docker_version}"
