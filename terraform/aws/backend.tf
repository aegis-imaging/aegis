# ── Terraform S3 backend + state bootstrap resources ──────────────────────────
#
# BOOTSTRAP (one-time, before `terraform init`):
#
#   aws s3 mb s3://aegis-prod-terraform-state --region us-east-1
#   aws s3api put-bucket-versioning \
#     --bucket aegis-prod-terraform-state \
#     --versioning-configuration Status=Enabled
#   aws dynamodb create-table \
#     --table-name aegis-terraform-locks \
#     --attribute-definitions AttributeName=LockID,AttributeType=S \
#     --key-schema AttributeName=LockID,KeyType=HASH \
#     --billing-mode PAY_PER_REQUEST \
#     --region us-east-1
#
# Then: terraform init
#
# NOTE: The S3 bucket and DynamoDB table below are managed by Terraform AFTER the
# bootstrap. They enforce encryption + versioning declaratively.

terraform {
  backend "s3" {
    bucket         = "aegis-prod-terraform-state"
    key            = "aws/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "aegis-terraform-locks"
  }
}

# ── State bucket (managed declaratively after bootstrap) ──────────────────────

resource "aws_s3_bucket" "tf_state" {
  bucket        = "aegis-prod-terraform-state"
  force_destroy = true  # Allow destroy even with state objects (teardown)

  # lifecycle {
  #   prevent_destroy = true  # Disabled for teardown
  # }

  tags = {
    Name        = "${var.project_name}-tf-state"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_s3_bucket_versioning" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "tf_state" {
  bucket                  = aws_s3_bucket.tf_state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# ── State lock table ───────────────────────────────────────────────────────────

resource "aws_dynamodb_table" "tf_locks" {
  name         = "aegis-terraform-locks"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  # lifecycle {
  #   prevent_destroy = true  # Disabled for teardown
  # }

  tags = {
    Name        = "${var.project_name}-tf-locks"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
