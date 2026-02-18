# AEGIS — GCP Project Bootstrap
#
# This module creates and configures the GCP project:
# - Enable required APIs
# - Service accounts and IAM bindings
# - Cloud KMS key rings and keys (CMEK)
# - VPC Service Controls perimeter
#
# Prerequisites:
# - GCP organization or folder to host the project
# - Billing account linked
# - Terraform service account with project creator permissions

terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
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

provider "google" {
  project = var.project_id
  region  = var.region
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
  ])

  project = var.project_id
  service = each.value

  disable_dependent_services = false
  disable_on_destroy         = false
}

# TODO: KMS key ring for CMEK encryption
# TODO: Service accounts (api, defacing, cloud-build)
# TODO: IAM bindings (least privilege)
# TODO: VPC Service Controls perimeter
# TODO: Audit log configuration

output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}
