# AEGIS Observability Snapshot

**Date:** 2026-02-22
**Environment:** Production (`<GCP_PROJECT_ID>`, `us-central1`)
**Monitoring provider:** GCP Cloud Monitoring

---

## Dashboard: "AEGIS Operations — prod"

GCP Console URL:
```
https://console.cloud.google.com/monitoring/dashboards?project=<GCP_PROJECT_ID>
```

The dashboard is provisioned by Terraform (`terraform/infra/main.tf` →
`google_monitoring_dashboard.aegis`) using the template
`terraform/infra/monitoring_dashboard.json`.

### Dashboard Layout (11 tiles, mosaic 12-column grid)

```
Row 0  [0,0]──────────── API Request Rate (req/s, by response code class)
       [6,0]──────────── API Error Rate 5xx/s
Row 4  [0,4]──────────── API Request Latency p50 + p99 (ms)
       [6,4]──────────── Cloud Run Instance Count (active/idle)
Row 8  [0,8]──[4,8]──[8,8]── Cloud SQL CPU % | Disk % | Active Connections
Row 12 [0,12]──────────── Cloud Run Memory Utilisation — All Services
       [6,12]──────────── Sidecar 5xx Error Rate (all Cloud Run services, by name)
Row 16 [0,16]──────────── Pipeline Failure Events (log-based, bar)
       [6,16]──────────── Stuck Study SLA Alerts (log-based, bar)
```

### Metric Sources

| Tile | Metric type | Notes |
|------|------------|-------|
| API Request Rate | `run.googleapis.com/request_count` | Grouped by `response_code_class`; ALIGN_RATE 60s |
| API Error Rate 5xx/s | `run.googleapis.com/request_count` | Filter: `response_code_class="5xx"` |
| API Latency | `run.googleapis.com/request_latencies` | p50 + p99 per 60s; filter: API service only |
| Cloud Run Instances | `run.googleapis.com/container/instance_count` | Grouped by `state` (active/idle) |
| Cloud SQL CPU | `cloudsql.googleapis.com/database/cpu/utilization` | ALIGN_MEAN 60s |
| Cloud SQL Disk | `cloudsql.googleapis.com/database/disk/utilization` | ALIGN_MEAN 60s |
| Cloud SQL Connections | `cloudsql.googleapis.com/database/postgresql/num_backends` | ALIGN_MAX 60s |
| Cloud Run Memory | `run.googleapis.com/container/memory/utilizations` | p99 REDUCE_MAX across all services |
| Sidecar 5xx Rate | `run.googleapis.com/request_count` (5xx) | All Cloud Run services, grouped by service name |
| Pipeline Failures | `logging.googleapis.com/user/<GCP_PROJECT_ID>/aegis-prod-pipeline-failures` | Log-based metric |
| Stuck Study SLA | `logging.googleapis.com/user/<GCP_PROJECT_ID>/aegis-prod-study-stuck` | Log-based metric |

---

## Alert Policies (9 total)

| Policy | Condition | Threshold | Duration | Severity |
|--------|-----------|-----------|----------|----------|
| AEGIS API 5xx rate high | `run.googleapis.com/request_count` 5xx | > 10/s | 5 min | warning |
| AEGIS API p99 latency high | `run.googleapis.com/request_latencies` p99 | > 5 000 ms | 5 min | warning |
| AEGIS API uptime check failing | `/healthz` uptime check | < 1 pass | 2 min | critical |
| AEGIS Cloud SQL CPU high | `database/cpu/utilization` | > 80% | 10 min | warning |
| AEGIS Cloud SQL disk usage high | `database/disk/utilization` | > 85% | 5 min | critical |
| AEGIS Cloud SQL connections high | `database/postgresql/num_backends` | > 80 | 5 min | warning |
| AEGIS Cloud Run memory high | `container/memory/utilizations` p99 | > 90% | 10 min | warning |
| AEGIS study stuck in pipeline > 30 min | `aegis-prod-study-stuck` log metric | > 0 events | immediate | warning |
| AEGIS pipeline step failure | `aegis-prod-pipeline-failures` log metric | > 3 events | 5 min | critical |

All alerts notify the `alert_email` recipient via the `aegis-prod-ops-email` notification channel.

---

## Log-Based Metrics

Defined in `terraform/infra/main.tf` as `google_logging_metric` resources.

### `aegis-prod-pipeline-failures`

```
resource.type="cloud_run_revision"
AND resource.labels.service_name="aegis-prod-api"
AND textPayload:"pipeline: send failure alert"
```

Fires when the Go API's `notifyPipelineFailure()` function logs a pipeline step failure
(handler/pipeline.go). Emits `DELTA INT64` per log entry.

### `aegis-prod-study-stuck`

```
resource.type="cloud_run_revision"
AND resource.labels.service_name="aegis-prod-api"
AND textPayload:"sla: sent alert"
```

Fires when the SLA background scheduler (`sla/scheduler.go`) detects studies stuck in
a pipeline stage beyond the configured threshold and logs the alert. Emits `DELTA INT64`
per log entry (one entry per SLA check cycle that found stuck studies).

---

## Runbooks

- `docs/runbooks/alert-response.md` — per-alert incident response steps

## Terraform Reference

- Dashboard JSON: `terraform/infra/monitoring_dashboard.json`
- Alert policies + log metrics: `terraform/infra/main.tf` lines ~1320–1680
- Monitoring overview: `terraform/monitoring/README.md`
