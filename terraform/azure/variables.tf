variable "azure_region" {
  description = "Primary Azure region"
  type        = string
  default     = "eastus"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "prod"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "aegis"
}

# ── Azure AD / Auth ──────────────────────────────────────────────────────────

variable "azure_ad_tenant_id" {
  description = "Azure AD tenant ID — used for Container Apps Easy Auth"
  type        = string
}

# ── Domains ──────────────────────────────────────────────────────────────────

variable "api_domain" {
  description = "Custom domain for the API (e.g. azure.api.aegisimaging.ai)"
  type        = string
  default     = ""
}

variable "admin_domain" {
  description = "Custom domain for the admin dashboard (e.g. azure.admin.aegisimaging.ai)"
  type        = string
  default     = ""
}

# ── Database ──────────────────────────────────────────────────────────────────

variable "db_admin_username" {
  description = "PostgreSQL administrator username"
  type        = string
  default     = "aegis"
}

variable "db_admin_password" {
  description = "PostgreSQL administrator password"
  type        = string
  sensitive   = true
}

variable "db_sku_name" {
  description = "PostgreSQL Flexible Server SKU (e.g. B_Standard_B2s, GP_Standard_D2s_v3)"
  type        = string
  default     = "B_Standard_B2s"
}

variable "db_storage_mb" {
  description = "PostgreSQL storage in MB (32768 = 32 GB)"
  type        = number
  default     = 32768
}

# ── Monitoring / Alerting ─────────────────────────────────────────────────────

variable "alert_email" {
  description = "Email address that receives Azure Monitor alerts"
  type        = string
  default     = ""
}

# ── SMTP / Email ──────────────────────────────────────────────────────────────

variable "smtp_from" {
  description = "Envelope sender address for transactional email"
  type        = string
  default     = "noreply@aegisimaging.ai"
}

# ── Container Images ──────────────────────────────────────────────────────────

variable "api_image_tag" {
  description = "Container image tag for the Go API"
  type        = string
  default     = "latest"
}

variable "api_min_replicas" {
  description = "Minimum replicas for the Go API Container App"
  type        = number
  default     = 1
}

variable "api_max_replicas" {
  description = "Maximum replicas for the Go API Container App"
  type        = number
  default     = 10
}

# ── DIMSE Receiver ────────────────────────────────────────────────────────────

variable "dimse_receiver_image" {
  description = "Container image for the DIMSE receiver VM. Empty = skip all DIMSE resources."
  type        = string
  default     = ""
}

variable "dimse_vm_size" {
  description = "Azure VM size for the DIMSE receiver"
  type        = string
  default     = "Standard_B2s"
}

variable "dimse_ssh_public_key" {
  description = "SSH public key for the DIMSE receiver VM (required when dimse_receiver_image is set)"
  type        = string
  default     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC0 placeholder-replace-with-real-key"
}

# ── MCP Server ────────────────────────────────────────────────────────────────

variable "azure_openai_endpoint" {
  description = "Azure OpenAI endpoint URL (e.g. https://aegis.openai.azure.com/). Leave empty to use Anthropic API directly."
  type        = string
  default     = ""
}

variable "azure_openai_api_key" {
  description = "Azure OpenAI API key (stored in Key Vault)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "anthropic_api_key" {
  description = "Anthropic API key for MCP server (stored in Key Vault). Used when azure_openai_endpoint is empty."
  type        = string
  sensitive   = true
  default     = ""
}

# ── First Admin Bootstrap ─────────────────────────────────────────────────────

variable "first_admin_email" {
  description = "Email address seeded as the first admin user on initial startup"
  type        = string
  default     = ""
}
