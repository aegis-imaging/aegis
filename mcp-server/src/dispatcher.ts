import { DisallowedPathError, UpstreamHttpError } from "./aegisClient.js";
import { client } from "./context.js";
import {
  writeToolNames,
  addProjectMemberArgsSchema,
  addStudyLabelArgsSchema,
  addStudyNoteArgsSchema,
  anonDiffArgsSchema,
  apiKeyIdArgsSchema,
  approveStudyArgsSchema,
  archiveRestoreProjectArgsSchema,
  batchImportStudiesArgsSchema,
  bulkCreateSharesArgsSchema,
  bulkLabelStudiesArgsSchema,
  bulkPipelineTriggerArgsSchema,
  bulkStudyActionArgsSchema,
  bulkToggleRoutingRulesArgsSchema,
  cloneProjectArgsSchema,
  complianceReportArgsSchema,
  createAdminUserArgsSchema,
  createAnonProfileArgsSchema,
  createApiKeyArgsSchema,
  createDestinationArgsSchema,
  createDigestSubscriptionArgsSchema,
  createFederationPeerArgsSchema,
  createInstitutionArgsSchema,
  createInviteCodeArgsSchema,
  createProjectArgsSchema,
  createProtocolTemplateArgsSchema,
  createRoutingRuleArgsSchema,
  createShareArgsSchema,
  createWebhookSubscriptionArgsSchema,
  deleteAdminUserArgsSchema,
  deleteAnonProfileArgsSchema,
  deleteDigestSubscriptionArgsSchema,
  deleteStudyArgsSchema,
  deleteWebhookSubscriptionArgsSchema,
  destinationIdArgsSchema,
  destinationStatsArgsSchema,
  dimseRetryStatusArgsSchema,
  emptyArgsSchema,
  exportProjectBatchArgsSchema,
  exportProtocolTemplatesArgsSchema,
  exportRoutingRulesArgsSchema,
  exportSharesCsvArgsSchema,
  extendShareArgsSchema,
  federationPeerIdArgsSchema,
  generateSyntheticStudyArgsSchema,
  getAuditActorsArgsSchema,
  getCohortReportArgsSchema,
  getDailySummaryArgsSchema,
  getDestinationHealthArgsSchema,
  getExpiringStudiesArgsSchema,
  getIngestionTimelineArgsSchema,
  getInstitutionStatsArgsSchema,
  getProjectBidsInfoArgsSchema,
  getProtocolTrendArgsSchema,
  getRetentionPreviewArgsSchema,
  getShareDownloadsArgsSchema,
  getStuckStudiesArgsSchema,
  getStudyDicomTagsArgsSchema,
  getTCIASeriesArgsSchema,
  getUserPreferencesArgsSchema,
  getWebhookDeliveriesArgsSchema,
  getWebhookStatsArgsSchema,
  importProtocolTemplatesArgsSchema,
  importRoutingRulesArgsSchema,
  importTCIASeriesArgsSchema,
  institutionIdArgsSchema,
  inviteCodeIdArgsSchema,
  inviteRequestActionArgsSchema,
  linkInstitutionProjectArgsSchema,
  linkStudiesArgsSchema,
  listAdminUsersArgsSchema,
  listAllSharesArgsSchema,
  listAllWebhookDeliveriesArgsSchema,
  listAuditArgsSchema,
  listDigestSubscriptionsArgsSchema,
  listInviteCodesArgsSchema,
  listInviteRequestsArgsSchema,
  listProjectMembersArgsSchema,
  listProtocolTemplatesArgsSchema,
  listStudiesArgsSchema,
  listStudyNotesArgsSchema,
  listStudyRelationshipsArgsSchema,
  pipelineFunnelArgsSchema,
  processingTimesArgsSchema,
  projectHealthArgsSchema,
  projectScopedArgsSchema,
  queryPacsArgsSchema,
  reEvaluateProjectRoutingArgsSchema,
  reEvaluateRoutingArgsSchema,
  reactivateStudyArgsSchema,
  reassignStudyArgsSchema,
  rejectStudyArgsSchema,
  removeProjectMemberArgsSchema,
  removeStudyLabelArgsSchema,
  reorderRoutingRulesArgsSchema,
  resetPipelineStepArgsSchema,
  restoreStudyArgsSchema,
  retrievePacsStudyArgsSchema,
  retryDimseArgsSchema,
  retryWebhookDeliveryArgsSchema,
  revokeShareArgsSchema,
  routingRuleIdArgsSchema,
  routingRuleStatsArgsSchema,
  routingStatsArgsSchema,
  sendAdminInviteArgsSchema,
  sendInviteCodeArgsSchema,
  setDefaultAnonProfileArgsSchema,
  setProjectRetentionArgsSchema,
  setProjectSLAThresholdArgsSchema,
  setStorageQuotaArgsSchema,
  setStudySubjectArgsSchema,
  setUserPreferencesArgsSchema,
  simulateRoutingArgsSchema,
  softDeleteStudyArgsSchema,
  storageUsageArgsSchema,
  studyIdArgsSchema,
  studyUidArgsSchema,
  templateIdArgsSchema,
  testDestinationArgsSchema,
  testWebhookArgsSchema,
  toggleProjectRestrictedArgsSchema,
  toggleStudyFlagArgsSchema,
  triggerLongitudinalAnalyticsArgsSchema,
  unlinkInstitutionProjectArgsSchema,
  unlinkStudiesArgsSchema,
  updateAdminUserArgsSchema,
  updateAnonProfileArgsSchema,
  updateDestinationArgsSchema,
  updateFederationPeerArgsSchema,
  updateInstitutionArgsSchema,
  updatePhiConfigArgsSchema,
  updateProjectArgsSchema,
  updateProjectMemberArgsSchema,
  updateProtocolTemplateArgsSchema,
  updateRoutingRuleArgsSchema,
  updateWebhookSubscriptionArgsSchema,
  writeArgsSchema,
  roiResultsArgsSchema,
  subjectRoiArgsSchema,
  projectRoiExportArgsSchema,
  qcRatingsArgsSchema,
  submitQcRatingArgsSchema,
  demographicsArgsSchema,
  upsertDemographicsArgsSchema,
  analyticsFilesArgsSchema,
} from "./schemas.js";
import type { ToolResponse } from "./types.js";
import { buildRequestId, formatError, formatSuccess, isKnownToolName } from "./types.js";
import {
  denyWriteTool,
  handleAddProjectMember,
  handleAddStudyLabel,
  handleAddStudyNote,
  handleApproveInviteRequest,
  handleApproveStudy,
  handleArchiveProject,
  handleBatchImportStudies,
  handleBulkCreateShares,
  handleBulkLabelStudies,
  handleBulkPipelineTrigger,
  handleBulkStudyAction,
  handleBulkToggleRoutingRules,
  handleCloneProject,
  handleCreateAdminUser,
  handleCreateAnonProfile,
  handleCreateApiKey,
  handleCreateDestination,
  handleCreateDigestSubscription,
  handleCreateFederationPeer,
  handleCreateInstitution,
  handleCreateInviteCode,
  handleCreateProject,
  handleCreateProtocolTemplate,
  handleCreateRoutingRule,
  handleCreateShare,
  handleCreateWebhookSubscription,
  handleDeleteAdminUser,
  handleDeleteAnonProfile,
  handleDeleteApiKey,
  handleDeleteDestination,
  handleDeleteDigestSubscription,
  handleDeleteFederationPeer,
  handleDeleteInstitution,
  handleDeleteInviteCode,
  handleDeleteProtocolTemplate,
  handleDeleteRoutingRule,
  handleDeleteStudy,
  handleDeleteWebhookSubscription,
  handleDenyInviteRequest,
  handleExportProjectBatch,
  handleExtendShare,
  handleGenerateSyntheticStudy,
  handleImportProtocolTemplates,
  handleImportRoutingRules,
  handleImportTCIASeries,
  handleLinkInstitutionProject,
  handleLinkStudies,
  handleReEvaluateRouting,
  handleReactivateStudy,
  handleReassignStudy,
  handleRejectStudy,
  handleRemoveProjectMember,
  handleRemoveStudyLabel,
  handleReorderRoutingRules,
  handleResetPipelineStep,
  handleRestoreProject,
  handleRestoreStudy,
  handleRetrievePacsStudy,
  handleRetryDimseStudy,
  handleRetryWebhookDelivery,
  handleRevokeInviteCode,
  handleRevokeShare,
  handleRotateApiKey,
  handleSendAdminInvite,
  handleSendInviteCode,
  handleSetApiKeyEnabled,
  handleSetDefaultAnonProfile,
  handleSetProjectRetention,
  handleSetProjectSLAThreshold,
  handleSetStorageQuota,
  handleSetStudySubject,
  handleSetUserPreferences,
  handleSoftDeleteStudy,
  handleTestWebhook,
  handleToggleProjectRestricted,
  handleToggleStudyFlag,
  handleTriggerAnalytics,
  handleTriggerSct,
  handleTriggerBidsConvert,
  handleTriggerClassification,
  handleTriggerDeface,
  handleTriggerExport,
  handleTriggerLongitudinalAnalytics,
  handleTriggerPhiScan,
  handleTriggerPixelRedaction,
  handleTriggerProtocolCheck,
  handleTriggerQcCheck,
  handleUnlinkInstitutionProject,
  handleUnlinkStudies,
  handleUpdateAdminUser,
  handleUpdateAnonProfile,
  handleUpdateDestination,
  handleUpdateFederationPeer,
  handleUpdateInstitution,
  handleUpdatePhiConfig,
  handleUpdateProject,
  handleUpdateProjectMember,
  handleUpdateProtocolTemplate,
  handleUpdateRoutingRule,
  handleUpdateWebhookSubscription,
  handleSubmitQcRating,
  handleUpsertDemographics,
} from "./handlers.js";

