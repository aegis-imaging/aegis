import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema, Tool } from "@modelcontextprotocol/sdk/types.js";
import { AegisApiClient, UpstreamHttpError } from "./aegisClient.js";
import { loadConfig } from "./config.js";
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
    code: "VALIDATION_ERROR" | "AUTH_ERROR" | "FORBIDDEN" | "NOT_FOUND" | "CONFLICT" | "UPSTREAM_ERROR" | "TIMEOUT";
    message: string;
    retryable: boolean;
  };
};

type ErrorCode = "VALIDATION_ERROR" | "AUTH_ERROR" | "FORBIDDEN" | "NOT_FOUND" | "CONFLICT" | "UPSTREAM_ERROR" | "TIMEOUT";
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
  const name = request.params.name as ToolName;
  const args = (request.params.arguments ?? {}) as Record<string, unknown>;

  if (![...readToolNames, ...writeToolNames].includes(name)) {
    return formatError("unknown", "VALIDATION_ERROR", `Unknown tool: ${name}`, false);
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
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "get_study_detail") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}`);
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "get_study_diagnostics") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/diagnostics`);
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "get_study_audit") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/audit`);
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "get_study_routing_log") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/routing-log`);
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "list_export_shares") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/shares`);
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
    }

    if (name === "get_system_health") {
      const parsed = emptyArgsSchema.parse(args);
      const data = await client.get("/healthz");
      return formatSuccess(parsed.request_id ?? buildRequestId(), name, data);
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
        return handleTriggerQcCheck(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_protocol_check") {
        return handleTriggerProtocolCheck(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_phi_scan") {
        return handleTriggerPhiScan(parsed.request_id ?? buildRequestId(), parsed);
      }

      return denyWriteTool(parsed.request_id ?? buildRequestId(), name);
    }

    return formatError(buildRequestId(), "VALIDATION_ERROR", `Unhandled tool: ${name}`, false);
  } catch (error) {
    if (error instanceof UpstreamHttpError) {
      const requestId = buildRequestId();
      if (error.status === 404) {
        return formatError(requestId, "NOT_FOUND", error.body || "Resource not found", false, name);
      }
      if (error.status === 409) {
        return formatError(requestId, "CONFLICT", error.body || "Conflict", false, name);
      }
      return formatError(requestId, "UPSTREAM_ERROR", error.body || error.message, error.status >= 500, name);
    }

    if (error instanceof Error && error.name === "ZodError") {
      return formatError(buildRequestId(), "VALIDATION_ERROR", error.message, false, name);
    }

    return formatError(buildRequestId(), "UPSTREAM_ERROR", error instanceof Error ? error.message : "Unknown error", false, name);
  }
});

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
) {
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
  return `req_${Date.now()}`;
}

main().catch((error) => {
  console.error("Fatal MCP server error", error);
  process.exit(1);
});
