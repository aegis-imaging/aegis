# AEGIS Next Feature Prioritization Backlog

Review date: 2026-02-20
Scope reviewed: API, sidecars, frontend, Terraform, CI/workflow docs, and new MCP research drafts.

## Current-State Snapshot

- Core product phases (routing, institutions, profiles, audit, processing services, export, DIMSE ingress, timezone hardening, importer hardening) are largely shipped.
- DIMSE operations are now strong at API level (`/ingest/retry*` + observability + controls), but still mostly API/operator-key driven.
- Biggest delivery gap for pilot readiness is infrastructure completion/hardening (especially GCP/AWS Terraform TODO blocks).
- New MCP design artifacts are drafted (`docs/research/mcp-*.md`, schemas JSON), but no implementation exists yet.

## Key Findings Driving Priorities

1. Deployment readiness is still the primary blocker for pilot rollout.
- Evidence: `CODEX.md` Next Feature Note and multiple TODOs in `terraform/project/main.tf`, `terraform/infra/main.tf`, `terraform/aws/main.tf`.

2. Infrastructure security has explicit placeholders that should be closed before production traffic.
- Evidence: plaintext DB password placeholders, pending IAM/CMEK/IAP/Cloud Armor/HTTPS/Cognito tasks.

3. DIMSE reliability improved quickly, but resilience and operator UX still have obvious next steps.
- Evidence: retry/dead-letter tooling is rich in `dimse-receiver`, but queue state is still in-memory and no admin UI panel for those controls.

4. Dev/CI parity can be tightened to reduce “passes in CI, missed locally” risk.
- Evidence: `Makefile` lint targets do not cover all Python services/apps that CI validates.

5. MCP track has clear strategic direction and artifacts; it is ready for implementation planning once pilot-blocking infra is addressed.
- Evidence: `docs/research/mcp-api-wrapper-mvp-spec.md`, `docs/research/mcp-threat-model.md`, `docs/research/mcp-tool-schemas-v1.json`.

## Prioritized Feature Candidates

Scoring legend:
- Impact: 1-5 (higher = more value/risk reduction)
- Effort: S/M/L/XL
- Priority: P0 (now), P1 (next), P2 (after)

| Rank | Priority | Feature | Why Next | Impact | Effort |
|---|---|---|---|---:|---|
| 1 | P0 | **Complete GCP Terraform production module** (Cloud Run services, Artifact Registry, VPC/subnets/NAT, Cloud Armor, IAP, PSC SMTP, monitoring) | Unblocks first realistic cloud deployment and security baseline | 5 | XL |
| 2 | P0 | **Complete AWS Terraform auth/edge hardening** (ACM HTTPS listener + Cognito integration) | Required to claim production-ready AWS path | 5 | L |
| 3 | P0 | **Secrets & credential hardening across Terraform** (remove static DB passwords, enforce Secret Manager/Secrets Manager patterns) | Direct security/compliance risk reduction | 5 | M |
| 4 | P0 | **Automated cloud smoke test suite** (auth, health, upload, pipeline progression, export) | Converts “manual checklist” to repeatable release gate | 5 | M |
| 5 | P0 | **DIMSE PACS end-to-end validation harness + runbook** (storescu/C-STORE path, retry path, failure drills) | Explicit pilot requirement for enterprise ingress confidence | 5 | M |
| 6 | P1 | **Persist DIMSE retry/dead-letter queue state** (durable store, restart-safe) | Eliminates in-memory loss risk during restart/crash | 5 | L |
| 7 | P1 | **Admin Dashboard DIMSE Ops panel** (summary/details/actions UI over `/ingest/retry*`) | Reduces operator friction vs API-only controls | 4 | M |
| 8 | P1 | **Alerting/notifications for retry thresholds** (dead-letter nonzero, age threshold breaches) | Faster incident detection and response | 4 | M |
| 9 | P1 | **Study diagnostics endpoint** (“why stuck?” summary from statuses + audit + routing) | High operator leverage, foundation for AI-assisted triage | 4 | M |
| 10 | P1 | **Local/CI parity improvements** (`Makefile` lint/test coverage alignment with CI matrix) | Reduces missed regressions and improves dev velocity | 3 | S |
| 11 | P2 | **MCP Server MVP Phase 0 (read-only)** (`list_studies`, `get_study_detail`, `get_study_audit`, `get_system_health`, etc.) | Fastest path to assistant-driven ops visibility | 4 | M |
| 12 | P2 | **MCP Server Phase 1 (guarded writes)** (single-study triggers with `confirm` + `reason`) | Enables safe operator acceleration | 4 | M |
| 13 | P2 | **MCP security hardening package** (idempotency keys, rate limits, outbound allowlist, redaction tests) | Required before broad production MCP adoption | 5 | M |
| 14 | P2 | **Optional MCP DIMSE operator tools** (`retry_dimse_study`) with key management controls | Extends MCP to ingress incident response | 3 | M |

## Recommended Execution Order (Suggested)

If the goal is pilot readiness first, run this order:

1. Feature #1 (GCP Terraform completion)
2. Feature #3 (secrets hardening)
3. Feature #4 (automated smoke tests)
4. Feature #5 (DIMSE PACS E2E validation)
5. Feature #2 (AWS HTTPS/Cognito completion)
6. Feature #6 (durable DIMSE retry state)
7. Feature #7 (DIMSE admin UI panel)
8. Feature #8 (alerts)
9. Feature #9 (study diagnostics)
10. Feature #10 (local/CI parity)
11. Feature #11-#14 (MCP implementation phases)

## Candidate Branch Naming (Optional)

- `feature/terraform-gcp-prod-completion`
- `feature/terraform-aws-https-cognito`
- `feature/secrets-hardening-infra`
- `feature/cloud-smoke-test-suite`
- `feature/dimse-pacs-e2e-validation`
- `feature/dimse-retry-durable-store`
- `feature/admin-dimse-ops-panel`
- `feature/dimse-alerting-thresholds`
- `feature/study-diagnostics-summary`
- `feature/dev-ci-parity-lint-checks`
- `feature/mcp-server-readonly-mvp`
- `feature/mcp-server-guarded-write-tools`

## Notes for Prioritization Session

- If you want to pursue MCP sooner, you can start #11 in parallel with #4 (smoke tests), but keep #3 (secrets hardening) ahead of any write-capable MCP rollout.
- If pilot timeline is aggressive, treat #1/#3/#4/#5 as non-negotiable gatekeepers before broad external usage.
