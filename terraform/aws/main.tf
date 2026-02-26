# AEGIS — AWS Infrastructure
#
# This module provisions the core infrastructure on AWS, functionally equivalent
# to the GCP module in terraform/infra/. AEGIS is cloud-agnostic at the
# application layer; this module provides the AWS-native underpinning.
#
# What it provisions:
# - VPC, subnets (public + private), NAT Gateway
# - S3 bucket for DICOM file storage
# - RDS PostgreSQL 15 instance for application data
# - ECS Fargate cluster + service definitions (API + sidecar tasks)
# - Application Load Balancer with Cognito authentication
# - ECR repositories for container images
# - SES for transactional email (or SMTP relay)
# - SNS + SQS for event notifications
# - KMS key for encryption at rest
#
# Depends on: AWS account configured, Terraform AWS provider authenticated

terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "aws_region" {
  description = "Primary AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "aegis"
}

variable "db_master_username" {
  description = "RDS master username (password is generated and rotated in AWS Secrets Manager)"
  type        = string
  default     = "aegis"
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN for ALB HTTPS listener"
  type        = string
}

variable "cognito_domain_prefix" {
  description = "Unique Cognito hosted UI domain prefix (for example: aegis-dev-auth)"
  type        = string
}

variable "cognito_callback_urls" {
  description = "OAuth callback URLs for Cognito app client (defaults to ALB /oauth2/idpresponse)"
  type        = list(string)
  default     = []
}

variable "cognito_logout_urls" {
  description = "OAuth logout URLs for Cognito app client (defaults to ALB /logout)"
  type        = list(string)
  default     = []
}

variable "cognito_allowed_oauth_scopes" {
  description = "OAuth scopes requested by ALB authenticate-cognito action"
  type        = list(string)
  default     = ["openid", "email", "profile"]
}

variable "api_image_tag" {
  description = "Container image tag used for the API ECS task"
  type        = string
  default     = "latest"
}

variable "api_desired_count" {
  description = "Desired number of API ECS tasks"
  type        = number
  default     = 1
}

variable "api_cpu" {
  description = "CPU units for API ECS task definition"
  type        = number
  default     = 1024
}

variable "api_memory" {
  description = "Memory (MiB) for API ECS task definition"
  type        = number
  default     = 2048
}

variable "api_allowed_origins" {
  description = "Optional CORS origins override for API runtime"
  type        = list(string)
  default     = []
}

variable "api_domain" {
  description = "Custom FQDN for the API (e.g. aws.api.aegisimaging.ai). Empty = use raw ALB DNS."
  type        = string
  default     = ""
}

variable "admin_domain" {
  description = "Custom FQDN for the admin dashboard (e.g. aws.admin.aegisimaging.ai). Empty = use raw ALB DNS."
  type        = string
  default     = ""
}

variable "admin_image_tag" {
  description = "Container image tag used for the admin dashboard ECS task"
  type        = string
  default     = "latest"
}

variable "admin_desired_count" {
  description = "Desired number of admin dashboard ECS tasks"
  type        = number
  default     = 1
}

variable "admin_cpu" {
  description = "CPU units for admin dashboard ECS task definition"
  type        = number
  default     = 512
}

variable "admin_memory" {
  description = "Memory (MiB) for admin dashboard ECS task definition"
  type        = number
  default     = 1024
}

variable "sidecar_image_tag" {
  description = "Container image tag used for all sidecar ECS tasks"
  type        = string
  default     = "latest"
}

variable "dimse_receiver_image" {
  description = "Full ECR image URI for the DIMSE receiver EC2 instance (empty = skip all DIMSE resources)"
  type        = string
  default     = ""
}

variable "dimse_source_ranges" {
  description = "CIDR ranges allowed to reach DIMSE C-STORE on TCP 11112. Restrict to known PACS IPs in production."
  type        = list(string)
  default     = ["203.0.113.0/24"] # RFC 5737 TEST-NET-3 placeholder — replace with real PACS IP ranges
}

