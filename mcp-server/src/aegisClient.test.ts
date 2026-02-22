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

test("AegisApiClient allows DIMSE retry summary and details paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const summary = await client.get("/api/dimse/retry/summary");
  const details = await client.get("/api/dimse/retry/details?limit=10&study_instance_uid=1.2.3");

  assert.deepEqual(summary, {});
  assert.deepEqual(details, {});
});

test("AegisApiClient allows audit log path with and without query params", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const plain = await client.get("/api/audit");
  const filtered = await client.get("/api/audit?action=study&actor=admin%40example.com&limit=50");

  assert.deepEqual(plain, {});
  assert.deepEqual(filtered, {});
});

test("AegisApiClient allows study lookup by UID path", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const result = await client.get("/api/studies/by-uid/1.2.840.10008.5.1");
  assert.deepEqual(result, {});
});

test("AegisApiClient allows global shares path with and without query params", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const all = await client.get("/api/shares");
  const active = await client.get("/api/shares?status=active&limit=50");

  assert.deepEqual(all, {});
  assert.deepEqual(active, {});
});

test.after(() => {
  globalThis.fetch = originalFetch;
});
