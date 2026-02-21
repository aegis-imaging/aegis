# AEGIS - GCP Project Bootstrap
#
# This module creates and configures the GCP project baseline:
# - Enable required APIs
# - Service accounts and IAM bindings
# - Cloud KMS key ring and CMEK key
# - Optional VPC Service Controls perimeter
# - Project-wide audit log configuration

terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 6.0"
    }
  }
}

variable "project_id" {
  description = "GCP project ID for AEGIS"
  type        = string
}

variable "region" {
  description = "Primary GCP region"
  type        = string
  default     = "us-central1"
}

variable "billing_account" {
  description = "Billing account ID"
  type        = string
}

variable "kms_key_ring_name" {
  description = "KMS key ring name for AEGIS CMEK assets"
  type        = string
  default     = "aegis"
}

variable "kms_crypto_key_name" {
  description = "KMS crypto key name for AEGIS CMEK assets"
  type        = string
  default     = "aegis-data"
}

variable "kms_rotation_period" {
  description = "KMS crypto key rotation period"
  type        = string
  default     = "7776000s"
}

variable "enable_vpc_service_controls" {
  description = "Enable VPC Service Controls perimeter creation"
  type        = bool
  default     = false
}

variable "access_policy_id" {
  description = "Access Context Manager policy ID or full resource name (required if enable_vpc_service_controls=true)"
  type        = string
  default     = ""
}

variable "vpc_service_restricted_services" {
  description = "Restricted services for VPC Service Controls perimeter"
  type        = list(string)
  default = [
    "storage.googleapis.com",
    "secretmanager.googleapis.com",
    "bigquery.googleapis.com",
    "healthcare.googleapis.com",
    "artifactregistry.googleapis.com"
  ]
}

variable "audit_log_data_read_exempted_members" {
  description = "Optional exempted members for DATA_READ audit logs"
  type        = list(string)
  default     = []
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

data "google_project" "current" {
  project_id = var.project_id
}

locals {
  workload_service_accounts = {
    api = {
      account_id   = "aegis-api"
      display_name = "AEGIS API service account"
    }
    defacing = {
      account_id   = "aegis-defacing"
      display_name = "AEGIS defacing service account"
    }
    cloud_build = {
      account_id   = "aegis-cloud-build"
      display_name = "AEGIS Cloud Build service account"
    }
  }

  project_role_bindings = [
    { sa = "api", role = "roles/logging.logWriter" },
    { sa = "api", role = "roles/monitoring.metricWriter" },
    { sa = "api", role = "roles/secretmanager.secretAccessor" },
    { sa = "api", role = "roles/cloudsql.client" },
    { sa = "api", role = "roles/storage.objectAdmin" },
    { sa = "api", role = "roles/pubsub.publisher" },

    { sa = "defacing", role = "roles/logging.logWriter" },
    { sa = "defacing", role = "roles/monitoring.metricWriter" },
    { sa = "defacing", role = "roles/storage.objectViewer" },
    { sa = "defacing", role = "roles/storage.objectCreator" },

    { sa = "cloud_build", role = "roles/logging.logWriter" },
    { sa = "cloud_build", role = "roles/artifactregistry.writer" },
    { sa = "cloud_build", role = "roles/run.admin" },
    { sa = "cloud_build", role = "roles/iam.serviceAccountUser" }
  ]

  access_policy_resource_name = startswith(var.access_policy_id, "accessPolicies/") ? var.access_policy_id : "accessPolicies/${var.access_policy_id}"
  perimeter_short_name        = substr(replace(var.project_id, "-", "_"), 0, 28)
}

# APIs to enable
resource "google_project_service" "apis" {
  for_each = toset([
    "healthcare.googleapis.com",
    "run.googleapis.com",
    "storage.googleapis.com",
    "cloudkms.googleapis.com",
    "iap.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "compute.googleapis.com",
    "pubsub.googleapis.com",
    "cloudfunctions.googleapis.com",
    "secretmanager.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "sqladmin.googleapis.com",
    "bigquery.googleapis.com",
    "aiplatform.googleapis.com",
    "servicenetworking.googleapis.com",
    "vpcaccess.googleapis.com",
    "accesscontextmanager.googleapis.com"
  ])

  project = var.project_id
  service = each.value

  disable_dependent_services = false
  disable_on_destroy         = false
}

resource "google_kms_key_ring" "aegis" {
  name     = var.kms_key_ring_name
  location = var.region
  project  = var.project_id

  depends_on = [google_project_service.apis["cloudkms.googleapis.com"]]
}

resource "google_kms_crypto_key" "aegis_data" {
  name            = var.kms_crypto_key_name
  key_ring        = google_kms_key_ring.aegis.id
  rotation_period = var.kms_rotation_period

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_service_account" "workload" {
  for_each = local.workload_service_accounts

  account_id   = each.value.account_id
  display_name = each.value.display_name
  project      = var.project_id
}

resource "google_project_iam_member" "workload_roles" {
  for_each = {
    for binding in local.project_role_bindings :
    "${binding.sa}:${binding.role}" => binding
  }

  project = var.project_id
  role    = each.value.role
  member  = "serviceAccount:${google_service_account.workload[each.value.sa].email}"
}

resource "google_kms_crypto_key_iam_member" "workload_kms_access" {
  for_each = {
    api      = google_service_account.workload["api"].email
    defacing = google_service_account.workload["defacing"].email
  }

  crypto_key_id = google_kms_crypto_key.aegis_data.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${each.value}"
}

resource "google_access_context_manager_service_perimeter" "aegis" {
  provider = google-beta
  count    = var.enable_vpc_service_controls && trimspace(var.access_policy_id) != "" ? 1 : 0

  parent         = local.access_policy_resource_name
  name           = "${local.access_policy_resource_name}/servicePerimeters/${local.perimeter_short_name}"
  title          = "AEGIS perimeter ${var.project_id}"
  perimeter_type = "PERIMETER_TYPE_REGULAR"
  description    = "AEGIS VPC Service Controls perimeter for project ${var.project_id}"

  status {
    resources           = ["projects/${data.google_project.current.number}"]
    restricted_services = var.vpc_service_restricted_services
  }

  depends_on = [google_project_service.apis["accesscontextmanager.googleapis.com"]]
}

resource "google_project_iam_audit_config" "all_services" {
  project = var.project_id
  service = "allServices"

  audit_log_config {
    log_type = "ADMIN_READ"
  }

  audit_log_config {
    log_type = "DATA_READ"

    exempted_members = var.audit_log_data_read_exempted_members
  }

  audit_log_config {
    log_type = "DATA_WRITE"
  }
}

output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}

output "kms_key_ring_id" {
  value = google_kms_key_ring.aegis.id
}

output "kms_crypto_key_id" {
  value = google_kms_crypto_key.aegis_data.id
}

output "service_account_emails" {
  value = { for name, sa in google_service_account.workload : name => sa.email }
}

output "vpc_service_perimeter_name" {
  value = length(google_access_context_manager_service_perimeter.aegis) > 0 ? google_access_context_manager_service_perimeter.aegis[0].name : null
}
