# ── AWS Cloud Map — private DNS service discovery ──────────────────────────────
#
# Creates a private Route53 namespace (aegis.local) and a service entry for each
# sidecar. When ECS services register with these entries, they become reachable at
# http://<service-name>.aegis.local:8080 from within the VPC.
#
# The Go API's sidecar URLs (DEFACING_SERVICE_URL etc.) are set to these addresses
# in the ECS task definition environment variables in main.tf.

resource "aws_service_discovery_private_dns_namespace" "aegis" {
  name        = "aegis.local"
  description = "Private DNS namespace for AEGIS sidecar service discovery"
  vpc         = aws_vpc.main.id

  tags = {
    Name        = "${var.project_name}-${var.environment}-namespace"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ── Service entries (one per sidecar) ─────────────────────────────────────────

locals {
  sidecar_services = [
    "defacing",
    "phi-detection",
    "qc-service",
    "bids-service",
    "classification-service",
    "protocol-service",
    "synth-service",
    "dimse-receiver",
    "mcp-server",
    "api",
  ]
}

resource "aws_service_discovery_service" "sidecars" {
  for_each = toset(local.sidecar_services)

  name = each.key

  dns_config {
    namespace_id   = aws_service_discovery_private_dns_namespace.aegis.id
    routing_policy = "MULTIVALUE"

    dns_records {
      ttl  = 10
      type = "A"
    }
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-${each.key}"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
