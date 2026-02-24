import { readFileSync } from "node:fs";
import { z } from "zod";
import { BedrockRuntimeClient, ConverseCommand } from "@aws-sdk/client-bedrock-runtime";
import { AegisApiClient } from "./aegisClient.js";

const agentDataSchema = z.object({
  summary: z.string(),
  evidence: z.array(
    z.object({
      field: z.string(),
      value: z.union([z.string(), z.number(), z.boolean(), z.null()]),
      timestamp: z.string().nullable().optional()
    })
  ),
  diagnostics: z.object({
    terminal: z.boolean(),
    stuck: z.boolean(),
    blockers: z.array(z.string()),
    recommended_actions: z.array(z.string())
  }),
  timeline: z.array(z.object({ event: z.string(), timestamp: z.string() })),
  next_steps: z.array(z.string())
});

type AgentData = z.infer<typeof agentDataSchema>;

export type AgentRequest = {
  request_id?: string;
  question?: string;
  study_id?: string;
  study_instance_uid?: string;
  include_next_steps?: boolean;
  model?: string;
};

type LlmConfig = {
  baseUrl: string;
  apiKey: string;
  model: string;
  temperature: number;
  maxTokens: number;
  useGcpAuth: boolean;
  gcpProject?: string;
  useAwsBedrock?: boolean;
  awsRegion?: string;
};

type ToolHandler = (args: Record<string, unknown>) => Promise<unknown>;

// Allowed Gemini model overrides — must be models available on Vertex AI OpenAI-compatible endpoint.
export const ALLOWED_BEDROCK_MODELS = new Set([
  // Cross-region inference profiles (recommended for production — multi-AZ resilience)
  "us.anthropic.claude-3-5-haiku-20241022-v1:0",
  "us.anthropic.claude-3-5-sonnet-20241022-v2:0",
  "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
  "us.amazon.nova-lite-v1:0",
  "us.amazon.nova-pro-v1:0",
]);

const ALLOWED_MODEL_OVERRIDES = new Set([
  // Gemini 3 series (latest)
  "google/gemini-3.1-pro-preview",
  "google/gemini-3-pro-preview",
  "google/gemini-3-flash-preview",
  // Gemini 2.5 series
  "google/gemini-2.5-pro",
  "google/gemini-2.5-flash",
  "google/gemini-2.5-flash-lite",
  // Gemini 2.0 series
  "google/gemini-2.0-flash-001",
  "google/gemini-2.0-flash-lite-001",
  // Gemini 1.5 series (legacy)
  "google/gemini-1.5-flash-001",
  "google/gemini-1.5-pro-001",
]);

function resolveModel(requested: string | undefined, config: LlmConfig): string {
  if (requested && ALLOWED_MODEL_OVERRIDES.has(requested)) {
    return requested;
  }
  return config.model;
}

const DEFAULT_PROMPT = `AEGIS Agent Prompt (Read-only)

You are the AEGIS Agent, a read-only operations assistant for the Anonymization & Exchange Gateway for Imaging Studies (AEGIS).

Mission:
- Answer questions about study status, pipeline diagnostics, routing outcomes, and audit timelines.
- Use only the provided tool results; never infer or guess.

Safety and Compliance:
- Never expose PHI or unredacted identifiers beyond study UUIDs and DICOM StudyInstanceUIDs.
- If requested data is unavailable, say so and ask for the exact study ID or StudyInstanceUID.
- Provide evidence and timestamps for every claim.

Behavior Rules:
- Prefer study_id if provided; otherwise use study_instance_uid.
- If diagnostics indicate stuck=true, lead with blockers and recommended actions.
- Only include next steps when the caller explicitly asks for them.

Output:
- Return a single raw JSON object — no markdown, no code fences, no prose.
- The object must have exactly these top-level keys: summary, evidence, diagnostics, timeline, next_steps.
- Do not wrap the response in an envelope (no "ok", "request_id", or "data" wrapper).
- Schema:
  {
    "summary": "<string>",
    "evidence": [{"field":"<str>","value":"<str|num|bool|null>","timestamp":"<str|null>"}],
    "diagnostics": {"terminal":<bool>,"stuck":<bool>,"blockers":["<str>"],"recommended_actions":["<str>"]},
    "timeline": [{"event":"<str>","timestamp":"<str>"}],
    "next_steps": ["<str>"]
  }
`;

