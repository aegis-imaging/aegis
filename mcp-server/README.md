# AEGIS MCP Server (Scaffold)

Minimal MCP API-wrapper scaffold for Anonymization & Exchange Gateway for Imaging Studies (AEGIS).

## Current status

- Read tools implemented and wired to AEGIS API:
  - `list_studies`
  - `get_study_detail`
  - `get_study_audit`
  - `get_study_routing_log`
  - `list_export_shares`
  - `get_system_health`
- Write tools:
  - Implemented (with operator mode + feature-flag guard + preconditions):
    - `trigger_qc_check`
    - `trigger_protocol_check`
    - `trigger_phi_scan`
  - Registered but currently stubbed:
  - `trigger_classification`
  - `trigger_bids_convert`
  - `trigger_export`
  - `trigger_deface`
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
- All other write tools currently validate input (`confirm` + `reason`) and return guarded denial/not-implemented responses by design.
