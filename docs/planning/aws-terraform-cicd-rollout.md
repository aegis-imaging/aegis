# AWS Terraform CI/CD Rollout (Safe Path)

Goal: move `terraform/aws` from manual apply to controlled CI/CD with explicit safeguards, while keeping production risk low.

## Current State

- App deploys are already CI/CD via `.github/workflows/deploy-aws.yml` on push to `develop`.
- AWS infrastructure (`terraform/aws`) is manual by policy.

## Target State

- Terraform plan runs automatically for AWS infra changes.
- Terraform apply requires protected-environment approval.
- No silent/automatic destructive infra changes.

## Guardrails (Required)

1. **Path filtering**
   - Trigger infra workflow only when files under `terraform/aws/**` change.
2. **Two-step pipeline**
   - `plan` always first, upload `tfplan` artifact.
   - `apply` consumes the exact saved plan artifact (no re-planning drift).
3. **Environment approval gate**
   - Use GitHub Environment `aws-prod` with required reviewers.
4. **Concurrency lock**
   - One AWS infra deployment at a time (`concurrency` group).
5. **Apply only from protected branch**
   - Apply only for `develop` pushes (or manual dispatch with explicit confirmation).
6. **Pinned Terraform version**
   - Pin Terraform and providers; fail if lockfile drifts unexpectedly.
7. **No `-lock=false`**
   - Keep backend state locking enabled.

## Recommended Rollout Phases

### Phase 1 (Now): Plan-Only CI

- Add workflow with:
  - `on: pull_request` and `on: push` path-filtered to `terraform/aws/**`
  - `terraform fmt -check`, `init`, `validate`, `plan`
  - Upload plan artifact + short summary comment
- No auto-apply yet.

### Phase 2: Manual Apply with Approval

- Add `workflow_dispatch` input `confirm_apply`.
- `apply` job requires `environment: aws-prod` approval.
- Apply uses previously generated plan artifact.

### Phase 3: Auto-Apply on Develop (With Approval)

- Enable `push` auto-apply only for `develop` and only when `terraform/aws/**` changed.
- Keep `environment: aws-prod` approval so a human still approves.

## Proposed Workflow (Production-Safe)

```yaml
name: Terraform AWS Infra

on:
  pull_request:
    branches: [develop]
    paths:
      - terraform/aws/**
  push:
    branches: [develop]
    paths:
      - terraform/aws/**
  workflow_dispatch:
    inputs:
      confirm_apply:
        description: "Type APPLY to allow terraform apply"
        required: false
        default: ""

concurrency:
  group: terraform-aws-prod
  cancel-in-progress: false

permissions:
  contents: read
  id-token: write

jobs:
  plan:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: terraform/aws
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: 1.9.8
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-region: us-east-1
          role-to-assume: arn:aws:iam::<ACCOUNT_ID>:role/github-actions-terraform-aws
      - run: terraform init
      - run: terraform fmt -check
      - run: terraform validate
      - run: terraform plan -input=false -out=tfplan
      - uses: actions/upload-artifact@v4
        with:
          name: terraform-aws-tfplan
          path: terraform/aws/tfplan

  apply:
    if: |
      github.ref == 'refs/heads/develop' &&
      github.event_name != 'pull_request' &&
      (github.event_name != 'workflow_dispatch' || github.event.inputs.confirm_apply == 'APPLY')
    needs: plan
    runs-on: ubuntu-latest
    environment: aws-prod
    defaults:
      run:
        working-directory: terraform/aws
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: 1.9.8
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-region: us-east-1
          role-to-assume: arn:aws:iam::<ACCOUNT_ID>:role/github-actions-terraform-aws
      - run: terraform init
      - uses: actions/download-artifact@v4
        with:
          name: terraform-aws-tfplan
          path: terraform/aws
      - run: terraform apply -input=false tfplan
```

## IAM for GitHub OIDC Role

Use a dedicated role for Terraform with:
- trust policy bound to this repository and workflow
- least privilege for resources managed in `terraform/aws`
- separate role from app-deploy role

## Definition of Done

- PR modifying only `terraform/aws/**` produces plan automatically.
- Apply cannot run without `aws-prod` environment approval.
- Apply uses saved plan artifact.
- Re-running workflow with no infra changes shows no-op.

## Recommendation

Start with **Phase 1 plan-only** for one week, then enable **Phase 2**. Promote to **Phase 3** after 3-5 successful approved applies with no rollback.
