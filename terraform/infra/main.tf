# AEGIS — GCP Infrastructure (Production Baseline)
#
# This module provisions:
# - Custom VPC, subnets, private-service networking, Cloud NAT
# - Cloud SQL PostgreSQL with private IP
# - Artifact Registry for AEGIS container images
# - Cloud Run services (API, admin dashboard, processing sidecars)
# - Global external HTTPS load balancer (serverless NEGs)
# - Cloud Armor policy attached to API backend
# - IAP enabled on admin dashboard backend
# - Healthcare API dataset + DICOM stores (raw, clean)
# - Cloud Storage buckets (staging + archive)
# - Pub/Sub ingestion topic/subscription
# - BigQuery analytics dataset/tables
# - Monitoring notification channel + baseline alert policies
#
# SMTP note:
# This module uses a documented equivalent to PSC SMTP by routing Cloud Run
# egress through Cloud NAT with a dedicated static IP. Allowlist that IP on
# your SMTP relay.

terraform {
  required_version = ">= 1.5"

  backend "gcs" {
    bucket = "aegis-prod-488120-tfstate"
    prefix = "aegis-infra"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.12"
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

variable "environment" {
  description = "Environment label used for naming/tagging (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "api_domain" {
  description = "FQDN routed to the public API backend (example: api.aegisimaging.ai)"
  type        = string
}

variable "admin_domain" {
  description = "FQDN routed to the admin dashboard backend (example: admin.aegisimaging.ai)"
  type        = string
}

variable "landing_domain" {
  description = "FQDN for the public marketing / landing page (example: aegisimaging.ai). Leave empty to skip landing page deployment."
  type        = string
  default     = ""
}

variable "export_portal_base_url" {
  description = "Public base URL of the export portal UI (example: https://export.aegisimaging.ai). When set, share email links point to the portal instead of the raw API endpoint. Leave empty to fall back to the API URL."
  type        = string
  default     = ""
}

variable "iap_oauth_client_id" {
  description = "OAuth2 client ID used by IAP on the admin backend service"
  type        = string
  sensitive   = true
}

variable "iap_oauth_client_secret" {
  description = "OAuth2 client secret used by IAP on the admin backend service"
  type        = string
  sensitive   = true
}

variable "iap_access_members" {
  description = "IAM members granted IAP access (roles/iap.httpsResourceAccessor) to the admin dashboard"
  type        = list(string)
  default     = []
}

variable "first_admin_email" {
  description = "Email address to seed as the first admin user on initial API startup (FIRST_ADMIN_EMAIL). Idempotent — ignored once any admin user exists."
  type        = string
  default     = ""
}

variable "enable_admin_iap" {
  description = "Enable IAP on the admin dashboard backend service"
  type        = bool
  default     = true
}

variable "deletion_protection" {
  description = "Enable deletion protection on stateful/critical resources"
  type        = bool
  default     = true
}

variable "artifact_repository_id" {
  description = "Artifact Registry Docker repository ID"
  type        = string
  default     = "aegis-services"
}

variable "api_image" {
  description = "Container image URI for the Go API service"
  type        = string
}

variable "admin_dashboard_image" {
  description = "Container image URI for the admin dashboard (React + nginx) service"
  type        = string
}

variable "defacing_image" {
  description = "Container image URI for the defacing sidecar"
  type        = string
}

variable "phi_detection_image" {
  description = "Container image URI for the PHI detection sidecar"
  type        = string
}

variable "qc_service_image" {
  description = "Container image URI for the QC sidecar"
  type        = string
}

variable "bids_service_image" {
  description = "Container image URI for the BIDS sidecar"
  type        = string
}

variable "classification_service_image" {
  description = "Container image URI for the classification sidecar"
  type        = string
}

variable "protocol_service_image" {
  description = "Container image URI for the protocol sidecar"
  type        = string
}

variable "landing_image" {
  description = "Container image URI for the landing page (React + nginx). Empty = landing Cloud Run service not deployed."
  type        = string
  default     = ""
}

variable "synth_service_image" {
  description = "Container image URI for the synthetic MRI sidecar (empty = service not deployed)"
  type        = string
  default     = ""
}

variable "sct_service_image" {
  description = "Container image URI for the SCT (Spinal Cord Toolbox) sidecar (empty = service not deployed)"
  type        = string
  default     = ""
}

variable "analytics_service_image" {
  description = "Container image URI for the analytics sidecar (empty = service not deployed)"
  type        = string
  default     = ""
}

variable "mcp_server_image" {
  description = "Container image URI for the MCP agent server (empty = not deployed)"
  type        = string
  default     = ""
}

variable "mcp_server_aegis_api_token" {
  description = "AEGIS API bearer token for the MCP server to call the Go API (stored in Secret Manager)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "synth_service_url" {
  description = "Direct HTTPS URL for the synthetic MRI service when managed outside Terraform (e.g. a manually-deployed Cloud Run service). Takes precedence over the synth_service_image-derived URL."
  type        = string
  default     = ""
}

variable "weasis_image" {
  description = "Container image URI for the DWV/WEASIS viewer (empty = disabled)"
  type        = string
  default     = ""
}

variable "api_cpu" {
  description = "CPU limit for Cloud Run API container"
  type        = string
  default     = "2000m"
}

variable "api_memory" {
  description = "Memory limit for Cloud Run API container"
  type        = string
  default     = "2Gi"
}

variable "api_min_instances" {
  description = "Minimum API Cloud Run instances"
  type        = number
  default     = 1
}

variable "api_max_instances" {
  description = "Maximum API Cloud Run instances"
  type        = number
  default     = 10
}

variable "sidecar_cpu" {
  description = "CPU limit for sidecar Cloud Run containers"
  type        = string
  default     = "1000m"
}

variable "sidecar_memory" {
  description = "Memory limit for sidecar Cloud Run containers"
  type        = string
  default     = "1Gi"
}

variable "sidecar_min_instances" {
  description = "Minimum sidecar Cloud Run instances (0 = scale-to-zero, 1 = always-warm). Set to 1 to eliminate cold-start latency; adds ~$38/month per sidecar."
  type        = number
  default     = 0
}

variable "sidecar_max_instances" {
  description = "Maximum sidecar Cloud Run instances"
  type        = number
  default     = 5
}

variable "vpc_cidr" {
  description = "CIDR range for primary application subnet"
  type        = string
  default     = "10.20.0.0/20"
}

variable "vpc_connector_cidr" {
  description = "CIDR range for Serverless VPC Access connector"
  type        = string
  default     = "10.8.0.0/28"
}

variable "db_private_peering_prefix_length" {
  description = "Prefix length for private service networking allocated range"
  type        = number
  default     = 16
}

variable "db_tier" {
  description = "Cloud SQL machine tier"
  type        = string
  default     = "db-custom-2-7680"
}

variable "db_availability_type" {
  description = "Cloud SQL availability type (ZONAL or REGIONAL)"
  type        = string
  default     = "ZONAL"
}

variable "db_disk_size_gb" {
  description = "Cloud SQL disk size in GB"
  type        = number
  default     = 50
}

variable "db_password" {
  description = "Optional explicit password for the aegis-api Cloud SQL user. Leave empty to auto-generate."
  type        = string
  default     = ""
  sensitive   = true

  validation {
    condition     = var.db_password == "" || length(var.db_password) >= 16
    error_message = "db_password must be empty (auto-generates a secure password) or at least 16 characters."
  }

  validation {
    condition     = !contains(["changeme", "aegis", "postgres", "password", "admin", "secret", "letmein", "root", "12345", "qwerty", "test"], lower(var.db_password))
    error_message = "db_password must not be a known-weak value. Leave it empty to auto-generate a secure password."
  }
}

variable "db_password_secret_id" {
  description = "Optional Secret Manager secret ID for API DB password (defaults to aegis-<env>-db-password)"
  type        = string
  default     = ""
}

variable "generate_db_password" {
  description = "Generate a random DB password when db_password is empty"
  type        = bool
  default     = true
}

variable "generated_db_password_length" {
  description = "Length for generated DB password"
  type        = number
  default     = 32

  validation {
    condition     = var.generated_db_password_length >= 16
    error_message = "generated_db_password_length must be at least 16."
  }
}

variable "cloud_armor_allowed_ip_ranges" {
  description = "Allowed client IP ranges for API ingress (use [\"*\"] for open access)"
  type        = list(string)
  default     = ["*"]
}

variable "alert_email" {
  description = "Optional alert email recipient; leave empty to skip notification channel creation"
  type        = string
  default     = ""
}

variable "enable_monitoring_alerts" {
  description = "Enable baseline Cloud Monitoring alert policies"
  type        = bool
  default     = true
}

variable "storage_mode" {
  description = "Storage mode for API runtime (local, gcs, s3)"
  type        = string
  default     = "gcs"
}

variable "smtp_relay_host" {
  description = "SMTP relay host for outbound email (optional)"
  type        = string
  default     = ""
}

variable "smtp_relay_port" {
  description = "SMTP relay port"
  type        = number
  default     = 587
}

variable "smtp_from" {
  description = "SMTP FROM address used by AEGIS"
  type        = string
  default     = "noreply@aegisimaging.ai"
}

variable "contact_email" {
  description = "Recipient address for contact form submissions (CONTACT_EMAIL). Defaults to contact@aegisimaging.ai."
  type        = string
  default     = "contact@aegisimaging.ai"
}

variable "gate_enabled" {
  description = "Enable server-side invite gate on landing page (GATE_ENABLED env var)"
  type        = bool
  default     = false
}

variable "gate_secret" {
  description = "HMAC secret for landing page invite gate tokens (GATE_SECRET env var)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "allowed_origins" {
  description = "Optional CORS origins override. If empty, defaults to API + admin domains."
  type        = list(string)
  default     = []
}

# ── DIMSE receiver (Compute Engine) ──────────────────────────────────────────
# Leave dimse_receiver_image empty (the default) to skip all DIMSE resources.

variable "dimse_receiver_image" {
  description = "Full Artifact Registry image URI for the dimse-receiver container. Empty string disables all DIMSE Compute Engine resources."
  type        = string
  default     = ""
}

variable "dimse_receiver_machine_type" {
  description = "GCE machine type for the DIMSE receiver VM."
  type        = string
  default     = "e2-small"
}

variable "dimse_receiver_zone" {
  description = "Zone for the DIMSE receiver VM. Defaults to <region>-a when empty."
  type        = string
  default     = ""
}

variable "dimse_api_url" {
  description = "Base URL of the AEGIS API that the DIMSE receiver calls for ingest (e.g. https://api.aegisimaging.ai)."
  type        = string
  default     = ""
}

variable "dimse_project_slug" {
  description = "Project slug passed to POST /api/ingest for studies received via DIMSE."
  type        = string
  default     = "default"
}

variable "dimse_source_ranges" {
  description = "CIDR ranges allowed to reach TCP 11112 (DICOM C-STORE). Defaults to open (0.0.0.0/0) — restrict to PACS IP ranges in production."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Resolve project metadata (number required for IAP service agent email).
data "google_project" "this" {
  project_id = var.project_id
}

locals {
  name_prefix                    = "aegis-${var.environment}"
  resolved_db_password_secret_id = var.db_password_secret_id != "" ? var.db_password_secret_id : "${local.name_prefix}-db-password"
  resolved_db_password           = var.db_password != "" ? var.db_password : try(random_password.db_password[0].result, "")

  sidecar_services = merge(
    {
      defacing               = var.defacing_image
      phi-detection          = var.phi_detection_image
      qc-service             = var.qc_service_image
      bids-service           = var.bids_service_image
      classification-service = var.classification_service_image
      protocol-service       = var.protocol_service_image
    },
    # synth-service: optional sidecar for synthetic brain MRI generation.
    # Omit from the map when the image is not provided so the for_each loop
    # does not attempt to create a Cloud Run service with an empty image URI.
    var.synth_service_image != "" ? { synth-service = var.synth_service_image } : {},
    var.sct_service_image != "" ? { sct-service = var.sct_service_image } : {},
    var.analytics_service_image != "" ? { analytics-service = var.analytics_service_image } : {}
  )

  lb_domains = distinct(compact([
    var.api_domain,
    var.admin_domain,
    var.landing_domain,
    var.landing_domain != "" ? "www.${var.landing_domain}" : "",
  ]))

  resolved_allowed_origins = length(var.allowed_origins) > 0 ? var.allowed_origins : [
    "https://${var.api_domain}",
    "https://${var.admin_domain}",
  ]
}

check "db_password_source" {
  assert {
    condition     = var.db_password != "" || var.generate_db_password
    error_message = "Set db_password or keep generate_db_password=true."
  }
}

# --- Networking ---

resource "google_compute_network" "aegis" {
  name                    = "${local.name_prefix}-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "app" {
  name          = "${local.name_prefix}-app-subnet"
  region        = var.region
  network       = google_compute_network.aegis.id
  ip_cidr_range = var.vpc_cidr
}

resource "google_compute_global_address" "private_service_range" {
  name          = "${local.name_prefix}-private-services"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = var.db_private_peering_prefix_length
  network       = google_compute_network.aegis.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.aegis.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_service_range.name]
}

resource "google_compute_network_peering_routes_config" "private_vpc_routes" {
  peering              = google_service_networking_connection.private_vpc_connection.peering
  network              = google_compute_network.aegis.name
  import_custom_routes = true
  export_custom_routes = true
}

resource "google_compute_router" "nat" {
  name    = "${local.name_prefix}-router"
  region  = var.region
  network = google_compute_network.aegis.id
}

resource "google_compute_address" "smtp_egress_ip" {
  name   = "${local.name_prefix}-smtp-egress-ip"
  region = var.region
}

resource "google_compute_router_nat" "nat" {
  name                               = "${local.name_prefix}-nat"
  region                             = var.region
  router                             = google_compute_router.nat.name
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = [google_compute_address.smtp_egress_ip.self_link]

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

resource "google_vpc_access_connector" "cloud_run" {
  name          = "${local.name_prefix}-connector"
  region        = var.region
  ip_cidr_range = var.vpc_connector_cidr
  network       = google_compute_network.aegis.name
  min_instances = 2
  max_instances = 3
}

# --- Core data services ---

resource "google_healthcare_dataset" "aegis" {
  name     = "aegis"
  location = var.region
}

resource "google_pubsub_topic" "dicom_ingest" {
  name = "${local.name_prefix}-dicom-ingest"
}

resource "google_pubsub_subscription" "dicom_ingest_sub" {
  name  = "${local.name_prefix}-dicom-ingest-sub"
  topic = google_pubsub_topic.dicom_ingest.name
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

resource "google_storage_bucket" "staging" {
  name     = "${var.project_id}-staging"
  location = var.region

  uniform_bucket_level_access = true

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 7
    }
  }
}

resource "google_storage_bucket" "archive" {
  name     = "${var.project_id}-archive"
  location = var.region

  uniform_bucket_level_access = true

  lifecycle_rule {
    action {
      type          = "SetStorageClass"
      storage_class = "COLDLINE"
    }
    condition {
      age = 30
    }
  }
}

resource "google_sql_database_instance" "aegis" {
  name             = "${local.name_prefix}-postgres"
  database_version = "POSTGRES_15"
  region           = var.region

  depends_on = [google_service_networking_connection.private_vpc_connection]

  settings {
    tier              = var.db_tier
    availability_type = var.db_availability_type
    disk_size         = var.db_disk_size_gb
    disk_type         = "PD_SSD"

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = google_compute_network.aegis.id
      enable_private_path_for_google_cloud_services = true
    }

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
    }
  }

  deletion_protection = var.deletion_protection
}

