#!/usr/bin/env bash
set -euo pipefail

missing=0

check_file() {
  local file="$1"
  local label="$2"
  if [[ ! -f "$file" ]]; then
    echo "❌ Missing required artifact: $label ($file)"
    missing=1
  fi
}

check_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing control signal: $label ($file :: $pattern)"
    missing=1
  fi
}

# Required runbooks / planning artifacts
check_file "docs/runbooks/incident-response.md" "Incident response runbook"
check_file "docs/runbooks/secret-rotation.md" "Secret rotation runbook"
check_file "docs/runbooks/alert-response.md" "Alert response runbook"
check_file "docs/planning/phase-5-hipaa-day1-controls/hipaa-day1-control-checklist.md" "Phase 5 HIPAA checklist"

# Auth guardrails (non-dev)
check_file "scripts/check-auth-guardrails.sh" "Auth guardrails checker"
check_pattern "terraform/infra/main.tf" 'name  = "AUTH_ENABLED"' "GCP auth enabled env"
check_pattern "terraform/aws/main.tf" 'AUTH_ENABLED", value = "true"' "AWS auth enabled"
check_pattern "terraform/azure/container_apps.tf" 'name  = "AUTH_ENABLED"' "Azure auth enabled env"

# TLS / transport / network controls
check_pattern "terraform/azure/main.tf" 'min_tls_version                 = "TLS1_2"' "Azure storage TLS 1.2"
check_pattern "terraform/azure/container_apps.tf" 'allow_insecure_connections = false' "Azure container apps disallow insecure ingress"
check_pattern "terraform/azure/main.tf" 'public_network_access_enabled = false' "Azure PostgreSQL private access"

# Edge controls + monitoring baseline
check_file "scripts/check-edge-security-parity.sh" "Edge parity checker"
check_file "scripts/check-monitoring-alert-parity.sh" "Monitoring parity checker"

# Secret management controls
check_pattern "terraform/infra/main.tf" 'google_secret_manager_secret' "GCP Secret Manager usage"
check_pattern "terraform/aws/ses.tf" 'kms_key_id' "AWS KMS-backed secret usage"
check_pattern "terraform/azure/main.tf" 'resource "azurerm_key_vault" "main"' "Azure Key Vault usage"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ HIPAA day-1 baseline control check failed"
  exit 1
fi

echo "✅ HIPAA day-1 baseline control check passed"
