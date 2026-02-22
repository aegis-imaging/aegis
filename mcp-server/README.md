# AEGIS MCP Server

MCP API-wrapper server for Anonymization & Exchange Gateway for Imaging Studies (AEGIS).

Implements the tools defined in `docs/research/mcp-api-wrapper-mvp-spec.md` and
`docs/research/mcp-tool-schemas-v1.json`.

## Tools

### Read tools (always available)

| Tool | Endpoint | Purpose |
|------|----------|---------|
| `list_studies` | `GET /api/studies` | List studies with filters for triage |
| `get_study_detail` | `GET /api/studies/{id}` | Full detail for one study |
| `get_study_diagnostics` | `GET /api/studies/{id}/diagnostics` | "Why stuck?" summary |
| `get_study_audit` | `GET /api/studies/{id}/audit` | Audit trail for one study |
| `get_study_routing_log` | `GET /api/studies/{id}/routing-log` | Routing evaluation log |
| `list_export_shares` | `GET /api/studies/{id}/shares` | Active/expired/revoked shares |
| `get_system_health` | `GET /healthz` | System health snapshot |

### Write tools (require `MCP_MODE=operator` + `MCP_ENABLE_WRITE_TOOLS=true`)

| Tool | Endpoint | Precondition checks |
|------|----------|-------------------|
| `trigger_classification` | `POST /api/studies/{uid}/classify` | `classification_required`, not already running |
| `trigger_phi_scan` | `POST /api/studies/{uid}/phi-scan` | `phi_scan_required`, not already scanning |
| `trigger_protocol_check` | `POST /api/studies/{uid}/protocol-check` | `protocol_required`, not already checking |
| `trigger_qc_check` | `POST /api/studies/{uid}/qc-check` | `qc_required`, not already checking |
| `trigger_bids_convert` | `POST /api/studies/{uid}/bids-convert` | `bids_required`, not already converting |
| `trigger_export` | `POST /api/studies/{uid}/trigger-export` | Study approved, export required |
| `trigger_deface` | `POST /api/studies/{uid}/trigger-deface` | `defacing_required`, not terminal status |
| `retry_dimse_study` | `POST /api/dimse/retry/process/{uid}` or `.../replay/{uid}` | Checks pending/dead-letter queue first |

All write tools require `confirm: true` and a `reason` (≥10 chars) in the input.

## Environment variables

Required:
```
AEGIS_API_BASE_URL    Base URL of the AEGIS API (e.g. http://localhost:8080)
AEGIS_API_TOKEN       Service credential passed as Bearer token to API
```

Optional:
```
MCP_MODE                          readonly (default) | operator
MCP_ENABLE_WRITE_TOOLS            false (default) | true
MCP_READ_RATE_LIMIT_PER_MINUTE    240 (default)
MCP_WRITE_RATE_LIMIT_PER_MINUTE   60 (default)
MCP_WRITE_IDEMPOTENCY_TTL_SECONDS 900 (default)
MCP_CALLER_ID                     Label for audit logs (default: mcp-stdio)
```

See `.env.example` for annotated defaults.

## Run locally (stdio transport)

```bash
cd mcp-server
npm install
npm run dev
```

## Build and run compiled

```bash
cd mcp-server
npm run build
npm start
```

## Claude Desktop integration

Copy the template from `claude-desktop-config-example.json` into your
`claude_desktop_config.json` under `mcpServers`. Update the `args` path and
`env` values to match your environment.

## Docker

```bash
# Build
docker build -t aegis-mcp-server mcp-server/

# Run (read-only mode)
docker run --rm -i \
  -e AEGIS_API_BASE_URL=http://api:8080 \
  -e AEGIS_API_TOKEN=your-token \
  aegis-mcp-server
```

With docker-compose (starts alongside the rest of the stack, requires `--profile mcp`):

```bash
MCP_API_TOKEN=your-token docker compose --profile mcp up mcp-server
```

## Tests

```bash
npm test          # unit tests (aegisClient path allowlist + redaction)
npm run typecheck # TypeScript strict type check
```

## Security notes

- All outbound API paths are allowlisted in `src/aegisClient.ts`; any unexpected
  path throws `DisallowedPathError` before the network call is made.
- Write tools are disabled by default. Enable only after reviewing
  `docs/research/mcp-threat-model.md`.
- Sensitive fields (tokens, emails, reasons) are redacted from invocation logs.
