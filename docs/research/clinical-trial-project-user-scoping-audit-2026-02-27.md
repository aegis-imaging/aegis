# Clinical Trial Project/User Scoping Audit — 2026-02-27

## Scope

Audit objective: assess current codebase support for a clinical-trial operating model in **Anonymization & Exchange Gateway for Imaging Studies (AEGIS)** where:

- multiple satellite enrollment sites upload participant brain MRI data,
- satellite personnel can only see their own site’s data,
- coordinating center PI/site can see all data within their project,
- multiple projects exist in the same platform,
- users can hold different roles across different projects,
- one global owner-level operator (you) retains cross-project visibility.

This is a read-only audit. No implementation changes were made.

---

## Executive Summary

Current implementation has strong foundational support for project and site scoping via `project_members` and `institution_id`-based visibility, and this behavior is cloud-agnostic at the application layer.

However, enforcement is currently **inconsistent across endpoints**:

1. **Core study list/detail paths are scoped correctly** for `researcher` users by project membership and site institution.
2. **Several other read endpoints are not scoped** and can expose cross-project data to authenticated non-admin users.
3. **Most write endpoints remain platform-`admin` gated**, so project-level roles (`owner`, `coordinator`, `site_coordinator`) are not fully leveraged for operational workflows.
4. **Global owner-level access exists today as platform role `admin`** (not as a distinct `owner` platform role), and should be retained.

---

## Evidence of Current Support

## 1) Role model supports clinical-trial project/site semantics

- `project_members` migration defines project-scoped roles:
  - coordinating-center roles: `owner`, `coordinator`, `reviewer`
  - site roles: `site_coordinator`, `site_viewer`
- `institution_id` null/non-null determines all-project-in-project vs site-only visibility intent.

Evidence:
- `api/migrate/migrations/066_create_project_members.sql`
- `api/model/project_member.go`

## 2) Multi-role across projects is supported

- Membership is per `(project_id, admin_user_id)`; same user can have different roles in different projects.

Evidence:
- `api/migrate/migrations/066_create_project_members.sql` (`UNIQUE(project_id, admin_user_id)`)
- `api/model/project_member.go`

## 3) Researcher project visibility is implemented

- `admin_users.role='researcher'` exists and project list is filtered to membership.

Evidence:
- `api/migrate/migrations/068_add_researcher_role.sql`
- `api/handler/project.go` (`ListProjects` routes researchers to `ListProjectsForResearcher`)
- `api/model/project.go` (`ListProjectsForResearcher`)

## 4) Site-level study scoping is implemented on primary study endpoints

- For researchers, `project_id` is required.
- If membership has `institution_id`, handlers force study filter `institution_id=<member institution>`.

Evidence:
- `api/handler/access_control.go` (`requireResearcherProjectScope`, `requireStudyReadAccessByID/UID`)
- `api/handler/study.go` (`ListStudies`, `GetStudy`, `GetStudyByUID`)
- `api/handler/study_csv.go` (scoped export)

## 5) Cross-cloud parity for auth plumbing exists

- Same API middleware is used across clouds; only identity header source differs (`IAP`, `Azure`, `AWS ALB`), then all map into `admin_users`.

Evidence:
- `api/middleware/auth.go` (`extractUser`, `iapUser`, `azureUser`, `awsUser`)
- `api/config/config.go` (`AUTH_PROVIDER`, `AUTH_ENABLED`)

---

## Gaps and Risks vs Target Model

## A) Inconsistent scoping across non-primary read endpoints (high priority)

Some authenticated endpoints read study/project-sensitive data without calling shared project/site access checks.

Examples observed:
- study diagnostics without `requireStudyReadAccess*`: `api/handler/study_diagnostics.go`
- study series without scoped check: `api/handler/series.go`
- DICOM tag inspection without scoped check: `api/handler/dicom_tags.go`
- anonymization diff without scoped check: `api/handler/anon_diff.go`
- share listing/analytics global paths without researcher scoping: `api/handler/export.go`
- global stats/audit/institution endpoints not researcher-scoped by default:
  - `api/handler/stats.go`
  - `api/handler/audit.go`
  - `api/handler/institution.go`

