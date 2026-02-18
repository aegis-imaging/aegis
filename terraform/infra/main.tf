# AEGIS — Infrastructure
#
# This module provisions the core infrastructure:
# - VPC, subnets, Cloud NAT, Private Service Connect
# - Cloud Run services (API, admin dashboard, defacing)
# - Cloud Storage buckets (staging, archive)
# - Healthcare API dataset + DICOM stores (raw, clean)
# - Cloud SQL (PostgreSQL 15) for application data
# - BigQuery dataset for DICOM metadata analytics
# - Vertex AI endpoint configuration
# - Private Service Connect for on-prem SMTP relay
# - Cloud Armor security policies
# - Pub/Sub topics and subscriptions
# - Artifact Registry for Docker images
# - IAP configuration for admin dashboard
#
# Depends on: terraform/project/ having been applied first

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

provider "google" {
  project = var.project_id
  region  = var.region
}

# --- Healthcare API ---

resource "google_healthcare_dataset" "aegis" {
  name     = "aegis"
  location = var.region
}

resource "google_healthcare_dicom_store" "raw" {
  name    = "raw"
  dataset = google_healthcare_dataset.aegis.id

  notification_config {
    pubsub_topic = google_pubsub_topic.dicom_ingest.id
  }
}

resource "google_healthcare_dicom_store" "clean" {
  name    = "clean"
  dataset = google_healthcare_dataset.aegis.id
}

# --- Pub/Sub ---

resource "google_pubsub_topic" "dicom_ingest" {
  name = "dicom-ingest"
}

resource "google_pubsub_subscription" "dicom_ingest_sub" {
  name  = "dicom-ingest-sub"
  topic = google_pubsub_topic.dicom_ingest.name
}

# --- Cloud Storage ---

resource "google_storage_bucket" "staging" {
  name     = "${var.project_id}-staging"
  location = var.region

  uniform_bucket_level_access = true

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 7 # Auto-delete staging files after 7 days
    }
  }
}

# --- Cloud SQL (PostgreSQL) ---

resource "google_sql_database_instance" "aegis" {
  name             = "aegis-postgres"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier              = "db-f1-micro" # Small for dev; upgrade for prod
    availability_type = "ZONAL"       # REGIONAL for prod HA
    disk_size         = 10
    disk_type         = "PD_SSD"

    ip_configuration {
      ipv4_enabled    = false
      private_network = "projects/${var.project_id}/global/networks/default"
    }

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
    }
  }

  deletion_protection = true
}

resource "google_sql_database" "aegis" {
  name     = "aegis"
  instance = google_sql_database_instance.aegis.name
}

resource "google_sql_user" "api" {
  name     = "aegis-api"
  instance = google_sql_database_instance.aegis.name
  password = "CHANGE_ME" # Use Secret Manager in production
}

# --- BigQuery ---

resource "google_bigquery_dataset" "aegis" {
  dataset_id = "aegis"
  location   = var.region

  default_table_expiration_ms = null # No auto-expiration

  labels = {
    environment = "dev"
  }
}

# BigQuery table for Healthcare API DICOM metadata export
resource "google_bigquery_table" "dicom_metadata" {
  dataset_id = google_bigquery_dataset.aegis.dataset_id
  table_id   = "dicom_metadata"

  deletion_protection = false

  labels = {
    source = "healthcare-api"
  }
}

# BigQuery table for application audit trail
resource "google_bigquery_table" "audit_log" {
  dataset_id = google_bigquery_dataset.aegis.dataset_id
  table_id   = "audit_log"

  deletion_protection = false

  time_partitioning {
    type  = "DAY"
    field = "timestamp"
  }

  labels = {
    source = "aegis-api"
  }
}

# TODO: Artifact Registry repository
# TODO: Cloud Run service definitions
# TODO: Cloud Armor security policy
# TODO: VPC + subnets + Cloud NAT
# TODO: IAP for admin dashboard
# TODO: Private Service Connect endpoint for on-prem SMTP
# TODO: Vertex AI endpoint configuration
# TODO: Monitoring and alerting

output "dicom_store_raw" {
  value = google_healthcare_dicom_store.raw.self_link
}

output "dicom_store_clean" {
  value = google_healthcare_dicom_store.clean.self_link
}

output "staging_bucket" {
  value = google_storage_bucket.staging.name
}

output "postgres_connection" {
  value     = google_sql_database_instance.aegis.connection_name
  sensitive = true
}

output "bigquery_dataset" {
  value = google_bigquery_dataset.aegis.dataset_id
}
