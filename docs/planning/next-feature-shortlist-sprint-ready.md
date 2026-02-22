# AEGIS Sprint-Ready Shortlist (Top 5)

Date: 2026-02-22
Source backlog: `docs/planning/next-feature-prioritization.md`

## Status: Pilot-Ready — Moving to Production Hardening

All previous P0/P1/P2 backlog items are complete. GCP is live (`aegis-prod-488119`).
This shortlist covers the next sprint for post-pilot production hardening.

## Completion Status (previous sprint)

| # | Item | Status | Completed |
|---|------|--------|-----------|
| 1 | GCP Terraform Production Completion | ✅ DONE | 2026-02-21 |
| 2 | Secrets and Credential Hardening | ✅ DONE | 2026-02-21 |
| 3 | Automated Cloud Smoke Test Suite | ✅ DONE | 2026-02-22 (11/11 PASS) |
| 4 | DIMSE PACS E2E Validation Harness | ✅ DONE | 2026-02-21 |
| 5 | AWS HTTPS + Cognito Edge/Auth | ✅ DONE | 2026-02-21 |

## New Sprint Shortlist (Top 5)

### 1) Production Observability Dashboard

Suggested branch: `feature/prod-observability-dashboard`

**Goal**: Real-time visibility into the live GCP deployment — study ingestion rate, pipeline stage
latency, sidecar error rates, DIMSE queue depth, Cloud Run CPU/memory.

**In Scope**:
- Cloud Monitoring dashboard JSON export committed to `terraform/monitoring/`
- Key metrics: studies received/day, pipeline stage durations, sidecar HTTP 5xx rate,
  DIMSE pending/dead-letter counts, Cloud Run revision health, DB connection pool utilization
- Alert policies linked to notification channels (already provisioned in Terraform)
- Dashboard accessible at GCP Console link from `SETUP_CHECKLIST.md`

**Acceptance Criteria**:
- Dashboard covers at least 6 key operational metrics with meaningful thresholds
- At least 2 alert policies: high 5xx rate + study stuck > 30 min
- Dashboard snapshot included in docs/

**Test Plan**:
- Deploy dashboard with `gcloud monitoring dashboards create`
- Verify metrics populate within 5 min of live traffic
- Trigger test alert by pushing a known-bad request

---

### 2) Study SLA / Stuck-Detection Alerting

Suggested branch: `feature/study-sla-alerting`

**Goal**: Proactively notify operators when a study has been in any pipeline stage longer than a
configurable threshold — replacing manual dashboard polling.

**In Scope**:
- New API: `GET /api/studies/stuck` — returns studies stuck in a given stage > N minutes
- Config: per-stage SLA thresholds (env vars or DB-backed project settings)
- Scheduler: hourly check (same pattern as email digest), emits `study.stuck` audit entry
- Email alert: reuses existing SMTP mailer with plain-text "studies stuck in pipeline" digest
- Dashboard badge: stuck count indicator in Studies tab header

**Acceptance Criteria**:
- Stuck studies surface in `/api/studies/stuck` response within one scheduler cycle
- Email alert fires when at least one study exceeds threshold
- Scheduler is a no-op when SMTP is not configured
- Alert does not re-fire for the same study on every cycle (cooldown per study per stage)

**Test Plan**:
- Unit test: stuck threshold logic with mock time
- Integration test: seed a study with old `updated_at` and verify it appears in stuck response
- E2E: confirm email fires in Mailpit during local dev

---

### 3) Webhook / Event Notification System

Suggested branch: `feature/webhook-event-notifications`

**Goal**: Allow external systems to subscribe to study status changes without polling the API.

**In Scope**:
- New DB table: `webhook_subscriptions` (url, events[], project_id, secret, enabled)
- New API: CRUD `/api/webhook-subscriptions` (admin-only)
- Delivery: async goroutine, POST JSON payload to subscriber URL on study status change
- Payload: `{event, study_id, study_instance_uid, project_id, timestamp}`; HMAC-SHA256 signature header
- Retry: 3× exponential backoff; failure logged to audit trail
- Events: `study.approved`, `study.rejected`, `study.phi_flagged`, `study.export_complete`
- Dashboard: Webhooks tab under Notifications showing subscriptions and recent delivery log

**Acceptance Criteria**:
- Webhook fires within 5 seconds of study status change
- HMAC-SHA256 `X-Aegis-Signature` header allows receiver to verify origin
- Delivery failures are logged and don't block the study workflow
- CRUD API covered by handler tests

**Test Plan**:
- Unit test: HMAC signature generation and verification
- Integration test: stub HTTP server receives webhook POST with correct payload
- E2E: approve a study via dashboard, verify webhook fires against test receiver

---

### 4) Study Re-Processing Workflow

Suggested branch: `feature/study-reprocessing`

**Goal**: Allow operators to re-trigger any pipeline step on an already-processed study without
requiring a full re-upload. Essential for fixing defacing errors, re-scanning with an updated
PHI model, or re-classifying after model improvement.

**In Scope**:
- New admin endpoints: `POST /api/studies/{id}/reset-pipeline-step` with `{step}` param
  (`deface`, `phi_scan`, `qc`, `bids`, `classify`, `protocol`)
- Resets the relevant `*_status` field back to `pending` (or `required+pending`)
- `PIPELINE_AUTO=true` picks it up automatically and re-dispatches
- Dashboard: "Re-run" button per pipeline stage dot in study detail panel
- Audit trail: `study.pipeline_reset` entry with which step was reset and by whom

**Acceptance Criteria**:
- Each pipeline step can be individually reset to `pending` via API
- Auto-pipeline picks up and re-dispatches without duplicate processing
- Admin-only (requires `role=admin`)
- Cannot reset a step that is currently `in-flight` (returns 409 Conflict)

**Test Plan**:
- Unit test: reset logic with various status combinations
- Integration test: set status to `complete`, reset, verify pending state
- E2E: reset a defaced study in dev, verify defacing re-triggers

---

### 5) Per-Project Dashboard Views

Suggested branch: `feature/per-project-dashboard-views`

**Goal**: Allow operators managing multiple projects to filter the entire admin dashboard to a
single project context — studies, audit, routing rules, institutions, shares, and stats.

**In Scope**:
- Global project selector dropdown in top navigation (persisted in localStorage)
- "All Projects" option (default, current behavior)
- When a project is selected:
  - Studies panel: auto-applies `project_id` filter
  - Stats panel: shows project-scoped counts
  - Audit panel: filters to project-related entries
  - Routing panel: shows only rules and destinations linked to the project
  - Shares panel: shows only shares from studies in the project
- URL query param `?project=<slug>` for shareable project-scoped links
- Dashboard title/header shows active project name when scoped

**Acceptance Criteria**:
- Project selector updates all visible panels simultaneously
- Selection persists across page refresh (localStorage)
- "All Projects" returns to current unfiltered behavior
- Deep-linked URLs with `?project=slug` restore the project filter on load

**Test Plan**:
- TypeScript strict check passes
- Manually verify each panel respects project filter in dev
- Verify localStorage persistence across hard refresh
- Verify URL sharing with `?project=slug`

---

## Definition of Done (applies to each item)

- Code merged to `develop` via feature branch
- Automated tests included and passing (Go unit/integration or Python pytest)
- TypeScript strict check passes (`npx tsc --noEmit`)
- `CLAUDE.md` updated when new env vars, endpoints, or behaviors are added
- Deployment/runbook steps reproducible by another engineer without tribal context
