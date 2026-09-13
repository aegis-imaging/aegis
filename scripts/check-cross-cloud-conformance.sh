#!/usr/bin/env bash
set -euo pipefail

missing=0

check_file() {
  local file="$1"
  local label="$2"
  if [[ ! -f "$file" ]]; then
    echo "❌ Missing file: $label ($file)"
    missing=1
  fi
}

check_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing pattern: $label ($file :: $pattern)"
    missing=1
  fi
}

# Canonical deployment service coverage (AWS workflow as explicit matrix example)
check_pattern ".github/workflows/deploy-aws.yml" 'service: api' "AWS build matrix api"
check_pattern ".github/workflows/deploy-aws.yml" 'service: mcp-server' "AWS build matrix mcp-server"
check_pattern ".github/workflows/deploy-aws.yml" 'service: dimse-receiver' "AWS build matrix dimse-receiver"
check_pattern ".github/workflows/deploy-aws.yml" 'deploy_if_exists "aegis-landing"' "AWS conditional landing deployment"

# Azure deploy coverage signals
check_pattern ".github/workflows/deploy-azure.yml" 'service: mcp-server' "Azure build mcp-server"
check_pattern ".github/workflows/deploy-azure.yml" 'service: dimse-receiver' "Azure build dimse-receiver"
check_pattern ".github/workflows/deploy-azure.yml" 'Deploy — mcp-server' "Azure deploy mcp-server"

# CI guardrail/check coverage
check_pattern ".github/workflows/ci.yml" 'run: ./scripts/check-auth-guardrails.sh' "CI auth guardrails"
check_pattern ".github/workflows/ci.yml" 'run: ./scripts/check-edge-security-parity.sh' "CI edge parity"
check_pattern ".github/workflows/ci.yml" 'run: ./scripts/check-monitoring-alert-parity.sh' "CI monitoring parity"
check_pattern ".github/workflows/ci.yml" 'run: ./scripts/check-hipaa-day1-controls.sh' "CI HIPAA baseline"

# Health-check signals in cloud IaC
check_pattern "terraform/aws/monitoring.tf" 'resource "aws_route53_health_check" "api_healthz"' "AWS API health check"
check_pattern "terraform/infra/main.tf" 'google_monitoring_alert_policy" "api_uptime"' "GCP API uptime alert"
check_pattern "terraform/azure/app_gateway_waf.tf" 'path                                      = "/healthz"' "Azure App Gateway API probe"

# Ensure enforcement scripts exist
check_file "scripts/check-aws-deploy-parity.sh" "AWS deploy parity checker"
check_file "scripts/check-auth-guardrails.sh" "Auth guardrail checker"
check_file "scripts/check-edge-security-parity.sh" "Edge parity checker"
check_file "scripts/check-monitoring-alert-parity.sh" "Monitoring parity checker"
check_file "scripts/check-hipaa-day1-controls.sh" "HIPAA day-1 checker"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ Cross-cloud conformance check failed"
  exit 1
fi

echo "✅ Cross-cloud conformance check passed"
