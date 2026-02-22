# AEGIS Sprint-Ready Shortlist (Top 5)

Date: 2026-02-20
Updated: 2026-02-22
Source backlog: `docs/planning/next-feature-prioritization.md`

## Completion Status (as of 2026-02-22)

| # | Item | Status | Completed |
|---|------|--------|-----------|
| 1 | GCP Terraform Production Completion | ✅ DONE | 2026-02-21 |
| 2 | Secrets and Credential Hardening | ✅ DONE | 2026-02-21 |
| 3 | Automated Cloud Smoke Test Suite | ✅ DONE | 2026-02-22 (11/11 PASS) |
| 4 | DIMSE PACS E2E Validation Harness | ⏳ Pending | — |
| 5 | AWS HTTPS + Cognito Edge/Auth | ⏳ Pending | — |

## Selection Rationale

This shortlist focuses on the fastest path to pilot readiness and production risk reduction:
- close infra TODOs that block real cloud deployment,
- harden secrets/auth posture,
- automate release confidence checks,
- validate enterprise DIMSE ingress behavior end-to-end.

## 1) GCP Terraform Production Completion

Suggested branch: `feature/terraform-gcp-prod-completion`

### Goal
Make the GCP path deployable and operationally complete for pilot use.

### In Scope
- `terraform/infra/` TODOs:
  - Artifact Registry repository
  - Cloud Run service definitions (API + required sidecars)
  - Cloud Armor policy
  - VPC/subnets/Cloud NAT
  - IAP configuration for admin dashboard
  - Private Service Connect SMTP endpoint (or documented equivalent)
  - Monitoring/alerting baseline
- Required variables/outputs and documentation updates.

### Acceptance Criteria
- `terraform/infra/main.tf` no longer contains placeholder TODO blocks for the listed production resources.
- `terraform init`, `terraform validate`, and `terraform plan` pass cleanly with documented `tfvars`.
- Apply in a dev GCP project deploys API + sidecars with reachable health checks.
- Admin dashboard auth is gated by IAP in deployed environment.
- `SETUP_CHECKLIST.md` includes exact deployment verification steps for the new infra resources.

### Test Plan
- Static: `terraform fmt -check`, `terraform validate`.
- Deploy: `terraform apply` in isolated dev project.
- Runtime:
  - `GET /healthz` from API returns healthy DB/storage/services map.
  - Upload one synthetic DICOM study through upload portal.
  - Verify study appears in admin dashboard and can be approved/exported.

## 2) Secrets and Credential Hardening (Terraform + Runtime)

Suggested branch: `feature/secrets-hardening-infra`

### Goal
Remove static credential patterns and enforce secret-manager driven runtime config.

### In Scope
- Replace hardcoded/placeholder DB credentials in Terraform modules.
- Use cloud-native secret services (GCP Secret Manager, AWS Secrets Manager) patterns.
- Document bootstrap and rotation procedure.

### Acceptance Criteria
- No static DB password placeholders remain in Terraform resources for deployed environments.
- API/sidecar runtime config reads credentials from secret manager paths/references.
- Rotation runbook exists in docs and is tested once in dev.
- CI/lint guard added to fail if known insecure placeholder strings are introduced.

### Test Plan
- Static grep guard in CI (e.g., reject `CHANGE_ME` credentials in infra code).
- Deploy updated infra and verify API DB connectivity succeeds.
- Rotate secret once in dev and confirm service recovers/restarts correctly.

## 3) Automated Cloud Smoke Test Suite

Suggested branch: `feature/cloud-smoke-test-suite`

### Goal
Convert manual pilot checks into a repeatable gate.

### In Scope
- New smoke harness (scripts + docs) for deployed environment.
- Minimal critical flow checks:
  - API health
  - Auth identity endpoint
  - Upload/init/complete path
  - One pipeline progression check
  - Export share create/redeem/download basic check
- Exit non-zero on failures.

### Acceptance Criteria
- Single command executes cloud smoke suite against target environment.
- Suite produces human-readable pass/fail summary with failed-step context.
- Suite integrated into pre-pilot checklist and documented in `SETUP_CHECKLIST.md`.
- At least one CI/manual workflow can run suite with environment secrets.

### Test Plan
- Local dry run (mock or dev stack target).
- Cloud dev environment run with real credentials.
- Intentionally break one dependency and verify smoke suite fails fast with clear diagnostics.

## 4) DIMSE PACS End-to-End Validation Harness + Runbook

Suggested branch: `feature/dimse-pacs-e2e-validation`

### Goal
Prove enterprise ingress reliability from PACS sender through AEGIS ingest and retry controls.

### In Scope
- Repeatable DIMSE sender test harness (e.g., `pynetdicom` sender or `storescu` based).
- Scenario coverage:
  - successful C-STORE ingest
  - transient API failure leading to retry queue entry
  - replay/process controls restoring ingestion
  - dead-letter path + operator recovery
- Operator runbook with expected API/health/queue signals.

### Acceptance Criteria
- Harness can send at least one deterministic synthetic study to DIMSE receiver.
- All four scenarios above have reproducible steps and expected outcomes.
- Runbook links specific endpoints (`/healthz`, `/ingest/retry*`) and expected fields.
- Results captured in doc-friendly output format for pilot evidence.

### Test Plan
- Automated harness run in Docker Compose dev stack.
- One scripted fault-injection test (API unavailable during association release).
- Verify queue/dead-letter metrics and action endpoints produce expected transitions.

## 5) AWS HTTPS + Cognito Edge/Auth Completion

Suggested branch: `feature/terraform-aws-https-cognito`

### Goal
Close AWS auth/edge TODOs so AWS path is production-credible.

### In Scope
- `terraform/aws/main.tf` TODOs:
  - ACM-backed HTTPS listener
  - ALB listener/auth integration
  - Cognito user pool/client/domain resources
- Redirect + protected route behavior docs.

### Acceptance Criteria
- ALB serves HTTPS with ACM certificate.
- Admin/API routes requiring auth are protected by Cognito at ALB.
- Terraform plan/apply works with documented required inputs.
- AWS deployment verification section added to `SETUP_CHECKLIST.md`.

### Test Plan
- Terraform validate/plan/apply in AWS dev account.
- Verify HTTP→HTTPS redirect.
- Verify unauthenticated request is blocked and authenticated request passes.
- Verify API `AUTH_PROVIDER=aws` flow and `/api/auth/me` behavior.

## Suggested Sprint Sequence

1. GCP Terraform Production Completion  
2. Secrets and Credential Hardening  
3. Automated Cloud Smoke Test Suite  
4. DIMSE PACS End-to-End Validation Harness  
5. AWS HTTPS + Cognito Completion

## Definition of Done (applies to each item)

- Code merged to `develop` via feature PR.
- Automated tests/smoke checks included and passing.
- Docs updated (`CLAUDE.md` + `SETUP_CHECKLIST.md` when behavior changes).
- Deployment/runbook steps are reproducible by another engineer without tribal context.
