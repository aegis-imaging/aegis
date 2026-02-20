# MCP API-Wrapper MVP Spec for Anonymization & Exchange Gateway for Imaging Studies (AEGIS)

_Date: 2026-02-20_
_Status: Draft v0.1 (implementation-oriented)_

Companion artifacts:
- `docs/research/mcp-tool-schemas-v1.json`
- `docs/research/mcp-threat-model.md`

## 1) Scope

This spec defines a **minimal MCP server** that wraps existing AEGIS API endpoints and exposes safe, operator-focused tools for AI assistants.

### Goals
- Reduce operator time spent diagnosing and unblocking studies.
- Reuse existing API auth, RBAC, and audit behavior.
- Keep PHI/PII exposure minimized in tool responses.

### Non-goals (MVP)
- No direct sidecar/database access from MCP.
- No bulk mutating actions.
- No autonomous multi-study write workflows.
- No external system integrations (Jira/Slack/SIEM) in MVP.

## 2) Architecture

### Deployment pattern
- Deploy MCP as a **separate service** (`mcp-server`) in the same private network boundary as `api`.
- MCP communicates only with the Go API over HTTPS.

### Request flow
1. MCP client invokes tool.
2. MCP validates tool args against strict schema.
3. MCP checks tool policy (read-only vs write-capable).
4. MCP calls mapped AEGIS endpoint.
5. MCP returns a normalized response.
6. MCP emits MCP-level audit log.

### Why separate service first
- Clear trust boundary and explicit policy layer.
- Easier to disable/rollback independently.
- Avoid coupling MCP runtime concerns to API release cadence.

## 3) Identity, Auth, and RBAC

## Caller identities
- `mcp-readonly`: may call read tools only.
- `mcp-operator`: may call read + write tools.

### Recommended auth model
- MCP authenticates to AEGIS API using a service credential.
- MCP enforces its own per-tool allowlist by caller identity.
- For user-delegated use, pass user context as metadata (`requested_by`) for audit only; do not bypass API RBAC.

### Hard rules
- Write tools require explicit confirmation token (`confirm=true`).
- Single-study scope only for all write tools in MVP.
- Reject wildcard/bulk selectors in write tool inputs.

## 4) Audit requirements

Each MCP tool invocation logs:
- `timestamp_utc`
- `request_id`
- `caller_id` (service principal)
- `requested_by` (optional user context)
- `tool_name`
- `tool_args_redacted`
- `result_status` (`ok|error|denied`)
- `aegis_endpoint`
- `aegis_http_status`
- `reason` (required for write tools)

Write tools should also include an `audit_note` in outbound API calls where feasible.

## 5) Tool contract conventions

### Common input fields
- `request_id: string` (UUID recommended)
- `reason: string` (required for write tools; min length 10)
- `confirm: boolean` (required true for write tools)

### Common output envelope
```json
{
  "ok": true,
  "request_id": "...",
  "tool": "...",
  "data": { },
  "warnings": []
}
```

### Error envelope
```json
{
  "ok": false,
  "request_id": "...",
  "tool": "...",
  "error": {
    "code": "VALIDATION_ERROR|AUTH_ERROR|FORBIDDEN|NOT_FOUND|CONFLICT|UPSTREAM_ERROR|TIMEOUT",
    "message": "human-readable",
    "retryable": false
  }
}
```

## 6) MVP tools and endpoint map

## Read tools

### 6.1 `list_studies`
Purpose: List studies with filters for queue triage.

Input:
```json
{
  "limit": 50,
  "offset": 0,
  "project_id": "uuid?",
  "status": "received|defacing|clean|defaced|approved|rejected?",
  "modality": "MRI|CT|PET?",
  "source": "external|internal?",
  "search": "string?"
}
```
Mapping:
- `GET /api/studies?limit=&offset=&project_id=&status=&modality=&source=&search=`

### 6.2 `get_study_detail`
Purpose: Retrieve one study for diagnosis context.

Input:
```json
{ "study_id": "uuid" }
```
Mapping:
- `GET /api/studies/{id}`

### 6.3 `get_study_audit`
Purpose: Explain timeline and failure reasons.

Input:
```json
{ "study_id": "uuid" }
```
Mapping:
- `GET /api/studies/{id}/audit`

### 6.4 `get_study_routing_log`
Purpose: Explain which routing rules fired.

Input:
```json
{ "study_id": "uuid" }
```
Mapping:
- `GET /api/studies/{studyID}/routing-log`

### 6.5 `list_export_shares`
Purpose: Check active/expired/revoked shares.

Input:
```json
{ "study_id": "uuid" }
```
Mapping:
- `GET /api/studies/{id}/shares`

### 6.6 `get_system_health`
Purpose: Quick operational snapshot.