variable "weasis_domain" {
  description = "Custom FQDN for Weasis viewer (e.g. aws.weasis.aegisimaging.ai). Empty = use raw ALB DNS."
  type        = string
  default     = ""
}

variable "weasis_image_tag" {
  description = "Container image tag for Weasis ECS task"
  type        = string
  default     = "latest"
}

variable "weasis_cpu" {
  description = "CPU units for Weasis ECS task definition"
  type        = number
  default     = 256
}

variable "weasis_memory" {
  description = "Memory (MiB) for Weasis ECS task definition"
  type        = number
  default     = 512
}

variable "smtp_from" {
  description = "SMTP FROM address for AEGIS transactional email"
  type        = string
  default     = "noreply@aegisimaging.ai"
}

variable "ses_smtp_region" {
  description = "AWS region for the SES SMTP endpoint. Defaults to var.aws_region."
  type        = string
  default     = ""
}

variable "first_admin_email" {
  description = "Seeds the first admin user in admin_users on startup (idempotent). Set to ops email."
  type        = string
  default     = ""
}

variable "mcp_image_tag" {
  description = "Container image tag for the MCP server."
  type        = string
  default     = "latest"
}

variable "mcp_aegis_api_token" {
  description = "AEGIS API bearer token for the MCP server (stored in Secrets Manager; sensitive)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "landing_domain" {
  description = "Custom FQDN for the landing page (e.g. aws.aegisimaging.ai). Empty = no landing page."
  type        = string
  default     = ""
}

variable "landing_image_tag" {
  description = "Container image tag for the landing page ECS task. Empty = skip landing page resources."
  type        = string
  default     = ""
}

variable "landing_cpu" {
  description = "CPU units for landing page ECS task definition"
  type        = number
  default     = 256
}

variable "landing_memory" {
  description = "Memory (MiB) for landing page ECS task definition"
  type        = number
  default     = 512
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "aegis"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# --- Networking ---

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = { Name = "${var.project_name}-vpc" }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.project_name}-igw" }
}

resource "aws_subnet" "public_a" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "${var.aws_region}a"
  map_public_ip_on_launch = true
  tags                    = { Name = "${var.project_name}-public-a" }
}

resource "aws_subnet" "public_b" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.2.0/24"
  availability_zone       = "${var.aws_region}b"
  map_public_ip_on_launch = true
  tags                    = { Name = "${var.project_name}-public-b" }
}

resource "aws_subnet" "private_a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.10.0/24"
  availability_zone = "${var.aws_region}a"
  tags              = { Name = "${var.project_name}-private-a" }
}

resource "aws_subnet" "private_b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.11.0/24"
  availability_zone = "${var.aws_region}b"
  tags              = { Name = "${var.project_name}-private-b" }
}

resource "aws_eip" "nat" {
  domain = "vpc"
  tags   = { Name = "${var.project_name}-nat-eip" }
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public_a.id
  tags          = { Name = "${var.project_name}-nat" }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.project_name}-public-rt" }

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.project_name}-private-rt" }

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }
}

