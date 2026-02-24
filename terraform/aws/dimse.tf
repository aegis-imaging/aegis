# ── AWS EC2 — DIMSE Receiver ───────────────────────────────────────────────────
#
# DICOM C-STORE SCP requires raw TCP port 11112, which ECS Fargate cannot expose.
# This module deploys dimse-receiver to a t3.small EC2 instance (Amazon Linux 2023)
# in a public subnet so PACS systems can reach it on port 11112.
#
# CI/CD pattern (mirrors GCP dimse.tf):
#   - SSM Parameter Store holds the current image URI (/aegis/dimse-image)
#   - GitHub Actions updates the SSM param and reboots the instance after each push
#   - The startup script reads the SSM param on every boot and re-pulls the image
#   - lifecycle { ignore_changes = [user_data] } prevents terraform apply from
#     reverting to the tfvars value after CI/CD has updated the SSM param
#
# All resources are conditional on var.dimse_receiver_image != "".

locals {
  dimse_enabled = var.dimse_receiver_image != ""
}

# ── IAM — instance profile ────────────────────────────────────────────────────

data "aws_iam_policy_document" "dimse_assume_role" {
  count = local.dimse_enabled ? 1 : 0

  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "dimse" {
  count              = local.dimse_enabled ? 1 : 0
  name               = "${var.project_name}-dimse-receiver"
  assume_role_policy = data.aws_iam_policy_document.dimse_assume_role[0].json

  tags = {
    Name        = "${var.project_name}-dimse-receiver"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

data "aws_iam_policy_document" "dimse_policy" {
  count = local.dimse_enabled ? 1 : 0

  # ECR — pull container images
  statement {
    effect = "Allow"
    actions = [
      "ecr:GetAuthorizationToken",
      "ecr:BatchCheckLayerAvailability",
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage"
    ]
    resources = ["*"]
  }

  # S3 — read/write DICOM files
  statement {
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket"
    ]
    resources = [
      aws_s3_bucket.dicom.arn,
      "${aws_s3_bucket.dicom.arn}/*"
    ]
  }

  # SSM — read the current image URI on startup
  statement {
    effect    = "Allow"
    actions   = ["ssm:GetParameter"]
    resources = ["arn:aws:ssm:${var.aws_region}:*:parameter/aegis/dimse-image"]
  }

  # CloudWatch — container logs
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["*"]
  }

  # KMS — decrypt S3 objects encrypted with the project key
  statement {
    effect    = "Allow"
    actions   = ["kms:Decrypt", "kms:GenerateDataKey"]
    resources = [aws_kms_key.main.arn]
  }
}

resource "aws_iam_role_policy" "dimse" {
  count  = local.dimse_enabled ? 1 : 0
  name   = "${var.project_name}-dimse-policy"
  role   = aws_iam_role.dimse[0].id
  policy = data.aws_iam_policy_document.dimse_policy[0].json
}

resource "aws_iam_instance_profile" "dimse" {
  count = local.dimse_enabled ? 1 : 0
  name  = "${var.project_name}-dimse-receiver"
  role  = aws_iam_role.dimse[0].name

  tags = {
    Name        = "${var.project_name}-dimse-receiver"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── Security group ─────────────────────────────────────────────────────────────

resource "aws_security_group" "dimse" {
  count       = local.dimse_enabled ? 1 : 0
  name_prefix = "${var.project_name}-dimse-"
  vpc_id      = aws_vpc.main.id

  # DICOM C-STORE from PACS systems — restrict this to known PACS IP ranges
  # in production (kept open here for initial bring-up; tighten via NACL or
  # separate SG rule once PACS source IPs are known)
  ingress {
    description = "DICOM C-STORE SCP"
    from_port   = 11112
    to_port     = 11112
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTP ops API — internal callers only (Go API → /ingest/retry* endpoints)
  ingress {
    description = "DIMSE ops API from VPC"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.project_name}-dimse-sg"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── SSM Parameter — stores current image URI; updated by GitHub Actions ───────

resource "aws_ssm_parameter" "dimse_image" {
  count = local.dimse_enabled ? 1 : 0

  name      = "/aegis/dimse-image"
  type      = "String"
  value     = var.dimse_receiver_image
  overwrite = true

  # GitHub Actions writes a new value here on each deploy without running
  # terraform apply. Ignore so terraform plan doesn't flag it as drift.
  lifecycle {
    ignore_changes = [value]
  }

  tags = {
    Name        = "${var.project_name}-dimse-image"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── Elastic IP — stable address for PACS AE title registration ────────────────

resource "aws_eip" "dimse" {
  count  = local.dimse_enabled ? 1 : 0
  domain = "vpc"

  tags = {
    Name        = "${var.project_name}-dimse-eip"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_eip_association" "dimse" {
  count         = local.dimse_enabled ? 1 : 0
  instance_id   = aws_instance.dimse_receiver[0].id
  allocation_id = aws_eip.dimse[0].id
}

# ── AMI data source — latest Amazon Linux 2023 ────────────────────────────────

data "aws_ami" "al2023" {
  count       = local.dimse_enabled ? 1 : 0
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# ── EC2 instance ──────────────────────────────────────────────────────────────

resource "aws_instance" "dimse_receiver" {
  count = local.dimse_enabled ? 1 : 0

  ami                    = data.aws_ami.al2023[0].id
  instance_type          = "t3.small"
  subnet_id              = aws_subnet.public_a.id
  vpc_security_group_ids = [aws_security_group.dimse[0].id]
  iam_instance_profile   = aws_iam_instance_profile.dimse[0].name

  # User data runs once on first boot (and after reboot triggered by CI/CD).
  # CI/CD updates the SSM parameter and reboots the instance; the startup script
  # always reads the SSM param so it pulls whatever image CI/CD last deployed.
  user_data = base64encode(<<-SCRIPT
    #!/bin/bash
    # AEGIS DIMSE Receiver startup script (Amazon Linux 2023)
    # Reads the current image URI from SSM Parameter Store and starts the container.
    set -euo pipefail
    exec > >(tee -a /var/log/dimse-startup.log) 2>&1
    echo "==> AEGIS DIMSE startup: $(date)"

    # ---- Install Docker (idempotent) ----
    if ! command -v docker &>/dev/null; then
      echo "==> Installing Docker..."
      dnf install -y docker
      systemctl enable --now docker
    fi

    # ---- Wait for instance identity to be available ----
    AWS_REGION="${var.aws_region}"
    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text --region "$AWS_REGION")

    # ---- Authenticate Docker to ECR ----
    echo "==> Logging in to ECR..."
    aws ecr get-login-password --region "$AWS_REGION" \
      | docker login --username AWS --password-stdin \
          "$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

    # ---- Read the current image URI from SSM ----
    IMAGE=$(aws ssm get-parameter \
      --name /aegis/dimse-image \
      --query 'Parameter.Value' \
      --output text \
      --region "$AWS_REGION" \
      || echo "${var.dimse_receiver_image}")

    echo "==> Pulling image: $IMAGE"
    docker pull "$IMAGE"

    # ---- Start the container ----
    echo "==> Starting dimse-receiver container..."
    docker stop dimse-receiver 2>/dev/null || true
    docker rm   dimse-receiver 2>/dev/null || true
    # Bind-mount /var/aegis-dimse on the host for retry/dead-letter state
    # persistence across container restarts (DICOM files go directly to S3).
    mkdir -p /var/aegis-dimse
    docker run -d \
      --name dimse-receiver \
      --restart unless-stopped \
      -p 11112:11112 \
      -p 8080:8080 \
      -v /var/aegis-dimse:/app/data \
      -e DIMSE_DATA_DIR=/app/data \
      -e DIMSE_INGEST_DURABLE_STORE_PATH=/app/data/dimse-ingest-retry-state.json \
      -e API_URL="http://api.aegis.local:8080" \
      -e DIMSE_PROJECT_SLUG="default" \
      -e STORAGE_MODE="s3" \
      -e S3_BUCKET="${aws_s3_bucket.dicom.bucket}" \
      -e S3_REGION="${var.aws_region}" \
      "$IMAGE"

    echo "==> DIMSE startup complete: $(date)"
  SCRIPT
  )

  # CI/CD updates the SSM parameter and reboots the instance to deploy a new
  # image — it does NOT update user_data. Ignore user_data drift so terraform
  # plan stays clean after CI/CD deploys.
  lifecycle {
    ignore_changes = [user_data, ami]
  }

  tags = {
    Name        = "${var.project_name}-dimse-receiver"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── Outputs ───────────────────────────────────────────────────────────────────

output "dimse_receiver_ip" {
  description = "Static Elastic IP for DIMSE receiver — configure PACS C-STORE destination to this address on TCP port 11112."
  value       = local.dimse_enabled ? aws_eip.dimse[0].public_ip : null
}

output "dimse_receiver_instance_id" {
  description = "EC2 instance ID for DIMSE receiver"
  value       = local.dimse_enabled ? aws_instance.dimse_receiver[0].id : null
}

output "dimse_ssm_parameter" {
  description = "SSM parameter name that stores the current DIMSE image URI (updated by GitHub Actions)"
  value       = local.dimse_enabled ? aws_ssm_parameter.dimse_image[0].name : null
}
