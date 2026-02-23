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
  label: z.string().min(1).max(80).optional(),
  subject_id: z.string().min(1).max(256).optional(),
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
  actor: z.string().min(1).max(256).optional(),
  search: z.string().min(1).max(256).optional(),
  date_from: z.string().datetime().optional(),
  date_to: z.string().datetime().optional()
});

export const getStuckStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  minutes: z.number().int().min(1).max(10080).optional(),
  project_id: z.string().uuid().optional()
});

export const projectScopedArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().optional()
});

export const getAuditActorsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  limit: z.number().int().min(1).max(100).optional()
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
  rejection_reason: z.string().max(500).optional(), // shown to uploader in notification email
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
  max_downloads: z.number().int().min(1).max(1000).optional(), // nil = unlimited
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

export const getWebhookDeliveriesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid()
});

export const getIngestionTimelineArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional(),
  project_id: z.string().uuid().optional()
});

export const resetPipelineStepArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  step: z.enum(["deface", "phi_scan", "qc", "bids", "classify", "protocol", "export"]),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const reassignStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  project_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const addStudyLabelArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  label: z.string().min(1).max(80),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const removeStudyLabelArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  label_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const setStudySubjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  subject_id: z.string().max(256), // empty string clears the subject
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const addStudyNoteArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  note: z.string().min(1).max(2000),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const extendShareArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  share_id: z.string().uuid(),
  extend_hours: z.number().int().min(1).max(8760),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const exportProjectBatchArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const getStudyDicomTagsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_instance_uid: z.string().min(4).max(256).regex(/^[0-9.]+$/)
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
  "get_pipeline_stats",
  "get_stuck_studies",
  "get_breakdown_stats",
  "get_storage_stats",
  "get_audit_actors",
  "list_projects",
  "list_institutions",
  "list_routing_rules",
  "list_destinations",
  "get_study_series",
  "get_study_labels",
  "list_subjects",
  "get_export_analytics",
  "list_webhook_subscriptions",
  "get_webhook_deliveries",
  "get_ingestion_timeline",
  "get_study_dicom_tags"
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
  "re_evaluate_routing",
  "reset_pipeline_step",
  "reassign_study",
  "add_study_label",
  "remove_study_label",
  "set_study_subject",
  "add_study_note",
  "extend_share",
  "export_project_batch"
] as const;

export type ReadToolName = (typeof readToolNames)[number];
export type WriteToolName = (typeof writeToolNames)[number];
export type ToolName = ReadToolName | WriteToolName;
