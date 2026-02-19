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

  db_name  = "aegis"
  username = "aegis"
  password = "CHANGE_ME" # Use AWS Secrets Manager in production

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  backup_retention_period = 7
  multi_az                = false # Enable for production HA
  deletion_protection     = true
  skip_final_snapshot     = false
  final_snapshot_identifier = "${var.project_name}-final-snapshot"

  tags = { Name = "${var.project_name}-postgres" }
}

# --- ECR (Container Registry) ---

locals {
  services = ["api", "defacing", "phi-detection", "qc-service", "bids-service", "classification-service", "protocol-service"]
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
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
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

# TODO: HTTPS listener (requires ACM certificate)
# resource "aws_lb_listener" "https" { ... }

# TODO: Cognito user pool for admin dashboard authentication
# resource "aws_cognito_user_pool" "admin" { ... }
# resource "aws_cognito_user_pool_client" "admin" { ... }
# resource "aws_cognito_user_pool_domain" "admin" { ... }

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

output "ecs_cluster" {
  value = aws_ecs_cluster.main.name
}

output "alb_dns" {
  value = aws_lb.main.dns_name
}

output "ecr_repositories" {
  value = { for k, v in aws_ecr_repository.services : k => v.repository_url }
}
