# AEGIS — Azure Infrastructure
#
# Functionally equivalent to terraform/infra/ (GCP) and terraform/aws/ (AWS).
# AEGIS is cloud-agnostic at the application layer; this module provides the
# Azure-native underpinning.
#
# What it provisions:
# - Resource Group, Virtual Network, subnets (apps, db, gateway, dimse)
# - Azure NAT Gateway (static SMTP egress IP)
# - Azure Container Registry (ACR) for container images
# - Azure Database for PostgreSQL - Flexible Server (private, PostgreSQL 15)
# - Azure Blob Storage account + container for DICOM files
# - Azure Key Vault for secrets
# - Azure Container Apps Environment (shared, VNet-integrated)
# - Azure Application Gateway v2 with WAF_v2 (external traffic + TLS)
# - Azure Communication Services Email (transactional email via SMTP relay)
# - Log Analytics Workspace + Application Insights (telemetry)
# - User-assigned managed identity (ACR pull + Key Vault + Blob access)
#
# Depends on: Azure subscription, azurerm provider authenticated

locals {
  prefix = "${var.project_name}-${var.environment}"
  tags = {
    project     = var.project_name
    environment = var.environment
    managed_by  = "terraform"
  }
}

data "azurerm_client_config" "current" {}

# ── Resource Group ────────────────────────────────────────────────────────────

resource "azurerm_resource_group" "main" {
  name     = local.prefix
  location = var.azure_region
  tags     = local.tags
}

# ── Virtual Network ───────────────────────────────────────────────────────────

