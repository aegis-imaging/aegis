import { readFileSync } from "node:fs";
import { AegisApiClient } from "./aegisClient.js";

type AgentData = {
  summary: string;
  evidence: Array<{ field: string; value: string | number | boolean | null; timestamp?: string | null }>;
  diagnostics: {
    terminal: boolean;
    stuck: boolean;
    blockers: string[];
    recommended_actions: string[];
  };
  timeline: Array<{ event: string; timestamp: string }>;
  next_steps: string[];
};

type AgentRequest = {
  request_id?: string;
  question?: string;
  study_id?: string;
  study_instance_uid?: string;
  include_next_steps?: boolean;
};

type LlmConfig = {
  baseUrl: string;
  apiKey: string;
  model: string;
  temperature: number;
  maxTokens: number;
};

type ToolHandler = (args: Record<string, unknown>) => Promise<unknown>;

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
- Return JSON matching the AEGIS Agent response schema file: aegis-agent.response-schema.json.
- Do not add extra fields outside the schema.`;

function loadPrompt(): string {
  try {
    return readFileSync(new URL("../agent/aegis-agent.prompt.txt", import.meta.url), "utf-8").trim();
  } catch {
    return DEFAULT_PROMPT;
  }
}

const AGENT_PROMPT = loadPrompt();

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
  const parsed = JSON.parse(cleaned) as Partial<AgentData>;

  if (!parsed || typeof parsed.summary !== "string") {
    throw new Error("Invalid agent response: missing summary");
  }
  if (!Array.isArray(parsed.evidence) || !parsed.diagnostics || !Array.isArray(parsed.timeline) || !Array.isArray(parsed.next_steps)) {
    throw new Error("Invalid agent response: missing sections");
  }

  return {
    summary: parsed.summary,
    evidence: parsed.evidence,
    diagnostics: {
      terminal: Boolean(parsed.diagnostics.terminal),
      stuck: Boolean(parsed.diagnostics.stuck),
      blockers: Array.isArray(parsed.diagnostics.blockers) ? parsed.diagnostics.blockers.map(String) : [],
      recommended_actions: Array.isArray(parsed.diagnostics.recommended_actions)
        ? parsed.diagnostics.recommended_actions.map(String)
        : []
    },
    timeline: parsed.timeline.map((item) => ({
      event: String((item as { event?: string }).event ?? ""),
      timestamp: String((item as { timestamp?: string }).timestamp ?? "")
    })),
    next_steps: parsed.next_steps.map(String)
  };
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

async function callLlm(
  config: LlmConfig,
  messages: LlmMessage[],
  tools: unknown
): Promise<{ content?: string; tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }> }> {
  const url = `${config.baseUrl.replace(/\/$/, "")}/chat/completions`;
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${config.apiKey}`
    },
    body: JSON.stringify({
      model: config.model,
      messages,
      tools,
      tool_choice: "auto",
      temperature: config.temperature,
      max_tokens: config.maxTokens
    })
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`LLM request failed: ${response.status} ${body}`);
  }

  const payload = (await response.json()) as {
    choices?: Array<{ message?: { content?: string; tool_calls?: Array<{ id: string; function: { name: string; arguments: string } }> } }>;
  };

  const message = payload.choices?.[0]?.message;
  return { content: message?.content, tool_calls: message?.tool_calls };
}

export async function runAgentWithLlm(
  client: AegisApiClient,
  request: AgentRequest,
  llmConfig: LlmConfig
): Promise<AgentData> {
  const toolHandlers = buildToolHandlers(client);
  const tools = buildToolSchema();

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

  for (let i = 0; i < 4; i += 1) {
    const result = await callLlm(llmConfig, messages, tools);

    if (result.tool_calls && result.tool_calls.length > 0) {
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

    if (!result.content) {
      throw new Error("LLM response missing content");
    }

    return parseAgentData(result.content);
  }

  throw new Error("LLM did not return a final response after tool calls");
}

export function buildLlmConfig(env: {
  baseUrl: string | undefined;
  apiKey: string | undefined;
  model: string | undefined;
  temperature: number;
  maxTokens: number;
}): LlmConfig | null {
  if (!env.baseUrl || !env.apiKey) {
    return null;
  }

  return {
    baseUrl: env.baseUrl,
    apiKey: env.apiKey,
    model: env.model || "gpt-4.1-mini",
    temperature: env.temperature,
    maxTokens: env.maxTokens
  };
}
