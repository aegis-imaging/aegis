# AEGIS — DIMSE Receiver Compute Engine Instance
# Activated: dimse_receiver_image set in tfvars (Secret Manager version 2).
#
# DICOM C-STORE SCP requires raw TCP port 11112, which Cloud Run cannot expose.
# This module deploys dimse-receiver to a Compute Engine VM (Debian 12) that:
#   - Exposes TCP 11112 for inbound DICOM from PACS systems
#   - Mounts the staging GCS bucket via gcsfuse so DICOM files land in the same
#     location the Cloud Run API reads from (STORAGE_MODE=gcs, staging bucket)
#   - Persists retry/dead-letter state on a separate attached persistent disk
#   - Auto-updates via Cloud Build: 'dimse-image' metadata key is updated, then
#     the instance is reset (startup script pulls new image and restarts container)
#
# All resources are conditional on var.dimse_receiver_image != "".

locals {
  dimse_zone    = var.dimse_receiver_zone != "" ? var.dimse_receiver_zone : "${var.region}-a"
  dimse_enabled = var.dimse_receiver_image != ""
}

# Service account for the DIMSE receiver VM
resource "google_service_account" "dimse" {
  count        = local.dimse_enabled ? 1 : 0
  account_id   = "${replace(local.name_prefix, "-", "")}dimse"
  display_name = "AEGIS DIMSE receiver service account"
  project      = var.project_id
}

# Read/write access to staging bucket — DICOM files written here are served by Cloud Run API
resource "google_storage_bucket_iam_member" "dimse_staging_rw" {
  count  = local.dimse_enabled ? 1 : 0
  bucket = google_storage_bucket.staging.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.dimse[0].email}"
}

# Pull container images from Artifact Registry
resource "google_artifact_registry_repository_iam_member" "dimse_ar_reader" {
  count      = local.dimse_enabled ? 1 : 0
  project    = var.project_id
  location   = var.region
  repository = google_artifact_registry_repository.services.repository_id
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.dimse[0].email}"
}

# Write to Cloud Logging
resource "google_project_iam_member" "dimse_log_writer" {
  count   = local.dimse_enabled ? 1 : 0
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.dimse[0].email}"
}

# Write custom metrics to Cloud Monitoring
resource "google_project_iam_member" "dimse_metric_writer" {
  count   = local.dimse_enabled ? 1 : 0
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.dimse[0].email}"
}

# Static regional IP — PACS systems need a fixed address to send DICOM to
resource "google_compute_address" "dimse" {
  count   = local.dimse_enabled ? 1 : 0
  name    = "${local.name_prefix}-dimse-ip"
  region  = var.region
  project = var.project_id
}

# Allow DICOM C-STORE (TCP 11112) inbound from configured source ranges
resource "google_compute_firewall" "dimse_ingress" {
  count   = local.dimse_enabled ? 1 : 0
  name    = "${local.name_prefix}-allow-dimse"
  network = google_compute_network.aegis.name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["11112"]
  }

  source_ranges = var.dimse_source_ranges
  target_tags   = ["dimse-receiver"]
  description   = "Allow DICOM C-STORE (TCP 11112) to DIMSE receiver from PACS systems"
}

# Allow internal HTTP access (port 8080) from VPC — used by Cloud Run API
# to call dimse-receiver's retry/ops endpoints (/ingest/retry*)
resource "google_compute_firewall" "dimse_internal" {
  count   = local.dimse_enabled ? 1 : 0
  name    = "${local.name_prefix}-allow-dimse-internal"
  network = google_compute_network.aegis.name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["8080"]
  }

  source_ranges = [var.vpc_cidr]
  target_tags   = ["dimse-receiver"]
  description   = "Allow VPC-internal HTTP access to DIMSE receiver ops API (port 8080)"
}

# Persistent disk for durable retry/dead-letter state across VM restarts
resource "google_compute_disk" "dimse_data" {
  count   = local.dimse_enabled ? 1 : 0
  name    = "${local.name_prefix}-dimse-data"
  type    = "pd-ssd"
  size    = 10
  zone    = local.dimse_zone
  project = var.project_id

  labels = {
    service     = "dimse-receiver"
    environment = var.environment
  }
}

