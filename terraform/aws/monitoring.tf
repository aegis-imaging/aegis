# ── AWS Monitoring & Alerting ──────────────────────────────────────────────────
#
# Mirrors GCP's Cloud Monitoring baseline: SNS notification topic, CloudWatch
# alarms (API 5xx, ALB latency, RDS CPU/disk/connections, ECS memory),
# log-based metric filters (pipeline failures, stuck studies, pipeline
# dispatches), Route53 uptime health check, WAF v2 WebACL, and a CloudWatch
# dashboard for operational visibility.

# ── Variables ─────────────────────────────────────────────────────────────────

variable "alert_email" {
  description = "Email address for CloudWatch alarm notifications. Empty = no notifications."
  type        = string
  default     = ""
}

variable "enable_monitoring_alerts" {
  description = "Enable CloudWatch alarms and Route53 health checks."
  type        = bool
  default     = true
}

variable "waf_allowed_ip_ranges" {
  description = "IP CIDR ranges allowed through the WAF. Empty list = allow all (rate-limit only)."
  type        = list(string)
  default     = []
}

variable "waf_rate_limit" {
  description = "Maximum requests per 5-minute window per IP before WAF blocks."
  type        = number
  default     = 2000
}

# ── SNS Notification Topic ────────────────────────────────────────────────────

resource "aws_sns_topic" "alerts" {
  count = var.alert_email != "" ? 1 : 0

  name = "${var.project_name}-alerts"
  tags = { Name = "${var.project_name}-alerts" }
}

resource "aws_sns_topic_subscription" "alerts_email" {
  count = var.alert_email != "" ? 1 : 0

  topic_arn = aws_sns_topic.alerts[0].arn
  protocol  = "email"
  endpoint  = var.alert_email
}

locals {
  alarm_actions = var.alert_email != "" ? [aws_sns_topic.alerts[0].arn] : []
}

# ── WAF v2 WebACL (mirrors GCP Cloud Armor) ───────────────────────────────────

