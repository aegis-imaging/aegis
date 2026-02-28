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
  date_to: z.string().datetime().optional(),
  flagged: z.boolean().optional(),
  sort_by: z.enum(["created_at", "updated_at", "status", "modality", "body_part", "source", "instance_count"]).optional(),
  sort_dir: z.enum(["asc", "desc"]).optional()
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

export const getExpiringStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional().describe("Look-ahead window in days (default 7). Returns studies expiring within this many days."),
  project_id: z.string().uuid().optional().describe("Scope to a specific project UUID; omit for all projects."),
  limit: z.number().int().min(1).max(500).optional().describe("Max results to return (default 200).")
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

export const routingStatsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional()
});

export const routingRuleStatsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional()
});

export const destinationStatsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  destination_id: z.string().uuid(),
  days: z.number().int().min(1).max(365).optional()
});

export const pipelineFunnelArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional(),
  project_id: z.string().uuid().optional()
});

export const projectHealthArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  days: z.number().int().min(1).max(365).optional(),
  project_id: z.string().uuid().optional(),
  stuck_minutes: z.number().int().min(1).optional()
});

export const complianceReportArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  days: z.number().int().min(1).max(365).optional()
});

export const storageUsageArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid()
});

export const anonDiffArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_uid: z.string().min(1)
});

export const listStudyRelationshipsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid()
});

export const linkStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  related_study_id: z.string().uuid(),
  relationship: z.enum(["baseline", "follow_up", "comparison", "replicate"]),
  notes: z.string().max(500).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const unlinkStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  relationship_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const getDailySummaryArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  hours: z.number().int().min(1).max(168).optional()
});

const webhookEvents = z.array(z.enum([
  "study.created", "study.processing_complete",
  "study.approved", "study.rejected", "study.phi_flagged",
  "study.export_complete", "study.stuck"
])).min(1);

export const createWebhookSubscriptionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  url: z.string().url(),
  events: webhookEvents,
  secret: z.string().min(8).max(256),
  project_id: z.string().uuid().optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateWebhookSubscriptionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid(),
  url: z.string().url().optional(),
  events: webhookEvents.optional(),
  secret: z.string().min(8).max(256).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const deleteWebhookSubscriptionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const retryWebhookDeliveryArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  delivery_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const listDigestSubscriptionsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().optional()
});

export const createDigestSubscriptionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  email: z.string().email(),
  frequency: z.enum(["weekly", "monthly"]),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const deleteDigestSubscriptionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
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

export const exportSharesCsvArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().optional(),
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

export const reEvaluateProjectRoutingArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  status: z.string().optional(),
  limit: z.number().int().min(1).max(2000).optional(),
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

export const getDestinationHealthArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  destination_id: z.string().uuid().optional().describe("Specific destination UUID. Omit to get health summary for ALL destinations."),
  limit: z.number().int().min(1).max(100).optional().describe("Max recent test entries to return per destination (default 20, only for single-destination queries).")
});

export const simulateRoutingArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().optional().describe("Scope simulation to a specific project (only project-scoped rules and any-project rules will match)."),
  modality: z.string().max(16).optional().describe("e.g. MRI, CT, PET"),
  body_part: z.string().max(64).optional().describe("e.g. HEAD, CHEST, ABDOMEN"),
  source: z.enum(["external", "internal"]).optional().describe("Study ingest source (default: external)")
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

// ─── Project management ───────────────────────────────────────────────────────
export const createProjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  name: z.string().min(1).max(128),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with dashes").optional(),
  description: z.string().max(1024).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateProjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  name: z.string().min(1).max(128),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with dashes").optional(),
  description: z.string().max(1024).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const archiveRestoreProjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const setProjectRetentionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  retention_days: z.number().int().positive().nullable(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const setProjectSLAThresholdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  stuck_threshold_minutes: z.number().int().positive().nullable(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Admin user management ────────────────────────────────────────────────────
export const listAdminUsersArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional()
});

export const createAdminUserArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  email: z.string().email(),
  name: z.string().min(1).max(255).optional(),
  role: z.enum(["admin", "viewer", "researcher"]),
  notes: z.string().max(1024).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateAdminUserArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  user_id: z.string().uuid(),
  email: z.string().email(),
  name: z.string().min(1).max(255).optional(),
  role: z.enum(["admin", "viewer", "researcher"]),
  enabled: z.boolean().optional(),
  notes: z.string().max(1024).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const deleteAdminUserArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  user_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const sendAdminInviteArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  user_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Invite codes + access requests ──────────────────────────────────────────