resource "aws_route_table_association" "public_a" {
  subnet_id      = aws_subnet.public_a.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "public_b" {
  subnet_id      = aws_subnet.public_b.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "private_a" {
  subnet_id      = aws_subnet.private_a.id
  route_table_id = aws_route_table.private.id
}

resource "aws_route_table_association" "private_b" {
  subnet_id      = aws_subnet.private_b.id
  route_table_id = aws_route_table.private.id
}

# --- KMS ---

resource "aws_kms_key" "main" {
  description         = "AEGIS encryption key"
  enable_key_rotation = true
  tags                = { Name = "${var.project_name}-kms" }
}

resource "aws_kms_alias" "main" {
  name          = "alias/${var.project_name}"
  target_key_id = aws_kms_key.main.key_id
}

# --- S3 (DICOM file storage) ---

resource "aws_s3_bucket" "dicom" {
  bucket = "${var.project_name}-dicom-${var.environment}"
  tags   = { Name = "${var.project_name}-dicom" }
}

resource "aws_s3_bucket_versioning" "dicom" {
  bucket = aws_s3_bucket.dicom.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "dicom" {
  bucket = aws_s3_bucket.dicom.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.main.arn
    }
  }
}

resource "aws_s3_bucket_public_access_block" "dicom" {
  bucket = aws_s3_bucket.dicom.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "dicom" {
  bucket = aws_s3_bucket.dicom.id

  rule {
    id     = "staging-cleanup"
    status = "Enabled"

    filter { prefix = "staging/" }

    expiration { days = 7 }
  }

  rule {
    id     = "archive-to-glacier"
    status = "Enabled"

    filter { prefix = "dicom/" }

    transition {
      days          = 30
      storage_class = "GLACIER_IR"
    }
  }
}

# --- RDS (PostgreSQL) ---

resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db"
  subnet_ids = [aws_subnet.private_a.id, aws_subnet.private_b.id]
  tags       = { Name = "${var.project_name}-db-subnets" }
}

resource "aws_security_group" "rds" {
  name_prefix = "${var.project_name}-rds-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project_name}-rds-sg" }
}

resource "aws_db_instance" "main" {
  identifier     = "${var.project_name}-postgres"
  engine         = "postgres"
  engine_version = "15"
  instance_class = "db.t3.micro" # Upgrade for production

  allocated_storage     = 20
  max_allocated_storage = 100
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.main.arn

  db_name                       = "aegis"
  username                      = var.db_master_username
  manage_master_user_password   = true
  master_user_secret_kms_key_id = aws_kms_key.main.arn

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  backup_retention_period   = 7
  multi_az                  = false # Enable for production HA
  deletion_protection       = true
  skip_final_snapshot       = false
  final_snapshot_identifier = "${var.project_name}-final-snapshot"

  tags = { Name = "${var.project_name}-postgres" }
}

# --- ECR (Container Registry) ---

locals {
  services = ["api", "admin-dashboard", "defacing", "phi-detection", "qc-service", "bids-service", "classification-service", "protocol-service", "synth-service", "dimse-receiver", "weasis", "mcp-server", "landing"]

  api_image    = "${aws_ecr_repository.services["api"].repository_url}:${var.api_image_tag}"
  admin_image  = "${aws_ecr_repository.services["admin-dashboard"].repository_url}:${var.admin_image_tag}"
  weasis_image = "${aws_ecr_repository.services["weasis"].repository_url}:${var.weasis_image_tag}"

  # Friendly FQDNs — use custom domains when set, fall back to raw ALB DNS.
  api_fqdn    = var.api_domain != "" ? var.api_domain : aws_lb.main.dns_name
  admin_fqdn  = var.admin_domain != "" ? var.admin_domain : aws_lb.main.dns_name
  weasis_fqdn = var.weasis_domain != "" ? var.weasis_domain : aws_lb.main.dns_name

  cognito_callback_urls = length(var.cognito_callback_urls) > 0 ? var.cognito_callback_urls : [
    "https://${local.admin_fqdn}/oauth2/idpresponse"
  ]

  cognito_logout_urls = length(var.cognito_logout_urls) > 0 ? var.cognito_logout_urls : [
    "https://${local.admin_fqdn}/logout"
  ]

  resolved_api_allowed_origins = length(var.api_allowed_origins) > 0 ? var.api_allowed_origins : [
    "https://${local.admin_fqdn}",
    "https://${local.weasis_fqdn}",
  ]

  # SES SMTP endpoint — region-specific. Use ses_smtp_region override when set,
  # otherwise fall back to the primary deployment region.
  ses_smtp_region   = var.ses_smtp_region != "" ? var.ses_smtp_region : var.aws_region
  ses_smtp_hostname = "email-smtp.${local.ses_smtp_region}.amazonaws.com"

  public_path_rules = {
    healthz = {
      priority = 10
      paths    = ["/healthz"]
    }
    upload = {
      priority = 20
      paths    = ["/api/upload/*"]
    }
    export = {
      priority = 30
      paths    = ["/api/export/*"]
    }
    contact = {
      priority = 40
      paths    = ["/api/contact"]
    }
    storage = {
      priority = 50
      paths    = ["/api/storage/*"]
    }
    projects = {
      priority = 60
      paths    = ["/api/projects"]
    }
    active_profile = {
      priority = 70
      paths    = ["/api/projects/*/active-anon-profile"]
    }
    dicomweb = {
      priority = 80
      paths    = ["/dicomweb*", "/dicomweb-raw*"]
    }
    stow = {
      priority = 85
      paths    = ["/api/stow", "/api/stow/*"]
    }
  }
}

