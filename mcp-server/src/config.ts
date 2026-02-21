export type McpConfig = {
  aegisApiBaseUrl: string;
  aegisApiToken: string;
  mcpMode: "readonly" | "operator";
  enableWriteTools: boolean;
  readRateLimitPerMinute: number;
  writeRateLimitPerMinute: number;
  writeIdempotencyTtlSeconds: number;
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
    writeIdempotencyTtlSeconds: getPositiveIntEnv("MCP_WRITE_IDEMPOTENCY_TTL_SECONDS", 900)
  };
}
