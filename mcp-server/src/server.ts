import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema } from "@modelcontextprotocol/sdk/types.js";
import { startAgentHttpServer } from "./agentServer.js";
import { client, config } from "./context.js";
import { executeTool } from "./dispatcher.js";
import { InMemoryIdempotencyCache } from "./idempotencyCache.js";
import { InMemoryRateLimiter } from "./rateLimiter.js";
import { redactToolArgs } from "./redaction.js";
import type { ToolName } from "./schemas.js";
import { tools } from "./tools.js";
import type {
  InvocationLog,
  InvocationLogResult,
  ToolClass,
  ToolPayload,
  ToolResponse,
} from "./types.js";
import {
  buildRequestId,
  formatError,
  isKnownToolName,
  isWriteToolName,
} from "./types.js";

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

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
    name === "unlink_studies" ||
    name === "submit_qc_rating"
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
  if (name === "upsert_demographics") {
    return typeof args.subject_id === "string" ? args.subject_id : null;
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

// ---------------------------------------------------------------------------
// Server setup
// ---------------------------------------------------------------------------

const rateLimiter = new InMemoryRateLimiter();
const writeIdempotencyCache = new InMemoryIdempotencyCache<ToolResponse>();

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

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

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

main().catch((error) => {
  console.error("Fatal MCP server error", error);
  process.exit(1);
});
