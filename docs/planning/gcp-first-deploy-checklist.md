# AEGIS GCP First Deploy Checklist

Created: 2026-02-20

This is a fast path to stand up the first GCP-backed AEGIS environment from a clean machine.

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

## 2) Bootstrap project services via Terraform

```bash
cp terraform/project/terraform.tfvars.example terraform/project/terraform.tfvars
# set: project_id, region, billing_account

terraform -chdir=terraform/project init
terraform -chdir=terraform/project plan
terraform -chdir=terraform/project apply
```

## 3) Create IAP OAuth credentials

In Google Cloud Console:
1. Configure OAuth consent screen.
2. Create OAuth client credentials (Web application).
3. Capture the client ID and client secret for `terraform/infra/terraform.tfvars`.

## 4) Prepare infra tfvars

```bash
cp terraform/infra/terraform.tfvars.example terraform/infra/terraform.tfvars
```

Fill in:
- `project_id`, `region`, `environment`
- `api_domain`, `admin_domain`
- `iap_oauth_client_id`, `iap_oauth_client_secret`, `iap_access_members`
- `db_password`, `db_password_secret_id`
- image URIs for API, admin dashboard, and all sidecars

## 5) Create Artifact Registry first

```bash
terraform -chdir=terraform/infra init
terraform -chdir=terraform/infra apply -target=google_artifact_registry_repository.services
```

## 6) Build and push container images

```bash
export REPO="${REGION}-docker.pkg.dev/${PROJECT_ID}/aegis-services"
gcloud auth configure-docker "${REGION}-docker.pkg.dev"

docker buildx build --platform linux/amd64 -t "${REPO}/api:latest" --push api
docker buildx build --platform linux/amd64 -t "${REPO}/defacing:latest" --push defacing
docker buildx build --platform linux/amd64 -t "${REPO}/phi-detection:latest" --push phi-detection
docker buildx build --platform linux/amd64 -t "${REPO}/qc-service:latest" --push qc-service
docker buildx build --platform linux/amd64 -t "${REPO}/bids-service:latest" --push bids-service
docker buildx build --platform linux/amd64 -t "${REPO}/classification-service:latest" --push classification-service
docker buildx build --platform linux/amd64 -t "${REPO}/protocol-service:latest" --push protocol-service
```

## 7) Admin dashboard image prerequisite

Current repo state includes no `frontend/admin-dashboard/Dockerfile`, but Terraform infra expects `admin_dashboard_image`.

Before full apply:
- add/build/push an admin dashboard container image, then
- set `admin_dashboard_image` in `terraform/infra/terraform.tfvars`.

## 8) Full infra plan/apply

```bash
terraform -chdir=terraform/infra plan
terraform -chdir=terraform/infra apply
```

## 9) Point DNS at load balancer IP

```bash
terraform -chdir=terraform/infra output load_balancer_ip
```

Create DNS `A` records for:
- `api_domain`
- `admin_domain`

## 10) Verify deployment

```bash
curl -f https://<api_domain>/healthz
gcloud run services list --region "$REGION"
terraform -chdir=terraform/infra output smtp_egress_ip
```

Then:
- open `https://<admin_domain>` in incognito,
- confirm IAP login challenge,
- confirm unauthorized user access is denied.

## 11) Secret rotation drill (recommended before pilot)

```bash
export NEW_DB_PASSWORD='<new-strong-password>'
gcloud secrets versions add <db_password_secret_id> --data-file=- <<<"$NEW_DB_PASSWORD"
gcloud sql users set-password aegis-api --instance=aegis-dev-postgres --password="$NEW_DB_PASSWORD"
gcloud run services update aegis-api --region="$REGION" --update-env-vars=ROTATION_EPOCH=$(date +%s)
curl -f https://<api_domain>/healthz
```
