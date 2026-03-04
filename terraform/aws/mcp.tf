# ── AWS ECS — MCP Server ───────────────────────────────────────────────────────
#
# Deploys the AEGIS MCP server to ECS Fargate.
# LLM backend: Amazon Bedrock Claude 3.5 Haiku via ECS task IAM role —
# no API keys needed, mirrors GCP's Workload Identity approach.
#
# Service registers in aegis.local namespace so the admin dashboard nginx
# (MCP_SERVER_URL=http://mcp-server.aegis.local:8080) can reach it.

# ── Secrets Manager — AEGIS API token ─────────────────────────────────────────

resource "aws_secretsmanager_secret" "mcp_aegis_api_token" {
  name       = "${var.project_name}-mcp-aegis-api-token"
  kms_key_id = aws_kms_key.main.arn

  tags = {
    Name        = "${var.project_name}-mcp-aegis-api-token"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "mcp_aegis_api_token" {
  # Skip creating the version when no token is provided yet — set the value
  # directly in Secrets Manager after creating an API key in the dashboard.
  count = var.mcp_aegis_api_token != "" ? 1 : 0

  secret_id     = aws_secretsmanager_secret.mcp_aegis_api_token.id
  secret_string = var.mcp_aegis_api_token

  # CI/CD or manual Secrets Manager updates rotate the token without terraform apply.
  lifecycle {
    ignore_changes = [secret_string]
  }
}

# Allow the ECS task execution role to read this secret at container start.
resource "aws_iam_role_policy" "ecs_task_execution_mcp_secret" {
  name = "${var.project_name}-ecs-task-execution-mcp-secret"
  role = aws_iam_role.ecs_task_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["secretsmanager:GetSecretValue", "kms:Decrypt"]
      Resource = [
        aws_secretsmanager_secret.mcp_aegis_api_token.arn,
        aws_kms_key.main.arn
      ]
    }]
  })
}

# ── IAM — dedicated task role (Bedrock only, no S3) ───────────────────────────

resource "aws_iam_role" "mcp_server" {
  name               = "${var.project_name}-mcp-server"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume_role.json

  tags = {
    Name        = "${var.project_name}-mcp-server"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_iam_role_policy" "mcp_server_bedrock" {
  name = "${var.project_name}-mcp-server-bedrock"
  role = aws_iam_role.mcp_server.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "bedrock:InvokeModel",
          "bedrock:InvokeModelWithResponseStream"
        ]
        Resource = [
          # Foundation model ARNs (direct single-region access)
          "arn:aws:bedrock:*::foundation-model/anthropic.*",
          "arn:aws:bedrock:*::foundation-model/amazon.*",
          # Cross-region inference profiles (recommended for production resilience)
          "arn:aws:bedrock:${var.aws_region}:*:inference-profile/us.anthropic.*",
          "arn:aws:bedrock:${var.aws_region}:*:inference-profile/us.amazon.*"
        ]
      },
      {
        # Bedrock checks Marketplace subscription status at invocation time for
        # cross-region inference profiles. aws-marketplace actions require Resource "*".
        Effect   = "Allow"
        Action   = ["aws-marketplace:ViewSubscriptions", "aws-marketplace:Subscribe"]
        Resource = "*"
      }
    ]
  })
}

# ── ECS task definition ────────────────────────────────────────────────────────

resource "aws_ecs_task_definition" "mcp_server" {
  family                   = "${var.project_name}-mcp-server"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "512"
  memory                   = "1024"
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.mcp_server.arn

  container_definitions = jsonencode([
    {
      name      = "mcp-server"
      image     = "${aws_ecr_repository.services["mcp-server"].repository_url}:${var.mcp_image_tag}"
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "AEGIS_API_BASE_URL", value = "https://${local.api_fqdn}" },
        { name = "MCP_AGENT_HTTP_PORT", value = "8080" },
        { name = "MCP_AGENT_LLM_USE_AWS_BEDROCK", value = "true" },
        { name = "MCP_AGENT_LLM_AWS_REGION", value = var.aws_region },
        { name = "MCP_AGENT_LLM_MODEL", value = "us.anthropic.claude-3-5-haiku-20241022-v1:0" },
        { name = "MCP_AGENT_ALLOWED_ORIGIN", value = "https://${local.admin_fqdn}" },
        { name = "MCP_AGENT_REQUIRE_AUTH", value = "false" },
      ]
      secrets = [
        { name = "AEGIS_API_TOKEN", valueFrom = aws_secretsmanager_secret.mcp_aegis_api_token.arn }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.main.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "mcp-server"
        }
      }
    }
  ])

  tags = {
    Name        = "${var.project_name}-mcp-server"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── ECS service ────────────────────────────────────────────────────────────────

resource "aws_ecs_service" "mcp_server" {
  name                   = "${var.project_name}-mcp-server"
  cluster                = aws_ecs_cluster.main.id
  task_definition        = aws_ecs_task_definition.mcp_server.arn
  desired_count          = 1
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

  service_registries {
    registry_arn = aws_service_discovery_service.sidecars["mcp-server"].arn
  }

  tags = {
    Name        = "${var.project_name}-mcp-server"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
