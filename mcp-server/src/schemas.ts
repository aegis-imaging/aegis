import { z } from "zod";

export const listStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  limit: z.number().int().min(1).max(200).optional(),
  offset: z.number().int().min(0).optional(),
  project_id: z.string().uuid().optional(),
  status: z.enum(["received", "defacing", "clean", "defaced", "approved", "rejected"]).optional(),
  modality: z.string().min(1).max(16).regex(/^[A-Za-z0-9_]+$/).optional(),
  body_part: z.string().min(1).max(64).regex(/^[A-Za-z0-9_]+$/).optional(),
  source: z.enum(["external", "internal"]).optional(),
  search: z.string().min(1).max(256).optional(),
  date_from: z.string().datetime().optional(),
  date_to: z.string().datetime().optional()
});

export const studyIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid()
});

export const studyUidArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_instance_uid: z.string().min(4).max(256).regex(/^[0-9.]+$/)
});

export const emptyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional()
});

export const writeArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_uid: z.string().min(4).max(256).regex(/^[0-9.]+$/),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const retryDimseArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_instance_uid: z.string().min(4).max(256).regex(/^[0-9.]+$/),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const dimseRetryStatusArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_instance_uid: z.string().min(4).max(256).regex(/^[0-9.]+$/).optional()
});

export const listAuditArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  limit: z.number().int().min(1).max(500).optional(),
  offset: z.number().int().min(0).optional(),
  action: z.string().min(1).max(128).regex(/^[a-z0-9_.]+$/).optional(),
  resource_type: z.string().min(1).max(64).regex(/^[a-z0-9_]+$/).optional(),
  actor: z.string().min(1).max(256).optional()
});

export const listAllSharesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  limit: z.number().int().min(1).max(200).optional(),
  offset: z.number().int().min(0).optional(),
  status: z.enum(["active", "expired", "revoked"]).optional()
});

export const approveStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const rejectStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const revokeShareArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  share_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const createShareArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  recipient_email: z.string().email().max(256),
  note: z.string().max(512).optional(),
  expiry_hours: z.number().int().min(1).max(8760).optional(), // max 1 year
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const reEvaluateRoutingArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const getShareDownloadsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  share_id: z.string().uuid()
});

export const readToolNames = [
  "list_studies",
  "get_study_detail",
  "get_study_diagnostics",
  "get_study_audit",
  "get_study_routing_log",
  "list_export_shares",
  "get_system_health",
  "get_dimse_retry_status",
  "get_audit_log",
  "get_study_by_uid",
  "list_all_shares",
  "get_share_downloads",
  "get_pipeline_stats"
] as const;

export const writeToolNames = [
  "trigger_classification",
  "trigger_phi_scan",
  "trigger_protocol_check",
  "trigger_qc_check",
  "trigger_bids_convert",
  "trigger_export",
  "trigger_deface",
  "retry_dimse_study",
  "approve_study",
  "reject_study",
  "revoke_share",
  "create_share",
  "re_evaluate_routing"
] as const;

export type ReadToolName = (typeof readToolNames)[number];
export type WriteToolName = (typeof writeToolNames)[number];
export type ToolName = ReadToolName | WriteToolName;
