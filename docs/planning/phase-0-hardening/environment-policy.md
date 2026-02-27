# Environment & Authentication Policy (Phase 0)

## Purpose

Define environment classes and deployment authentication invariants that later CI/CD checks must enforce.

## Environment Classes

- **local-dev**: developer laptop and local Docker Compose only.
- **dev**: shared development environment.
- **staging**: pre-production validation environment.
- **prod**: production environment.

## Scope for Hardening Program

The hardening program applies to: **staging** and **prod**.

## Authentication Invariants

### Non-dev invariant

For any non-dev deployment (staging/prod):

- `AUTH_ENABLED` **must** be `true`.
- Deployments must fail if `AUTH_ENABLED=false`.

### Provider invariant

For any non-dev deployment (staging/prod):

- `AUTH_PROVIDER` must be explicitly set.
- Allowed values: `iap`, `aws`, `azure`, or `auto` (when documented and approved).
- Empty or unknown provider values must fail deployment.

### API key policy

- API keys are allowed only as controlled machine-to-machine credentials.
- API keys must be stored only in cloud secret managers.
- API keys must never be hardcoded in code, workflow YAML, or plaintext docs.

## Baseline Environment Mapping

| Environment | AUTH_ENABLED | AUTH_PROVIDER | Notes |
|---|---|---|---|
| local-dev | false (allowed) | auto (allowed) | Developer convenience only |
| dev | true (recommended) | explicit preferred | Teams may allow temporary exceptions |
| staging | true (required) | explicit required | Same posture as prod |
| prod | true (required) | explicit required | No exceptions |

## Change Control

- Any exception requires written approval from Security Lead and Compliance Lead.
- Exceptions must include expiry date and rollback/remediation owner.

## Evidence Requirements

Store policy approval and enforcement evidence under:

- `docs/evidence/hardening-phase-0/global/`
