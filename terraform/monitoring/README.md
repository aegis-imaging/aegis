# AEGIS Monitoring

Cloud Monitoring resources for AEGIS are defined in `terraform/infra/main.tf` under the `# --- Monitoring baseline ---` and `# --- Log-based metrics ---` sections, with the dashboard layout in `terraform/infra/monitoring_dashboard.json`.

## Dashboard

The GCP Cloud Monitoring dashboard is provisioned automatically when `enable_monitoring_alerts = true` in `terraform/infra/terraform.tfvars`. It covers 11 tiles across 5 rows:

| Row | Tile | Metric |
|-----|------|--------|
| 0 | API Request Rate | `run.googleapis.com/request_count` (by response code class) |
| 0 | API Error Rate (5xx/s) | `run.googleapis.com/request_count` filtered to `5xx` |
| 4 | API Request Latency | `run.googleapis.com/request_latencies` (p50 + p99) |
| 4 | Cloud Run Instance Count | `run.googleapis.com/container/instance_count` |
| 8 | Cloud SQL CPU | `cloudsql.googleapis.com/database/cpu/utilization` |
| 8 | Cloud SQL Disk | `cloudsql.googleapis.com/database/disk/utilization` |
| 8 | Cloud SQL Active Connections | `cloudsql.googleapis.com/database/postgresql/num_backends` |
| 12 | Cloud Run Memory — All Services | `run.googleapis.com/container/memory/utilizations` |
| 12 | Sidecar 5xx Error Rate | `run.googleapis.com/request_count` (all services, grouped by name) |
| 16 | Pipeline Failure Events | Log-based metric: `aegis-{env}-pipeline-failures` |
| 16 | Stuck Study SLA Alerts | Log-based metric: `aegis-{env}-study-stuck` |

**GCP Console URL** (production):
```
https://console.cloud.google.com/monitoring/dashboards?project=aegis-prod-488120
```

## Alert Policies

| Policy | Threshold | Severity |
|--------|-----------|----------|
| `api_5xx_rate` | > 10 errors/s for 5 min | warning |
| `api_latency` | p99 > 5 000 ms for 5 min | warning |
| `api_uptime` | `/healthz` failing for 2 min | critical |
| `cloudsql_cpu` | > 80% for 10 min | warning |
| `cloudsql_disk` | > 85% for 5 min | critical |
| `cloudsql_connections` | > 80 connections for 5 min | warning |
| `cloud_run_memory` | > 90% for 10 min | warning |
| `study_stuck_alert` | Any SLA stuck-study log event | warning |
| `pipeline_failure_alert` | > 3 pipeline failure log events in 5 min | critical |

All alert policies email `alert_email` (configured in `terraform.tfvars`).

## Log-Based Metrics

Two custom log-based metrics capture AEGIS application events from Cloud Run stdout:

| Metric | Log filter |
|--------|------------|
| `aegis-{env}-pipeline-failures` | `textPayload:"pipeline: send failure alert"` from the API Cloud Run service |
| `aegis-{env}-study-stuck` | `textPayload:"sla: sent alert"` from the API Cloud Run service |

These feed both dashboard tiles and alert policies.

## Runbooks

See `docs/runbooks/alert-response.md` for incident response guidance per alert.
