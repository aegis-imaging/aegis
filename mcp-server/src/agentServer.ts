import http from "node:http";
import { AegisApiClient, DisallowedPathError, UpstreamHttpError } from "./aegisClient.js";
import { InMemoryRateLimiter } from "./rateLimiter.js";
import { buildLlmConfig, runAgentWithLlm, ALLOWED_MODEL_OVERRIDES, ALLOWED_BEDROCK_MODELS, ALLOWED_VERTEX_MODELS } from "./agentOrchestrator.js";

export type AgentHttpConfig = {
  port: number;
  allowedOrigin: string;
  apiKey?: string;
  bearerToken?: string;
  requireAuth: boolean;
  rateLimitPerMinute: number;
  llmBaseUrl?: string;
  llmApiKey?: string;
  llmModel: string;
  llmTemperature: number;
  llmMaxTokens: number;
  llmUseGcpAuth: boolean;
  llmGcpProject?: string;
  llmUseAwsBedrock: boolean;
  llmAwsRegion?: string;
};

type AgentRequest = {
  request_id?: string;
  question?: string;
  study_id?: string;
  study_instance_uid?: string;
  include_next_steps?: boolean;
  model?: string;
};

type AgentResponse = {
  ok: boolean;
  request_id: string;
  error?: {
    code: string;
    message: string;
  };
  data?: {
    summary: string;
    evidence: Array<{ field: string; value: string | number | boolean | null; timestamp?: string | null }>;
    diagnostics: {
      terminal: boolean;
      stuck: boolean;
      blockers: string[];
      recommended_actions: string[];
    };
    timeline: Array<{ event: string; timestamp: string | null }>;
    next_steps: string[];
  };
};

