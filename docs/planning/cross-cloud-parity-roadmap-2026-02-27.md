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

1. **Post-deploy smoke consistency**
   - Ensure all cloud deploy workflows emit similarly structured smoke-check summary blocks for quick operator scan.

---

## Recommended Execution Sequence

### Phase D — Remaining Parity Closure
1. Normalize post-deploy smoke summary formatting across GCP/AWS/Azure workflows.

---

## Guardrails for Implementation

- Preserve current stable behavior unless explicitly changing workflow logic.
- Keep rollouts incremental with one cloud/workflow change per PR where practical.
- Prefer plan artifact reuse for apply stages where supported.
- Maintain protected-branch + environment-approval semantics for production-impacting apply paths.

---

## Immediate Next Candidate (if approved)

**Normalize post-deploy smoke summary output**:
- align smoke-check section formatting in GCP, AWS, and Azure deploy workflows,
- ensure each workflow surfaces endpoint health in a scan-friendly summary,
- validate one run per cloud after formatting alignment.
