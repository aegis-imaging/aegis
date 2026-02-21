import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema, Tool } from "@modelcontextprotocol/sdk/types.js";
import { AegisApiClient, DisallowedPathError, UpstreamHttpError } from "./aegisClient.js";
import { loadConfig } from "./config.js";
import { redactToolArgs } from "./redaction.js";
import {
  emptyArgsSchema,
  listStudiesArgsSchema,
  readToolNames,
  retryDimseArgsSchema,
  studyIdArgsSchema,
  ToolName,
  writeArgsSchema,
  writeToolNames
} from "./schemas.js";

type ToolPayload = {
  ok: boolean;
  request_id: string;
  tool: ToolName | "unknown";
  data?: unknown;
  warnings?: string[];
  error?: {
    code:
      | "VALIDATION_ERROR"
      | "AUTH_ERROR"
      | "FORBIDDEN"
      | "NOT_FOUND"
      | "CONFLICT"
      | "UPSTREAM_ERROR"
      | "TIMEOUT"
      | "RATE_LIMITED";
    message: string;
    retryable: boolean;
  };
};

type ErrorCode =
  | "VALIDATION_ERROR"
  | "AUTH_ERROR"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "UPSTREAM_ERROR"
  | "TIMEOUT"
  | "RATE_LIMITED";
type ToolResponse = {
  isError?: boolean;
  content: Array<{ type: "text"; text: string }>;
};

type ToolClass = "read" | "write";

type InvocationLogResult = "success" | "error" | "idempotent";

type InvocationLog = {
  event: "mcp_tool_invocation";
  ts: string;
  request_id: string;
  caller_id: string;
  requested_by?: string;
  tool_name: ToolName | "unknown";
  tool_class: ToolClass | "unknown";
  args_redacted: Record<string, unknown>;
  result: InvocationLogResult;
  error_code?: ErrorCode;
  duration_ms: number;
};
type StudySummary = {
  id: string;
  status?: string;
  study_instance_uid?: string;
  defacing_required?: boolean;
  classification_required?: boolean;
  classification_status?: string;
  bids_required?: boolean;
  bids_status?: string;
  export_required?: boolean;
  export_status?: string;
  qc_required?: boolean;
  qc_status?: string;
  protocol_required?: boolean;
  protocol_status?: string;
  phi_scan_required?: boolean;
  phi_scan_status?: string;
};

type ListStudiesResponse = {
  studies?: StudySummary[];
  total?: number;
  limit?: number;
  offset?: number;
};

type DimseRetryDetails = {
  pending_total?: number;
  dead_letter_total?: number;
};

type DimseRetryDetailsResponse = {
  ingest_retry?: DimseRetryDetails;
};

const writeInputSchema: Tool["inputSchema"] = {
  type: "object",
  required: ["study_uid", "reason", "confirm"],
  properties: {
    request_id: { type: "string" },
    study_uid: { type: "string", pattern: "^[0-9.]+$" },
    reason: { type: "string", minLength: 10, maxLength: 512 },
    confirm: { type: "boolean", const: true }
  },
  additionalProperties: false
};

const config = loadConfig();
const client = new AegisApiClient(config.aegisApiBaseUrl, config.aegisApiToken);

