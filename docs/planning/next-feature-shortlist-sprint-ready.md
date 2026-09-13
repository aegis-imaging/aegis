# AEGIS Sprint-Ready Shortlist

Date: 2026-02-26
Source backlog: `docs/planning/next-feature-prioritization.md`

## Status: Three Clouds Live — Moving to Beta Hardening + Enterprise Readiness

GCP live (day 7) · AWS live + cross-cloud routing verified (day 8) · Azure Container Apps deploying (day 9).
Private beta is active. Next sprint focuses on production hardening, compliance, and enterprise integrations.

---

## Completion Status (previous sprints — all done)

| # | Item | Status | Completed |
|---|------|--------|-----------|
| 1 | GCP Terraform Production Completion | ✅ DONE | 2026-02-21 |
| 2 | Secrets and Credential Hardening | ✅ DONE | 2026-02-21 |
| 3 | Automated Cloud Smoke Test Suite (11/11 PASS) | ✅ DONE | 2026-02-22 |
| 4 | DIMSE PACS E2E Validation Harness | ✅ DONE | 2026-02-21 |
| 5 | AWS HTTPS + Cognito Edge/Auth | ✅ DONE | 2026-02-21 |
| 6 | DIMSE Receiver on Compute Engine VM (static IP `<GCP_DIMSE_PUBLIC_IP>`, TCP 11112) | ✅ DONE | 2026-02-23 |
| 7 | Cloud Build CI/CD pipeline (deploy-on-develop + terraform-apply-on-develop) | ✅ DONE | 2026-02-23 |
| 8 | MCP Server Phase 0+1 (52+ read tools, 29+ write tools) | ✅ DONE | 2026-02-23 |
| 9 | Production Observability Dashboard (9 alert policies, Cloud Monitoring) | ✅ DONE | 2026-02-23 |
| 10 | Study SLA / Stuck-Detection Alerting (`GET /api/studies/stuck`, scheduler) | ✅ DONE | 2026-02-23 |
| 11 | Webhook / Event Notification System (5 events, HMAC, retry+delivery log) | ✅ DONE | 2026-02-24 |
| 12 | Study Re-Processing Workflow (`POST /api/studies/{id}/reset-pipeline-step`) | ✅ DONE | 2026-02-24 |
| 13 | Per-Project Dashboard Views (global project selector, localStorage) | ✅ DONE | 2026-02-24 |
| 14 | AWS Production Deployment (ECS Fargate, RDS, S3, ALB + Cognito) | ✅ DONE | 2026-02-25 |
| 15 | Cross-Cloud DICOM Routing (GCP→AWS STOW-RS verified live) | ✅ DONE | 2026-02-25 |
| 16 | Agent Orchestrator (DICOM tag provenance + 9 diagnostic tools) | ✅ DONE | 2026-02-25 |
| 17 | Azure Container Apps Deployment (Terraform + GitHub Actions OIDC) | ✅ DONE | 2026-02-26 |

---

## New Sprint Shortlist (Q1–Q2 2026)

### 1) Azure Production Verification + Cross-Cloud Routing

Suggested branch: `feature/azure-crosscloud-routing`

**Goal**: Confirm Azure deployment is fully healthy end-to-end and verify GCP→Azure (and AWS→Azure) STOW-RS routing.

**In Scope**:
- Verify all Container Apps services healthy (`/healthz` returns ok for API + all 7 sidecars)
- Confirm DIMSE receiver VM is reachable on TCP 11112 with Elastic IP
- Add Azure health endpoints to cloud smoke test suite
- Configure bidirectional STOW-RS routing: create route_to destinations in GCP and AWS pointing to Azure API
- Verify routing: approve a study in GCP → confirm it arrives in Azure S3-equivalent (Blob Storage)
- Document Azure API key auth pattern (same `aegis_` bearer token format)

**Acceptance Criteria**:
- Azure `/healthz` returns `{"status":"ok"}` with all sidecars healthy
- `storescu` test from external host reaches Azure DIMSE receiver
- GCP→Azure STOW-RS routing test passes end-to-end
- Azure added to cloud smoke suite

**Test Plan**:
- Run `scripts/cloud_smoke_test.py --base-url https://azure.api.aegisimaging.ai`
- Verify routing in admin dashboard: study received in Azure after GCP approval

---

### 2) SMTP / Email Configuration for Production

Suggested branch: `feature/production-smtp-config`

**Goal**: Enable email features (share notifications, upload confirmations, pipeline failure alerts, digest subscriptions) in production on all three clouds.

