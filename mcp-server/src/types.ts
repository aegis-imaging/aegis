import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import type { ToolName } from "./schemas.js";
import { readToolNames, writeToolNames } from "./schemas.js";

export type ToolPayload = {
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

export type ErrorCode =
  | "VALIDATION_ERROR"
  | "AUTH_ERROR"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "UPSTREAM_ERROR"
  | "TIMEOUT"
  | "RATE_LIMITED";

export type ToolResponse = {
  isError?: boolean;
  content: Array<{ type: "text"; text: string }>;
};

export type ToolClass = "read" | "write";

export type InvocationLogResult = "success" | "error" | "idempotent";

export type InvocationLog = {
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

export type StudySummary = {
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
  pixel_redaction_required?: boolean;
  pixel_redaction_status?: string;
  analytics_required?: boolean;
  analytics_status?: string;
  sct_required?: boolean;
  sct_status?: string;
};

export type ListStudiesResponse = {
  studies?: StudySummary[];
  total?: number;
  limit?: number;
  offset?: number;
};

export type DimseRetryDetails = {
  pending_total?: number;
  dead_letter_total?: number;
};

export type DimseRetryDetailsResponse = {
  ingest_retry?: DimseRetryDetails;
};

export const writeInputSchema: Tool["inputSchema"] = {
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

export function formatSuccess(requestId: string, tool: ToolName, data: unknown): ToolResponse {
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

export function formatError(
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

export function extractStudies(value: unknown): StudySummary[] {
  if (!value || typeof value !== "object") {
    return [];
  }

  const maybe = value as ListStudiesResponse;
  if (!Array.isArray(maybe.studies)) {
    return [];
  }

  return maybe.studies.filter((item) => item && typeof item === "object" && typeof item.id === "string");
}

export function extractDimseRetryDetails(value: unknown): Required<DimseRetryDetails> {
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

const knownToolNameSet = new Set<string>([...readToolNames, ...writeToolNames]);

export function isKnownToolName(name: string): name is ToolName {
  return knownToolNameSet.has(name);
}

export function isWriteToolName(name: string): name is (typeof writeToolNames)[number] {
  return (writeToolNames as readonly string[]).includes(name);
}

export function buildRequestId() {
  return `req_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
}
