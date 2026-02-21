# MCP Server Layering Ideas for Anonymization & Exchange Gateway for Imaging Studies (AEGIS)

_Date: 2026-02-20_
_Status: Working notes for iteration_

## TL;DR
Yes — layering an MCP server on top of AEGIS makes sense, especially for **operator productivity, controlled automation, and cross-system orchestration**. The key is to treat MCP as a **policy-constrained control plane**, not a direct bypass around existing API/auth/audit rules.

Related implementation draft: `docs/research/mcp-api-wrapper-mvp-spec.md`.

## Why this could be useful
- AEGIS already has many operational actions spread across API endpoints and admin UI (ingest, routing, QC, protocol, PHI scan, export, digests, users, institutions).
- MCP can provide a single “tooling interface” for AI assistants and internal automation to perform these actions in a safe, auditable way.
- It can reduce click-heavy admin workflows and speed up triage.

## High-value use cases (ranked)

### 1) Study Operations Copilot (highest value)
AI-assisted operations over approved tool actions:
- Find studies stuck in `pending` for any pipeline stage.
- Trigger retries for specific services (classification, PHI scan, protocol, QC, BIDS, export).
- Summarize why a study is blocked by reading audit + routing log.
- Draft a recommended action plan for an operator (without auto-executing unless approved).

Why good:
- Immediate time savings for admins.
- Uses existing APIs and audit data.
- Low product risk if write actions require explicit confirmation.

### 2) Routing Rule Assistant
- Explain which routing rules matched a given study and why.
- Simulate rule changes against recent studies (“what would have happened?”).
- Propose simplified rule sets (remove duplicates/conflicts) and output a change diff.

Why good:
- Routing complexity grows fast.
- Clear ROI in fewer misroutes and less manual troubleshooting.

### 3) Export Compliance Assistant
- List shares that are near expiry, expired, or revoked.
- Detect unusual export patterns (e.g., repeated share creation for same study).
- Generate plain-language weekly/monthly compliance summaries for admins.

Why good:
- Strong fit with existing export + digest capabilities.
- Helps governance without exposing PHI in output.

### 4) DIMSE Ingest Recovery Assistant
- Inspect retry/dead-letter queues in `dimse-receiver`.
- Recommend replay/clear actions with rationale.
- Perform targeted retry actions for specific StudyInstanceUIDs.

Why good:
- Operationally painful area today.
- MCP maps naturally to existing retry control endpoints.

### 5) Protocol/Quality Review Assistant
- Summarize protocol deviations and QC findings into concise “pass/warn/fail + reason” notes.
- Identify recurrent scanner/model drift trends by project.

Why good:
- Converts verbose machine findings into operator-friendly summaries.

## Architecture patterns to consider

### Pattern A: API Wrapper MCP (recommended first)
- MCP server calls existing Go API endpoints only.
- Reuses current auth + RBAC + audit model.
- Fastest to deliver and lowest risk.

### Pattern B: Orchestrator MCP
- MCP server coordinates API + sidecar endpoints + optional external systems (ticketing, SIEM, Slack/Teams).
- Better for incident response and cross-tool workflows.
- More complex trust boundaries.

### Pattern C: Read-only Analytics MCP
- Focuses on querying and summarizing state (studies, audit, exports, routing outcomes).
- No mutating tools.
- Safest starting point for production rollout.

## Suggested first tool set (MVP)
Start read-heavy with carefully scoped writes:

Read tools:
- `list_studies(filters)`
- `get_study_detail(study_id)`
- `get_study_audit(study_id)`
- `get_study_routing_log(study_id)`
- `list_export_shares(study_id)`
- `get_pipeline_health()`
- `get_dimse_retry_status()`

Write tools (explicit confirmation required):
- `trigger_classification(study_uid)`
- `trigger_phi_scan(study_uid)`
- `trigger_protocol_check(study_uid)`
- `trigger_qc_check(study_uid)`
- `trigger_bids_convert(study_uid)`
- `trigger_export(study_uid)`
- `retry_dimse_study(study_uid)`

## Security and governance guardrails (non-negotiable)
- Enforce least privilege: separate read-only and admin-write MCP credentials.
- Keep all write actions behind explicit confirmation in the client/agent UX.
- Preserve and enrich audit trail (`actor`, `tool_name`, `request_id`, `reason`).
- Block PHI in generated summaries by policy (same no-PHI email/digest principle).
- Rate-limit mutating tools and add idempotency keys where practical.
- Restrict bulk operations unless explicitly enabled.

## Risks / failure modes
- Over-automation causing accidental mass actions (e.g., triggering exports broadly).
- Prompt ambiguity leading to wrong scope (“all pending” vs “this study only”).
- Security drift if MCP auth bypasses current API middleware.
- Operator trust issues if assistant explanations are not tied to concrete logs.

Mitigations:
- Safe defaults (`dry_run=true` behavior where possible).
- Scope confirmation (“I will trigger QC for 1 study: <UID>”).
- Strong tool schemas and strict argument validation.
- Response templates that cite source objects (study ID, rule ID, audit event names).

## Incremental rollout plan
1. **Phase 0 (internal-only, read-only):** state inspection + summarization tools.
2. **Phase 1 (guarded writes):** single-study trigger tools with mandatory confirmation.
3. **Phase 2 (workflow macros):** constrained multi-step runbooks (e.g., “unstick study pipeline”).
4. **Phase 3 (cross-system):** optional integrations (ticketing/alerts/chatops) with strict policy controls.

## Good candidate workflows to prototype first
- “Why is this study stuck?”
- “Show me all studies pending >24h and suggest next action per study.”
- “For this study, rerun only the missing required stages in correct order.”
- “Summarize export activity this week by project with no PHI.”

## Open questions for next iteration
- Should MCP live inside `api/` (same deploy unit) or as a separate service?
- Do we want a hard separation between read and write MCP servers?
- Which identity model should be primary for MCP callers (service account, user-delegated, both)?
- Should “bulk tools” exist at all in MVP, or be deferred?
- What is the minimum audit format needed for compliance review of MCP actions?

## Notes to self
- Keep MCP as an abstraction over stable API contracts, not sidecar internals.
- Prioritize explainability and reversibility over aggressive automation.
- Start with 3-5 workflows that are painful today and measurable (time-to-resolution, stuck-study count, manual clicks avoided).