export async function executeTool(name: string, args: Record<string, unknown>, requestId: string): Promise<ToolResponse> {
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

    // ---- Biomarker database: ROI results, QC ratings, demographics, analytics files ----

    if (name === "get_roi_results") {
      const parsed = roiResultsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.tool) params.set("tool", parsed.tool);
      if (parsed.atlas_name) params.set("atlas_name", parsed.atlas_name);
      if (parsed.metric_type) params.set("metric_type", parsed.metric_type);
      if (parsed.roi_name) params.set("roi_name", parsed.roi_name);
      if (parsed.hemisphere) params.set("hemisphere", parsed.hemisphere);
      if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
      const qs = params.toString() ? `?${params.toString()}` : "";
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/roi-results${qs}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_roi_results_summary") {
      const parsed = roiResultsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.tool) params.set("tool", parsed.tool);
      if (parsed.atlas_name) params.set("atlas_name", parsed.atlas_name);
      const qs = params.toString() ? `?${params.toString()}` : "";
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/roi-results/summary${qs}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_composite_scores") {
      const parsed = roiResultsArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/composite-scores`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_longitudinal_roi_results") {
      const parsed = roiResultsArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.tool) params.set("tool", parsed.tool);
      if (parsed.atlas_name) params.set("atlas_name", parsed.atlas_name);
      if (parsed.roi_name) params.set("roi_name", parsed.roi_name);
      const qs = params.toString() ? `?${params.toString()}` : "";
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/longitudinal-roi-results${qs}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_subject_roi_comparison") {
      const parsed = subjectRoiArgsSchema.parse(args);
      const params = new URLSearchParams();
      if (parsed.project_id) params.set("project_id", parsed.project_id);
      const qs = params.toString() ? `?${params.toString()}` : "";
      const data = await client.get(`/api/subjects/${encodeURIComponent(parsed.subject_id)}/roi-results${qs}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "export_project_roi_data") {
      const parsed = projectRoiExportArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/roi-export`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_qc_ratings") {
      const parsed = qcRatingsArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/qc-ratings`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "get_subject_demographics") {
      const parsed = demographicsArgsSchema.parse(args);
      const data = await client.get(`/api/subjects/${encodeURIComponent(parsed.subject_id)}/demographics?project_id=${encodeURIComponent(parsed.project_id)}`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_project_demographics") {
      const parsed = projectRoiExportArgsSchema.parse(args);
      const data = await client.get(`/api/projects/${encodeURIComponent(parsed.project_id)}/demographics`);
      return formatSuccess(requestId, name, data);
    }

    if (name === "list_analytics_files") {
      const parsed = analyticsFilesArgsSchema.parse(args);
      const data = await client.get(`/api/studies/${encodeURIComponent(parsed.study_id)}/analytics-files`);
      return formatSuccess(requestId, name, data);
    }

    if ((writeToolNames as readonly string[]).includes(name)) {
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

      if (name === "trigger_sct") {
        return handleTriggerSct(parsed.request_id ?? buildRequestId(), parsed);
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

      if (name === "submit_qc_rating") {
        const parsed = submitQcRatingArgsSchema.parse(args);
        return handleSubmitQcRating(parsed.request_id ?? buildRequestId(), parsed);
      }

      if (name === "upsert_demographics") {
        const parsed = upsertDemographicsArgsSchema.parse(args);
        return handleUpsertDemographics(parsed.request_id ?? buildRequestId(), parsed);
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
