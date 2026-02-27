# Unified Cross-Cloud Operations Runbook

This runbook consolidates common operational actions across GCP, AWS, and Azure and links to provider-specific procedures.

## Scope

- API availability incidents
- Pipeline failures and stuck studies
- Destination probe failures
- DIMSE retry/dead-letter incidents
- Authentication and access incidents

## Common Triage Sequence

1. Confirm alert details and affected scope (environment/project/service).
2. Verify `/healthz` and current error/latency indicators.
3. Check pipeline and routing telemetry.
4. Check sidecar and DIMSE service health.
5. Execute cloud-specific remediation steps.
6. Record timeline, actor, and outcome in incident notes.

## Cloud-Specific Command References

- GCP deployment/ops: [gcp-first-deploy-checklist.md](../planning/gcp-first-deploy-checklist.md)
- AWS deployment/ops: [aws-first-deploy-checklist.md](../planning/aws-first-deploy-checklist.md)
- DIMSE E2E validation: [dimse-pacs-e2e-validation-runbook.md](../planning/dimse-pacs-e2e-validation-runbook.md)

## Existing Runbooks (Authoritative Procedures)

- Incident response: [incident-response.md](incident-response.md)
- Per-alert responses: [alert-response.md](alert-response.md)
- Secret rotation: [secret-rotation.md](secret-rotation.md)
- Cross-cloud DICOM routing: [cross-cloud-routing.md](cross-cloud-routing.md)

## Minimum Response Artifacts

For each incident, capture:

- Alert source and timestamp
- Initial severity and rationale
- Containment actions executed
- Recovery actions and validation outcome
- Follow-up remediations with owners/dates
