# AEGIS MCP Server (Scaffold)

Minimal MCP API-wrapper scaffold for Anonymization & Exchange Gateway for Imaging Studies (AEGIS).

## Current status

- Read tools implemented and wired to AEGIS API:
  - `list_studies`
  - `get_study_detail`
  - `get_study_diagnostics`
  - `get_study_audit`
  - `get_study_routing_log`
  - `list_export_shares`
  - `get_system_health`
- Write tools:
  - Implemented (with operator mode + feature-flag guard + preconditions):
    - `trigger_classification`
    - `trigger_bids_convert`
    - `trigger_export`
    - `trigger_deface`
    - `trigger_qc_check`
    - `trigger_protocol_check`
    - `trigger_phi_scan`
    - `retry_dimse_study`

## Environment

Required:
- `AEGIS_API_BASE_URL` (example: `http://localhost:8080`)
- `AEGIS_API_TOKEN` (service credential used by MCP when calling API)

Optional:
- `MCP_MODE=readonly|operator` (default `readonly`)
- `MCP_ENABLE_WRITE_TOOLS=true|false` (default `false`)

## Run locally

```bash
cd mcp-server
npm install
npm run dev
```

Build:

```bash
npm run build
npm run start
```

## Notes

- This is an implementation scaffold matching:
  - `docs/research/mcp-api-wrapper-mvp-spec.md`
  - `docs/research/mcp-tool-schemas-v1.json`
  - `docs/research/mcp-threat-model.md`
- `trigger_qc_check` resolves study UID via `list_studies` search, enforces QC preconditions, then calls `POST /api/studies/{studyUID}/qc-check`.
- `trigger_protocol_check` resolves study UID via `list_studies` search, enforces protocol preconditions, then calls `POST /api/studies/{studyUID}/protocol-check`.
- `trigger_phi_scan` resolves study UID via `list_studies` search, enforces PHI-scan preconditions, then calls `POST /api/studies/{studyUID}/phi-scan`.
- `trigger_classification` resolves study UID via `list_studies` search, enforces classification preconditions, then calls `POST /api/studies/{studyUID}/classify`.
- `trigger_bids_convert` resolves study UID via `list_studies` search, enforces BIDS preconditions, then calls `POST /api/studies/{studyUID}/bids-convert`.
- `trigger_export` resolves study UID via `list_studies` search, enforces approved/export preconditions, then calls `POST /api/studies/{studyUID}/trigger-export`.
- `trigger_deface` resolves study UID via `list_studies` search, enforces defacing preconditions, then calls `POST /api/deface/{studyUID}`.
- `retry_dimse_study` inspects `/api/dimse/retry/details` and then targets either `POST /api/dimse/retry/process/{studyUID}` or `POST /api/dimse/retry/replay/{studyUID}`.
- `get_study_diagnostics` calls `GET /api/studies/{study_id}/diagnostics` for "why stuck" summary output.
- All currently registered write tools execute with operator/feature-flag guards and endpoint precondition checks.
