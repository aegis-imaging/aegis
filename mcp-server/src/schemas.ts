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
  search: z.string().min(1).max(256).optional()
});

export const studyIdArgsSchema = z.object({
  request_id: z.string().min(8).max(128).optional(),
  study_id: z.string().uuid()
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

export const readToolNames = [
  "list_studies",
  "get_study_detail",
  "get_study_diagnostics",
  "get_study_audit",
  "get_study_routing_log",
  "list_export_shares",
  "get_system_health",
  "get_dimse_retry_status"
] as const;

export const writeToolNames = [
  "trigger_classification",
  "trigger_phi_scan",
  "trigger_protocol_check",
  "trigger_qc_check",
  "trigger_bids_convert",
  "trigger_export",
  "trigger_deface",
  "retry_dimse_study"
] as const;

export type ReadToolName = (typeof readToolNames)[number];
export type WriteToolName = (typeof writeToolNames)[number];
export type ToolName = ReadToolName | WriteToolName;