const tools: Tool[] = [
  {
    name: "list_studies",
    description: "List studies with filters for operations triage.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        limit: { type: "integer", minimum: 1, maximum: 200 },
        offset: { type: "integer", minimum: 0 },
        project_id: { type: "string", format: "uuid" },
        status: { type: "string", enum: ["received", "defacing", "clean", "defaced", "approved", "rejected"] },
        modality: { type: "string" },
        source: { type: "string", enum: ["external", "internal"] },
        search: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_detail",
    description: "Get detail for a specific study UUID.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_diagnostics",
    description: "Get per-study diagnostics summary for stuck-state triage.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_audit",
    description: "Get audit entries for a study UUID.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_routing_log",
    description: "Get routing evaluation log for a study UUID.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_export_shares",
    description: "List export shares for a study UUID.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_system_health",
    description: "Get AEGIS system health snapshot from /healthz.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "trigger_classification",
    description: "Trigger metadata classification for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_phi_scan",
    description: "Trigger PHI scan for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_protocol_check",
    description: "Trigger protocol compliance check for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_qc_check",
    description: "Trigger QC check for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_bids_convert",
    description: "Trigger BIDS conversion for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_export",
    description: "Trigger export forwarding for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_deface",
    description: "Trigger defacing for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "retry_dimse_study",
    description: "Process pending or dead-letter DIMSE retry state for one study instance UID.",
    inputSchema: {
      type: "object",
      required: ["study_instance_uid", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_instance_uid: { type: "string", pattern: "^[0-9.]+$" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  }
];

const server = new Server(
  {
    name: "aegis-mcp-server",
    version: "0.1.0"
  },
  {
    capabilities: {
      tools: {}
    }
  }
);

server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools }));

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const name = String(request.params.name ?? "");
  const args = (request.params.arguments ?? {}) as Record<string, unknown>;
  const requestId = resolveRequestId(args);
  const startedAt = Date.now();
  const knownTool = isKnownToolName(name);
  const toolClass: ToolClass | "unknown" = knownTool ? (isWriteToolName(name) ? "write" : "read") : "unknown";
  const callerId = process.env.MCP_CALLER_ID?.trim() || "mcp-stdio";
  const requestedBy = extractRequestedBy(args);

  let idempotencyKey: string | null = null;
  let idempotencyHit = false;
  let response: ToolResponse;

  if (toolClass !== "unknown") {
    const rateLimitResult = rateLimiter.consume(
      `tool_class:${toolClass}`,
      toolClass === "write" ? config.writeRateLimitPerMinute : config.readRateLimitPerMinute
    );

    if (!rateLimitResult.allowed) {
      response = formatError(
        requestId,
        "RATE_LIMITED",
        `Rate limit exceeded for ${toolClass} tools; retry after ${rateLimitResult.retryAfterSeconds}s`,
        true,
        knownTool ? name : "unknown"
      );
    } else if (toolClass === "write" && isWriteToolName(name)) {
      const target = extractWriteTarget(name, args);
      if (target) {
        idempotencyKey = buildIdempotencyKey(requestId, name, target);
        const cached = writeIdempotencyCache.get(idempotencyKey);
        if (cached) {
          response = cached;
          idempotencyHit = true;
        } else {
          response = await executeTool(name, args, requestId);
        }
      } else {
        response = await executeTool(name, args, requestId);
      }
    } else {
      response = await executeTool(name, args, requestId);
    }
  } else {
    response = await executeTool(name, args, requestId);
  }

  if (toolClass === "write" && idempotencyKey && !idempotencyHit) {
    writeIdempotencyCache.set(idempotencyKey, response, config.writeIdempotencyTtlSeconds);
  }

  const payload = extractPayload(response);
  const result: InvocationLogResult = idempotencyHit ? "idempotent" : payload?.ok ? "success" : "error";
  emitInvocationLog({
    event: "mcp_tool_invocation",
    ts: new Date().toISOString(),
    request_id: requestId,
    caller_id: callerId,
    requested_by: requestedBy,
    tool_name: knownTool ? name : "unknown",
    tool_class: toolClass,
    args_redacted: redactToolArgs(args),
    result,
    error_code: payload?.error?.code,
    duration_ms: Date.now() - startedAt
  });

  return response;
});

