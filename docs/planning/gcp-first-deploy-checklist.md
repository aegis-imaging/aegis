# AEGIS GCP First Deploy Checklist

Created: 2026-02-20  
Updated: 2026-02-20

This is the fastest path to stand up the first GCP-backed AEGIS environment using the deployment automation now in-repo.

## One-Command Option

After both tfvars files are configured, you can run the full flow with:

```bash
make gcp-install-poc PROJECT_ID="<gcp-project-id>" REGION="us-central1" TAG="latest"
```

The detailed steps below are the same sequence, broken out for control and troubleshooting.

## 1) Create GCP project and attach billing

```bash
export PROJECT_ID="aegis-dev-$(date +%s)"
export REGION="us-central1"

gcloud auth login
gcloud auth application-default login
gcloud projects create "$PROJECT_ID" --name="AEGIS Dev"
gcloud beta billing accounts list
gcloud beta billing projects link "$PROJECT_ID" --billing-account="<BILLING_ACCOUNT_ID>"
gcloud config set project "$PROJECT_ID"
```

## 2) Run local preflight checks

```bash
make gcp-preflight
```

## 3) Configure Terraform variables

```bash
cp terraform/project/terraform.tfvars.example terraform/project/terraform.tfvars
cp terraform/infra/terraform.tfvars.example terraform/infra/terraform.tfvars
```

Set required values in:
- `terraform/project/terraform.tfvars`: `project_id`, `region`, `billing_account`
- `terraform/infra/terraform.tfvars`: domains, IAP OAuth values, image tags, and environment options

## 4) Bootstrap project-level services and security baseline

```bash
./scripts/gcp_bootstrap_project.sh --tfvars=terraform/project/terraform.tfvars --apply
```

## 5) Bootstrap Artifact Registry first

```bash
./scripts/gcp_apply_infra.sh --tfvars=terraform/infra/terraform.tfvars --target-artifact-repo
```

## 6) Build and push images

```bash
./scripts/gcp_build_push_images.sh --project-id="$PROJECT_ID" --region="$REGION" --tag="latest"
```

Equivalent Make target:

```bash
make gcp-build-images PROJECT_ID="$PROJECT_ID" REGION="$REGION" TAG=latest
```

## 7) Apply full infra

```bash
./scripts/gcp_apply_infra.sh --tfvars=terraform/infra/terraform.tfvars --apply
```

## 8) Point DNS at load balancer IP

```bash
terraform -chdir=terraform/infra output load_balancer_ip
```

Create DNS `A` records for:
- `api_domain`
- `admin_domain`

## 9) Verify deployment

```bash
curl -f https://<api_domain>/healthz
gcloud run services list --region "$REGION"
terraform -chdir=terraform/infra output smtp_egress_ip
```

Then:
- open `https://<admin_domain>` in incognito,
- confirm IAP login challenge,
- confirm unauthorized user access is denied.

## 10) Run cloud smoke suite

```bash
python3 scripts/cloud_smoke_test.py \
  --base-url "https://<api_domain>" \
  --project-slug default \
  --admin-header "X-Goog-Authenticated-User-Email: accounts.google.com:<you@example.com>"
```

## 11) Secret rotation drill (recommended before pilot)

```bash
export NEW_DB_PASSWORD='<new-strong-password>'
gcloud secrets versions add <db_password_secret_id> --data-file=- <<<"$NEW_DB_PASSWORD"
gcloud sql users set-password aegis-api --instance=aegis-dev-postgres --password="$NEW_DB_PASSWORD"
gcloud run services update aegis-api --region="$REGION" --update-env-vars=ROTATION_EPOCH=$(date +%s)
curl -f https://<api_domain>/healthz
```
