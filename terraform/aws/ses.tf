# ── Amazon SES — domain identity + SMTP credentials ──────────────────────────
#
# Provisions:
#   1. SES domain identity for aegisimaging.ai (provides DNS TXT verification token)
#   2. SES DKIM signing (provides 3 DNS CNAME records)
#   3. Dedicated IAM user with ses:SendRawEmail permission
#   4. IAM access key — Terraform auto-derives ses_smtp_password_v4
#   5. Secrets Manager secret storing the SMTP password (KMS-encrypted)
#
# ⚠ MANUAL STEPS REQUIRED AFTER APPLY (see outputs):
#   a) Add TXT + 3 CNAME DNS records to the aegisimaging.ai zone
#   b) Submit AWS Support case to exit SES sandbox (required to send to non-verified recipients)
# ─────────────────────────────────────────────────────────────────────────────

resource "aws_ses_domain_identity" "main" {
  domain = "aegisimaging.ai"
}

resource "aws_ses_domain_dkim" "main" {
  domain = aws_ses_domain_identity.main.domain
}

# Dedicated IAM user for SES SMTP authentication.
# SES SMTP requires a derived password — NOT the raw IAM secret access key.
# Terraform computes the correct V4 password automatically via ses_smtp_password_v4.
resource "aws_iam_user" "ses_smtp" {
  name = "${var.project_name}-ses-smtp"
  path = "/service/"
}

resource "aws_iam_user_policy" "ses_smtp" {
  name = "${var.project_name}-ses-send"
  user = aws_iam_user.ses_smtp.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ses:SendRawEmail"]
      Resource = "*"
    }]
  })
}

resource "aws_iam_access_key" "ses_smtp" {
  user = aws_iam_user.ses_smtp.name
}

# Store the derived SMTP password in Secrets Manager, encrypted with the
# project KMS key. The ECS task execution role reads this at task start.
resource "aws_secretsmanager_secret" "smtp_password" {
  name        = "${var.project_name}-ses-smtp-password"
  description = "SES SMTP password for AEGIS transactional email"
  kms_key_id  = aws_kms_key.main.arn
}

resource "aws_secretsmanager_secret_version" "smtp_password" {
  secret_id     = aws_secretsmanager_secret.smtp_password.id
  secret_string = aws_iam_access_key.ses_smtp.ses_smtp_password_v4
}
