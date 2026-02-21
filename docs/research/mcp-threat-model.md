# MCP Threat Model for Anonymization & Exchange Gateway for Imaging Studies (AEGIS)

_Date: 2026-02-20_
_Status: Draft v0.1_
_Scope: MCP API-wrapper MVP defined in `docs/research/mcp-api-wrapper-mvp-spec.md` and schemas in `docs/research/mcp-tool-schemas-v1.json`_

## 1) Objective

Identify realistic abuse and failure modes for an MCP layer on top of AEGIS, then define practical controls for MVP and near-term hardening.

## 2) System and trust boundaries

### In-scope components
- MCP server (`mcp-server`) exposing tool calls.
- Go API (`api`) endpoints invoked by MCP.
- Optional DIMSE operator endpoints (`dimse-receiver /ingest/retry*`).
- MCP caller identities (`mcp-readonly`, `mcp-operator`).

### Trust boundaries
1. MCP client ↔ MCP server
2. MCP server ↔ AEGIS API
3. MCP server ↔ DIMSE operator endpoints (optional)
4. Human prompt intent ↔ concrete tool arguments (translation risk)

### Security assumptions
- API is already protected by auth middleware and RBAC.
- MCP service credentials are provisioned securely (secret manager).
- Network path between MCP and API is private and encrypted.

## 3) Assets to protect

- Study lifecycle integrity (correct pipeline actions, no unintended triggers).
- Export control integrity (no unauthorized forwarding/export actions).
- Audit integrity and non-repudiation.
- Availability of operations plane (MCP and API).
- Confidential metadata (even if no pixel data is returned).

## 4) Threat actors

- External attacker with no valid credentials.
- Internal authenticated caller with insufficient privileges.
- Compromised MCP client/session.
- Well-intentioned operator issuing ambiguous natural-language requests.
- Malicious insider attempting mass or stealth actions.

## 5) Primary threats (STRIDE-style)

## Spoofing
### T1: Caller identity spoofing to obtain write access
- Attack: forge/steal token for `mcp-operator`.
- Impact: unauthorized mutating actions.
- MVP controls:
  - Strong token validation + issuer/audience checks.
  - Separate credentials for read vs write caller classes.
  - Short token TTL and key rotation.
- Residual risk: Medium.

## Tampering
### T2: Tool argument tampering (scope escalation)
- Attack: mutate single-study argument into bulk/wildcard behavior.
- Impact: accidental or malicious broad changes.
- MVP controls:
  - Strict JSON schema validation.
  - Reject unknown fields (`additionalProperties=false`).
  - Write tools require exactly one target (`study_uid` only).
- Residual risk: Low.

### T3: Replay of mutating requests
- Attack: resend identical write calls repeatedly.
- Impact: duplicate dispatches/noise/instability.
- MVP controls:
  - Require `request_id` for correlation.
  - Add idempotency cache (`request_id`, `tool`, `target`, TTL).
  - Rate-limit mutating endpoints per caller/target.
- Residual risk: Medium until idempotency fully implemented.

## Repudiation
### T4: Operator denies having initiated an action
- Attack: insufficient attribution in logs.
- Impact: compliance and incident response gaps.
- MVP controls:
  - Immutable structured logs with `caller_id`, optional `requested_by`, `tool_name`, `args_redacted`, `result`, `request_id`.
  - UTC timestamps and downstream API status.
- Residual risk: Low.

## Information disclosure
### T5: Metadata overexposure in tool responses
- Attack: prompt asks for broad data; tool returns unnecessary fields (emails/notes).
- Impact: confidentiality breach.
- MVP controls:
  - Response shaping allowlist by tool.
  - Exclude uploader email and note text by default.
  - No pixel-derived content ever returned by MCP.
- Residual risk: Medium (depends on strict response mapping discipline).

### T6: Prompt-injection through untrusted data reflected into operator context
- Attack: crafted metadata in study descriptions/audit fields influences agent behavior.
- Impact: unsafe follow-on actions.
- MVP controls:
  - Treat all upstream strings as untrusted.
  - Do not execute text from records as commands.
  - Require explicit confirmation + `reason` before writes.
- Residual risk: Medium.

## Denial of service
### T7: Tool call floods degrade API or MCP
- Attack: high-rate read/write bursts.
- Impact: control-plane slowdown/outage.
- MVP controls:
  - Per-caller rate limits and concurrency caps.
  - Circuit breaker/timeouts/retries with backoff.
  - Read pagination enforced (`limit <= 200`).
