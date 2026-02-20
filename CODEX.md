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
