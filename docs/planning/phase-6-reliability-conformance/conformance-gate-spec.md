# Cross-Cloud Conformance Gate Spec (Phase 6)

## Gate purpose

Prevent drift between intended architecture and deployment/CI implementation.

## Required checks

1. Canonical services are represented in deploy workflows.
2. Auth guardrail check is present in CI.
3. Monitoring, edge, and HIPAA baseline checkers are present in CI.
4. Health-check paths exist in deploy/monitoring definitions.

## Enforcement

- Script: `scripts/check-cross-cloud-conformance.sh`
- CI job: `cross-cloud-conformance`
- Trigger: Pull requests to `develop` and `main`

## Failure behavior

- Any missing required signal fails the PR.
- Exception process requires security + platform lead approval in PR comments.