- Residual risk: Medium.

### T8: Expensive chained workflows create cascading load
- Attack: repeated trigger calls on already-running stages.
- Impact: queue pressure/noise.
- MVP controls:
  - Precondition checks (pending/failed-only where appropriate).
  - Return `CONFLICT` for in-progress states.
- Residual risk: Low-Medium.

## Elevation of privilege
### T9: Readonly caller invokes write tools via MCP bug
- Attack: policy bypass in tool routing.
- Impact: unauthorized mutations.
- MVP controls:
  - Central policy gate before tool handler execution.
  - Defense-in-depth: API RBAC still enforced.
  - Security test cases for every write tool using readonly identity.
- Residual risk: Low.

### T10: MCP credential has broader API privileges than intended
- Attack: compromised MCP can access endpoints outside tool scope.
- Impact: blast-radius increase.
- MVP controls:
  - Dedicated API credential with minimal permissions.
  - Outbound allowlist in MCP client (`method + path` constraints).
- Residual risk: Medium.

## 6) Top risks and priorities

Priority is based on likelihood × impact for MVP:
1. T1 identity spoofing / token misuse
2. T3 replay and duplicate writes
3. T5 metadata overexposure
4. T7 request flood / availability degradation
5. T10 over-privileged MCP credential

## 7) Control matrix (threat → required control)

- T1 → token verification, short TTL, key rotation, mTLS/private network
- T2 → strict schemas, reject unknown fields, single-target write schema
- T3 → idempotency keying + request dedupe + rate limits
- T4 → immutable structured logs + centralized retention
- T5 → response field allowlists + redaction tests
- T6 → untrusted text handling + mandatory confirmation for writes
- T7 → rate limits, concurrency caps, timeout budgets, circuit breaker
- T8 → state preconditions and conflict responses
- T9 → central authz gate + negative tests
- T10 → least-privilege service principal + endpoint allowlist

## 8) Security requirements for MVP go-live

MUST:
- Enforce per-tool authorization for caller role.
- Enforce schema validation with `additionalProperties=false`.
- Require `confirm=true` and `reason` for all write tools.
- Emit structured audit logs for every invocation.
- Implement rate limiting for read and write classes.
- Implement response allowlists to prevent overexposure.

SHOULD:
- Implement idempotency cache for write operations.
- Add outbound endpoint allowlist to MCP HTTP client.
- Add anomaly alerts (write denial spikes, unusual write volume).

## 9) Security testing plan

### Unit tests
- Reject malformed tool inputs and unknown fields.
- Reject write calls without `confirm=true` or without `reason`.
- Verify readonly identity is denied on every write tool.
- Verify redaction of sensitive fields in each response mapper.

### Integration tests
- MCP → API happy path for each tool.
- API failure mapping to normalized error envelope.
- Retry + timeout behavior under upstream degradation.

### Abuse tests
- Replay same mutating request rapidly with same and different `request_id`.
- Attempt path/method outside allowlist.
- Attempt over-limit pagination and bulk-like selectors.

## 10) Operational monitoring

Suggested metrics:
- `mcp_auth_fail_total{reason}`
- `mcp_tool_calls_total{tool,result}`
- `mcp_write_calls_total{tool}`
- `mcp_write_denied_total{reason}`
- `mcp_idempotency_deduped_total{tool}`
- `mcp_response_redaction_events_total{tool}`
- `mcp_upstream_latency_ms{endpoint}`

Alert examples:
- Sudden spike in `mcp_write_calls_total` outside business hours.
- Repeated `FORBIDDEN` attempts from one caller.
- Elevated `UPSTREAM_ERROR`/`TIMEOUT` for health and study endpoints.

## 11) Incident response playbook (minimum)

1. Disable write tools globally via feature flag.
2. Rotate MCP credentials.
3. Review audit trail by `request_id` and caller.
4. Reconstruct affected study IDs and triggered actions.
5. Apply remediation runbook (e.g., pause exports, rerun safe stages).
6. Produce post-incident control update and test case.

## 12) Deferred hardening (post-MVP)

- Per-tool signed approvals for high-risk actions (e.g., export triggers).
- Dual-control for selected write tools.
- User-delegated auth with end-user identity propagation and cryptographic attestation.
- Formal policy engine (OPA/Cedar) for fine-grained authz decisions.