async function executeTool(name: string, args: Record<string, unknown>, requestId: string): Promise<ToolResponse> {
  if (!isKnownToolName(name)) {
    return formatError(requestId, "VALIDATION_ERROR", `Unknown tool: ${name}`, false);
  }

  try {
    if (name === "list_studies") {
      const parsed = listStudiesArgsSchema.parse(args);
      const query = new URLSearchParams();
      if (parsed.limit !== undefined) query.set("limit", String(parsed.limit));
      if (parsed.offset !== undefined) query.set("offset", String(parsed.offset));
      if (parsed.project_id) query.set("project_id", parsed.project_id);
      if (parsed.status) query.set("status", parsed.status);
      if (parsed.modality) query.set("modality", parsed.modality);
      if (parsed.source) query.set("source", parsed.source);
      if (parsed.search) query.set("search", parsed.search);

      const suffix = query.toString() ? `?${query.toString()}` : "";
      const data = await client.get(`/api/studies${suffix}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_detail") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_diagnostics") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/diagnostics`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_diagnostics") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/diagnostics`);
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "get_study_audit") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/audit`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_routing_log") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/routing-log`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_export_shares") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/shares`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_system_health") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/healthz");
      return formatSuccess(requestId, name, data);
    }

    if (writeToolNames.includes(name)) {
      if (name === "retry_dimse_study") {
        const parsed = retryDimseArgsSchema.parse(args);
        return handleRetryDimseStudy(parsed.request_id ?? buildRequestId(), parsed);
      }

      const parsed = writeArgsSchema.parse(args);

      if (name === "trigger_classification") {
        return handleTriggerClassification(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_bids_convert") {
        return handleTriggerBidsConvert(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_export") {
        return handleTriggerExport(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_deface") {
        return handleTriggerDeface(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_qc_check") {
        return handleTriggerQcCheck(requestId, parsed);
      }

      if (name === "trigger_protocol_check") {
        return handleTriggerProtocolCheck(requestId, parsed);
      }

      if (name === "trigger_phi_scan") {
        return handleTriggerPhiScan(requestId, parsed);
      }

      return denyWriteTool(requestId, name);
    }

    return formatError(requestId, "VALIDATION_ERROR", `Unhandled tool: ${name}`, false, name);
  } catch (error) {
    if (error instanceof DisallowedPathError) {
      return formatError(requestId, "FORBIDDEN", error.message, false, name);
    }

    if (error instanceof UpstreamHttpError) {
      if (error.status === 404) {
        return formatError(requestId, "NOT_FOUND", error.body || "Resource not found", false, name);
      }
      if (error.status === 409) {
        return formatError(requestId, "CONFLICT", error.body || "Conflict", false, name);
      }
      return formatError(requestId, "UPSTREAM_ERROR", error.body || error.message, error.status >= 500, name);
    }

    if (error instanceof Error && error.name === "ZodError") {
      return formatError(requestId, "VALIDATION_ERROR", error.message, false, name);
    }

    return formatError(requestId, "UPSTREAM_ERROR", error instanceof Error ? error.message : "Unknown error", false, name);
  }
}

const knownToolNameSet = new Set<string>([...readToolNames, ...writeToolNames]);

function isKnownToolName(name: string): name is ToolName {
  return knownToolNameSet.has(name);
}

function isWriteToolName(name: string): name is (typeof writeToolNames)[number] {
  return (writeToolNames as readonly string[]).includes(name);
}

function resolveRequestId(args: Record<string, unknown>): string {
  const raw = args.request_id;
  if (typeof raw !== "string") {
    return buildRequestId();
  }

  const trimmed = raw.trim();
  if (trimmed.length < 8 || trimmed.length > 128) {
    return buildRequestId();
  }

  return trimmed;
}

function extractRequestedBy(args: Record<string, unknown>): string | undefined {
  const raw = args.requested_by;
  if (typeof raw !== "string") {
    return undefined;
  }

  const trimmed = raw.trim();
  if (!trimmed) {
    return undefined;
  }

  return trimmed.slice(0, 128);
}

function extractWriteTarget(name: ToolName, args: Record<string, unknown>): string | null {
  if (name === "retry_dimse_study") {
    return typeof args.study_instance_uid === "string" ? args.study_instance_uid : null;
  }
  return typeof args.study_uid === "string" ? args.study_uid : null;
}

function buildIdempotencyKey(requestId: string, tool: ToolName, target: string): string {
  return `${requestId}|${tool}|${target}`;
}

function extractPayload(response: ToolResponse): ToolPayload | null {
  const text = response.content[0]?.text;
  if (typeof text !== "string") {
    return null;
  }

  try {
    const parsed = JSON.parse(text);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    return parsed as ToolPayload;
  } catch {
    return null;
  }
}

function emitInvocationLog(log: InvocationLog): void {
  console.error(JSON.stringify(log));
}

class InMemoryRateLimiter {
  private readonly windowMs = 60_000;
  private readonly counters = new Map<string, { windowStart: number; count: number }>();

  consume(key: string, limit: number, now = Date.now()): { allowed: boolean; retryAfterSeconds: number } {
    if (limit <= 0) {
      return { allowed: false, retryAfterSeconds: Math.ceil(this.windowMs / 1000) };
    }

    const windowStart = now - (now % this.windowMs);
    const current = this.counters.get(key);

    if (!current || current.windowStart !== windowStart) {
      this.counters.set(key, { windowStart, count: 1 });
      this.prune(windowStart);
      return { allowed: true, retryAfterSeconds: 0 };
    }

    if (current.count >= limit) {
      const retryAfterSeconds = Math.max(1, Math.ceil((windowStart + this.windowMs - now) / 1000));
      return { allowed: false, retryAfterSeconds };
    }

    current.count += 1;
    return { allowed: true, retryAfterSeconds: 0 };
  }

  private prune(activeWindowStart: number): void {
    if (this.counters.size <= 512) {
      return;
    }

    for (const [key, value] of this.counters.entries()) {
      if (value.windowStart !== activeWindowStart) {
        this.counters.delete(key);
      }
    }
  }
}

class InMemoryIdempotencyCache {
  private readonly responses = new Map<string, { expiresAtMs: number; response: ToolResponse }>();

  get(key: string, now = Date.now()): ToolResponse | null {
    const entry = this.responses.get(key);
    if (!entry) {
      return null;
    }
    if (entry.expiresAtMs <= now) {
      this.responses.delete(key);
      return null;
    }
    return entry.response;
  }

  set(key: string, response: ToolResponse, ttlSeconds: number, now = Date.now()): void {
    if (ttlSeconds <= 0) {
      return;
    }

    this.responses.set(key, { response, expiresAtMs: now + ttlSeconds * 1000 });
    this.prune(now);
  }

  private prune(now: number): void {
    if (this.responses.size <= 2048) {
      return;
    }

    for (const [key, entry] of this.responses.entries()) {
      if (entry.expiresAtMs <= now) {
        this.responses.delete(key);
      }
    }
  }
}

const rateLimiter = new InMemoryRateLimiter();
const writeIdempotencyCache = new InMemoryIdempotencyCache();

async function main(): Promise<void> {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error(`AEGIS MCP server started in ${config.mcpMode} mode`);
}

function denyWriteTool(requestId: string, tool: ToolName) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, tool);
  }

  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are intentionally disabled in this scaffold (set MCP_ENABLE_WRITE_TOOLS=true only after implementation)", false, tool);
  }

  return formatError(requestId, "FORBIDDEN", "Write tool handler not implemented in scaffold", false, tool);
}

