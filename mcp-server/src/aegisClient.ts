export class UpstreamHttpError extends Error {
  status: number;
  body: string;

  constructor(status: number, body: string) {
    super(`Upstream request failed with status ${status}`);
    this.status = status;
    this.body = body;
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
}
