#!/usr/bin/env bash
set -euo pipefail

missing=0

check() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing Terraform security baseline: $label ($file :: $pattern)"
    missing=1
  fi
}

# GCP baseline signals
check "terraform/infra/main.tf" 'resource "google_compute_security_policy" "api"' "GCP Cloud Armor policy"
check "terraform/infra/main.tf" 'google_secret_manager_secret' "GCP Secret Manager usage"

# AWS baseline signals
check "terraform/aws/monitoring.tf" 'resource "aws_wafv2_web_acl" "api"' "AWS WAF policy"
check "terraform/aws/ses.tf" 'kms_key_id' "AWS KMS-backed secrets"
check "terraform/aws/main.tf" 'manage_master_user_password' "AWS RDS managed master password setting"

# Azure baseline signals
check "terraform/azure/main.tf" 'resource "azurerm_key_vault" "main"' "Azure Key Vault"
check "terraform/azure/main.tf" 'public_network_access_enabled = false' "Azure PostgreSQL private networking"
check "terraform/azure/main.tf" 'min_tls_version                 = "TLS1_2"' "Azure storage TLS minimum"
check "terraform/azure/app_gateway_waf.tf" 'resource "azurerm_web_application_firewall_policy" "app_gateway"' "Azure WAF policy resource"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ Terraform security baseline check failed"
  exit 1
fi

echo "✅ Terraform security baseline check passed"
