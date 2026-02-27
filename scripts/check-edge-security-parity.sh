#!/usr/bin/env bash
set -euo pipefail

missing=0

check_pattern() {
  local file="$1"
  local pattern="$2"
  local description="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing: $description ($file :: $pattern)"
    missing=1
  fi
}

check_pattern "terraform/infra/main.tf" "resource \"google_compute_security_policy\" \"api\"" "GCP Cloud Armor API policy"
check_pattern "terraform/aws/monitoring.tf" "resource \"aws_wafv2_web_acl\" \"api\"" "AWS WAF v2 API Web ACL"
check_pattern "terraform/azure/app_gateway_waf.tf" "resource \"azurerm_web_application_firewall_policy\" \"app_gateway\"" "Azure WAF policy resource"
check_pattern "terraform/azure/app_gateway_waf.tf" "resource \"azurerm_application_gateway\" \"main\"" "Azure Application Gateway resource"
check_pattern "terraform/azure/variables.tf" "variable \"enable_application_gateway_waf\"" "Azure WAF feature toggle variable"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ Edge security parity check failed"
  exit 1
fi

echo "✅ Edge security parity check passed"
