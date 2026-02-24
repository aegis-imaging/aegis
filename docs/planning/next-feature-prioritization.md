# AEGIS Next Feature Prioritization Backlog

Review date: 2026-02-23
Scope reviewed: All services, API, frontend, Terraform, CI, MCP, docs.

## Current-State Snapshot (as of 2026-02-23)

- **GCP production is live**: deployed to `aegis-prod-488119`, `us-central1`
- All P0/P1/P2 items from the 2026-02-20 backlog are **complete** (see archived section below)
- Full operator tooling in place: MCP server, batch import, DIMSE retry proxy, cloud smoke tests
- All 7 Python processing sidecars deployed and tested (206+ pytest tests)
- Go API: 120+ tests, full routing/audit/export/share pipeline
- Admin dashboard: diagnostics tab, bulk actions, CSV export, study notes, shares management
- Export portal: enhanced study info display with badges, countdown timers
- CI matrix: 14 jobs covering Go, Python (7×), TypeScript (5×), Docker (8×)

## Completed Items (Archived from Previous Backlog)

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
| P2-12 | MCP Server Phase 1 (guarded write tools, MCP_ENABLE_WRITE_TOOLS) | 2026-02-21 |
| P2-13 | MCP security hardening (readonly mode, MCP_CALLER_ID, operator key) | 2026-02-21 |
| P2-14 | MCP DIMSE operator tools (retry/replay via /api/dimse/retry proxy) | 2026-02-21 |
| Q-1 | Docker-compose OHIF service fix + CI docker matrix | 2026-02-22 |
| Q-2 | MCP get_study_by_uid path fix + /api/studies/by-uid alias | 2026-02-22 |
| Q-3 | Bulk study approve/reject (multi-select + POST /api/studies/bulk) | 2026-02-22 |
| Q-4 | Study CSV export (GET /api/studies.csv + dashboard download button) | 2026-02-22 |
| Q-5 | Export portal enhancement (badges, description, UID display, instance count) | 2026-02-22 |
| Q-6 | Admin study notes (stored as audit entries, inline form in detail panel) | 2026-02-22 |
| Q-7 | Shares tab improvements (email search, countdown display, Note column) | 2026-02-22 |
| Q-8 | Executive summary + architecture diagram + PDF regeneration | 2026-02-22 |
| Q-9 | DIMSE receiver deployed to Compute Engine VM (`aegis-prod-dimse-receiver`, static IP `35.232.172.221`, TCP 11112) | 2026-02-23 |
| Q-10 | Cloud Build CI/CD triggers (`deploy-on-develop` + `terraform-apply-on-develop`) active in `us-central1` | 2026-02-23 |
| Q-11 | MCP agent Zod validation fix (invalid response: missing summary bug) | 2026-02-23 |
| Q-12 | Cloud Build SA IAM hardening — all required roles for `terraform apply` tracked in Terraform + setup script | 2026-02-23 |

## New Feature Candidates

Scoring:
- Impact: 1-5 (higher = more operator/researcher/patient value)
- Effort: S/M/L/XL
- Priority: P0 (now), P1 (next), P2 (after)

| Rank | Priority | Feature | Why Next | Impact | Effort |
|---|---|---|---|---:|---|
| 1 | P0 | **Production observability dashboard** (Cloud Monitoring / Grafana — study throughput, pipeline latency, error rates, sidecar health) | Blind-flying in prod; first real pilot traffic needs visibility | 5 | M |
| 2 | P0 | **Study SLA / stuck-detection alerting** (configurable age thresholds per pipeline stage → email/webhook alert when study is stuck longer than N minutes) | Operators need proactive notification, not manual dashboard polling | 5 | M |
| 3 | P1 | **Webhook / event notification system** (POST to external URL on study status transitions — approved, rejected, export complete) | Enables downstream automation without polling | 4 | M |
| 4 | P1 | **Study re-processing workflow** (re-trigger any pipeline step on an already-processed study; re-deface, re-scan PHI, re-classify without full re-upload) | Critical for production error recovery and pilot QC cycles | 4 | M |
| 5 | P1 | **Automated defacing visual QA** (pixel-level diff scoring raw vs. clean; flag studies where defacing similarity is too high or too low) | Catch defacing failures without manual OHIF review of every study | 4 | L |
| 6 | P1 | **Study annotation / structured labeling** (admin-applied key-value tags on studies; searchable, filterable, exportable in CSV) | Researchers need custom metadata beyond DICOM tags | 3 | M |
| 7 | P1 | **Per-project dashboard views** (filter all dashboard panels by project; project-scoped stats and audit trail) | Multi-project pilots need scoped operator views | 3 | M |
| 8 | P2 | **Subject-session linking** (group studies from the same de-identified subject across sessions using deterministic pseudonym; privacy-preserving cross-visit tracking) | Longitudinal studies require linking visits without exposing real patient IDs | 5 | L |
| 9 | P2 | **API key management** (named long-lived API keys for programmatic access; scoped to project + role; key rotation and revocation) | Some integrations cannot use IAP/Cognito browser flows | 3 | M |
| 10 | P2 | **Batch export improvements** (export entire project to ZIP or BIDS; schedule nightly export; progress tracking for large batches) | Research delivery workflows need full-project exports | 3 | M |
| 11 | P2 | **PHI detection confidence tuning UI** (per-project OCR threshold configuration; flagged-findings review workflow with accept/dismiss per finding) | Production PHI flag rates need operator-tunable thresholds | 3 | M |
| 12 | P2 | **Multi-site / federated query stub** (list studies across multiple AEGIS instances via MCP aggregation; groundwork for consortium deployments) | Long-term strategic: multi-site neuroimaging consortia | 4 | XL |
| 13 | P2 | **DICOM conformance statement** (formal documentation of AEGIS DICOMweb and DIMSE conformance — modalities, transfer syntaxes, SOP classes) | Required for enterprise PACS vendor certification and pilot sign-off | 2 | S |

## Recommended Execution Order

For post-pilot production hardening:

1. Feature #1 (observability dashboard) — deploy first, then watch
2. Feature #2 (SLA stuck-detection alerts) — operators must be notified proactively
3. Feature #3 (webhooks) — enables downstream automation pipelines
4. Feature #4 (re-processing) — essential for QC cycles and production error recovery
5. Feature #5 (automated defacing QA) — close the manual review bottleneck
6. Feature #6-#7 (annotations, per-project views) — researcher UX improvements
7. Feature #8 (subject-session linking) — longitudinal capability
8. Feature #9-#11 (API keys, batch export, PHI tuning)
9. Feature #12-#13 (federated query, conformance statement)

## Branch Naming Suggestions

- `feature/prod-observability-dashboard`
- `feature/study-sla-alerting`
- `feature/webhook-event-notifications`
- `feature/study-reprocessing`
- `feature/defacing-visual-qa`
- `feature/study-annotations`
- `feature/per-project-dashboard-views`
- `feature/subject-session-linking`
- `feature/api-key-management`
- `feature/batch-export-improvements`
- `feature/phi-confidence-tuning-ui`