resource "aws_wafv2_web_acl" "api" {
  count = var.enable_monitoring_alerts ? 1 : 0

  name        = "${var.project_name}-api-waf"
  description = "Baseline WAF for AEGIS ALB - rate limiting and optional IP allowlisting"
  scope       = "REGIONAL"

  default_action {
    # If IP allowlist is configured, default deny; otherwise allow (rate-limit only).
    dynamic "allow" {
      for_each = length(var.waf_allowed_ip_ranges) == 0 ? [1] : []
      content {}
    }
    dynamic "block" {
      for_each = length(var.waf_allowed_ip_ranges) > 0 ? [1] : []
      content {}
    }
  }

  # Rule 1: Allow configured IP ranges (only when allowlist is set).
  dynamic "rule" {
    for_each = length(var.waf_allowed_ip_ranges) > 0 ? [1] : []
    content {
      name     = "allow-configured-ips"
      priority = 1

      action {
        allow {}
      }

      statement {
        ip_set_reference_statement {
          arn = aws_wafv2_ip_set.allowed[0].arn
        }
      }

      visibility_config {
        sampled_requests_enabled   = true
        cloudwatch_metrics_enabled = true
        metric_name                = "${var.project_name}-waf-allowed-ips"
      }
    }
  }

  # Rule 2: Rate limiting per IP (always active).
  rule {
    name     = "rate-limit"
    priority = 10

    action {
      block {}
    }

    statement {
      rate_based_statement {
        limit              = var.waf_rate_limit
        aggregate_key_type = "IP"
      }
    }

    visibility_config {
      sampled_requests_enabled   = true
      cloudwatch_metrics_enabled = true
      metric_name                = "${var.project_name}-waf-rate-limit"
    }
  }

  # Rule 3: Allow STOW-RS paths before managed rules.
  # DICOMweb STOW-RS receives large multipart/related DICOM bodies that exceed the
  # 8 KB body-size limit enforced by AWSManagedRulesCommonRuleSet SizeRestrictions_BODY.
  # This ALLOW rule short-circuits WAF inspection for /api/stow* so binary DICOM
  # payloads are not blocked. The Go API enforces its own API-key auth on these routes.
  rule {
    name     = "allow-stow-rs"
    priority = 15

    action {
      allow {}
    }

    statement {
      byte_match_statement {
        search_string = "/api/stow"
        field_to_match {
          uri_path {}
        }
        text_transformation {
          priority = 0
          type     = "NONE"
        }
        positional_constraint = "STARTS_WITH"
      }
    }

    visibility_config {
      sampled_requests_enabled   = true
      cloudwatch_metrics_enabled = true
      metric_name                = "${var.project_name}-waf-stow-allow"
    }
  }

  # Rule 4: AWS Managed — common exploits (SQLi, XSS, etc.).
  rule {
    name     = "aws-managed-common"
    priority = 20

    override_action {
      none {}
    }

    statement {
      managed_rule_group_statement {
        vendor_name = "AWS"
        name        = "AWSManagedRulesCommonRuleSet"
      }
    }

    visibility_config {
      sampled_requests_enabled   = true
      cloudwatch_metrics_enabled = true
      metric_name                = "${var.project_name}-waf-common-rules"
    }
  }

  visibility_config {
    sampled_requests_enabled   = true
    cloudwatch_metrics_enabled = true
    metric_name                = "${var.project_name}-waf"
  }

  tags = {
    Name        = "${var.project_name}-api-waf"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_wafv2_ip_set" "allowed" {
  count = length(var.waf_allowed_ip_ranges) > 0 ? 1 : 0

  name               = "${var.project_name}-allowed-ips"
  description        = "IP ranges allowed through the AEGIS WAF"
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = var.waf_allowed_ip_ranges

  tags = {
    Name        = "${var.project_name}-allowed-ips"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_wafv2_web_acl_association" "alb" {
  count = var.enable_monitoring_alerts ? 1 : 0

  resource_arn = aws_lb.main.arn
  web_acl_arn  = aws_wafv2_web_acl.api[0].arn
}

# ── Route53 Health Check (mirrors GCP uptime check) ──────────────────────────

resource "aws_route53_health_check" "api_healthz" {
  count = var.api_domain != "" && var.enable_monitoring_alerts ? 1 : 0

  fqdn              = var.api_domain
  port              = 443
  type              = "HTTPS"
  resource_path     = "/healthz"
  request_interval  = 30
  failure_threshold = 3
  measure_latency   = true
  enable_sni        = true

  tags = {
    Name        = "${var.project_name}-api-healthz"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_metric_alarm" "api_uptime" {
  count = var.api_domain != "" && var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-api-uptime-${var.environment}"
  alarm_description = "API /healthz endpoint is not responding. Check ECS service health, DB connectivity, and ALB configuration."

  namespace           = "AWS/Route53"
  metric_name         = "HealthCheckStatus"
  statistic           = "Minimum"
  period              = 60
  evaluation_periods  = 2
  threshold           = 1
  comparison_operator = "LessThanThreshold"
  treat_missing_data  = "breaching"

  dimensions = {
    HealthCheckId = aws_route53_health_check.api_healthz[0].id
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "api"
    Severity = "critical"
  }
}

# ── ALB 5xx Alarm (mirrors GCP api_5xx_rate) ──────────────────────────────────

resource "aws_cloudwatch_metric_alarm" "alb_5xx" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-alb-5xx-${var.environment}"
  alarm_description = "ALB is returning elevated 5xx responses. Check ECS tasks, DB connectivity, and sidecar health."

  namespace           = "AWS/ApplicationELB"
  metric_name         = "HTTPCode_Target_5XX_Count"
  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 5
  threshold           = 10
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = aws_lb.main.arn_suffix
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "api"
    Severity = "warning"
  }
}

# ── ALB Latency Alarm (mirrors GCP api_latency p99 > 5s) ─────────────────────

resource "aws_cloudwatch_metric_alarm" "alb_latency" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-alb-latency-p99-${var.environment}"
  alarm_description = "API p99 latency exceeded 5 seconds. Check for slow DB queries, sidecar timeouts, or cold start spikes."

  namespace           = "AWS/ApplicationELB"
  metric_name         = "TargetResponseTime"
  extended_statistic  = "p99"
  period              = 60
  evaluation_periods  = 5
  threshold           = 5
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = aws_lb.main.arn_suffix
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "api"
    Severity = "warning"
  }
}

# ── RDS CPU Alarm (mirrors GCP cloudsql_cpu > 80% for 10m) ────────────────────

resource "aws_cloudwatch_metric_alarm" "rds_cpu" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-rds-cpu-${var.environment}"
  alarm_description = "RDS CPU remained high for 10 minutes. Review DB load and query patterns."

  namespace           = "AWS/RDS"
  metric_name         = "CPUUtilization"
  statistic           = "Average"
  period              = 60
  evaluation_periods  = 10
  threshold           = 80
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DBInstanceIdentifier = aws_db_instance.main.identifier
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "postgres"
    Severity = "warning"
  }
}

