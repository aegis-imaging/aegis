# AEGIS Alert Response Runbook

This runbook maps each Terraform-provisioned alert policy to a triage procedure.
All alerts fire to the email address set in `alert_email` (terraform/infra/terraform.tfvars).

## Cross-Cloud Deployment Ops Entry Points

Use this quick index to jump to the right deployment or failure surface.

### AWS

- **App deploy workflow**: GitHub Actions → `Deploy to AWS`
- **Terraform workflow**: GitHub Actions → `Terraform AWS`
- **Terraform failure signal**: GitHub Actions → `Terraform AWS Failure Alert`
- **Apply approval gate**: GitHub Environment `aws-prod`

### Azure

- **App deploy workflow**: GitHub Actions → `Deploy to Azure`
- **Terraform workflow**: GitHub Actions → `Terraform Azure`
- **Terraform failure signal**: GitHub Actions → `Terraform Azure Failure Alert`
- **Apply approval gate**: GitHub Environment `azure-prod`

### GCP

- **App deploy pipeline**: Cloud Build trigger using `cloudbuild.yaml`
- **Terraform infra pipeline**: Cloud Build trigger using `cloudbuild.terraform.yaml`
- **Primary health check endpoint**: `https://api.aegisimaging.ai/healthz`

## AWS Terraform Apply Approval Gate (`aws-prod`)

Use this section whenever the `Terraform AWS` workflow is waiting for deployment approval.

### Approve a pending apply

1. Open **Actions → Terraform AWS** and select the most recent run on `develop`.
2. Confirm the `Terraform Plan (AWS)` job completed successfully.
3. Open the pending deployment card for environment `aws-prod`.
4. Click **Review deployments** and then **Approve and deploy**.
5. Monitor `Terraform Apply (AWS)` until completion.

### Reject a pending apply

1. Open the pending deployment card for `aws-prod`.
2. Click **Review deployments** and choose **Reject**.
3. Add a short reason (for example: unexpected resource replacement).
4. Open a follow-up issue/PR to fix the plan before re-running.

### Fast checks before approving

- Verify the run is from `develop` and repository `aegis-imaging/aegis`.
- Open `tfplan.txt` artifact and confirm no unexpected destructive changes.
- Confirm the triggering commit/PR matches the intended infrastructure change.

### If apply fails

- Open **Actions → Terraform AWS Failure Alert** for the failure summary and direct run URL.
- Triage from the failed step in `Terraform Apply (AWS)` first.
- If state lock-related, wait for lock expiry or clear lock only with operator approval.

## Azure Terraform Apply Approval Gate (`azure-prod`)

Use this section whenever the `Terraform Azure` workflow is waiting for deployment approval.

### Approve a pending apply

1. Open **Actions → Terraform Azure** and select the most recent run on `develop`.
2. Confirm the `Terraform Plan (Azure)` job completed successfully.
3. Open the pending deployment card for environment `azure-prod`.
4. Click **Review deployments** and then **Approve and deploy**.
5. Monitor `Terraform Apply (Azure)` until completion.

### Reject a pending apply

1. Open the pending deployment card for `azure-prod`.
2. Click **Review deployments** and choose **Reject**.
3. Add a short reason (for example: unexpected resource replacement).
4. Open a follow-up issue/PR to fix the plan before re-running.

### Fast checks before approving

- Verify the run is from `develop` and repository `aegis-imaging/aegis`.
- Open `tfplan.txt` artifact and confirm no unexpected destructive changes.
- Confirm the triggering commit/PR matches the intended infrastructure change.

### If apply fails

- Open **Actions → Terraform Azure Failure Alert** for the failure summary and direct run URL.
- Triage from the failed step in `Terraform Apply (Azure)` first.
- If state lock-related, wait for lock expiry or clear lock only with operator approval.

---

## API 5xx Rate High

**Alert**: `AEGIS API 5xx rate high`
**Threshold**: >10 errors/min for 5 minutes
**Severity**: Warning

### Triage

1. **Check Cloud Run logs**

   ```bash
   gcloud logging read \
     'resource.type="cloud_run_revision" AND resource.labels.service_name="aegis-api" AND severity>=ERROR' \
     --project=aegis-prod-488119 --limit=50 --format=json | jq '.[].textPayload'
   ```

2. **Check `/healthz`**

   ```bash
   curl -s https://api.aegisimaging.ai/healthz | jq .
   ```

   Look for `"database":"unhealthy"` or sidecar services degraded.

