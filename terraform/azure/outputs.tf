output "acr_login_server" {
  description = "Azure Container Registry login server (use for docker push)"
  value       = azurerm_container_registry.main.login_server
}

output "api_fqdn" {
  description = "Go API Container App fully-qualified domain name"
  value       = azurerm_container_app.api.ingress[0].fqdn
}

output "api_url" {
  description = "Go API base URL"
  value       = "https://${azurerm_container_app.api.ingress[0].fqdn}"
}

output "admin_dashboard_url" {
  description = "Admin Dashboard URL"
  value       = "https://${azurerm_container_app.admin_dashboard.ingress[0].fqdn}"
}

output "landing_url" {
  description = "Landing Page URL"
  value       = var.landing_domain != "" ? "https://${var.landing_domain}" : "https://${azurerm_container_app.landing.ingress[0].fqdn}"
}

output "dwv_url" {
  description = "DWV Viewer URL"
  value       = "https://${azurerm_container_app.dwv.ingress[0].fqdn}"
}

output "mcp_server_url" {
  description = "MCP Server URL"
  value       = "https://${azurerm_container_app.mcp_server.ingress[0].fqdn}"
}

output "storage_account_name" {
  description = "Azure Blob Storage account name (set as AZURE_STORAGE_ACCOUNT)"
  value       = azurerm_storage_account.dicom.name
}

output "storage_container_name" {
  description = "Azure Blob Storage container name (set as AZURE_STORAGE_CONTAINER)"
  value       = azurerm_storage_container.dicom.name
}

output "postgres_fqdn" {
  description = "PostgreSQL Flexible Server FQDN"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "nat_gateway_ip" {
  description = "Static outbound IP for the NAT Gateway (allowlist for external SMTP relays)"
  value       = azurerm_public_ip.nat.ip_address
}

output "key_vault_uri" {
  description = "Azure Key Vault URI"
  value       = azurerm_key_vault.main.vault_uri
}

output "acs_smtp_host" {
  description = "Azure Communication Services SMTP relay hostname"
  value       = "smtp.azurecomm.net"
}

output "acs_smtp_port" {
  description = "Azure Communication Services SMTP relay port"
  value       = "587"
}

output "acs_resource_name" {
  description = "ACS resource name — used as part of SMTP username: <EntraAppClientId>|<TenantId>|<AcsResourceName>"
  value       = azurerm_communication_service.main.name
}

output "dimse_public_ip" {
  description = "DIMSE receiver VM static public IP (only set when dimse_receiver_image is configured)"
  value       = local.dimse_enabled ? azurerm_public_ip.dimse[0].ip_address : null
}

output "app_gateway_public_ip" {
  description = "Application Gateway public IP (only when WAF edge path is enabled)"
  value       = var.enable_application_gateway_waf ? azurerm_public_ip.app_gateway[0].ip_address : null
}

output "app_gateway_waf_policy_id" {
  description = "Application Gateway WAF policy ID (only when enabled)"
  value       = var.enable_application_gateway_waf ? azurerm_web_application_firewall_policy.app_gateway[0].id : null
}
