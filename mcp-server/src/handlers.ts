import { client, config } from "./context.js";
import type { ToolResponse } from "./types.js";
import { extractStudies, extractDimseRetryDetails, formatSuccess, formatError } from "./types.js";

import type { ToolName } from "./schemas.js";

export function denyWriteTool(requestId: string, tool: ToolName) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, tool);
  }

  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are intentionally disabled in this scaffold (set MCP_ENABLE_WRITE_TOOLS=true only after implementation)", false, tool);
  }

  return formatError(requestId, "FORBIDDEN", "Write tool handler not implemented in scaffold", false, tool);
}

export async function handleTriggerClassification(
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

export async function handleTriggerBidsConvert(
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

export async function handleTriggerAnalytics(
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

export async function handleTriggerLongitudinalAnalytics(
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

export async function handleTriggerExport(
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

export async function handleTriggerDeface(
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

export async function handleRetryDimseStudy(
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

export async function handleTriggerQcCheck(
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

export async function handleTriggerProtocolCheck(
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

export async function handleTriggerPhiScan(
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

export async function handleTriggerPixelRedaction(
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

export async function handleCreateShare(
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

export async function handleReEvaluateRouting(
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

export async function handleApproveStudy(
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

export async function handleRejectStudy(
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

export async function handleRevokeShare(
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

export async function handleResetPipelineStep(
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

export async function handleReassignStudy(
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

export async function handleAddStudyLabel(
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

export async function handleRemoveStudyLabel(
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

export async function handleSetStudySubject(
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

export async function handleLinkStudies(
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

export async function handleUnlinkStudies(
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

export async function handleCreateDigestSubscription(
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

export async function handleDeleteDigestSubscription(
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

export async function handleAddStudyNote(
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

export async function handleBulkCreateShares(
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

export async function handleToggleStudyFlag(
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

export async function handleCloneProject(
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

export async function handleExtendShare(
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

export async function handleExportProjectBatch(
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

export async function handleReactivateStudy(
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

export async function handleCreateInviteCode(
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

export async function handleRevokeInviteCode(
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

export async function handleDeleteInviteCode(
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

export async function handleSendInviteCode(
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

export async function handleApproveInviteRequest(
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

export async function handleDenyInviteRequest(
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

export async function handleCreateProject(
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

export async function handleUpdateProject(
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

export async function handleArchiveProject(
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

export async function handleRestoreProject(
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

export async function handleSetProjectRetention(
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

export async function handleSetProjectSLAThreshold(
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

export async function handleAddProjectMember(
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

export async function handleUpdateProjectMember(
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

export async function handleRemoveProjectMember(
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

export async function handleToggleProjectRestricted(
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

export async function handleCreateAdminUser(
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

export async function handleUpdateAdminUser(
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

export async function handleDeleteAdminUser(
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

export async function handleSendAdminInvite(
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

export async function handleCreateWebhookSubscription(
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

export async function handleUpdateWebhookSubscription(
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

export async function handleDeleteWebhookSubscription(
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

export async function handleRetryWebhookDelivery(
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

export async function handleTestWebhook(
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

export async function handleCreateApiKey(
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

export async function handleRotateApiKey(
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

export async function handleSetApiKeyEnabled(
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

export async function handleDeleteApiKey(
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

export async function handleBulkStudyAction(
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

export async function handleBulkLabelStudies(
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

export async function handleGenerateSyntheticStudy(
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

export async function handleUpdateDestination(
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

export async function handleCreateDestination(
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

export async function handleDeleteDestination(
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

export async function handleCreateRoutingRule(
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

export async function handleUpdateRoutingRule(
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

export async function handleDeleteRoutingRule(
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

export async function handleCreateProtocolTemplate(
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

export async function handleUpdateProtocolTemplate(
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

export async function handleDeleteProtocolTemplate(
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

export async function handleCreateAnonProfile(
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

export async function handleUpdateAnonProfile(
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

export async function handleDeleteAnonProfile(
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

export async function handleSetDefaultAnonProfile(
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

export async function handleCreateInstitution(
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

export async function handleUpdateInstitution(
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

export async function handleDeleteInstitution(
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

export async function handleLinkInstitutionProject(
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

export async function handleUnlinkInstitutionProject(
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

export async function handleCreateFederationPeer(
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

export async function handleUpdateFederationPeer(
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

export async function handleDeleteFederationPeer(
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

export async function handleSetStorageQuota(
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

export async function handleUpdatePhiConfig(
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

export async function handleDeleteStudy(
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

export async function handleBulkPipelineTrigger(
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

export async function handleImportTCIASeries(
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

export async function handleImportProtocolTemplates(
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

export async function handleBatchImportStudies(
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

export async function handleSetUserPreferences(
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

export async function handleImportRoutingRules(
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

export async function handleReorderRoutingRules(
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

export async function handleBulkToggleRoutingRules(
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

export async function handleRetrievePacsStudy(
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

export async function handleSoftDeleteStudy(
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

export async function handleRestoreStudy(
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

// ---- Biomarker database: QC ratings, demographics ----

export async function handleSubmitQcRating(
  requestId: string,
  parsed: {
    study_id: string;
    rating_quality: number;
    rating_motion: number;
    rating_snr?: number;
    rating_coverage?: number;
    rating_artifacts?: number;
    rating_overall: number;
    comments?: string;
    review_type?: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "submit_qc_rating");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow submit_qc_rating", false, "submit_qc_rating");
  }

  const body: Record<string, unknown> = {
    rating_quality: parsed.rating_quality,
    rating_motion: parsed.rating_motion,
    rating_overall: parsed.rating_overall,
  };
  if (parsed.rating_snr !== undefined) body.rating_snr = parsed.rating_snr;
  if (parsed.rating_coverage !== undefined) body.rating_coverage = parsed.rating_coverage;
  if (parsed.rating_artifacts !== undefined) body.rating_artifacts = parsed.rating_artifacts;
  if (parsed.comments) body.comments = parsed.comments;
  if (parsed.review_type) body.review_type = parsed.review_type;

  const data = await client.post(`/api/studies/${encodeURIComponent(parsed.study_id)}/qc-ratings`, body);
  return formatSuccess(requestId, "submit_qc_rating", {
    accepted: true,
    study_id: parsed.study_id,
    reason: parsed.reason,
    result: data,
  });
}

export async function handleUpsertDemographics(
  requestId: string,
  parsed: {
    subject_id: string;
    project_id: string;
    sex?: string;
    age_at_scan?: number;
    diagnosis?: string;
    education_years?: number;
    mmse_score?: number;
    moca_score?: number;
    cdr_global?: number;
    apoe_genotype?: string;
    notes?: string;
    reason: string;
    confirm: true;
  }
) {
  if (config.mcpMode !== "operator") {
    return formatError(requestId, "FORBIDDEN", "Caller is not permitted to execute write tools in readonly mode", false, "upsert_demographics");
  }
  if (!config.enableWriteTools) {
    return formatError(requestId, "FORBIDDEN", "Write tools are disabled; set MCP_ENABLE_WRITE_TOOLS=true to allow upsert_demographics", false, "upsert_demographics");
  }

  const body: Record<string, unknown> = {
    project_id: parsed.project_id,
  };
  if (parsed.sex !== undefined) body.sex = parsed.sex;
  if (parsed.age_at_scan !== undefined) body.age_at_scan = parsed.age_at_scan;
  if (parsed.diagnosis !== undefined) body.diagnosis = parsed.diagnosis;
  if (parsed.education_years !== undefined) body.education_years = parsed.education_years;
  if (parsed.mmse_score !== undefined) body.mmse_score = parsed.mmse_score;
  if (parsed.moca_score !== undefined) body.moca_score = parsed.moca_score;
  if (parsed.cdr_global !== undefined) body.cdr_global = parsed.cdr_global;
  if (parsed.apoe_genotype !== undefined) body.apoe_genotype = parsed.apoe_genotype;
  if (parsed.notes !== undefined) body.notes = parsed.notes;

  const data = await client.put(
    `/api/subjects/${encodeURIComponent(parsed.subject_id)}/demographics`,
    body
  );
  return formatSuccess(requestId, "upsert_demographics", {
    accepted: true,
    subject_id: parsed.subject_id,
    project_id: parsed.project_id,
    reason: parsed.reason,
    result: data,
  });
}