resource "google_sql_database" "aegis" {
  name     = "aegis"
  instance = google_sql_database_instance.aegis.name
}

resource "google_sql_user" "api" {
  name     = "aegis-api"
  instance = google_sql_database_instance.aegis.name
  password = local.resolved_db_password
}

resource "random_password" "db_password" {
  count            = var.db_password == "" && var.generate_db_password ? 1 : 0
  length           = var.generated_db_password_length
  special          = true
  override_special = "_%@"
}

resource "google_secret_manager_secret" "db_password" {
  secret_id = local.resolved_db_password_secret_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = local.resolved_db_password
}

resource "google_bigquery_dataset" "aegis" {
  dataset_id = "aegis"
  location   = var.region

  labels = {
    environment = var.environment
  }
}

resource "google_bigquery_table" "dicom_metadata" {
  dataset_id = google_bigquery_dataset.aegis.dataset_id
  table_id   = "dicom_metadata"

  deletion_protection = false

  labels = {
    source = "healthcare-api"
  }
}

resource "google_bigquery_table" "audit_log" {
  dataset_id = google_bigquery_dataset.aegis.dataset_id
  table_id   = "audit_log"

  deletion_protection = false

  time_partitioning {
    type  = "DAY"
    field = "timestamp"
  }

  schema = jsonencode([
    { name = "timestamp", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "event", type = "STRING", mode = "NULLABLE" },
    { name = "user", type = "STRING", mode = "NULLABLE" },
    { name = "resource", type = "STRING", mode = "NULLABLE" },
    { name = "payload", type = "JSON", mode = "NULLABLE" },
  ])

  labels = {
    source = "aegis-api"
  }
}