function loadPrompt(): string {
  try {
    return readFileSync(new URL("../agent/aegis-agent.prompt.txt", import.meta.url), "utf-8").trim();
  } catch {
    return DEFAULT_PROMPT;
  }
}

const AGENT_PROMPT = loadPrompt();

const FINAL_JSON_REQUEST =
  'Based on the tool results above, produce ONLY the JSON response object. ' +
  'No prose, no code fences, no explanation — just the raw JSON object with exactly these top-level keys: ' +
  'summary, evidence, diagnostics, timeline, next_steps.';

function stripCodeFence(input: string): string {
  const trimmed = input.trim();
  if (trimmed.startsWith("```")) {
    const withoutFence = trimmed.replace(/^```[a-zA-Z0-9]*\n/, "").replace(/```$/, "");
    return withoutFence.trim();
  }
  return trimmed;
}

function parseAgentData(content: string): AgentData {
  const cleaned = stripCodeFence(content);

  let raw: unknown;
  try {
    raw = JSON.parse(cleaned);
  } catch {
    throw new Error(`Agent returned non-JSON content: ${cleaned.slice(0, 200)}`);
  }

  // Unwrap full-envelope responses: {"ok":true,"data":{summary,...}} → {summary,...}
  const candidate =
    raw &&
    typeof raw === "object" &&
    "data" in (raw as object) &&
    typeof (raw as Record<string, unknown>).data === "object" &&
    !("summary" in (raw as object))
      ? (raw as Record<string, unknown>).data
      : raw;

  const result = agentDataSchema.safeParse(candidate);
  if (!result.success) {
    const issues = result.error.issues.map((i) => `${i.path.join(".")}: ${i.message}`).join("; ");
    throw new Error(`Invalid agent response: ${issues}`);
  }
  return result.data;
}

function buildToolHandlers(client: AegisApiClient): Record<string, ToolHandler> {
  return {
    list_studies: async (args) => {
      const params = new URLSearchParams();
      for (const [key, value] of Object.entries(args)) {
        if (value === undefined || value === null) continue;
        params.set(key, String(value));
      }
      const qs = params.toString();
      return client.get(`/api/studies${qs ? `?${qs}` : ""}`);
    },
    get_study_detail: async (args) => {
      const studyId = String(args.study_id ?? "");
      return client.get(`/api/studies/${encodeURIComponent(studyId)}`);
    },
    get_study_by_uid: async (args) => {
      const studyUid = String(args.study_instance_uid ?? "");
      return client.get(`/api/study-uid/${encodeURIComponent(studyUid)}`);
    },
    get_study_diagnostics: async (args) => {
      const studyId = String(args.study_id ?? "");
      return client.get(`/api/studies/${encodeURIComponent(studyId)}/diagnostics`);
    },
    get_study_audit: async (args) => {
      const studyId = String(args.study_id ?? "");
      return client.get(`/api/studies/${encodeURIComponent(studyId)}/audit`);
    },
    get_study_routing_log: async (args) => {
      const studyId = String(args.study_id ?? "");
      return client.get(`/api/studies/${encodeURIComponent(studyId)}/routing-log`);
    },
    get_system_health: async () => client.get(`/healthz`)
  };
}