resource "aws_ecr_repository" "services" {
  for_each = toset(local.services)

  name                 = "${var.project_name}/${each.value}"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration { scan_on_push = true }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.main.arn
  }

  tags = { Name = "${var.project_name}-${each.value}" }
}

# --- ECS (Container Orchestration) ---

resource "aws_ecs_cluster" "main" {
  name = "${var.project_name}-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = { Name = "${var.project_name}-cluster" }
}

resource "aws_security_group" "ecs_tasks" {
  name_prefix = "${var.project_name}-ecs-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  # Allow inter-service communication within private subnets.
  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project_name}-ecs-sg" }
}

# --- Application Load Balancer ---

resource "aws_security_group" "alb" {
  name_prefix = "${var.project_name}-alb-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project_name}-alb-sg" }
}

resource "aws_lb" "main" {
  name               = "${var.project_name}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = [aws_subnet.public_a.id, aws_subnet.public_b.id]

  tags = { Name = "${var.project_name}-alb" }
}

resource "aws_lb_target_group" "api" {
  name        = "${var.project_name}-api"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.main.id

  health_check {
    path                = "/healthz"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  tags = { Name = "${var.project_name}-api-tg" }
}

resource "aws_lb_target_group" "admin" {
  name        = "${var.project_name}-admin"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.main.id

  health_check {
    path                = "/healthz"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  tags = { Name = "${var.project_name}-admin-tg" }
}

resource "aws_lb_target_group" "weasis" {
  name        = "${var.project_name}-weasis"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.main.id

  health_check {
    path                = "/"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
    matcher             = "200-399"
  }

  tags = { Name = "${var.project_name}-weasis-tg" }
}

# HTTP listener — redirects to HTTPS
resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

# Cognito user pool for ALB authenticate-cognito action.
resource "aws_cognito_user_pool" "admin" {
  name = "${var.project_name}-${var.environment}-admin-users"

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  mfa_configuration = "OFF"

  password_policy {
    minimum_length                   = 12
    require_lowercase                = true
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = false
    temporary_password_validity_days = 7
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  admin_create_user_config {
    allow_admin_create_user_only = true
  }

  tags = { Name = "${var.project_name}-admin-user-pool" }
}

resource "aws_cognito_user_pool_client" "admin" {
  name         = "${var.project_name}-${var.environment}-alb-client"
  user_pool_id = aws_cognito_user_pool.admin.id

  generate_secret = true

  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = var.cognito_allowed_oauth_scopes
  supported_identity_providers         = ["COGNITO"]

  callback_urls = local.cognito_callback_urls
  logout_urls   = local.cognito_logout_urls
}

resource "aws_cognito_user_pool_domain" "admin" {
  domain       = var.cognito_domain_prefix
  user_pool_id = aws_cognito_user_pool.admin.id
}

# HTTPS listener with Cognito auth.
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.acm_certificate_arn

  # Public routes remain unauthenticated via explicit higher-priority listener rules.
  default_action {
    type = "authenticate-cognito"

    authenticate_cognito {
      user_pool_arn              = aws_cognito_user_pool.admin.arn
      user_pool_client_id        = aws_cognito_user_pool_client.admin.id
      user_pool_domain           = aws_cognito_user_pool_domain.admin.domain
      on_unauthenticated_request = "authenticate"
      scope                      = join(" ", var.cognito_allowed_oauth_scopes)
    }
  }

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.admin.arn
  }
}

resource "aws_lb_listener_rule" "https_public_paths" {
  for_each = local.public_path_rules

  listener_arn = aws_lb_listener.https.arn
  priority     = each.value.priority

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }

  condition {
    path_pattern {
      values = each.value.paths
    }
  }
}