# --- Container registry ---

resource "google_artifact_registry_repository" "services" {
  location      = var.region
  repository_id = var.artifact_repository_id
  description   = "AEGIS service containers"
  format        = "DOCKER"
}

# --- Runtime identities ---

resource "google_service_account" "api" {
  account_id   = "${replace(local.name_prefix, "-", "")}api"
  display_name = "AEGIS API runtime service account"
}

resource "google_service_account" "sidecars" {
  account_id   = "${replace(local.name_prefix, "-", "")}sidecars"
  display_name = "AEGIS sidecar runtime service account"
}

resource "google_service_account" "admin" {
  account_id   = "${replace(local.name_prefix, "-", "")}admin"
  display_name = "AEGIS admin dashboard runtime service account"
}

resource "google_storage_bucket_iam_member" "api_staging_rw" {
  bucket = google_storage_bucket.staging.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.api.email}"
}

# Sidecars need read/write access to the staging bucket for GCS FUSE mounts
# (defacing reads raw files and writes clean output, QC/BIDS/phi-detection read raw files).
resource "google_storage_bucket_iam_member" "sidecars_staging_rw" {
  bucket = google_storage_bucket.staging.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.sidecars.email}"
}

resource "google_storage_bucket_iam_member" "api_archive_rw" {
  bucket = google_storage_bucket.archive.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.api.email}"
}

resource "google_secret_manager_secret_iam_member" "api_db_password_access" {
  secret_id = google_secret_manager_secret.db_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api.email}"
}

# Allow the API service account to sign blobs as itself (for GCS signed URLs).
resource "google_service_account_iam_member" "api_self_token_creator" {
  service_account_id = google_service_account.api.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${google_service_account.api.email}"
}

# --- Cloud Run sidecars ---

