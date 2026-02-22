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

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
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
  description = "Container image URI for the admin dashboard service"
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
  default     = "noreply@aegis.local"
}

variable "allowed_origins" {
  description = "Optional CORS origins override. If empty, defaults to API + admin domains."
  type        = list(string)
  default     = []
}

provider "google" {
  project = var.project_id
  region  = var.region
}

locals {
  name_prefix                    = "aegis-${var.environment}"
  resolved_db_password_secret_id = var.db_password_secret_id != "" ? var.db_password_secret_id : "${local.name_prefix}-db-password"
  resolved_db_password           = var.db_password != "" ? var.db_password : try(random_password.db_password[0].result, "")

  sidecar_services = {
    defacing               = var.defacing_image
    phi-detection          = var.phi_detection_image
    qc-service             = var.qc_service_image
    bids-service           = var.bids_service_image
    classification-service = var.classification_service_image
    protocol-service       = var.protocol_service_image
  }

  lb_domains = distinct([var.api_domain, var.admin_domain])

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

# --- Cloud Run sidecars ---

resource "google_cloud_run_v2_service" "sidecars" {
  for_each = local.sidecar_services

  name     = each.key
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  deletion_protection = var.deletion_protection

  template {
    service_account = google_service_account.sidecars.email

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

      liveness_probe {
        failure_threshold     = 5
        initial_delay_seconds = 15
        timeout_seconds       = 5
        period_seconds        = 15

        http_get {
          path = "/healthz"
        }
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.cloud_run.id
      egress    = "ALL_TRAFFIC"
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
        value = "/tmp/aegis-data"
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

      liveness_probe {
        failure_threshold     = 5
        initial_delay_seconds = 20
        timeout_seconds       = 5
        period_seconds        = 15

        http_get {
          path = "/healthz"
        }
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.cloud_run.id
      egress    = "ALL_TRAFFIC"
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
  name = "${local.name_prefix}-lb-cert"
  managed {
    domains = local.lb_domains
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
  default_service = google_compute_backend_service.api.id

  host_rule {
    hosts        = [var.api_domain]
    path_matcher = "api"
  }

  host_rule {
    hosts        = [var.admin_domain]
    path_matcher = "admin"
  }

  path_matcher {
    name            = "api"
    default_service = google_compute_backend_service.api.id
  }

  path_matcher {
    name            = "admin"
    default_service = google_compute_backend_service.admin.id
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
