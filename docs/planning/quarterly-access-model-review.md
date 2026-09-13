# Quarterly Access Model Review (P2-51)

This runbook defines the recurring quarterly review process for project/site authorization in **Anonymization & Exchange Gateway for Imaging Studies (AEGIS)**.

## Purpose

Verify that access control behavior remains aligned with the clinical-trial scoping model across backend, frontend, and cloud auth providers.

## Cadence and Owners

- Cadence: quarterly (once per calendar quarter)
- Primary owner: Platform Lead
- Required reviewers: Security Lead, Compliance/Privacy Lead

## Required Inputs

- Current access matrix: `docs/planning/access-matrix-project-site-scoping.md`
- Current test catalog: `docs/planning/access-test-catalog-project-site-scoping.md`
- Latest pipeline/access-related runbooks in `docs/runbooks/`
- Audit sample exports from `/api/audit` and `/api/studies` (scoped by project/site personas)

## Review Checklist

### 1) Role matrix validation

- [ ] Reconcile `api/main.go` authenticated routes against access matrix
- [ ] Run route coverage guard: `./scripts/check-access-matrix-coverage.sh`
- [ ] Confirm role capability expectations still match implementation (owner/coordinator/reviewer/site_coordinator/site_viewer/admin)

### 2) Cloud-auth parity validation

- [ ] Validate equivalent outcomes for GCP IAP identity headers
- [ ] Validate equivalent outcomes for Azure Easy Auth identity headers
- [ ] Validate equivalent outcomes for AWS ALB/Cognito identity headers
- [ ] Record any provider-specific divergence and remediation owner

### 3) Access log sampling

- [ ] Sample at least 10 recent researcher requests across read/write/scoped endpoints
- [ ] Confirm expected denials (400/403/404) for out-of-scope access attempts
- [ ] Confirm no cross-project leakage on global surfaces (shares/audit/stats)
- [ ] Attach evidence links/queries used for sampling

## Evidence Capture

Store completed review records in:

- `docs/evidence/hardening-phase-5/global/`

Use filename pattern:

- `YYYY-MM-DD_quarterly-access-model-review.md`

Start from template:

- `docs/evidence/hardening-phase-5/global/quarterly-access-model-review-template.md`

## Exit Criteria

- All checklist items are marked complete
- Any findings have tracked remediation tickets with owners/dates
- Platform Lead and Security/Compliance reviewers sign off
