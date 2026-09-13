# AEGIS — Azure Container Apps
#
# Defines all 14 Container App services:
#   - api (Go API)
#   - admin-dashboard, landing, dwv, mcp-server (frontends / tooling)
#   - defacing, phi-detection, qc-service, bids-service,
#     classification-service, protocol-service, synth-service,
#     analytics-service, sct-service (Python sidecars)
#
# All apps share the same Container Apps Environment and user-assigned
# managed identity (ACR pull + Key Vault + Blob Storage access).
#
# Internal service-to-service URLs use the ACA internal DNS pattern:
#   http://<app-name>.internal.<env-default-domain>
#
# The Go API's *_SERVICE_URL env vars are set to these internal addresses.

locals {
  acr_server         = azurerm_container_registry.main.login_server
  aca_env_id         = azurerm_container_app_environment.main.id
  identity_id        = azurerm_user_assigned_identity.aca_workload.id
  identity_client_id = azurerm_user_assigned_identity.aca_workload.client_id
  storage_account    = azurerm_storage_account.dicom.name
  storage_container  = azurerm_storage_container.dicom.name

  # Internal ACA DNS suffix for service discovery
  aca_internal_domain = azurerm_container_app_environment.main.default_domain

  # Placeholder image for initial terraform provisioning before CI/CD pushes real images.
  # Use: terraform apply -var 'api_image_tag=placeholder'  (first apply only)
  # GitHub Actions (deploy-azure.yml) updates images via `az containerapp update`.
  # lifecycle { ignore_changes = [template[0].container[0].image] } on each app
  # prevents terraform from reverting after CI/CD deploys.
  placeholder_image     = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest"
  use_placeholder_image = var.api_image_tag == "placeholder"

  # Internal URLs the API uses to reach each sidecar. Only injected when the
  # sidecars are deployed, so the API skips those pipeline steps otherwise.
  sidecar_service_urls = {
    DEFACING_SERVICE_URL       = "https://${local.prefix}-defacing.internal.${local.aca_internal_domain}"
    PHI_DETECTION_SERVICE_URL  = "https://${local.prefix}-phi-detection.internal.${local.aca_internal_domain}"
    QC_SERVICE_URL             = "https://${local.prefix}-qc-service.internal.${local.aca_internal_domain}"
    BIDS_SERVICE_URL           = "https://${local.prefix}-bids-service.internal.${local.aca_internal_domain}"
    CLASSIFICATION_SERVICE_URL = "https://${local.prefix}-classify.internal.${local.aca_internal_domain}"
    PROTOCOL_SERVICE_URL       = "https://${local.prefix}-protocol-service.internal.${local.aca_internal_domain}"
    SYNTH_SERVICE_URL          = "https://${local.prefix}-synth-service.internal.${local.aca_internal_domain}"
    ANALYTICS_SERVICE_URL      = "https://${local.prefix}-analytics-service.internal.${local.aca_internal_domain}"
    SCT_SERVICE_URL            = "https://${local.prefix}-sct-service.internal.${local.aca_internal_domain}"
  }

  # Common env vars injected into every sidecar
  sidecar_common_env = [
    {
      name  = "STORAGE_MODE"
      value = "azure"
    },
    {
      name  = "AZURE_CLIENT_ID"
      value = azurerm_user_assigned_identity.aca_workload.client_id
    },
    {
      name  = "AZURE_STORAGE_ACCOUNT"
      value = azurerm_storage_account.dicom.name
    },
    {
      name  = "AZURE_STORAGE_CONTAINER"
      value = "dicom"
    },
    {
      name  = "LOCAL_STORAGE_DIR"
      value = "/app/data"
    },
    {
      name  = "PORT"
      value = "8080"
    },
  ]
}

# ── Go API ────────────────────────────────────────────────────────────────────

