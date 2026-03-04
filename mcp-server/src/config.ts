export type McpConfig = {
  aegisApiBaseUrl: string;
  aegisApiToken: string;
  mcpMode: "readonly" | "operator";
  enableWriteTools: boolean;
  readRateLimitPerMinute: number;
  writeRateLimitPerMinute: number;
  writeIdempotencyTtlSeconds: number;
  agentHttpPort?: number;
  agentAllowedOrigin: string;
  agentApiKey?: string;
  agentBearerToken?: string;
  agentRequireAuth: boolean;
  agentRateLimitPerMinute: number;
  agentLlmBaseUrl?: string;
  agentLlmApiKey?: string;
  agentLlmModel: string;
  agentLlmTemperature: number;
  agentLlmMaxTokens: number;
  agentLlmUseGcpAuth: boolean;
  agentLlmGcpProject?: string;
  agentLlmUseAwsBedrock: boolean;
  agentLlmAwsRegion?: string;
};

function getRequiredEnv(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function getPositiveIntEnv(name: string, fallback: number): number {
  const raw = process.env[name]?.trim();
  if (!raw) {
    return fallback;
  }
  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return fallback;
  }
  return Math.trunc(parsed);
}

function getOptionalIntEnv(name: string): number | undefined {
  const raw = process.env[name]?.trim();
  if (!raw) {
    return undefined;
  }
  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined;
  }
  return Math.trunc(parsed);
}

function getFloatEnv(name: string, fallback: number): number {
  const raw = process.env[name]?.trim();
  if (!raw) {
    return fallback;
  }
  const parsed = Number(raw);
  if (!Number.isFinite(parsed)) {
    return fallback;
  }
  return parsed;
}

export function loadConfig(): McpConfig {
  const mode = (process.env.MCP_MODE ?? "readonly").trim().toLowerCase();
  const mcpMode: McpConfig["mcpMode"] = mode === "operator" ? "operator" : "readonly";

  return {
    aegisApiBaseUrl: getRequiredEnv("AEGIS_API_BASE_URL"),
    aegisApiToken: getRequiredEnv("AEGIS_API_TOKEN"),
    mcpMode,
    enableWriteTools: process.env.MCP_ENABLE_WRITE_TOOLS === "true",
    readRateLimitPerMinute: getPositiveIntEnv("MCP_READ_RATE_LIMIT_PER_MINUTE", 240),
    writeRateLimitPerMinute: getPositiveIntEnv("MCP_WRITE_RATE_LIMIT_PER_MINUTE", 60),
    writeIdempotencyTtlSeconds: getPositiveIntEnv("MCP_WRITE_IDEMPOTENCY_TTL_SECONDS", 900),
    agentHttpPort: getOptionalIntEnv("MCP_AGENT_HTTP_PORT"),
    agentAllowedOrigin: process.env.MCP_AGENT_ALLOWED_ORIGIN?.trim() || "*",
    agentApiKey: process.env.MCP_AGENT_API_KEY?.trim() || undefined,
    agentBearerToken: process.env.MCP_AGENT_BEARER_TOKEN?.trim() || undefined,
    agentRequireAuth: process.env.MCP_AGENT_REQUIRE_AUTH === "true",
    agentRateLimitPerMinute: getPositiveIntEnv("MCP_AGENT_RATE_LIMIT_PER_MINUTE", 120),
    agentLlmBaseUrl: process.env.MCP_AGENT_LLM_BASE_URL?.trim() || undefined,
    agentLlmApiKey: process.env.MCP_AGENT_LLM_API_KEY?.trim() || undefined,
    agentLlmModel: process.env.MCP_AGENT_LLM_MODEL?.trim() || "gpt-4.1-mini",
    agentLlmTemperature: getFloatEnv("MCP_AGENT_LLM_TEMPERATURE", 0.2),
    agentLlmMaxTokens: getPositiveIntEnv("MCP_AGENT_LLM_MAX_TOKENS", 4096),
    agentLlmUseGcpAuth: process.env.MCP_AGENT_LLM_USE_GCP_AUTH === "true",
    agentLlmGcpProject: process.env.MCP_AGENT_LLM_GCP_PROJECT?.trim() || undefined,
    agentLlmUseAwsBedrock: process.env.MCP_AGENT_LLM_USE_AWS_BEDROCK === "true",
    agentLlmAwsRegion: process.env.MCP_AGENT_LLM_AWS_REGION?.trim() || undefined
  };
}
