# Project/Site Scoping Baseline Test Catalog (Phase 0)

## Purpose
Baseline regression scenarios for clinical-trial project/user scoping before implementation changes.

## Personas
- **Platform admin (global owner equivalent):** unrestricted cross-project visibility and mutation rights.
- **Project owner/coordinator/reviewer (researcher):** full in-project visibility; role-appropriate writes.
- **Site coordinator:** scoped to assigned institution(s) within a project; allowed operational writes only for in-scope studies.
- **Site viewer:** scoped to assigned institution(s) within a project; read-only.
- **Non-member researcher:** no visibility into project-scoped data.

## Core Access Expectations
- Platform admin remains globally unrestricted.
- Coordinating-center project roles can see all studies in their project.
- Site roles can only see studies where `study.institution_id` matches assigned institution membership.
- Non-members are denied for project-scoped surfaces.
- Unauthorized study reads should return `404` on study-specific endpoints.

## Baseline Test Matrix

### A) Project Discovery and Scope Selection
- `GET /api/projects` returns all projects for platform admin, membership-limited projects for researcher/site users.
- `GET /api/projects/{id}` denies non-members.
- Project summaries (`/summary`, `/milestones`, `/custom-fields`) respect membership and role constraints.

### B) Study List / Lookup / CSV
- `GET /api/studies` with and without `project_id`:
  - platform admin sees global set;
  - coordinating member sees full project set;
  - site roles see only matching `institution_id` rows;
  - non-member sees none.
- `GET /api/studies.csv` mirrors list scoping exactly.
- `GET /api/study-uid/{studyUID}` and `GET /api/studies/{id}` return `404` for out-of-scope studies.

### C) Study Diagnostics and Derived Reads
- `GET /api/studies/{id}/diagnostics`, `/series`, `/processing-summary`, `/routing-log`, `/audit`, `/audit.csv`:
  - in-scope access succeeds;
  - out-of-scope access returns `404`.
- `GET /api/studies/{studyUID}/dicom-tags` and `/anonymization-diff` follow the same scope behavior.

### D) Shares and Export Surfaces
- `GET /api/studies/{id}/shares` and `GET /api/shares/{shareID}/downloads` enforce study/project scope.
- `GET /api/shares`, `GET /api/export-shares.csv`, `GET /api/export-analytics`:
  - platform admin global;
  - non-admin researchers require safe project scoping and see only in-scope project data.

### E) Stats Surfaces
- `GET /api/stats*`, `GET /api/storage/stats`, `GET /api/system/health-summary`:
  - platform admin global;
  - non-admin researchers constrained to authorized project scope.
- Validate trend endpoints (`protocol-trend`, `phi-trend`, `modality-trend`, `label-usage`, `source-trend`) are scope-safe.

### F) Audit Surfaces
- `GET /api/audit`, `GET /api/audit.csv`, `GET /api/audit/actors`:
  - platform admin global;
  - researchers constrained to project-scoped events only.

### G) Institution / Routing / Destination Metadata
- `GET /api/institutions*`, `GET /api/destinations*`, `GET /api/routing-rules*`:
  - non-admin researchers receive only data for authorized project context.
- Ensure no cross-project leakage through aggregate stats endpoints.

### H) Write Baseline (Pre-Phase-2)
- Confirm all current `adminOnly` routes remain platform-admin-gated before capability migration work begins.
- Validate site/coordinating project roles cannot mutate through platform-admin routes yet.

## Required Fixture Coverage
- At least 2 projects.
- At least 2 institutions/sites in one project.
- Studies across both projects and both institutions.
- Users representing each persona, including one non-member.

## Sign-Off Checklist
- [ ] Read surfaces covered: project list/detail, studies, diagnostics, CSV, shares, stats, audit.
- [ ] Write baseline documented for future capability migration.
- [ ] Platform admin global behavior verified and explicitly preserved.
- [ ] Site-scoped institution filtering validated for list and detail endpoints.
- [ ] Non-member denial behavior validated across representative endpoints.
