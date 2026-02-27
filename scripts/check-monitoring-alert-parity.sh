#!/usr/bin/env bash
set -euo pipefail

missing=0

check() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing: $label ($file :: $pattern)"
    missing=1
  fi
}

# GCP
check "terraform/infra/main.tf" 'google_monitoring_alert_policy" "api_uptime"' "GCP api uptime alert"
check "terraform/infra/main.tf" 'google_monitoring_alert_policy" "api_latency"' "GCP api latency alert"
check "terraform/infra/main.tf" 'google_monitoring_alert_policy" "pipeline_failure_alert"' "GCP pipeline failure alert"
check "terraform/infra/main.tf" 'google_monitoring_alert_policy" "destination_probe_failure_alert"' "GCP destination probe alert"
check "terraform/infra/main.tf" 'google_monitoring_alert_policy" "dimse_dead_letter_alert"' "GCP DIMSE dead-letter alert"

# AWS
check "terraform/aws/monitoring.tf" 'aws_cloudwatch_metric_alarm" "api_uptime"' "AWS api uptime alert"
check "terraform/aws/monitoring.tf" 'aws_cloudwatch_metric_alarm" "alb_latency"' "AWS api latency alert"
check "terraform/aws/monitoring.tf" 'aws_cloudwatch_metric_alarm" "pipeline_failure_alert"' "AWS pipeline failure alert"
check "terraform/aws/monitoring.tf" 'aws_cloudwatch_metric_alarm" "destination_probe_failure_alert"' "AWS destination probe alert"
check "terraform/aws/monitoring.tf" 'aws_cloudwatch_metric_alarm" "dimse_dead_letter_alert"' "AWS DIMSE dead-letter alert"

# Azure
check "terraform/azure/monitoring.tf" 'azurerm_monitor_metric_alert" "api_latency"' "Azure api latency alert"
check "terraform/azure/monitoring.tf" 'azurerm_monitor_scheduled_query_rules_alert_v2" "pipeline_failures"' "Azure pipeline failure alert"
check "terraform/azure/monitoring.tf" 'azurerm_monitor_scheduled_query_rules_alert_v2" "destination_probe_failures"' "Azure destination probe alert"
check "terraform/azure/monitoring.tf" 'azurerm_monitor_scheduled_query_rules_alert_v2" "dimse_dead_letter"' "Azure DIMSE dead-letter alert"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ Monitoring alert parity check failed"
  exit 1
fi

echo "✅ Monitoring alert parity check passed"