# ── RDS Disk Alarm (mirrors GCP cloudsql_disk > 85%) ──────────────────────────

resource "aws_cloudwatch_metric_alarm" "rds_disk" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-rds-disk-${var.environment}"
  alarm_description = "RDS free storage is below 3 GB. Enable storage auto-scaling or increase allocated_storage in Terraform."

  namespace           = "AWS/RDS"
  metric_name         = "FreeStorageSpace"
  statistic           = "Minimum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 3000000000 # 3 GB in bytes
  comparison_operator = "LessThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DBInstanceIdentifier = aws_db_instance.main.identifier
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "postgres"
    Severity = "critical"
  }
}

# ── RDS Connections Alarm (mirrors GCP cloudsql_connections > 80) ──────────────

resource "aws_cloudwatch_metric_alarm" "rds_connections" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-rds-connections-${var.environment}"
  alarm_description = "Active PostgreSQL connections are near the limit. Review ECS task count, enable PgBouncer, or increase max_connections."

  namespace           = "AWS/RDS"
  metric_name         = "DatabaseConnections"
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 5
  threshold           = 80
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DBInstanceIdentifier = aws_db_instance.main.identifier
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "postgres"
    Severity = "warning"
  }
}

# ── ECS Memory Alarm (mirrors GCP cloud_run_memory > 90%) ────────────────────

resource "aws_cloudwatch_metric_alarm" "ecs_api_memory" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-ecs-api-memory-${var.environment}"
  alarm_description = "API container memory utilisation is above 90%. Review for memory leaks, large DICOM uploads, or undersized task memory."

  namespace           = "AWS/ECS"
  metric_name         = "MemoryUtilization"
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 10
  threshold           = 90
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ClusterName = aws_ecs_cluster.main.name
    ServiceName = aws_ecs_service.api.name
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "api"
    Severity = "warning"
  }
}

# ── ECS CPU Alarm ─────────────────────────────────────────────────────────────

resource "aws_cloudwatch_metric_alarm" "ecs_api_cpu" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-ecs-api-cpu-${var.environment}"
  alarm_description = "API ECS task CPU utilisation is above 80% for 10 minutes."

  namespace           = "AWS/ECS"
  metric_name         = "CPUUtilization"
  statistic           = "Average"
  period              = 60
  evaluation_periods  = 10
  threshold           = 80
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ClusterName = aws_ecs_cluster.main.name
    ServiceName = aws_ecs_service.api.name
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "api"
    Severity = "warning"
  }
}

# ── CloudWatch Log Metric Filters (mirrors GCP log-based metrics) ─────────────

resource "aws_cloudwatch_log_metric_filter" "pipeline_failures" {
  name           = "${var.project_name}-pipeline-failures"
  log_group_name = aws_cloudwatch_log_group.main.name
  pattern        = "\"pipeline: send failure alert\""

  metric_transformation {
    name          = "PipelineFailures"
    namespace     = "AEGIS/${var.environment}"
    value         = "1"
    default_value = "0"
  }
}

resource "aws_cloudwatch_log_metric_filter" "study_stuck" {
  name           = "${var.project_name}-study-stuck"
  log_group_name = aws_cloudwatch_log_group.main.name
  pattern        = "\"sla: sent alert\""

  metric_transformation {
    name          = "StudyStuck"
    namespace     = "AEGIS/${var.environment}"
    value         = "1"
    default_value = "0"
  }
}

resource "aws_cloudwatch_log_metric_filter" "pipeline_dispatches" {
  name           = "${var.project_name}-pipeline-dispatches"
  log_group_name = aws_cloudwatch_log_group.main.name
  pattern        = "\"pipeline: dispatching\""

  metric_transformation {
    name          = "PipelineDispatches"
    namespace     = "AEGIS/${var.environment}"
    value         = "1"
    default_value = "0"
  }
}

resource "aws_cloudwatch_log_metric_filter" "destination_probe_failures" {
  name           = "${var.project_name}-destination-probe-failures"
  log_group_name = aws_cloudwatch_log_group.main.name
  pattern        = "\"destination.tested\" \"success\":false"

  metric_transformation {
    name          = "DestinationProbeFailures"
    namespace     = "AEGIS/${var.environment}"
    value         = "1"
    default_value = "0"
  }
}

