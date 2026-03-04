# AEGIS Next Feature Prioritization Backlog

Review date: 2026-02-26
Scope reviewed: All services, API, frontend, Terraform, CI, MCP, docs.

## Current-State Snapshot (as of 2026-02-26)

- **GCP production is live**: `aegis-prod-488120`, `us-central1` — day 7
- **AWS production is live**: `301691475234`, `us-east-1` — day 8; cross-cloud routing GCP→AWS verified
- **Azure live**: Container Apps + PostgreSQL Flex + Azure Blob — live as of Feb 26
- Go API: 126+ routes, 64 handler files, all fully implemented
- 400+ automated tests (160+ Go integration tests, 244+ Python pytest tests across 8 sidecars)
- MCP Server: 52+ read tools + 29+ write tools — all APIs covered
- Agent orchestrator: DICOM tag provenance + 9 diagnostic tools (PRs #245–246)
- Admin dashboard: 17 management tabs — full-featured
- All 7 Python processing sidecars deployed and tested on GCP and AWS
- Private beta active (invite-gated via landing page)

## Completed Items (Archived from All Previous Backlogs)

| # | Feature | Completed |
|---|---------|-----------|
| P0-1 | GCP Terraform production module (Cloud Run, VPC, IAP, Cloud Armor, monitoring) | 2026-02-21 |
| P0-2 | AWS Terraform HTTPS + Cognito edge/auth completion | 2026-02-21 |
| P0-3 | Secrets & credential hardening (Secret Manager / Secrets Manager patterns) | 2026-02-21 |
| P0-4 | Automated cloud smoke test suite (11/11 pass) | 2026-02-22 |
| P0-5 | DIMSE PACS E2E validation harness + runbook | 2026-02-21 |
| P1-6 | DIMSE durable retry/dead-letter state (restart-safe JSON persistence) | 2026-02-21 |
| P1-7 | Admin DIMSE ops panel + retry proxy endpoints | 2026-02-21 |
| P1-8 | DIMSE retry threshold alerts + webhook delivery | 2026-02-21 |
| P1-9 | Study diagnostics endpoint + diagnostics panel in dashboard | 2026-02-22 |
| P1-10 | Makefile lint/CI parity improvements | 2026-02-22 |
| P2-11 | MCP Server MVP Phase 0 (read-only tools) | 2026-02-21 |
| P2-12 | MCP Server Phase 1 (guarded write tools) | 2026-02-21 |
| P2-13 | MCP security hardening (readonly mode, operator key) | 2026-02-21 |
| P2-14 | MCP DIMSE operator tools (retry/replay via /api/dimse/retry proxy) | 2026-02-21 |
| Q-1 | Bulk study approve/reject/delete (POST /api/studies/bulk) | 2026-02-22 |
| Q-2 | Study CSV export (GET /api/studies.csv + dashboard button) | 2026-02-22 |
| Q-3 | Admin study notes (stored as audit entries) | 2026-02-22 |
| Q-4 | Export portal enhancement (badges, description, UID, countdown) | 2026-02-22 |
| Q-5 | Shares tab improvements (email search, countdown, Note column) | 2026-02-22 |
| Q-6 | Executive summary + architecture diagram + PDF | 2026-02-22 |
| Q-7 | DIMSE receiver on Compute Engine VM (static IP 35.232.172.221, TCP 11112) | 2026-02-23 |
| Q-8 | Cloud Build CI/CD triggers active | 2026-02-23 |
| P1-F1 | Production observability dashboard (9 alert policies, Cloud Monitoring) | 2026-02-23 |
| P1-F2 | Study SLA / stuck-detection alerting (GET /api/studies/stuck, scheduler) | 2026-02-23 |
| P1-F3 | Webhook / event notification system (5 events, HMAC, 3× retry) | 2026-02-24 |
| P1-F4 | Study re-processing workflow (POST /api/studies/{id}/reset-pipeline-step) | 2026-02-24 |
| P1-F5 | Defacing visual QA score (SSIM-based, stored on study record) | 2026-02-24 |
| P1-F6 | Study annotation / label system (bulk label, search, filter) | 2026-02-24 |
| P1-F7 | Per-project dashboard views (global project selector, localStorage) | 2026-02-24 |
| P2-F8 | Subject-session linking (subject_id, cohort report) | 2026-02-24 |
| P2-F9 | API key management (named bearer tokens, scoped, rotatable) | 2026-02-24 |
| P2-F10 | BIDS bulk export (project-level merged ZIP, GET /api/projects/{id}/bids-export) | 2026-02-24 |
| P2-F11 | PHI detection confidence tuning (per-project thresholds via API + dashboard) | 2026-02-24 |
| P2-F12 | Study relationships (baseline, follow_up, comparison, replicate) | 2026-02-24 |
| P2-F13 | Export share extension + revocation reason | 2026-02-24 |
| P2-F14 | Routing rule export/import/reorder/simulate | 2026-02-24 |
| P2-F15 | Destination health probe scheduler + connectivity test | 2026-02-24 |
| P2-F16 | Pipeline funnel + project health summary stats | 2026-02-25 |
| P2-F17 | AWS production deployment (ECS Fargate, RDS, S3, ALB + Cognito) | 2026-02-25 |
| P2-F18 | Cross-cloud DICOM routing GCP→AWS STOW-RS verified live | 2026-02-25 |
| P2-F19 | Agent orchestrator expansion (DICOM tag provenance + 9 diagnostic tools) | 2026-02-25 |
| P2-F20 | Azure Container Apps deployment (Terraform + GitHub Actions OIDC) | 2026-02-26 |

## New Feature Candidates (Q2 2026)

Scoring:
- Impact: 1-5 (higher = more operator/researcher/patient value)
- Effort: S/M/L/XL
- Priority: P0 (now), P1 (next), P2 (after)

| Rank | Priority | Feature | Why Next | Impact | Effort |
|---|---|---|---|---:|---|
| 1 | P0 | **Azure cross-cloud routing verification** (smoke test + GCP→Azure STOW-RS routing) | Azure is live — close the loop on three-cloud routing | 4 | S |
| 2 | P0 | **SMTP / email configuration for production** (share notifications, pipeline alerts, digests) | Email features are silent no-ops; needed before first real pilot users | 5 | S |
| 3 | P0 | **SOC 2 Type I audit initiation** (select auditor, complete controls inventory, remediate gaps) | Required for enterprise customer pilots — HIPAA + SOC 2 together make the strongest compliance story | 5 | M |
| 4 | P1 | **DIMSE C-MOVE / C-FIND** (active PACS pull — SCU role for query+retrieve) | Enables AEGIS to pull from PACS rather than waiting for push; unlocks enterprise PACS integration | 5 | L |
| 5 | P1 | **AWS Marketplace listing** (SaaS contract listing for enterprise procurement) | Many enterprise buyers require AWS Marketplace for procurement; direct revenue channel | 4 | M |
| 6 | P1 | **HL7 FHIR notifications** (fire FHIR DiagnosticReport or ImagingStudy resources to EMR/RIS on study events) | Required for deep EMR integration; connects AEGIS to Epic, Cerner, etc. | 4 | L |
| 7 | P1 | **Production BAA** (Business Associate Agreement template + countersigning workflow) | Required before any real PHI touches the platform; first pilot customer needs signed BAA | 5 | S |
| 8 | P2 | **DICOM conformance statement refinement** (publish formal PS3.2 conformance statement for GCP, AWS, Azure deployments) | Required for PACS vendor certification; enterprise IT departments require this before deployment | 3 | S |
| 9 | P2 | **On-premises deployment guide** (Helm chart or Docker Compose for airgapped/on-premises environments) | Some research institutions have strict data residency; on-prem expands TAM | 4 | L |
| 10 | P2 | **Multi-tenant SaaS mode** (per-organization database isolation; tenant provisioning API) | Enables AEGIS to serve multiple customers from one platform; scales revenue without N cloud deployments | 5 | XL |
| 11 | P2 | **Cross-tenant federated sharing** (activate federation peer registry — peer AEGIS instances exchange approved studies) | Long-term strategic: multi-institution neuroimaging consortia; federation registry already exists | 4 | L |
| 12 | P2 | **TCIA import automation** (scheduled batch imports from TCIA collections; enriches pilot data for demos) | Demo / pilot value — shows AEGIS processing real clinical datasets hands-free | 3 | S |

## Recommended Execution Order (Q2 2026)

For post-beta production readiness:

1. Feature #1 (Azure verification) — close three-cloud loop immediately
2. Feature #7 (BAA) + Feature #3 (SOC 2) — legal must happen in parallel with dev
3. Feature #2 (SMTP) — email before first real pilot user signs up
4. Feature #4 (DIMSE C-MOVE) — unlocks enterprise PACS pull integration
5. Feature #5 (AWS Marketplace) — enterprise procurement channel
6. Feature #6 (HL7 FHIR) — EMR integration for clinical sites
7. Feature #8 (conformance statement) — PACS vendor cert
8. Feature #9-#11 (on-prem, multi-tenant, federation) — longer horizon

## Branch Naming Suggestions

- `feature/azure-crosscloud-routing`
- `feature/production-smtp-config`
- `feature/soc2-audit-prep`
- `feature/dimse-cmove-cfind`
- `feature/aws-marketplace-listing`
- `feature/hl7-fhir-notifications`
- `feature/baa-workflow`
- `feature/dicom-conformance-v2`
- `feature/onprem-deployment-guide`
- `feature/multi-tenant-saas`
- `feature/cross-tenant-federation`
