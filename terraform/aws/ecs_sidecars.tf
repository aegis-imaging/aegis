# ── ECS Fargate — Python processing sidecars ──────────────────────────────────
#
# 7 sidecar services that run alongside the Go API. Each registers with
# AWS Cloud Map so the API can reach it at http://<name>.aegis.local:8080.
#
# All sidecars share the same IAM roles and ECS security group as the API.
# They access DICOM files directly via S3 (STORAGE_MODE=s3).

locals {
  sidecar_configs = {
    "defacing" = {
      cpu    = 512
      memory = 2048
      env = [
        { name = "DEFACE_TOOL", value = "auto" },
      ]
    }
    "phi-detection" = {
      cpu    = 256
      memory = 512
      env = [
        { name = "PHI_TOOL",                 value = "auto" },
        { name = "PHI_CONFIDENCE_THRESHOLD", value = "0.4" },
        { name = "PHI_MIN_TEXT_LENGTH",      value = "3" },
      ]
    }
    "qc-service" = {
      cpu    = 256
      memory = 512
      env = [
        { name = "QC_TOOL",          value = "auto" },
        { name = "QC_SNR_THRESHOLD", value = "10.0" },
        { name = "QC_GAP_RATIO",     value = "2.0" },
      ]
    }
    "bids-service" = {
      cpu    = 512
      memory = 1024
      env = [
        { name = "BIDS_TOOL", value = "auto" },
      ]
    }
    "classification-service" = {
      cpu    = 256
      memory = 512
      env = [
        { name = "CLASSIFY_TOOL",                 value = "auto" },
        { name = "CLASSIFY_CONFIDENCE_THRESHOLD", value = "0.5" },
      ]
    }
    "protocol-service" = {
      cpu    = 256
      memory = 512
      env = [
        { name = "PROTOCOL_TOOL",              value = "auto" },
        { name = "PROTOCOL_DEFAULT_TOLERANCE", value = "5.0" },
      ]
    }
    "synth-service" = {
      cpu    = 512
      memory = 1024
      env    = []
    }
  }

  # Common environment variables injected into every sidecar
  sidecar_common_env = [
    { name = "STORAGE_MODE",      value = "s3" },
    { name = "S3_BUCKET",         value = aws_s3_bucket.dicom.bucket },
    { name = "S3_REGION",         value = var.aws_region },
    { name = "LOCAL_STORAGE_DIR", value = "/app/data" },
    { name = "PORT",              value = "8080" },
  ]
}

# ── Task definitions ───────────────────────────────────────────────────────────

resource "aws_ecs_task_definition" "sidecar" {
  for_each = local.sidecar_configs

  family                   = "${var.project_name}-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(each.value.cpu)
  memory                   = tostring(each.value.memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = each.key
      image     = "${aws_ecr_repository.services[each.key].repository_url}:${var.sidecar_image_tag}"
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = concat(local.sidecar_common_env, each.value.env)
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.main.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = each.key
        }
      }
    }
  ])

  tags = {
    Name        = "${var.project_name}-${each.key}"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── ECS services ──────────────────────────────────────────────────────────────

resource "aws_ecs_service" "sidecar" {
  for_each = local.sidecar_configs

  name                   = "${var.project_name}-${each.key}"
  cluster                = aws_ecs_cluster.main.id
  task_definition        = aws_ecs_task_definition.sidecar[each.key].arn
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
    registry_arn = aws_service_discovery_service.sidecars[each.key].arn
  }

  tags = {
    Name        = "${var.project_name}-${each.key}"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
