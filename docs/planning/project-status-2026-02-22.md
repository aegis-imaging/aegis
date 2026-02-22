# AEGIS Project Status

Date: 2026-02-22
Previous status: `docs/planning/project-status-and-top-priorities-2026-02-20.md`

## Summary

GCP production environment is **fully deployed and operational**. The platform is pilot-ready for the cloud path. All P0/P1 items from the 2026-02-20 backlog are complete.

## What Was Completed (2026-02-21 → 2026-02-22)

### GCP Production Deployment (`aegis-prod-488120`)

- **Project**: `aegis-prod-488120`, region `us-central1`, billing `016DEE-91CE5C-ECB970`
- **API**: `https://api.aegisimaging.ai` — Cloud Run, distroless Go container, all routes live
- **Admin dashboard**: `https://admin.aegisimaging.ai` — IAP-gated, `<your-google-account-email>` provisioned as first admin
- **6 Python sidecars** deployed on Cloud Run: defacing, phi-detection, qc-service, bids-service, classification-service, protocol-service — all healthy
- **DIMSE receiver** deployed as sidecar
- **Cloud SQL** PostgreSQL 15 on private IP, credentials in Secret Manager
- **GCS bucket** for DICOM storage with IAM-based signed URL generation (no private key required)
- **HTTPS load balancer** with managed TLS cert, Cloud Armor on API backend, IAP on admin backend
- **DNS**: `api.aegisimaging.ai` and `admin.aegisimaging.ai` pointed at LB IP

### Key Technical Fixes Applied

1. **Sidecar `/health` endpoint** — Cloud Run v2 intercepts external traffic to the liveness probe path (`/healthz`). Added `/health` alias to all 7 Python services. API health loop changed to probe `/health`. Terraform liveness probes updated to `/health`.

2. **GCS IAM-based signed URLs** — API now signs GCS upload/download URLs via `iamcredentials.SignBlob` (no long-lived key needed). API service account granted `roles/iam.serviceAccountTokenCreator` on itself. `GCS_SIGNING_EMAIL` injected via Terraform.

3. **Cloud smoke test fixes** — Two smoke test bugs fixed:
   - `upload.file`: Missing `Content-Type: application/dicom` header on GCS PUT (signed URL requires matching content-type)
   - `share.download`: GCP LB forwards backend headers in HTTP/2 lowercase (`content-type`, not `Content-Type`). Fixed with case-insensitive header lookup.

### Cloud Smoke Suite Result (2026-02-22)

```
[PASS] healthz              (0.16s)
[PASS] auth.me              (0.11s) - <your-google-account-email>, role=admin
[PASS] admin.users.registered (0.11s) - 1 enabled admin
[PASS] upload.init          (0.16s)
[PASS] upload.file          (0.17s)
[PASS] upload.complete      (0.32s)
[PASS] pipeline.progression (0.11s) - status=received
[PASS] study.approve        (0.12s)
[PASS] share.create         (0.12s)
[PASS] share.redeem         (0.26s)
[PASS] share.download       (0.92s)
Result: PASS (2.56s, steps=11)
```

## Current Status Findings

| # | Finding | Status |
|---|---------|--------|
| 1 | AWS deploy path not production-complete (ECS service definitions + HTTPS/Cognito TODOs) | Open |
| 2 | OHIF viewer in admin dashboard hardcoded to `localhost:3002` — broken in production | **In progress** |
| 3 | SMTP not configured — email features (share notify, upload confirm, digest) are no-ops in prod | Open |
| 4 | DIMSE retry/dead-letter state is durable (disk persistence added) | ✅ Resolved |
| 5 | MCP server partially implemented | Open (lower priority) |
| 6 | Secret rotation drill not yet done | Open |
| 7 | Cloud Armor policy attached but not verified end-to-end | Open |
| 8 | Monitoring alert policies created but alerting email not confirmed | Open |

## Top Remaining Priorities

1. **OHIF viewer production fix** — deploy OHIF to Cloud Run with `API_URL=https://api.aegisimaging.ai`; update admin dashboard to use Cloud Run OHIF URL (BLOCKING for study review workflow)
2. **DIMSE PACS E2E validation harness** — prove enterprise ingress reliability end-to-end
3. **AWS HTTPS + Cognito completion** — close AWS auth/edge TODOs for multi-cloud parity
4. **SMTP configuration** — configure Brevo or similar relay for email notifications
5. **Secret rotation drill** — do one GCP DB password rotation to confirm runbook

## Environment Reference

| Item | Value |
|------|-------|
| GCP Project | `aegis-prod-488120` |
| Region | `us-central1` |
| API | `https://api.aegisimaging.ai` |
| Admin Dashboard | `https://admin.aegisimaging.ai` |
| AR Registry | `us-central1-docker.pkg.dev/aegis-prod-488120/aegis-services` |
| First Admin | `<your-google-account-email>` |
| DB Secret | `aegis-prod-db-password` (Secret Manager) |
| GCS Signing | IAM-based (no private key), `GCS_SIGNING_EMAIL` = API SA email |
