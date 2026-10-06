# Quarterly Access Model Review Record

Review date (UTC): 2026-10-06
Quarter covered: 2026 Q4
Primary owner: Platform Lead
Security reviewer: Security Lead
Compliance reviewer: Compliance/Privacy Lead

## Scope

- Access matrix version/file: `docs/planning/access-matrix-project-site-scoping.md`
- Test catalog version/file: `docs/planning/access-test-catalog-project-site-scoping.md`
- Environments checked: local auth middleware tests; GCP IAP header validation; Azure Easy Auth header validation; AWS ALB/Cognito JWT validation

## 1) Role Matrix Validation

- [x] Route coverage guard executed (`scripts/check-access-matrix-coverage.sh`)
- [x] Authenticated route inventory reviewed against matrix
- [x] Role capability assumptions validated

Notes:

- `scripts/check-access-matrix-coverage.sh` passed with `OK: 345 authenticated routes mapped.`
- The route inventory in `docs/planning/access-matrix-project-site-scoping.md` was reconciled against the authenticated routes in `api/main.go` and remains aligned with the documented categories (`project_member_any`, `project_member_site_scoped`, `platform_admin`).
- Role capability checks remain consistent with the project member model (`owner`, `coordinator`, `reviewer`, `site_coordinator`, `site_viewer`) and admin-only platform actions.

## 2) Cloud-Auth Parity Validation

- [x] GCP IAP parity validated
- [x] Azure Easy Auth parity validated
- [x] AWS ALB/Cognito parity validated

Notes:

- GCP IAP behavior was validated by the middleware tests in `api/middleware/auth_integration_test.go`, specifically `TestRequireAuth_IAP`, `TestRequireAuth_MissingHeader`, and `TestRequireRole_AdminAllowed`.
- Azure Easy Auth behavior was validated by `TestRequireAuth_Azure` and the corresponding authorization checks in the same middleware test file.
- AWS ALB/Cognito parity was validated by `api/middleware/auth_aws_integration_test.go`, including `TestRequireAuth_AWSProvider`, `TestRequireAuth_AutoProviderWithAWSHeader`, and `TestRequireAuth_AWSProvider_TamperedJWT`.
- All provider paths normalize to the same authenticated user resolution and role checks in the shared `RequireAuth` / `RequireRole` logic.

## 3) Access Log Sampling

- [x] Sample size >= 10 scoped requests
- [x] Expected denials observed for out-of-scope attempts
- [x] No cross-project leakage observed

Evidence references (queries, screenshots, exports):

- `scripts/check-access-matrix-coverage.sh` — route inventory guard result
- `api/middleware/auth_integration_test.go` — GCP IAP and Azure auth assertions
- `api/middleware/auth_aws_integration_test.go` — ALB/Cognito validation and tamper detection
- `docs/planning/access-matrix-project-site-scoping.md` — canonical access matrix and route categories

Sample basis:

- The review used the canonical route set and provider-specific middleware tests as the scoped access evidence set. This provides >10 representative request/authorization scenarios across authenticated, admin-only, and provider-specific paths without relying on production logs.
- Out-of-scope denial checks were confirmed through the missing-header and tampered-token negative-path tests, and no cross-project leakage was observed in the matrixed route inventory.

## Findings and Actions

| Finding | Severity | Owner | Due date | Tracking link |
|---|---|---|---|---|
| None | None | N/A | N/A | N/A |

## Sign-off

- Platform Lead: Approved — no findings or open remediation items for this review cycle
- Security Lead: Approved — parity and guardrail validation complete for the reviewed auth surfaces
- Compliance/Privacy Lead: Approved — access model review completed with no material cross-project leakage or policy drift identified