resource "google_cloud_run_v2_service" "sidecars" {
  for_each = local.sidecar_services

  name     = each.key
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  deletion_protection = var.deletion_protection

  template {
    service_account = google_service_account.sidecars.email

    annotations = {
      # gen2 execution environment is required for GCS FUSE CSI volume mounts.
      "run.googleapis.com/execution-environment" = "gen2"
    }

    scaling {
      min_instance_count = var.sidecar_min_instances
      max_instance_count = var.sidecar_max_instances
    }

    containers {
      image = each.value

      resources {
        limits = {
          cpu    = var.sidecar_cpu
          memory = var.sidecar_memory
        }
      }

      # Mount the shared DICOM GCS bucket so sidecars (defacing, QC, BIDS, etc.)
      # can read raw files and write processed output via the same filesystem path
      # as the Go API.
      volume_mounts {
        name       = "gcs-dicom"
        mount_path = "/app/data"
      }

      liveness_probe {
        failure_threshold     = 5
        initial_delay_seconds = 15
        timeout_seconds       = 5
        period_seconds        = 15

        http_get {
          path = "/health"
        }
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.cloud_run.id
      egress    = "ALL_TRAFFIC"
    }

    volumes {
      name = "gcs-dicom"
      gcs {
        bucket    = google_storage_bucket.staging.name
        read_only = false
      }
    }
  }
}

resource "google_cloud_run_service_iam_member" "sidecar_invoker" {
  for_each = local.sidecar_services

  location = var.region
  service  = google_cloud_run_v2_service.sidecars[each.key].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# --- Cloud Run DWV (WEASIS) Viewer ---

resource "google_cloud_run_v2_service" "weasis" {
  count    = var.weasis_image != "" ? 1 : 0
  name     = "weasis"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  deletion_protection = var.deletion_protection

  template {
    service_account = google_service_account.sidecars.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = var.weasis_image

      env {
        name  = "API_URL"
        value = "https://${var.api_domain}"
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      liveness_probe {
        failure_threshold     = 3
        initial_delay_seconds = 10
        timeout_seconds       = 5
        period_seconds        = 30

        http_get {
          path = "/health"
        }
      }
    }
  }
}

resource "google_cloud_run_service_iam_member" "weasis_invoker" {
  count    = var.weasis_image != "" ? 1 : 0
  location = var.region
  service  = google_cloud_run_v2_service.weasis[0].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# --- Cloud Run API ---

resource "google_cloud_run_v2_service" "api" {
  name     = "aegis-api"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  deletion_protection = var.deletion_protection

  depends_on = [
    google_sql_user.api,
    google_secret_manager_secret_version.db_password,
    google_cloud_run_v2_service.sidecars,
  ]

  template {
    service_account = google_service_account.api.email

    annotations = {
      # gen2 execution environment is required for GCS FUSE CSI volume mounts.
      "run.googleapis.com/execution-environment" = "gen2"
    }

    scaling {
      min_instance_count = var.api_min_instances
      max_instance_count = var.api_max_instances
    }

    containers {
      image = var.api_image

      resources {
        limits = {
          cpu    = var.api_cpu
          memory = var.api_memory
        }
      }

      env {
        name  = "DB_HOST"
        value = google_sql_database_instance.aegis.private_ip_address
      }
      env {
        name  = "DB_PORT"
        value = "5432"
      }
      env {
        name  = "DB_NAME"
        value = google_sql_database.aegis.name
      }
      env {
        name  = "DB_USER"
        value = google_sql_user.api.name
      }
      env {
        name = "DB_PASSWORD"

        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.db_password.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "STORAGE_MODE"
        value = var.storage_mode
      }
      env {
        name  = "LOCAL_STORAGE_DIR"
        value = "/app/data"
      }
      env {
        name  = "GCP_PROJECT"
        value = var.project_id
      }
      env {
        name  = "GCS_BUCKET"
        value = google_storage_bucket.staging.name
      }
      env {
        name  = "GCS_SIGNING_EMAIL"
        value = google_service_account.api.email
      }
      env {
        name  = "DICOM_DATASET"
        value = google_healthcare_dataset.aegis.name
      }
      env {
        name  = "DICOM_STORE_RAW"
        value = google_healthcare_dicom_store.raw.name
      }
      env {
        name  = "DICOM_STORE_CLEAN"
        value = google_healthcare_dicom_store.clean.name
      }
      env {
        name  = "API_BASE_URL"
        value = "https://${var.api_domain}"
      }
      env {
        name  = "EXPORT_PORTAL_BASE_URL"
        value = var.export_portal_base_url
      }
      env {
        name  = "APP_TIMEZONE"
        value = "UTC"
      }
      env {
        name  = "ALLOWED_ORIGINS"
        value = join(",", local.resolved_allowed_origins)
      }
      env {
        name  = "AUTH_ENABLED"
        value = "true"
      }
      env {
        name  = "AUTH_PROVIDER"
        value = "iap"
      }
      env {
        name  = "FIRST_ADMIN_EMAIL"
        value = var.first_admin_email
      }
      env {
        name  = "SMTP_HOST"
        value = var.smtp_relay_host
      }
      env {
        name  = "SMTP_PORT"
        value = tostring(var.smtp_relay_port)
      }
      env {
        name  = "SMTP_FROM"
        value = var.smtp_from
      }
      env {
        name  = "CONTACT_EMAIL"
        value = var.contact_email
      }
      env {
        name  = "DEFACING_SERVICE_URL"
        value = google_cloud_run_v2_service.sidecars["defacing"].uri
      }
      env {
        name  = "PHI_DETECTION_SERVICE_URL"
        value = google_cloud_run_v2_service.sidecars["phi-detection"].uri
      }
      env {
        name  = "QC_SERVICE_URL"
        value = google_cloud_run_v2_service.sidecars["qc-service"].uri
      }
      env {
        name  = "BIDS_SERVICE_URL"
        value = google_cloud_run_v2_service.sidecars["bids-service"].uri
      }
      env {
        name  = "CLASSIFICATION_SERVICE_URL"
        value = google_cloud_run_v2_service.sidecars["classification-service"].uri
      }
      env {
        name  = "PROTOCOL_SERVICE_URL"
        value = google_cloud_run_v2_service.sidecars["protocol-service"].uri
      }
      dynamic "env" {
        # Prefer an explicit URL override; fall back to the Terraform-managed
        # synth-service Cloud Run URI when synth_service_image is set.
        for_each = var.synth_service_url != "" ? [var.synth_service_url] : (
          var.synth_service_image != "" ? [google_cloud_run_v2_service.sidecars["synth-service"].uri] : []
        )
        content {
          name  = "SYNTH_SERVICE_URL"
          value = env.value
        }
      }
      dynamic "env" {
        for_each = var.sct_service_image != "" ? [google_cloud_run_v2_service.sidecars["sct-service"].uri] : []
        content {
          name  = "SCT_SERVICE_URL"
          value = env.value
        }
      }
      dynamic "env" {
        for_each = var.analytics_service_image != "" ? [google_cloud_run_v2_service.sidecars["analytics-service"].uri] : []
        content {
          name  = "ANALYTICS_SERVICE_URL"
          value = env.value
        }
      }

      liveness_probe {
        failure_threshold     = 5
        initial_delay_seconds = 20
        timeout_seconds       = 5
        period_seconds        = 15

        http_get {
          path = "/healthz"
        }
      }

      # Mount the shared DICOM GCS bucket so the API can resolve filesystem paths
      # for defacing, QC, BIDS, and DICOMweb serving (LOCAL_STORAGE_DIR=/app/data).
      volume_mounts {
        name       = "gcs-dicom"
        mount_path = "/app/data"
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.cloud_run.id
      egress    = "ALL_TRAFFIC"
    }

    volumes {
      name = "gcs-dicom"
      gcs {
        bucket    = google_storage_bucket.staging.name
        read_only = false
      }
    }
  }
}

resource "google_cloud_run_service_iam_member" "api_invoker" {
  location = var.region
  service  = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# --- Cloud Run admin dashboard ---

resource "google_cloud_run_v2_service" "admin_dashboard" {
  name     = "aegis-admin-dashboard"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  deletion_protection = var.deletion_protection

  template {
    service_account = google_service_account.admin.email

    scaling {
      min_instance_count = 1
      max_instance_count = 5
    }

    containers {
      image = var.admin_dashboard_image

      resources {
        limits = {
          cpu    = "1000m"
          memory = "1Gi"
        }
      }

      liveness_probe {
        failure_threshold     = 5
        initial_delay_seconds = 20
        timeout_seconds       = 5
        period_seconds        = 15

        http_get {
          path = "/"
        }
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.cloud_run.id
      egress    = "PRIVATE_RANGES_ONLY"
    }
  }
}

# --- Landing page Cloud Run service ---

resource "google_cloud_run_v2_service" "landing" {
  count    = var.landing_image != "" ? 1 : 0
  name     = "${local.name_prefix}-landing"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  # Landing is a stateless nginx container — no data to protect, safe to replace.
  deletion_protection = false

  template {
    scaling {
      min_instance_count = 0
      max_instance_count = 5
    }

    containers {
      image = var.landing_image

      resources {
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
        cpu_idle = true
      }

      env {
        name  = "API_BASE_URL"
        value = "https://${var.api_domain}"
      }

      env {
        name  = "GATE_ENABLED"
        value = var.gate_enabled ? "true" : "false"
      }

      env {
        name  = "GATE_SECRET"
        value = var.gate_secret
      }

      liveness_probe {
        failure_threshold     = 3
        initial_delay_seconds = 5
        timeout_seconds       = 3
        period_seconds        = 15

        http_get {
          path = "/healthz"
        }
      }
    }
  }
}

# Landing page is public — no IAP, no auth required.
resource "google_cloud_run_service_iam_member" "landing_invoker" {
  count    = var.landing_image != "" ? 1 : 0
  location = var.region
  service  = google_cloud_run_v2_service.landing[0].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# When IAP is enabled the IAP service agent (iap_invoker_admin) is the
# only identity that needs run.invoker on the admin Cloud Run service.
# Removing allUsers provides defense-in-depth: even if the LB IAP config
# is misconfigured, the Cloud Run service itself requires the IAP SA.
# When IAP is disabled, allUsers is still required so the LB can forward.
resource "google_cloud_run_service_iam_member" "admin_invoker" {
  count    = var.enable_admin_iap ? 0 : 1
  location = var.region
  service  = google_cloud_run_v2_service.admin_dashboard.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Grant the IAP service agent permission to invoke the admin Cloud Run service.
# The IAP service agent must be provisioned before this binding takes effect:
#   gcloud beta services identity create --service=iap.googleapis.com --project=PROJECT_ID
# This is a one-time bootstrap step per project (safe to run multiple times).
resource "google_cloud_run_service_iam_member" "iap_invoker_admin" {
  count    = var.enable_admin_iap ? 1 : 0
  location = var.region
  service  = google_cloud_run_v2_service.admin_dashboard.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-iap.iam.gserviceaccount.com"
}


# --- MCP Agent Server ---

resource "google_service_account" "mcp_server" {
  count        = var.mcp_server_image != "" ? 1 : 0
  account_id   = "${replace(local.name_prefix, "-", "")}mcp"
  display_name = "AEGIS MCP agent server service account"
}

# Grant Vertex AI user role so the MCP server can call Gemini via Workload Identity.
resource "google_project_iam_member" "mcp_server_vertex_ai" {
  count   = var.mcp_server_image != "" ? 1 : 0
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.mcp_server[0].email}"
}

resource "google_secret_manager_secret" "mcp_aegis_api_token" {
  count     = var.mcp_server_image != "" ? 1 : 0
  secret_id = "${local.name_prefix}-mcp-aegis-api-token"

  replication {
    auto {}
  }
}

# Populate the secret only when a token value is provided.
resource "google_secret_manager_secret_version" "mcp_aegis_api_token" {
  count       = var.mcp_server_image != "" && var.mcp_server_aegis_api_token != "" ? 1 : 0
  secret      = google_secret_manager_secret.mcp_aegis_api_token[0].id
  secret_data = var.mcp_server_aegis_api_token
}

resource "google_secret_manager_secret_iam_member" "mcp_server_token_access" {
  count     = var.mcp_server_image != "" ? 1 : 0
  secret_id = google_secret_manager_secret.mcp_aegis_api_token[0].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.mcp_server[0].email}"
}

resource "google_cloud_run_v2_service" "mcp_server" {
  count    = var.mcp_server_image != "" ? 1 : 0
  name     = "aegis-mcp-server"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  deletion_protection = false

  template {
    service_account = google_service_account.mcp_server[0].email

    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }

    containers {
      image = var.mcp_server_image

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      env {
        name  = "AEGIS_API_BASE_URL"
        value = "https://${var.api_domain}"
      }
      env {
        name = "AEGIS_API_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.mcp_aegis_api_token[0].secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "MCP_AGENT_HTTP_PORT"
        value = "8080"
      }
      env {
        name  = "MCP_AGENT_LLM_USE_GCP_AUTH"
        value = "true"
      }
      env {
        name  = "MCP_AGENT_LLM_GCP_PROJECT"
        value = var.project_id
      }
      env {
        name  = "MCP_AGENT_LLM_MODEL"
        value = "google/gemini-2.5-flash"
      }
      env {
        name  = "MCP_AGENT_ALLOWED_ORIGIN"
        value = "https://${var.admin_domain}"
      }
      env {
        name  = "MCP_AGENT_REQUIRE_AUTH"
        value = "false"
      }

      liveness_probe {
        failure_threshold     = 3
        initial_delay_seconds = 10
        timeout_seconds       = 5
        period_seconds        = 30

        http_get {
          path = "/healthz"
        }
      }
    }
    # No VPC connector — MCP server only calls external HTTPS endpoints
    # (Vertex AI and the AEGIS API load balancer). No private network needed.
  }

  depends_on = [google_secret_manager_secret_version.mcp_aegis_api_token]
}

# MCP server is proxied through the admin dashboard nginx — allUsers invoker
# allows nginx to call it without credentials.
resource "google_cloud_run_service_iam_member" "mcp_server_invoker" {
  count    = var.mcp_server_image != "" ? 1 : 0
  location = var.region
  service  = google_cloud_run_v2_service.mcp_server[0].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# --- Cloud Armor ---

resource "google_compute_security_policy" "api" {
  name        = "${local.name_prefix}-api-armor"
  description = "Baseline Cloud Armor policy for AEGIS API"

  rule {
    action      = "allow"
    priority    = "1000"
    description = "Allow configured source IP ranges"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = var.cloud_armor_allowed_ip_ranges
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = "2147483647"
    description = "Default deny"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
  }
}

# --- HTTPS load balancer + IAP ---

resource "google_compute_global_address" "lb_ip" {
  name = "${local.name_prefix}-lb-ip"
}

resource "google_compute_managed_ssl_certificate" "lb_cert" {
  name = "${local.name_prefix}-lb-cert-v3"
  managed {
    domains = local.lb_domains
  }
  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_region_network_endpoint_group" "landing_neg" {
  count                 = var.landing_image != "" ? 1 : 0
  name                  = "${local.name_prefix}-landing-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.landing[0].name
  }
}

resource "google_compute_backend_service" "landing" {
  count                 = var.landing_image != "" ? 1 : 0
  name                  = "${local.name_prefix}-landing-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  protocol              = "HTTP"

  log_config {
    enable      = true
    sample_rate = 0.1
  }

  backend {
    group = google_compute_region_network_endpoint_group.landing_neg[0].id
  }
}

resource "google_compute_region_network_endpoint_group" "api_neg" {
  name                  = "${local.name_prefix}-api-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.api.name
  }
}

resource "google_compute_region_network_endpoint_group" "admin_neg" {
  name                  = "${local.name_prefix}-admin-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.admin_dashboard.name
  }
}

resource "google_compute_backend_service" "api" {
  name                  = "${local.name_prefix}-api-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  protocol              = "HTTP"
  security_policy       = google_compute_security_policy.api.id

  log_config {
    enable      = true
    sample_rate = 1.0
  }

  backend {
    group = google_compute_region_network_endpoint_group.api_neg.id
  }
}

resource "google_compute_backend_service" "admin" {
  name                  = "${local.name_prefix}-admin-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  protocol              = "HTTP"

  dynamic "iap" {
    for_each = var.enable_admin_iap ? [1] : []
    content {
      enabled              = true
      oauth2_client_id     = var.iap_oauth_client_id
      oauth2_client_secret = var.iap_oauth_client_secret
    }
  }

  log_config {
    enable      = true
    sample_rate = 1.0
  }

  backend {
    group = google_compute_region_network_endpoint_group.admin_neg.id
  }
}

resource "google_compute_url_map" "https" {
  name            = "${local.name_prefix}-https-map"
  default_service = var.landing_image != "" ? google_compute_backend_service.landing[0].id : google_compute_backend_service.api.id

  dynamic "host_rule" {
    for_each = var.landing_domain != "" ? [1] : []
    content {
      hosts        = [var.landing_domain]
      path_matcher = "landing"
    }
  }

  host_rule {
    hosts        = [var.api_domain]
    path_matcher = "api"
  }

  host_rule {
    hosts        = [var.admin_domain]
    path_matcher = "admin"
  }

  dynamic "path_matcher" {
    for_each = var.landing_image != "" ? [1] : []
    content {
      name            = "landing"
      default_service = google_compute_backend_service.landing[0].id
    }
  }

  path_matcher {
    name            = "api"
    default_service = google_compute_backend_service.api.id
  }

  path_matcher {
    name            = "admin"
    default_service = google_compute_backend_service.admin.id
    # No path rules needed — nginx proxies /api/* to the API backend internally.
  }

}

resource "google_compute_target_https_proxy" "https" {
  name    = "${local.name_prefix}-https-proxy"
  url_map = google_compute_url_map.https.id
  ssl_certificates = [
    google_compute_managed_ssl_certificate.lb_cert.name
  ]
}

resource "google_compute_global_forwarding_rule" "https" {
  name                  = "${local.name_prefix}-https-fr"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  target                = google_compute_target_https_proxy.https.id
  ip_address            = google_compute_global_address.lb_ip.id
  port_range            = "443"
}

resource "google_compute_url_map" "http_redirect" {
  name = "${local.name_prefix}-http-redirect"
  default_url_redirect {
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    https_redirect         = true
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "http_redirect" {
  name    = "${local.name_prefix}-http-proxy"
  url_map = google_compute_url_map.http_redirect.id
}

resource "google_compute_global_forwarding_rule" "http" {
  name                  = "${local.name_prefix}-http-fr"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  target                = google_compute_target_http_proxy.http_redirect.id
  ip_address            = google_compute_global_address.lb_ip.id
  port_range            = "80"
}

resource "google_iap_web_backend_service_iam_binding" "admin_access" {
  count = var.enable_admin_iap && length(var.iap_access_members) > 0 ? 1 : 0

  project             = var.project_id
  web_backend_service = google_compute_backend_service.admin.name
  role                = "roles/iap.httpsResourceAccessor"
  members             = var.iap_access_members
}

# --- Monitoring baseline ---

resource "google_monitoring_notification_channel" "email" {
  count = var.alert_email == "" ? 0 : 1

  display_name = "${local.name_prefix}-ops-email"
  type         = "email"
  labels = {
    email_address = var.alert_email
  }
}

locals {
  notification_channels = var.alert_email == "" ? [] : [google_monitoring_notification_channel.email[0].name]
}

resource "google_monitoring_alert_policy" "api_5xx_rate" {
  display_name = "AEGIS API 5xx rate high (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Cloud Run API 5xx request count"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND resource.label.service_name = \"${google_cloud_run_v2_service.api.name}\" AND metric.type = \"run.googleapis.com/request_count\" AND metric.label.response_code_class = \"5xx\""
      comparison      = "COMPARISON_GT"
      threshold_value = 10
      duration        = "300s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "API is returning elevated 5xx responses. Check Cloud Run logs, DB connectivity, and sidecar health."
  }

  user_labels = {
    service  = "api"
    severity = "warning"
  }
}

resource "google_monitoring_alert_policy" "cloudsql_cpu" {
  display_name = "AEGIS Cloud SQL CPU high (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Cloud SQL CPU > 80% for 10m"
    condition_threshold {
      filter          = "resource.type = \"cloudsql_database\" AND resource.label.database_id = \"${var.project_id}:${google_sql_database_instance.aegis.name}\" AND metric.type = \"cloudsql.googleapis.com/database/cpu/utilization\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      duration        = "600s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "Cloud SQL CPU remained high for 10 minutes. Review DB load and query patterns."
  }

  user_labels = {
    service  = "postgres"
    severity = "warning"
  }
}

# --- Uptime check ---

resource "google_monitoring_uptime_check_config" "api_healthz" {
  count        = var.api_domain != "" && var.enable_monitoring_alerts ? 1 : 0
  display_name = "AEGIS API /healthz (${var.environment})"
  timeout      = "10s"
  period       = "60s"

  http_check {
    path         = "/healthz"
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = var.api_domain
    }
  }
}

resource "google_monitoring_alert_policy" "api_uptime" {
  count        = var.api_domain != "" && var.enable_monitoring_alerts ? 1 : 0
  display_name = "AEGIS API uptime check failing (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Uptime check failure"
    condition_threshold {
      filter          = "metric.type = \"monitoring.googleapis.com/uptime_check/check_passed\" AND resource.type = \"uptime_url\" AND metric.label.check_id = \"${google_monitoring_uptime_check_config.api_healthz[0].uptime_check_id}\""
      comparison      = "COMPARISON_LT"
      threshold_value = 1
      duration        = "120s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_NEXT_OLDER"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.label.*"]
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "The API /healthz endpoint is not responding. Check Cloud Run service health, DB connectivity, and load balancer configuration."
  }

  user_labels = {
    service  = "api"
    severity = "critical"
  }
}

resource "google_monitoring_alert_policy" "api_latency" {
  display_name = "AEGIS API p99 latency high (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Cloud Run API request latency p99 > 5s"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND resource.label.service_name = \"${google_cloud_run_v2_service.api.name}\" AND metric.type = \"run.googleapis.com/request_latencies\""
      comparison      = "COMPARISON_GT"
      threshold_value = 5000
      duration        = "300s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_99"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "API p99 latency exceeded 5 seconds. Check for slow DB queries, sidecar timeouts, or cold start spikes. Consider increasing Cloud Run min-instances."
  }

  user_labels = {
    service  = "api"
    severity = "warning"
  }
}

resource "google_monitoring_alert_policy" "cloudsql_disk" {
  display_name = "AEGIS Cloud SQL disk usage high (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Cloud SQL disk utilisation > 85%"
    condition_threshold {
      filter          = "resource.type = \"cloudsql_database\" AND resource.label.database_id = \"${var.project_id}:${google_sql_database_instance.aegis.name}\" AND metric.type = \"cloudsql.googleapis.com/database/disk/utilization\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.85
      duration        = "300s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "Cloud SQL disk is above 85%. Enable storage auto-resize in the GCP Console or increase disk_size in Terraform. See docs/runbooks/alert-response.md."
  }

  user_labels = {
    service  = "postgres"
    severity = "critical"
  }
}

resource "google_monitoring_alert_policy" "cloudsql_connections" {
  display_name = "AEGIS Cloud SQL connections high (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Cloud SQL active connections > 80"
    condition_threshold {
      filter          = "resource.type = \"cloudsql_database\" AND resource.label.database_id = \"${var.project_id}:${google_sql_database_instance.aegis.name}\" AND metric.type = \"cloudsql.googleapis.com/database/postgresql/num_backends\""
      comparison      = "COMPARISON_GT"
      threshold_value = 80
      duration        = "300s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "Active PostgreSQL connections are near the limit. Review Cloud Run max-instances, enable PgBouncer, or increase max_connections in Cloud SQL flags."
  }

  user_labels = {
    service  = "postgres"
    severity = "warning"
  }
}

# --- Log-based metrics for AEGIS pipeline observability ---

resource "google_logging_metric" "pipeline_failures" {
  name   = "aegis-${var.environment}-pipeline-failures"
  filter = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${google_cloud_run_v2_service.api.name}\" AND textPayload:\"pipeline: send failure alert\""
  metric_descriptor {
    metric_kind  = "DELTA"
    value_type   = "INT64"
    unit         = "1"
    display_name = "AEGIS pipeline step failures"
  }
}

resource "google_logging_metric" "study_stuck" {
  name   = "aegis-${var.environment}-study-stuck"
  filter = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${google_cloud_run_v2_service.api.name}\" AND textPayload:\"sla: sent alert\""
  metric_descriptor {
    metric_kind  = "DELTA"
    value_type   = "INT64"
    unit         = "1"
    display_name = "AEGIS stuck study SLA alerts"
  }
}

resource "google_logging_metric" "destination_probe_failures" {
  name   = "aegis-${var.environment}-destination-probe-failures"
  filter = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${google_cloud_run_v2_service.api.name}\" AND textPayload:\"destination.tested\" AND textPayload:\"success\\\":false\""
  metric_descriptor {
    metric_kind  = "DELTA"
    value_type   = "INT64"
    unit         = "1"
    display_name = "AEGIS destination probe failures"
  }
}

resource "google_logging_metric" "dimse_dead_letter" {
  name   = "aegis-${var.environment}-dimse-dead-letter"
  filter = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${google_cloud_run_v2_service.api.name}\" AND textPayload:\"dead-letter\""
  metric_descriptor {
    metric_kind  = "DELTA"
    value_type   = "INT64"
    unit         = "1"
    display_name = "AEGIS DIMSE dead-letter indicators"
  }
}

# Counts every "pipeline: dispatching ..." log line from the Go API.
# Each dispatch fires once per pipeline service dispatched per study, so this
# metric tracks pipeline throughput / study processing activity over time.
resource "google_logging_metric" "pipeline_dispatches" {
  name   = "aegis-${var.environment}-pipeline-dispatches"
  filter = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${google_cloud_run_v2_service.api.name}\" AND textPayload:\"pipeline: dispatching\""
  metric_descriptor {
    metric_kind  = "DELTA"
    value_type   = "INT64"
    unit         = "1"
    display_name = "AEGIS pipeline dispatch activity"
  }
}

# GCP log-based metrics take up to 10 minutes to propagate before alert
# policies can reference them. This sleep guards against a race on first apply.
resource "time_sleep" "wait_for_log_metrics" {
  depends_on = [
    google_logging_metric.pipeline_failures,
    google_logging_metric.study_stuck,
    google_logging_metric.destination_probe_failures,
    google_logging_metric.dimse_dead_letter,
  ]
  create_duration = "600s"
}

# --- Additional alert policies ---

resource "google_monitoring_alert_policy" "cloud_run_memory" {
  display_name = "AEGIS Cloud Run memory high (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts

  conditions {
    display_name = "Cloud Run container memory utilisation > 90% for 10m"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND resource.label.service_name = \"${google_cloud_run_v2_service.api.name}\" AND metric.type = \"run.googleapis.com/container/memory/utilizations\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.9
      duration        = "600s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_99"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "API container memory is above 90%. Review for memory leaks, large DICOM upload processing, or undersized Cloud Run memory limits."
  }

  user_labels = {
    service  = "api"
    severity = "warning"
  }
}

resource "google_monitoring_alert_policy" "study_stuck_alert" {
  display_name = "AEGIS study stuck in pipeline > 30 min (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts
  depends_on   = [time_sleep.wait_for_log_metrics]

  conditions {
    display_name = "Stuck study SLA alert log events"
    condition_threshold {
      filter          = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.study_stuck.name}\" AND resource.type = \"cloud_run_revision\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "0s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_SUM"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "One or more studies have been stuck in a pipeline stage for longer than the SLA threshold. Check GET /api/studies/stuck for details and review the pipeline dashboard for failed sidecar services."
  }

  user_labels = {
    service  = "pipeline"
    severity = "warning"
  }
}

resource "google_monitoring_alert_policy" "pipeline_failure_alert" {
  display_name = "AEGIS pipeline step failure (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts
  depends_on   = [time_sleep.wait_for_log_metrics]

  conditions {
    display_name = "Pipeline failure log events > 3 in 5m"
    condition_threshold {
      filter          = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.pipeline_failures.name}\" AND resource.type = \"cloud_run_revision\""
      comparison      = "COMPARISON_GT"
      threshold_value = 3
      duration        = "0s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_SUM"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "Pipeline step failures are elevated. Check Cloud Run logs for the API service with filter textPayload:\"pipeline: send failure alert\". Review sidecar service health endpoints and study audit trails."
  }

  user_labels = {
    service  = "pipeline"
    severity = "critical"
  }
}

resource "google_monitoring_alert_policy" "destination_probe_failure_alert" {
  display_name = "AEGIS destination probe failures (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts
  depends_on   = [time_sleep.wait_for_log_metrics]

  conditions {
    display_name = "Destination probe failures > 0 in 5m"
    condition_threshold {
      filter          = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.destination_probe_failures.name}\" AND resource.type = \"cloud_run_revision\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "0s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_SUM"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "Destination connectivity probes are failing. Check destination configuration, egress network rules, and /api/destinations/{id}/test results."
  }

  user_labels = {
    service  = "routing"
    severity = "warning"
  }
}

resource "google_monitoring_alert_policy" "dimse_dead_letter_alert" {
  display_name = "AEGIS DIMSE dead-letter risk (${var.environment})"
  combiner     = "OR"
  enabled      = var.enable_monitoring_alerts
  depends_on   = [time_sleep.wait_for_log_metrics]

  conditions {
    display_name = "DIMSE dead-letter indicators > 0 in 5m"
    condition_threshold {
      filter          = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.dimse_dead_letter.name}\" AND resource.type = \"cloud_run_revision\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "0s"
      trigger {
        count = 1
      }
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_SUM"
      }
    }
  }

  notification_channels = local.notification_channels

  documentation {
    content = "DIMSE dead-letter/retry risk detected. Check DIMSE retry status endpoints and ingest queue health before data loss risk increases."
  }

  user_labels = {
    service  = "dimse"
    severity = "critical"
  }
}

