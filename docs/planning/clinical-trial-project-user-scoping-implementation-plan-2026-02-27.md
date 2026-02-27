# Clinical Trial Project/User Scoping — Prioritized Implementation Plan (2026-02-27)

## Objective

Implement project/user scoping that matches the clinical-trial operating model for **Anonymization & Exchange Gateway for Imaging Studies (AEGIS)**:

- Satellite site personnel can see only their site’s studies within a project.
- Coordinating center project roles can see all studies in that project.
- Users can hold different roles across different projects.
- Platform superuser access is preserved for a single global owner/operator (current platform `admin` behavior).
- Behavior remains consistent across GCP, AWS, and Azure.

This document now serves as a completion record plus operational maintenance checklist.

---

## Constraints and Non-Negotiables

1. Preserve global superuser visibility for platform `admin` users across all projects.
2. Do not weaken existing site-scoped read protections already present on core study endpoints.
3. Keep rollout incremental, with one logical authorization surface per PR.
4. Require parity checks on all three cloud auth providers (`iap`, `azure`, `aws`) before merge for auth-surface changes.

---

## Delivered Scope Summary (Historical)

- **Phase 0 (Preparation):** access policy matrix + endpoint inventory.
- **Phase 1 (P0):** close backend read-surface scoping gaps.
- **Phase 2 (P0):** move project-scoped writes from platform-admin-only to project-member capability checks.
- **Phase 3 (P1):** align admin dashboard UI gating with capability model.
- **Phase 4 (P1):** make browser upload institution attribution deterministic for satellite workflows.
- **Phase 5 (P2):** hardening, observability, and regression protection.

## Status (Updated 2026-02-27)

- **Completed:** Phase 0 (`P0-01`, `P0-02`), Phase 1 `P0-10`, `P0-11`, `P0-12`, Phase 2 `P0-20`, `P0-21`, `P0-22`, Phase 3 `P1-30`, `P1-31`, Phase 4 `P1-40`, `P1-41`, Phase 5 `P2-50`, `P2-51`
  - `docs/planning/access-matrix-project-site-scoping.md`
  - `docs/planning/access-test-catalog-project-site-scoping.md`
- **Next active:** none (implementation complete; operational maintenance continues)
- **Queued:** follow-on hardening backlog

---

## Phase 0 — Access Policy Matrix and Endpoint Inventory (P0)

### Ticket P0-01: Authoritative Access Matrix
**Goal:** Define canonical access categories and map each API route.

**Status:** ✅ Completed (2026-02-27)

**Deliverables**
- `docs/planning/access-matrix-project-site-scoping.md` containing:
  - Access categories: `public`, `platform_admin`, `project_member_any`, `project_member_site_scoped`, `project_owner_only`.
  - Route-by-route expected behavior.
  - Data-scope rule (`institution_id` match required for site-scoped roles).

**Acceptance Criteria**
- Every authenticated route in `api/main.go` is mapped.
- Matrix explicitly preserves platform admin cross-project access.
- Matrix reviewed/approved before implementation tickets proceed.

### Ticket P0-02: Gap Baseline Test Catalog
**Goal:** Define baseline regression scenarios before behavior changes.

**Status:** ✅ Completed (2026-02-27)

**Deliverables**
- Test checklist (doc) with actor personas:
  - platform admin (global owner equivalent),
  - researcher owner/coordinator/reviewer,
  - site_coordinator/site_viewer,
  - non-member researcher.

**Acceptance Criteria**
- Checklist covers read + write + project list + CSV + diagnostics + shares + stats surfaces.

---

## Phase 1 — Backend Read-Surface Scoping Closure (P0)

### Ticket P0-10: Scope Non-Primary Study Reads
**Goal:** Ensure all study-derived reads use shared project/site access checks.

**Status:** ✅ Completed (2026-02-27)

**Primary targets**
- `api/handler/study_diagnostics.go`
- `api/handler/series.go`
- `api/handler/dicom_tags.go`
- `api/handler/anon_diff.go`

**Implementation direction**
- Reuse `requireStudyReadAccessByID` or `requireStudyReadAccessByUID` patterns.
- Return `404` for unauthorized study access to avoid existence leakage.

**Acceptance Criteria**
- Site-scoped member cannot read out-of-site study metadata on these endpoints.
- Coordinating center roles retain full in-project visibility.
- Platform admin still sees all.

### Ticket P0-11: Scope Global Share/Audit/Stats Surfaces for Researchers
**Goal:** Prevent cross-project leakage through global metadata endpoints.

**Status:** ✅ Completed (2026-02-27)

**Primary targets**
- Shares: global listing/analytics routes in `api/handler/export.go`
- Audit listing routes in `api/handler/audit.go`
- Stats routes in `api/handler/stats.go`
- Institution list/stats routes in `api/handler/institution.go`

**Implementation direction**
- For `researcher` role, enforce `project_id` where global aggregation is unsafe.
- Apply project membership check + optional site scoping where data is study-derived.
- Keep platform admin unrestricted.

**Acceptance Criteria**
- Researcher requests without required project scope receive explicit validation error.
- Researcher with membership only sees in-scope project data.
- Platform admin behavior unchanged.

### Ticket P0-12: Regression Tests for Read Scoping
**Goal:** Add handler/model tests proving no read leakage.

**Status:** ✅ Completed (2026-02-27)

**Acceptance Criteria**
- New tests cover each endpoint family updated in P0-10/P0-11.
- Tests include out-of-site and out-of-project denial cases.
- Existing tests remain green.

---

## Phase 2 — Project-Scoped Write Authorization (P0)

### Ticket P0-20: Capability-Based Write Guard Helpers
**Goal:** Introduce reusable guard functions keyed to project member role capabilities.