# The VM — Debian 12 chosen over Container-Optimized OS because gcsfuse
# is easier to install and configure on standard Debian. Cloud Build deploys
# by updating the 'dimse-image' metadata key and resetting the instance;
# the startup script pulls the new image on every boot.
resource "google_compute_instance" "dimse_receiver" {
  count        = local.dimse_enabled ? 1 : 0
  name         = "${local.name_prefix}-dimse-receiver"
  machine_type = var.dimse_receiver_machine_type
  zone         = local.dimse_zone
  project      = var.project_id

  tags = ["dimse-receiver"]

  # Allow Terraform to stop the instance when changing machine type or disk config
  allow_stopping_for_update = true

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 20
      type  = "pd-balanced"
    }
  }

  attached_disk {
    source      = google_compute_disk.dimse_data[0].self_link
    device_name = "dimse-data"
  }

  network_interface {
    network    = google_compute_network.aegis.name
    subnetwork = google_compute_subnetwork.app.name
    access_config {
      nat_ip = google_compute_address.dimse[0].address
    }
  }

  service_account {
    email  = google_service_account.dimse[0].email
    scopes = ["cloud-platform"]
  }

  metadata = {
    # 'dimse-image' is updated by Cloud Build on each deploy without triggering
    # a full terraform apply. See lifecycle.ignore_changes below.
    dimse-image = var.dimse_receiver_image

    # Startup script runs on every boot: installs deps, mounts storage, starts container.
    startup-script = <<-SCRIPT
      #!/bin/bash
      # AEGIS DIMSE Receiver startup script
      # Runs on every boot — installs dependencies, mounts storage, starts container.
      set -euo pipefail
      exec > >(tee -a /var/log/dimse-startup.log) 2>&1
      echo "==> AEGIS DIMSE startup: $(date)"

      # ---- Install Docker (idempotent) ----
      if ! command -v docker &>/dev/null; then
        echo "==> Installing Docker..."
        curl -fsSL https://get.docker.com | sh
      fi

      # ---- Install gcsfuse (idempotent) ----
      if ! command -v gcsfuse &>/dev/null; then
        echo "==> Installing gcsfuse..."
        apt-get install -y -q lsb-release gnupg
        CODENAME=$(lsb_release -cs)
        GCSFUSE_REPO="gcsfuse-$CODENAME"
        curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg \
          | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
        echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt $${GCSFUSE_REPO} main" \
          > /etc/apt/sources.list.d/gcsfuse.list
        apt-get update -q && apt-get install -y -q gcsfuse
      fi

      # ---- Format + mount persistent data disk (idempotent) ----
      DISK_DEV="/dev/disk/by-id/google-dimse-data"
      MOUNT_DATA="/mnt/dimse-data"
      mkdir -p "$MOUNT_DATA"
      if ! mountpoint -q "$MOUNT_DATA"; then
        if ! blkid "$DISK_DEV" 2>/dev/null | grep -q ext4; then
          echo "==> Formatting data disk..."
          mkfs.ext4 -F "$DISK_DEV"
        fi
        mount -o discard,defaults "$DISK_DEV" "$MOUNT_DATA"
        grep -qF "$DISK_DEV" /etc/fstab || \
          echo "$DISK_DEV $MOUNT_DATA ext4 discard,defaults 0 2" >> /etc/fstab
      fi

      # ---- Mount staging GCS bucket at /app/data ----
      # DICOM files written here are in the same GCS bucket the Cloud Run API reads from.
      STAGING_BUCKET="${google_storage_bucket.staging.name}"
      mkdir -p /app/data
      if ! mountpoint -q /app/data; then
        echo "==> Mounting GCS bucket $STAGING_BUCKET at /app/data..."
        gcsfuse --implicit-dirs --file-mode=0666 --dir-mode=0777 \
          "$STAGING_BUCKET" /app/data
      fi

      # ---- Pull and start container ----
      # Image URI is read from instance metadata — updated by Cloud Build on each deploy.
      IMAGE=$(curl -sf -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/attributes/dimse-image" \
        || echo "${var.dimse_receiver_image}")

      echo "==> Configuring Docker auth for Artifact Registry..."
      gcloud auth configure-docker ${var.region}-docker.pkg.dev --quiet

      echo "==> Pulling image: $IMAGE"
      docker pull "$IMAGE"

      echo "==> Starting dimse-receiver container..."
      docker stop dimse-receiver 2>/dev/null || true
      docker rm   dimse-receiver 2>/dev/null || true
      docker run -d \
        --name dimse-receiver \
        --restart unless-stopped \
        --log-driver=gcplogs \
        --log-opt gcp-project=${var.project_id} \
        -p 11112:11112 \
        -p 8080:8080 \
        -v /app/data:/app/data \
        -v /mnt/dimse-data:/app/persist \
        -e DIMSE_DATA_DIR=/app/data \
        -e DIMSE_INGEST_DURABLE_STORE_PATH=/app/persist/dimse-ingest-retry-state.json \
        -e API_URL="${var.dimse_api_url}" \
        -e DIMSE_PROJECT_SLUG="${var.dimse_project_slug}" \
        "$IMAGE"

      echo "==> DIMSE startup complete: $(date)"
    SCRIPT
  }

  labels = {
    service     = "dimse-receiver"
    environment = var.environment
  }

  # Cloud Build updates 'dimse-image' metadata via gcloud outside of Terraform.
  # Ignore this key so terraform apply doesn't revert it to the tfvars value.
  lifecycle {
    ignore_changes = [metadata["dimse-image"]]
  }
}

output "dimse_receiver_ip" {
  description = "Static external IP for DIMSE receiver. Configure PACS C-STORE destination to this address on TCP port 11112."
  value       = local.dimse_enabled ? google_compute_address.dimse[0].address : null
}

output "dimse_receiver_instance_name" {
  description = "GCE instance name for DIMSE receiver"
  value       = local.dimse_enabled ? google_compute_instance.dimse_receiver[0].name : null
}

output "dimse_receiver_zone" {
  description = "Zone of the DIMSE receiver instance"
  value       = local.dimse_enabled ? local.dimse_zone : null
}