export const listInviteCodesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional()
});

export const listInviteRequestsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  status: z.enum(["pending", "approved", "denied", "all"]).optional()
});

export const createInviteCodeArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  label: z.string().min(1).max(256),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const inviteCodeIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  code_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const sendInviteCodeArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  code_id: z.string().uuid(),
  email: z.string().email(),
  name: z.string().min(1).max(255).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const inviteRequestActionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  invite_request_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Protocol templates ───────────────────────────────────────────────────────
const parameterRule = z.object({
  tag_keyword: z.string().min(1),
  target: z.union([z.string(), z.number(), z.array(z.string())]),
  tolerance: z.number().optional(),
  match_type: z.enum(["numeric", "exact", "contains_all", "range"]).optional(),
  severity: z.enum(["critical", "warning", "info"]).optional(),
  description: z.string().optional()
});

export const createProtocolTemplateArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  name: z.string().min(1).max(128),
  description: z.string().max(512).optional(),
  manufacturer: z.string().max(128).optional(),
  model: z.string().max(128).optional(),
  software_version: z.string().max(128).optional(),
  sequence_type: z.string().max(128).optional(),
  rules: z.array(parameterRule).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateProtocolTemplateArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  template_id: z.string().uuid(),
  name: z.string().min(1).max(128),
  description: z.string().max(512).optional(),
  manufacturer: z.string().max(128).optional(),
  model: z.string().max(128).optional(),
  software_version: z.string().max(128).optional(),
  sequence_type: z.string().max(128).optional(),
  rules: z.array(parameterRule).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const templateIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  template_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Anonymization profiles ───────────────────────────────────────────────────
export const createAnonProfileArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  name: z.string().min(1).max(128),
  description: z.string().max(512).optional(),
  retained_tags: z.array(z.string()).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateAnonProfileArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  profile_id: z.string().uuid(),
  name: z.string().min(1).max(128),
  description: z.string().max(512).optional(),
  retained_tags: z.array(z.string()).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const deleteAnonProfileArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  profile_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const setDefaultAnonProfileArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  profile_id: z.string().uuid().or(z.literal("")),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Institutions ─────────────────────────────────────────────────────────────
export const createInstitutionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  name: z.string().min(1).max(256),
  type: z.enum(["sender", "receiver", "both"]),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with hyphens").optional(),
  description: z.string().max(1024).optional(),
  contact_name: z.string().max(256).optional(),
  contact_email: z.string().email().optional(),
  ip_ranges: z.string().optional(),
  ae_title: z.string().max(16).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateInstitutionArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  institution_id: z.string().uuid(),
  name: z.string().min(1).max(256),
  type: z.enum(["sender", "receiver", "both"]),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with hyphens").optional(),
  description: z.string().max(1024).optional(),
  contact_name: z.string().max(256).optional(),
  contact_email: z.string().email().optional(),
  ip_ranges: z.string().optional(),
  ae_title: z.string().max(16).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const institutionIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  institution_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const linkInstitutionProjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  institution_id: z.string().uuid(),
  project_id: z.string().uuid(),
  role: z.enum(["sender", "receiver", "admin"]),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const unlinkInstitutionProjectArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  institution_id: z.string().uuid(),
  project_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Federation peers ─────────────────────────────────────────────────────────
export const createFederationPeerArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  name: z.string().min(1).max(256),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with hyphens").optional(),
  api_url: z.string().url(),
  notes: z.string().max(1024).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateFederationPeerArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  peer_id: z.string().uuid(),
  name: z.string().min(1).max(256),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with hyphens").optional(),
  api_url: z.string().url(),
  notes: z.string().max(1024).optional(),
  enabled: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const federationPeerIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  peer_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Project config ───────────────────────────────────────────────────────────
export const setStorageQuotaArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  storage_quota_bytes: z.number().int().positive().nullable(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updatePhiConfigArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  confidence_threshold: z.number().min(0).max(1).optional(),
  min_text_length: z.number().int().positive().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Study management ─────────────────────────────────────────────────────────
export const deleteStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const bulkPipelineTriggerArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_ids: z.array(z.string().uuid()).min(1).max(200),
  step: z.enum(["classify", "phi_scan", "protocol", "deface", "qc", "bids", "export"]),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── TCIA (The Cancer Imaging Archive) ───────────────────────────────────────
export const getTCIASeriesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  collection: z.string().min(1).max(128),
  min_slices: z.number().int().positive().optional()
});

