# AEGIS Project Status and Top Priorities

Date: 2026-02-20  
Reviewed branch: `working/from-origin-develop` (tracking `origin/develop`)

## Scope Reviewed

- Core docs and setup runbooks (`README.md`, `SETUP_CHECKLIST.md`, `CODEX.md`)
- CI and smoke workflows (`.github/workflows/ci.yml`, `.github/workflows/cloud-smoke.yml`)
- Core infrastructure Terraform (`terraform/project`, `terraform/infra`, `terraform/aws`)
- API, Python sidecars, MCP server, and local automation (`Makefile`, `scripts/cloud_smoke_test.py`)

## Verification Snapshot

- `go test -count=1 ./...` in `api/`: passes.
- `terraform validate` in `terraform/project`, `terraform/infra`, and `terraform/aws`: blocked locally until `terraform init` downloads providers.
- Local sidecar `pytest` runs currently fail without Python deps installed (for example missing `pydicom`).

## Current Status Findings

1. AWS deploy path is not production-complete yet.
   - `terraform/aws/main.tf` still contains TODO placeholders for HTTPS and Cognito (`terraform/aws/main.tf:416`).
   - ALB/network resources are present, but ECS runtime service resources are not yet defined.
2. GCP project bootstrap security baseline is still incomplete.
   - Explicit TODOs remain for KMS, IAM, VPC-SC, and audit log configuration (`terraform/project/main.tf:75`).
3. Secret handling is improved, but plaintext secret input flow remains.
   - `db_password` is still passed through Terraform vars and written as a secret version (`terraform/infra/main.tf:466`, `terraform/infra/main.tf:480`).
4. DIMSE retry/dead-letter queue state is currently in-memory.
   - This creates restart-loss risk for queued/dead-letter items (`dimse-receiver/app/ingest.py:32`, `dimse-receiver/app/ingest.py:42`).
5. MCP server is partially implemented.
   - Read tools and some write tools are implemented, but multiple write tools remain scaffolded/stubbed (`mcp-server/src/server.ts:267`).
6. Local and CI checks are not fully aligned.
   - `Makefile` lint targets do not cover all services/apps currently validated by CI (`Makefile:33`, `.github/workflows/ci.yml:48`).
7. Legal/license packaging is still unresolved.
   - `README.md` still lists License as `TBD` (`README.md:71`).

## Top 10 Build-Out Priorities (No Sprint Framing)

1. Complete `terraform/project` security baseline (KMS, IAM least-privilege, VPC-SC, audit logging).
2. Complete AWS runtime deployment resources (ECS task/service definitions and service wiring).
3. Complete AWS HTTPS + Cognito edge/auth integration end-to-end.
4. Remove remaining plaintext credential input patterns from infra workflows and enforce secret-manager-first runtime config.
5. Keep cloud smoke tests as a required release gate for deployed environments.
6. Build DIMSE PACS end-to-end validation harness with success, retry, dead-letter, and recovery scenarios.
7. Add durable persistence for DIMSE retry/dead-letter state.
8. Add Admin Dashboard DIMSE operations panel for retry/dead-letter visibility and control.
9. Align local pre-PR checks with CI matrix coverage.
10. Expand integration/frontend critical-path tests and finalize licensing/release packaging.

## Recommended Execution Order

1. `terraform/project` security baseline
2. secrets hardening
3. cloud smoke gate enforcement
4. DIMSE PACS E2E validation harness
5. AWS runtime + HTTPS/Cognito completion
6. DIMSE durable retry storage
7. DIMSE ops UI
8. local/CI parity
9. test expansion + release/legal packaging

