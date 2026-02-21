import assert from "node:assert/strict";
import test from "node:test";
import { AegisApiClient, DisallowedPathError } from "./aegisClient.js";

const originalFetch = globalThis.fetch;

test("AegisApiClient rejects disallowed outbound path before fetch", async () => {
  let fetchCalls = 0;
  globalThis.fetch = (async () => {
    fetchCalls += 1;
    return new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  await assert.rejects(client.get("/api/admin/users"), DisallowedPathError);
  await assert.rejects(client.post("/api/studies/1.2.3/delete"), DisallowedPathError);
  assert.equal(fetchCalls, 0);
});

test("AegisApiClient allows approved paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const health = await client.get("/healthz");
  const classify = await client.post("/api/studies/1.2.840.10008.1/classify");

  assert.deepEqual(health, {});
  assert.deepEqual(classify, {});
});

test.after(() => {
  globalThis.fetch = originalFetch;
});