**Status:** ✅ Completed (2026-02-27)

**Implementation direction**
- Use `GetUserAccessForProject` and role capability semantics (`owner`, `coordinator`, `reviewer`, `site_coordinator`, `site_viewer`).
- Preserve platform `admin` override.

**Acceptance Criteria**
- Shared helper layer exists for write intents:
  - project management,
  - study mutation,
  - approval/rejection,
  - site-level operational actions.

### Ticket P0-21: Migrate Selected Write Routes off Platform-Admin-Only
**Goal:** Allow project-role-authorized writes without granting platform-admin.

**Status:** ✅ Completed (2026-02-27)

**Initial route scope (incremental)
- Study operational actions for in-scope project members.
- Project member management remains `owner` (or platform admin).

**Acceptance Criteria**
- `site_coordinator` can perform allowed site-scoped writes only in their project/site.
- `reviewer/coordinator/owner` can perform approved in-project actions.
- `site_viewer` remains read-only.
- Platform admin retains global capability.

### Ticket P0-22: Write Authorization Regression Suite
**Goal:** Prevent privilege escalation/regressions.

**Status:** ✅ Completed (2026-02-27)

**Acceptance Criteria**
- Negative tests: non-member, wrong-site member, site_viewer mutation attempts.
- Positive tests: owner/coordinator/reviewer/site_coordinator allowed paths.
- Platform admin still passes all global write paths.

---

## Phase 3 — Frontend Authorization Alignment (P1)

### Ticket P1-30: Replace `isAdmin`-Only UI Gating with Capability Model
**Goal:** UI reflects backend authorization reality for project roles.

**Status:** ✅ Completed (2026-02-27)

**Primary target**
- `frontend/admin-dashboard/src/App.tsx`

**Implementation direction**
- Derive per-project capabilities from `/api/auth/me` + project membership access.
- Keep platform admin shortcut for global owner-level behavior.

**Acceptance Criteria**
- Project members see the actions they are allowed to execute.
- Disallowed actions are hidden/disabled predictably.
- No UI path suggests actions that backend denies for in-scope role.

### Ticket P1-31: Project/Site Context UX Hardening
**Goal:** Reduce operator mistakes in multi-project multi-site use.

**Status:** ✅ Completed (2026-02-27)

**Acceptance Criteria**
- Active project/scope is always visible in header context.
- Site-scoped users cannot clear into unsafe “all projects” mode for endpoints requiring project scope.

---

## Phase 4 — Satellite Upload Attribution Reliability (P1)

### Ticket P1-40: Institution Attribution for Browser Uploads
**Goal:** Ensure uploaded studies are reliably tagged with site institution.

**Status:** ✅ Completed (2026-02-27)

**Primary targets**
- `client/src/upload/client.ts`
- `frontend/upload-portal` upload flow
- Existing backend support in `api/handler/upload.go`

**Implementation direction**
- Add explicit institution selector (or deterministic mapping path) for satellite uploads.
- Validate institution-project sender linkage at upload init.

**Acceptance Criteria**
- Satellite uploads consistently produce `study.institution_id`.
- Site-scoped visibility remains correct after upload.
- Ambiguous attribution is rejected with actionable error.

### Ticket P1-41: Attribution-Failure Monitoring Signals
**Goal:** Detect drift where uploads miss institution assignment.

**Status:** ✅ Completed (2026-02-27)

**Acceptance Criteria**
- Operational metric or audit query for `project-restricted studies with null institution_id`.
- Runbook note for triage/remediation.

---

## Phase 5 — Hardening and Operationalization (P2)

### Ticket P2-50: Endpoint Access Lint/Guardrail
**Goal:** Reduce future scoping drift.

**Status:** ✅ Completed (2026-02-27)

**Acceptance Criteria**
- CI check or static review checklist ensures new authenticated endpoints declare access category.

### Ticket P2-51: Quarterly Access Model Review
**Goal:** Keep policy aligned with trial operations.

**Status:** ✅ Completed (2026-02-27)

**Acceptance Criteria**
- Scheduled review includes role matrix validation, cloud-auth parity, and sampling of access logs.

---

## Post-Completion Operational Next Steps (Required to Sustain Desired State)

The implementation scope in this plan is complete. To maintain the desired operating state over time:

1. Run the quarterly access-model review process and file evidence using:
  - `docs/planning/quarterly-access-model-review.md`
  - `docs/evidence/hardening-phase-5/global/quarterly-access-model-review-template.md`
2. Keep endpoint/matrix parity enforced in CI via:
  - `scripts/check-access-matrix-coverage.sh`
  - `.github/workflows/ci.yml`
3. For every future auth or route-surface change, require cloud parity validation across GCP IAP, Azure Easy Auth, and AWS ALB/Cognito before merge.
4. Treat new authenticated endpoints as blocked from merge until they are added to the access matrix and covered by regression tests.

---

## Cloud Auth Parity Maintenance Checklist

For each change that affects authorization behavior:

1. Validate in local/dev with `AUTH_PROVIDER=auto` and test users.
2. Validate in cloud-staged paths for:
   - GCP IAP headers,
   - Azure Easy Auth headers,
   - AWS ALB/Cognito JWT headers.
3. Confirm identical authorization outcomes for representative personas and routes.

Require parity sign-off across all three providers before merge for changed surfaces.

---

## Definition of Done (Program-Level)

- Satellite users can only see and act on their own site data within assigned projects.
- Coordinating center project roles can see full project data and perform role-appropriate actions.
- Users can hold different roles across projects with correct enforcement.
- Platform admin (global owner equivalent) retains full cross-project visibility and control.
- Behavior is validated as equivalent across GCP, AWS, and Azure auth paths.
