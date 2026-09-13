# GitHub Actions OIDC role for AWS deployment workflow.
# This enables short-lived credentials in CI/CD instead of long-lived IAM user keys.

variable "github_actions_repo" {
  description = "GitHub repository in owner/name format allowed to assume deploy role"
  type        = string
  default     = "aegis-imaging/aegis"
}

variable "github_actions_deploy_role_name" {
  description = "IAM role name assumed by GitHub Actions deploy workflow"
  type        = string
  default     = "aegis-github-actions-deploy"
}

variable "enable_github_actions_oidc" {
  description = "Whether to provision GitHub Actions OIDC provider and deploy role"
  type        = bool
  default     = true
}

resource "aws_iam_openid_connect_provider" "github_actions" {
  count = var.enable_github_actions_oidc ? 1 : 0

  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}

data "aws_iam_policy_document" "github_actions_deploy_assume_role" {
  count = var.enable_github_actions_oidc ? 1 : 0

  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github_actions[0].arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_actions_repo}:ref:refs/heads/main",
        "repo:${var.github_actions_repo}:ref:refs/heads/develop",
        "repo:${var.github_actions_repo}:pull_request"
      ]
    }
  }
}

resource "aws_iam_role" "github_actions_deploy" {
  count = var.enable_github_actions_oidc ? 1 : 0

  name               = var.github_actions_deploy_role_name
  assume_role_policy = data.aws_iam_policy_document.github_actions_deploy_assume_role[0].json

  description = "Assumed by GitHub Actions for AEGIS AWS deploy automation"
}

data "aws_iam_policy_document" "github_actions_deploy_permissions" {
  count = var.enable_github_actions_oidc ? 1 : 0

  statement {
    sid = "ECRAuth"

    actions = [
      "ecr:GetAuthorizationToken"
    ]

    resources = ["*"]
  }

  statement {
    sid = "ECRPushPull"

    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:CompleteLayerUpload",
      "ecr:DescribeImages",
      "ecr:DescribeRepositories",
      "ecr:InitiateLayerUpload",
      "ecr:PutImage",
      "ecr:UploadLayerPart"
    ]

    resources = ["arn:aws:ecr:${var.aws_region}:*:repository/${var.project_name}/*"]
  }

  statement {
    sid = "ECSDeploy"

    actions = [
      "ecs:DescribeServices",
      "ecs:UpdateService"
    ]

    resources = ["*"]
  }

  statement {
    sid = "DIMSEReboot"

    actions = [
      "ec2:DescribeInstances",
      "ec2:RebootInstances"
    ]

    resources = ["*"]
  }

  statement {
    sid = "DIMSEImageParameter"

    actions = [
      "ssm:PutParameter"
    ]

    resources = ["arn:aws:ssm:${var.aws_region}:*:parameter/aegis/dimse-image"]
  }
}

resource "aws_iam_policy" "github_actions_deploy" {
  count = var.enable_github_actions_oidc ? 1 : 0

  name        = "${var.project_name}-github-actions-deploy"
  description = "Permissions for GitHub Actions deploy workflow"
  policy      = data.aws_iam_policy_document.github_actions_deploy_permissions[0].json
}

resource "aws_iam_role_policy_attachment" "github_actions_deploy" {
  count = var.enable_github_actions_oidc ? 1 : 0

  role       = aws_iam_role.github_actions_deploy[0].name
  policy_arn = aws_iam_policy.github_actions_deploy[0].arn
}

# ── Terraform role ────────────────────────────────────────────────────────────
#
# The deploy role above is scoped to ECR/ECS. Planning and applying this
# module needs far more (IAM, KMS, VPC, RDS, ALB, Cognito, WAF), so the
# terraform-aws.yml workflow assumes a separate administrator role with the
# same repository-scoped trust policy.

variable "github_actions_terraform_role_name" {
  description = "IAM role name assumed by the GitHub Actions terraform workflow (plan on PRs, apply on main)"
  type        = string
  default     = "aegis-github-actions-terraform"
}

resource "aws_iam_role" "github_actions_terraform" {
  count = var.enable_github_actions_oidc ? 1 : 0

  name               = var.github_actions_terraform_role_name
  assume_role_policy = data.aws_iam_policy_document.github_actions_deploy_assume_role[0].json

  description = "Assumed by GitHub Actions to plan and apply terraform/aws"
}

resource "aws_iam_role_policy_attachment" "github_actions_terraform_admin" {
  count = var.enable_github_actions_oidc ? 1 : 0

  role       = aws_iam_role.github_actions_terraform[0].name
  policy_arn = "arn:aws:iam::aws:policy/AdministratorAccess"
}

output "github_actions_terraform_role_arn" {
  description = "Value for the AWS_TERRAFORM_ROLE_ARN GitHub secret"
  value       = var.enable_github_actions_oidc ? aws_iam_role.github_actions_terraform[0].arn : null
}

output "github_actions_deploy_role_arn" {
  description = "IAM role ARN for GitHub Actions AWS deploy workflow OIDC authentication"
  value       = var.enable_github_actions_oidc ? aws_iam_role.github_actions_deploy[0].arn : null
}
