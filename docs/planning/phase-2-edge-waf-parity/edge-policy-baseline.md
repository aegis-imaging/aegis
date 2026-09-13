# Phase 2 Edge Security Baseline (WAF Parity)

Source plan: `implementation-plan.MD` (Phase 2)

## Policy Intent (Cross-Cloud)

All public ingress paths for API/admin must enforce equivalent edge controls:

1. TLS termination with modern cipher policy.
2. Managed WAF rule set enabled.
3. IP allowlist support for restricted environments.
4. Rate limiting for abuse protection.
5. Explicit, documented exceptions for DICOM binary upload paths.

## Cloud Mapping

- **GCP**: Cloud Armor policy on API backend.
- **AWS**: WAF v2 Web ACL on ALB.
- **Azure**: Application Gateway WAF v2 (optional Terraform resources; enabled in staging/prod with cert + domains).

## Azure Enablement Requirements

To enable Azure WAF ingress path:

- set `enable_application_gateway_waf = true`
- set `app_gateway_certificate_secret_id` to Key Vault cert secret version URI
- set `api_domain` and `admin_domain`
- apply Terraform and validate host-based routing + health probes

## Validation Checklist

- [ ] API route served through WAF endpoint
- [ ] Admin route served through WAF endpoint
- [ ] WAF policy attached to gateway
- [ ] OWASP managed rules enabled
- [ ] Health probes return healthy
- [ ] Rollback path documented and tested
