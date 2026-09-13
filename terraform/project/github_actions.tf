# GitHub Actions Workload Identity Federation (WIF)
#
# Allows GitHub Actions workflows to authenticate to GCP using short-lived OIDC
# tokens instead of long-lived service account keys. This is the same pattern
# used by the Azure deployment (OIDC federated identity via azure/login@v2).
#
# After applying this module, set the following in GitHub repo secrets/variables:
#   Secrets:
#     GCP_WORKLOAD_IDENTITY_PROVIDER  — output: workload_identity_provider
#     GCP_SERVICE_ACCOUNT             — output: deploy_service_account (aegis-cloud-build@...)
#     GCP_TERRAFORM_TFVARS            — full contents of terraform/infra/terraform.tfvars
#   Variables:
#     GCP_PROJECT_ID    = <GCP_PROJECT_ID>
#     GCP_REGION        = us-central1
#     GCP_DWV_URL       = https://dwv-<CLOUD_RUN_HASH>-uc.a.run.app
#     GCP_API_URL       = https://api.aegisimaging.ai
#     GCP_ADMIN_URL     = https://app.aegisimaging.ai
#     GCP_DIMSE_INSTANCE = aegis-prod-dimse-receiver   (optional — skip to disable DIMSE deploy)
#     GCP_DIMSE_ZONE     = us-central1-a               (optional)

variable "github_org" {
  description = "GitHub organisation or username that owns the repo (e.g. aegis-imaging)"
  type        = string
  default     = "aegis-imaging"
}

variable "github_repo" {
  description = "GitHub repository name (e.g. aegis)"
  type        = string
  default     = "aegis"
}

# ── Workload Identity Pool ──────────────────────────────────────────────────

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-pool"
  display_name              = "GitHub Actions Pool"
  description               = "Identity pool for GitHub Actions OIDC federation"
  project                   = var.project_id

  lifecycle {
    # Pool deletion requires the pool to be disabled first and takes 30 days.
    # Prevent accidental destruction.
    prevent_destroy = true
  }
}

# ── OIDC Provider ───────────────────────────────────────────────────────────

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "GitHub Actions Provider"
  description                        = "GitHub OIDC token provider for aegis-imaging/aegis"
  project                            = var.project_id

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.repository" = "assertion.repository"
  }

  # Restrict to tokens from this specific repository only.
  attribute_condition = "attribute.repository == '${var.github_org}/${var.github_repo}'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# ── SA Impersonation binding ────────────────────────────────────────────────
#
# Reuses the existing Cloud Build service account — it already has all required
# IAM roles (Artifact Registry writer/admin, Cloud Run admin, Compute admin,
# Secret Manager accessor, Storage admin, etc.) granted by setup_cloudbuild.sh.
# No new role grants are needed.

locals {
  # The BYOSA created in main.tf (account_id = "aegis-cloud-build").
  # This SA already has all required IAM roles from project bootstrap.
  deploy_sa = "aegis-cloud-build@${var.project_id}.iam.gserviceaccount.com"
}

resource "google_service_account_iam_member" "github_wif_impersonate" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/${local.deploy_sa}"
  role               = "roles/iam.workloadIdentityUser"
  member = join("", [
    "principalSet://iam.googleapis.com/",
    google_iam_workload_identity_pool.github.name,
    "/attribute.repository/${var.github_org}/${var.github_repo}"
  ])
}

# ── Outputs (copy these into GitHub secrets/variables) ─────────────────────

output "workload_identity_provider" {
  description = "Value for the GCP_WORKLOAD_IDENTITY_PROVIDER GitHub secret"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "deploy_service_account" {
  description = "Value for the GCP_SERVICE_ACCOUNT GitHub secret"
  value       = local.deploy_sa
}
