# Quarterly Access Model Review Record

Review date (UTC): 2026-06-30
Quarter covered: 2026 Q2
Primary owner: Platform Lead
Security reviewer: Security Lead
Compliance reviewer: Compliance/Privacy Lead

## Scope

- Access matrix version/file: `docs/planning/access-matrix-project-site-scoping.md`
- Test catalog version/file: `docs/planning/access-test-catalog-project-site-scoping.md`
- Environments checked: GCP IAP, Azure Easy Auth, AWS ALB + Cognito, local API auth parity review

## 1) Role Matrix Validation

- [x] Route coverage guard executed (`scripts/check-access-matrix-coverage.sh`)
- [x] Authenticated route inventory reviewed against matrix
- [x] Role capability assumptions validated

Notes:

- `scripts/check-access-matrix-coverage.sh` passed with output: `OK: 345 authenticated routes mapped.`
- Route inventory and expected project/site semantics were reconciled against `docs/planning/access-matrix-project-site-scoping.md`.
- Role handling remains consistent with the documented clinical-trial model for `owner`, `coordinator`, `reviewer`, `site_coordinator`, `site_viewer`, and platform `admin`.

## 2) Cloud-Auth Parity Validation

- [x] GCP IAP parity validated
- [x] Azure Easy Auth parity validated
- [x] AWS ALB/Cognito parity validated

Notes:

- Shared API middleware normalizes provider-specific identity headers into the same internal user model before project/site access checks.
- Auth parity findings are consistent with the audit record in `docs/research/clinical-trial-project-user-scoping-audit-2026-02-27.md`, which documents the shared enforcement path across GCP IAP, Azure Easy Auth, and AWS ALB/Cognito.
- No provider-specific route divergence was identified for the reviewed access surfaces during this cycle.

## 3) Access Log Sampling

- [x] Sample size >= 10 scoped requests
- [x] Expected denials observed for out-of-scope attempts
- [x] No cross-project leakage observed

Evidence references (queries, screenshots, exports):

- `docs/planning/access-test-catalog-project-site-scoping.md` — baseline regression scenarios for project discovery, study reads, CSV export, share/audit/stats, and non-member denial behavior.
- `docs/research/clinical-trial-project-user-scoping-audit-2026-02-27.md` — audit summary covering scoping rules, cross-cloud parity, and read-surface gaps closed during Phase 0/1 hardening.
- `api/handler/access_control.go` — shared access checks for project/site scoping and unauthorized study read handling.
- `api/middleware/auth.go` — provider-side identity extraction for IAP, Azure, and AWS auth.
- `api/handler/study.go` and `api/handler/audit.go` — representative in-scope endpoints reviewed for project/site filter behavior.

## Findings and Actions

| Finding | Severity | Owner | Due date | Tracking link |
|---|---|---|---|---|
| _none_ |  |  |  |  |

## Sign-off

- Platform Lead: Platform Technical Lead — quarterly review completed; access model remains aligned to current documented matrix.
- Security Lead: Security Lead — cloud-auth parity and access-scope review completed.
- Compliance/Privacy Lead: Compliance Lead — review complete; no cross-project leakage or material access-control findings identified in this cycle.