function buildRequestId(): string {
  return `agent_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
}

function sendJson(res: http.ServerResponse, status: number, payload: AgentResponse | Record<string, unknown>, allowedOrigin: string): void {
  res.statusCode = status;
  res.setHeader("Content-Type", "application/json");
  res.setHeader("Access-Control-Allow-Origin", allowedOrigin);
  res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Aegis-Agent-Key");
  res.setHeader("Access-Control-Allow-Methods", "POST, OPTIONS, GET");
  res.end(JSON.stringify(payload));
}

function readJsonBody(req: http.IncomingMessage): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      const raw = Buffer.concat(chunks).toString("utf-8").trim();
      if (!raw) {
        resolve({});
        return;
      }
      try {
        resolve(JSON.parse(raw));
      } catch (err) {
        reject(err);
      }
    });
    req.on("error", reject);
  });
}

function resolveClientKey(req: http.IncomingMessage): string | null {
  const headerKey = req.headers["x-aegis-agent-key"];
  if (typeof headerKey === "string" && headerKey.trim()) {
    return headerKey.trim();
  }
  return null;
}

function resolveBearerToken(req: http.IncomingMessage): string | null {
  const auth = req.headers.authorization;
  if (typeof auth === "string" && auth.toLowerCase().startsWith("bearer ")) {
    return auth.slice(7).trim();
  }
  return null;
}

function resolveClientIp(req: http.IncomingMessage): string {
  const forwarded = req.headers["x-forwarded-for"];
  if (typeof forwarded === "string" && forwarded.trim()) {
    return forwarded.split(",")[0].trim();
  }
  return req.socket.remoteAddress || "unknown";
}

function buildError(requestId: string, code: string, message: string): AgentResponse {
  return { ok: false, request_id: requestId, error: { code, message } };
}

const rateLimiter = new InMemoryRateLimiter();

export function startAgentHttpServer(client: AegisApiClient, config: AgentHttpConfig): void {
  const {
    port,
    allowedOrigin,
    apiKey,
    bearerToken,
    requireAuth,
    rateLimitPerMinute,
    llmBaseUrl,
    llmApiKey,
    llmModel,
    llmTemperature,
    llmMaxTokens,
    llmUseGcpAuth,
    llmGcpProject,
    llmUseAwsBedrock,
    llmAwsRegion
  } = config;

  if (requireAuth && !apiKey && !bearerToken) {
    throw new Error("Agent auth is required but no MCP_AGENT_API_KEY or MCP_AGENT_BEARER_TOKEN is configured");
  }

  const llmConfig = buildLlmConfig({
    baseUrl: llmBaseUrl,
    apiKey: llmApiKey,
    model: llmModel,
    temperature: llmTemperature,
    maxTokens: llmMaxTokens,
    useGcpAuth: llmUseGcpAuth,
    gcpProject: llmGcpProject,
    useAwsBedrock: llmUseAwsBedrock,
    awsRegion: llmAwsRegion
  });

  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url ?? "/", "http://localhost");

    if (req.method === "OPTIONS") {
      sendJson(res, 204, { ok: true, request_id: buildRequestId() }, allowedOrigin);
      return;
    }

    if (req.method === "GET" && url.pathname === "/healthz") {
      sendJson(res, 200, { ok: true, request_id: buildRequestId() }, allowedOrigin);
      return;
    }

    if (req.method === "GET" && url.pathname === "/agent/info") {
      const backend = llmConfig?.useAwsBedrock ? "bedrock" : "vertex";
      const modelSet = llmConfig?.useAwsBedrock ? ALLOWED_BEDROCK_MODELS : ALLOWED_VERTEX_MODELS;
      const BEDROCK_LABELS: Record<string, string> = {
        "us.anthropic.claude-3-5-haiku-20241022-v1:0":   "Claude 3.5 Haiku",
        "us.anthropic.claude-3-5-sonnet-20241022-v2:0":  "Claude 3.5 Sonnet",
        "us.anthropic.claude-3-7-sonnet-20250219-v1:0":  "Claude 3.7 Sonnet",
        "us.amazon.nova-lite-v1:0":                       "Amazon Nova Lite",
        "us.amazon.nova-pro-v1:0":                        "Amazon Nova Pro",
      };
      const VERTEX_LABELS: Record<string, string> = {
        "google/gemini-2.5-pro":        "Gemini 2.5 Pro",
        "google/gemini-2.5-flash":      "Gemini 2.5 Flash",
        "google/gemini-2.5-flash-lite": "Gemini 2.5 Flash-Lite",
      };
      const labels = llmConfig?.useAwsBedrock ? BEDROCK_LABELS : VERTEX_LABELS;
      const models = Array.from(modelSet).map((v) => ({ value: v, label: labels[v] ?? v }));
      sendJson(res, 200, { ok: true, request_id: buildRequestId(), data: { backend, models } }, allowedOrigin);
      return;
    }

    if (req.method !== "POST" || url.pathname !== "/agent/ask") {
      sendJson(res, 404, buildError(buildRequestId(), "NOT_FOUND", "Endpoint not found"), allowedOrigin);
      return;
    }

    const clientKey = resolveClientKey(req);
    const providedBearer = resolveBearerToken(req);
    const authenticated =
      (apiKey && clientKey === apiKey) ||
      (bearerToken && providedBearer === bearerToken);

    if (requireAuth && !authenticated) {
      sendJson(res, 401, buildError(buildRequestId(), "AUTH_ERROR", "Missing or invalid agent credentials"), allowedOrigin);
      return;
    }

    if (!requireAuth && (apiKey || bearerToken) && !authenticated && (clientKey || providedBearer)) {
      sendJson(res, 401, buildError(buildRequestId(), "AUTH_ERROR", "Invalid agent credentials"), allowedOrigin);
      return;
    }

    const rateKey = authenticated && (clientKey || providedBearer)
      ? `agent_auth:${clientKey || providedBearer}`
      : `agent_ip:${resolveClientIp(req)}`;

    const rate = rateLimiter.consume(rateKey, rateLimitPerMinute);
    if (!rate.allowed) {
      res.setHeader("Retry-After", String(rate.retryAfterSeconds));
      sendJson(res, 429, buildError(buildRequestId(), "RATE_LIMITED", "Rate limit exceeded"), allowedOrigin);
      return;
    }

    let body: unknown;
    try {
      body = await readJsonBody(req);
    } catch {
      sendJson(res, 400, buildError(buildRequestId(), "VALIDATION_ERROR", "Invalid JSON payload"), allowedOrigin);
      return;
    }

    const request = (body ?? {}) as AgentRequest;
    const requestId = request.request_id ?? buildRequestId();

    if (!request.study_id && !request.study_instance_uid) {
      sendJson(res, 400, buildError(requestId, "VALIDATION_ERROR", "Provide study_id or study_instance_uid"), allowedOrigin);
      return;
    }

    if (request.model !== undefined && !ALLOWED_MODEL_OVERRIDES.has(request.model)) {
      const allowed = Array.from(ALLOWED_MODEL_OVERRIDES).join(", ");
      sendJson(res, 400, buildError(requestId, "VALIDATION_ERROR", `Unknown model. Allowed: ${allowed}`), allowedOrigin);
      return;
    }

    try {
      if (!llmConfig) {
        sendJson(res, 503, buildError(requestId, "UPSTREAM_ERROR", "Agent LLM is not configured"), allowedOrigin);
        return;
      }

      const data = await runAgentWithLlm(client, request, llmConfig);
      sendJson(res, 200, { ok: true, request_id: requestId, data }, allowedOrigin);
    } catch (error) {
      if (error instanceof UpstreamHttpError) {
        const code = error.status === 404 ? "NOT_FOUND" : "UPSTREAM_ERROR";
        sendJson(res, error.status, buildError(requestId, code, error.body || "Upstream error"), allowedOrigin);
        return;
      }
      if (error instanceof DisallowedPathError) {
        sendJson(res, 500, buildError(requestId, "UPSTREAM_ERROR", error.message), allowedOrigin);
        return;
      }
      const message = error instanceof Error ? error.message : "Unexpected agent error";
      sendJson(res, 500, buildError(requestId, "UPSTREAM_ERROR", message), allowedOrigin);
    }
  });

  server.listen(port, () => {
    console.error(`AEGIS agent HTTP server listening on ${port}`);
  });
}