export const importTCIASeriesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  series_uid: z.string().min(1).max(256),
  collection: z.string().min(1).max(128).optional(),
  project_slug: z.string().min(1).max(64).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Protocol template import ─────────────────────────────────────────────────
export const exportProtocolTemplatesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid()
});

export const importProtocolTemplatesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  templates: z.array(z.object({
    name: z.string().min(1).max(128),
    description: z.string().max(512).optional(),
    manufacturer: z.string().max(128).optional(),
    model: z.string().max(128).optional(),
    software_version: z.string().max(128).optional(),
    sequence_type: z.string().max(128).optional(),
    rules: z.array(z.record(z.unknown())).optional()
  })).min(1).max(200),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Project BIDS availability ────────────────────────────────────────────────
export const getProjectBidsInfoArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  status: z.enum(["approved", "received", "clean", "defaced"]).optional()
    .describe("Study status filter (default: 'approved')")
});

export const getCohortReportArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().describe("Project UUID to generate cohort report for.")
});

export const getRetentionPreviewArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().describe("Project UUID to preview retention for."),
  days: z.number().int().min(1).max(3650).optional()
    .describe("Simulated retention period in days (default 90). Studies older than this would be expired.")
});

// ─── Webhook stats + all deliveries ──────────────────────────────────────────
export const getWebhookStatsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid()
});

export const listAllWebhookDeliveriesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  subscription_id: z.string().uuid().optional(),
  success: z.enum(["true", "false"]).optional(),
  limit: z.number().int().min(1).max(200).optional(),
  offset: z.number().int().min(0).optional()
});

// ─── Batch import ─────────────────────────────────────────────────────────────
export const batchImportStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  dir: z.string().min(1).max(1024).describe("Absolute path on the server to directory containing DICOM files"),
  project_slug: z.string().min(1).max(64).optional(),
  institution_id: z.string().uuid().optional(),
  institution_slug: z.string().min(1).max(64).optional(),
  source: z.enum(["internal", "external"]).optional(),
  dry_run: z.boolean().optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Admin user preferences ───────────────────────────────────────────────────
export const getUserPreferencesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  user_id: z.string().uuid()
});

export const setUserPreferencesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  user_id: z.string().uuid(),
  digest_frequency: z.enum(["none", "daily", "weekly", "monthly"]),
  notify_events: z.array(z.string().min(1).max(128)).max(20).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const exportRoutingRulesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid()
});

export const importRoutingRulesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  rules: z.array(z.object({
    name: z.string().min(1).max(128),
    description: z.string().max(512).optional(),
    priority: z.number().int().min(0).optional(),
    enabled: z.boolean().optional(),
    modality: z.string().max(16).optional(),
    body_part: z.string().max(64).optional(),
    source: z.enum(["external", "internal"]).optional().nullable(),
    action: z.string().min(1).max(64)
  })).min(1).max(200),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const reorderRoutingRulesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  rules: z.array(z.object({
    id: z.string().uuid(),
    priority: z.number().int().min(0)
  })).min(1).max(200),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const bulkToggleRoutingRulesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  rule_ids: z.array(z.string().uuid()).min(1).max(200),
  enabled: z.boolean(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const queryPacsArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  ae_title: z.string().min(1).max(64),
  host: z.string().min(1).max(256),
  port: z.number().int().min(1).max(65535),
  query_level: z.enum(["PATIENT", "STUDY", "SERIES"]).default("STUDY"),
  query_params: z.record(z.string(), z.string()).optional()
});

export const retrievePacsStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  ae_title: z.string().min(1).max(64),
  host: z.string().min(1).max(256),
  port: z.number().int().min(1).max(65535),
  study_instance_uid: z.string().min(4).max(256).regex(/^[0-9.]+$/),
  move_destination: z.string().max(64).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Protocol compliance trend ────────────────────────────────────────────────
export const getProtocolTrendArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().optional(),
  days: z.number().int().min(1).max(365).optional()
});

// ─── Soft-delete / restore ────────────────────────────────────────────────────
export const listDeletedStudiesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid().optional(),
  limit: z.number().int().min(1).max(200).optional(),
  offset: z.number().int().min(0).optional()
});

export const softDeleteStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const restoreStudyArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Study notes + bulk share ─────────────────────────────────────────────────
export const listStudyNotesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid()
});