function buildToolSchema() {
  return [
    {
      type: "function",
      function: {
        name: "list_studies",
        description: "List studies with optional filters for triage.",
        parameters: {
          type: "object",
          properties: {
            limit: { type: "integer" },
            offset: { type: "integer" },
            project_id: { type: "string" },
            status: { type: "string" },
            modality: { type: "string" },
            body_part: { type: "string" },
            source: { type: "string" },
            search: { type: "string" }
          },
          additionalProperties: false
        }
      }
    },
    {
      type: "function",
      function: {
        name: "get_study_detail",
        description: "Get study detail by database UUID.",
        parameters: {
          type: "object",
          required: ["study_id"],
          properties: { study_id: { type: "string" } },
          additionalProperties: false
        }
      }
    },
    {
      type: "function",
      function: {
        name: "get_study_by_uid",
        description: "Get study detail by DICOM StudyInstanceUID.",
        parameters: {
          type: "object",
          required: ["study_instance_uid"],
          properties: { study_instance_uid: { type: "string" } },
          additionalProperties: false
        }
      }
    },
    {
      type: "function",
      function: {
        name: "get_study_diagnostics",
        description: "Get per-study diagnostics summary.",
        parameters: {
          type: "object",
          required: ["study_id"],
          properties: { study_id: { type: "string" } },
          additionalProperties: false
        }
      }
    },
    {
      type: "function",
      function: {
        name: "get_study_audit",
        description: "Get audit entries for a study UUID.",
        parameters: {
          type: "object",
          required: ["study_id"],
          properties: { study_id: { type: "string" } },
          additionalProperties: false
        }
      }
    },
    {
      type: "function",
      function: {
        name: "get_study_routing_log",
        description: "Get routing log for a study UUID.",
        parameters: {
          type: "object",
          required: ["study_id"],
          properties: { study_id: { type: "string" } },
          additionalProperties: false
        }
      }
    },
    {
      type: "function",
      function: {
        name: "get_system_health",
        description: "Get system health snapshot.",
        parameters: { type: "object", properties: {}, additionalProperties: false }
      }
    }
  ];
}

type LlmMessage = {
  role: "system" | "user" | "assistant" | "tool";
  content: string;
  tool_call_id?: string;
  tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }>;
};

async function fetchGcpAccessToken(): Promise<string> {
  const url = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token";
  const response = await fetch(url, { headers: { "Metadata-Flavor": "Google" } });
  if (!response.ok) {
    throw new Error(`Failed to fetch GCP access token: ${response.status}`);
  }
  const data = (await response.json()) as { access_token: string };
  return data.access_token;
}