resource "aws_cloudwatch_log_metric_filter" "dimse_dead_letter" {
  name           = "${var.project_name}-dimse-dead-letter"
  log_group_name = aws_cloudwatch_log_group.main.name
  pattern        = "\"dead-letter\""

  metric_transformation {
    name          = "DimseDeadLetter"
    namespace     = "AEGIS/${var.environment}"
    value         = "1"
    default_value = "0"
  }
}

# ── Alarms on log-based metrics ───────────────────────────────────────────────

resource "aws_cloudwatch_metric_alarm" "pipeline_failure_alert" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-pipeline-failures-${var.environment}"
  alarm_description = "Pipeline step failures are elevated. Check ECS logs for the API service. Review sidecar health endpoints and study audit trails."

  namespace           = "AEGIS/${var.environment}"
  metric_name         = "PipelineFailures"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 3
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "pipeline"
    Severity = "critical"
  }
}

resource "aws_cloudwatch_metric_alarm" "study_stuck_alert" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-study-stuck-${var.environment}"
  alarm_description = "One or more studies have been stuck in a pipeline stage for longer than the SLA threshold. Check GET /api/studies/stuck for details."

  namespace           = "AEGIS/${var.environment}"
  metric_name         = "StudyStuck"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "pipeline"
    Severity = "warning"
  }
}

resource "aws_cloudwatch_metric_alarm" "destination_probe_failure_alert" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-destination-probe-failures-${var.environment}"
  alarm_description = "Destination connectivity probes are failing. Check destination endpoint reachability and routing configuration."

  namespace           = "AEGIS/${var.environment}"
  metric_name         = "DestinationProbeFailures"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "routing"
    Severity = "warning"
  }
}

resource "aws_cloudwatch_metric_alarm" "dimse_dead_letter_alert" {
  count = var.enable_monitoring_alerts ? 1 : 0

  alarm_name        = "${var.project_name}-dimse-dead-letter-${var.environment}"
  alarm_description = "DIMSE dead-letter indicators detected in logs. Review DIMSE retry/dead-letter queues and ingest connectivity."

  namespace           = "AEGIS/${var.environment}"
  metric_name         = "DimseDeadLetter"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions

  tags = {
    Service  = "dimse"
    Severity = "critical"
  }
}

# ── CloudWatch Dashboard (mirrors GCP monitoring dashboard) ───────────────────

