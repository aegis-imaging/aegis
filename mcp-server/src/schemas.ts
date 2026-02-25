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

export const processingTimesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional(),
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
  revocation_reason: z.string().max(500).optional(), // stored in DB and included in audit entry
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

export const toggleStudyFlagArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  flagged: z.boolean(),
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

export const reactivateStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const getInstitutionStatsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  institution_id: z.string().uuid()
});

export const listProtocolTemplatesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid()
});

export const testWebhookArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const createApiKeyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  name: z.string().min(1).max(128),
  expires_at: z.string().datetime().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const apiKeyIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  key_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const bulkStudyActionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_ids: z.array(z.string().uuid()).min(1).max(200),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const bulkLabelStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_ids: z.array(z.string().uuid()).min(1).max(200),
  label: z.string().min(1).max(80),
  action: z.enum(["add", "remove"]),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const generateSyntheticStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_slug: z.string().min(1).max(64).optional(),
  slices: z.number().int().min(1).max(100).optional(),
  size: z.number().int().min(64).max(512).optional(),
  seed: z.number().int().min(0).optional(),
  with_face: z.boolean().optional(),
  use_gpu: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const testDestinationArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  destination_id: z.string().uuid()
});

export const cloneProjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  name: z.string().min(1).max(128).optional(),
  slug: z.string().min(1).max(128).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateDestinationArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  destination_id: z.string().uuid(),
  name: z.string().min(1).max(256).optional(),
  description: z.string().max(512).optional(),
  dicomweb_url: z.string().url().optional(),
  dicomweb_auth_header: z.string().max(1024).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const createDestinationArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  name: z.string().min(1).max(256),
  description: z.string().max(512).optional(),
  type: z.enum(["dicomweb", "dimse"]),
  dicomweb_url: z.string().url().optional(),
  dicomweb_auth_header: z.string().max(1024).optional(),
  ae_title: z.string().max(64).optional(),
  host: z.string().max(256).optional(),
  port: z.number().int().min(1).max(65535).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const destinationIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  destination_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const createRoutingRuleArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  name: z.string().min(1).max(256),
  description: z.string().max(512).optional(),
  priority: z.number().int().min(1).max(9999),
  enabled: z.boolean().optional(),
  project_id: z.string().uuid().optional(),
  modality: z.string().max(16).optional(),
  body_part: z.string().max(64).optional(),
  source: z.enum(["external", "internal"]).optional(),
  action: z.enum(["route_to", "require_defacing", "require_phi_scan", "require_qc_check",
    "require_bids_conversion", "require_classification", "require_protocol_check",
    "require_export", "auto_approve", "require_qa", "reject"]),
  destination_id: z.string().uuid().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateRoutingRuleArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  rule_id: z.string().uuid(),
  name: z.string().min(1).max(256).optional(),
  description: z.string().max(512).optional(),
  priority: z.number().int().min(1).max(9999).optional(),
  enabled: z.boolean().optional(),
  project_id: z.string().uuid().optional().nullable(),
  modality: z.string().max(16).optional().nullable(),
  body_part: z.string().max(64).optional().nullable(),
  source: z.enum(["external", "internal"]).optional().nullable(),
  action: z.enum(["route_to", "require_defacing", "require_phi_scan", "require_qc_check",
    "require_bids_conversion", "require_classification", "require_protocol_check",
    "require_export", "auto_approve", "require_qa", "reject"]).optional(),
  destination_id: z.string().uuid().optional().nullable(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const routingRuleIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  rule_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
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
  "get_processing_stats",
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
  "get_study_dicom_tags",
  "get_institution_stats",
  "list_protocol_templates",
  "list_federation_peers",
  "get_phi_config",
  "list_anon_profiles",
  "list_api_keys",
  "test_destination"
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
  "export_project_batch",
  "reactivate_study",
  "test_webhook",
  "bulk_approve_studies",
  "bulk_reject_studies",
  "bulk_label_studies",
  "generate_synthetic_study",
  "create_api_key",
  "rotate_api_key",
  "enable_api_key",
  "disable_api_key",
  "delete_api_key",
  "toggle_study_flag",
  "clone_project",
  "update_destination",
  "create_destination",
  "delete_destination",
  "create_routing_rule",
  "update_routing_rule",
  "delete_routing_rule"
] as const;

export type ReadToolName = (typeof readToolNames)[number];
export type WriteToolName = (typeof writeToolNames)[number];
export type ToolName = ReadToolName | WriteToolName;
