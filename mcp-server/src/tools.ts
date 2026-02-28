import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { writeInputSchema } from "./types.js";

export const tools: Tool[] = [
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
  },
  // --- Biomarker database tools ---
  {
    name: "get_roi_results",
    description: "Get structured ROI biomarker results for a study. Filterable by tool (freesurfer, atlas_roi), atlas (aseg, aparc, AAL3), metric_type (volume_mm3, mean, median, std), roi_name, hemisphere (L/R/B). Returns per-ROI measurements from neuroimaging analytics.",
    inputSchema: {
      type: "object",
      required: ["study_id"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        tool: { type: "string", description: "Filter by analytics tool (e.g. freesurfer, atlas_roi)" },
        atlas_name: { type: "string", description: "Filter by atlas (e.g. aseg, aparc, AAL3)" },
        metric_type: { type: "string", description: "Filter by metric type (e.g. volume_mm3, mean)" },
        roi_name: { type: "string", description: "Filter by ROI name" },
        hemisphere: { type: "string", enum: ["L", "R", "B"], description: "Filter by hemisphere" },
        limit: { type: "integer", minimum: 1, maximum: 2000, description: "Max results (default 500)" }
      },
      additionalProperties: false
    }
  },
  {
    name: "get_roi_results_summary",
    description: "Get aggregate summary of ROI results for a study: total ROIs, tools used, atlases, metric types. Quick way to check what analytics data is available.",
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
    name: "get_composite_scores",
    description: "Get composite scores for a study (e.g. AD-signature atrophy composite from TBM-SyN). Returns score name, value, tool, and metadata.",
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
    name: "get_longitudinal_roi_results",
    description: "Get longitudinal ROI results for a study — paired baseline/follow-up measurements with change values, percent change, and annualized change rates. Includes data from TBM-SyN and FreeSurfer longitudinal stream.",
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
    name: "get_subject_roi_comparison",
    description: "Get ROI results across all studies for a subject — enables cross-study longitudinal comparison. Returns results annotated with study_id, study_uid, and study_date.",
    inputSchema: {
      type: "object",
      required: ["subject_id"],
      properties: {
        request_id: { type: "string" },
        subject_id: { type: "string", description: "Subject identifier" },
        project_id: { type: "string", format: "uuid", description: "Scope to a project" }
      },
      additionalProperties: false
    }
  },
  {
    name: "export_project_roi_data",
    description: "Export all ROI results for a project as a CSV payload (up to 50k rows). Includes study_id, study_uid, subject_id, tool, atlas, ROI name, metric type/value, hemisphere.",
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
    name: "get_qc_ratings",
    description: "Get human analyst QC ratings for a study. Includes analyst name/email, Likert-scale ratings (quality, motion, SNR, coverage, artifacts, overall 0-4), comments, and review type.",
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
    name: "get_subject_demographics",
    description: "Get de-identified demographics for a subject in a project: sex, age_at_scan, diagnosis, education_years, MMSE/MoCA scores, CDR global, APOE genotype.",
    inputSchema: {
      type: "object",
      required: ["subject_id", "project_id"],
      properties: {
        request_id: { type: "string" },
        subject_id: { type: "string" },
        project_id: { type: "string", format: "uuid" }
      },
      additionalProperties: false
    }
  },
  {
    name: "list_project_demographics",
    description: "List all subject demographics for a project. Returns an array of de-identified research metadata records.",
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
    name: "list_analytics_files",
    description: "List analytics output files for a study (NIfTI volumes, FreeSurfer surfaces/overlays, stats files, CSVs, JSON). Files are grouped by tool with type classification.",
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
    name: "submit_qc_rating",
    description: "Submit or update a human analyst QC rating for a study. Ratings use Likert scales 0-4 (0=unacceptable, 4=excellent). One rating per analyst per review_type per study (upserts on conflict). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["study_id", "rating_quality", "rating_motion", "rating_overall", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        study_id: { type: "string", format: "uuid" },
        rating_quality: { type: "integer", minimum: 0, maximum: 4, description: "Overall image quality (0-4)" },
        rating_motion: { type: "integer", minimum: 0, maximum: 4, description: "Motion artifact severity (0=severe, 4=none)" },
        rating_snr: { type: "integer", minimum: 0, maximum: 4, description: "Signal-to-noise ratio (0-4)" },
        rating_coverage: { type: "integer", minimum: 0, maximum: 4, description: "Brain coverage completeness (0-4)" },
        rating_artifacts: { type: "integer", minimum: 0, maximum: 4, description: "Non-motion artifacts (0-4)" },
        rating_overall: { type: "integer", minimum: 0, maximum: 4, description: "Overall usability rating (0-4)" },
        comments: { type: "string", maxLength: 2000, description: "Free-text comments" },
        review_type: { type: "string", enum: ["initial", "consensus", "adjudication"], description: "Type of review (default: initial)" },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  },
  {
    name: "upsert_demographics",
    description: "Create or update de-identified demographics for a subject in a project. All fields except subject_id and project_id are optional (partial updates supported). Requires confirm=true and a reason.",
    inputSchema: {
      type: "object",
      required: ["subject_id", "project_id", "reason", "confirm"],
      properties: {
        request_id: { type: "string" },
        subject_id: { type: "string" },
        project_id: { type: "string", format: "uuid" },
        sex: { type: "string", enum: ["M", "F", "O", ""], description: "Biological sex" },
        age_at_scan: { type: "integer", minimum: 0, maximum: 150, description: "Age at scan (years)" },
        diagnosis: { type: "string", maxLength: 256 },
        education_years: { type: "integer", minimum: 0, maximum: 30 },
        mmse_score: { type: "integer", minimum: 0, maximum: 30, description: "Mini-Mental State Exam score" },
        moca_score: { type: "integer", minimum: 0, maximum: 30, description: "Montreal Cognitive Assessment score" },
        cdr_global: { type: "number", minimum: 0, maximum: 3, description: "Clinical Dementia Rating global score" },
        apoe_genotype: { type: "string", maxLength: 10, description: "APOE genotype (e.g. e3/e4)" },
        notes: { type: "string", maxLength: 2000 },
        reason: { type: "string", minLength: 10, maxLength: 512 },
        confirm: { type: "boolean", const: true }
      },
      additionalProperties: false
    }
  }
];