resource "aws_cloudwatch_dashboard" "aegis" {
  count = var.enable_monitoring_alerts ? 1 : 0

  dashboard_name = "${var.project_name}-${var.environment}"

  dashboard_body = jsonencode({
    widgets = [
      # Row 1: ALB overview
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6
        properties = {
          title  = "ALB Request Count"
          region = var.aws_region
          metrics = [
            ["AWS/ApplicationELB", "RequestCount", "LoadBalancer", aws_lb.main.arn_suffix, { stat = "Sum", period = 60 }],
            ["AWS/ApplicationELB", "HTTPCode_Target_5XX_Count", "LoadBalancer", aws_lb.main.arn_suffix, { stat = "Sum", period = 60, color = "#d62728" }],
            ["AWS/ApplicationELB", "HTTPCode_Target_4XX_Count", "LoadBalancer", aws_lb.main.arn_suffix, { stat = "Sum", period = 60, color = "#ff7f0e" }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 12
        height = 6
        properties = {
          title  = "ALB Latency (p50 / p99)"
          region = var.aws_region
          metrics = [
            ["AWS/ApplicationELB", "TargetResponseTime", "LoadBalancer", aws_lb.main.arn_suffix, { stat = "p50", period = 60 }],
            ["AWS/ApplicationELB", "TargetResponseTime", "LoadBalancer", aws_lb.main.arn_suffix, { stat = "p99", period = 60, color = "#d62728" }],
          ]
          view = "timeSeries"
        }
      },
      # Row 2: ECS API
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 12
        height = 6
        properties = {
          title  = "ECS API — CPU & Memory"
          region = var.aws_region
          metrics = [
            ["AWS/ECS", "CPUUtilization", "ClusterName", aws_ecs_cluster.main.name, "ServiceName", aws_ecs_service.api.name, { stat = "Average", period = 60 }],
            ["AWS/ECS", "MemoryUtilization", "ClusterName", aws_ecs_cluster.main.name, "ServiceName", aws_ecs_service.api.name, { stat = "Average", period = 60, color = "#ff7f0e" }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 6
        width  = 12
        height = 6
        properties = {
          title  = "ECS Running Tasks"
          region = var.aws_region
          metrics = [
            ["ECS/ContainerInsights", "RunningTaskCount", "ClusterName", aws_ecs_cluster.main.name, "ServiceName", aws_ecs_service.api.name, { stat = "Average", period = 60 }],
            ["ECS/ContainerInsights", "RunningTaskCount", "ClusterName", aws_ecs_cluster.main.name, "ServiceName", aws_ecs_service.admin.name, { stat = "Average", period = 60, color = "#2ca02c" }],
          ]
          view = "timeSeries"
        }
      },
      # Row 3: RDS
      {
        type   = "metric"
        x      = 0
        y      = 12
        width  = 8
        height = 6
        properties = {
          title  = "RDS CPU Utilisation"
          region = var.aws_region
          metrics = [
            ["AWS/RDS", "CPUUtilization", "DBInstanceIdentifier", aws_db_instance.main.identifier, { stat = "Average", period = 60 }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 8
        y      = 12
        width  = 8
        height = 6
        properties = {
          title  = "RDS Connections"
          region = var.aws_region
          metrics = [
            ["AWS/RDS", "DatabaseConnections", "DBInstanceIdentifier", aws_db_instance.main.identifier, { stat = "Maximum", period = 60 }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 16
        y      = 12
        width  = 8
        height = 6
        properties = {
          title  = "RDS Free Storage (GB)"
          region = var.aws_region
          metrics = [
            ["AWS/RDS", "FreeStorageSpace", "DBInstanceIdentifier", aws_db_instance.main.identifier, { stat = "Minimum", period = 300 }],
          ]
          view  = "timeSeries"
          yAxis = { left = { label = "Bytes" } }
        }
      },
      # Row 4: Pipeline metrics
      {
        type   = "metric"
        x      = 0
        y      = 18
        width  = 8
        height = 6
        properties = {
          title  = "Pipeline Dispatches"
          region = var.aws_region
          metrics = [
            ["AEGIS/${var.environment}", "PipelineDispatches", { stat = "Sum", period = 300 }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 8
        y      = 18
        width  = 8
        height = 6
        properties = {
          title  = "Pipeline Failures"
          region = var.aws_region
          metrics = [
            ["AEGIS/${var.environment}", "PipelineFailures", { stat = "Sum", period = 300, color = "#d62728" }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 16
        y      = 18
        width  = 8
        height = 6
        properties = {
          title  = "Stuck Study Alerts"
          region = var.aws_region
          metrics = [
            ["AEGIS/${var.environment}", "StudyStuck", { stat = "Sum", period = 300, color = "#ff7f0e" }],
          ]
          view = "timeSeries"
        }
      },
      # Row 5: WAF
      {
        type   = "metric"
        x      = 0
        y      = 24
        width  = 12
        height = 6
        properties = {
          title  = "WAF Allowed vs Blocked Requests"
          region = var.aws_region
          metrics = [
            ["AWS/WAFV2", "AllowedRequests", "WebACL", "${var.project_name}-api-waf", "Rule", "ALL", { stat = "Sum", period = 300 }],
            ["AWS/WAFV2", "BlockedRequests", "WebACL", "${var.project_name}-api-waf", "Rule", "ALL", { stat = "Sum", period = 300, color = "#d62728" }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 24
        width  = 12
        height = 6
        properties = {
          title  = "WAF Rate-Limited Requests"
          region = var.aws_region
          metrics = [
            ["AWS/WAFV2", "BlockedRequests", "WebACL", "${var.project_name}-api-waf", "Rule", "rate-limit", { stat = "Sum", period = 300, color = "#d62728" }],
          ]
          view = "timeSeries"
        }
      },
    ]
  })
}

# ── Outputs ───────────────────────────────────────────────────────────────────

output "waf_web_acl_arn" {
  description = "ARN of the WAF WebACL attached to the ALB"
  value       = var.enable_monitoring_alerts ? aws_wafv2_web_acl.api[0].arn : null
}

output "sns_alerts_topic_arn" {
  description = "ARN of the SNS topic for CloudWatch alarm notifications"
  value       = var.alert_email != "" ? aws_sns_topic.alerts[0].arn : null
}

output "cloudwatch_dashboard_name" {
  description = "Name of the CloudWatch monitoring dashboard"
  value       = var.enable_monitoring_alerts ? aws_cloudwatch_dashboard.aegis[0].dashboard_name : null
}