resource "azurerm_virtual_network" "main" {
  name                = "${local.prefix}-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

resource "azurerm_subnet" "apps" {
  name                 = "apps"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.1.0/24"]

  delegation {
    name = "aca-delegation"
    service_delegation {
      name    = "Microsoft.App/environments"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_subnet" "db" {
  name                 = "db"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.2.0/24"]

  delegation {
    name = "postgres-delegation"
    service_delegation {
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_subnet" "gateway" {
  name                 = "gateway"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.3.0/24"]
}

# ── NAT Gateway (static SMTP egress IP) ──────────────────────────────────────

resource "azurerm_public_ip" "nat" {
  name                = "${local.prefix}-nat-ip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.tags
}

resource "azurerm_nat_gateway" "main" {
  name                    = "${local.prefix}-nat"
  location                = azurerm_resource_group.main.location
  resource_group_name     = azurerm_resource_group.main.name
  sku_name                = "Standard"
  idle_timeout_in_minutes = 10
  tags                    = local.tags
}

resource "azurerm_nat_gateway_public_ip_association" "main" {
  nat_gateway_id       = azurerm_nat_gateway.main.id
  public_ip_address_id = azurerm_public_ip.nat.id
}

resource "azurerm_subnet_nat_gateway_association" "apps" {
  subnet_id      = azurerm_subnet.apps.id
  nat_gateway_id = azurerm_nat_gateway.main.id
}

# ── User-Assigned Managed Identity ───────────────────────────────────────────
# Used by all Container Apps for: ACR pull, Key Vault read, Blob Storage read/write

resource "azurerm_user_assigned_identity" "aca_workload" {
  name                = "${local.prefix}-aca-workload"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

# ── Azure Container Registry ──────────────────────────────────────────────────

resource "azurerm_container_registry" "main" {
  name                = replace("${var.project_name}${var.environment}acr", "-", "")
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
  admin_enabled       = false
  tags                = local.tags
}

resource "azurerm_role_assignment" "acr_pull" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.aca_workload.principal_id
}

# ── PostgreSQL Flexible Server ─────────────────────────────────────────────────

resource "azurerm_private_dns_zone" "postgres" {
  name                = "${local.prefix}.private.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  name                  = "${local.prefix}-postgres-dns-link"
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = azurerm_virtual_network.main.id
  resource_group_name   = azurerm_resource_group.main.name
}

resource "azurerm_postgresql_flexible_server" "main" {
  name                          = "${local.prefix}-postgres-${random_string.pg_suffix.result}"
  resource_group_name           = azurerm_resource_group.main.name
  location                      = azurerm_resource_group.main.location
  version                       = "15"
  delegated_subnet_id           = azurerm_subnet.db.id
  private_dns_zone_id           = azurerm_private_dns_zone.postgres.id
  public_network_access_enabled = false

  administrator_login    = var.db_admin_username
  administrator_password = var.db_admin_password

  sku_name   = var.db_sku_name
  storage_mb = var.db_storage_mb

  backup_retention_days        = 7
  geo_redundant_backup_enabled = false

  tags = local.tags

  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgres]

  # PostgreSQL Flexible Server provisioning takes 10-20 minutes on Azure.
  timeouts {
    create = "60m"
    update = "60m"
    delete = "60m"
  }

  # Azure auto-assigns a zone on creation; ignore drift to prevent spurious
  # in-place updates that Azure rejects when HA is not configured.
  lifecycle {
    ignore_changes = [zone]
  }
}

resource "azurerm_postgresql_flexible_server_database" "aegis" {
  name      = "aegis"
  server_id = azurerm_postgresql_flexible_server.main.id
  collation = "en_US.utf8"
  charset   = "utf8"
}

resource "azurerm_postgresql_flexible_server_configuration" "ssl" {
  name      = "require_secure_transport"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "off" # TLS handled at app layer; VNet isolation provides transport security
}

# ── Azure Blob Storage ────────────────────────────────────────────────────────

resource "azurerm_storage_account" "dicom" {
  name                     = replace("${var.project_name}${var.environment}dicom", "-", "")
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  access_tier              = "Hot"

  allow_nested_items_to_be_public = false
  min_tls_version                 = "TLS1_2"

  blob_properties {
    versioning_enabled = true
    delete_retention_policy {
      days = 7
    }
  }

  tags = local.tags
}

resource "azurerm_storage_container" "dicom" {
  name                  = "dicom"
  storage_account_id    = azurerm_storage_account.dicom.id
  container_access_type = "private"
}

resource "azurerm_role_assignment" "blob_contributor" {
  scope                = azurerm_storage_account.dicom.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.aca_workload.principal_id
}

# Allows generating user-delegation SAS tokens (needed for GenerateUploadURL / GenerateDownloadURL)
resource "azurerm_role_assignment" "blob_delegator" {
  scope                = azurerm_storage_account.dicom.id
  role_definition_name = "Storage Blob Delegator"
  principal_id         = azurerm_user_assigned_identity.aca_workload.principal_id
}

# ── Key Vault ─────────────────────────────────────────────────────────────────

resource "azurerm_key_vault" "main" {
  name                      = "${local.prefix}-kv-${random_string.pg_suffix.result}"
  location                  = azurerm_resource_group.main.location
  resource_group_name       = azurerm_resource_group.main.name
  tenant_id                 = data.azurerm_client_config.current.tenant_id
  sku_name                  = "standard"
  purge_protection_enabled  = true
  soft_delete_retention_days = 7

  # Allow Container Apps managed identity to read secrets
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = azurerm_user_assigned_identity.aca_workload.principal_id

    secret_permissions = ["Get", "List"]
  }

  # Allow the Terraform principal to manage secrets
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id

    secret_permissions = ["Get", "List", "Set", "Delete", "Purge"]
  }

  tags = local.tags
}

resource "azurerm_key_vault_secret" "db_password" {
  name         = "db-password"
  value        = var.db_admin_password
  key_vault_id = azurerm_key_vault.main.id
}

resource "azurerm_key_vault_secret" "mcp_api_token" {
  name         = "mcp-aegis-api-token"
  value        = random_password.mcp_api_token.result
  key_vault_id = azurerm_key_vault.main.id
}

resource "random_password" "mcp_api_token" {
  length  = 32
  special = false
}

# PostgreSQL Flexible Server names must be globally unique in Azure.
# This 6-char suffix is generated once and stored in Terraform state.
resource "random_string" "pg_suffix" {
  length  = 6
  upper   = false
  special = false
}

# Azure OpenAI key (optional — stored when azure_openai_api_key is provided)
resource "azurerm_key_vault_secret" "azure_openai_key" {
  count        = var.azure_openai_api_key != "" ? 1 : 0
  name         = "azure-openai-api-key"
  value        = var.azure_openai_api_key
  key_vault_id = azurerm_key_vault.main.id
}

# Anthropic key (optional — stored when anthropic_api_key is provided)
resource "azurerm_key_vault_secret" "anthropic_key" {
  count        = var.anthropic_api_key != "" ? 1 : 0
  name         = "anthropic-api-key"
  value        = var.anthropic_api_key
  key_vault_id = azurerm_key_vault.main.id
}

# Full DATABASE_URL stored as a Key Vault secret so the Container App
# receives it via a secret reference rather than an inline plaintext env var.
resource "azurerm_key_vault_secret" "database_url" {
  name         = "database-url"
  value        = "postgres://${var.db_admin_username}:${var.db_admin_password}@${azurerm_postgresql_flexible_server.main.fqdn}:5432/aegis?sslmode=require"
  key_vault_id = azurerm_key_vault.main.id
}

# ACS SMTP credentials (optional — stored when smtp_username/smtp_password are set).
# After Terraform provisions ACS, retrieve credentials from Azure portal:
#   Communication Services resource → Settings → Keys → copy resource name + Primary Key.
resource "azurerm_key_vault_secret" "smtp_username" {
  count        = var.smtp_username != "" ? 1 : 0
  name         = "smtp-username"
  value        = var.smtp_username
  key_vault_id = azurerm_key_vault.main.id
}

resource "azurerm_key_vault_secret" "smtp_password" {
  count        = var.smtp_password != "" ? 1 : 0
  name         = "smtp-password"
  value        = var.smtp_password
  key_vault_id = azurerm_key_vault.main.id
}

# ── Azure Communication Services (Email) ─────────────────────────────────────
# Provides SMTP relay at smtp.azurecomm.net:587 for transactional email.
# The Go API uses standard SMTP env vars (SMTP_HOST/PORT/USERNAME/PASSWORD) —
# no Go code changes needed.

resource "azurerm_communication_service" "main" {
  name                = "${local.prefix}-acs"
  resource_group_name = azurerm_resource_group.main.name
  data_location       = "United States"
  tags                = local.tags
}

resource "azurerm_email_communication_service" "main" {
  name                = "${local.prefix}-email"
  resource_group_name = azurerm_resource_group.main.name
  data_location       = "United States"
  tags                = local.tags
}

resource "azurerm_email_communication_service_domain" "aegis" {
  name             = "AzureManagedDomain"
  email_service_id = azurerm_email_communication_service.main.id

  domain_management = "AzureManaged"
}

resource "azurerm_communication_service_email_domain_association" "main" {
  communication_service_id = azurerm_communication_service.main.id
  email_service_domain_id  = azurerm_email_communication_service_domain.aegis.id
}

# ── Log Analytics Workspace + Application Insights ───────────────────────────

resource "azurerm_log_analytics_workspace" "main" {
  name                = "${local.prefix}-logs"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  tags                = local.tags
}

resource "azurerm_application_insights" "main" {
  name                = "${local.prefix}-appinsights"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  workspace_id        = azurerm_log_analytics_workspace.main.id
  application_type    = "web"
  tags                = local.tags
}

# ── Container Apps Environment ────────────────────────────────────────────────

resource "azurerm_container_app_environment" "main" {
  name                       = "${local.prefix}-aca-env"
  location                   = azurerm_resource_group.main.location
  resource_group_name        = azurerm_resource_group.main.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  infrastructure_subnet_id   = azurerm_subnet.apps.id

  # Public ingress — each Container App with external_enabled=true gets its own FQDN.
  # Application Gateway (for custom domains + WAF) can be layered on later.
  internal_load_balancer_enabled = false

  tags = local.tags
}

# ── Application Gateway (WAF v2) ──────────────────────────────────────────────
# NOTE: Application Gateway requires a TLS certificate stored as a proper secret
# in Key Vault (full URI: https://vault.vault.azure.net/secrets/<name>/<version>).
# For initial provisioning, we use Container Apps' built-in HTTPS (each app gets
# a free *.azurecontainerapps.io cert). Add Application Gateway back once a cert
# is provisioned and custom domains are ready.
#
# To re-enable: provision a cert (e.g. via azurerm_app_service_certificate or
# upload to Key Vault), set internal_load_balancer_enabled=true on the ACA env,
# and uncomment the Application Gateway + WAF resources below.