**In Scope**:
- Provision SMTP relay (Brevo, SendGrid, or institutional relay — **not Brevo**, AEGIS uses standard SMTP)
- Configure `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`, `SMTP_PASSWORD` on GCP Cloud Run via Secret Manager
- Mirror SMTP config to AWS ECS task definition via Secrets Manager
- Test: create an export share → verify recipient receives notification email
- Test: pipeline failure (set `PIPELINE_ALERT_EMAIL`) → verify alert fires
- Verify digest subscription fires on schedule

**Acceptance Criteria**:
- Share creation sends email to recipient
- Upload completion sends confirmation to uploader (if email provided)
- Pipeline failure alert fires to `PIPELINE_ALERT_EMAIL`
- Digest scheduler sends weekly summary

**Test Plan**:
- Local dev: `docker run -p 1025:1025 -p 8025:8025 axllent/mailpit`, verify emails in Mailpit UI
- Production: create share, verify email arrives via real SMTP relay

---

### 3) SOC 2 Type I Audit Initiation

Suggested branch: `feature/soc2-audit-prep`

**Goal**: Engage a SOC 2 auditor and begin the formal audit process, targeting Type I report by Q3 2026.

**In Scope**:
- Select auditor (Drata / Vanta / Secureframe for continuous compliance + manual auditor)
- Complete security questionnaire / vendor onboarding
- Document all technical controls already implemented:
  - Encryption at rest (CMEK/KMS on GCP, KMS on AWS, customer-managed keys on Azure)
  - Encryption in transit (TLS 1.3)
  - Access control (IAP, Cognito, Easy Auth, RBAC, API keys)
  - Audit trail (immutable, timestamped, actor-attributed)
  - Vulnerability management (distroless containers, Artifact Registry scanning, dependabot)
  - Incident response runbook (`docs/runbooks/incident-response.md`)
- Identify gaps and create remediation tasks
- Enable GCP Access Transparency (log all Google admin access)

**Acceptance Criteria**:
- Auditor engaged and kickoff meeting scheduled
- Control inventory document created and reviewed
- No critical gaps identified before audit start

---

### 4) DIMSE C-MOVE / C-FIND (Active PACS Pull)

Suggested branch: `feature/dimse-cmove-cfind`

**Goal**: Enable AEGIS to actively pull studies from PACS systems on demand (C-FIND for query, C-MOVE to retrieve), rather than waiting for incoming C-STORE push.

**In Scope**:
- Add `dimse-receiver/app/scu.py` — SCU role (C-FIND + C-MOVE)
- New API: `POST /api/dimse/query` — `{ae_title, host, port, patient_id?, study_date?, modality?}` → returns matching studies from remote PACS
- New API: `POST /api/dimse/retrieve` — `{ae_title, host, port, study_instance_uid}` → pulls study via C-MOVE to AEGIS storage, triggers ingest
- Admin dashboard: "Query PACS" panel on the DIMSE ops page
- Destination model: add SCU fields (`move_destination_ae`) for C-MOVE response routing

**Acceptance Criteria**:
- C-FIND query returns patient/study list from a real PACS (tested with `dcmtk` test SCP)
- C-MOVE retrieve imports study into AEGIS and fires normal pipeline
- Admin dashboard shows query results and allows one-click retrieve
- Auth: admin-only, DIMSE operator key required if configured

**Test Plan**:
- Run `storescp` as mock PACS, test C-FIND query from AEGIS
- Retrieve a study via C-MOVE, verify it appears in Studies tab

---

### 5) AWS Marketplace Listing Prep

Suggested branch: `feature/aws-marketplace-listing`

**Goal**: Prepare AEGIS for listing on AWS Marketplace to enable enterprise procurement via AWS billing.

**In Scope**:
- Create AWS Marketplace seller account (register as ISV on AWS Marketplace)
- Choose listing type: SaaS contract (recommended for HIPAA-eligible services) or AMI
- Prepare product description, pricing model, screenshots
- HIPAA Eligible Services confirmation (S3, RDS, ECS are all HIPAA-eligible; sign BAA with AWS)
- Metering integration (if SaaS subscription): implement `aws-marketplace-metering-service` calls
- Review required: AWS Security Review for HIPAA workloads
- Draft BAA template with legal counsel

**Acceptance Criteria**:
- Marketplace listing submitted for AWS review
- BAA template drafted and reviewed by counsel
- HIPAA eligibility checklist completed

---

## Definition of Done (applies to each item)

- Code merged to `develop` via feature branch
- Automated tests included and passing (Go unit/integration or Python pytest)
- TypeScript strict check passes (`npx tsc --noEmit`)
- `CLAUDE.md` updated when new env vars, endpoints, or behaviors are added
- Deployment/runbook steps reproducible by another engineer without tribal context
