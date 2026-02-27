# Optional Azure Application Gateway + WAF v2 edge path.
# Disabled by default; enable in staging/prod with custom domains and TLS cert.

resource "azurerm_public_ip" "app_gateway" {
  count = var.enable_application_gateway_waf ? 1 : 0

  name                = "${local.prefix}-appgw-ip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.tags
}

resource "azurerm_web_application_firewall_policy" "app_gateway" {
  count = var.enable_application_gateway_waf ? 1 : 0

  name                = "${local.prefix}-waf-policy"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags

  policy_settings {
    enabled                     = true
    mode                        = "Prevention"
    request_body_check          = true
    file_upload_limit_in_mb     = 100
    max_request_body_size_in_kb = 128
  }

  managed_rules {
    managed_rule_set {
      type    = "OWASP"
      version = "3.2"
    }
  }
}

resource "azurerm_application_gateway" "main" {
  count = var.enable_application_gateway_waf ? 1 : 0

  name                = "${local.prefix}-appgw"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  firewall_policy_id  = azurerm_web_application_firewall_policy.app_gateway[0].id

  sku {
    name     = "WAF_v2"
    tier     = "WAF_v2"
    capacity = var.app_gateway_capacity
  }

  gateway_ip_configuration {
    name      = "gateway-ip-config"
    subnet_id = azurerm_subnet.gateway.id
  }

  frontend_port {
    name = "https-port"
    port = 443
  }

  frontend_ip_configuration {
    name                 = "frontend-ip"
    public_ip_address_id = azurerm_public_ip.app_gateway[0].id
  }

  ssl_certificate {
    name                = "tls-cert"
    key_vault_secret_id = var.app_gateway_certificate_secret_id
  }

  backend_address_pool {
    name  = "api-backend"
    fqdns = [azurerm_container_app.api.ingress[0].fqdn]
  }

  backend_address_pool {
    name  = "admin-backend"
    fqdns = [azurerm_container_app.admin_dashboard.ingress[0].fqdn]
  }

  probe {
    name                                      = "api-probe"
    protocol                                  = "Https"
    host                                      = azurerm_container_app.api.ingress[0].fqdn
    path                                      = "/healthz"
    interval                                  = 30
    timeout                                   = 30
    unhealthy_threshold                       = 3
    pick_host_name_from_backend_http_settings = false
    match {
      status_code = ["200-399"]
    }
  }

  probe {
    name                                      = "admin-probe"
    protocol                                  = "Https"
    host                                      = azurerm_container_app.admin_dashboard.ingress[0].fqdn
    path                                      = "/"
    interval                                  = 30
    timeout                                   = 30
    unhealthy_threshold                       = 3
    pick_host_name_from_backend_http_settings = false
    match {
      status_code = ["200-399"]
    }
  }

  backend_http_settings {
    name                                = "api-https-settings"
    protocol                            = "Https"
    port                                = 443
    cookie_based_affinity               = "Disabled"
    request_timeout                     = 60
    pick_host_name_from_backend_address = true
    probe_name                          = "api-probe"
  }

  backend_http_settings {
    name                                = "admin-https-settings"
    protocol                            = "Https"
    port                                = 443
    cookie_based_affinity               = "Disabled"
    request_timeout                     = 60
    pick_host_name_from_backend_address = true
    probe_name                          = "admin-probe"
  }

  http_listener {
    name                           = "api-listener"
    frontend_ip_configuration_name = "frontend-ip"
    frontend_port_name             = "https-port"
    protocol                       = "Https"
    ssl_certificate_name           = "tls-cert"
    host_name                      = var.api_domain
  }

  http_listener {
    name                           = "admin-listener"
    frontend_ip_configuration_name = "frontend-ip"
    frontend_port_name             = "https-port"
    protocol                       = "Https"
    ssl_certificate_name           = "tls-cert"
    host_name                      = var.admin_domain
  }

  request_routing_rule {
    name                       = "api-routing"
    rule_type                  = "Basic"
    http_listener_name         = "api-listener"
    backend_address_pool_name  = "api-backend"
    backend_http_settings_name = "api-https-settings"
    priority                   = 100
  }

  request_routing_rule {
    name                       = "admin-routing"
    rule_type                  = "Basic"
    http_listener_name         = "admin-listener"
    backend_address_pool_name  = "admin-backend"
    backend_http_settings_name = "admin-https-settings"
    priority                   = 110
  }

  tags = local.tags

  lifecycle {
    precondition {
      condition = (
        var.app_gateway_certificate_secret_id != "" &&
        var.api_domain != "" &&
        var.admin_domain != ""
      )
      error_message = "When enable_application_gateway_waf=true, set app_gateway_certificate_secret_id, api_domain, and admin_domain."
    }
  }
}
