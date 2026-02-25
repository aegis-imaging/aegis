# AEGIS — Azure Monitor Alerts
#
# 9 metric alerts mirroring the GCP Cloud Monitoring + AWS CloudWatch policies.
# All alerts route to an action group that emails var.alert_email.

resource "azurerm_monitor_action_group" "alerts" {
  count               = var.alert_email != "" ? 1 : 0
  name                = "${local.prefix}-alerts"
  resource_group_name = azurerm_resource_group.main.name
  short_name          = "aegis"

  email_receiver {
    name                    = "ops"
    email_address           = var.alert_email
    use_common_alert_schema = true
  }

  tags = local.tags
}

locals {
  action_group_ids = var.alert_email != "" ? [azurerm_monitor_action_group.alerts[0].id] : []
}

# ── API 5xx Error Rate ────────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "api_5xx" {
  name                = "${local.prefix}-api-5xx-rate"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.api.id]
  description         = "API HTTP 5xx error rate > 1% over 15 minutes"
  severity            = 2
  frequency           = "PT5M"
  window_size         = "PT15M"

  criteria {
    metric_namespace = "Microsoft.App/containerapps"
    metric_name      = "Requests"
    aggregation      = "Total"
    operator         = "GreaterThan"
    threshold        = 10

    dimension {
      name     = "StatusCodeCategory"
      operator = "Include"
      values   = ["5xx"]
    }
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── API Request Latency ───────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "api_latency" {
  name                = "${local.prefix}-api-latency"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.api.id]
  description         = "API P99 request latency > 2 seconds over 15 minutes"
  severity            = 3
  frequency           = "PT5M"
  window_size         = "PT15M"

  criteria {
    metric_namespace = "Microsoft.App/containerapps"
    metric_name      = "RespondTime"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 2000
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── API CPU Usage ─────────────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "api_cpu" {
  name                = "${local.prefix}-api-cpu"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.api.id]
  description         = "API Container App CPU usage > 80% over 10 minutes"
  severity            = 3
  frequency           = "PT5M"
  window_size         = "PT15M"

  criteria {
    metric_namespace = "Microsoft.App/containerapps"
    metric_name      = "UsageNanoCores"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 800000000 # 800m cores (80% of 1 vCPU)
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── Database CPU ──────────────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "db_cpu" {
  name                = "${local.prefix}-db-cpu"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_postgresql_flexible_server.main.id]
  description         = "PostgreSQL CPU usage > 80% over 10 minutes"
  severity            = 2
  frequency           = "PT5M"
  window_size         = "PT15M"

  criteria {
    metric_namespace = "Microsoft.DBforPostgreSQL/flexibleServers"
    metric_name      = "cpu_percent"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 80
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── Database Storage ──────────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "db_storage" {
  name                = "${local.prefix}-db-storage"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_postgresql_flexible_server.main.id]
  description         = "PostgreSQL storage used > 80% of allocated"
  severity            = 2
  frequency           = "PT15M"
  window_size         = "PT1H"

  criteria {
    metric_namespace = "Microsoft.DBforPostgreSQL/flexibleServers"
    metric_name      = "storage_percent"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 80
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── Database Connections ──────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "db_connections" {
  name                = "${local.prefix}-db-connections"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_postgresql_flexible_server.main.id]
  description         = "PostgreSQL active connections > 80 over 5 minutes"
  severity            = 3
  frequency           = "PT5M"
  window_size         = "PT5M"

  criteria {
    metric_namespace = "Microsoft.DBforPostgreSQL/flexibleServers"
    metric_name      = "active_connections"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 80
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── API Memory ────────────────────────────────────────────────────────────────

resource "azurerm_monitor_metric_alert" "api_memory" {
  name                = "${local.prefix}-api-memory"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.api.id]
  description         = "API Container App memory usage > 80% of limit over 10 minutes"
  severity            = 3
  frequency           = "PT5M"
  window_size         = "PT15M"

  criteria {
    metric_namespace = "Microsoft.App/containerapps"
    metric_name      = "WorkingSetBytes"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 1717986918 # ~80% of 2Gi
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_group_id = action.value
    }
  }

  tags = local.tags
}

# ── Pipeline Failures (Log-based) ─────────────────────────────────────────────
# Queries Application Insights for "pipeline step failed" log entries.

resource "azurerm_monitor_scheduled_query_rules_alert_v2" "pipeline_failures" {
  name                = "${local.prefix}-pipeline-failures"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  description         = "One or more pipeline processing steps failed in the last 5 minutes"
  severity            = 2
  evaluation_frequency = "PT5M"
  window_duration      = "PT5M"

  scopes = [azurerm_log_analytics_workspace.main.id]

  criteria {
    query = <<-EOQ
      ContainerAppConsoleLogs_CL
      | where ContainerAppName_s contains "aegis-${var.environment}-api"
      | where Log_s contains "pipeline step failed"
      | summarize count() by bin(TimeGenerated, 5m)
      | where count_ > 0
    EOQ

    time_aggregation_method = "Count"
    threshold               = 0
    operator                = "GreaterThan"

    failing_periods {
      minimum_failing_periods_to_trigger_alert = 1
      number_of_evaluation_periods             = 1
    }
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_groups = [action.value]
    }
  }

  tags = local.tags
}

# ── Stuck Studies (Log-based) ─────────────────────────────────────────────────

resource "azurerm_monitor_scheduled_query_rules_alert_v2" "stuck_studies" {
  name                = "${local.prefix}-stuck-studies"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  description         = "Studies have been stuck in the pipeline beyond the SLA threshold"
  severity            = 2
  evaluation_frequency = "PT15M"
  window_duration      = "PT15M"

  scopes = [azurerm_log_analytics_workspace.main.id]

  criteria {
    query = <<-EOQ
      ContainerAppConsoleLogs_CL
      | where ContainerAppName_s contains "aegis-${var.environment}-api"
      | where Log_s contains "study.stuck"
      | summarize count() by bin(TimeGenerated, 15m)
      | where count_ > 0
    EOQ

    time_aggregation_method = "Count"
    threshold               = 0
    operator                = "GreaterThan"

    failing_periods {
      minimum_failing_periods_to_trigger_alert = 1
      number_of_evaluation_periods             = 1
    }
  }

  dynamic "action" {
    for_each = local.action_group_ids
    content {
      action_groups = [action.value]
    }
  }

  tags = local.tags
}
