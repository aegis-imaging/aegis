export type McpConfig = {
  aegisApiBaseUrl: string;
  aegisApiToken: string;
  mcpMode: "readonly" | "operator";
  enableWriteTools: boolean;
};

function getRequiredEnv(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

export function loadConfig(): McpConfig {
  const mode = (process.env.MCP_MODE ?? "readonly").trim().toLowerCase();
  const mcpMode: McpConfig["mcpMode"] = mode === "operator" ? "operator" : "readonly";

  return {
    aegisApiBaseUrl: getRequiredEnv("AEGIS_API_BASE_URL"),
    aegisApiToken: getRequiredEnv("AEGIS_API_TOKEN"),
    mcpMode,
    enableWriteTools: process.env.MCP_ENABLE_WRITE_TOOLS === "true"
  };
}