resource "aws_lb_listener_rule" "https_api_authenticated" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 90

  action {
    type = "authenticate-cognito"

    authenticate_cognito {
      user_pool_arn              = aws_cognito_user_pool.admin.arn
      user_pool_client_id        = aws_cognito_user_pool_client.admin.id
      user_pool_domain           = aws_cognito_user_pool_domain.admin.domain
      on_unauthenticated_request = "authenticate"
      scope                      = join(" ", var.cognito_allowed_oauth_scopes)
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }

  condition {
    path_pattern {
      values = ["/api/*"]
    }
  }
}

# ── Host-based routing — custom subdomains (e.g. aws.api.aegisimaging.ai) ──────
# Created only when api_domain / admin_domain are set in terraform.tfvars.
# Priorities 1 and 2 fire before all path-based rules.
# api_domain → forwards directly to API target group (Go API handles its own auth).
# admin_domain → Cognito authenticate-cognito + forward to admin target group.

resource "aws_lb_listener_rule" "api_subdomain" {
  count        = var.api_domain != "" ? 1 : 0
  listener_arn = aws_lb_listener.https.arn
  priority     = 1

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }

  condition {
    host_header {
      values = [var.api_domain]
    }
  }
}

resource "aws_lb_listener_rule" "admin_subdomain" {
  count        = var.admin_domain != "" ? 1 : 0
  listener_arn = aws_lb_listener.https.arn
  priority     = 2

  action {
    type = "authenticate-cognito"

    authenticate_cognito {
      user_pool_arn              = aws_cognito_user_pool.admin.arn
      user_pool_client_id        = aws_cognito_user_pool_client.admin.id
      user_pool_domain           = aws_cognito_user_pool_domain.admin.domain
      on_unauthenticated_request = "authenticate"
      scope                      = join(" ", var.cognito_allowed_oauth_scopes)
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.admin.arn
  }

  condition {
    host_header {
      values = [var.admin_domain]
    }
  }
}

resource "aws_lb_listener_rule" "weasis_subdomain" {
  count        = var.weasis_domain != "" ? 1 : 0
  listener_arn = aws_lb_listener.https.arn
  priority     = 3

  # Weasis is served as a public iframe target — no Cognito gate on the container itself.
  # Security is provided by the Go API's DICOMweb auth (X-Amzn-Oidc-Data on /api/* calls).
  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.weasis.arn
  }

  condition {
    host_header {
      values = [var.weasis_domain]
    }
  }
}

# --- ECS API runtime ---

