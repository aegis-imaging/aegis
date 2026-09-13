# Phase 4 Critical Alert Catalog

Source plan: `implementation-plan.MD` (Phase 4)

## Required Alert Families (Cross-Cloud)

1. API availability (`/healthz`/equivalent)
2. API latency
3. Pipeline failures
4. Stuck studies
5. Destination probe failures
6. DIMSE dead-letter / retry-queue risk signal

## Canonical Resource Mapping

| Family | GCP | AWS | Azure |
|---|---|---|---|
| API availability | `google_monitoring_alert_policy.api_uptime` | `aws_cloudwatch_metric_alarm.api_uptime` | `azurerm_monitor_metric_alert.api_5xx` + availability via platform checks |
| API latency | `google_monitoring_alert_policy.api_latency` | `aws_cloudwatch_metric_alarm.alb_latency` | `azurerm_monitor_metric_alert.api_latency` |
| Pipeline failures | `google_monitoring_alert_policy.pipeline_failure_alert` | `aws_cloudwatch_metric_alarm.pipeline_failure_alert` | `azurerm_monitor_scheduled_query_rules_alert_v2.pipeline_failures` |
| Stuck studies | `google_monitoring_alert_policy.study_stuck_alert` | `aws_cloudwatch_metric_alarm.study_stuck_alert` | `azurerm_monitor_scheduled_query_rules_alert_v2.stuck_studies` |
| Destination probe failures | `google_monitoring_alert_policy.destination_probe_failure_alert` | `aws_cloudwatch_metric_alarm.destination_probe_failure_alert` | `azurerm_monitor_scheduled_query_rules_alert_v2.destination_probe_failures` |
| DIMSE dead-letter risk | `google_monitoring_alert_policy.dimse_dead_letter_alert` | `aws_cloudwatch_metric_alarm.dimse_dead_letter_alert` | `azurerm_monitor_scheduled_query_rules_alert_v2.dimse_dead_letter` |

## Validation

- Enforced in CI by `scripts/check-monitoring-alert-parity.sh`.
- Terraform validation required for `terraform/infra`, `terraform/aws`, `terraform/azure`.