async function callLlm(
  config: LlmConfig,
  messages: LlmMessage[],
  tools: unknown,
  forceJsonOutput = false
): Promise<{ content?: string; tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }> }> {
  const url = `${config.baseUrl.replace(/\/$/, "")}/chat/completions`;
  const authToken = config.useGcpAuth ? await fetchGcpAccessToken() : config.apiKey;

  const body: Record<string, unknown> = {
    model: config.model,
    messages,
    temperature: config.temperature,
    max_tokens: config.maxTokens
  };

  if (tools) {
    body.tools = tools;
    body.tool_choice = "auto";
  }

  if (forceJsonOutput) {
    body.response_format = { type: "json_object" };
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${authToken}`
    },
    body: JSON.stringify(body)
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`LLM request failed: ${response.status} ${text}`);
  }

  const payload = (await response.json()) as {
    choices?: Array<{ message?: { content?: string; tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }> } }>;
  };

  const message = payload.choices?.[0]?.message;
  return { content: message?.content ?? undefined, tool_calls: message?.tool_calls };
}

// ── AWS Bedrock Converse API backend ─────────────────────────────────────────

async function callBedrockLlm(
  config: LlmConfig,
  messages: LlmMessage[],
  tools: unknown,
  _forceJsonOutput = false
): Promise<{ content?: string; tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }> }> {
  const client = new BedrockRuntimeClient({ region: config.awsRegion || "us-east-1" });

  // Extract system messages (Bedrock takes them separately)
  const systemMessages = messages.filter((m) => m.role === "system");
  const conversationMessages = messages.filter((m) => m.role !== "system");

  // Convert OpenAI message format → Bedrock Converse format
  // Use unknown[] and cast to avoid SDK discriminated-union strictness.
  type BMsg = { role: "user" | "assistant"; content: unknown[] };
  const bedrockMessages: BMsg[] = [];
  for (const msg of conversationMessages) {
    if (msg.role === "assistant") {
      const content: unknown[] = [];
      if (msg.content) content.push({ text: msg.content });
      if (msg.tool_calls) {
        for (const tc of msg.tool_calls) {
          let input: unknown = {};
          try { input = tc.function.arguments ? JSON.parse(tc.function.arguments) : {}; } catch { /* empty */ }
          content.push({ toolUse: { toolUseId: tc.id, name: tc.function.name, input } });
        }
      }
      bedrockMessages.push({ role: "assistant", content });
    } else if (msg.role === "tool") {
      // Bedrock requires tool results as user messages
      bedrockMessages.push({
        role: "user",
        content: [{
          toolResult: {
            toolUseId: msg.tool_call_id ?? "",
            content: [{ text: msg.content }]
          }
        }]
      });
    } else {
      // user role
      bedrockMessages.push({ role: "user", content: [{ text: msg.content }] });
    }
  }

  // Translate OpenAI tool schema → Bedrock toolSpec format
  type OpenAiTool = { type: string; function: { name: string; description?: string; parameters?: unknown } };
  const bedrockTools = Array.isArray(tools) && tools.length > 0
    ? (tools as OpenAiTool[]).map((t) => ({
        toolSpec: {
          name: t.function.name,
          description: t.function.description ?? "",
          inputSchema: { json: (t.function.parameters ?? {}) as unknown }
        }
      }))
    : undefined;

  const response = await client.send(new ConverseCommand({
    modelId: config.model,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    messages: bedrockMessages as any,
    system: systemMessages.length > 0
      ? systemMessages.map((m) => ({ text: m.content }))
      : undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    toolConfig: bedrockTools ? { tools: bedrockTools as any } : undefined,
    inferenceConfig: {
      temperature: config.temperature,
      maxTokens: config.maxTokens
    }
  }));

  const outputContent = response.output?.message?.content ?? [];
  let textContent: string | undefined;
  const toolCalls: Array<{ id: string; function: { name: string; arguments: string } }> = [];

  for (const block of outputContent) {
    if ("text" in block && block.text) {
      textContent = block.text;
    } else if ("toolUse" in block && block.toolUse) {
      toolCalls.push({
        id: block.toolUse.toolUseId ?? "",
        function: {
          name: block.toolUse.name ?? "",
          arguments: JSON.stringify(block.toolUse.input ?? {})
        }
      });
    }
  }

  return {
    content: textContent,
    tool_calls: toolCalls.length > 0 ? toolCalls : undefined
  };
}

// Dispatch to the correct LLM backend based on config
async function dispatchLlm(
  config: LlmConfig,
  messages: LlmMessage[],
  tools: unknown,
  forceJsonOutput = false
): Promise<{ content?: string; tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }> }> {
  if (config.useAwsBedrock) {
    return callBedrockLlm(config, messages, tools, forceJsonOutput);
  }
  return callLlm(config, messages, tools, forceJsonOutput);
}

export async function runAgentWithLlm(
  client: AegisApiClient,
  request: AgentRequest,
  llmConfig: LlmConfig
): Promise<AgentData> {
  const toolHandlers = buildToolHandlers(client);
  const tools = buildToolSchema();

  const effectiveModel = resolveModel(request.model, llmConfig);
  const effectiveConfig = effectiveModel !== llmConfig.model ? { ...llmConfig, model: effectiveModel } : llmConfig;

  const userContext = {
    question: request.question ?? "",
    study_id: request.study_id ?? null,
    study_instance_uid: request.study_instance_uid ?? null,
    include_next_steps: request.include_next_steps ?? false
  };

  const messages: LlmMessage[] = [
    { role: "system", content: AGENT_PROMPT },
    { role: "user", content: JSON.stringify(userContext) }
  ];

  let finalContent: string | null = null;
  let madeToolCalls = false;

  // Phase 1: tool-call loop — gather data via tools (up to 6 rounds).
  for (let i = 0; i < 6; i += 1) {
    const result = await dispatchLlm(effectiveConfig, messages, tools);

    if (result.tool_calls && result.tool_calls.length > 0) {
      madeToolCalls = true;
      messages.push({
        role: "assistant",
        content: result.content ?? "",
        tool_calls: result.tool_calls
      });

      for (const toolCall of result.tool_calls) {
        const handler = toolHandlers[toolCall.function.name];
        if (!handler) {
          messages.push({
            role: "tool",
            content: JSON.stringify({ error: `Unknown tool ${toolCall.function.name}` }),
            tool_call_id: toolCall.id
          });
          continue;
        }

        let parsedArgs: Record<string, unknown> = {};
        try {
          parsedArgs = toolCall.function.arguments ? JSON.parse(toolCall.function.arguments) : {};
        } catch {
          parsedArgs = {};
        }

        const data = await handler(parsedArgs);
        messages.push({
          role: "tool",
          content: JSON.stringify(data),
          tool_call_id: toolCall.id
        });
      }
      continue;
    }

    // No tool calls — model produced a text response.
    if (result.content) {
      finalContent = result.content;
    }
    break;
  }

  // Phase 2: if we have no valid final content yet, explicitly request the JSON answer.
  // This handles models (like Gemini) that exhaust tool rounds without producing structured output.
  if (!finalContent) {
    messages.push({ role: "user", content: FINAL_JSON_REQUEST });
    const finalResult = await dispatchLlm(effectiveConfig, messages, null, true);
    finalContent = finalResult.content ?? null;
  } else if (madeToolCalls) {
    // We got content after tool calls — try to parse it.
    // If it fails, retry with an explicit JSON request and response_format.
    try {
      return parseAgentData(finalContent);
    } catch {
      messages.push({ role: "user", content: FINAL_JSON_REQUEST });
      const retryResult = await dispatchLlm(effectiveConfig, messages, null, true);
      finalContent = retryResult.content ?? null;
    }
  }

  if (!finalContent) {
    throw new Error("LLM did not return a final response after tool calls");
  }

  return parseAgentData(finalContent);
}

export function buildLlmConfig(env: {
  baseUrl: string | undefined;
  apiKey: string | undefined;
  model: string | undefined;
  temperature: number;
  maxTokens: number;
  useGcpAuth: boolean;
  gcpProject?: string;
  useAwsBedrock?: boolean;
  awsRegion?: string;
}): LlmConfig | null {
  if (env.useAwsBedrock) {
    return {
      baseUrl: "",
      apiKey: "",
      useGcpAuth: false,
      useAwsBedrock: true,
      awsRegion: env.awsRegion || "us-east-1",
      model: env.model || "us.anthropic.claude-3-5-haiku-20241022-v1:0",
      temperature: env.temperature,
      maxTokens: env.maxTokens
    };
  }

  if (env.useGcpAuth) {
    if (!env.gcpProject) {
      return null;
    }
    const vertexBaseUrl = `https://us-central1-aiplatform.googleapis.com/v1beta1/projects/${env.gcpProject}/locations/us-central1/endpoints/openapi`;
    return {
      baseUrl: env.baseUrl || vertexBaseUrl,
      apiKey: "",
      model: env.model || "google/gemini-2.0-flash-001",
      temperature: env.temperature,
      maxTokens: env.maxTokens,
      useGcpAuth: true,
      gcpProject: env.gcpProject
    };
  }

  if (!env.baseUrl || !env.apiKey) {
    return null;
  }

  return {
    baseUrl: env.baseUrl,
    apiKey: env.apiKey,
    model: env.model || "gpt-4.1-mini",
    temperature: env.temperature,
    maxTokens: env.maxTokens,
    useGcpAuth: false
  };
}

export { ALLOWED_MODEL_OVERRIDES };