Impact: a site-scoped user may access broader metadata than intended, depending on endpoint usage.

## B) Project-member write roles are not yet reflected in route-level write authorization (high priority)

- Most mutating routes are still wrapped by `RequireRole("admin")` in `api/main.go`.
- This bypasses intended project-member semantics where `owner/coordinator/site_coordinator` should be able to perform certain project-scoped actions.

Evidence:
- `api/main.go` (`adminOnly := middleware.RequireRole("admin", ...)` applied broadly)
- role capability helpers exist but are underused globally: `api/model/project_member.go` (`CanWrite`, `CanApprove`, `CanManageProject`)

Impact: coordinating-center and site operations require platform admin, which does not match the trial operating model.

## C) Frontend authorization model is still primarily `isAdmin`-based (medium priority)

- Dashboard action visibility is largely controlled by `isAdmin = currentUser?.role === 'admin'`.
- Project-member role semantics are partially represented in UI labels/panels but not consistently used for action gating.

Evidence:
- `frontend/admin-dashboard/src/App.tsx`

Impact: even where backend becomes project-role aware, frontend may still hide allowed actions or show incorrect UX.

## D) Upload attribution for site scoping is available in API but not exposed by upload client (medium priority)

- Upload init supports `institution_id`, `institution_slug`, `institution_ae_title` and source-IP fallback.
- Browser upload client currently sends `project_slug`, file metadata, and uploader email only.

Evidence:
- `api/handler/upload.go` (`uploadInitRequest`)
- `client/src/upload/client.ts` (no institution fields in init payload)

Impact: study `institution_id` may be absent/ambiguous for web uploads, weakening site-level visibility guarantees.

---

## Owner-Level Visibility Requirement (Your Added Constraint)

Requirement: retain a single global operator (you) who can see everything across all projects.

Current state:
- This is effectively provided by platform `admin` role today (`IsPlatformAdmin` logic and route wrapping).
- Project-member `owner` is project-scoped, not platform-scoped.

Evidence:
- `api/middleware/project_access.go` (`IsPlatformAdmin` => `admin`/`viewer` bypass)
- `api/middleware/auth.go` (`RequireRole` with admin override)
- `api/model/project_member.go` (`owner` applies per project)

Conclusion:
- Global owner capability is currently represented by platform `admin` and can be retained while introducing stricter project/site scoping for non-admin users.

---

## Multi-Cloud Assessment (GCP / AWS / Azure)

Project/user scoping behavior is implemented in shared Go API logic and shared PostgreSQL schema, so parity is inherently high across clouds.

Cloud-specific differences are limited to identity extraction before user lookup:
- GCP: IAP header
- Azure: Easy Auth header
- AWS: ALB/Cognito JWT header

After identity resolution, the same role/scoping logic applies.

---

## Readiness Against Target Trial Model

## What is already aligned

- Multi-project architecture
- Project membership model with site/center roles
- Site-scoped study list/detail filtering for researcher users
- Cross-cloud parity of auth + API enforcement path
- Global superuser capability (via platform `admin`)

## What is not yet fully aligned

- Uniform endpoint-level scoping for all read surfaces
- Project-role-based mutation authorization (vs platform-admin-only)
- Frontend action gating based on project member access (vs `isAdmin`)
- Deterministic institution attribution in browser upload path

---

## Recommended Next Planning Slice (No Implementation in this document)

1. Build an endpoint-by-endpoint access matrix (`public`, `platform-admin`, `researcher-any-member`, `site-scoped`, `owner/coordinator-only`).
2. Normalize all study/project read endpoints to shared access helpers.
3. Introduce project-member capability checks for project-scoped writes.
4. Update dashboard UI gating to consume capability signals instead of only `isAdmin`.
5. Add upload attribution strategy for satellite web uploads (institution selection/assignment policy).
6. Preserve platform `admin` for global owner-level access.

---

## Notes

- This file is intentionally implementation-neutral and intended as an architecture/authorization audit artifact.
- No code behavior was changed in this step.