# --- Cloud Monitoring Dashboard ---

resource "google_monitoring_dashboard" "aegis" {
  count = var.enable_monitoring_alerts ? 1 : 0
  dashboard_json = templatefile("${path.module}/monitoring_dashboard.json", {
    project_id  = var.project_id
    api_service = google_cloud_run_v2_service.api.name
    environment = var.environment
  })
}

# --- Outputs ---

output "network" {
  value = google_compute_network.aegis.name
}

output "artifact_registry_repository" {
  value = google_artifact_registry_repository.services.id
}

output "dicom_store_raw" {
  value = google_healthcare_dicom_store.raw.self_link
}

output "dicom_store_clean" {
  value = google_healthcare_dicom_store.clean.self_link
}

output "staging_bucket" {
  value = google_storage_bucket.staging.name
}

output "archive_bucket" {
  value = google_storage_bucket.archive.name
}

output "postgres_connection" {
  value     = google_sql_database_instance.aegis.connection_name
  sensitive = true
}

output "postgres_private_ip" {
  value = google_sql_database_instance.aegis.private_ip_address
}

output "db_password_secret_id" {
  value = google_secret_manager_secret.db_password.secret_id
}

output "bigquery_dataset" {
  value = google_bigquery_dataset.aegis.dataset_id
}