data "aws_iam_policy_document" "ecs_task_execution_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ecs_task_execution" {
  name               = "${var.project_name}-ecs-task-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_execution_assume_role.json
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_managed" {
  role       = aws_iam_role.ecs_task_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

data "aws_iam_policy_document" "ecs_task_execution_secrets" {
  statement {
    effect = "Allow"
    actions = [
      "secretsmanager:GetSecretValue",
      "kms:Decrypt"
    ]
    resources = [
      aws_db_instance.main.master_user_secret[0].secret_arn,
      aws_secretsmanager_secret.smtp_password.arn,
      aws_kms_key.main.arn
    ]
  }
}

resource "aws_iam_role_policy" "ecs_task_execution_secrets" {
  name   = "${var.project_name}-ecs-task-execution-secrets"
  role   = aws_iam_role.ecs_task_execution.id
  policy = data.aws_iam_policy_document.ecs_task_execution_secrets.json
}

data "aws_iam_policy_document" "ecs_task_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ecs_task" {
  name               = "${var.project_name}-ecs-task-runtime"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume_role.json
}

data "aws_iam_policy_document" "ecs_task_runtime" {
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

  # Required for KMS-encrypted S3 bucket: PutObject needs GenerateDataKey,
  # GetObject needs Decrypt.
  statement {
    effect = "Allow"
    actions = [
      "kms:GenerateDataKey",
      "kms:Decrypt"
    ]
    resources = [aws_kms_key.main.arn]
  }
}

resource "aws_iam_role_policy" "ecs_task_runtime" {
  name   = "${var.project_name}-ecs-task-runtime-policy"
  role   = aws_iam_role.ecs_task.id
  policy = data.aws_iam_policy_document.ecs_task_runtime.json
}

resource "aws_ecs_task_definition" "api" {
  family                   = "${var.project_name}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(var.api_cpu)
  memory                   = tostring(var.api_memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "api"
      image     = local.api_image
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "PORT", value = "8080" },
        { name = "DB_HOST", value = aws_db_instance.main.address },
        { name = "DB_PORT", value = "5432" },
        { name = "DB_NAME", value = "aegis" },
        { name = "DB_USER", value = var.db_master_username },
        { name = "DB_SSLMODE", value = "require" },
        { name = "STORAGE_MODE", value = "s3" },
        { name = "S3_BUCKET", value = aws_s3_bucket.dicom.bucket },
        { name = "S3_REGION", value = var.aws_region },
        { name = "API_BASE_URL", value = "https://${local.api_fqdn}" },
        { name = "APP_TIMEZONE", value = "UTC" },
        { name = "ALLOWED_ORIGINS", value = join(",", local.resolved_api_allowed_origins) },
        { name = "AUTH_ENABLED", value = "true" },
        { name = "AUTH_PROVIDER", value = "aws" },
        { name = "PIPELINE_AUTO", value = "true" },
        { name = "DEFACING_SERVICE_URL", value = "http://defacing.aegis.local:8080" },
        { name = "PHI_DETECTION_SERVICE_URL", value = "http://phi-detection.aegis.local:8080" },
        { name = "QC_SERVICE_URL", value = "http://qc-service.aegis.local:8080" },
        { name = "BIDS_SERVICE_URL", value = "http://bids-service.aegis.local:8080" },
        { name = "CLASSIFICATION_SERVICE_URL", value = "http://classification-service.aegis.local:8080" },
        { name = "PROTOCOL_SERVICE_URL", value = "http://protocol-service.aegis.local:8080" },
        { name = "SYNTH_SERVICE_URL", value = "http://synth-service.aegis.local:8080" },
        { name = "DIMSE_RECEIVER_URL", value = try("http://${aws_instance.dimse_receiver[0].private_ip}:8080", "") },
        { name = "FIRST_ADMIN_EMAIL", value = var.first_admin_email },
        { name = "SMTP_HOST", value = local.ses_smtp_hostname },
        { name = "SMTP_PORT", value = "587" },
        { name = "SMTP_FROM", value = var.smtp_from },
        { name = "SMTP_USERNAME", value = aws_iam_access_key.ses_smtp.id }
      ]
      secrets = [
        { name = "DB_PASSWORD", valueFrom = "${aws_db_instance.main.master_user_secret[0].secret_arn}:password::" },
        { name = "SMTP_PASSWORD", valueFrom = aws_secretsmanager_secret.smtp_password.arn }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.main.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "api"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "api" {
  name                   = "${var.project_name}-api"
  cluster                = aws_ecs_cluster.main.id
  task_definition        = aws_ecs_task_definition.api.arn
  desired_count          = var.api_desired_count
  launch_type            = "FARGATE"
  enable_execute_command = true

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  network_configuration {
    subnets          = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.api.arn
    container_name   = "api"
    container_port   = 8080
  }

  service_registries {
    registry_arn = aws_service_discovery_service.sidecars["api"].arn
  }

  depends_on = [aws_lb_listener.https]
}

resource "aws_ecs_task_definition" "admin" {
  family                   = "${var.project_name}-admin-dashboard"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(var.admin_cpu)
  memory                   = tostring(var.admin_memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "admin-dashboard"
      image     = local.admin_image
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        # nginx uses these at startup (envsubst) to configure backend proxies.
        # Use internal Cloud Map DNS for API so the Cognito JWT injected by the
        # admin ALB is forwarded intact — the public ALB strips X-Amzn-Oidc-Data
        # on a second hop, breaking auth middleware.
        { name = "API_URL", value = "http://api.aegis.local:8080" },
        # MCP server not yet deployed on AWS; route /agent/* to GCP MCP server as fallback.
        # Update to http://mcp-server.aegis.local:8080 once AWS MCP is deployed.
        { name = "MCP_SERVER_URL", value = "https://aegis-mcp-server-uk5cvzf5nq-uc.a.run.app" },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.main.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "admin"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "admin" {
  name                   = "${var.project_name}-admin-dashboard"
  cluster                = aws_ecs_cluster.main.id
  task_definition        = aws_ecs_task_definition.admin.arn
  desired_count          = var.admin_desired_count
  launch_type            = "FARGATE"
  enable_execute_command = true

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  network_configuration {
    subnets          = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.admin.arn
    container_name   = "admin-dashboard"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.https]
}

# --- Weasis DWV viewer ---

resource "aws_ecs_task_definition" "weasis" {
  family                   = "${var.project_name}-weasis"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(var.weasis_cpu)
  memory                   = tostring(var.weasis_memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "weasis"
      image     = local.weasis_image
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "API_URL", value = "https://${local.api_fqdn}" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.main.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "weasis"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "weasis" {
  name            = "${var.project_name}-weasis"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.weasis.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  network_configuration {
    subnets          = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.weasis.arn
    container_name   = "weasis"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.https]
}

# --- SNS + SQS (Event Notifications) ---

resource "aws_sns_topic" "dicom_ingest" {
  name              = "${var.project_name}-dicom-ingest"
  kms_master_key_id = aws_kms_key.main.id
  tags              = { Name = "${var.project_name}-dicom-ingest" }
}

resource "aws_sqs_queue" "dicom_ingest" {
  name              = "${var.project_name}-dicom-ingest"
  kms_master_key_id = aws_kms_key.main.id

  visibility_timeout_seconds = 300
  message_retention_seconds  = 86400

  tags = { Name = "${var.project_name}-dicom-ingest" }
}

resource "aws_sns_topic_subscription" "dicom_ingest_sqs" {
  topic_arn = aws_sns_topic.dicom_ingest.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.dicom_ingest.arn
}

# --- CloudWatch Log Group ---

resource "aws_cloudwatch_log_group" "main" {
  name              = "/ecs/${var.project_name}"
  retention_in_days = 30
  tags              = { Name = "${var.project_name}-logs" }
}

# --- Outputs ---

output "vpc_id" {
  value = aws_vpc.main.id
}

output "s3_bucket" {
  value = aws_s3_bucket.dicom.bucket
}

output "rds_endpoint" {
  value     = aws_db_instance.main.endpoint
  sensitive = true
}

output "rds_master_user_secret_arn" {
  value     = aws_db_instance.main.master_user_secret[0].secret_arn
  sensitive = true
}

output "ecs_cluster" {
  value = aws_ecs_cluster.main.name
}

output "ecs_api_service" {
  value = aws_ecs_service.api.name
}

output "ecs_api_task_definition" {
  value = aws_ecs_task_definition.api.arn
}

output "ecs_admin_service" {
  value = aws_ecs_service.admin.name
}

output "ecs_admin_task_definition" {
  value = aws_ecs_task_definition.admin.arn
}

output "alb_dns" {
  value = aws_lb.main.dns_name
}

output "api_base_url" {
  value = "https://${local.api_fqdn}"
}

output "api_healthz_url" {
  value = "https://${local.api_fqdn}/healthz"
}

output "admin_base_url" {
  value = "https://${local.admin_fqdn}/"
}

output "alb_https_listener_arn" {
  value = aws_lb_listener.https.arn
}

output "api_target_group_name" {
  value = aws_lb_target_group.api.name
}

output "api_target_group_arn" {
  value = aws_lb_target_group.api.arn
}

output "admin_target_group_name" {
  value = aws_lb_target_group.admin.name
}

output "admin_target_group_arn" {
  value = aws_lb_target_group.admin.arn
}

output "cognito_user_pool_id" {
  value = aws_cognito_user_pool.admin.id
}

output "cognito_user_pool_client_id" {
  value = aws_cognito_user_pool_client.admin.id
}

output "cognito_user_pool_domain" {
  value = aws_cognito_user_pool_domain.admin.domain
}

output "weasis_base_url" {
  value = "https://${local.weasis_fqdn}/"
}

output "ecr_repositories" {
  value = { for k, v in aws_ecr_repository.services : k => v.repository_url }
}

# ── SES outputs — DNS records required after apply ───────────────────────────

output "ses_domain_verification_token" {
  description = "Add TXT record: _amazonses.aegisimaging.ai → this value"
  value       = aws_ses_domain_identity.main.verification_token
}

output "ses_dkim_tokens" {
  description = "Add 3 CNAME records: <token>._domainkey.aegisimaging.ai → <token>.dkim.amazonses.com"
  value       = aws_ses_domain_dkim.main.dkim_tokens
}

output "ses_smtp_username" {
  description = "SES SMTP username (IAM access key ID)"
  value       = aws_iam_access_key.ses_smtp.id
}

output "nat_egress_ip" {
  description = "Static NAT Gateway EIP — use for SMTP allowlisting and firewall rules"
  value       = aws_eip.nat.public_ip
}

# ── Landing Page (optional) ──────────────────────────────────────────────────

locals {
  landing_enabled = var.landing_image_tag != ""
  landing_image   = local.landing_enabled ? "${aws_ecr_repository.services["landing"].repository_url}:${var.landing_image_tag}" : ""
  landing_fqdn    = var.landing_domain != "" ? var.landing_domain : aws_lb.main.dns_name
}

resource "aws_lb_target_group" "landing" {
  count = local.landing_enabled ? 1 : 0

  name        = "${var.project_name}-landing"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.main.id

  health_check {
    path                = "/healthz"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
    matcher             = "200-399"
  }

  tags = { Name = "${var.project_name}-landing-tg" }
}

resource "aws_ecs_task_definition" "landing" {
  count = local.landing_enabled ? 1 : 0

  family                   = "${var.project_name}-landing"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(var.landing_cpu)
  memory                   = tostring(var.landing_memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "landing"
      image     = local.landing_image
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "VITE_API_BASE_URL", value = "https://${local.api_fqdn}" },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.main.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "landing"
        }
      }
    }
  ])

  tags = {
    Name        = "${var.project_name}-landing"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_ecs_service" "landing" {
  count = local.landing_enabled ? 1 : 0

  name            = "${var.project_name}-landing"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.landing[0].arn
  desired_count   = 1
  launch_type     = "FARGATE"

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  network_configuration {
    subnets          = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.landing[0].arn
    container_name   = "landing"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.https]

  tags = {
    Name        = "${var.project_name}-landing"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Landing page uses host-based routing (priority 4, after api/admin/weasis subdomains).
# No Cognito auth — public-facing marketing site.
resource "aws_lb_listener_rule" "landing_subdomain" {
  count        = local.landing_enabled && var.landing_domain != "" ? 1 : 0
  listener_arn = aws_lb_listener.https.arn
  priority     = 4

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.landing[0].arn
  }

  condition {
    host_header {
      values = [var.landing_domain]
    }
  }
}

output "landing_base_url" {
  description = "Landing page URL (empty when landing is not deployed)"
  value       = local.landing_enabled ? "https://${local.landing_fqdn}/" : ""
}
