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
  toggleStudyFlagArgsSchema,
  apiKeyIdArgsSchema,
  approveStudyArgsSchema,
  bulkLabelStudiesArgsSchema,
  bulkStudyActionArgsSchema,
  createApiKeyArgsSchema,
  generateSyntheticStudyArgsSchema,
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
  getExpiringStudiesArgsSchema,
  getRetentionPreviewArgsSchema,
  getDestinationHealthArgsSchema,
  simulateRoutingArgsSchema,
  getCohortReportArgsSchema,
  getStudyDicomTagsArgsSchema,
  getWebhookDeliveriesArgsSchema,
  listAllSharesArgsSchema,
  listAuditArgsSchema,
  listProtocolTemplatesArgsSchema,
  listStudiesArgsSchema,
  processingTimesArgsSchema,
  routingStatsArgsSchema,
  destinationStatsArgsSchema,
  routingRuleStatsArgsSchema,
  pipelineFunnelArgsSchema,
  projectHealthArgsSchema,
  complianceReportArgsSchema,
  storageUsageArgsSchema,
  anonDiffArgsSchema,
  listStudyRelationshipsArgsSchema,
  linkStudiesArgsSchema,
  unlinkStudiesArgsSchema,
  triggerLongitudinalAnalyticsArgsSchema,
  getDailySummaryArgsSchema,
  listDigestSubscriptionsArgsSchema,
  createDigestSubscriptionArgsSchema,
  deleteDigestSubscriptionArgsSchema,
  createWebhookSubscriptionArgsSchema,
  updateWebhookSubscriptionArgsSchema,
  deleteWebhookSubscriptionArgsSchema,
  retryWebhookDeliveryArgsSchema,
  createProjectArgsSchema,
  updateProjectArgsSchema,
  archiveRestoreProjectArgsSchema,
  setProjectRetentionArgsSchema,
  setProjectSLAThresholdArgsSchema,
  listAdminUsersArgsSchema,
  createAdminUserArgsSchema,
  updateAdminUserArgsSchema,
  deleteAdminUserArgsSchema,
  sendAdminInviteArgsSchema,
  listProjectMembersArgsSchema,
  addProjectMemberArgsSchema,
  updateProjectMemberArgsSchema,
  removeProjectMemberArgsSchema,
  toggleProjectRestrictedArgsSchema,
  listInviteCodesArgsSchema,
  listInviteRequestsArgsSchema,
  createInviteCodeArgsSchema,
  inviteCodeIdArgsSchema,
  sendInviteCodeArgsSchema,
  inviteRequestActionArgsSchema,
  createProtocolTemplateArgsSchema,
  updateProtocolTemplateArgsSchema,
  templateIdArgsSchema,
  createAnonProfileArgsSchema,
  updateAnonProfileArgsSchema,
  deleteAnonProfileArgsSchema,
  setDefaultAnonProfileArgsSchema,
  createInstitutionArgsSchema,
  updateInstitutionArgsSchema,
  institutionIdArgsSchema,
  linkInstitutionProjectArgsSchema,
  unlinkInstitutionProjectArgsSchema,
  createFederationPeerArgsSchema,
  updateFederationPeerArgsSchema,
  federationPeerIdArgsSchema,
  setStorageQuotaArgsSchema,
  updatePhiConfigArgsSchema,
  deleteStudyArgsSchema,
  bulkPipelineTriggerArgsSchema,
  getTCIASeriesArgsSchema,
  importTCIASeriesArgsSchema,
  exportProtocolTemplatesArgsSchema,
  importProtocolTemplatesArgsSchema,
  getWebhookStatsArgsSchema,
  listAllWebhookDeliveriesArgsSchema,
  batchImportStudiesArgsSchema,
  getUserPreferencesArgsSchema,
  setUserPreferencesArgsSchema,
  exportRoutingRulesArgsSchema,
  importRoutingRulesArgsSchema,
  reorderRoutingRulesArgsSchema,
  bulkToggleRoutingRulesArgsSchema,
  queryPacsArgsSchema,
  retrievePacsStudyArgsSchema,
  listDeletedStudiesArgsSchema,
  softDeleteStudyArgsSchema,
  restoreStudyArgsSchema,
  getProtocolTrendArgsSchema,
  listStudyNotesArgsSchema,
  bulkCreateSharesArgsSchema,
  getProjectBidsInfoArgsSchema,
  projectScopedArgsSchema,
  exportSharesCsvArgsSchema,
  reactivateStudyArgsSchema,
  cloneProjectArgsSchema,
  reEvaluateProjectRoutingArgsSchema,
  createDestinationArgsSchema,
  updateDestinationArgsSchema,
  destinationIdArgsSchema,
  createRoutingRuleArgsSchema,
  updateRoutingRuleArgsSchema,
  routingRuleIdArgsSchema,
  testDestinationArgsSchema,
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
  pixel_redaction_required?: boolean;
  pixel_redaction_status?: string;
  analytics_required?: boolean;
  analytics_status?: string;
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
    description: "List studies with filters for operations triage. Supports filtering by label text and subject ID in addition to the standard filters. Results can be sorted by any indexed column.",
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
        date_to: { type: "string", format: "date-time", description: "ISO 8601 upper bound on created_at (inclusive)" },
        flagged: { type: "boolean", description: "If true, return only priority-flagged studies" },
        sort_by: { type: "string", enum: ["created_at", "updated_at", "status", "modality", "body_part", "source", "instance_count"], description: "Column to sort by (default: created_at)" },
        sort_dir: { type: "string", enum: ["asc", "desc"], description: "Sort direction (default: desc)" }
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
    name: "get_study_processing_summary",
    description: "Get a structured summary of all 7 pipeline steps (classification, phi_scan, protocol_check, defacing, qc_check, bids_conversion, export) for a study — shows required/status/terminal/in_progress/failed per step, overall pipeline_complete flag, and list of blockers. Use this for quick pipeline status checks without parsing the full study record.",
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
    name: "get_export_shares_csv",
    description: "Fetch the export shares CSV data as text. Returns all export shares matching the optional filters in CSV format. Useful for compliance reporting and bulk share auditing.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Filter shares to studies in this project" },
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
    name: "trigger_pixel_redaction",
    description: "Trigger pixel redaction for one study UID with precondition checks.",
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
    name: "trigger_analytics",
    description: "Trigger analytics for one study UID with precondition checks.",
    inputSchema: writeInputSchema
  },
  {
    name: "trigger_longitudinal_analytics",
    description: "Trigger longitudinal analytics (TBM-SyN) for a follow-up study against its baseline. Auto-computes scan_interval_days from study_date if both studies have it.",
    inputSchema: {
      type: "object",
      required: ["study_uid", "baseline_study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_uid: { type: "string", pattern: "^[0-9.]+$", description: "Follow-up study DICOM UID" },
        baseline_study_id: { type: "string", format: "uuid", description: "Baseline study database UUID" },
        scan_interval_days: { type: "number", minimum: 0, description: "Days between scans; auto-computed from study_date if omitted" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      }
    }
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
    name: "bulk_create_shares",
    description: "Create export share links for multiple approved studies in one call (max 200 per call). Each study gets its own unique share token. Non-approved or not-found studies are reported in the errors array — partial success is supported. Sends notification email to recipient for each created share. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_ids", "recipient_email", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_ids: { type: "array", items: { type: "string", format: "uuid" }, minItems: 1, maxItems: 200, description: "Array of study UUIDs to share (max 200)" },
        recipient_email: { type: "string", format: "email", description: "Email address of the share recipient" },
        note: { type: "string", maxLength: 500 },
        expiry_hours: { type: "integer", minimum: 1, maximum: 8760, description: "Share expiry in hours (default 168 = 7 days)" },
        max_downloads: { type: "integer", minimum: 1, description: "Max downloads per share (omit for unlimited)" },
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
    description: "Immediately revoke an export share link by share UUID. Recipients lose access immediately. Optionally supply a revocation_reason (stored in DB and audit). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["share_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        share_id: { type: "string", format: "uuid" },
        revocation_reason: { type: "string", maxLength: 500, description: "Optional reason stored in the database and audit trail" },
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
    name: "get_expiring_studies",
    description: "List approved studies that will be auto-expired by the retention policy within the next N days (default 7). Each result includes retention_days, expires_at (ISO 8601), and days_until_expiry. Only studies from projects with a non-null retention_days appear. Use this for proactive warnings before data is lost. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: { type: "integer", minimum: 1, maximum: 365, description: "Look-ahead window in days (default 7)" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" },
        limit: { type: "integer", minimum: 1, maximum: 500, description: "Max results (default 200)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_retention_preview",
    description: "Dry-run preview of how many approved studies would be expired if a project's retention policy were set to N days. Returns would_expire_count, total_approved, and an age_distribution histogram (6 buckets: 0–7d, 8–30d, 31–90d, 91–180d, 181–365d, 365d+). Each bucket shows count and whether those studies would be affected at the given policy. Does NOT modify any data — safe to call at any time.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID to preview retention for" },
        days: { type: "integer", minimum: 1, maximum: 3650, description: "Simulated retention period in days (default 90)" }
      },
      required: ["project_id"],
      additionalProperties: false
    }
  },
  {
    name: "get_label_usage",
    description: "Get aggregated label usage statistics across all studies: label text, total count, and distinct study count per label. Optionally scope to a single project.",
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
    name: "get_modality_trend",
    description: "Get daily study counts grouped by modality over the last N days. Returns per-day breakdown and aggregate totals. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: { type: "integer", minimum: 1, maximum: 365, description: "Look-back period in days (default 30)" },
        project_id: { type: "string", format: "uuid", description: "Scope to a single project" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_source_trend",
    description: "Get daily study ingestion counts split by source (external vs internal) over the last N days. Returns per-day breakdown and aggregate totals. Optionally scope to a single project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: { type: "integer", minimum: 1, maximum: 365, description: "Look-back period in days (default 30)" },
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
    name: "get_processing_stats",
    description: "Get per-stage pipeline processing-time statistics (avg, p95, min, max, count) derived from the audit trail. Stages: deface, phi_scan, qc_check, bids_conversion, classification, protocol_check, export. Useful for identifying bottlenecks, SLA compliance monitoring, and capacity planning. Results are ordered slowest-to-fastest by average duration.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days (default 30)"
        },
        project_id: {
          type: "string",
          format: "uuid",
          description: "Scope to a single project (optional)"
        }
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
    description: "List all AEGIS projects. Returns array of {id, name, slug, description, archived, restricted, member_count, retention_days, stuck_threshold_minutes, created_at}. restricted=true means only project members and platform admins can see the project. member_count shows how many users are assigned to the project. Use project IDs to scope other tools (list_studies, get_pipeline_stats, get_stuck_studies, etc.) to a specific project.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_project_members",
    description: "List all members of a project and their access levels. Returns {members: [{id, admin_user_id, role, institution_id, institution_name, user_email, user_name, notes, created_at}]}. Coordinating center roles (owner/coordinator/reviewer) have institution_id=null and see all studies. Site roles (site_coordinator/site_viewer) are scoped to a single institution's studies.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" }
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
    name: "test_destination",
    description: "Test connectivity to a DICOM forwarding destination. DICOMweb: sends GET /studies?limit=1 with auth headers. DIMSE: sends C-ECHO via the dimse-receiver. Returns {destination_id, type, success, latency_ms, status_code?, error?}. Useful before adding routing rules to verify the destination is reachable.",
    inputSchema: {
      type: "object",
      required: ["destination_id"],
      properties: {
        request_id: { type: "string" },
        destination_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_routing_stats",
    description: "Get aggregate routing health statistics across all destinations for the last N days. Returns totals (attempts, successful, failed, success_rate) plus a per-destination breakdown ordered by attempt count. Useful for monitoring cross-cloud routing health (GCP↔AWS↔Azure) and identifying failing destinations.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days (default 30)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_destination_stats",
    description: "Get routing health statistics for a specific DICOM forwarding destination: total attempts, success/failure counts, success rate, recent error messages, and daily breakdown. Use this to diagnose why a specific cross-cloud forwarding destination is failing.",
    inputSchema: {
      type: "object",
      required: ["destination_id"],
      properties: {
        request_id: { type: "string" },
        destination_id: { type: "string", format: "uuid", description: "UUID of the destination" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days (default 30)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_destination_health",
    description: "Get connectivity test history for DICOM forwarding destinations. When destination_id is provided, returns a detailed summary (test count, success rate, last_tested_at, last_success, last_failure, last_error, last_latency_ms, status) plus a recent test log from the audit trail. Status is 'healthy' (≥90% success), 'degraded' (50-90%), 'failing' (<50%), or 'unknown' (never tested). When destination_id is omitted, returns a summary for ALL destinations. Use this to check if cross-cloud routing is healthy before diagnosing failures.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        destination_id: { type: "string", format: "uuid", description: "Specific destination; omit for all" },
        limit: { type: "integer", minimum: 1, maximum: 100, description: "Max recent test log entries (default 20, single destination only)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "simulate_routing",
    description: "Dry-run simulation of routing rule evaluation against a hypothetical study with given attributes. Returns which enabled rules would match (matched_rules), which would not (skipped_rules), and an action_summary showing boolean flags for each action type (require_defacing, require_phi_scan, require_qc_check, require_bids_conversion, require_classification, require_protocol_check, require_export, require_analytics, auto_approve, reject). For route_to rules, destination_name is also resolved. Use this to test routing rule changes before activating them, or to explain why a study did or did not trigger a pipeline step.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Optional project scope; omit to test with no project scoping" },
        modality: { type: "string", description: "e.g. MRI, CT, PET" },
        body_part: { type: "string", description: "e.g. HEAD, CHEST, ABDOMEN" },
        source: { type: "string", enum: ["external", "internal"], description: "Ingest source (default: external)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_routing_rule_stats",
    description: "Get per-rule hit analytics for the last N days. Returns {total_hits, active_rules, by_rule: [{rule_id, rule_name, rule_action, rule_enabled, destination_id, destination_name, hit_count, last_matched_at, first_matched_at}], unused_rules: [{rule_id, rule_name, rule_action, rule_enabled, destination_id}]}. Useful for identifying stale/unused routing rules, understanding which rules fire most, and auditing routing configuration health.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days (default 30)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_pipeline_funnel",
    description: "Get a conversion funnel showing how many studies pass through each pipeline stage. Returns {period_days, generated_at, project_id, funnel: [{stage, count, pct_of_total, pct_of_prev}]}. Stages: received → classified → phi_scanned → defaced → qc_passed → bids_converted → approved → exported. pct_of_total = % of received studies; pct_of_prev = conversion rate from prior stage. Useful for identifying pipeline bottlenecks.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days (default 30)"
        },
        project_id: {
          type: "string",
          format: "uuid",
          description: "Scope to a specific project (omit for all projects)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_project_health",
    description: "Get a consolidated health snapshot for a project (or all projects). Returns {period_days, generated_at, project_id, studies: {received, defacing, clean, defaced, approved, rejected}, storage: {raw_file_count, clean_file_count, total_file_count, total_studies}, stuck_count, routing: {attempts, successful, failed, success_rate}, funnel: [{stage, count, pct_of_total, pct_of_prev}]}. Single call combining study counts, storage, routing health, stuck count, and pipeline funnel — ideal for a quick project status check.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days for routing and funnel (default 30)"
        },
        project_id: {
          type: "string",
          format: "uuid",
          description: "Scope to a specific project (omit for all projects)"
        },
        stuck_minutes: {
          type: "integer",
          minimum: 1,
          description: "Idle threshold for stuck detection in minutes (default 60)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_compliance_report",
    description: "Get a compliance report for a project covering PHI detection, defacing, protocol compliance, and export activity. Returns {project_id, period_days, studies: {total, approved, rejected, pending}, phi_detection: {scanned, flagged, flag_rate_pct}, defacing: {required, completed, failed, avg_qa_score}, protocol_compliance: {checked, compliant, minor_deviations, non_compliant}, exports: {shares_created, shares_downloaded, total_downloads}}. Useful for audits and IRB reporting.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        days: {
          type: "integer",
          minimum: 1,
          maximum: 365,
          description: "Look-back window in days (default 30)"
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_compliance_report_csv",
    description: "Fetch the compliance report for a project as CSV text. Returns section/metric/value rows covering studies, PHI detection, defacing, protocol compliance, and exports. Useful for downloading or processing compliance data programmatically.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        days: { type: "integer", minimum: 1, maximum: 365, description: "Look-back window in days (default 30)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_cohort_report",
    description: "Get a per-subject cohort summary for a project. Returns total_subjects, subjects_multi_study (subjects with ≥2 studies), total_studies_with_subject, modality_coverage {modality: subject_count}, and subjects[] each with: subject_id, study_count, approved_count, rejected_count, pending_count, modalities[], earliest_study_at, latest_study_at, all_approved, has_defaced, has_exported. Use to identify data completeness gaps (e.g. missing follow-up scans), subjects with multiple modalities, and longitudinal cohort health.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_storage_usage",
    description: "Get storage usage for a project in bytes, with optional quota information. Returns {project_id, used_bytes, quota_bytes (null if no quota), usage_pct (null if no quota)}. Use to check if a project is approaching its storage quota.",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_anonymization_diff",
    description: "Get the tag-level diff between the raw and de-identified DICOM stores for a study, showing exactly which tags were removed, modified, or added during anonymization. Returns {study_id, diff: {removed: [{tag, keyword, vr, raw_value}], modified: [{tag, keyword, vr, raw_value, clean_value}], added: []}}. Returns 404 if no raw files, 204 if no clean files yet. Requires the study's DICOM StudyInstanceUID (not the database ID).",
    inputSchema: {
      type: "object",
      required: ["study_uid"],
      properties: {
        request_id: { type: "string" },
        study_uid: { type: "string", description: "DICOM StudyInstanceUID (not the database UUID)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_system_health_summary",
    description: "Get a cached (30s TTL) system-wide health summary aggregating the API healthz probe, sidecar statuses, pipeline activity (studies received/approved in last 24h, stuck count, error rate), and DIMSE retry/dead-letter queue depths. Returns {generated_at, api: {status, database, storage}, services: {defacing, phi-detection, qc-service, ...}, pipeline: {studies_received_24h, studies_approved_24h, studies_stuck, pipeline_error_rate}, dimse: {pending_retries, dead_letter}}. Use for a quick operational health check.",
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
    name: "list_study_relationships",
    description: "List all relationships for a study (as source or target). Returns {relationships: [{id, study_id, related_study_id, relationship, notes, created_by, created_at, related_study: {id, study_instance_uid, modality, status, ...}}]}. Relationship types: baseline, follow_up, comparison, replicate. Use to navigate longitudinal imaging series and multi-session studies.",
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
    name: "get_daily_summary",
    description: "Get a system-wide ops briefing for the last N hours (default 24, max 168). Returns: ingestion counts (received/approved/rejected/stuck), current pipeline state (pending review, in-processing, failed), routing stats (attempts/success rate), top 5 projects by received count, and last 10 significant audit events. Use as a morning briefing or when triaging platform health.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        hours: { type: "integer", minimum: 1, maximum: 168, description: "Lookback window in hours (default 24, max 168 = 7 days)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_digest_subscriptions",
    description: "List email digest subscriptions. Returns array of {id, project_id, email, frequency, last_sent_at, created_at}. Digests send weekly or monthly plain-text study summaries (no PHI — counts only). Optionally filter by project_id.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Filter subscriptions for a specific project" }
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
    description: "List all webhook subscriptions. Returns array of {id, url, events, project_id, enabled, created_at}. Webhooks push study event notifications (created, processing_complete, approved, rejected, phi_flagged, export_complete, stuck) to external HTTP endpoints.",
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
    description: "Reset a single pipeline step back to 'pending' so it can be re-processed. Use for production error recovery: re-deface, re-scan PHI, re-run QC, re-convert BIDS, re-classify, re-check protocol, re-export, or re-run analytics. Returns 409 if the step is currently in-flight. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "step", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        step: { type: "string", enum: ["deface", "phi_scan", "qc", "bids", "classify", "protocol", "export", "analytics"], description: "Pipeline step to reset" },
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
    name: "link_studies",
    description: "Create a typed relationship between two studies. Relationship types: 'baseline' (initial scan), 'follow_up' (repeat scan of same subject), 'comparison' (different subjects or conditions), 'replicate' (same protocol repeated). Requires confirm=true and a reason. Returns the created relationship record.",
    inputSchema: {
      type: "object",
      required: ["study_id", "related_study_id", "relationship", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "Source study UUID" },
        related_study_id: { type: "string", format: "uuid", description: "Target study UUID (must differ from study_id)" },
        relationship: { type: "string", enum: ["baseline", "follow_up", "comparison", "replicate"] },
        notes: { type: "string", maxLength: 500, description: "Optional free-text annotation" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "unlink_studies",
    description: "Remove a study relationship by its relationship UUID. Use list_study_relationships first to find the relationship_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "relationship_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "The study that owns the relationship" },
        relationship_id: { type: "string", format: "uuid", description: "Relationship UUID from list_study_relationships" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_digest_subscription",
    description: "Create an email digest subscription for a project. Sends weekly or monthly plain-text study activity summaries (counts only — no PHI). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "email", "frequency", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        email: { type: "string", format: "email", description: "Recipient email address" },
        frequency: { type: "string", enum: ["weekly", "monthly"] },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_digest_subscription",
    description: "Delete an email digest subscription by ID. Use list_digest_subscriptions first to find the subscription_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["subscription_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_webhook_subscription",
    description: "Create a new webhook subscription to receive push notifications on study events. Payloads are signed with HMAC-SHA256 using the provided secret (X-AEGIS-Signature header). Supported events: study.created, study.processing_complete, study.approved, study.rejected, study.phi_flagged, study.export_complete, study.stuck. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["url", "events", "secret", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        url: { type: "string", format: "uri", description: "HTTPS endpoint that receives POST notifications" },
        events: {
          type: "array",
          items: { type: "string", enum: ["study.created", "study.processing_complete", "study.approved", "study.rejected", "study.phi_flagged", "study.export_complete", "study.stuck"] },
          minItems: 1,
          description: "Event names to subscribe to"
        },
        secret: { type: "string", minLength: 8, maxLength: 256, description: "HMAC-SHA256 signing key for X-AEGIS-Signature header" },
        project_id: { type: "string", format: "uuid", description: "Scope to one project (omit for all projects)" },
        enabled: { type: "boolean", description: "Enable or disable subscription on creation (default true)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_webhook_subscription",
    description: "Update an existing webhook subscription — change URL, events, secret, or enabled state. Use list_all_webhooks (get_webhook_deliveries with list) or the admin dashboard Notifications tab to find the subscription_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["subscription_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid" },
        url: { type: "string", format: "uri", description: "New HTTPS endpoint URL (omit to keep current)" },
        events: {
          type: "array",
          items: { type: "string", enum: ["study.created", "study.processing_complete", "study.approved", "study.rejected", "study.phi_flagged", "study.export_complete", "study.stuck"] },
          minItems: 1,
          description: "New event list (omit to keep current)"
        },
        secret: { type: "string", minLength: 8, maxLength: 256, description: "New HMAC-SHA256 signing key (omit to keep current)" },
        enabled: { type: "boolean", description: "Enable or disable the subscription" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_webhook_subscription",
    description: "Permanently delete a webhook subscription and all its delivery history. This cannot be undone — use update_webhook_subscription with enabled=false to pause without deleting. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["subscription_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "retry_webhook_delivery",
    description: "Retry a specific failed webhook delivery attempt. Use get_webhook_deliveries to find a failed delivery_id, then use this tool to re-send it immediately. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["delivery_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        delivery_id: { type: "string", format: "uuid", description: "UUID of the delivery record to retry" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_project",
    description: "Create a new AEGIS project. Slug is auto-derived from name if omitted (lowercase, dashes). Returns 409 Conflict if the slug already exists. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        name: { type: "string", minLength: 1, maxLength: 128, description: "Human-readable project name" },
        slug: { type: "string", minLength: 1, maxLength: 64, pattern: "^[a-z0-9-]+$", description: "URL-safe identifier (auto-derived from name if omitted)" },
        description: { type: "string", maxLength: 1024 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_project",
    description: "Update a project's name, slug, or description. Name is required. Slug is auto-derived from name if omitted. Returns 409 if slug conflict. Use list_projects to find the project_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        name: { type: "string", minLength: 1, maxLength: 128 },
        slug: { type: "string", minLength: 1, maxLength: 64, pattern: "^[a-z0-9-]+$", description: "URL-safe identifier (auto-derived from name if omitted)" },
        description: { type: "string", maxLength: 1024 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "archive_project",
    description: "Archive a project. Archived projects are visually flagged in the dashboard; studies remain accessible. Use restore_project to unarchive. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "restore_project",
    description: "Restore (unarchive) a previously archived project. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "set_project_retention",
    description: "Set or clear the study retention policy for a project. Approved studies older than retention_days will be soft-expired. Pass retention_days=null to keep studies indefinitely (default). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "retention_days", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        retention_days: { type: ["integer", "null"], minimum: 1, description: "Days to retain approved studies, or null to keep indefinitely" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "set_project_sla_threshold",
    description: "Set or clear the per-project stuck-study SLA threshold. Studies idle longer than stuck_threshold_minutes trigger SLA alerts. Pass stuck_threshold_minutes=null to use the global default (60 minutes). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "stuck_threshold_minutes", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        stuck_threshold_minutes: { type: ["integer", "null"], minimum: 1, description: "Minutes before a study is considered stuck, or null to use global default (60)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_admin_users",
    description: "List all registered admin users with their roles, enabled state, and last-seen info. Use before create_admin_user to check if a user already exists, or before update_admin_user/delete_admin_user to find the user_id.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_admin_user",
    description: "Register a new admin dashboard user with email and role. Role 'admin' has full write access; 'viewer' is read-only; 'researcher' is project-scoped (access determined by project_members entries — use add_project_member after creating). The user must already be authenticated via IAP/Azure/AWS — this just registers them in the access control table. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["email", "role", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        email: { type: "string", format: "email", description: "User's email (must match their IAP/Easy Auth identity)" },
        name: { type: "string", minLength: 1, maxLength: 255 },
        role: { type: "string", enum: ["admin", "viewer", "researcher"], description: "'researcher' = project-scoped; must also be added via add_project_member" },
        notes: { type: "string", maxLength: 1024 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_admin_user",
    description: "Update an existing admin user's email, name, role, enabled state, or notes. Use list_admin_users to find the user_id. Setting enabled=false disables access without deleting the record. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["user_id", "email", "role", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        user_id: { type: "string", format: "uuid" },
        email: { type: "string", format: "email" },
        name: { type: "string", minLength: 1, maxLength: 255 },
        role: { type: "string", enum: ["admin", "viewer", "researcher"] },
        enabled: { type: "boolean", description: "false to disable access without deleting" },
        notes: { type: "string", maxLength: 1024 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_admin_user",
    description: "Permanently remove an admin user record. This cannot be undone — use update_admin_user with enabled=false to revoke access without deleting. Use list_admin_users to find the user_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["user_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        user_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "send_admin_invite",
    description: "Send a dashboard invite email to a registered admin user. The email contains the dashboard URL and SSO sign-in instructions (Google, Microsoft, or AWS). Requires SMTP to be configured on the server (SMTP_HOST env var). Use list_admin_users to find the user_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["user_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        user_id: { type: "string", format: "uuid", description: "UUID of the admin user to invite" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "add_project_member",
    description: "Add a user to a project with a specific role. Coordinating center roles (owner/coordinator/reviewer) leave institution_id null — they see all studies. Site roles (site_coordinator/site_viewer) require institution_id — they see only that institution's studies. Use list_admin_users to find the admin_user_id and list_institutions to find institution_id for site roles. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "admin_user_id", "role", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        admin_user_id: { type: "string", format: "uuid", description: "UUID of the admin/researcher user to add" },
        role: { type: "string", enum: ["owner", "coordinator", "reviewer", "site_coordinator", "site_viewer"] },
        institution_id: { type: "string", format: "uuid", description: "Required for site_coordinator and site_viewer; null for coordinating center roles" },
        notes: { type: "string", maxLength: 500 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_project_member",
    description: "Update a project member's role or institution scoping. Use list_project_members to find the member_id. Changing from a site role to a center role requires clearing institution_id (null). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "member_id", "role", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        member_id: { type: "string", format: "uuid" },
        role: { type: "string", enum: ["owner", "coordinator", "reviewer", "site_coordinator", "site_viewer"] },
        institution_id: { type: "string", format: "uuid", description: "Set for site roles; null clears site scoping (coordinating center roles)" },
        notes: { type: "string", maxLength: 500 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "remove_project_member",
    description: "Remove a user from a project. The user loses access to the project immediately. Use list_project_members to find the member_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "member_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        member_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "toggle_project_restricted",
    description: "Set or clear the restricted flag on a project. When restricted=true, only project members and platform admins (admin/viewer role) can see the project — researcher-role users without membership cannot. When restricted=false (default), all authenticated users can see the project. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "restricted", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        restricted: { type: "boolean", description: "true to restrict to members only; false to open to all" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_invite_codes",
    description: "List all beta invite codes with usage stats (used_at, used_by_ip). Use before create_invite_code to check existing codes, or to find a code_id for revoke/send/delete operations.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_invite_requests",
    description: "List access requests submitted by prospective users from the landing page. Returns {requests, total}. Filter by status=pending|approved|denied (omit for all pending). Use before approve_invite_request or deny_invite_request to find the invite_request_id.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        status: { type: "string", enum: ["pending", "approved", "denied", "all"], description: "Filter by request status (omit = all pending)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_invite_code",
    description: "Generate a new XXXX-XXXX-XXXX invite code with a human-readable label (e.g. 'Dr. Smith – Stanford'). The code can be sent to the recipient via send_invite_code or shared manually via the landing page URL ?invite=CODE. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["label", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        label: { type: "string", minLength: 1, maxLength: 256, description: "Human-readable label identifying the intended recipient (e.g. 'Dr. Smith – Stanford')" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "revoke_invite_code",
    description: "Revoke (disable) an invite code so it can no longer be used to access the landing page. The code record is kept for audit purposes. Use list_invite_codes to find the code_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["code_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        code_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_invite_code",
    description: "Permanently delete an invite code record. This cannot be undone — use revoke_invite_code to disable without deleting. Use list_invite_codes to find the code_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["code_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        code_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "send_invite_code",
    description: "Email an invite code directly to a recipient. Requires SMTP to be configured on the server (SMTP_HOST env var). Use list_invite_codes to find the code_id. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["code_id", "email", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        code_id: { type: "string", format: "uuid" },
        email: { type: "string", format: "email", description: "Recipient email address" },
        name: { type: "string", minLength: 1, maxLength: 255, description: "Recipient name (defaults to code label if omitted)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "approve_invite_request",
    description: "Approve a pending access request — creates a new invite code, emails it to the requester, and marks the request approved. Returns {status, invite_code}. Use list_invite_requests to find pending invite_request_id values. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["invite_request_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        invite_request_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "deny_invite_request",
    description: "Deny a pending access request and mark it as denied. Use list_invite_requests to find pending invite_request_id values. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["invite_request_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        invite_request_id: { type: "string", format: "uuid" },
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
    name: "toggle_study_flag",
    description: "Set or clear the priority flag on a study. Flagged studies are marked with ★ in the dashboard and can be filtered with ?flagged=true. Use to mark studies that require urgent review or special handling. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "flagged", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        flagged: { type: "boolean", description: "true to set priority flag, false to clear it" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "clone_project",
    description: "Duplicate a project with all its settings: routing rules (project-scoped), anon profiles (with default profile pointer), protocol templates, PHI config, retention_days, stuck_threshold_minutes. Studies, audit entries, and invite codes are NOT copied. Returns the new project record. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        name: { type: "string", minLength: 1, maxLength: 128, description: "New project name (default: 'Copy of <source>')" },
        slug: { type: "string", minLength: 1, maxLength: 128, description: "New project slug (auto-derived from name if omitted)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "re_evaluate_project_routing",
    description: "Re-apply all enabled routing rules against every study in a project. Use after adding or modifying routing rules to retroactively apply them to existing studies without re-uploading. Optionally filter to a specific study status (e.g. 'approved', 'received') and limit the number of studies processed (default 500, max 2000). Returns {evaluated, status_filter, errors[]}. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID to re-evaluate" },
        status: { type: "string", description: "Optional study status filter (e.g. 'received', 'approved')" },
        limit: { type: "integer", minimum: 1, maximum: 2000, description: "Max studies to process (default 500)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_destination",
    description: "Update a DICOMweb or DIMSE routing destination. Use to fix auth headers (dicomweb_auth_header must include 'Bearer ' prefix), correct URLs, rename destinations, or enable/disable them. Only the fields you provide are updated (others keep their current values). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["destination_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        destination_id: { type: "string", format: "uuid" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        description: { type: "string", maxLength: 512 },
        dicomweb_url: { type: "string", format: "uri", description: "STOW-RS base URL (without /studies suffix)" },
        dicomweb_auth_header: { type: "string", maxLength: 1024, description: "Full Authorization header value, e.g. 'Bearer aegis_...' (must include Bearer prefix)" },
        enabled: { type: "boolean" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_destination",
    description: "Create a new DICOMweb or DIMSE routing destination. DICOMweb destinations require dicomweb_url. DIMSE destinations require ae_title, host, and port. The dicomweb_auth_header must include the 'Bearer ' prefix if set. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["name", "type", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        description: { type: "string", maxLength: 512 },
        type: { type: "string", enum: ["dicomweb", "dimse"] },
        dicomweb_url: { type: "string", format: "uri", description: "Required for type=dicomweb" },
        dicomweb_auth_header: { type: "string", maxLength: 1024, description: "e.g. 'Bearer aegis_...' — full Authorization header value" },
        ae_title: { type: "string", maxLength: 64, description: "Required for type=dimse" },
        host: { type: "string", maxLength: 256, description: "Required for type=dimse" },
        port: { type: "integer", minimum: 1, maximum: 65535, description: "Required for type=dimse" },
        enabled: { type: "boolean" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_destination",
    description: "Permanently delete a routing destination. Any routing rules referencing this destination will lose their destination link. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["destination_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        destination_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_routing_rule",
    description: "Create a new routing rule. Rules are evaluated on every study ingest in priority order (lower = first). All matching rules fire. Use action=route_to with a destination_id to forward DICOM files. Other actions: require_defacing, require_phi_scan, require_qc_check, require_bids_conversion, require_classification, require_protocol_check, require_export, require_analytics, auto_approve, require_qa, reject. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["name", "priority", "action", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        description: { type: "string", maxLength: 512 },
        priority: { type: "integer", minimum: 1, maximum: 9999, description: "Lower number = higher priority; rules are evaluated in ascending priority order" },
        enabled: { type: "boolean" },
        project_id: { type: "string", format: "uuid", description: "Scope rule to a specific project (omit for all projects)" },
        modality: { type: "string", maxLength: 16, description: "Filter by modality e.g. MRI, CT, PET (omit for any)" },
        body_part: { type: "string", maxLength: 64, description: "Filter by body part e.g. HEAD, CHEST (omit for any)" },
        source: { type: "string", enum: ["external", "internal"], description: "Filter by study source (omit for any)" },
        action: { type: "string", enum: ["route_to", "require_defacing", "require_phi_scan", "require_qc_check", "require_bids_conversion", "require_classification", "require_protocol_check", "require_export", "require_analytics", "auto_approve", "require_qa", "reject"] },
        destination_id: { type: "string", format: "uuid", description: "Required when action=route_to" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_routing_rule",
    description: "Update an existing routing rule. Only the fields you provide are updated. Pass null to clear optional fields (project_id, modality, body_part, source, destination_id) so the rule matches any value. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["rule_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        rule_id: { type: "string", format: "uuid" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        description: { type: "string", maxLength: 512 },
        priority: { type: "integer", minimum: 1, maximum: 9999 },
        enabled: { type: "boolean" },
        project_id: { type: ["string", "null"], format: "uuid", description: "null clears project scoping (match any project)" },
        modality: { type: ["string", "null"], maxLength: 16, description: "null matches any modality" },
        body_part: { type: ["string", "null"], maxLength: 64, description: "null matches any body part" },
        source: { type: ["string", "null"], enum: ["external", "internal", null], description: "null matches any source" },
        action: { type: "string", enum: ["route_to", "require_defacing", "require_phi_scan", "require_qc_check", "require_bids_conversion", "require_classification", "require_protocol_check", "require_export", "require_analytics", "auto_approve", "require_qa", "reject"] },
        destination_id: { type: ["string", "null"], format: "uuid", description: "null clears destination link" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_routing_rule",
    description: "Permanently delete a routing rule. Studies already ingested are not affected. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["rule_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        rule_id: { type: "string", format: "uuid" },
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
    name: "get_institution_breakdown",
    description: "Get study counts grouped by institution for a project: total studies, approved, rejected, and pending per institution. Useful for understanding which institutions contribute the most studies.",
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
  },
  {
    name: "list_api_keys",
    description: "List all machine-to-machine API keys. Returns each key's ID, name, prefix (first 14 chars of the key for identification), created_by, enabled status, last_used_at, expires_at, and timestamps. The raw key value is never returned after creation.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_api_key",
    description: "Create a new machine-to-machine API key. Returns the API key record plus the raw key value (shown exactly once — must be copied immediately). The key is used as a Bearer token in the Authorization header. Optionally set an expiry date (RFC3339). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        name: { type: "string", minLength: 1, maxLength: 128, description: "Human-readable key name" },
        expires_at: { type: "string", format: "date-time", description: "Optional RFC3339 expiry; must be in the future" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "rotate_api_key",
    description: "Rotate an API key: generates a new cryptographically random key, updates the stored hash and prefix, and returns the new raw key exactly once. The old key immediately stops working. Use this instead of delete+create to avoid an authentication gap in dependent services. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["key_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        key_id: { type: "string", format: "uuid", description: "API key UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "enable_api_key",
    description: "Enable a disabled API key so it can authenticate requests again. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["key_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        key_id: { type: "string", format: "uuid", description: "API key UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "disable_api_key",
    description: "Disable an active API key so it can no longer authenticate requests. The key record is preserved and can be re-enabled. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["key_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        key_id: { type: "string", format: "uuid", description: "API key UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_api_key",
    description: "Permanently delete an API key. This cannot be undone. The key will immediately stop authenticating requests. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["key_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        key_id: { type: "string", format: "uuid", description: "API key UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "bulk_approve_studies",
    description: "Approve up to 200 studies in a single call. Already-terminal studies (already approved/rejected) are skipped with an error entry. On approval, export forwarding and uploader notification emails fire automatically (same as single approve). Returns {processed, errors[]}. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_ids", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_ids: {
          type: "array",
          items: { type: "string", format: "uuid" },
          minItems: 1,
          maxItems: 200,
          description: "List of study UUIDs to approve (max 200)"
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "bulk_reject_studies",
    description: "Reject up to 200 studies in a single call. Already-terminal studies are skipped with an error entry. Returns {processed, errors[]}. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_ids", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_ids: {
          type: "array",
          items: { type: "string", format: "uuid" },
          minItems: 1,
          maxItems: 200,
          description: "List of study UUIDs to reject (max 200)"
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "bulk_label_studies",
    description: "Add or remove a label across up to 200 studies in a single call. action='add' inserts the label (duplicates silently ignored). action='remove' deletes it (case-insensitive). Returns {applied/removed, total}. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_ids", "label", "action", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_ids: {
          type: "array",
          items: { type: "string", format: "uuid" },
          minItems: 1,
          maxItems: 200,
          description: "List of study UUIDs to label (max 200)"
        },
        label: { type: "string", minLength: 1, maxLength: 80, description: "Label text" },
        action: { type: "string", enum: ["add", "remove"], description: "'add' inserts the label; 'remove' deletes it" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "generate_synthetic_study",
    description: "Generate a synthetic brain MRI DICOM study via the synth-service and import it into AEGIS. Useful for populating test data, validating the pipeline, or producing demo studies. The generated phantom is a Shepp-Logan brain with optional facial anatomy (with_face=true) so the defacing pipeline can be exercised. Returns {study_id, study_uid, file_count, tool_used, duration_seconds}. Requires SYNTH_SERVICE_URL to be configured on the API. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_slug: { type: "string", minLength: 1, maxLength: 64, description: "Target project slug (default: 'default')" },
        slices: { type: "integer", minimum: 1, maximum: 100, description: "Number of axial slices to generate (default: 20)" },
        size: { type: "integer", minimum: 64, maximum: 512, description: "Image matrix size in pixels (default: 256)" },
        seed: { type: "integer", minimum: 0, description: "Random seed for reproducible phantoms (default: random)" },
        with_face: { type: "boolean", description: "Include facial anatomy to exercise the defacing pipeline (default: false)" },
        use_gpu: { type: "boolean", description: "Use GPU-accelerated MONAI BraTS LDM if available (default: false)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_protocol_template",
    description: "Create a new MRI protocol compliance template for a project. The template defines expected acquisition parameters (TR, TE, flip angle, slice thickness, etc.) for a specific scanner/sequence. Rules specify tag_keyword, target value, match_type (numeric/exact/contains_all/range), optional tolerance, and severity (critical/warning/info). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" },
        name: { type: "string", minLength: 1, maxLength: 128, description: "Template name (unique within project)" },
        description: { type: "string", maxLength: 512 },
        manufacturer: { type: "string", maxLength: 128, description: "Scanner manufacturer (e.g. SIEMENS, PHILIPS). Empty = match any." },
        model: { type: "string", maxLength: 128, description: "Scanner model (e.g. MAGNETOM Prisma). Empty = match any." },
        software_version: { type: "string", maxLength: 128, description: "Software version (e.g. VE11C). Empty = match any." },
        sequence_type: { type: "string", maxLength: 128, description: "Pulse sequence identifier (e.g. T1w_MPRAGE, FLAIR, DWI)." },
        rules: {
          type: "array",
          description: "Parameter compliance rules",
          items: {
            type: "object",
            required: ["tag_keyword", "target"],
            properties: {
              tag_keyword: { type: "string", minLength: 1, description: "DICOM keyword (e.g. RepetitionTime, EchoTime)" },
              target: { description: "Expected value — string, number, or array of strings" },
              tolerance: { type: "number", description: "Percentage tolerance for numeric matches (default from PROTOCOL_DEFAULT_TOLERANCE env var)" },
              match_type: { type: "string", enum: ["numeric", "exact", "contains_all", "range"], description: "Comparison type (default: numeric for numbers, exact for strings)" },
              severity: { type: "string", enum: ["critical", "warning", "info"], description: "Violation severity (default: warning)" },
              description: { type: "string", description: "Human-readable description of this rule" }
            },
            additionalProperties: false
          }
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_protocol_template",
    description: "Update an existing protocol compliance template. All fields are optional except template_id — omitted fields are preserved. Use list_protocol_templates to find template UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["template_id", "name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        template_id: { type: "string", format: "uuid", description: "Protocol template UUID" },
        name: { type: "string", minLength: 1, maxLength: 128 },
        description: { type: "string", maxLength: 512 },
        manufacturer: { type: "string", maxLength: 128 },
        model: { type: "string", maxLength: 128 },
        software_version: { type: "string", maxLength: 128 },
        sequence_type: { type: "string", maxLength: 128 },
        rules: {
          type: "array",
          items: {
            type: "object",
            required: ["tag_keyword", "target"],
            properties: {
              tag_keyword: { type: "string", minLength: 1 },
              target: {},
              tolerance: { type: "number" },
              match_type: { type: "string", enum: ["numeric", "exact", "contains_all", "range"] },
              severity: { type: "string", enum: ["critical", "warning", "info"] },
              description: { type: "string" }
            },
            additionalProperties: false
          }
        },
        enabled: { type: "boolean", description: "Enable or disable this template" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_protocol_template",
    description: "Permanently delete a protocol compliance template. This cannot be undone — any routing rules referencing this template will no longer apply it. Use list_protocol_templates to find template UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["template_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        template_id: { type: "string", format: "uuid", description: "Protocol template UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_anon_profile",
    description: "Create a new anonymization profile for a project. The profile defines which DICOM tags are retained (not stripped) during PS3.15 Basic Profile de-identification. retained_tags is an array of DICOM keyword strings (e.g. ['PatientAge', 'StudyDate']). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" },
        name: { type: "string", minLength: 1, maxLength: 128, description: "Profile name (unique within project)" },
        description: { type: "string", maxLength: 512 },
        retained_tags: {
          type: "array",
          items: { type: "string" },
          description: "DICOM keyword strings to retain (e.g. ['PatientAge', 'StudyDate', 'InstitutionName'])"
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_anon_profile",
    description: "Update an existing anonymization profile. All fields are optional except profile_id and name. retained_tags replaces the full existing list when provided. Use list_anon_profiles to find profile UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["profile_id", "name", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        profile_id: { type: "string", format: "uuid", description: "Anonymization profile UUID" },
        name: { type: "string", minLength: 1, maxLength: 128 },
        description: { type: "string", maxLength: 512 },
        retained_tags: {
          type: "array",
          items: { type: "string" },
          description: "Full replacement list of DICOM keyword strings to retain"
        },
        enabled: { type: "boolean", description: "Enable or disable this profile" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_anon_profile",
    description: "Permanently delete an anonymization profile. If this profile is the project's default, the default will be cleared. Use list_anon_profiles to find profile UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["profile_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        profile_id: { type: "string", format: "uuid", description: "Anonymization profile UUID" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "set_default_anon_profile",
    description: "Set or clear the default anonymization profile for a project. The default profile is automatically applied by the upload portal for every upload. Pass profile_id='' (empty string) to clear the default (revert to full PS3.15 strip). Use list_anon_profiles to find profile UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "profile_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" },
        profile_id: { type: "string", description: "Profile UUID to set as default, or empty string to clear" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_institution",
    description: "Register a new institution (hospital, research site, imaging center). type must be 'sender', 'receiver', or 'both'. ip_ranges is an optional comma-separated list of CIDR blocks used for automatic source attribution on internal ingest. ae_title is the DICOM AE title used for DIMSE attribution. Slug is auto-derived from name if omitted. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["name", "type", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        type: { type: "string", enum: ["sender", "receiver", "both"], description: "Institution role" },
        slug: { type: "string", minLength: 1, maxLength: 64, pattern: "^[a-z0-9-]+$", description: "URL-safe identifier (auto-derived from name if omitted)" },
        description: { type: "string", maxLength: 1024 },
        contact_name: { type: "string", maxLength: 256 },
        contact_email: { type: "string", format: "email" },
        ip_ranges: { type: "string", description: "Comma-separated CIDR blocks for IP-based auto-attribution (e.g. '10.0.0.0/8,192.168.1.5')" },
        ae_title: { type: "string", maxLength: 16, description: "DICOM AE title for DIMSE attribution" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_institution",
    description: "Update an existing institution's metadata, type, contact info, IP ranges, or AE title. name and type are required; all others are optional. enabled=false soft-disables the institution. Use list_institutions to find institution UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["institution_id", "name", "type", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        institution_id: { type: "string", format: "uuid" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        type: { type: "string", enum: ["sender", "receiver", "both"] },
        slug: { type: "string", minLength: 1, maxLength: 64, pattern: "^[a-z0-9-]+$" },
        description: { type: "string", maxLength: 1024 },
        contact_name: { type: "string", maxLength: 256 },
        contact_email: { type: "string", format: "email" },
        ip_ranges: { type: "string", description: "Comma-separated CIDR blocks" },
        ae_title: { type: "string", maxLength: 16 },
        enabled: { type: "boolean" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_institution",
    description: "Permanently delete an institution record. Studies that referenced this institution will retain their institution_id FK but the institution row will be gone. Use list_institutions to find institution UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["institution_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        institution_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "link_institution_project",
    description: "Link an institution to a project with a role (sender/receiver/admin). Senders can submit studies; receivers are valid forwarding destinations; admin links grant institution access to all project data. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["institution_id", "project_id", "role", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        institution_id: { type: "string", format: "uuid" },
        project_id: { type: "string", format: "uuid" },
        role: { type: "string", enum: ["sender", "receiver", "admin"], description: "Institution role in this project" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "unlink_institution_project",
    description: "Remove a project link from an institution. Studies already attributed to this institution are not affected. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["institution_id", "project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        institution_id: { type: "string", format: "uuid" },
        project_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "create_federation_peer",
    description: "Register a trusted remote AEGIS instance as a federation peer. This is a stub registry entry for future cross-tenant federation — no data flows yet. name and api_url are required; slug is auto-derived from name if omitted. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["name", "api_url", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        slug: { type: "string", minLength: 1, maxLength: 64, pattern: "^[a-z0-9-]+$", description: "URL-safe identifier (auto-derived from name if omitted)" },
        api_url: { type: "string", format: "uri", description: "Base URL of the remote AEGIS instance" },
        notes: { type: "string", maxLength: 1024 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_federation_peer",
    description: "Update a federation peer's name, URL, notes, or enabled state. Use list_federation_peers to find peer UUIDs. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["peer_id", "name", "api_url", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        peer_id: { type: "string", format: "uuid" },
        name: { type: "string", minLength: 1, maxLength: 256 },
        slug: { type: "string", minLength: 1, maxLength: 64, pattern: "^[a-z0-9-]+$" },
        api_url: { type: "string", format: "uri" },
        notes: { type: "string", maxLength: 1024 },
        enabled: { type: "boolean" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_federation_peer",
    description: "Permanently delete a federation peer registration. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["peer_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        peer_id: { type: "string", format: "uuid" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "set_storage_quota",
    description: "Set or clear the per-project DICOM storage quota in bytes. When set, new uploads are rejected once the project's total storage exceeds this limit. Pass storage_quota_bytes=null to clear the quota (unlimited). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "storage_quota_bytes", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        storage_quota_bytes: { type: ["integer", "null"], minimum: 1, description: "Quota in bytes (e.g. 10737418240 = 10 GB), or null to remove the quota" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "update_phi_config",
    description: "Override the global PHI detection thresholds for a specific project. confidence_threshold (0.0–1.0) is the minimum OCR confidence to flag text; min_text_length is the minimum character count. Omit a field to keep its current value. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        confidence_threshold: { type: "number", minimum: 0, maximum: 1, description: "Minimum OCR confidence to flag (0.0–1.0)" },
        min_text_length: { type: "integer", minimum: 1, description: "Minimum text length to consider as PHI" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "delete_study",
    description: "Permanently delete a study record and ALL associated DICOM files from storage. This action cannot be undone. The study must not be in an actively processing state. Use for removing incorrectly uploaded, duplicate, or test studies. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "Study UUID to permanently delete" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "bulk_pipeline_trigger",
    description: "Trigger a pipeline step for multiple studies in one call. Resets the specified step to 'pending' and lets the auto-pipeline re-dispatch it. Useful for batch re-processing after a service outage or configuration change. step must be one of: classify, phi_scan, protocol, deface, qc, bids, export, analytics. Returns {triggered, skipped, errors[]}. Skipped = step not required or already in-flight. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_ids", "step", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_ids: {
          type: "array",
          items: { type: "string", format: "uuid" },
          minItems: 1,
          maxItems: 200,
          description: "Study UUIDs to trigger (max 200)"
        },
        step: {
          type: "string",
          enum: ["classify", "phi_scan", "protocol", "deface", "qc", "bids", "export", "analytics"],
          description: "Pipeline step to reset and re-trigger"
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_tcia_series",
    description: "List available MRI series from The Cancer Imaging Archive (TCIA) for a given collection. Returns series with their UIDs, modality, body part, description, and slice count. Use before import_tcia_series to find a series_uid. Filters to volumetric MR series with at least min_slices slices (default 20).",
    inputSchema: {
      type: "object",
      required: ["collection"],
      properties: {
        request_id: { type: "string" },
        collection: { type: "string", minLength: 1, maxLength: 128, description: "TCIA collection name (e.g. TCGA-GBM, ADNI, TCIA-Prostate)" },
        min_slices: { type: "integer", minimum: 1, description: "Minimum slice count filter (default 20)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "import_tcia_series",
    description: "Download a DICOM series from The Cancer Imaging Archive (TCIA) and import it into AEGIS. Use get_tcia_series first to find a valid series_uid for a collection. The series is downloaded as a ZIP archive from TCIA's NBIA API, extracted, and processed through the normal AEGIS ingest pipeline. Large series may take several minutes. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["series_uid", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        series_uid: { type: "string", minLength: 1, maxLength: 256, description: "DICOM SeriesInstanceUID from TCIA" },
        collection: { type: "string", minLength: 1, maxLength: 128, description: "TCIA collection name (for logging)" },
        project_slug: { type: "string", minLength: 1, maxLength: 64, description: "Target project slug (default: 'default')" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "export_protocol_templates",
    description: "Export all protocol compliance templates for a project as a structured JSON array. Returns {project_id, templates, count}. Use to inspect current templates, back them up before changes, or copy them to another project via import_protocol_templates.",
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
    name: "import_protocol_templates",
    description: "Bulk-import protocol compliance templates into a project. Templates whose name already exists in the project are silently skipped (no overwrite). Returns {imported, skipped, errors[]}. Accepts up to 200 templates per call. Use export_protocol_templates to get the source format. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "templates", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Destination project UUID" },
        templates: {
          type: "array",
          minItems: 1,
          maxItems: 200,
          description: "Array of template objects to import",
          items: {
            type: "object",
            required: ["name"],
            properties: {
              name: { type: "string", minLength: 1, maxLength: 128 },
              description: { type: "string", maxLength: 512 },
              manufacturer: { type: "string", maxLength: 128 },
              model: { type: "string", maxLength: 128 },
              software_version: { type: "string", maxLength: 128 },
              sequence_type: { type: "string", maxLength: 128 },
              rules: { type: "array", items: {} }
            },
            additionalProperties: false
          }
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_webhook_stats",
    description: "Get aggregate delivery statistics for a single webhook subscription: total deliveries, success/failure counts, success rate percent, last delivery time, and breakdown by event type. Useful for monitoring webhook health.",
    inputSchema: {
      type: "object",
      required: ["subscription_id"],
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid", description: "Webhook subscription UUID" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_all_webhook_deliveries",
    description: "List all webhook delivery attempts across all subscriptions (or filtered to one subscription). Paginated. Optional success filter. Use get_webhook_deliveries for per-subscription history instead.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        subscription_id: { type: "string", format: "uuid", description: "Filter to one subscription" },
        success: { type: "string", enum: ["true", "false"], description: "Filter by success/failure" },
        limit: { type: "number", minimum: 1, maximum: 200, description: "Page size (default 50)" },
        offset: { type: "number", minimum: 0, description: "Row offset for pagination" }
      },
      additionalProperties: false
    }
  },
  {
    name: "batch_import_studies",
    description: "Import DICOM files from a server-local directory into AEGIS. The directory must be accessible on the server's filesystem. Files are assumed already de-identified (no re-anonymization is applied). Supports dry_run=true to preview what would be imported. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["dir", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        dir: { type: "string", minLength: 1, maxLength: 1024, description: "Absolute path on the server to a directory containing DICOM files" },
        project_slug: { type: "string", minLength: 1, maxLength: 64, description: "Target project slug (default: 'default')" },
        institution_id: { type: "string", format: "uuid", description: "Institution UUID for provenance (mutually exclusive with institution_slug)" },
        institution_slug: { type: "string", minLength: 1, maxLength: 64, description: "Institution slug for provenance (mutually exclusive with institution_id)" },
        source: { type: "string", enum: ["internal", "external"], description: "Study source designation (default: 'internal')" },
        dry_run: { type: "boolean", description: "If true, scan and report without importing (default: false)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_user_preferences",
    description: "Get notification preferences for a specific admin user: digest frequency (none/daily/weekly/monthly) and subscribed notify events (study.stuck, pipeline.failed, etc.). Use list_admin_users to find user IDs.",
    inputSchema: {
      type: "object",
      required: ["user_id"],
      properties: {
        request_id: { type: "string" },
        user_id: { type: "string", format: "uuid", description: "Admin user UUID" }
      },
      additionalProperties: false
    }
  },
  {
    name: "set_user_preferences",
    description: "Update notification preferences for an admin user. Sets the email digest frequency (none/daily/weekly/monthly) and which system events trigger personal email notifications. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["user_id", "digest_frequency", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        user_id: { type: "string", format: "uuid", description: "Admin user UUID" },
        digest_frequency: { type: "string", enum: ["none", "daily", "weekly", "monthly"], description: "How often to receive email digests" },
        notify_events: {
          type: "array",
          maxItems: 20,
          items: { type: "string" },
          description: "Event types that trigger immediate email: 'study.stuck', 'pipeline.failed', 'study.approved', 'study.rejected', 'study.phi_flagged', 'study.export_complete'"
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_project_bids_info",
    description: "Get metadata about BIDS-converted studies available for bulk download in a project: count, list of study UIDs, and the download URL for the merged ZIP archive. Use before directing a user to download via the /api/projects/{id}/bids-export endpoint. Optional status filter (default: 'approved').",
    inputSchema: {
      type: "object",
      required: ["project_id"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Project UUID" },
        status: { type: "string", enum: ["approved", "received", "clean", "defaced"], description: "Study status filter (default: 'approved')" }
      },
      additionalProperties: false
    }
  },
  {
    name: "export_routing_rules",
    description: "Export all project-scoped routing rules for a project as a structured JSON payload. Returns {project_id, rules, count}. Only rules whose project_id matches the given project are included — global rules (project_id IS NULL) are excluded. Use to inspect, back up, or copy routing rules to another project via import_routing_rules.",
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
    name: "import_routing_rules",
    description: "Bulk-import routing rules into a project from a structured array. Rules whose name already exists in the project are silently skipped (no overwrite). Returns {imported, skipped, errors[]}. Note: destination_id values are stripped during import because destinations are instance-specific — reassign route_to destinations via update_routing_rule after import. Use export_routing_rules to get the source format. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["project_id", "rules", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Destination project UUID" },
        rules: {
          type: "array",
          minItems: 1,
          maxItems: 200,
          description: "Array of routing rule objects to import",
          items: {
            type: "object",
            required: ["name", "action"],
            properties: {
              name: { type: "string", minLength: 1, maxLength: 128 },
              description: { type: "string", maxLength: 512 },
              priority: { type: "number", minimum: 0 },
              enabled: { type: "boolean" },
              modality: { type: "string", maxLength: 16 },
              body_part: { type: "string", maxLength: 64 },
              source: { type: "string", enum: ["external", "internal"] },
              action: { type: "string", minLength: 1, maxLength: 64 }
            },
            additionalProperties: false
          }
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "reorder_routing_rules",
    description: "Atomically update the priority of multiple routing rules in a single transaction. Pass an array of {id, priority} objects. The caller is responsible for providing a consistent, non-conflicting set of priorities. Use list_routing_rules to get current priorities. Returns {rules, updated} with the full updated rule list sorted by new priority. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["rules", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        rules: {
          type: "array",
          minItems: 1,
          maxItems: 200,
          description: "Array of {id, priority} pairs to update",
          items: {
            type: "object",
            required: ["id", "priority"],
            properties: {
              id: { type: "string", format: "uuid", description: "Routing rule UUID" },
              priority: { type: "number", minimum: 0, description: "New priority value (lower = evaluated first)" }
            },
            additionalProperties: false
          }
        },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "bulk_toggle_routing_rules",
    description: "Enable or disable multiple routing rules in a single atomic operation. Pass an array of rule UUIDs and the desired enabled state. Returns {updated, rule_ids} with the count and IDs of rules that were changed. Unknown IDs are silently ignored. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["rule_ids", "enabled", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        rule_ids: {
          type: "array",
          minItems: 1,
          maxItems: 200,
          description: "UUIDs of routing rules to enable or disable",
          items: { type: "string", format: "uuid" }
        },
        enabled: { type: "boolean", description: "true to enable, false to disable" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "query_pacs",
    description: "Send a DICOM C-FIND query to a remote PACS to discover studies, series, or patients. Returns a list of matching datasets as key/value pairs. Useful for finding studies before retrieving them with retrieve_pacs_study. Requires the DIMSE receiver sidecar to be configured.",
    inputSchema: {
      type: "object",
      required: ["ae_title", "host", "port"],
      properties: {
        request_id: { type: "string" },
        ae_title: { type: "string", minLength: 1, maxLength: 64, description: "Remote PACS AE title (e.g. PACS-SERVER)" },
        host: { type: "string", minLength: 1, maxLength: 256, description: "Remote PACS hostname or IP address" },
        port: { type: "integer", minimum: 1, maximum: 65535, description: "Remote PACS DICOM port (typically 104 or 11112)" },
        query_level: { type: "string", enum: ["PATIENT", "STUDY", "SERIES"], description: "Query/retrieve level (default: STUDY)" },
        query_params: {
          type: "object",
          description: "Optional DICOM keyword filter map (e.g. {\"PatientID\": \"12345\", \"StudyDate\": \"20240101-20241231\"}). Empty-string values act as wildcard matches.",
          additionalProperties: { type: "string" }
        }
      },
      additionalProperties: false
    }
  },
  {
    name: "retrieve_pacs_study",
    description: "Send a DICOM C-MOVE request to a remote PACS to push a study to the AEGIS SCP (move_destination AE). Files arrive via the existing DIMSE SCP ingest path. Requires confirm=true and a reason. The move_destination AE must be configured on the remote PACS to allow the push.",
    inputSchema: {
      type: "object",
      required: ["ae_title", "host", "port", "study_instance_uid", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        ae_title: { type: "string", minLength: 1, maxLength: 64, description: "Remote PACS AE title" },
        host: { type: "string", minLength: 1, maxLength: 256, description: "Remote PACS hostname or IP address" },
        port: { type: "integer", minimum: 1, maximum: 65535, description: "Remote PACS DICOM port" },
        study_instance_uid: { type: "string", minLength: 4, maxLength: 256, pattern: "^[0-9.]+$", description: "DICOM StudyInstanceUID to retrieve" },
        move_destination: { type: "string", maxLength: 64, description: "Destination AE title for the push (defaults to AEGIS SCP AE when omitted)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_deleted_studies",
    description: "List soft-deleted studies (moved to trash). Returns studies where deleted_at is set, ordered newest deletion first. Complements list_studies which always excludes deleted studies. Use restore_study to recover a study or delete_study to permanently remove it.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Scope to a specific project UUID; omit for all projects" },
        limit: { type: "integer", minimum: 1, maximum: 200, description: "Page size (default 50)" },
        offset: { type: "integer", minimum: 0, description: "Row offset for pagination (default 0)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "soft_delete_study",
    description: "Move a study to the trash (soft-delete). The study is hidden from normal listings but not permanently removed — use restore_study to recover it or delete_study to permanently delete. Fires study.soft_deleted webhook event. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "Study UUID to soft-delete" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "restore_study",
    description: "Restore a soft-deleted study from the trash, making it visible in normal listings again. Fires study.restored webhook event. Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "Study UUID to restore from trash" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_protocol_trend",
    description: "Returns daily protocol compliance counts over the last N days. Shows how many studies per day had compliant, minor_deviations, or non_compliant protocol status, plus a compliance_pct per day. Useful for spotting degrading scanner compliance over time.",
    inputSchema: {
      type: "object",
      properties: {
        request_id: { type: "string" },
        project_id: { type: "string", format: "uuid", description: "Scope to a specific project UUID; omit for all projects" },
        days: { type: "integer", minimum: 1, maximum: 365, description: "Look-back window in days (default 30)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_study_notes",
    description: "List all operator notes recorded for a study. Notes are stored as audit entries (action='study.note') and returned newest-first. Use add_study_note to add new notes. Note text and author (actor email) are included.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid", description: "Study UUID (database ID, not DICOM UID)" }
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
      if (parsed.flagged) query.set("flagged", "true");
      if (parsed.study_date_from) query.set("study_date_from", parsed.study_date_from);
      if (parsed.study_date_to) query.set("study_date_to", parsed.study_date_to);
      if (parsed.sort_by) query.set("sort_by", parsed.sort_by);
      if (parsed.sort_dir) query.set("sort_dir", parsed.sort_dir);

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

    if (name === "get_study_processing_summary") {
      const parsed = studyIdArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/processing-summary`);
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

    if (name === "get_export_shares_csv") {
      const parsed = exportSharesCsvArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      if (parsed.status) params.set("status", parsed.status);
      const qs = params.toString();
      const data = await client.get(`/api/export-shares.csv${qs ? "?" + qs : ""}`);
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

    if (name === "get_expiring_studies") {
      const parsed = getExpiringStudiesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
      const qs = params.toString();
      const data = await client.get(`/api/studies/expiring${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_retention_preview") {
      const parsed = getRetentionPreviewArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/retention-preview${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_label_usage") {
      const parsed = projectScopedArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/label-usage${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_modality_trend") {
      const parsed = processingTimesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/modality-trend${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_source_trend") {
      const parsed = processingTimesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/source-trend${qs ? "?" + qs : ""}`);
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

    if (name === "get_processing_stats") {
      const parsed = processingTimesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/processing-times${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_routing_stats") {
      const parsed = routingStatsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/stats/routing${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_destination_stats") {
      const parsed = destinationStatsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/destinations/${parsed.destination_id}/stats${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_destination_health") {
      const parsed = getDestinationHealthArgsSchema.parse(args);
      if (parsed.destination_id) {
        const params = new URLSearchParams();
        if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
        const qs = params.toString();
        const data = await client.get(`/api/destinations/${encodeURIComponent(parsed.destination_id)}/health${qs ? "?" + qs : ""}`);
        return formatSuccess(requestId, name, data);
      } else {
        const data = await client.get("/api/destinations/health");
        return formatSuccess(requestId, name, data);
      }
    }

    if (name === "simulate_routing") {
      const parsed = simulateRoutingArgsSchema.parse(args);
      const body: Record<string, string> = {};
      if (parsed.project_id) body.project_id = parsed.project_id;
      if (parsed.modality) body.modality = parsed.modality;
      if (parsed.body_part) body.body_part = parsed.body_part;
      if (parsed.source) body.source = parsed.source;
      const data = await client.post("/api/routing-rules/simulate", body);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_routing_rule_stats") {
      const parsed = routingRuleStatsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/stats/routing-rules${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_pipeline_funnel") {
      const parsed = pipelineFunnelArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id !== undefined) params.set("project_id", parsed.project_id);
      const qs = params.toString();
      const data = await client.get(`/api/stats/pipeline-funnel${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_project_health") {
      const parsed = projectHealthArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      if (parsed.project_id !== undefined) params.set("project_id", parsed.project_id);
      if (parsed.stuck_minutes !== undefined) params.set("stuck_minutes", String(parsed.stuck_minutes));
      const qs = params.toString();
      const data = await client.get(`/api/stats/project-health${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_compliance_report") {
      const parsed = complianceReportArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/projects/${parsed.project_id}/compliance-report${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_compliance_report_csv") {
      const parsed = complianceReportArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.days !== undefined) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/projects/${parsed.project_id}/compliance-report.csv${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_cohort_report") {
      const parsed = getCohortReportArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/cohort-report`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "export_routing_rules") {
      const parsed = exportRoutingRulesArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/routing-rules/export`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_storage_usage") {
      const parsed = storageUsageArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${parsed.project_id}/storage-usage`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_anonymization_diff") {
      const parsed = anonDiffArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_uid)}/anonymization-diff`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_system_health_summary") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/system/health-summary");
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

    if (name === "list_project_members") {
      const parsed = listProjectMembersArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/members`);
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

    if (name === "test_destination") {
      const parsed = testDestinationArgsSchema.parse(args);
      const data = await client.post(`/api/destinations/${encodeURIComponent(parsed.destination_id)}/test`);
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

    if (name === "list_study_relationships") {
      const parsed = listStudyRelationshipsArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${parsed.study_id}/relationships`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_daily_summary") {
      const parsed = getDailySummaryArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.hours !== undefined) params.set("hours", String(parsed.hours));
      const qs = params.toString();
      const data = await client.get(`/api/stats/daily-summary${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_digest_subscriptions") {
      const parsed = listDigestSubscriptionsArgsSchema.parse(args);
      if (parsed.project_id) {
        const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/digest-subscriptions`);
        return formatSuccess(requestId, name, data);
      }
      const data = await client.get("/api/digest-subscriptions");
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

    if (name === "get_institution_breakdown") {
      const parsed = complianceReportArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/institution-breakdown`);
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

    if (name === "list_api_keys") {
      emptyArgsSchema.parse(args);
      const data = await client.get("/api/api-keys");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_admin_users") {
      listAdminUsersArgsSchema.parse(args);
      const data = await client.get("/api/admin-users");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_invite_codes") {
      listInviteCodesArgsSchema.parse(args);
      const data = await client.get("/api/invite-codes");
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_invite_requests") {
      const parsedIR = listInviteRequestsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsedIR.status && parsedIR.status !== "all") params.set("status", parsedIR.status);
      const qs = params.toString();
      const data = await client.get(`/api/invite/requests${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_tcia_series") {
      const parsed = getTCIASeriesArgsSchema.parse(args);
      const params = new URLSearchParams({ collection: parsed.collection });
      if (parsed.min_slices !== undefined) params.set("min_slices", String(parsed.min_slices));
      const data = await client.get(`/api/tcia/series?${params.toString()}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "export_protocol_templates") {
      const parsed = exportProtocolTemplatesArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/protocol-templates/export`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_webhook_stats") {
      const parsed = getWebhookStatsArgsSchema.parse(args);
      const data = await client.get(`/api/webhook-subscriptions/${encodeURIComponent(parsed.subscription_id)}/stats`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_all_webhook_deliveries") {
      const parsed = listAllWebhookDeliveriesArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.subscription_id) params.set("subscription_id", parsed.subscription_id);
      if (parsed.success !== undefined) params.set("success", parsed.success);
      if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
      if (parsed.offset !== undefined) params.set("offset", String(parsed.offset));
      const qs = params.toString();
      const data = await client.get(`/api/webhook-deliveries${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_user_preferences") {
      const parsed = getUserPreferencesArgsSchema.parse(args);
      const data = await client.get(`/api/admin-users/${encodeURIComponent(parsed.user_id)}/preferences`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_project_bids_info") {
      const parsed = getProjectBidsInfoArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.status) params.set("status", parsed.status);
      const qs = params.toString();
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/bids-info${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_deleted_studies") {
      const params = new URLSearchParams();
      if (typeof args.project_id === "string") params.set("project_id", args.project_id);
      if (typeof args.limit === "number") params.set("limit", String(args.limit));
      if (typeof args.offset === "number") params.set("offset", String(args.offset));
      const qs = params.toString();
      const data = await client.get(`/api/studies/deleted${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_protocol_trend") {
      const parsed = getProtocolTrendArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      if (parsed.days) params.set("days", String(parsed.days));
      const qs = params.toString();
      const data = await client.get(`/api/stats/protocol-trend${qs ? "?" + qs : ""}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_study_notes") {
      const parsed = listStudyNotesArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/notes`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "query_pacs") {
      const parsed = queryPacsArgsSchema.parse(args);
      const body: Record<string, unknown> = {
        ae_title: parsed.ae_title,
        host: parsed.host,
        port: parsed.port,
        query_level: parsed.query_level ?? "STUDY"
      };
      if (parsed.query_params) body.query_params = parsed.query_params;
      const data = await client.post("/api/dimse/query", body);
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

      if (name === "link_studies") {
        const parsedLink = linkStudiesArgsSchema.parse(args);
        return handleLinkStudies(parsedLink.request_id ?? buildRequestId(), parsedLink);
      }

      if (name === "unlink_studies") {
        const parsedUnlink = unlinkStudiesArgsSchema.parse(args);
        return handleUnlinkStudies(parsedUnlink.request_id ?? buildRequestId(), parsedUnlink);
      }

      if (name === "create_digest_subscription") {
        const parsedSub = createDigestSubscriptionArgsSchema.parse(args);
        return handleCreateDigestSubscription(parsedSub.request_id ?? buildRequestId(), parsedSub);
      }

      if (name === "delete_digest_subscription") {
        const parsedDelSub = deleteDigestSubscriptionArgsSchema.parse(args);
        return handleDeleteDigestSubscription(parsedDelSub.request_id ?? buildRequestId(), parsedDelSub);
      }

      if (name === "create_webhook_subscription") {
        const parsedCWH = createWebhookSubscriptionArgsSchema.parse(args);
        return handleCreateWebhookSubscription(parsedCWH.request_id ?? buildRequestId(), parsedCWH);
      }

      if (name === "update_webhook_subscription") {
        const parsedUWH = updateWebhookSubscriptionArgsSchema.parse(args);
        return handleUpdateWebhookSubscription(parsedUWH.request_id ?? buildRequestId(), parsedUWH);
      }

      if (name === "delete_webhook_subscription") {
        const parsedDWH = deleteWebhookSubscriptionArgsSchema.parse(args);
        return handleDeleteWebhookSubscription(parsedDWH.request_id ?? buildRequestId(), parsedDWH);
      }

      if (name === "retry_webhook_delivery") {
        const parsedRWD = retryWebhookDeliveryArgsSchema.parse(args);
        return handleRetryWebhookDelivery(parsedRWD.request_id ?? buildRequestId(), parsedRWD);
      }

      if (name === "create_project") {
        const parsedCP = createProjectArgsSchema.parse(args);
        return handleCreateProject(parsedCP.request_id ?? buildRequestId(), parsedCP);
      }

      if (name === "update_project") {
        const parsedUP = updateProjectArgsSchema.parse(args);
        return handleUpdateProject(parsedUP.request_id ?? buildRequestId(), parsedUP);
      }

      if (name === "archive_project") {
        const parsedAP = archiveRestoreProjectArgsSchema.parse(args);
        return handleArchiveProject(parsedAP.request_id ?? buildRequestId(), parsedAP);
      }

      if (name === "restore_project") {
        const parsedRP = archiveRestoreProjectArgsSchema.parse(args);
        return handleRestoreProject(parsedRP.request_id ?? buildRequestId(), parsedRP);
      }

      if (name === "set_project_retention") {
        const parsedPR = setProjectRetentionArgsSchema.parse(args);
        return handleSetProjectRetention(parsedPR.request_id ?? buildRequestId(), parsedPR);
      }

      if (name === "set_project_sla_threshold") {
        const parsedSLA = setProjectSLAThresholdArgsSchema.parse(args);
        return handleSetProjectSLAThreshold(parsedSLA.request_id ?? buildRequestId(), parsedSLA);
      }

      if (name === "add_project_member") {
        const parsed = addProjectMemberArgsSchema.parse(args);
        return handleAddProjectMember(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "update_project_member") {
        const parsed = updateProjectMemberArgsSchema.parse(args);
        return handleUpdateProjectMember(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "remove_project_member") {
        const parsed = removeProjectMemberArgsSchema.parse(args);
        return handleRemoveProjectMember(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "toggle_project_restricted") {
        const parsed = toggleProjectRestrictedArgsSchema.parse(args);
        return handleToggleProjectRestricted(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "create_admin_user") {
        const parsedCAU = createAdminUserArgsSchema.parse(args);
        return handleCreateAdminUser(parsedCAU.request_id ?? buildRequestId(), parsedCAU);
      }

      if (name === "update_admin_user") {
        const parsedUAU = updateAdminUserArgsSchema.parse(args);
        return handleUpdateAdminUser(parsedUAU.request_id ?? buildRequestId(), parsedUAU);
      }

      if (name === "delete_admin_user") {
        const parsedDAU = deleteAdminUserArgsSchema.parse(args);
        return handleDeleteAdminUser(parsedDAU.request_id ?? buildRequestId(), parsedDAU);
      }

      if (name === "send_admin_invite") {
        const parsedSAI = sendAdminInviteArgsSchema.parse(args);
        return handleSendAdminInvite(parsedSAI.request_id ?? buildRequestId(), parsedSAI);
      }

      if (name === "create_invite_code") {
        const parsedCIC = createInviteCodeArgsSchema.parse(args);
        return handleCreateInviteCode(parsedCIC.request_id ?? buildRequestId(), parsedCIC);
      }

      if (name === "revoke_invite_code") {
        const parsedRIC = inviteCodeIdArgsSchema.parse(args);
        return handleRevokeInviteCode(parsedRIC.request_id ?? buildRequestId(), parsedRIC);
      }

      if (name === "delete_invite_code") {
        const parsedDIC = inviteCodeIdArgsSchema.parse(args);
        return handleDeleteInviteCode(parsedDIC.request_id ?? buildRequestId(), parsedDIC);
      }

      if (name === "send_invite_code") {
        const parsedSIC = sendInviteCodeArgsSchema.parse(args);
        return handleSendInviteCode(parsedSIC.request_id ?? buildRequestId(), parsedSIC);
      }

      if (name === "approve_invite_request") {
        const parsedAIR = inviteRequestActionArgsSchema.parse(args);
        return handleApproveInviteRequest(parsedAIR.request_id ?? buildRequestId(), parsedAIR);
      }

      if (name === "deny_invite_request") {
        const parsedDIR = inviteRequestActionArgsSchema.parse(args);
        return handleDenyInviteRequest(parsedDIR.request_id ?? buildRequestId(), parsedDIR);
      }

      if (name === "create_protocol_template") {
        const parsed = createProtocolTemplateArgsSchema.parse(args);
        return handleCreateProtocolTemplate(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "update_protocol_template") {
        const parsed = updateProtocolTemplateArgsSchema.parse(args);
        return handleUpdateProtocolTemplate(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "delete_protocol_template") {
        const parsed = templateIdArgsSchema.parse(args);
        return handleDeleteProtocolTemplate(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "create_anon_profile") {
        const parsed = createAnonProfileArgsSchema.parse(args);
        return handleCreateAnonProfile(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "update_anon_profile") {
        const parsed = updateAnonProfileArgsSchema.parse(args);
        return handleUpdateAnonProfile(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "delete_anon_profile") {
        const parsed = deleteAnonProfileArgsSchema.parse(args);
        return handleDeleteAnonProfile(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "set_default_anon_profile") {
        const parsed = setDefaultAnonProfileArgsSchema.parse(args);
        return handleSetDefaultAnonProfile(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "create_institution") {
        const parsed = createInstitutionArgsSchema.parse(args);
        return handleCreateInstitution(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "update_institution") {
        const parsed = updateInstitutionArgsSchema.parse(args);
        return handleUpdateInstitution(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "delete_institution") {
        const parsed = institutionIdArgsSchema.parse(args);
        return handleDeleteInstitution(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "link_institution_project") {
        const parsed = linkInstitutionProjectArgsSchema.parse(args);
        return handleLinkInstitutionProject(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "unlink_institution_project") {
        const parsed = unlinkInstitutionProjectArgsSchema.parse(args);
        return handleUnlinkInstitutionProject(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "create_federation_peer") {
        const parsed = createFederationPeerArgsSchema.parse(args);
        return handleCreateFederationPeer(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "update_federation_peer") {
        const parsed = updateFederationPeerArgsSchema.parse(args);
        return handleUpdateFederationPeer(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "delete_federation_peer") {
        const parsed = federationPeerIdArgsSchema.parse(args);
        return handleDeleteFederationPeer(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "set_storage_quota") {
        const parsed = setStorageQuotaArgsSchema.parse(args);
        return handleSetStorageQuota(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "update_phi_config") {
        const parsed = updatePhiConfigArgsSchema.parse(args);
        return handleUpdatePhiConfig(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "delete_study") {
        const parsed = deleteStudyArgsSchema.parse(args);
        return handleDeleteStudy(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "bulk_pipeline_trigger") {
        const parsed = bulkPipelineTriggerArgsSchema.parse(args);
        return handleBulkPipelineTrigger(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "import_tcia_series") {
        const parsed = importTCIASeriesArgsSchema.parse(args);
        return handleImportTCIASeries(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "import_protocol_templates") {
        const parsed = importProtocolTemplatesArgsSchema.parse(args);
        return handleImportProtocolTemplates(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "import_routing_rules") {
        const parsed = importRoutingRulesArgsSchema.parse(args);
        return handleImportRoutingRules(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "reorder_routing_rules") {
        const parsed = reorderRoutingRulesArgsSchema.parse(args);
        return handleReorderRoutingRules(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "bulk_toggle_routing_rules") {
        const parsed = bulkToggleRoutingRulesArgsSchema.parse(args);
        return handleBulkToggleRoutingRules(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "batch_import_studies") {
        const parsed = batchImportStudiesArgsSchema.parse(args);
        return handleBatchImportStudies(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "set_user_preferences") {
        const parsed = setUserPreferencesArgsSchema.parse(args);
        return handleSetUserPreferences(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "add_study_note") {
        const parsedNote = addStudyNoteArgsSchema.parse(args);
        return handleAddStudyNote(parsedNote.request_id ?? buildRequestId(), parsedNote);
      }

      if (name === "bulk_create_shares") {
        const parsed = bulkCreateSharesArgsSchema.parse(args);
        return handleBulkCreateShares(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "toggle_study_flag") {
        const parsedFlag = toggleStudyFlagArgsSchema.parse(args);
        return handleToggleStudyFlag(parsedFlag.request_id ?? buildRequestId(), parsedFlag);
      }

      if (name === "clone_project") {
        const parsed = cloneProjectArgsSchema.parse(args);
        return handleCloneProject(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "re_evaluate_project_routing") {
        const parsed = reEvaluateProjectRoutingArgsSchema.parse(args);
        const requestId = parsed.request_id ?? buildRequestId();
        const params = new URLSearchParams();
        if (parsed.status) params.set("status", parsed.status);
        if (parsed.limit) params.set("limit", String(parsed.limit));
        const qs = params.toString() ? `?${params.toString()}` : "";
        const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/re-evaluate-routing${qs}`, {});
        return formatSuccess(requestId, "re_evaluate_project_routing", {
          accepted: true,
          project_id: parsed.project_id,
          reason: parsed.reason,
          result: data
        });
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

      if (name === "create_api_key") {
        const parsedCreate = createApiKeyArgsSchema.parse(args);
        return handleCreateApiKey(parsedCreate.request_id ?? buildRequestId(), parsedCreate);
      }

      if (name === "rotate_api_key") {
        const parsedRotate = apiKeyIdArgsSchema.parse(args);
        return handleRotateApiKey(parsedRotate.request_id ?? buildRequestId(), parsedRotate);
      }

      if (name === "enable_api_key") {
        const parsedEnable = apiKeyIdArgsSchema.parse(args);
        return handleSetApiKeyEnabled(parsedEnable.request_id ?? buildRequestId(), parsedEnable, true);
      }

      if (name === "disable_api_key") {
        const parsedDisable = apiKeyIdArgsSchema.parse(args);
        return handleSetApiKeyEnabled(parsedDisable.request_id ?? buildRequestId(), parsedDisable, false);
      }

      if (name === "delete_api_key") {
        const parsedDelete = apiKeyIdArgsSchema.parse(args);
        return handleDeleteApiKey(parsedDelete.request_id ?? buildRequestId(), parsedDelete);
      }

      if (name === "bulk_approve_studies") {
        const parsedBulkApprove = bulkStudyActionArgsSchema.parse(args);
        return handleBulkStudyAction(parsedBulkApprove.request_id ?? buildRequestId(), parsedBulkApprove, "approve");
      }

      if (name === "bulk_reject_studies") {
        const parsedBulkReject = bulkStudyActionArgsSchema.parse(args);
        return handleBulkStudyAction(parsedBulkReject.request_id ?? buildRequestId(), parsedBulkReject, "reject");
      }

      if (name === "bulk_label_studies") {
        const parsedBulkLabel = bulkLabelStudiesArgsSchema.parse(args);
        return handleBulkLabelStudies(parsedBulkLabel.request_id ?? buildRequestId(), parsedBulkLabel);
      }

      if (name === "generate_synthetic_study") {
        const parsedSynth = generateSyntheticStudyArgsSchema.parse(args);
        return handleGenerateSyntheticStudy(parsedSynth.request_id ?? buildRequestId(), parsedSynth);
      }

      if (name === "update_destination") {
        const parsedDest = updateDestinationArgsSchema.parse(args);
        return handleUpdateDestination(parsedDest.request_id ?? buildRequestId(), parsedDest);
      }

      if (name === "create_destination") {
        const parsedDest = createDestinationArgsSchema.parse(args);
        return handleCreateDestination(parsedDest.request_id ?? buildRequestId(), parsedDest);
      }

      if (name === "delete_destination") {
        const parsedDest = destinationIdArgsSchema.parse(args);
        return handleDeleteDestination(parsedDest.request_id ?? buildRequestId(), parsedDest);
      }

      if (name === "create_routing_rule") {
        const parsedRule = createRoutingRuleArgsSchema.parse(args);
        return handleCreateRoutingRule(parsedRule.request_id ?? buildRequestId(), parsedRule);
      }

      if (name === "update_routing_rule") {
        const parsedRule = updateRoutingRuleArgsSchema.parse(args);
        return handleUpdateRoutingRule(parsedRule.request_id ?? buildRequestId(), parsedRule);
      }

      if (name === "delete_routing_rule") {
        const parsedRule = routingRuleIdArgsSchema.parse(args);
        return handleDeleteRoutingRule(parsedRule.request_id ?? buildRequestId(), parsedRule);
      }

      if (name === "trigger_longitudinal_analytics") {
        const laParsed = triggerLongitudinalAnalyticsArgsSchema.parse(args);
        return handleTriggerLongitudinalAnalytics(laParsed.request_id ?? buildRequestId(), laParsed);
      }

      const parsed = writeArgsSchema.parse(args);

      if (name === "trigger_classification") {
        return handleTriggerClassification(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_bids_convert") {
        return handleTriggerBidsConvert(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "trigger_analytics") {
        return handleTriggerAnalytics(parsed.request_id ?? buildRequestId(), parsed);
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

      if (name === "trigger_pixel_redaction") {
        return handleTriggerPixelRedaction(requestId, parsed);
      }

      if (name === "retrieve_pacs_study") {
        const parsed = retrievePacsStudyArgsSchema.parse(args);
        return handleRetrievePacsStudy(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "soft_delete_study") {
        const parsed = softDeleteStudyArgsSchema.parse(args);
        return handleSoftDeleteStudy(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "restore_study") {
        const parsed = restoreStudyArgsSchema.parse(args);
        return handleRestoreStudy(parsed.request_id ?? buildRequestId(), parsed);
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
    name === "reactivate_study" ||
    name === "toggle_study_flag" ||
    name === "link_studies" ||
    name === "unlink_studies"
  ) {
    return typeof args.study_id === "string" ? args.study_id : null;
  }
  if (name === "revoke_share" || name === "extend_share") {
    return typeof args.share_id === "string" ? args.share_id : null;
  }
  if (name === "export_project_batch" || name === "clone_project" || name === "re_evaluate_project_routing") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "test_webhook") {
    return typeof args.subscription_id === "string" ? args.subscription_id : null;
  }
  if (
    name === "rotate_api_key" ||
    name === "enable_api_key" ||
    name === "disable_api_key" ||
    name === "delete_api_key"
  ) {
    return typeof args.key_id === "string" ? args.key_id : null;
  }
  if (name === "create_api_key") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "bulk_approve_studies" || name === "bulk_reject_studies" || name === "bulk_label_studies" || name === "bulk_create_shares") {
    const ids = args.study_ids;
    if (Array.isArray(ids) && ids.length > 0) {
      return `${ids.length} studies`;
    }
    return null;
  }
  if (name === "generate_synthetic_study") {
    return typeof args.project_slug === "string" ? args.project_slug : "default";
  }
  if (name === "update_destination" || name === "delete_destination") {
    return typeof args.destination_id === "string" ? args.destination_id : null;
  }
  if (name === "create_destination") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "create_routing_rule") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "update_routing_rule" || name === "delete_routing_rule") {
    return typeof args.rule_id === "string" ? args.rule_id : null;
  }
  if (name === "create_digest_subscription" || name === "delete_digest_subscription") {
    return typeof args.project_id === "string" ? args.project_id : (typeof args.subscription_id === "string" ? args.subscription_id : null);
  }
  if (name === "create_webhook_subscription") {
    return typeof args.url === "string" ? args.url : null;
  }
  if (name === "update_webhook_subscription" || name === "delete_webhook_subscription") {
    return typeof args.subscription_id === "string" ? args.subscription_id : null;
  }
  if (name === "retry_webhook_delivery") {
    return typeof args.delivery_id === "string" ? args.delivery_id : null;
  }
  if (name === "create_project") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (
    name === "update_project" ||
    name === "archive_project" ||
    name === "restore_project" ||
    name === "set_project_retention" ||
    name === "set_project_sla_threshold"
  ) {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "add_project_member") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "update_project_member" || name === "remove_project_member") {
    return typeof args.member_id === "string" ? args.member_id : null;
  }
  if (name === "toggle_project_restricted") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "create_admin_user") {
    return typeof args.email === "string" ? args.email : null;
  }
  if (name === "update_admin_user" || name === "delete_admin_user" || name === "send_admin_invite") {
    return typeof args.user_id === "string" ? args.user_id : null;
  }
  if (name === "create_invite_code") {
    return typeof args.label === "string" ? args.label : null;
  }
  if (
    name === "revoke_invite_code" ||
    name === "delete_invite_code" ||
    name === "send_invite_code"
  ) {
    return typeof args.code_id === "string" ? args.code_id : null;
  }
  if (name === "approve_invite_request" || name === "deny_invite_request") {
    return typeof args.invite_request_id === "string" ? args.invite_request_id : null;
  }
  if (name === "create_protocol_template") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "update_protocol_template" || name === "delete_protocol_template") {
    return typeof args.template_id === "string" ? args.template_id : null;
  }
  if (name === "create_anon_profile") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "update_anon_profile" || name === "delete_anon_profile") {
    return typeof args.profile_id === "string" ? args.profile_id : null;
  }
  if (name === "set_default_anon_profile") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "create_institution") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "update_institution" || name === "delete_institution" ||
      name === "link_institution_project" || name === "unlink_institution_project") {
    return typeof args.institution_id === "string" ? args.institution_id : null;
  }
  if (name === "create_federation_peer") {
    return typeof args.name === "string" ? args.name : null;
  }
  if (name === "update_federation_peer" || name === "delete_federation_peer") {
    return typeof args.peer_id === "string" ? args.peer_id : null;
  }
  if (name === "set_storage_quota" || name === "update_phi_config") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "delete_study" || name === "soft_delete_study" || name === "restore_study") {
    return typeof args.study_id === "string" ? args.study_id : null;
  }
  if (name === "bulk_pipeline_trigger") {
    return Array.isArray(args.study_ids) ? (args.study_ids as string[]).join(",") : null;
  }
  if (name === "import_tcia_series") {
    return typeof args.series_uid === "string" ? args.series_uid : null;
  }
  if (name === "import_protocol_templates") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "batch_import_studies") {
    return typeof args.dir === "string" ? args.dir : null;
  }
  if (name === "set_user_preferences") {
    return typeof args.user_id === "string" ? args.user_id : null;
  }
  if (name === "import_routing_rules") {
    return typeof args.project_id === "string" ? args.project_id : null;
  }
  if (name === "reorder_routing_rules") {
    return Array.isArray(args.rules) ? `${(args.rules as unknown[]).length} rules` : null;
  }
  if (name === "bulk_toggle_routing_rules") {
    return Array.isArray(args.rule_ids) ? `${(args.rule_ids as string[]).length} rules` : null;
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
      llmMaxTokens: config.agentLlmMaxTokens,
      llmUseGcpAuth: config.agentLlmUseGcpAuth,
      llmGcpProject: config.agentLlmGcpProject,
      llmUseAwsBedrock: config.agentLlmUseAwsBedrock,
      llmAwsRegion: config.agentLlmAwsRegion
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

async function handleTriggerAnalytics(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_analytics");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_analytics",
      false,
      "trigger_analytics"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_analytics");
  }

  if (matched.analytics_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require analytics", false, "trigger_analytics");
  }

  if (matched.analytics_status === "analyzing") {
    return formatError(requestId, "CONFLICT", "Analytics already in progress", false, "trigger_analytics");
  }

  if (matched.analytics_status && !["pending", "failed"].includes(matched.analytics_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `Analytics trigger blocked for current status: ${matched.analytics_status}`,
      false,
      "trigger_analytics"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/analytics`);
  return formatSuccess(requestId, "trigger_analytics", {
    accepted: true,
    study_uid: parsed.study_uid,
    reason: parsed.reason,
    result: data
  });
}

async function handleTriggerLongitudinalAnalytics(
  requestId: string,
  parsed: {
    study_uid: string;
    baseline_study_id: string;
    scan_interval_days?: number;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_longitudinal_analytics");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_longitudinal_analytics",
      false,
      "trigger_longitudinal_analytics"
    );
  }

  const body: Record<string, unknown> = {
    baseline_study_id: parsed.baseline_study_id
  };
  if (parsed.scan_interval_days !== undefined && parsed.scan_interval_days > 0) {
    body.scan_interval_days = parsed.scan_interval_days;
  }

  const data = await client.post(
    `/api/studies/${encodeURIComponent(parsed.study_uid)}/longitudinal-analytics`,
    body
  );
  return formatSuccess(requestId, "trigger_longitudinal_analytics", {
    accepted: true,
    study_uid: parsed.study_uid,
    baseline_study_id: parsed.baseline_study_id,
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

async function handleTriggerPixelRedaction(
  requestId: string,
  parsed: {
    study_uid: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "trigger_pixel_redaction");
  }

  if (!config.enableWriteTools) {
    return formatError(
      requestId,
      "FORBIDDEN",
      "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow trigger_pixel_redaction",
      false,
      "trigger_pixel_redaction"
    );
  }

  const studyResult = await client.get(`/api/studies?limit=200&offset=0&search=${encodeURIComponent(parsed.study_uid)}`);
  const studies = extractStudies(studyResult);
  const matched = studies.find((study) => study.study_instance_uid === parsed.study_uid);

  if (!matched) {
    return formatError(requestId, "NOT_FOUND", `Study UID not found: ${parsed.study_uid}`, false, "trigger_pixel_redaction");
  }

  if (matched.pixel_redaction_required === false) {
    return formatError(requestId, "CONFLICT", "Study does not require pixel redaction", false, "trigger_pixel_redaction");
  }

  if (matched.pixel_redaction_status === "redacting") {
    return formatError(requestId, "CONFLICT", "Pixel redaction already in progress", false, "trigger_pixel_redaction");
  }

  if (matched.pixel_redaction_status && !["pending", "failed"].includes(matched.pixel_redaction_status)) {
    return formatError(
      requestId,
      "CONFLICT",
      `Pixel redaction trigger blocked for current status: ${matched.pixel_redaction_status}`,
      false,
      "trigger_pixel_redaction"
    );
  }

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_uid)}/pixel-redaction`);
  return formatSuccess(requestId, "trigger_pixel_redaction", {
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
  parsed: { share_id: string; revocation_reason?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "revoke_share");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow revoke_share", false, "revoke_share");
  }

  const body = parsed.revocation_reason ? { reason: parsed.revocation_reason } : {};
  const data = await client.delete(`/api/shares/${encodeURIComponent(parsed.share_id)}`, body);
  return formatSuccess(requestId, "revoke_share", {
    accepted: true,
    share_id: parsed.share_id,
    revocation_reason: parsed.revocation_reason ?? null,
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

async function handleLinkStudies(
  requestId: string,
  parsed: { study_id: string; related_study_id: string; relationship: string; notes?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "link_studies");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow link_studies", false, "link_studies");
  }

  const body: Record<string, string> = { related_study_id: parsed.related_study_id, relationship: parsed.relationship };
  if (parsed.notes) body.notes = parsed.notes;
  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/relationships`, body);
  return formatSuccess(requestId, "link_studies", {
    accepted: true,
    study_id: parsed.study_id,
    related_study_id: parsed.related_study_id,
    relationship: parsed.relationship,
    reason: parsed.reason,
    result: data
  });
}

async function handleUnlinkStudies(
  requestId: string,
  parsed: { study_id: string; relationship_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "unlink_studies");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow unlink_studies", false, "unlink_studies");
  }

  await client.delete(`/api/studies/${encodeURIComponent(parsed.study_id)}/relationships/${encodeURIComponent(parsed.relationship_id)}`);
  return formatSuccess(requestId, "unlink_studies", {
    accepted: true,
    study_id: parsed.study_id,
    relationship_id: parsed.relationship_id,
    reason: parsed.reason
  });
}

async function handleCreateDigestSubscription(
  requestId: string,
  parsed: { project_id: string; email: string; frequency: "weekly" | "monthly"; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_digest_subscription");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_digest_subscription", false, "create_digest_subscription");
  }

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/digest-subscriptions`, {
    email: parsed.email,
    frequency: parsed.frequency
  });
  return formatSuccess(requestId, "create_digest_subscription", {
    accepted: true,
    project_id: parsed.project_id,
    email: parsed.email,
    frequency: parsed.frequency,
    subscription: data,
    reason: parsed.reason
  });
}

async function handleDeleteDigestSubscription(
  requestId: string,
  parsed: { subscription_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_digest_subscription");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_digest_subscription", false, "delete_digest_subscription");
  }

  await client.delete(`/api/digest-subscriptions/${encodeURIComponent(parsed.subscription_id)}`);
  return formatSuccess(requestId, "delete_digest_subscription", {
    accepted: true,
    subscription_id: parsed.subscription_id,
    reason: parsed.reason
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

async function handleBulkCreateShares(
  requestId: string,
  parsed: {
    study_ids: string[];
    recipient_email: string;
    note?: string;
    expiry_hours?: number;
    max_downloads?: number;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "bulk_create_shares");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow bulk_create_shares", false, "bulk_create_shares");
  }

  const body: Record<string, unknown> = {
    study_ids: parsed.study_ids,
    recipient_email: parsed.recipient_email
  };
  if (parsed.note !== undefined) body.note = parsed.note;
  if (parsed.expiry_hours !== undefined) body.expiry_hours = parsed.expiry_hours;
  if (parsed.max_downloads !== undefined) body.max_downloads = parsed.max_downloads;

  const data = await client.post("/api/studies/bulk-share", body);
  return formatSuccess(requestId, "bulk_create_shares", {
    accepted: true,
    recipient_email: parsed.recipient_email,
    reason: parsed.reason,
    result: data
  });
}

async function handleToggleStudyFlag(
  requestId: string,
  parsed: { study_id: string; flagged: boolean; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "toggle_study_flag");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow toggle_study_flag", false, "toggle_study_flag");
  }

  const data = await client.patch(`/api/studies/${encodeURIComponent(parsed.study_id)}/flag`, { flagged: parsed.flagged });
  return formatSuccess(requestId, "toggle_study_flag", {
    accepted: true,
    study_id: parsed.study_id,
    flagged: parsed.flagged,
    reason: parsed.reason,
    result: data
  });
}

async function handleCloneProject(
  requestId: string,
  parsed: { project_id: string; name?: string; slug?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "clone_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow clone_project", false, "clone_project");
  }
  const body: Record<string, string> = {};
  if (parsed.name) body.name = parsed.name;
  if (parsed.slug) body.slug = parsed.slug;
  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/clone`, body);
  return formatSuccess(requestId, "clone_project", {
    accepted: true,
    source_project_id: parsed.project_id,
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

async function handleCreateInviteCode(
  requestId: string,
  parsed: { label: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_invite_code");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_invite_code", false, "create_invite_code");
  }

  const data = await client.post("/api/invite-codes", { label: parsed.label });
  return formatSuccess(requestId, "create_invite_code", {
    accepted: true,
    label: parsed.label,
    invite_code: data,
    reason: parsed.reason
  });
}

async function handleRevokeInviteCode(
  requestId: string,
  parsed: { code_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "revoke_invite_code");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow revoke_invite_code", false, "revoke_invite_code");
  }

  const data = await client.post(`/api/invite-codes/${encodeURIComponent(parsed.code_id)}/revoke`);
  return formatSuccess(requestId, "revoke_invite_code", {
    accepted: true,
    code_id: parsed.code_id,
    result: data,
    reason: parsed.reason
  });
}

async function handleDeleteInviteCode(
  requestId: string,
  parsed: { code_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_invite_code");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_invite_code", false, "delete_invite_code");
  }

  await client.delete(`/api/invite-codes/${encodeURIComponent(parsed.code_id)}`);
  return formatSuccess(requestId, "delete_invite_code", {
    accepted: true,
    code_id: parsed.code_id,
    reason: parsed.reason
  });
}

async function handleSendInviteCode(
  requestId: string,
  parsed: { code_id: string; email: string; name?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "send_invite_code");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow send_invite_code", false, "send_invite_code");
  }

  const body: Record<string, unknown> = { email: parsed.email };
  if (parsed.name !== undefined) body.name = parsed.name;

  const data = await client.post(`/api/invite-codes/${encodeURIComponent(parsed.code_id)}/send`, body);
  return formatSuccess(requestId, "send_invite_code", {
    accepted: true,
    code_id: parsed.code_id,
    email: parsed.email,
    result: data,
    reason: parsed.reason
  });
}

async function handleApproveInviteRequest(
  requestId: string,
  parsed: { invite_request_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "approve_invite_request");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow approve_invite_request", false, "approve_invite_request");
  }

  const data = await client.post(`/api/invite/requests/${encodeURIComponent(parsed.invite_request_id)}/approve`);
  return formatSuccess(requestId, "approve_invite_request", {
    accepted: true,
    invite_request_id: parsed.invite_request_id,
    result: data,
    reason: parsed.reason
  });
}

async function handleDenyInviteRequest(
  requestId: string,
  parsed: { invite_request_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "deny_invite_request");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow deny_invite_request", false, "deny_invite_request");
  }

  const data = await client.post(`/api/invite/requests/${encodeURIComponent(parsed.invite_request_id)}/deny`);
  return formatSuccess(requestId, "deny_invite_request", {
    accepted: true,
    invite_request_id: parsed.invite_request_id,
    result: data,
    reason: parsed.reason
  });
}

async function handleCreateProject(
  requestId: string,
  parsed: { name: string; slug?: string; description?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_project", false, "create_project");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.slug !== undefined) body.slug = parsed.slug;
  if (parsed.description !== undefined) body.description = parsed.description;

  const data = await client.post("/api/projects", body);
  return formatSuccess(requestId, "create_project", {
    accepted: true,
    name: parsed.name,
    project: data,
    reason: parsed.reason
  });
}

async function handleUpdateProject(
  requestId: string,
  parsed: { project_id: string; name: string; slug?: string; description?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_project", false, "update_project");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.slug !== undefined) body.slug = parsed.slug;
  if (parsed.description !== undefined) body.description = parsed.description;

  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}`, body);
  return formatSuccess(requestId, "update_project", {
    accepted: true,
    project_id: parsed.project_id,
    project: data,
    reason: parsed.reason
  });
}

async function handleArchiveProject(
  requestId: string,
  parsed: { project_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "archive_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow archive_project", false, "archive_project");
  }

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/archive`);
  return formatSuccess(requestId, "archive_project", {
    accepted: true,
    project_id: parsed.project_id,
    project: data,
    reason: parsed.reason
  });
}

async function handleRestoreProject(
  requestId: string,
  parsed: { project_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "restore_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow restore_project", false, "restore_project");
  }

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/restore`);
  return formatSuccess(requestId, "restore_project", {
    accepted: true,
    project_id: parsed.project_id,
    project: data,
    reason: parsed.reason
  });
}

async function handleSetProjectRetention(
  requestId: string,
  parsed: { project_id: string; retention_days: number | null; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "set_project_retention");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow set_project_retention", false, "set_project_retention");
  }

  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/retention`, {
    retention_days: parsed.retention_days
  });
  return formatSuccess(requestId, "set_project_retention", {
    accepted: true,
    project_id: parsed.project_id,
    retention_days: parsed.retention_days,
    project: data,
    reason: parsed.reason
  });
}

async function handleSetProjectSLAThreshold(
  requestId: string,
  parsed: { project_id: string; stuck_threshold_minutes: number | null; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "set_project_sla_threshold");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow set_project_sla_threshold", false, "set_project_sla_threshold");
  }

  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/sla-threshold`, {
    stuck_threshold_minutes: parsed.stuck_threshold_minutes
  });
  return formatSuccess(requestId, "set_project_sla_threshold", {
    accepted: true,
    project_id: parsed.project_id,
    stuck_threshold_minutes: parsed.stuck_threshold_minutes,
    project: data,
    reason: parsed.reason
  });
}

async function handleAddProjectMember(
  requestId: string,
  parsed: { project_id: string; admin_user_id: string; role: string; institution_id?: string | null; notes?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "add_project_member");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow add_project_member", false, "add_project_member");
  }
  const body: Record<string, unknown> = {
    admin_user_id: parsed.admin_user_id,
    role: parsed.role,
    institution_id: parsed.institution_id ?? null,
  };
  if (parsed.notes !== undefined) body.notes = parsed.notes;
  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/members`, body);
  return formatSuccess(requestId, "add_project_member", { accepted: true, project_id: parsed.project_id, member: data, reason: parsed.reason });
}

async function handleUpdateProjectMember(
  requestId: string,
  parsed: { project_id: string; member_id: string; role: string; institution_id?: string | null; notes?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_project_member");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_project_member", false, "update_project_member");
  }
  const body: Record<string, unknown> = {
    role: parsed.role,
    institution_id: parsed.institution_id ?? null,
  };
  if (parsed.notes !== undefined) body.notes = parsed.notes;
  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/members/${encodeURIComponent(parsed.member_id)}`, body);
  return formatSuccess(requestId, "update_project_member", { accepted: true, member_id: parsed.member_id, member: data, reason: parsed.reason });
}

async function handleRemoveProjectMember(
  requestId: string,
  parsed: { project_id: string; member_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "remove_project_member");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow remove_project_member", false, "remove_project_member");
  }
  await client.delete(`/api/projects/${encodeURIComponent(parsed.project_id)}/members/${encodeURIComponent(parsed.member_id)}`);
  return formatSuccess(requestId, "remove_project_member", { accepted: true, member_id: parsed.member_id, reason: parsed.reason });
}

async function handleToggleProjectRestricted(
  requestId: string,
  parsed: { project_id: string; restricted: boolean; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "toggle_project_restricted");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow toggle_project_restricted", false, "toggle_project_restricted");
  }
  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/restricted`, { restricted: parsed.restricted });
  return formatSuccess(requestId, "toggle_project_restricted", { accepted: true, project_id: parsed.project_id, restricted: parsed.restricted, project: data, reason: parsed.reason });
}

async function handleCreateAdminUser(
  requestId: string,
  parsed: { email: string; name?: string; role: "admin" | "viewer" | "researcher"; notes?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_admin_user");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_admin_user", false, "create_admin_user");
  }

  const body: Record<string, unknown> = { email: parsed.email, role: parsed.role, enabled: true };
  if (parsed.name !== undefined) body.name = parsed.name;
  if (parsed.notes !== undefined) body.notes = parsed.notes;

  const data = await client.post("/api/admin-users", body);
  return formatSuccess(requestId, "create_admin_user", {
    accepted: true,
    email: parsed.email,
    role: parsed.role,
    user: data,
    reason: parsed.reason
  });
}

async function handleUpdateAdminUser(
  requestId: string,
  parsed: { user_id: string; email: string; name?: string; role: "admin" | "viewer" | "researcher"; enabled?: boolean; notes?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_admin_user");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_admin_user", false, "update_admin_user");
  }

  const body: Record<string, unknown> = { email: parsed.email, role: parsed.role };
  if (parsed.name !== undefined) body.name = parsed.name;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;
  if (parsed.notes !== undefined) body.notes = parsed.notes;

  const data = await client.put(`/api/admin-users/${encodeURIComponent(parsed.user_id)}`, body);
  return formatSuccess(requestId, "update_admin_user", {
    accepted: true,
    user_id: parsed.user_id,
    user: data,
    reason: parsed.reason
  });
}

async function handleDeleteAdminUser(
  requestId: string,
  parsed: { user_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_admin_user");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_admin_user", false, "delete_admin_user");
  }

  await client.delete(`/api/admin-users/${encodeURIComponent(parsed.user_id)}`);
  return formatSuccess(requestId, "delete_admin_user", {
    accepted: true,
    user_id: parsed.user_id,
    reason: parsed.reason
  });
}

async function handleSendAdminInvite(
  requestId: string,
  parsed: { user_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "send_admin_invite");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow send_admin_invite", false, "send_admin_invite");
  }

  const data = await client.post(`/api/admin-users/${encodeURIComponent(parsed.user_id)}/send-invite`, {});
  return formatSuccess(requestId, "send_admin_invite", {
    accepted: true,
    user_id: parsed.user_id,
    result: data,
    reason: parsed.reason
  });
}

async function handleCreateWebhookSubscription(
  requestId: string,
  parsed: { url: string; events: string[]; secret: string; project_id?: string; enabled?: boolean; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_webhook_subscription");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_webhook_subscription", false, "create_webhook_subscription");
  }

  const body: Record<string, unknown> = {
    url: parsed.url,
    events: parsed.events,
    secret: parsed.secret
  };
  if (parsed.project_id !== undefined) body.project_id = parsed.project_id;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;

  const data = await client.post("/api/webhook-subscriptions", body);
  return formatSuccess(requestId, "create_webhook_subscription", {
    accepted: true,
    url: parsed.url,
    events: parsed.events,
    subscription: data,
    reason: parsed.reason
  });
}

async function handleUpdateWebhookSubscription(
  requestId: string,
  parsed: { subscription_id: string; url?: string; events?: string[]; secret?: string; enabled?: boolean; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_webhook_subscription");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_webhook_subscription", false, "update_webhook_subscription");
  }

  const body: Record<string, unknown> = {};
  if (parsed.url !== undefined) body.url = parsed.url;
  if (parsed.events !== undefined) body.events = parsed.events;
  if (parsed.secret !== undefined) body.secret = parsed.secret;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;

  const data = await client.put(`/api/webhook-subscriptions/${encodeURIComponent(parsed.subscription_id)}`, body);
  return formatSuccess(requestId, "update_webhook_subscription", {
    accepted: true,
    subscription_id: parsed.subscription_id,
    subscription: data,
    reason: parsed.reason
  });
}

async function handleDeleteWebhookSubscription(
  requestId: string,
  parsed: { subscription_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_webhook_subscription");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_webhook_subscription", false, "delete_webhook_subscription");
  }

  await client.delete(`/api/webhook-subscriptions/${encodeURIComponent(parsed.subscription_id)}`);
  return formatSuccess(requestId, "delete_webhook_subscription", {
    accepted: true,
    subscription_id: parsed.subscription_id,
    reason: parsed.reason
  });
}

async function handleRetryWebhookDelivery(
  requestId: string,
  parsed: { delivery_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "retry_webhook_delivery");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow retry_webhook_delivery", false, "retry_webhook_delivery");
  }

  const data = await client.post(`/api/webhook-deliveries/${encodeURIComponent(parsed.delivery_id)}/retry`);
  return formatSuccess(requestId, "retry_webhook_delivery", {
    accepted: true,
    delivery_id: parsed.delivery_id,
    result: data,
    reason: parsed.reason
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

async function handleCreateApiKey(
  requestId: string,
  parsed: { name: string; expires_at?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_api_key");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_api_key", false, "create_api_key");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.expires_at) body.expires_at = parsed.expires_at;
  const data = await client.post("/api/api-keys", body);
  return formatSuccess(requestId, "create_api_key", {
    accepted: true,
    name: parsed.name,
    reason: parsed.reason,
    result: data
  });
}

async function handleRotateApiKey(
  requestId: string,
  parsed: { key_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "rotate_api_key");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow rotate_api_key", false, "rotate_api_key");
  }

  const data = await client.post(`/api/api-keys/${encodeURIComponent(parsed.key_id)}/rotate`);
  return formatSuccess(requestId, "rotate_api_key", {
    accepted: true,
    key_id: parsed.key_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleSetApiKeyEnabled(
  requestId: string,
  parsed: { key_id: string; reason: string; confirm: true },
  enabled: boolean
) {
  const toolName = enabled ? "enable_api_key" : "disable_api_key";
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, toolName);
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", `Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow ${toolName}`, false, toolName);
  }

  const action = enabled ? "enable" : "disable";
  const data = await client.patch(`/api/api-keys/${encodeURIComponent(parsed.key_id)}/${action}`, null);
  return formatSuccess(requestId, toolName, {
    accepted: true,
    key_id: parsed.key_id,
    enabled,
    reason: parsed.reason,
    result: data
  });
}

async function handleDeleteApiKey(
  requestId: string,
  parsed: { key_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_api_key");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_api_key", false, "delete_api_key");
  }

  const data = await client.delete(`/api/api-keys/${encodeURIComponent(parsed.key_id)}`);
  return formatSuccess(requestId, "delete_api_key", {
    accepted: true,
    key_id: parsed.key_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleBulkStudyAction(
  requestId: string,
  parsed: { study_ids: string[]; reason: string; confirm: true },
  action: "approve" | "reject"
) {
  const toolName = action === "approve" ? "bulk_approve_studies" : "bulk_reject_studies";
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, toolName);
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", `Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow ${toolName}`, false, toolName);
  }

  const data = await client.post("/api/studies/bulk", { action, study_ids: parsed.study_ids });
  return formatSuccess(requestId, toolName, {
    accepted: true,
    action,
    count: parsed.study_ids.length,
    reason: parsed.reason,
    result: data
  });
}

async function handleBulkLabelStudies(
  requestId: string,
  parsed: { study_ids: string[]; label: string; action: "add" | "remove"; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "bulk_label_studies");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow bulk_label_studies", false, "bulk_label_studies");
  }

  const data = await client.post("/api/studies/bulk-label", {
    study_ids: parsed.study_ids,
    label: parsed.label,
    action: parsed.action
  });
  return formatSuccess(requestId, "bulk_label_studies", {
    accepted: true,
    action: parsed.action,
    label: parsed.label,
    count: parsed.study_ids.length,
    reason: parsed.reason,
    result: data
  });
}

async function handleGenerateSyntheticStudy(
  requestId: string,
  parsed: {
    project_slug?: string;
    slices?: number;
    size?: number;
    seed?: number;
    with_face?: boolean;
    use_gpu?: boolean;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "generate_synthetic_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow generate_synthetic_study", false, "generate_synthetic_study");
  }

  const body: Record<string, unknown> = {};
  if (parsed.project_slug) body.project_slug = parsed.project_slug;
  if (parsed.slices !== undefined) body.slices = parsed.slices;
  if (parsed.size !== undefined) body.size = parsed.size;
  if (parsed.seed !== undefined) body.seed = parsed.seed;
  if (parsed.with_face !== undefined) body.with_face = parsed.with_face;
  if (parsed.use_gpu !== undefined) body.use_gpu = parsed.use_gpu;

  const data = await client.post("/api/studies/generate-synthetic", body);
  return formatSuccess(requestId, "generate_synthetic_study", {
    accepted: true,
    project_slug: parsed.project_slug ?? "default",
    reason: parsed.reason,
    result: data
  });
}

async function handleUpdateDestination(
  requestId: string,
  parsed: {
    destination_id: string;
    name?: string;
    description?: string;
    dicomweb_url?: string;
    dicomweb_auth_header?: string;
    enabled?: boolean;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_destination");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_destination", false, "update_destination");
  }

  // Fetch existing destination first to apply partial updates.
  const existing = await client.get(`/api/destinations`) as { destinations?: unknown[] } | unknown[];
  const list: unknown[] = Array.isArray(existing) ? existing : ((existing as Record<string, unknown>)?.destinations ?? []) as unknown[];
  const dest = (list as Array<Record<string, unknown>>).find((d) => d.id === parsed.destination_id);
  if (!dest) {
    return formatError(requestId, "NOT_FOUND", `Destination ${parsed.destination_id} not found`, false, "update_destination");
  }

  const body: Record<string, unknown> = {
    name: parsed.name ?? dest.name,
    slug: dest.slug,
    description: parsed.description !== undefined ? parsed.description : dest.description,
    type: dest.type,
    dicomweb_url: parsed.dicomweb_url ?? dest.dicomweb_url ?? "",
    dicomweb_auth_header: parsed.dicomweb_auth_header !== undefined ? parsed.dicomweb_auth_header : dest.dicomweb_auth_header ?? "",
    ae_title: dest.ae_title ?? "",
    host: dest.host ?? "",
    port: dest.port ?? 0,
    enabled: parsed.enabled !== undefined ? parsed.enabled : dest.enabled
  };

  const data = await client.put(`/api/destinations/${encodeURIComponent(parsed.destination_id)}`, body);
  return formatSuccess(requestId, "update_destination", {
    accepted: true,
    destination_id: parsed.destination_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleCreateDestination(
  requestId: string,
  parsed: {
    name: string;
    description?: string;
    type: "dicomweb" | "dimse";
    dicomweb_url?: string;
    dicomweb_auth_header?: string;
    ae_title?: string;
    host?: string;
    port?: number;
    enabled?: boolean;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_destination");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_destination", false, "create_destination");
  }

  const body: Record<string, unknown> = {
    name: parsed.name,
    type: parsed.type,
    enabled: parsed.enabled ?? true
  };
  if (parsed.description) body.description = parsed.description;
  if (parsed.dicomweb_url) body.dicomweb_url = parsed.dicomweb_url;
  if (parsed.dicomweb_auth_header) body.dicomweb_auth_header = parsed.dicomweb_auth_header;
  if (parsed.ae_title) body.ae_title = parsed.ae_title;
  if (parsed.host) body.host = parsed.host;
  if (parsed.port) body.port = parsed.port;

  const data = await client.post("/api/destinations", body);
  return formatSuccess(requestId, "create_destination", {
    accepted: true,
    name: parsed.name,
    type: parsed.type,
    reason: parsed.reason,
    result: data
  });
}

async function handleDeleteDestination(
  requestId: string,
  parsed: { destination_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_destination");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_destination", false, "delete_destination");
  }

  await client.delete(`/api/destinations/${encodeURIComponent(parsed.destination_id)}`);
  return formatSuccess(requestId, "delete_destination", {
    accepted: true,
    destination_id: parsed.destination_id,
    reason: parsed.reason
  });
}

async function handleCreateRoutingRule(
  requestId: string,
  parsed: {
    name: string;
    description?: string;
    priority: number;
    enabled?: boolean;
    project_id?: string;
    modality?: string;
    body_part?: string;
    source?: "external" | "internal";
    action: string;
    destination_id?: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_routing_rule");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_routing_rule", false, "create_routing_rule");
  }

  const body: Record<string, unknown> = {
    name: parsed.name,
    priority: parsed.priority,
    action: parsed.action,
    enabled: parsed.enabled ?? true
  };
  if (parsed.description) body.description = parsed.description;
  if (parsed.project_id) body.project_id = parsed.project_id;
  if (parsed.modality) body.modality = parsed.modality;
  if (parsed.body_part) body.body_part = parsed.body_part;
  if (parsed.source) body.source = parsed.source;
  if (parsed.destination_id) body.destination_id = parsed.destination_id;

  const data = await client.post("/api/routing-rules", body);
  return formatSuccess(requestId, "create_routing_rule", {
    accepted: true,
    name: parsed.name,
    action: parsed.action,
    reason: parsed.reason,
    result: data
  });
}

async function handleUpdateRoutingRule(
  requestId: string,
  parsed: {
    rule_id: string;
    name?: string;
    description?: string;
    priority?: number;
    enabled?: boolean;
    project_id?: string | null;
    modality?: string | null;
    body_part?: string | null;
    source?: "external" | "internal" | null;
    action?: string;
    destination_id?: string | null;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_routing_rule");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_routing_rule", false, "update_routing_rule");
  }

  // Fetch existing rule to apply partial updates.
  const existing = await client.get("/api/routing-rules") as { rules?: unknown[] } | unknown[];
  const list: unknown[] = Array.isArray(existing) ? existing : ((existing as Record<string, unknown>)?.rules ?? []) as unknown[];
  const rule = (list as Array<Record<string, unknown>>).find((r) => r.id === parsed.rule_id);
  if (!rule) {
    return formatError(requestId, "NOT_FOUND", `Routing rule ${parsed.rule_id} not found`, false, "update_routing_rule");
  }

  const body: Record<string, unknown> = {
    name: parsed.name ?? rule.name,
    description: parsed.description !== undefined ? parsed.description : rule.description,
    priority: parsed.priority ?? rule.priority,
    enabled: parsed.enabled !== undefined ? parsed.enabled : rule.enabled,
    action: parsed.action ?? rule.action
  };
  // Explicitly settable nullable fields
  body.project_id = parsed.project_id !== undefined ? parsed.project_id : rule.project_id;
  body.modality = parsed.modality !== undefined ? parsed.modality : rule.modality;
  body.body_part = parsed.body_part !== undefined ? parsed.body_part : rule.body_part;
  body.source = parsed.source !== undefined ? parsed.source : rule.source;
  body.destination_id = parsed.destination_id !== undefined ? parsed.destination_id : rule.destination_id;

  const data = await client.put(`/api/routing-rules/${encodeURIComponent(parsed.rule_id)}`, body);
  return formatSuccess(requestId, "update_routing_rule", {
    accepted: true,
    rule_id: parsed.rule_id,
    reason: parsed.reason,
    result: data
  });
}

async function handleDeleteRoutingRule(
  requestId: string,
  parsed: { rule_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_routing_rule");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_routing_rule", false, "delete_routing_rule");
  }

  await client.delete(`/api/routing-rules/${encodeURIComponent(parsed.rule_id)}`);
  return formatSuccess(requestId, "delete_routing_rule", {
    accepted: true,
    rule_id: parsed.rule_id,
    reason: parsed.reason
  });
}

async function handleCreateProtocolTemplate(
  requestId: string,
  parsed: {
    project_id: string; name: string; description?: string; manufacturer?: string;
    model?: string; software_version?: string; sequence_type?: string;
    rules?: unknown[]; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_protocol_template");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_protocol_template", false, "create_protocol_template");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.description !== undefined) body.description = parsed.description;
  if (parsed.manufacturer !== undefined) body.manufacturer = parsed.manufacturer;
  if (parsed.model !== undefined) body.model = parsed.model;
  if (parsed.software_version !== undefined) body.software_version = parsed.software_version;
  if (parsed.sequence_type !== undefined) body.sequence_type = parsed.sequence_type;
  if (parsed.rules !== undefined) body.rules = parsed.rules;

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/protocol-templates`, body);
  return formatSuccess(requestId, "create_protocol_template", {
    accepted: true,
    project_id: parsed.project_id,
    name: parsed.name,
    template: data,
    reason: parsed.reason
  });
}

async function handleUpdateProtocolTemplate(
  requestId: string,
  parsed: {
    template_id: string; name: string; description?: string; manufacturer?: string;
    model?: string; software_version?: string; sequence_type?: string;
    rules?: unknown[]; enabled?: boolean; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_protocol_template");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_protocol_template", false, "update_protocol_template");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.description !== undefined) body.description = parsed.description;
  if (parsed.manufacturer !== undefined) body.manufacturer = parsed.manufacturer;
  if (parsed.model !== undefined) body.model = parsed.model;
  if (parsed.software_version !== undefined) body.software_version = parsed.software_version;
  if (parsed.sequence_type !== undefined) body.sequence_type = parsed.sequence_type;
  if (parsed.rules !== undefined) body.rules = parsed.rules;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;

  const data = await client.put(`/api/protocol-templates/${encodeURIComponent(parsed.template_id)}`, body);
  return formatSuccess(requestId, "update_protocol_template", {
    accepted: true,
    template_id: parsed.template_id,
    template: data,
    reason: parsed.reason
  });
}

async function handleDeleteProtocolTemplate(
  requestId: string,
  parsed: { template_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_protocol_template");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_protocol_template", false, "delete_protocol_template");
  }

  await client.delete(`/api/protocol-templates/${encodeURIComponent(parsed.template_id)}`);
  return formatSuccess(requestId, "delete_protocol_template", {
    accepted: true,
    template_id: parsed.template_id,
    reason: parsed.reason
  });
}

async function handleCreateAnonProfile(
  requestId: string,
  parsed: {
    project_id: string; name: string; description?: string;
    retained_tags?: string[]; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_anon_profile");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_anon_profile", false, "create_anon_profile");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.description !== undefined) body.description = parsed.description;
  if (parsed.retained_tags !== undefined) body.retained_tags = parsed.retained_tags;

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/anon-profiles`, body);
  return formatSuccess(requestId, "create_anon_profile", {
    accepted: true,
    project_id: parsed.project_id,
    name: parsed.name,
    profile: data,
    reason: parsed.reason
  });
}

async function handleUpdateAnonProfile(
  requestId: string,
  parsed: {
    profile_id: string; name: string; description?: string;
    retained_tags?: string[]; enabled?: boolean; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_anon_profile");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_anon_profile", false, "update_anon_profile");
  }

  const body: Record<string, unknown> = { name: parsed.name };
  if (parsed.description !== undefined) body.description = parsed.description;
  if (parsed.retained_tags !== undefined) body.retained_tags = parsed.retained_tags;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;

  const data = await client.put(`/api/anon-profiles/${encodeURIComponent(parsed.profile_id)}`, body);
  return formatSuccess(requestId, "update_anon_profile", {
    accepted: true,
    profile_id: parsed.profile_id,
    profile: data,
    reason: parsed.reason
  });
}

async function handleDeleteAnonProfile(
  requestId: string,
  parsed: { profile_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_anon_profile");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_anon_profile", false, "delete_anon_profile");
  }

  await client.delete(`/api/anon-profiles/${encodeURIComponent(parsed.profile_id)}`);
  return formatSuccess(requestId, "delete_anon_profile", {
    accepted: true,
    profile_id: parsed.profile_id,
    reason: parsed.reason
  });
}

async function handleSetDefaultAnonProfile(
  requestId: string,
  parsed: { project_id: string; profile_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "set_default_anon_profile");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow set_default_anon_profile", false, "set_default_anon_profile");
  }

  await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/default-anon-profile`, { profile_id: parsed.profile_id });
  return formatSuccess(requestId, "set_default_anon_profile", {
    accepted: true,
    project_id: parsed.project_id,
    profile_id: parsed.profile_id || null,
    action: parsed.profile_id ? "set" : "cleared",
    reason: parsed.reason
  });
}

async function handleCreateInstitution(
  requestId: string,
  parsed: {
    name: string; type: string; slug?: string; description?: string;
    contact_name?: string; contact_email?: string; ip_ranges?: string;
    ae_title?: string; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_institution");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_institution", false, "create_institution");
  }

  const body: Record<string, unknown> = { name: parsed.name, institution_type: parsed.type };
  if (parsed.slug !== undefined) body.slug = parsed.slug;
  if (parsed.description !== undefined) body.description = parsed.description;
  if (parsed.contact_name !== undefined) body.contact_name = parsed.contact_name;
  if (parsed.contact_email !== undefined) body.contact_email = parsed.contact_email;
  if (parsed.ip_ranges !== undefined) body.ip_ranges = parsed.ip_ranges;
  if (parsed.ae_title !== undefined) body.ae_title = parsed.ae_title;

  const data = await client.post("/api/institutions", body);
  return formatSuccess(requestId, "create_institution", {
    accepted: true,
    name: parsed.name,
    institution: data,
    reason: parsed.reason
  });
}

async function handleUpdateInstitution(
  requestId: string,
  parsed: {
    institution_id: string; name: string; type: string; slug?: string; description?: string;
    contact_name?: string; contact_email?: string; ip_ranges?: string;
    ae_title?: string; enabled?: boolean; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_institution");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_institution", false, "update_institution");
  }

  const body: Record<string, unknown> = { name: parsed.name, institution_type: parsed.type };
  if (parsed.slug !== undefined) body.slug = parsed.slug;
  if (parsed.description !== undefined) body.description = parsed.description;
  if (parsed.contact_name !== undefined) body.contact_name = parsed.contact_name;
  if (parsed.contact_email !== undefined) body.contact_email = parsed.contact_email;
  if (parsed.ip_ranges !== undefined) body.ip_ranges = parsed.ip_ranges;
  if (parsed.ae_title !== undefined) body.ae_title = parsed.ae_title;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;

  const data = await client.put(`/api/institutions/${encodeURIComponent(parsed.institution_id)}`, body);
  return formatSuccess(requestId, "update_institution", {
    accepted: true,
    institution_id: parsed.institution_id,
    institution: data,
    reason: parsed.reason
  });
}

async function handleDeleteInstitution(
  requestId: string,
  parsed: { institution_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_institution");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_institution", false, "delete_institution");
  }

  await client.delete(`/api/institutions/${encodeURIComponent(parsed.institution_id)}`);
  return formatSuccess(requestId, "delete_institution", {
    accepted: true,
    institution_id: parsed.institution_id,
    reason: parsed.reason
  });
}

async function handleLinkInstitutionProject(
  requestId: string,
  parsed: { institution_id: string; project_id: string; role: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "link_institution_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow link_institution_project", false, "link_institution_project");
  }

  const data = await client.post(`/api/institutions/${encodeURIComponent(parsed.institution_id)}/projects`, {
    project_id: parsed.project_id,
    role: parsed.role
  });
  return formatSuccess(requestId, "link_institution_project", {
    accepted: true,
    institution_id: parsed.institution_id,
    project_id: parsed.project_id,
    role: parsed.role,
    result: data,
    reason: parsed.reason
  });
}

async function handleUnlinkInstitutionProject(
  requestId: string,
  parsed: { institution_id: string; project_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "unlink_institution_project");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow unlink_institution_project", false, "unlink_institution_project");
  }

  await client.delete(`/api/institutions/${encodeURIComponent(parsed.institution_id)}/projects/${encodeURIComponent(parsed.project_id)}`);
  return formatSuccess(requestId, "unlink_institution_project", {
    accepted: true,
    institution_id: parsed.institution_id,
    project_id: parsed.project_id,
    reason: parsed.reason
  });
}

async function handleCreateFederationPeer(
  requestId: string,
  parsed: { name: string; slug?: string; api_url: string; notes?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "create_federation_peer");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow create_federation_peer", false, "create_federation_peer");
  }

  const body: Record<string, unknown> = { name: parsed.name, api_url: parsed.api_url };
  if (parsed.slug !== undefined) body.slug = parsed.slug;
  if (parsed.notes !== undefined) body.notes = parsed.notes;

  const data = await client.post("/api/federation-peers", body);
  return formatSuccess(requestId, "create_federation_peer", {
    accepted: true,
    name: parsed.name,
    peer: data,
    reason: parsed.reason
  });
}

async function handleUpdateFederationPeer(
  requestId: string,
  parsed: {
    peer_id: string; name: string; slug?: string; api_url: string;
    notes?: string; enabled?: boolean; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_federation_peer");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_federation_peer", false, "update_federation_peer");
  }

  const body: Record<string, unknown> = { name: parsed.name, api_url: parsed.api_url };
  if (parsed.slug !== undefined) body.slug = parsed.slug;
  if (parsed.notes !== undefined) body.notes = parsed.notes;
  if (parsed.enabled !== undefined) body.enabled = parsed.enabled;

  const data = await client.put(`/api/federation-peers/${encodeURIComponent(parsed.peer_id)}`, body);
  return formatSuccess(requestId, "update_federation_peer", {
    accepted: true,
    peer_id: parsed.peer_id,
    peer: data,
    reason: parsed.reason
  });
}

async function handleDeleteFederationPeer(
  requestId: string,
  parsed: { peer_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_federation_peer");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_federation_peer", false, "delete_federation_peer");
  }

  await client.delete(`/api/federation-peers/${encodeURIComponent(parsed.peer_id)}`);
  return formatSuccess(requestId, "delete_federation_peer", {
    accepted: true,
    peer_id: parsed.peer_id,
    reason: parsed.reason
  });
}

async function handleSetStorageQuota(
  requestId: string,
  parsed: { project_id: string; storage_quota_bytes: number | null; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "set_storage_quota");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow set_storage_quota", false, "set_storage_quota");
  }

  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/storage-quota`, {
    storage_quota_bytes: parsed.storage_quota_bytes
  });
  return formatSuccess(requestId, "set_storage_quota", {
    accepted: true,
    project_id: parsed.project_id,
    storage_quota_bytes: parsed.storage_quota_bytes,
    action: parsed.storage_quota_bytes !== null ? "set" : "cleared",
    result: data,
    reason: parsed.reason
  });
}

async function handleUpdatePhiConfig(
  requestId: string,
  parsed: {
    project_id: string; confidence_threshold?: number;
    min_text_length?: number; reason: string; confirm: true
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "update_phi_config");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow update_phi_config", false, "update_phi_config");
  }

  const body: Record<string, unknown> = {};
  if (parsed.confidence_threshold !== undefined) body.confidence_threshold = parsed.confidence_threshold;
  if (parsed.min_text_length !== undefined) body.min_text_length = parsed.min_text_length;

  const data = await client.put(`/api/projects/${encodeURIComponent(parsed.project_id)}/phi-config`, body);
  return formatSuccess(requestId, "update_phi_config", {
    accepted: true,
    project_id: parsed.project_id,
    config: data,
    reason: parsed.reason
  });
}

async function handleDeleteStudy(
  requestId: string,
  parsed: { study_id: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "delete_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow delete_study", false, "delete_study");
  }

  await client.delete(`/api/studies/${encodeURIComponent(parsed.study_id)}`);
  return formatSuccess(requestId, "delete_study", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason
  });
}

async function handleBulkPipelineTrigger(
  requestId: string,
  parsed: { study_ids: string[]; step: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "bulk_pipeline_trigger");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow bulk_pipeline_trigger", false, "bulk_pipeline_trigger");
  }

  const data = await client.post("/api/studies/bulk-pipeline-trigger", {
    study_ids: parsed.study_ids,
    step: parsed.step
  });
  return formatSuccess(requestId, "bulk_pipeline_trigger", {
    accepted: true,
    study_count: parsed.study_ids.length,
    step: parsed.step,
    result: data,
    reason: parsed.reason
  });
}

async function handleImportTCIASeries(
  requestId: string,
  parsed: { series_uid: string; collection?: string; project_slug?: string; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "import_tcia_series");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow import_tcia_series", false, "import_tcia_series");
  }

  const body: Record<string, unknown> = { series_uid: parsed.series_uid };
  if (parsed.collection !== undefined) body.collection = parsed.collection;
  if (parsed.project_slug !== undefined) body.project_slug = parsed.project_slug;

  const data = await client.post("/api/tcia/import", body);
  return formatSuccess(requestId, "import_tcia_series", {
    accepted: true,
    series_uid: parsed.series_uid,
    result: data,
    reason: parsed.reason
  });
}

async function handleImportProtocolTemplates(
  requestId: string,
  parsed: { project_id: string; templates: unknown[]; reason: string; confirm: true }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "import_protocol_templates");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow import_protocol_templates", false, "import_protocol_templates");
  }

  const data = await client.post(`/api/projects/${encodeURIComponent(parsed.project_id)}/protocol-templates/import`, parsed.templates);
  return formatSuccess(requestId, "import_protocol_templates", {
    accepted: true,
    project_id: parsed.project_id,
    template_count: parsed.templates.length,
    result: data,
    reason: parsed.reason
  });
}

async function handleBatchImportStudies(
  requestId: string,
  parsed: {
    dir: string;
    project_slug?: string;
    institution_id?: string;
    institution_slug?: string;
    source?: string;
    dry_run?: boolean;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "batch_import_studies");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow batch_import_studies", false, "batch_import_studies");
  }

  const body: Record<string, unknown> = {
    dir: parsed.dir,
    project_slug: parsed.project_slug ?? "default",
    source: parsed.source ?? "internal",
    dry_run: parsed.dry_run ?? false
  };
  if (parsed.institution_id) body.institution_id = parsed.institution_id;
  if (parsed.institution_slug) body.institution_slug = parsed.institution_slug;

  const data = await client.post("/api/import/batch", body);
  return formatSuccess(requestId, "batch_import_studies", {
    accepted: true,
    dir: parsed.dir,
    dry_run: parsed.dry_run ?? false,
    result: data,
    reason: parsed.reason
  });
}

async function handleSetUserPreferences(
  requestId: string,
  parsed: {
    user_id: string;
    digest_frequency: string;
    notify_events?: string[];
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "set_user_preferences");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow set_user_preferences", false, "set_user_preferences");
  }

  const data = await client.put(`/api/admin-users/${encodeURIComponent(parsed.user_id)}/preferences`, {
    digest_frequency: parsed.digest_frequency,
    notify_events: parsed.notify_events ?? []
  });
  return formatSuccess(requestId, "set_user_preferences", {
    accepted: true,
    user_id: parsed.user_id,
    preferences: data,
    reason: parsed.reason
  });
}

async function handleImportRoutingRules(
  requestId: string,
  parsed: {
    project_id: string;
    rules: Array<{
      name: string;
      description?: string;
      priority?: number;
      enabled?: boolean;
      modality?: string;
      body_part?: string;
      source?: string | null;
      action: string;
    }>;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "import_routing_rules");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow import_routing_rules", false, "import_routing_rules");
  }

  const data = await client.post(
    `/api/projects/${encodeURIComponent(parsed.project_id)}/routing-rules/import`,
    { rules: parsed.rules }
  );
  return formatSuccess(requestId, "import_routing_rules", {
    accepted: true,
    project_id: parsed.project_id,
    rule_count: parsed.rules.length,
    result: data,
    reason: parsed.reason
  });
}

async function handleReorderRoutingRules(
  requestId: string,
  parsed: {
    rules: Array<{ id: string; priority: number }>;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "reorder_routing_rules");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow reorder_routing_rules", false, "reorder_routing_rules");
  }

  const data = await client.post("/api/routing-rules/reorder", { rules: parsed.rules });
  return formatSuccess(requestId, "reorder_routing_rules", {
    accepted: true,
    updated: parsed.rules.length,
    result: data,
    reason: parsed.reason
  });
}

async function handleBulkToggleRoutingRules(
  requestId: string,
  parsed: {
    rule_ids: string[];
    enabled: boolean;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "bulk_toggle_routing_rules");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow bulk_toggle_routing_rules", false, "bulk_toggle_routing_rules");
  }

  const data = await client.post("/api/routing-rules/bulk-toggle", { rule_ids: parsed.rule_ids, enabled: parsed.enabled });
  return formatSuccess(requestId, "bulk_toggle_routing_rules", {
    accepted: true,
    enabled: parsed.enabled,
    result: data,
    reason: parsed.reason
  });
}

async function handleRetrievePacsStudy(
  requestId: string,
  parsed: {
    ae_title: string;
    host: string;
    port: number;
    study_instance_uid: string;
    move_destination?: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "retrieve_pacs_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow retrieve_pacs_study", false, "retrieve_pacs_study");
  }

  const body: Record<string, unknown> = {
    ae_title: parsed.ae_title,
    host: parsed.host,
    port: parsed.port,
    study_instance_uid: parsed.study_instance_uid
  };
  if (parsed.move_destination) body.move_destination = parsed.move_destination;

  const data = await client.post("/api/dimse/retrieve", body);
  return formatSuccess(requestId, "retrieve_pacs_study", {
    accepted: true,
    study_instance_uid: parsed.study_instance_uid,
    move_destination: parsed.move_destination ?? null,
    result: data,
    reason: parsed.reason
  });
}

async function handleSoftDeleteStudy(
  requestId: string,
  parsed: {
    study_id: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "soft_delete_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow soft_delete_study", false, "soft_delete_study");
  }

  await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/soft-delete`, {});
  return formatSuccess(requestId, "soft_delete_study", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason
  });
}

async function handleRestoreStudy(
  requestId: string,
  parsed: {
    study_id: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "restore_study");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow restore_study", false, "restore_study");
  }

  await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/restore`, {});
  return formatSuccess(requestId, "restore_study", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason
  });
}

function buildRequestId() {
  return `req_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
}

main().catch((error) => {
  console.error("Fatal MCP server error", error);
  process.exit(1);
});