resource "azurerm_container_app" "api" {
  name                         = "${local.prefix}-api"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  # DATABASE_URL stored in Key Vault — not visible as plaintext in portal.
  secret {
    name                = "database-url"
    key_vault_secret_id = azurerm_key_vault_secret.database_url.id
    identity            = local.identity_id
  }


  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = var.api_min_replicas
    max_replicas = var.api_max_replicas

    http_scale_rule {
      name                = "http-scale"
      concurrent_requests = "100"
    }

    container {
      name   = "api"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/api:${var.api_image_tag}"
      cpu    = var.api_cpu
      memory = var.api_memory

      env {
        name  = "PORT"
        value = "8080"
      }
      # DATABASE_URL via Key Vault secret reference — never exposed as plaintext.
      env {
        name        = "DATABASE_URL"
        secret_name = "database-url"
      }
      env {
        name  = "STORAGE_MODE"
        value = "azure"
      }
      env {
        name  = "AZURE_CLIENT_ID"
        value = local.identity_client_id
      }
      env {
        name  = "AZURE_STORAGE_ACCOUNT"
        value = local.storage_account
      }
      env {
        name  = "AZURE_STORAGE_CONTAINER"
        value = "dicom"
      }
      env {
        name  = "AUTH_ENABLED"
        value = "true"
      }
      env {
        name  = "AUTH_PROVIDER"
        value = "azure"
      }
      env {
        name  = "PIPELINE_AUTO"
        value = "true"
      }
      env {
        name  = "FIRST_ADMIN_EMAIL"
        value = var.first_admin_email
      }
      # Sidecar service URLs via internal ACA DNS
      dynamic "env" {
        for_each = { for name, url in local.sidecar_service_urls : name => url if var.enable_sidecars }
        content {
          name  = env.key
          value = env.value
        }
      }
      dynamic "env" {
        for_each = local.dimse_enabled ? [azurerm_network_interface.dimse[0].private_ip_address] : []
        content {
          name  = "DIMSE_RECEIVER_URL"
          value = "http://${env.value}:8080"
        }
      }
      # ── Email / SMTP (Azure Communication Services) ──────────────────────────
      # smtp.azurecomm.net:587 — blank SMTP_HOST disables email (Go API no-op).
      env {
        name  = "SMTP_HOST"
        value = var.smtp_username != "" ? "smtp.azurecomm.net" : ""
      }
      env {
        name  = "SMTP_PORT"
        value = "587"
      }
      env {
        name  = "SMTP_FROM"
        value = var.smtp_from
      }
      # SMTP credentials passed as env vars (empty = email disabled via SMTP_HOST check).
      env {
        name  = "SMTP_USERNAME"
        value = var.smtp_username
      }
      env {
        name  = "SMTP_PASSWORD"
        value = var.smtp_password
      }
      # ── Application URLs ─────────────────────────────────────────────────────
      env {
        name  = "CONTACT_EMAIL"
        value = var.contact_email
      }
      env {
        name  = "ADMIN_DASHBOARD_URL"
        value = var.admin_dashboard_url
      }
      env {
        name  = "LANDING_BASE_URL"
        value = var.landing_base_url
      }
    }
  }

  tags = local.tags

  # CI/CD (az containerapp update) owns image updates — Terraform manages config only.
  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

# ── Admin Dashboard ───────────────────────────────────────────────────────────

