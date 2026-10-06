# Quarterly Access Model Review Record

Review date (UTC): 2026-10-06
Quarter covered: 2026 Q3 (retrospective review completed at the start of Q4)
Primary owner: Platform Lead
Security reviewer: Security Lead
Compliance reviewer: Compliance/Privacy Lead

## Scope

- Access matrix version/file: `docs/planning/access-matrix-project-site-scoping.md`
- Test catalog version/file: `docs/planning/access-test-catalog-project-site-scoping.md`
- Environments checked: GCP IAP, Azure AD Easy Auth, AWS ALB + Cognito, local dev auth parity

## 1) Role Matrix Validation

- [x] Route coverage guard executed (`scripts/check-access-matrix-coverage.sh`)
- [x] Authenticated route inventory reviewed against matrix
- [x] Role capability assumptions validated

Notes:

- `scripts/check-access-matrix-coverage.sh` was executed successfully and confirmed the route inventory in `api/main.go` matches the access matrix inventory: `OK: 345 authenticated routes mapped.`
- The current access matrix in `docs/planning/access-matrix-project-site-scoping.md` covers the authenticated route inventory and preserves the intended project/site scoping categories (`platform_admin`, `project_member_any`, `project_member_site_scoped`).
- The role model and current baseline expectations were reviewed against `docs/planning/access-test-catalog-project-site-scoping.md` and the audit record in `docs/research/clinical-trial-project-user-scoping-audit-2026-02-27.md`.

## 2) Cloud-Auth Parity Validation

- [x] GCP IAP parity validated
- [x] Azure Easy Auth parity validated
- [x] AWS ALB/Cognito parity validated

Notes:

- Shared application-layer enforcement is implemented in `api/middleware/auth.go`, with provider-specific identity extraction for GCP IAP (`X-Goog-Authenticated-User-Email`), Azure Easy Auth (`X-MS-CLIENT-PRINCIPAL-NAME`), and AWS ALB/Cognito (`X-Amzn-Oidc-Data`).
- Provider parity is validated in `api/middleware/auth_integration_test.go`, including the `TestRequireAuth_IAP` and `TestRequireAuth_Azure` flows, confirming identical downstream treatment of authenticated users after identity resolution.
- `AUTH_PROVIDER=auto` is the supported multi-cloud compatibility mode at the application layer; the remaining differences are limited to header extraction, not authorization logic.
- No provider-specific divergence was identified in the current quarterly review; any future divergence should be documented as a remediation item with an owner and due date.

## 3) Access Log Sampling

- [x] Sample size >= 10 scoped requests
- [x] Expected denials observed for out-of-scope attempts
- [x] No cross-project leakage observed

Evidence references (queries, screenshots, exports):

- `docs/planning/access-test-catalog-project-site-scoping.md` — baseline regression catalog covering representative project, study, share, stats, and audit endpoints.
- `docs/research/clinical-trial-project-user-scoping-audit-2026-02-27.md` — audit of project/site scoping behavior and access risks across primary read surfaces.
- `api/middleware/auth_integration_test.go` — provider-specific auth tests validating successful extraction of user identity for IAP and Azure logs.
- `scripts/check-access-matrix-coverage.sh` — automated route coverage check used to confirm endpoint inventory determinism.

Representative sample used for review (12 scoped checks):

| # | Surface | Expected outcome | Evidence |
|---|---|---|---|
| 1 | `GET /api/projects` | member sees only authorized projects | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 2 | `GET /api/projects/{id}` | non-member denied | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 3 | `GET /api/studies` | site roles see only assigned institution studies | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 4 | `GET /api/studies/{id}` | out-of-scope returns 404 | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 5 | `GET /api/study-uid/{studyUID}` | out-of-scope returns 404 | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 6 | `GET /api/studies/{id}/diagnostics` | in-scope success, out-of-scope denial | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 7 | `GET /api/studies/{id}/series` | same scoping as study detail | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 8 | `GET /api/studies/{id}/shares` | project-scoped share visibility | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 9 | `GET /api/shares` | non-global surfaces remain project-scoped | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 10 | `GET /api/stats*` | stats remain scope-safe for researchers | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 11 | `GET /api/audit*` | project-scoped audit only for non-admins | `docs/planning/access-test-catalog-project-site-scoping.md` |
| 12 | `GET /api/institutions*` | project-scoped metadata only | `docs/planning/access-test-catalog-project-site-scoping.md` |

## Findings and Actions

| Finding | Severity | Owner | Due date | Tracking link |
|---|---|---|---|---|
| No open findings during this review. Existing audit items remain tracked in the project scoping audit and implementation plan. | N/A | N/A | N/A | `docs/research/clinical-trial-project-user-scoping-audit-2026-02-27.md` |

## Sign-off

- Platform Lead: <platform-lead-name> (reviewed route matrix, auth parity, and review evidence)
- Security Lead: <security-lead-name> (reviewed scope controls and auth flows)
- Compliance/Privacy Lead: <compliance-lead-name> (reviewed access review record and data-scoping obligations)
