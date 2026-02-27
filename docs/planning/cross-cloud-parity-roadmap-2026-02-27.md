# Cross-Cloud Parity Roadmap — 2026-02-27

## Context

This planning note captures current CI/CD and Terraform parity status across GCP, AWS, and Azure for Anonymization & Exchange Gateway for Imaging Studies (AEGIS), plus the next recommended execution sequence.

Recent completed milestones:
- AWS Terraform Phase 3 implemented (`Terraform AWS`) with approval-gated apply via `aws-prod` environment.
- AWS failure signal workflow added (`Terraform AWS Failure Alert`).
- Azure Terraform now runs plan + approval + apply with protected environment gate (`azure-prod`) and plan-artifact reuse.
- Azure Terraform and AWS Terraform plan steps use `pipefail` to avoid masked plan failures.
- Azure Terraform permission runbook automation scripts added and validated end-to-end (PR #348).
- Azure Terraform workflow run succeeded after permission remediation (`22497085026`).
- GCP failure-signal parity added via GitHub Actions workflow (`GCP Cloud Build Failure Alert`) with direct Cloud Build run URLs.
- Terraform AWS/Azure failure alert workflows scoped to `develop` failures only to reduce non-prod alert noise (PR #355).
- GCP failure checker default lookback tuned to 30 minutes for scheduled runs (manual checks remain 60-minute default) to reduce duplicate alert windows in recurring schedules.

---

## Current State Snapshot

### Application Deploy Automation
- **GCP**: Cloud Build (`cloudbuild.yaml`) auto-build and deploy on push to `develop`.
- **AWS**: GitHub Actions (`deploy-aws.yml`) auto-build and deploy on push to `develop`.
- **Azure**: GitHub Actions (`deploy-azure.yml`) auto-build and deploy on push to `develop`.

Status: **Strong parity**.

### Infrastructure Automation (Terraform)
- **GCP**: Terraform infra apply runs via Cloud Build trigger on `terraform/infra/**` changes (auto-apply model).
- **AWS**: `terraform-aws.yml` now supports plan + apply with approval-gated apply (Phase 3), plus manual dispatch path.
- **Azure**: `terraform-azure.yml` now uses plan + protected approval + apply flow with artifact handoff and OIDC.

Status: **Strong parity** for AWS/Azure Terraform guardrails.

### Failure Visibility / Ops Signal
- **AWS**: Dedicated failure workflow exists (`terraform-aws-failure-alert.yml`) triggered from `workflow_run`.
- **Azure**: Dedicated failure workflow exists (`terraform-azure-failure-alert.yml`) triggered from `workflow_run`.
- **GCP**: Dedicated failure workflow exists (`gcp-cloud-build-failure-alert.yml`) that checks recent failed Cloud Build runs and surfaces direct run URLs.

Status: **Strong parity**.

### Auth Posture in CI/CD
- **Azure deploy**: OIDC-based auth in workflow.
- **AWS deploy**: OIDC-only auth in workflow (`deploy-aws.yml`); legacy static key fallback removed.

Status: **Strong parity**.

---

## Major Gaps to Address Next

No remaining high-priority parity gaps in this phase.

---

## Recommended Execution Sequence

### Phase D — Remaining Parity Closure
Completed:
1. Normalize post-deploy smoke summary formatting across GCP/AWS/Azure workflows.

---

## Guardrails for Implementation

- Preserve current stable behavior unless explicitly changing workflow logic.
- Keep rollouts incremental with one cloud/workflow change per PR where practical.
- Prefer plan artifact reuse for apply stages where supported.
- Maintain protected-branch + environment-approval semantics for production-impacting apply paths.

---

## Immediate Next Candidate (if approved)

**Phase E — Reliability hardening (targeted, low-risk)**:
- keep AWS deploy OIDC-only with optional role-ARN override and account-derived fallback (validated success run `22501374876`),
- keep Azure deploy path on OIDC with latest successful validation (`22501374891`),
- keep GCP Cloud Build failure signal workflow validated end-to-end with OIDC (`22502258584`),
- continue incremental runbook tightening and alert-noise reduction without changing core deploy logic,
- next: periodically review failure-signal summaries for signal quality (false-positive rate, branch scope, and triage clarity) and apply low-risk wording/runbook refinements as needed.
- next: continue periodic signal-quality reviews and only ship incremental, low-risk tuning when repeated duplicate windows or unclear triage summaries are observed.

### Phase E Status Note — Post-merge signal snapshot (2026-02-27)

- **Terraform AWS Failure Alert**: recent runs observed as `skipped` on `develop` (e.g., `22495628324`, `22495503435`, `22494768989`), consistent with alert-on-failure-only behavior.
- **Terraform Azure Failure Alert**: recent sequence includes historical `failure` runs (`22497034941`, `22496444844`, `22495765287`, `22495674643`) followed by `skipped` (`22497146718`) after failure conditions cleared.
- **GCP Cloud Build Failure Alert**: manual validation remains green (`22502258584` success), with scheduled run `22502537637` continuing to detect real failed Cloud Build events when present.
- **Assessment**: current Phase E hardening is operating as intended; maintain periodic signal reviews and keep future changes low-risk and incremental.
