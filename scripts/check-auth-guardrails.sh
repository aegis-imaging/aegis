#!/usr/bin/env bash
set -euo pipefail

missing=0

require_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing auth guardrail: $label ($file :: $pattern)"
    missing=1
  fi
}

# GCP infra auth invariants
require_pattern "terraform/infra/main.tf" 'name  = "AUTH_ENABLED"' "GCP AUTH_ENABLED env var"
require_pattern "terraform/infra/main.tf" 'value = "true"' "GCP AUTH_ENABLED=true"
require_pattern "terraform/infra/main.tf" 'name  = "AUTH_PROVIDER"' "GCP AUTH_PROVIDER env var"
require_pattern "terraform/infra/main.tf" 'value = "iap"' "GCP AUTH_PROVIDER=iap"

# AWS infra auth invariants
require_pattern "terraform/aws/main.tf" 'AUTH_ENABLED", value = "true"' "AWS AUTH_ENABLED=true"
require_pattern "terraform/aws/main.tf" 'AUTH_PROVIDER", value = "aws"' "AWS AUTH_PROVIDER=aws"

# Azure infra auth invariants
require_pattern "terraform/azure/container_apps.tf" 'name  = "AUTH_ENABLED"' "Azure AUTH_ENABLED env var"
require_pattern "terraform/azure/container_apps.tf" 'value = "true"' "Azure AUTH_ENABLED=true"
require_pattern "terraform/azure/container_apps.tf" 'name  = "AUTH_PROVIDER"' "Azure AUTH_PROVIDER env var"
require_pattern "terraform/azure/container_apps.tf" 'value = "azure"' "Azure AUTH_PROVIDER=azure"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ Authentication guardrail check failed"
  exit 1
fi

echo "✅ Authentication guardrail check passed"
