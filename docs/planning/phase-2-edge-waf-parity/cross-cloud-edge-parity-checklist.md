# Cross-Cloud Edge Parity Checklist (Phase 2)

Use this checklist in staging and production reviews.

## GCP (Cloud Armor)

- [ ] `google_compute_security_policy.api` exists
- [ ] API backend is attached to Cloud Armor policy
- [ ] Allowed IP ranges policy is documented
- [ ] Rate-limit/abuse mitigation policy is enabled

## AWS (WAF v2)

- [ ] `aws_wafv2_web_acl.api` exists
- [ ] Web ACL associated with ALB
- [ ] Managed rule set enabled
- [ ] IP allowlist behavior documented (or explicit open rationale)
- [ ] STOW/DICOM exception rule documented and tested

## Azure (Application Gateway WAF v2)

- [ ] `enable_application_gateway_waf=true` in staging/prod tfvars
- [ ] `app_gateway_certificate_secret_id` is set to versioned Key Vault secret URI
- [ ] `api_domain` and `admin_domain` are set
- [ ] Application Gateway routes API/admin correctly
- [ ] WAF policy attached and in Prevention mode

## Global Validation

- [ ] TLS 1.2+ enforced at ingress
- [ ] Equivalent policy intent documented across all clouds
- [ ] Alerting exists for edge failures and anomalies
- [ ] Exceptions/risk acceptances are approved and time-bounded