3. **Check DB connectivity** — if `"database":"unhealthy"`, see [Cloud SQL CPU alert](#cloud-sql-cpu-high) below.

4. **Roll back if recent deploy**

   ```bash
   gcloud run services update-traffic aegis-api \
     --to-revisions=PREV_REVISION=100 \
     --region=us-central1 --project=aegis-prod-488119
   ```

5. **Scale out if overloaded**

   ```bash
   gcloud run services update aegis-api \
     --max-instances=50 \
     --region=us-central1 --project=aegis-prod-488119
   ```

---

## API Uptime Check Failing

**Alert**: `AEGIS API uptime check failing`
**Threshold**: Check failed for 2 consecutive minutes
**Severity**: Critical

### Triage

1. **Verify manually**

   ```bash
   curl -v https://api.aegisimaging.ai/healthz
   ```

2. **Check load balancer**

   ```bash
   gcloud compute backend-services describe aegis-api-backend \
     --global --project=aegis-prod-488119 --format=json | jq '.backends[].balancingMode'
   ```

3. **Check Cloud Run service status**

   ```bash
   gcloud run services describe aegis-api \
     --region=us-central1 --project=aegis-prod-488119 --format=json | jq '.status.conditions'
   ```

4. **Check SSL certificate** — an expired cert will cause uptime check failures:

   ```bash
   gcloud compute ssl-certificates list --project=aegis-prod-488119
   ```

---

## API p99 Latency High

**Alert**: `AEGIS API p99 latency high`
**Threshold**: p99 > 5 seconds for 5 minutes
**Severity**: Warning

### Triage

1. **Check for slow queries** — look for `"slow query"` in API logs.

2. **Check cold start rate** — low `min-instances` can cause spikes.

   ```bash
   gcloud run services update aegis-api \
     --min-instances=2 \
     --region=us-central1 --project=aegis-prod-488119
   ```

3. **Check sidecar timeouts** — if a sidecar (defacing, PHI scan) is slow, requests that
   trigger processing will inherit that latency.

4. **Check DB slow query log** — enable in Cloud SQL flags:

   ```sql
   SHOW log_min_duration_statement;
   -- Set to 1000 (ms) to log queries slower than 1s
   ```

---

## Cloud SQL CPU High

**Alert**: `AEGIS Cloud SQL CPU high`
**Threshold**: CPU > 80% for 10 minutes
**Severity**: Warning

### Triage

1. **Identify top queries**

   ```bash
   gcloud sql connect aegis-prod --user=aegis --project=aegis-prod-488119
   ```

   Then in psql:
   ```sql
   SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
   FROM pg_stat_activity
   WHERE (now() - pg_stat_activity.query_start) > interval '5 seconds'
   ORDER BY duration DESC;
   ```

2. **Kill runaway queries**

   ```sql
   SELECT pg_terminate_backend(<pid>);
   ```

3. **Check for missing indexes** — look for sequential scans on large tables in `pg_stat_user_tables`.

4. **Scale up instance** if legitimate load:

   In `terraform/infra/terraform.tfvars`, increase `db_tier` (e.g., `db-g1-small` → `db-n1-standard-2`), then:

   ```bash
   cd terraform/infra && terraform apply -target=google_sql_database_instance.aegis
   ```

---

## Cloud SQL Disk Usage High

**Alert**: `AEGIS Cloud SQL disk usage high`
**Threshold**: Disk > 85% for 5 minutes
**Severity**: Critical

### Immediate action

Enable automatic storage increase in the GCP Console:

1. Go to **Cloud SQL → aegis-prod → Edit**
2. Under **Storage**, enable **Enable automatic storage increases**
3. Save

Or via Terraform — update `disk_autoresize = true` in `main.tf` and apply.

### Investigation

```sql
-- Find largest tables
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;
```

**Common causes:**
- `audit_trail` table accumulating entries — consider partitioning or archiving entries older than 90 days
- DICOM metadata stored in DB instead of GCS (should not happen in AEGIS architecture)

---

## Cloud SQL Active Connections High

**Alert**: `AEGIS Cloud SQL connections high`
**Threshold**: >80 active connections for 5 minutes
**Severity**: Warning

### Triage

1. **Check Cloud Run max-instances** — each instance holds a connection pool.
   Reducing `max-instances` reduces peak connection usage.

2. **Check for connection leaks** in application logs — look for DB errors or
   unclosed transactions.

3. **Enable PgBouncer** (long-term) — pool connections between Cloud Run instances
   and Cloud SQL to reduce connection overhead.

4. **Increase `max_connections`** in Cloud SQL flags (default: 100 for shared-core tiers):

   In `main.tf`, add to the `google_sql_database_instance` resource:
   ```hcl
   database_flags {
     name  = "max_connections"
     value = "200"
   }
   ```
   Then `terraform apply`.

---

## Escalation

If an alert cannot be resolved within 30 minutes:

1. Check the [GCP Status Page](https://status.cloud.google.com) for service outages
2. Create a GCP support ticket if a managed service is at fault
3. Engage on-call engineer via PagerDuty / Slack `#aegis-incidents`
