export class UpstreamHttpError extends Error {
  status: number;
  body: string;

  constructor(status: number, body: string) {
    super(`Upstream request failed with status ${status}`);
    this.status = status;
    this.body = body;
  }
}

export class DisallowedPathError extends Error {
  method: "GET" | "POST" | "DELETE" | "PUT" | "PATCH";
  path: string;

  constructor(method: "GET" | "POST" | "DELETE" | "PUT" | "PATCH", path: string) {
    super(`Outbound ${method} path is not allowed: ${path}`);
    this.method = method;
    this.path = path;
  }
}

const allowedGetPathPatterns = [
  /^\/healthz(?:\?.*)?$/,
  /^\/api\/stats(?:\?.*)?$/,
  /^\/api\/stats\/breakdown(?:\?.*)?$/,
  /^\/api\/stats\/timeline(?:\?.*)?$/,
  /^\/api\/storage\/stats(?:\?.*)?$/,
  /^\/api\/studies(?:\?.*)?$/,
  /^\/api\/studies\/stuck(?:\?.*)?$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/diagnostics$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/audit$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/routing-log$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/shares$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/series$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/labels$/,
  /^\/api\/studies\/[0-9.]+\/dicom-tags$/,
  /^\/api\/dimse\/retry\/details(?:\?.*)?$/,
  /^\/api\/dimse\/retry\/summary(?:\?.*)?$/,
  /^\/api\/study-uid\/[0-9.]+$/,
  /^\/api\/audit(?:\?.*)?$/,
  /^\/api\/audit\/actors(?:\?.*)?$/,
  /^\/api\/shares(?:\?.*)?$/,
  /^\/api\/shares\/[0-9a-fA-F-]{36}\/downloads$/,
  /^\/api\/projects(?:\?.*)?$/,
  /^\/api\/institutions(?:\?.*)?$/,
  /^\/api\/routing-rules(?:\?.*)?$/,
  /^\/api\/destinations(?:\?.*)?$/,
  /^\/api\/subjects(?:\?.*)?$/,
  /^\/api\/export-analytics$/,
  /^\/api\/webhook-subscriptions(?:\?.*)?$/,
  /^\/api\/webhook-subscriptions\/[0-9a-fA-F-]{36}\/deliveries(?:\?.*)?$/
] as const;

const allowedPostPathPatterns = [
  /^\/api\/studies\/[0-9.]+\/classify$/,
  /^\/api\/studies\/[0-9.]+\/bids-convert$/,
  /^\/api\/studies\/[0-9.]+\/trigger-export$/,
  /^\/api\/studies\/[0-9.]+\/qc-check$/,
  /^\/api\/studies\/[0-9.]+\/protocol-check$/,
  /^\/api\/studies\/[0-9.]+\/phi-scan$/,
  /^\/api\/studies\/[0-9.]+\/trigger-deface$/,
  /^\/api\/dimse\/retry\/process\/[0-9.]+$/,
  /^\/api\/dimse\/retry\/replay\/[0-9.]+$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/approve$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/reject$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/share$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/reset-pipeline-step$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/labels$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/notes$/,
  /^\/api\/projects\/[0-9a-fA-F-]{36}\/export-batch$/,
  /^\/api\/routing-rules\/evaluate\/[0-9a-fA-F-]{36}$/
] as const;

const allowedDeletePathPatterns = [
  /^\/api\/shares\/[0-9a-fA-F-]{36}$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/labels\/[0-9a-fA-F-]{36}$/
] as const;

const allowedPutPathPatterns = [
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/project$/,
  /^\/api\/studies\/[0-9a-fA-F-]{36}\/subject$/
] as const;

const allowedPatchPathPatterns = [
  /^\/api\/shares\/[0-9a-fA-F-]{36}\/extend$/
] as const;

function assertAllowedPath(method: "GET" | "POST" | "DELETE" | "PUT" | "PATCH", path: string): void {
  if (!path.startsWith("/")) {
    throw new DisallowedPathError(method, path);
  }

  const patterns =
    method === "GET" ? allowedGetPathPatterns :
    method === "DELETE" ? allowedDeletePathPatterns :
    method === "PUT" ? allowedPutPathPatterns :
    method === "PATCH" ? allowedPatchPathPatterns :
    allowedPostPathPatterns;
  const allowed = (patterns as readonly RegExp[]).some((pattern) => pattern.test(path));
  if (!allowed) {
    throw new DisallowedPathError(method, path);
  }
}

export class AegisApiClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.token = token;
  }

  async get(path: string): Promise<unknown> {
    assertAllowedPath("GET", path);

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${this.token}`
      }
    });

    const body = await response.text();
    if (!response.ok) {
      throw new UpstreamHttpError(response.status, body);
    }

    if (!body) {
      return null;
    }

    try {
      return JSON.parse(body);
    } catch {
      return { raw: body };
    }
  }

  async post(path: string, payload?: unknown): Promise<unknown> {
    assertAllowedPath("POST", path);

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${this.token}`,
        ...(payload !== undefined ? { "Content-Type": "application/json" } : {})
      },
      ...(payload !== undefined ? { body: JSON.stringify(payload) } : {})
    });

    const body = await response.text();
    if (!response.ok) {
      throw new UpstreamHttpError(response.status, body);
    }

    if (!body) {
      return null;
    }

    try {
      return JSON.parse(body);
    } catch {
      return { raw: body };
    }
  }

  async put(path: string, payload: unknown): Promise<unknown> {
    assertAllowedPath("PUT", path);

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: "PUT",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${this.token}`,
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    });

    const body = await response.text();
    if (!response.ok) {
      throw new UpstreamHttpError(response.status, body);
    }

    if (!body) {
      return null;
    }

    try {
      return JSON.parse(body);
    } catch {
      return { raw: body };
    }
  }

  async patch(path: string, payload: unknown): Promise<unknown> {
    assertAllowedPath("PATCH", path);

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: "PATCH",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${this.token}`,
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    });

    const body = await response.text();
    if (!response.ok) {
      throw new UpstreamHttpError(response.status, body);
    }

    if (!body) {
      return null;
    }

    try {
      return JSON.parse(body);
    } catch {
      return { raw: body };
    }
  }

  async delete(path: string): Promise<unknown> {
    assertAllowedPath("DELETE", path);

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: "DELETE",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${this.token}`
      }
    });

    const body = await response.text();
    if (!response.ok) {
      throw new UpstreamHttpError(response.status, body);
    }

    if (!body) {
      return null;
    }

    try {
      return JSON.parse(body);
    } catch {
      return { raw: body };
    }
  }
}