resource "azurerm_container_app" "admin_dashboard" {
  name                         = "${local.prefix}-admin-dashboard"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  secret {
    name  = "microsoft-provider-authentication-secret"
    value = azuread_application_password.admin_easyauth.value
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = var.admin_min_replicas
    max_replicas = 3

    container {
      name   = "admin-dashboard"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/admin-dashboard:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "API_URL"
        value = "https://${azurerm_container_app.api.ingress[0].fqdn}"
      }
      # nginx fails to start on an empty upstream, so point /agent/ at a dead
      # local port (502) rather than blanking it when the MCP server is off.
      env {
        name  = "MCP_SERVER_URL"
        value = var.enable_mcp_server ? "https://${azurerm_container_app.mcp_server[0].ingress[0].fqdn}" : "http://127.0.0.1:9"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

# ── Landing Page ──────────────────────────────────────────────────────────────

resource "azurerm_container_app" "landing" {
  count = var.enable_landing ? 1 : 0

  name                         = "${local.prefix}-landing"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 1
    max_replicas = 3

    container {
      name   = "landing"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/landing:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

# ── Landing Page — Custom Domain (optional) ──────────────────────────────────
#
# Step 1 (Terraform): Bind the custom domain to the Container App.
# Step 2 (Manual, after DNS CNAME is live):
#   az containerapp hostname bind \
#     --hostname <landing_domain> \
#     -g <resource_group> \
#     -n aegis-prod-landing \
#     --environment <aca_env_name> \
#     --validation-method CNAME
# This provisions a free Azure-managed TLS certificate for the domain.

resource "azurerm_container_app_custom_domain" "landing" {
  count = var.enable_landing && var.landing_domain != "" ? 1 : 0

  name                     = var.landing_domain
  container_app_id         = azurerm_container_app.landing[0].id
  certificate_binding_type = "Disabled"

  # `az containerapp hostname bind` switches the binding to a managed cert.
  lifecycle {
    ignore_changes = [certificate_binding_type, container_app_environment_certificate_id]
  }
}

# ── Upload Portal ──────────────────────────────────────────────────────────────

resource "azurerm_container_app" "upload_portal" {
  count = var.enable_upload_portal ? 1 : 0

  name                         = "${local.prefix}-upload-portal"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 1
    max_replicas = 3

    container {
      name   = "upload-portal"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/upload-portal:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "API_URL"
        value = "https://${var.api_domain}"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

# ── Upload Portal — Custom Domain (optional) ──────────────────────────────────
#
# Step 1 (Terraform): Bind the custom domain to the Container App.
# Step 2 (Manual, after DNS CNAME is live):
#   az containerapp hostname bind \
#     --hostname <upload_portal_domain> \
#     -g <resource_group> \
#     -n aegis-prod-upload-portal \
#     --environment <aca_env_name> \
#     --validation-method CNAME
# This provisions a free Azure-managed TLS certificate for the domain.

resource "azurerm_container_app_custom_domain" "upload_portal" {
  count = var.enable_upload_portal && var.upload_portal_domain != "" ? 1 : 0

  name                     = var.upload_portal_domain
  container_app_id         = azurerm_container_app.upload_portal[0].id
  certificate_binding_type = "Disabled"

  lifecycle {
    ignore_changes = [certificate_binding_type, container_app_environment_certificate_id]
  }
}

# ── API / Admin — Custom Domains (optional) ───────────────────────────────────
#
# Same two-step flow as above. Skipped when the Application Gateway edge path
# is enabled, because the hostnames then point at the gateway instead of the
# Container Apps ingress. Before the first apply with these set, create in DNS:
#   CNAME <api_domain>         -> <api_fqdn output>
#   TXT   asuid.<api_domain>   -> <custom_domain_verification_id output>
#   (same pair for admin_domain -> admin_dashboard_fqdn)
# then run `az containerapp hostname bind` for each to get the managed cert.

resource "azurerm_container_app_custom_domain" "api" {
  count = var.api_domain != "" && !var.enable_application_gateway_waf ? 1 : 0

  name                     = var.api_domain
  container_app_id         = azurerm_container_app.api.id
  certificate_binding_type = "Disabled"

  lifecycle {
    ignore_changes = [certificate_binding_type, container_app_environment_certificate_id]
  }
}

resource "azurerm_container_app_custom_domain" "admin" {
  count = var.admin_domain != "" && !var.enable_application_gateway_waf ? 1 : 0

  name                     = var.admin_domain
  container_app_id         = azurerm_container_app.admin_dashboard.id
  certificate_binding_type = "Disabled"

  lifecycle {
    ignore_changes = [certificate_binding_type, container_app_environment_certificate_id]
  }
}

# ── DWV Viewer ────────────────────────────────────────────────────────────────

resource "azurerm_container_app" "dwv" {
  count = var.enable_dwv ? 1 : 0

  name                         = "${local.prefix}-dwv"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 1
    max_replicas = 3

    container {
      name   = "dwv"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/dwv:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

# ── MCP Server ────────────────────────────────────────────────────────────────

resource "azurerm_container_app" "mcp_server" {
  count = var.enable_mcp_server ? 1 : 0

  name                         = "${local.prefix}-mcp-server"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 1
    max_replicas = 3

    container {
      name   = "mcp-server"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/mcp-server:${var.api_image_tag}"
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "AEGIS_API_BASE_URL"
        value = "https://${azurerm_container_app.api.ingress[0].fqdn}"
      }
      env {
        name        = "AEGIS_API_TOKEN"
        secret_name = "mcp-api-token"
      }
      env {
        name  = "AZURE_OPENAI_ENDPOINT"
        value = var.azure_openai_endpoint
      }
    }
  }

  secret {
    name  = "mcp-api-token"
    value = azurerm_key_vault_secret.mcp_api_token.value
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

# ── Python Sidecars ───────────────────────────────────────────────────────────
# All sidecars: internal ingress only, receive file paths from Go API via HTTP,
# access Azure Blob Storage via managed identity.

resource "azurerm_container_app" "defacing" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-defacing"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "defacing"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/defacing:${var.api_image_tag}"
      cpu    = 1.0
      memory = "2Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "DEFACE_TOOL"
        value = "auto"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "phi_detection" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-phi-detection"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "phi-detection"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/phi-detection:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "PHI_TOOL"
        value = "auto"
      }
      env {
        name  = "PHI_CONFIDENCE_THRESHOLD"
        value = "0.4"
      }
      env {
        name  = "PHI_MIN_TEXT_LENGTH"
        value = "3"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "qc_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-qc-service"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "qc-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/qc-service:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "QC_TOOL"
        value = "auto"
      }
      env {
        name  = "QC_SNR_THRESHOLD"
        value = "10.0"
      }
      env {
        name  = "QC_GAP_RATIO"
        value = "2.0"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "bids_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-bids-service"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "bids-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/bids-service:${var.api_image_tag}"
      cpu    = 0.5
      memory = "1Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "BIDS_TOOL"
        value = "auto"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "classification_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-classify"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "classification-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/classification-service:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "CLASSIFY_TOOL"
        value = "auto"
      }
      env {
        name  = "CLASSIFY_CONFIDENCE_THRESHOLD"
        value = "0.5"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "protocol_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-protocol-service"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "protocol-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/protocol-service:${var.api_image_tag}"
      cpu    = 0.25
      memory = "0.5Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "PROTOCOL_TOOL"
        value = "auto"
      }
      env {
        name  = "PROTOCOL_DEFAULT_TOLERANCE"
        value = "5.0"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "synth_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-synth-service"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "synth-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/synth-service:${var.api_image_tag}"
      cpu    = 0.5
      memory = "1Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "analytics_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-analytics-service"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "analytics-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/analytics-service:${var.api_image_tag}"
      cpu    = 1.0
      memory = "2Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "ANALYTICS_TOOL"
        value = "auto"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}

resource "azurerm_container_app" "sct_service" {
  count = var.enable_sidecars ? 1 : 0

  name                         = "${local.prefix}-sct-service"
  container_app_environment_id = local.aca_env_id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  workload_profile_name        = "Consumption"

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  registry {
    server   = local.acr_server
    identity = local.identity_id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = false
    target_port                = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    min_replicas = 0
    max_replicas = 3

    container {
      name   = "sct-service"
      image  = local.use_placeholder_image ? local.placeholder_image : "${local.acr_server}/sct-service:${var.api_image_tag}"
      cpu    = 0.5
      memory = "1Gi"

      dynamic "env" {
        for_each = local.sidecar_common_env
        content {
          name  = env.value.name
          value = env.value.value
        }
      }
      env {
        name  = "SCT_TOOL"
        value = "auto"
      }
      env {
        name  = "SCT_CONTRAST"
        value = "t2"
      }
    }
  }

  tags = local.tags

  lifecycle {
    ignore_changes = [template[0].container[0].image]
  }
}