output "api_service_name" {
  value = google_cloud_run_v2_service.api.name
}

output "api_service_uri" {
  value = google_cloud_run_v2_service.api.uri
}

output "admin_service_name" {
  value = google_cloud_run_v2_service.admin_dashboard.name
}

output "admin_service_uri" {
  value = google_cloud_run_v2_service.admin_dashboard.uri
}

output "sidecar_service_uris" {
  value = { for name, svc in google_cloud_run_v2_service.sidecars : name => svc.uri }
}

output "weasis_service_uri" {
  value = var.weasis_image != "" ? google_cloud_run_v2_service.weasis[0].uri : ""
}

output "landing_service_uri" {
  value = var.landing_image != "" ? google_cloud_run_v2_service.landing[0].uri : ""
}

output "load_balancer_ip" {
  value = google_compute_global_address.lb_ip.address
}

output "api_base_url" {
  value = "https://${var.api_domain}"
}

output "api_healthz_url" {
  value = "https://${var.api_domain}/healthz"
}

output "api_auth_me_url" {
  value = "https://${var.api_domain}/api/auth/me"
}

output "admin_base_url" {
  value = "https://${var.admin_domain}"
}

output "admin_root_url" {
  value = "https://${var.admin_domain}/"
}

output "cloud_armor_policy_name" {
  value = google_compute_security_policy.api.name
}

output "smtp_egress_ip" {
  value = google_compute_address.smtp_egress_ip.address
}

output "iap_backend_service_name" {
  value = google_compute_backend_service.admin.name
}

output "mcp_server_uri" {
  value = var.mcp_server_image != "" ? google_cloud_run_v2_service.mcp_server[0].uri : ""
}
