# CODEX Working Process

This file is my local execution checklist for AEGIS feature work.

## Project Understanding (from root `*.md` docs)

1. AEGIS = **Anonymization & Exchange Gateway for Imaging Studies**: a multi-cloud (GCP/AWS/Azure) platform for HIPAA-compliant imaging sharing across all DICOM modalities.
2. It has **two ingest paths**:
   - External-site browser upload with client-side DICOM tag de-identification.
   - Internal-enterprise ingest (API/batch import), plus DIMSE receive support via sidecar.
3. Core architecture:
   - Go API (`api/`) for orchestration, routing, auth, DICOMweb proxy, exports.
   - PostgreSQL for app state/audit.
   - Shared DICOM storage (`raw` + `clean`) on local/S3/GCS.
   - 7 Python sidecars: defacing, PHI detection, QC, BIDS, classification, protocol, DIMSE receiver.
   - React apps: upload portal, admin dashboard, export portal, landing.
4. Processing pipeline is dependency-ordered:
   - Phase 0: classification
   - Phase 1: PHI + protocol + defacing (parallel)
   - Phase 2: QC + BIDS
5. Product and architecture docs state that Phases 1–4 are code-complete; active focus is deployment hardening, enterprise integrations, and continued feature expansion.

## Root Markdown Source Map

Use these files first before coding:

1. `CLAUDE.md`: primary engineering source of truth (architecture details, env vars, endpoints, CI/test expectations, workflow rules).
2. `AEGIS_Architecture.md`: system-level design, phased roadmap, security and data-flow model.
3. `SETUP_CHECKLIST.md`: operational runbook for local/cloud setup and manual verification steps.
4. `README.md`: concise product and repo overview.
5. `GCP_UPLOAD_NOTES.md`: GCP-specific upload runbook (`STORAGE_MODE=gcs`).
6. `AEGIS_Executive_Summary.md`: business framing, market context, references.

## Documentation Non-Negotiables

1. If markdown docs change, regenerate required PDFs in the same feature/PR.
2. Keep technical docs updated with shipped behavior (`CLAUDE.md`, `SETUP_CHECKLIST.md`, and architecture docs when design changes).
3. Use `docs/` as the research knowledge base; never store PHI/CBI in repo docs.

## Branching and PR Flow

1. Start every feature from `develop`:
   - `git checkout -b feature/<name> origin/develop`
2. Do all implementation on that feature branch (never directly on `develop` or `main`).
3. Validate changes (targeted tests first, then broader suites as needed).
4. Update docs for shipped behavior (`CLAUDE.md`, `SETUP_CHECKLIST.md`, and architecture docs when applicable).
5. Regenerate required PDFs when markdown changes (commit markdown + PDF together).
6. Commit with a clear message.
7. Push branch and open PR to `develop`:
   - `gh pr create --base develop --head feature/<name> ...`
8. Share the clickable PR link with Matt for review.
9. After merge, return to `develop` and pull latest.

## Quality Guardrails

1. Do not revert unrelated user changes.
2. Avoid destructive git/file commands unless explicitly requested.
3. Keep changes scoped to the requested feature.
4. Prefer reproducible, automated verification over manual claims.

## Next Feature Queue

Use this as the default execution order unless priorities change. Detailed scope/acceptance/test plans live in `docs/planning/next-feature-shortlist-sprint-ready.md`.
Next session start point (when unblocked): begin with `feature/terraform-gcp-prod-completion`.

1. `feature/terraform-gcp-prod-completion`
   - Complete GCP production Terraform module (Cloud Run services, Artifact Registry, networking, Cloud Armor, IAP, monitoring).
2. `feature/secrets-hardening-infra`
   - Remove static credential placeholders and enforce secret-manager patterns for deployed environments.
3. `feature/cloud-smoke-test-suite`
   - Implement automated cloud smoke tests for auth, health, upload, pipeline, and export baseline.
4. `feature/dimse-pacs-e2e-validation`
   - Build DIMSE PACS end-to-end validation harness + runbook (success, retry, dead-letter, recovery flows).
5. `feature/terraform-aws-https-cognito`
   - Complete AWS edge/auth hardening (ACM HTTPS listener + Cognito integration).

After this top-5 queue:
6. `feature/dimse-retry-durable-store`
7. `feature/admin-dimse-ops-panel`
8. `feature/dimse-alerting-thresholds`
9. `feature/study-diagnostics-summary`
10. `feature/dev-ci-parity-lint-checks`
11. `feature/mcp-server-readonly-mvp`
12. `feature/mcp-server-guarded-write-tools`