async function handleTriggerClassification(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Caller is not permitted to execute write tools in readonly mode",
      false,
      "trigger_classification"
    );
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_classification",
      false,
      "trigger_classification"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_classification");
  }

  if (matched.classification_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require classification", false, "trigger_classification");
  }

  if (matched.classification_status === "classifying") {
    return formatError(requestId, "CONFLICT", "Classification already in progress", false, "trigger_classification");
  }

  if (matched.classification_status && !["pending", "failed"].includes(matched.classification_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `Classification trigger blocked for current status: ${matched.classification_status}`,
      false,
      "trigger_classification"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/classify`);
  return formatSuccess(requestId, "trigger_classification", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleTriggerBidsConvert(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Caller is not permitted to execute write tools in readonly mode",
      false,
      "trigger_bids_convert"
    );
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_bids_convert",
      false,
      "trigger_bids_convert"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_bids_convert");
  }

  if (matched.bids_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require BIDS conversion", false, "trigger_bids_convert");
  }

  if (matched.bids_status === "converting") {
    return formatError(requestId, "CONFLICT", "BIDS conversion already in progress", false, "trigger_bids_convert");
  }

  if (matched.bids_status && !["pending", "failed"].includes(matched.bids_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `BIDS conversion trigger blocked for current status: ${matched.bids_status}`,
      false,
      "trigger_bids_convert"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/bids-convert`);
  return formatSuccess(requestId, "trigger_bids_convert", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleTriggerExport(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_export");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_export",
      false,
      "trigger_export"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_export");
  }

  if (matched.status !== "approved") {
    return formatError(requestId, "CONFLICT", "Only approved studies can be exported", false, "trigger_export");
  }

  if (matched.export_required === false) {
    return formatError(requestId, "CONFLICT", "Export is not required for this study", false, "trigger_export");
  }

  if (matched.export_status === "exporting") {
    return formatError(requestId, "CONFLICT", "Export is already in progress", false, "trigger_export");
  }

  if (matched.export_status && !["pending", "failed"].includes(matched.export_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `Export trigger blocked for current status: ${matched.export_status}`,
      false,
      "trigger_export"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/trigger-export`);
  return formatSuccess(requestId, "trigger_export", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleTriggerDeface(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_deface");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_deface",
      false,
      "trigger_deface"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_deface");
  }

  if (matched.defacing_required === false) {
    return formatError(requestId, "CONFLICT", "Defacing is not required for this study", false, "trigger_deface");
  }

  if (matched.status === "defacing") {
    return formatError(requestId, "CONFLICT", "Defacing is already in progress", false, "trigger_deface");
  }

  if (matched.status && ["approved", "rejected"].includes(matched.status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `Defacing trigger blocked for terminal status: ${matched.status}`,
      false,
      "trigger_deface"
    );
  }

  const data = await client.post(`/api/deface/${encodeURIComponent(parsed.study_uid)}`);
  return formatSuccess(requestId, "trigger_deface", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleRetryDimseStudy(
  requestId: string,
  parsed: {
    study_instance_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "retry_dimse_study");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow retry_dimse_study",
      false,
      "retry_dimse_study"
    );
  }

  const studyUID = encodeURIComponent(parsed.study_instance_uid);
  const detailsPath = `/api/dimse/retry/details?limit=1&study_instance_uid=${studyUID}`;
  const detailsRaw = await client.get(detailsPath);
  const details = extractDimseRetryDetails(detailsRaw);

  if (details.pending_total <= 0 && details.dead_letter_total <= 0) {
    return formatError(
      requestId,
      "CONFLICT",
      `No DIMSE retry/dead-letter entries found for study: ${parsed.study_instance_uid}`,
      false,
      "retry_dimse_study"
    );
  }

  let action = "retry_process_pending";
  let endpoint = `/api/dimse/retry/process/${studyUID}`;
  if (details.pending_total <= 0 && details.dead_letter_total > 0) {
    action = "retry_replay_dead_letter";
    endpoint = `/api/dimse/retry/replay/${studyUID}`;
  }

  const data = await client.post(endpoint);
  return formatSuccess(requestId, "retry_dimse_study", {
    accepted: true,
    study_instance_uid: parsed.study_instance_uid,
    reason: parsed.reason,
    action,
    result: data
  });
}

async function handleTriggerQcCheck(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_qc_check");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_qc_check",
      false,
      "trigger_qc_check"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_qc_check");
  }

  if (matched.qc_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require QC check", false, "trigger_qc_check");
  }

  if (matched.qc_status === "checking") {
    return formatError(requestId, "CONFLICT", "QC check already in progress", false, "trigger_qc_check");
  }

  if (matched.qc_status && !["pending", "failed"].includes(matched.qc_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `QC check trigger blocked for current status: ${matched.qc_status}`,
      false,
      "trigger_qc_check"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/qc-check`);
  return formatSuccess(requestId, "trigger_qc_check", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleTriggerProtocolCheck(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Caller is not permitted to execute write tools in readonly mode",
      false,
      "trigger_protocol_check"
    );
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_protocol_check",
      false,
      "trigger_protocol_check"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_protocol_check");
  }

  if (matched.protocol_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require protocol check", false, "trigger_protocol_check");
  }

  if (matched.protocol_status === "checking") {
    return formatError(requestId, "CONFLICT", "Protocol check already in progress", false, "trigger_protocol_check");
  }

  if (matched.protocol_status && !["pending", "failed"].includes(matched.protocol_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `Protocol check trigger blocked for current status: ${matched.protocol_status}`,
      false,
      "trigger_protocol_check"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/protocol-check`);
  return formatSuccess(requestId, "trigger_protocol_check", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleTriggerPhiScan(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_phi_scan");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_phi_scan",
      false,
      "trigger_phi_scan"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_phi_scan");
  }

  if (matched.phi_scan_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require PHI scan", false, "trigger_phi_scan");
  }

  if (matched.phi_scan_status === "scanning") {
    return formatError(requestId, "CONFLICT", "PHI scan already in progress", false, "trigger_phi_scan");
  }

  if (matched.phi_scan_status && !["pending", "failed"].includes(matched.phi_scan_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `PHI scan trigger blocked for current status: ${matched.phi_scan_status}`,
      false,
      "trigger_phi_scan"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/phi-scan`);
  return formatSuccess(requestId, "trigger_phi_scan", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

function extractStudies(value: unknown): StudySummary[] {
  if (!value || typeof value !== "object") {
    return [];
  }

  const maybe = value as ListStudiesResponse;
  if (!Array.isArray(maybe.studies)) {
    return [];
  }

  return maybe.studies.filter((item) => item && typeof item === "object" && typeof item.id === "string");
}

function extractDimseRetryDetails(value: unknown): Required<DimseRetryDetails> {
  if (!value || typeof value !== "object") {
    return { pending_total: 0, dead_letter_total: 0 };
  }

  const maybe = value as DimseRetryDetailsResponse;
  const pending = Number(maybe.ingest_retry?.pending_total ?? 0);
  const deadLetter = Number(maybe.ingest_retry?.dead_letter_total ?? 0);

  return {
    pending_total: Number.isFinite(pending) ? Math.max(0, Math.trunc(pending)) : 0,
    dead_letter_total: Number.isFinite(deadLetter) ? Math.max(0, Math.trunc(deadLetter)) : 0
  };
}

function formatSuccess(requestId: string, tool: ToolName, data: unknown) {
  const payload: ToolPayload = {
    ok: true,
    request_id: requestId,
    tool,
    data,
    warnings: []
  };
  return {
    content: [{ type: "text", text: JSON.stringify(payload, null, 2) }]
  };
}

function formatError(
  requestId: string,
  code: ErrorCode,
  message: string,
  retryable: boolean,
  tool: ToolName | "unknown" = "unknown"
): ToolResponse {
  const payload: ToolPayload = {
    ok: false,
    request_id: requestId,
    tool,
    error: {
      code,
      message,
      retryable
    }
  };

  return {
    isError: true,
    content: [{ type: "text", text: JSON.stringify(payload, null, 2) }]
  };
}

function buildRequestId() {
  return `req_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
}

main().catch((error) => {
  console.error("Fatal MCP server error", error);
  process.exit(1);
});
