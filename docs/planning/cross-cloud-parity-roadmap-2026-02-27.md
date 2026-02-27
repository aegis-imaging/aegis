# Cross-Cloud Parity Roadmap — 2026-02-27

## Context

This planning note captures current CI/CD and Terraform parity status across GCP, AWS, and Azure for Anonymization & Exchange Gateway for Imaging Studies (AEGIS), plus the next recommended execution sequence.

Recent completed milestones:
- AWS Terraform Phase 3 implemented (`Terraform AWS`) with approval-gated apply via `aws-prod` environment.
- AWS failure signal workflow added (`Terraform AWS Failure Alert`).
- Azure Terraform formatting regression fixed (`terraform/azure/auth.tf`).

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
- **Azure**: `terraform-azure.yml` currently plans and applies on push (no explicit environment approval gate).

Status: **Partial parity** (AWS has stronger apply guardrails than Azure).

### Failure Visibility / Ops Signal
- **AWS**: Dedicated failure workflow exists (`terraform-aws-failure-alert.yml`) triggered from `workflow_run`.
- **Azure**: No equivalent Terraform failure-alert workflow yet.
- **GCP**: No equivalent pipeline-level failure-alert workflow in GitHub Actions (Cloud Build notifications/logs used separately).

Status: **Gap**.

### Auth Posture in CI/CD
- **Azure deploy**: OIDC-based auth in workflow.
- **AWS deploy**: currently uses long-lived access key secrets.

Status: **Gap** (prefer OIDC parity).

---

## Major Gaps to Address Next

1. **Azure Terraform apply gate parity**
   - Add plan/apply split with apply protected by GitHub Environment approval (mirror AWS model).

2. **Cross-cloud Terraform failure alert parity**
   - Add Azure Terraform failure-alert workflow similar to AWS.
   - Define equivalent GCP signal path (GitHub- or Cloud Build-based) with direct run/build URL and triage pointers.

3. **AWS deploy auth hardening**
   - Migrate `deploy-aws.yml` from static IAM user secrets to OIDC role assumption.

4. **Unified post-deploy smoke gates**
   - Standardize cloud-specific smoke checks after deploy (health endpoint + critical route checks).

---

## Recommended Execution Sequence

### Phase A — Terraform Safety Parity
1. Implement approval-gated apply flow for Azure Terraform (`terraform-azure.yml`).
2. Add Azure Terraform failure-alert workflow (`workflow_run` pattern).
3. Confirm runbook updates for Azure equivalent of `aws-prod` approval operation.

### Phase B — Credentials & Identity Hardening
4. Introduce AWS OIDC role for deploy workflow and remove static key dependency in `deploy-aws.yml`.
5. Validate least-privilege policy boundaries and rollout plan.

### Phase C — Operational Consistency
6. Add per-cloud post-deploy smoke checks and standardized pass/fail summaries.
7. Add a concise cross-cloud operator runbook section linking all deployment/failure entry points.

---

## Guardrails for Implementation

- Preserve current stable behavior unless explicitly changing workflow logic.
- Keep rollouts incremental with one cloud/workflow change per PR where practical.
- Prefer plan artifact reuse for apply stages where supported.
- Maintain protected-branch + environment-approval semantics for production-impacting apply paths.

---

## Immediate Next Candidate (if approved)

**Implement Azure Terraform approval-gated apply parity**:
- split Azure Terraform into plan + apply jobs,
- require environment approval before apply,
- keep manual dispatch support,
- preserve OIDC authentication.
