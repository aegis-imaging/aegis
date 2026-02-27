# AEGIS Incident Response Runbook

Standard operating procedures for handling production incidents on the AEGIS platform.

---

## Severity Levels

| Severity | Definition | Target response | Target resolution |
|----------|-----------|-----------------|-------------------|
| P0 — Critical | Full service outage; no studies can be uploaded or approved | 15 min | 4 hours |
| P1 — High | Partial outage; core upload or approval flow broken for some projects | 30 min | 8 hours |
| P2 — Medium | Degraded performance or a non-critical feature unavailable | 2 hours | 48 hours |
| P3 — Low | Minor issue; workaround available; no data at risk | 1 business day | 1 week |

---

## Incident Response Steps

### 1. Detect

Incidents are usually detected via:
- Cloud Monitoring alert email (`ops@aegisimaging.ai`)
- GCP Uptime check failure
- User report to support

### 2. Acknowledge

Reply to the alert thread or create a Slack message in `#aegis-incidents`:

```
[INCIDENT] <brief description> — P<severity>
Time: <UTC timestamp>
Responder: <name>
Status: Investigating
```

### 3. Assess impact

```bash
# Health check
curl -s https://api.aegisimaging.ai/healthz | jq .

# Recent errors
gcloud logging read \
  'resource.type="cloud_run_revision" AND severity>=ERROR' \
  --project=aegis-prod-488119 --limit=20 --format=json | jq '.[].textPayload'

# Study pipeline status — check for stuck studies
curl -s 'https://api.aegisimaging.ai/api/studies?limit=10&status=defacing' \
  -H 'Authorization: Bearer <token>' | jq '.studies[].study_instance_uid'
```

### 4. Contain

Depending on the failure:

- **API down**: Check Cloud Run service health; roll back to previous revision if recent deploy caused it.
- **DB down**: Check Cloud SQL status; verify private IP connectivity from Cloud Run VPC connector.
- **Sidecar down**: Sidecars are optional — the main API continues serving even if defacing/PHI scan is unreachable. Studies will queue in "pending" state.

### 5. Diagnose

See [alert-response.md](./alert-response.md) for per-alert triage procedures.

For stuck studies, use the diagnostics endpoint:

```bash
curl -s https://api.aegisimaging.ai/api/studies/<id>/diagnostics \
  -H 'Authorization: Bearer <token>' | jq .summary
```

For upload attribution drift (restricted projects receiving studies with missing institution linkage), use:

```bash
# Platform-level snapshot (last 7 days)
curl -s 'https://api.aegisimaging.ai/api/stats/institution-attribution?days=7' \
  -H 'Authorization: Bearer <token>' | jq .

# Project-scoped snapshot (required for researcher role)
curl -s 'https://api.aegisimaging.ai/api/stats/institution-attribution?project_id=<project_uuid>&days=7' \
  -H 'Authorization: Bearer <token>' | jq .
```

SQL fallback (Cloud SQL / psql):

```sql
SELECT
  p.id,
  p.slug,
  p.name,
  COUNT(*) AS total_studies,
  COUNT(*) FILTER (WHERE s.institution_id IS NULL) AS unattributed_studies
FROM studies s
JOIN projects p ON p.id = s.project_id
WHERE p.restricted = true
  AND s.created_at >= now() - interval '7 days'
GROUP BY p.id, p.slug, p.name
ORDER BY unattributed_studies DESC, total_studies DESC;
```

Remediation:
- Confirm upload path is using explicit institution attribution for site-scoped users.
- Verify institution is enabled, type `sender|both`, and linked to the project as `sender|admin`.
- Re-run failed uploads after membership/institution link corrections.

### 6. Resolve

Apply the fix. Verify with:

```bash
curl -s https://api.aegisimaging.ai/healthz | jq .status
# Expected: "ok"
```

### 7. Communicate

Update `#aegis-incidents`:

```
[RESOLVED] <incident title>
Duration: <start> – <end> UTC
Root cause: <brief explanation>
Fix applied: <what was done>
Follow-up: <ticket/PR number> or "none"
```

### 8. Post-mortem (P0/P1 only)

Within 48 hours, add a post-mortem document to `docs/runbooks/postmortems/`:

- Timeline of events
- Root cause analysis
- What went well / what didn't
- Action items with owners and due dates

---

## Key Commands

### Cloud Run

```bash
# View service status
gcloud run services describe aegis-api --region=us-central1 --project=aegis-prod-488119

# View recent revisions
gcloud run revisions list --service=aegis-api --region=us-central1 --project=aegis-prod-488119

# Roll back to previous revision
PREV=$(gcloud run revisions list --service=aegis-api --region=us-central1 \
  --project=aegis-prod-488119 --format='value(name)' | sed -n '2p')
gcloud run services update-traffic aegis-api \
  --to-revisions=$PREV=100 --region=us-central1 --project=aegis-prod-488119

# Stream live logs
gcloud alpha logging tail \
  'resource.type="cloud_run_revision" AND resource.labels.service_name="aegis-api"' \
  --project=aegis-prod-488119
```

### Cloud SQL

```bash
# Connect to DB (requires Cloud SQL Auth Proxy or IAP tunnel)
cloud_sql_proxy -instances=aegis-prod-488119:us-central1:aegis-prod=tcp:5432 &
psql -h 127.0.0.1 -U aegis -d aegis

# Check replication lag (if replica exists)
SELECT now() - pg_last_xact_replay_timestamp() AS replication_lag;

# List active queries
SELECT pid, now() - query_start AS duration, state, query
FROM pg_stat_activity WHERE state != 'idle' ORDER BY duration DESC;
```

### Secrets

```bash
# Retrieve DB password
gcloud secrets versions access latest \
  --secret=aegis-prod-db-password --project=aegis-prod-488119
```

### Terraform

```bash
# Check drift (read-only)
cd terraform/infra
terraform plan -var-file=terraform.tfvars

# Apply targeted fix
terraform apply -target=google_cloud_run_v2_service.api -var-file=terraform.tfvars
```

---

## Service Dependencies

```
Browser / PACS
    │
    ▼
Load Balancer (HTTPS)
    │
    ├── Cloud Armor (WAF) → API (Cloud Run)
    │                           │
    │                           ├── Cloud SQL (PostgreSQL) ← critical path
    │                           ├── GCS (DICOM files)
    │                           ├── Secret Manager
    │                           │
    │                           └── Sidecars (Cloud Run) ← non-critical path
    │                                 defacing / phi-detection / qc / bids /
    │                                 classification / protocol / dimse-receiver
    │
    └── IAP → Admin Dashboard (Cloud Run)
```

The **critical path** for study upload is: Load Balancer → API → Cloud SQL + GCS.
All sidecar services are on the non-critical path — their unavailability degrades
pipeline automation but does not block manual study approval.

---

## HIPAA Incident Considerations

If the incident involves a potential PHI breach (data exposed to unauthorised parties):

1. **Do not discuss specifics in public channels** (Slack, email)
2. Notify the Privacy Officer within 1 hour of discovery
3. Preserve logs — do not rotate or delete any Cloud Logging entries
4. Follow the HIPAA Breach Notification Rule timeline (60 days from discovery for patient notification, if required)
5. Engage legal counsel before any external communication

See your organisation's HIPAA Security Incident Response Policy for full procedures.
