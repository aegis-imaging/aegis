locals {
  admin_auth_redirect_uri = "${trimsuffix(var.admin_dashboard_url, "/")}/.auth/login/aad/callback"
  admin_auth_issuer       = "https://sts.windows.net/${var.azure_ad_tenant_id}/"
}

resource "azuread_application" "admin_easyauth" {
  display_name     = "${local.prefix}-admin-easyauth"
  sign_in_audience = "AzureADMyOrg"

  web {
    redirect_uris = [local.admin_auth_redirect_uri]
  }
}

resource "azuread_service_principal" "admin_easyauth" {
  client_id = azuread_application.admin_easyauth.client_id
}

resource "azuread_application_password" "admin_easyauth" {
  application_id = azuread_application.admin_easyauth.id
  display_name   = "container-apps-easyauth"
  end_date_relative = "17520h"
}

resource "azapi_resource" "admin_dashboard_auth" {
  type      = "Microsoft.App/containerApps/authConfigs@2023-05-01"
  name      = "current"
  parent_id = azurerm_container_app.admin_dashboard.id

  body = {
    properties = {
      platform = {
        enabled = true
      }
      globalValidation = {
        unauthenticatedClientAction = "RedirectToLoginPage"
        redirectToProvider          = "AzureActiveDirectory"
      }
      identityProviders = {
        azureActiveDirectory = {
          registration = {
            clientId                = azuread_application.admin_easyauth.client_id
            clientSecretSettingName = "microsoft-provider-authentication-secret"
            openIdIssuer            = local.admin_auth_issuer
          }
        }
      }
      login = {
        preserveUrlFragmentsForLogins = false
      }
    }
  }
}