Input:
```json
{}
```
Mapping:
- `GET /healthz`

## Write tools (single-study only)

All write tools require:
```json
{
  "study_uid": "string",
  "reason": "string (>=10 chars)",
  "confirm": true
}
```

### 6.7 `trigger_classification`
- `POST /api/studies/{studyUID}/classify`

### 6.8 `trigger_phi_scan`
- `POST /api/studies/{studyUID}/phi-scan`

### 6.9 `trigger_protocol_check`
- `POST /api/studies/{studyUID}/protocol-check`

### 6.10 `trigger_qc_check`
- `POST /api/studies/{studyUID}/qc-check`

### 6.11 `trigger_bids_convert`
- `POST /api/studies/{studyUID}/bids-convert`

### 6.12 `trigger_export`
- `POST /api/studies/{studyUID}/trigger-export`

### 6.13 `trigger_deface`
- `POST /api/studies/{studyUID}/trigger-deface`

## Optional write tool (if MCP can reach DIMSE operator API)

### 6.14 `retry_dimse_study`
Input:
```json
{
  "study_instance_uid": "string",
  "reason": "string (>=10 chars)",
  "confirm": true
}
```
Mapping:
- `POST /ingest/retry/process/{study_instance_uid}` (dimse-receiver)

Note: Keep disabled by default unless operator key management is in place.

## 7) Response shaping and PHI safety

### Redaction policy (MVP)
- Do not include DICOM pixel-derived content.
- Do not return uploader emails by default in MCP responses.
- Include only fields needed for operations: IDs, statuses, timestamps, modality/body part, service states.

### Explainability requirement
For all “why stuck?” style prompts, MCP response should include:
- current study status fields
- last relevant audit events (bounded list)
- explicit next recommended action (readable text)

## 8) Guardrails and safety checks

Before any write call:
1. Validate `confirm=true`.
2. Validate `reason` present and meaningful.
3. Validate one and only one study target.
4. Optionally fetch study detail first; reject if state makes action invalid (for clearer errors).

### Suggested preconditions (soft-enforced)
- `trigger_export`: study should be `approved` and export required/pending/failed.
- `trigger_bids_convert`: bids status should be pending/failed.
- `trigger_qc_check`: qc status should be pending/failed.

If precondition fails, return `CONFLICT` with actionable message.

## 9) Minimal implementation blueprint

### Tech choice
- TypeScript Node MCP server (fastest team fit with frontend/client TS stack).

### Internal modules
- `auth.ts` — caller auth + role mapping
- `policy.ts` — tool allowlist and guardrails
- `schemas.ts` — zod/json-schema tool validation
- `aegisApi.ts` — typed API client wrappers
- `redaction.ts` — output shaping + safe fields
- `audit.ts` — structured MCP invocation logs

### Config env vars
- `AEGIS_API_BASE_URL`
- `AEGIS_API_TOKEN` (or equivalent service credential)
- `MCP_MODE=readonly|operator`
- `MCP_ENABLE_DIMSE_TOOLS=false`
- `DIMSE_BASE_URL` (optional)
- `DIMSE_OPERATOR_KEY` (optional)

## 10) Rollout plan

### Phase A (1 week)
- Implement read tools 6.1–6.6.
- Add structured logging + redaction.
- Validate against local docker-compose stack.

### Phase B (1 week)
- Implement write tools 6.7–6.13 with confirmation + reason.
- Add precondition checks and conflict messaging.

### Phase C (optional)
- Enable 6.14 DIMSE retry tool behind feature flag.
- Add metrics dashboard and SLOs.

## 11) MVP acceptance criteria

- Read tools return consistent envelopes and pass schema validation.
- Write tools are denied without `confirm=true` and `reason`.
- No write tool accepts multi-study selection.
- Every tool invocation emits MCP audit log entry.
- Tool failures map to normalized error codes.
- Smoke-tested against at least one real study in local stack.

## 12) Metrics to track

- `mcp_tool_calls_total{tool,result}`
- `mcp_tool_latency_ms{tool}`
- `mcp_write_denied_total{reason}`
- `mcp_precondition_conflicts_total{tool}`
- Operational outcome: median time to resolve “stuck study” incidents.

## 13) Open decisions

- Whether MCP should pass through end-user identity to API in a signed header for deeper attribution.
- Whether `trigger_deface` belongs in MVP if head-study policy already auto-dispatches correctly.
- Whether to include read-only tools for destinations/routing rules in v0.1 or v0.2.

## 14) Next document to produce

- `docs/research/mcp-tool-schemas-v1.json` with exact JSON Schemas for each tool input/output.
- `docs/research/mcp-threat-model.md` with abuse cases and mitigations.
