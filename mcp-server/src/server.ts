import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema, Tool } from "@modelcontextprotocol/sdk/types.js";
import { AegisApiClient, DisallowedPathError, UpstreamHttpError } from "./aegisClient.js";
import { startAgentHttpServer } from "./agentServer.js";
import { loadConfig } from "./config.js";
import { InMemoryIdempotencyCache } from "./idempotencyCache.js";
import { InMemoryRateLimiter } from "./rateLimiter.js";
import { redactToolArgs } from "./redaction.js";
import {
  addStudyLabelArgsSchema,
  addStudyNoteArgsSchema,
  approveStudyArgsSchema,
  createShareArgsSchema,
  dimseRetryStatusArgsSchema,
  emptyArgsSchema,
  exportProjectBatchArgsSchema,
  extendShareArgsSchema,
  getAuditActorsArgsSchema,
  getIngestionTimelineArgsSchema,
  getInstitutionStatsArgsSchema,
  getShareDownloadsArgsSchema,
  getStuckStudiesArgsSchema,
  getStudyDicomTagsArgsSchema,
  getWebhookDeliveriesArgsSchema,
  listAllSharesArgsSchema,
  listAuditArgsSchema,
  listProtocolTemplatesArgsSchema,
  listStudiesArgsSchema,
  projectScopedArgsSchema,
  reactivateStudyArgsSchema,
  testWebhookArgsSchema,
  readToolNames,
  reassignStudyArgsSchema,
  reEvaluateRoutingArgsSchema,
  rejectStudyArgsSchema,
  removeStudyLabelArgsSchema,
  resetPipelineStepArgsSchema,
  retryDimseArgsSchema,
  revokeShareArgsSchema,
  setStudySubjectArgsSchema,
  studyIdArgsSchema,
  studyUidArgsSchema,
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
    description: "List studies with filters for operations triage. Supports filtering by label text and subject ID in addition to the standard filters.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        limit: { type: "integer", minimum: 1, maximum: 200 },
        offset: { type: "integer", minimum: 0 },
        project_id: { type: "string", format: "uuid" },
        status: { type: "string", enum: ["received", "defacing", "clean", "defaced", "approved", "rejected"] },
        modality: { type: "string" },
        body_part: { type: "string" },
        source: { type: "string", enum: ["external", "internal"] },
        search: { type: "string", description: "Substring match on study UID or description" },
        label: { type: "string", maxLength: 80, description: "Substring match on any study label (case-insensitive)" },
        subject_id: { type: "string", maxLength: 256, description: "Exact match on subject_id" },
        date_from: { type: "string", format: "date-time", description: "ISO 8601 lower bound on created_at (inclusive)" },
        date_to: { type: "string", format: "date-time", description: "ISO 8601 upper bound on created_at (inclusive)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_detail",
    description: "Get detail for a specific study by its database UUID.",
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
    name: "get_study_by_uid",
    description: "Get study detail by DICOM StudyInstanceUID (the UID from PACS/DICOM headers). Use this when you have a DICOM UID instead of the AEGIS database UUID.",
    inputSchema: {
      type: "object",
      required: ["study_instance_uid"],
      properties: {
        request_id: { type: "string" },
        study_instance_uid: {
          type: "string",
          pattern: "^[0-9.]+$",
          description: "DICOM StudyInstanceUID (dot-separated numeric string)"
        }
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
    description: "List export shares for a specific study UUID.",
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
    name: "list_all_shares",
    description: "List all export shares across all studies with optional status filter and pagination. Returns {shares, total, limit, offset}. Use status='active' to audit currently accessible links, 'expired' to find stale shares, 'revoked' for revoked ones.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        limit: { type: "number", minimum: 1, maximum: 200, description: "Page size (default 50)" },
        offset: { type: "number", minimum: 0 },
        status: { type: "string", enum: ["active", "expired", "revoked"], description: "Filter by computed share status" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_share_downloads",
    description: "Get the complete download history for one export share by its UUID. Returns {share_id, downloads: [{id, share_id, client_ip, accessed_at}], total}. Use for compliance auditing to see who downloaded a study and from which IP.",
    inputSchema: {
      type: "object",
      required: ["share_id"],
      properties: {
        request_id: { type: "string" },
        share_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_pipeline_stats",
    description: "Get a lightweight snapshot of the study pipeline: study counts by status (received/defacing/clean/defaced/approved/rejected/total) plus active export share count. Optionally scope to a single project. Use for a quick pipeline health check.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" }
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
    name: "get_dimse_retry_status",
    description:
      "Get DIMSE ingest retry queue status: pending/dead-letter counts, queue utilization, age metrics, and optional per-study detail when study_instance_uid is provided.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        study_instance_uid: {
          type: "string",
          pattern: "^[0-9.]+$",
          description: "Optional DICOM StudyInstanceUID — when provided, also fetches per-study retry details."
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_audit_log",
    description:
      "Query the system-wide audit trail with optional filters and server-side pagination. Returns {entries, total, limit, offset}. Filter by action prefix, resource_type, actor email, keyword search (across actor/action/resource_id/detail), or date range.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        limit: { type: "number", minimum: 1, maximum: 500, description: "Page size (default 100)" },
        offset: { type: "number", minimum: 0, description: "Row offset for pagination" },
        action: { type: "string", pattern: "^[a-z0-9_.]+$", description: "Prefix filter on action name, e.g. 'study' or 'study.approved'" },
        resource_type: { type: "string", pattern: "^[a-z0-9_]+$", description: "Exact match on resource_type, e.g. 'study', 'admin_user'" },
        actor: { type: "string", description: "Exact match on actor email address" },
        search: { type: "string", description: "Case-insensitive substring search across actor, action, resource_id, and detail JSON" },
        date_from: { type: "string", format: "date-time", description: "ISO 8601 lower bound on created_at (inclusive)" },
        date_to: { type: "string", format: "date-time", description: "ISO 8601 upper bound on created_at (inclusive)" }
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
  },
  {
    name: "create_share",
    description: "Create an export share link for an approved study. Returns a one-time token and export URL for the recipient. Study must be in 'approved' status. Requires recipient_email, confirm=true, and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "recipient_email", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        recipient_email: { type: "string", format: "email" },
        note: { type: "string", maxLength: 512 },
        expiry_hours: { type: "integer", minimum: 1, maximum: 8760, description: "Share expiry in hours (default 168 = 7 days, max 8760 = 1 year)" },
        max_downloads: { type: "integer", minimum: 1, maximum: 1000, description: "Maximum number of times the share can be downloaded before it is automatically revoked (omit for unlimited)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "re_evaluate_routing",
    description: "Re-evaluate routing rules for an existing study. Use this to recover studies stuck in an incomplete pipeline state — re-running routing may trigger classification, defacing, QC, or other required steps that were missed at ingest.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "approve_study",
    description: "Approve a study (transitions status to 'approved', enables export sharing, triggers auto-export if required). Study must not already be approved or rejected. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "reject_study",
    description: "Reject a study (transitions status to 'rejected'). Study must not already be rejected. Optional rejection_reason is stored on the study and included in the uploader notification email. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        rejection_reason: { type: "string", maxLength: 500, description: "Optional human-readable reason shown to uploader in the rejection notification email (max 500 chars)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "revoke_share",
    description: "Immediately revoke an export share link by share UUID. Recipients lose access immediately. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["share_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        share_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_stuck_studies",
    description: "List studies stuck in a non-terminal pipeline state for longer than `minutes` (default 60). Returns studies with their current status and pipeline flags for triage. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        minutes: { type: "integer", minimum: 1, maximum: 10080, description: "Age threshold in minutes (default 60)" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_breakdown_stats",
    description: "Get study counts grouped by modality and body part. Useful for understanding the composition of the study corpus. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_storage_stats",
    description: "Get aggregate DICOM file storage counts: raw file count, clean file count, total file count, and total study count. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_audit_actors",
    description: "Get recent admin actor activity summary: top actors by recency with action counts and last-seen timestamp. Only covers the last 30 days. Useful for access auditing and detecting unusual activity.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        limit: { type: "integer", minimum: 1, maximum: 100, description: "Max actors to return (default 20)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_projects",
    description: "List all AEGIS projects. Returns array of {id, name, slug, description, archived, retention_days, created_at}. Use project IDs to scope other tools (list_studies, get_pipeline_stats, get_stuck_studies, etc.) to a specific project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_institutions",
    description: "List all registered institutions (senders, receivers, or both). Returns array of {id, name, slug, institution_type, ae_title, ip_ranges, enabled, created_at}. Use institution IDs to look up study provenance or to configure routing.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_routing_rules",
    description: "List all routing rules ordered by priority. Returns array of {id, name, priority, project_id, modality, body_part, source, action, destination_id, enabled}. Use this to understand how studies are classified, processed, and forwarded automatically at ingest.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_destinations",
    description: "List all DICOM forwarding destinations. Returns array of {id, name, type, dicomweb_url, ae_title, host, port, enabled, created_at}. Destinations are referenced by routing rules with action=route_to to forward approved studies via DICOMweb STOW-RS or DIMSE C-STORE.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_series",
    description: "Get per-series DICOM metadata for a study. Returns {study_id, series: [{id, series_instance_uid, series_description, modality, body_part, instance_count, created_at}], total}. Use to inspect multi-series studies or verify series-level modality and body part metadata.",
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
    name: "get_study_labels",
    description: "Get all labels attached to a study. Returns array of {id, study_id, label, created_by, created_at}. Labels are free-text tags added by operators for cohort tagging, triage, or workflow notes.",
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
    name: "list_subjects",
    description: "List unique research subject IDs with study counts for a project. Returns array of {subject_id, study_count}. Use to audit longitudinal subject coverage or find subjects with missing sessions.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project (recommended)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_export_analytics",
    description: "Get aggregate export share analytics: total shares created, total downloads, unique recipients, average downloads per share, and breakdown by status (active/expired/revoked). Use for compliance reporting and share activity monitoring.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_webhook_subscriptions",
    description: "List all webhook subscriptions. Returns array of {id, url, events, project_id, enabled, created_at}. Webhooks push study event notifications (approved, rejected, phi_flagged, export_complete, stuck) to external HTTP endpoints.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_webhook_deliveries",
    description: "Get the delivery history for a webhook subscription. Returns array of {id, subscription_id, event, url, attempt, status_code, success, error_message, delivered_at}. Use for debugging webhook delivery failures.",
    inputSchema: {
      type: "object",
      required: ["subscription_id"],
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_ingestion_timeline",
    description: "Get daily ingestion counts (received and approved studies per day) for the last N days. Returns {days: [{day, received, approved}]}. Use to identify ingestion spikes, slowdowns, or gaps in pipeline throughput. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: { type: "integer", minimum: 1, maximum: 365, description: "Number of days to include (default 30)" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" }
      },
      additionalProperties: false
    }
  },
  {
    name: "reset_pipeline_step",
    description: "Reset a single pipeline step back to 'pending' so it can be re-processed. Use for production error recovery: re-deface, re-scan PHI, re-run QC, re-convert BIDS, re-classify, re-check protocol, or re-export. Returns 409 if the step is currently in-flight. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "step", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        step: { type: "string", enum: ["deface", "phi_scan", "qc", "bids", "classify", "protocol", "export"], description: "Pipeline step to reset" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "reassign_study",
    description: "Move a study from one project to another. The study retains all its pipeline state and audit history. Use to correct mis-routed studies. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "Study to move" },
        project_id: { type: "string", format: "uuid", description: "Target project UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "add_study_label",
    description: "Attach a free-text label to a study (max 80 characters). Labels are used for cohort tagging, triage prioritisation, or workflow notes. Duplicate labels on the same study are silently ignored. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "label", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        label: { type: "string", minLength: 1, maxLength: 80, description: "Label text (max 80 chars)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "remove_study_label",
    description: "Remove a label from a study by its label UUID. Use get_study_labels first to find the label_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "label_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        label_id: { type: "string", format: "uuid", description: "Label UUID from get_study_labels" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "set_study_subject",
    description: "Set or clear the research subject ID on a study. Subject IDs link longitudinal imaging sessions from the same de-identified participant. Use empty string to clear the subject link. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "subject_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        subject_id: { type: "string", maxLength: 256, description: "Subject identifier string (empty string clears the link)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_study_dicom_tags",
    description: "Get all non-pixel DICOM tags from the first file of a study. Returns {tags: [{tag, keyword, vr, value}], file, store}. Use for debugging de-identification issues, verifying protocol parameters, or inspecting tag values after defacing. Reads from the study's current dicom_store (raw before defacing, clean after).",
    inputSchema: {
      type: "object",
      required: ["study_instance_uid"],
      properties: {
        request_id: { type: "string" },
        study_instance_uid: {
          type: "string",
          pattern: "^[0-9.]+$",
          description: "DICOM StudyInstanceUID (dot-separated numeric string)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "add_study_note",
    description: "Add a free-text operator note to a study (max 2000 characters). Notes are stored in the audit trail as 'study.note' entries — visible in the study audit tab. Use for triage observations, incident notes, or workflow decisions. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "note", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        note: { type: "string", minLength: 1, maxLength: 2000, description: "Operator note text (max 2000 chars)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "extend_share",
    description: "Extend the expiry of an export share by N hours. Works on both active and already-expired (but not revoked) shares — the extension is computed from max(current_expires_at, now). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["share_id", "extend_hours", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        share_id: { type: "string", format: "uuid" },
        extend_hours: { type: "integer", minimum: 1, maximum: 8760, description: "Hours to extend from the later of expires_at or now" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "export_project_batch",
    description: "Trigger batch export forwarding for all eligible studies in a project: approved studies with export_required=true and export_status in (pending, failed). Returns {dispatched, study_ids, message}. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID to batch-export" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "reactivate_study",
    description: "Reactivate an expired study by resetting its status back to 'approved'. Only works on studies in 'expired' status. Use when a study's retention window has passed but it still needs to be accessible. Emits a study.reactivated audit entry. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_institution_stats",
    description: "Get aggregate statistics for an institution: total study count, studies broken down by status and modality, and the timestamp of the most recent study. Useful for understanding an institution's contribution and activity patterns.",
    inputSchema: {
      type: "object",
      required: ["institution_id"],
      properties: {
        request_id: { type: "string" },
        institution_id: { type: "string", format: "uuid", description: "Institution UUID" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_protocol_templates",
    description: "List all MRI protocol compliance templates for a project. Each template defines expected acquisition parameters (TR, TE, flip angle, slice thickness, etc.) for a specific scanner/sequence combination. Use before running a protocol check to understand what rules will be applied.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_federation_peers",
    description: "List all federation peer registrations. Federation peers are trusted remote AEGIS instances configured for future cross-tenant study sharing. Returns peer name, slug, API URL, and enabled status. No data flows to peers yet — this is a registry for future activation.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_phi_config",
    description: "Get per-project PHI detection configuration: confidence_threshold (0.0–1.0, minimum OCR confidence to flag text) and min_text_length (minimum text length to report). If no project-specific config exists, returns global defaults from PHI_CONFIDENCE_THRESHOLD and PHI_MIN_TEXT_LENGTH env vars.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_anon_profiles",
    description: "List all anonymization profiles for a project. Each profile defines which DICOM tags are retained instead of being stripped/zeroed during PS3.15 Basic Profile de-identification. The default profile (if set) is applied automatically by the upload portal.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" }
      },
      additionalProperties: false
    }
  },
  {
    name: "test_webhook",
    description: "Send a synthetic test delivery to a webhook subscription's URL. Sends a 'study.approved' test payload immediately, records the attempt in webhook_deliveries, and returns {success, status_code, url, error?}. Use to verify connectivity before real study events fire. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["subscription_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid", description: "Webhook subscription UUID" },
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
      if (parsed.body_part) query.set("body_part", parsed.body_part);
      if (parsed.source) query.set("source", parsed.source);
      if (parsed.search) query.set("search", parsed.search);
      if (parsed.label) query.set("label", parsed.label);
      if (parsed.subject_id) query.set("subject_id", parsed.subject_id);
      if (parsed.date_from) query.set("date_from", parsed.date_from);
      if (parsed.date_to) query.set("date_to", parsed.date_to);

      const suffix = query.toString() ? `?${query.toString()}` : "";
      const data = await client.get(`/api/studies${suffix}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_detail") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_by_uid") {
      const parsed = studyUidArgsSchema.parse(args);
      const uid = encodeURIComponent(parsed.study_instance_uid);
      const data = await client.get(`/api/study-uid/${uid}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_diagnostics") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/diagnostics`);
      return formatSuccess(requestId, name, data);
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

    if (name === "list_all_shares") {
      const parsed = listAllSharesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
      if (parsed.offset !== undefined) params.set("offset", String(parsed.offset));
      if (parsed.status) params.set("status", parsed.status);
      const qs = params.toString();
      const data = await client.get(`/api/shares${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_share_downloads") {
      const parsed = getShareDownloadsArgsSchema.parse(args);
      const data = await client.get(`/api/shares/${encodeURIComponent(parsed.share_id)}/downloads`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_pipeline_stats") {
      const parsed = projectScopedArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_system_health") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/healthz");
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_dimse_retry_status") {
      const parsed = dimseRetryStatusArgsSchema.parse(args);
      const summary = await client.get("/api/dimse/retry/summary");
      let details: unknown = undefined;
      if (parsed.study_instance_uid) {
        const uid = encodeURIComponent(parsed.study_instance_uid);
        details = await client.get(`/api/dimse/retry/details?limit=10&study_instance_uid=${uid}`);
      }
      return formatSuccess(requestId, name, { summary, ...(details !== undefined ? { details } : {}) });
    }

    if (name === "get_audit_log") {
      const parsed = listAuditArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
      if (parsed.offset !== undefined) params.set("offset", String(parsed.offset));
      if (parsed.action) params.set("action", parsed.action);
      if (parsed.resource_type) params.set("resource_type", parsed.resource_type);
      if (parsed.actor) params.set("actor", parsed.actor);
      if (parsed.search) params.set("search", parsed.search);
      if (parsed.date_from) params.set("date_from", parsed.date_from);
      if (parsed.date_to) params.set("date_to", parsed.date_to);
      const qs = params.toString();
      const data = await client.get(`/api/audit${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_stuck_studies") {
      const parsed = getStuckStudiesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.minutes !== undefined) params.set("minutes", String(parsed.minutes));
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/studies/stuck${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_breakdown_stats") {
      const parsed = projectScopedArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/breakdown${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_storage_stats") {
      const parsed = projectScopedArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/storage/stats${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_audit_actors") {
      const parsed = getAuditActorsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
      const qs = params.toString();
      const data = await client.get(`/api/audit/actors${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_projects") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/projects");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_institutions") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/institutions");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_routing_rules") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/routing-rules");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_destinations") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/destinations");
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_series") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/series`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_labels") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/labels`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_subjects") {
      const parsed = projectScopedArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/subjects${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_export_analytics") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/export-analytics");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_webhook_subscriptions") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/webhook-subscriptions");
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_webhook_deliveries") {
      const parsed = getWebhookDeliveriesArgsSchema.parse(args);
      const data = await client.get(`/api/webhook-subscriptions/${parsed.subscription_id}/deliveries`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_ingestion_timeline") {
      const parsed = getIngestionTimelineArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/timeline${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_study_dicom_tags") {
      const parsed = getStudyDicomTagsArgsSchema.parse(args);
      const uid = encodeURIComponent(parsed.study_instance_uid);
      const data = await client.get(`/api/studies/${uid}/dicom-tags`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_institution_stats") {
      const parsed = getInstitutionStatsArgsSchema.parse(args);
      const data = await client.get(`/api/institutions/${parsed.institution_id}/stats`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_protocol_templates") {
      const parsed = listProtocolTemplatesArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${parsed.project_id}/protocol-templates`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_federation_peers") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/federation-peers");
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_phi_config") {
      const parsed = listProtocolTemplatesArgsSchema.parse(args); // same shape: {project_id}
      const data = await client.get(`/api/projects/${parsed.project_id}/phi-config`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_anon_profiles") {
      const parsed = listProtocolTemplatesArgsSchema.parse(args); // same shape: {project_id}
      const data = await client.get(`/api/projects/${parsed.project_id}/anon-profiles`);
      return formatSuccess(requestId, name, data);
    }

    if (writeToolNames.includes(name)) {
      if (name === "retry_dimse_study") {
        const parsed = retryDimseArgsSchema.parse(args);
        return handleRetryDimseStudy(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "approve_study") {
        const parsedApprove = approveStudyArgsSchema.parse(args);
        return handleApproveStudy(parsedApprove.request_id ?? buildRequestId(), parsedApprove);
      }

      if (name === "reject_study") {
        const parsedReject = rejectStudyArgsSchema.parse(args);
        return handleRejectStudy(parsedReject.request_id ?? buildRequestId(), parsedReject);
      }

      if (name === "revoke_share") {
        const parsedRevoke = revokeShareArgsSchema.parse(args);
        return handleRevokeShare(parsedRevoke.request_id ?? buildRequestId(), parsedRevoke);
      }

      if (name === "create_share") {
        const parsedShare = createShareArgsSchema.parse(args);
        return handleCreateShare(parsedShare.request_id ?? buildRequestId(), parsedShare);
      }

      if (name === "re_evaluate_routing") {
        const parsedReEval = reEvaluateRoutingArgsSchema.parse(args);
        return handleReEvaluateRouting(parsedReEval.request_id ?? buildRequestId(), parsedReEval);
      }

      if (name === "reset_pipeline_step") {
        const parsedReset = resetPipelineStepArgsSchema.parse(args);
        return handleResetPipelineStep(parsedReset.request_id ?? buildRequestId(), parsedReset);
      }

      if (name === "reassign_study") {
        const parsedReassign = reassignStudyArgsSchema.parse(args);
        return handleReassignStudy(parsedReassign.request_id ?? buildRequestId(), parsedReassign);
      }

      if (name === "add_study_label") {
        const parsedLabel = addStudyLabelArgsSchema.parse(args);
        return handleAddStudyLabel(parsedLabel.request_id ?? buildRequestId(), parsedLabel);
      }

      if (name === "remove_study_label") {
        const parsedRemLabel = removeStudyLabelArgsSchema.parse(args);
        return handleRemoveStudyLabel(parsedRemLabel.request_id ?? buildRequestId(), parsedRemLabel);
      }

      if (name === "set_study_subject") {
        const parsedSubject = setStudySubjectArgsSchema.parse(args);
        return handleSetStudySubject(parsedSubject.request_id ?? buildRequestId(), parsedSubject);
      }

      if (name === "add_study_note") {
        const parsedNote = addStudyNoteArgsSchema.parse(args);
        return handleAddStudyNote(parsedNote.request_id ?? buildRequestId(), parsedNote);
      }

      if (name === "extend_share") {
        const parsedExtend = extendShareArgsSchema.parse(args);
        return handleExtendShare(parsedExtend.request_id ?? buildRequestId(), parsedExtend);
      }

      if (name === "export_project_batch") {
        const parsedBatch = exportProjectBatchArgsSchema.parse(args);
        return handleExportProjectBatch(parsedBatch.request_id ?? buildRequestId(), parsedBatch);
      }

      if (name === "reactivate_study") {
        const parsedReactivate = reactivateStudyArgsSchema.parse(args);
        return handleReactivateStudy(parsedReactivate.request_id ?? buildRequestId(), parsedReactivate);
      }

      if (name === "test_webhook") {
        const parsedTestWebhook = testWebhookArgsSchema.parse(args);
        return handleTestWebhook(parsedTestWebhook.request_id ?? buildRequestId(), parsedTestWebhook);
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
  if (
    name === "approve_study" ||
    name === "reject_study" ||
    name === "create_share" ||
    name === "re_evaluate_routing" ||
    name === "reset_pipeline_step" ||
    name === "reassign_study" ||
    name === "add_study_label" ||
    name === "remove_study_label" ||
    name === "set_study_subject" ||
    name === "add_study_note" ||
    name === "reactivate_study"
  ) {
    return typeof args.study_id === "string" ? args.study_id : null;
  }
  if (name === "revoke_share" || name === "extend_share") {
    return typeof args.share_id === "string" ? args.share_id : null;
  }
  if (name === "export_project_batch") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "test_webhook") {
    return typeof args.subscription_id === "string" ? args.subscription_id : null;
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

const rateLimiter = new InMemoryRateLimiter();
const writeIdempotencyCache = new InMemoryIdempotencyCache<ToolResponse>();

async function main(): Promise<void> {
  if (config.agentHttpPort) {
    startAgentHttpServer(client, {
      port: config.agentHttpPort,
      allowedOrigin: config.agentAllowedOrigin,
      apiKey: config.agentApiKey,
      bearerToken: config.agentBearerToken,
      requireAuth: config.agentRequireAuth,
      rateLimitPerMinute: config.agentRateLimitPerMinute,
      llmBaseUrl: config.agentLlmBaseUrl,
      llmApiKey: config.agentLlmApiKey,
      llmModel: config.agentLlmModel,
      llmTemperature: config.agentLlmTemperature,
      llmMaxTokens: config.agentLlmMaxTokens
    });
  }
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

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/trigger-deface`);
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

function formatSuccess(requestId: string, tool: ToolName, data: unknown): ToolResponse {
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

async function handleCreateShare(
  requestId: string,
  parsed: { study_id: string; recipient_email: string; note?: string; expiry_hours?: number; max_downloads?: number; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_share");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_share", false, "create_share");
  }

  const study = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}`) as Record<string, unknown>;
  if (study?.status !== "approved") {
    return formatError(requestId, "CONFLICT", `Only approved studies can be shared; current status: ${study?.status ?? "unknown"}`, false, "create_share");
  }

  const payload: Record<string, unknown> = { recipient_email: parsed.recipient_email };
  if (parsed.note !== undefined) payload.note = parsed.note;
  if (parsed.expiry_hours !== undefined) payload.expiry_hours = parsed.expiry_hours;
  if (parsed.max_downloads !== undefined) payload.max_downloads = parsed.max_downloads;

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/share`, payload);
  return formatSuccess(requestId, "create_share", {
    accepted: true,
    study_id: parsed.study_id,
    recipient_email: parsed.recipient_email,
    reason: parsed.reason,
    result: data
  });
}

async function handleReEvaluateRouting(
  requestId: string,
  parsed: { study_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "re_evaluate_routing");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow re_evaluate_routing", false, "re_evaluate_routing");
  }

  const data = await client.post(`/api/routing-rules/evaluate/${encodeURIComponent(parsed.study_id)}`);
  return formatSuccess(requestId, "re_evaluate_routing", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleApproveStudy(
  requestId: string,
  parsed: { study_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "approve_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow approve_study", false, "approve_study");
  }

  const study = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}`) as Record<string, unknown>;
  const currentStatus = study?.status as string | undefined;
  if (currentStatus === "approved" || currentStatus === "rejected") {
    return formatError(requestId, "CONFLICT", `Study is already ${currentStatus}`, false, "approve_study");
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/approve`);
  return formatSuccess(requestId, "approve_study", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleRejectStudy(
  requestId: string,
  parsed: { study_id: string; rejection_reason?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "reject_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow reject_study", false, "reject_study");
  }

  const study = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}`) as Record<string, unknown>;
  const currentStatus = study?.status as string | undefined;
  if (currentStatus === "rejected") {
    return formatError(requestId, "CONFLICT", "Study is already rejected", false, "reject_study");
  }

  // Pass optional rejection_reason to the API (stored on study + included in uploader email).
  const rejectPayload = parsed.rejection_reason ? { reason: parsed.rejection_reason } : undefined;
  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/reject`, rejectPayload);
  return formatSuccess(requestId, "reject_study", {
    accepted: true,
    study_id: parsed.study_id,
    rejection_reason: parsed.rejection_reason,
    reason: parsed.reason,
    result: data
  });
}

async function handleRevokeShare(
  requestId: string,
  parsed: { share_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "revoke_share");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow revoke_share", false, "revoke_share");
  }

  const data = await client.delete(`/api/shares/${encodeURIComponent(parsed.share_id)}`);
  return formatSuccess(requestId, "revoke_share", {
    accepted: true,
    share_id: parsed.share_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleResetPipelineStep(
  requestId: string,
  parsed: { study_id: string; step: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "reset_pipeline_step");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow reset_pipeline_step", false, "reset_pipeline_step");
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/reset-pipeline-step`, { step: parsed.step });
  return formatSuccess(requestId, "reset_pipeline_step", {
    accepted: true,
    study_id: parsed.study_id,
    step: parsed.step,
    reason: parsed.reason,
    result: data
  });
}

async function handleReassignStudy(
  requestId: string,
  parsed: { study_id: string; project_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "reassign_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow reassign_study", false, "reassign_study");
  }

  const data = await client.put(`/api/studies/${encodeURIComponent(parsed.study_id)}/project`, { project_id: parsed.project_id });
  return formatSuccess(requestId, "reassign_study", {
    accepted: true,
    study_id: parsed.study_id,
    project_id: parsed.project_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleAddStudyLabel(
  requestId: string,
  parsed: { study_id: string; label: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "add_study_label");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow add_study_label", false, "add_study_label");
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/labels`, { label: parsed.label });
  return formatSuccess(requestId, "add_study_label", {
    accepted: true,
    study_id: parsed.study_id,
    label: parsed.label,
    reason: parsed.reason,
    result: data
  });
}

async function handleRemoveStudyLabel(
  requestId: string,
  parsed: { study_id: string; label_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "remove_study_label");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow remove_study_label", false, "remove_study_label");
  }

  const data = await client.delete(`/api/studies/${encodeURIComponent(parsed.study_id)}/labels/${encodeURIComponent(parsed.label_id)}`);
  return formatSuccess(requestId, "remove_study_label", {
    accepted: true,
    study_id: parsed.study_id,
    label_id: parsed.label_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleSetStudySubject(
  requestId: string,
  parsed: { study_id: string; subject_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "set_study_subject");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow set_study_subject", false, "set_study_subject");
  }

  const data = await client.put(`/api/studies/${encodeURIComponent(parsed.study_id)}/subject`, { subject_id: parsed.subject_id });
  return formatSuccess(requestId, "set_study_subject", {
    accepted: true,
    study_id: parsed.study_id,
    subject_id: parsed.subject_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleAddStudyNote(
  requestId: string,
  parsed: { study_id: string; note: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "add_study_note");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow add_study_note", false, "add_study_note");
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/notes`, { note: parsed.note });
  return formatSuccess(requestId, "add_study_note", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleExtendShare(
  requestId: string,
  parsed: { share_id: string; extend_hours: number; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "extend_share");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow extend_share", false, "extend_share");
  }

  const data = await client.patch(`/api/shares/${encodeURIComponent(parsed.share_id)}/extend`, { extend_hours: parsed.extend_hours });
  return formatSuccess(requestId, "extend_share", {
    accepted: true,
    share_id: parsed.share_id,
    extend_hours: parsed.extend_hours,
    reason: parsed.reason,
    result: data
  });
}

async function handleExportProjectBatch(
  requestId: string,
  parsed: { project_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "export_project_batch");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow export_project_batch", false, "export_project_batch");
  }

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/export-batch`);
  return formatSuccess(requestId, "export_project_batch", {
    accepted: true,
    project_id: parsed.project_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleReactivateStudy(
  requestId: string,
  parsed: { study_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "reactivate_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow reactivate_study", false, "reactivate_study");
  }

  const study = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}`) as Record<string, unknown>;
  if (study?.status !== "expired") {
    return formatError(requestId, "CONFLICT", `Only expired studies can be reactivated; current status: ${study?.status ?? "unknown"}`, false, "reactivate_study");
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/reactivate`);
  return formatSuccess(requestId, "reactivate_study", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleTestWebhook(
  requestId: string,
  parsed: { subscription_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "test_webhook");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow test_webhook", false, "test_webhook");
  }

  const data = await client.post(`/api/webhook-subscriptions/${encodeURIComponent(parsed.subscription_id)}/test`);
  return formatSuccess(requestId, "test_webhook", {
    accepted: true,
    subscription_id: parsed.subscription_id,
    reason: parsed.reason,
    result: data
  });
}

function buildRequestId() {
  return `req_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
}

main().catch((error) => {
  console.error("Fatal MCP server error", error);
  process.exit(1);
});
