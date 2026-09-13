# Phase 3 Auth Guardrail Policy

Source plan: `implementation-plan.MD` (Phase 3)

## Non-Dev Authentication Invariants

For staging and production deployments:

1. `AUTH_ENABLED` must be `true`.
2. `AUTH_PROVIDER` must be explicitly set to a valid provider.
3. IaC and deploy pipelines must fail early if these conditions are not met.

## Current Expected Provider Mapping

- GCP: `AUTH_PROVIDER=iap`
- AWS: `AUTH_PROVIDER=aws`
- Azure: `AUTH_PROVIDER=azure`

## Enforcement Points

- CI pull-request enforcement via `scripts/check-auth-guardrails.sh`
- Deployment preflight in cloud deploy workflows (`deploy-aws.yml`, `deploy-azure.yml`)

## Exception Handling

Any temporary exception requires Security + Compliance approval with expiry date and remediation owner.
