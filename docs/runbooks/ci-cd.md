# CI/CD

Every deployment runs from GitHub Actions on merge to `develop`. Nothing is
deployed by hand; the only human step is approving an infrastructure apply.
All three clouds follow the same pattern with OIDC authentication — no
long-lived keys or secrets in GitHub beyond the tfvars files.

| Workflow | Trigger | Does | Gate |
|---|---|---|---|
| `ci.yml` | pull request | build, vet, test, typecheck, terraform fmt/validate | — |
| `terraform-gcp.yml` / `-aws.yml` / `-azure.yml` | PR: plan · `develop` push touching that cloud's `terraform/**`: plan → apply | infrastructure | environment `gcp-prod` / `aws-prod` / `azure-prod` required reviewers |
| `deploy-gcp.yml` / `-aws.yml` / `-azure.yml` | `develop` push (code paths) | build + push every image, roll the services that exist, DIMSE VM if configured, health check | — |
| Cloudflare Pages | `develop` push touching `frontend/landing/**` | builds and publishes aegisimaging.ai | — |

Manual runs: Actions → workflow → **Run workflow**. For the terraform
workflows, leave `confirm_apply` empty for plan-only, or type `APPLY`.

The AWS and Azure workflows are no-ops until `AWS_CI_ENABLED` /
`AZURE_CI_ENABLED` repository variables are `true`, so they can be merged
before the clouds are bootstrapped.

## Public-repository rules

The repository is public, so workflow logs and artifacts are world-readable.

- `scripts/terraform_ci.sh` runs plan/apply in `-json` mode and prints
  resource-level messages only (what changes, never attribute values), and no
  plan file is uploaded as an artifact — plan files contain every variable in
  cleartext.
- The approver still needs the full plan. Set `GCP_PLAN_ARCHIVE`,
  `AWS_PLAN_ARCHIVE` or `AZURE_PLAN_ARCHIVE` to a location in the private
  state bucket (`gs://<bucket>/plans`, `s3://<bucket>/plans`,
  `https://<account>.blob.core.windows.net/tfstate/plans`) and the plan step
  writes `<run id>-plan.txt` there with every attribute change; the step
  summary names the object. Read it in the cloud console before approving.
- Cloud account identifiers are masked: the AWS credentials action masks the
  account ID, Azure identifiers are secrets, and the GCP workflows mask
  `GCP_PROJECT_ID`.
- `scripts/gcp_cloud_run_deploy.sh` prints a failed revision's logs; keep
  application logs free of secrets.

## One-time bootstrap

### GitHub

Settings → Environments: create `gcp-prod`, `aws-prod`, `azure-prod`, each
with yourself as a required reviewer. Removing the reviewers makes applies
fully automatic on merge.

### GCP (already provisioned)

Secrets `GCP_WORKLOAD_IDENTITY_PROVIDER`, `GCP_SERVICE_ACCOUNT`,
`GCP_TERRAFORM_TFVARS`; variables `GCP_PROJECT_ID`, `GCP_REGION`,
`GCP_DWV_URL`, `GCP_OHIF_URL`, `GCP_API_URL`, `GCP_ADMIN_URL`, optional
`GCP_DIMSE_INSTANCE` + `GCP_DIMSE_ZONE`. The identity comes from
`terraform/project/github_actions.tf`.

If both workflows show "disabled manually" under Actions, enable them. Then
retire the Cloud Build triggers so a merge does not deploy twice:

```bash
gcloud builds triggers list --project <GCP_PROJECT_ID> --format='value(name)'
gcloud builds triggers delete deploy-on-develop --project <GCP_PROJECT_ID>
gcloud builds triggers delete terraform-apply-on-develop --project <GCP_PROJECT_ID>
```

### AWS

The OIDC provider and both roles live in `terraform/aws/github_actions_oidc.tf`,
so the first apply is a targeted one from a laptop with your own credentials:

```bash
terraform -chdir=terraform/aws init
terraform -chdir=terraform/aws apply \
  -target=aws_iam_openid_connect_provider.github_actions \
  -target=aws_iam_role.github_actions_terraform \
  -target=aws_iam_role_policy_attachment.github_actions_terraform_admin \
  -target=aws_iam_role.github_actions_deploy \
  -target=aws_iam_role_policy_attachment.github_actions_deploy
terraform -chdir=terraform/aws output github_actions_terraform_role_arn github_actions_deploy_role_arn
```

Secrets: `AWS_TERRAFORM_ROLE_ARN`, `AWS_DEPLOY_ROLE_ARN`, `AWS_TERRAFORM_TFVARS`
(full `terraform/aws/terraform.tfvars`). Variables: `AWS_CI_ENABLED=true`,
`AWS_REGION`, `AWS_API_URL`, `AWS_ADMIN_URL`; optional `AWS_PROJECT_NAME`,
`AWS_DWV_URL`, `AWS_BUILD_SERVICES` (e.g. `api admin-dashboard` for the
minimal footprint), `AWS_DIMSE_INSTANCE_ID`.

The terraform role has `AdministratorAccess` because the module manages IAM,
KMS, VPC, RDS, ALB, Cognito and WAF; its trust policy only accepts tokens from
this repository's `develop` branch and pull requests.

### Azure

```bash
az login
./scripts/azure_bootstrap_github_oidc.sh          # prints the three IDs to store as secrets
```

Secrets: `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`,
`AZURE_TERRAFORM_TFVARS`. Variables: `AZURE_CI_ENABLED=true`,
`AZURE_RESOURCE_GROUP`, `AZURE_ACR_LOGIN_SERVER`, `AZURE_API_URL`,
`AZURE_ADMIN_URL`; optional `AZURE_APP_PREFIX` (default `aegis-prod`),
`AZURE_BUILD_SERVICES`, `AZURE_DIMSE_VM_NAME`.

The identity gets Contributor + User Access Administrator on the
subscription and the Application Administrator directory role (terraform
creates the Easy Auth app registration).

### Cloudflare Pages

Workers & Pages → connect `aegis-imaging/aegis`: production branch `develop`,
root directory `frontend/landing`, build `npm run build`, output `dist`,
environment variable `NODE_VERSION=20`, build watch path `frontend/landing/*`.
Custom domains `aegisimaging.ai` and `www.aegisimaging.ai`.

## Order of operations for a brand-new cloud

1. Bootstrap the identity (above) and set the secrets/variables.
2. Put the tfvars in the `*_TERRAFORM_TFVARS` secret and open a PR touching
   that cloud's `terraform/` directory — the plan runs on the PR.
3. Merge; approve the environment. Terraform creates registries, database,
   compute and load balancing. Services cannot start yet: no images.
4. Run the cloud's deploy workflow once by hand (Run workflow). From then on
   every merge builds, pushes and rolls automatically.
5. DNS records for the new hostnames (see `minimal-footprint.md` for the
   exact records per cloud).

## Reading a failure

- Terraform: the plan/apply step prints `[error]` lines with the provider's
  message. Re-run after fixing the tfvars secret or the code.
- GCP deploy: a Cloud Run service that fails to start prints its recent
  logs in the step; the previous revision keeps serving traffic.
- AWS deploy: `aws ecs wait services-stable` fails if the API or admin task
  keeps crashing; check the ECS service events and CloudWatch log group.
- Azure deploy: `az containerapp update` reports provisioning failures;
  check the revision's console logs in the portal.