export const bulkCreateSharesArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_ids: z.array(z.string().uuid()).min(1).max(200),
  recipient_email: z.string().email(),
  note: z.string().max(500).optional(),
  expiry_hours: z.number().int().min(1).max(8760).optional(),
  max_downloads: z.number().int().min(1).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

// ─── Project member schemas ───────────────────────────────────────────────────
export const listProjectMembersArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid()
});

export const addProjectMemberArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  admin_user_id: z.string().uuid(),
  role: z.enum(["owner", "coordinator", "reviewer", "site_coordinator", "site_viewer"]),
  institution_id: z.string().uuid().optional().nullable(),
  notes: z.string().max(500).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const updateProjectMemberArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  member_id: z.string().uuid(),
  role: z.enum(["owner", "coordinator", "reviewer", "site_coordinator", "site_viewer"]),
  institution_id: z.string().uuid().optional().nullable(),
  notes: z.string().max(500).optional(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const removeProjectMemberArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  member_id: z.string().uuid(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const toggleProjectRestrictedArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  project_id: z.string().uuid(),
  restricted: z.boolean(),
  reason: z.string().min(10).max(512),
  confirm: z.literal(true)
});

export const readToolNames = [
  "list_studies",
  "get_study_detail",
  "get_study_diagnostics",
  "get_study_processing_summary",
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
  "test_destination",
  "get_routing_stats",
  "get_destination_stats",
  "get_routing_rule_stats",
  "get_pipeline_funnel",
  "get_project_health",
  "get_compliance_report",
  "get_storage_usage",
  "get_anonymization_diff",
  "get_system_health_summary",
  "list_study_relationships",
  "list_digest_subscriptions",
  "get_daily_summary",
  "list_admin_users",
  "list_invite_codes",
  "list_invite_requests",
  "get_tcia_series",
  "export_protocol_templates",
  "get_webhook_stats",
  "list_all_webhook_deliveries",
  "get_user_preferences",
  "get_project_bids_info",
  "get_expiring_studies",
  "get_retention_preview",
  "get_destination_health",
  "simulate_routing",
  "get_cohort_report",
  "export_routing_rules",
  "list_project_members",
  "query_pacs",
  "list_deleted_studies",
  "get_protocol_trend",
  "list_study_notes",
  "get_label_usage",
  "get_modality_trend",
  "get_source_trend",
  "get_export_shares_csv",
  "get_institution_breakdown",
  "get_compliance_report_csv"
] as const;

export const writeToolNames = [
  "trigger_classification",
  "trigger_phi_scan",
  "trigger_pixel_redaction",
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
  "delete_routing_rule",
  "link_studies",
  "unlink_studies",
  "create_digest_subscription",
  "delete_digest_subscription",
  "create_webhook_subscription",
  "update_webhook_subscription",
  "delete_webhook_subscription",
  "retry_webhook_delivery",
  "create_project",
  "update_project",
  "archive_project",
  "restore_project",
  "set_project_retention",
  "set_project_sla_threshold",
  "create_admin_user",
  "update_admin_user",
  "delete_admin_user",
  "send_admin_invite",
  "create_invite_code",
  "revoke_invite_code",
  "delete_invite_code",
  "send_invite_code",
  "approve_invite_request",
  "deny_invite_request",
  "create_protocol_template",
  "update_protocol_template",
  "delete_protocol_template",
  "create_anon_profile",
  "update_anon_profile",
  "delete_anon_profile",
  "set_default_anon_profile",
  "create_institution",
  "update_institution",
  "delete_institution",
  "link_institution_project",
  "unlink_institution_project",
  "create_federation_peer",
  "update_federation_peer",
  "delete_federation_peer",
  "set_storage_quota",
  "update_phi_config",
  "delete_study",
  "bulk_pipeline_trigger",
  "import_tcia_series",
  "import_protocol_templates",
  "batch_import_studies",
  "set_user_preferences",
  "re_evaluate_project_routing",
  "import_routing_rules",
  "reorder_routing_rules",
  "bulk_toggle_routing_rules",
  "retrieve_pacs_study",
  "soft_delete_study",
  "restore_study",
  "bulk_create_shares",
  "add_project_member",
  "update_project_member",
  "remove_project_member",
  "toggle_project_restricted"
] as const;

export type ReadToolName = (typeof readToolNames)[number];
export type WriteToolName = (typeof writeToolNames)[number];
export type ToolName = ReadToolName | WriteToolName;
