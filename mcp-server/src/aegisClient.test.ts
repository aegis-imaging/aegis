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

  const result = await client.get("/api/study-uid/1.2.840.10008.5.1");
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

test("AegisApiClient allows share download history path", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const result = await client.get(`/api/shares/${uuid}/downloads`);
  assert.deepEqual(result, {});
});

test("AegisApiClient allows create share and re-evaluate routing POST paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const share = await client.post(`/api/studies/${uuid}/share`, { recipient_email: "test@example.com" });
  const reEval = await client.post(`/api/routing-rules/evaluate/${uuid}`);

  assert.deepEqual(share, {});
  assert.deepEqual(reEval, {});
});

test("AegisApiClient allows approve and reject study POST paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const approved = await client.post(`/api/studies/${uuid}/approve`);
  const rejected = await client.post(`/api/studies/${uuid}/reject`);

  assert.deepEqual(approved, {});
  assert.deepEqual(rejected, {});
});

test("AegisApiClient rejects disallowed POST study workflow paths", async () => {
  let fetchCalls = 0;
  globalThis.fetch = (async () => {
    fetchCalls += 1;
    return new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  // approve/reject require UUID-format study_id, not DICOM UID format in path
  await assert.rejects(client.post("/api/studies/1.2.3/approve"), DisallowedPathError);
  await assert.rejects(client.post("/api/studies/1.2.3/reject"), DisallowedPathError);
  assert.equal(fetchCalls, 0);
});

test("AegisApiClient allows DELETE share revoke path", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const result = await client.delete(`/api/shares/${uuid}`);
  assert.deepEqual(result, {});
});

test("AegisApiClient rejects disallowed DELETE paths", async () => {
  let fetchCalls = 0;
  globalThis.fetch = (async () => {
    fetchCalls += 1;
    return new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  await assert.rejects(client.delete("/api/admin/users/some-id"), DisallowedPathError);
  await assert.rejects(client.delete("/api/shares"), DisallowedPathError); // no UUID
  assert.equal(fetchCalls, 0);
});

test("AegisApiClient allows pipeline stats path", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const result = await client.get("/api/stats");
  assert.deepEqual(result, {});
});

test("AegisApiClient allows pipeline stats with project_id query param", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const projectId = "550e8400-e29b-41d4-a716-446655440000";

  const result = await client.get(`/api/stats?project_id=${projectId}`);
  assert.deepEqual(result, {});
});

test("AegisApiClient allows breakdown, storage stats, and audit actors GET paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const projectId = "550e8400-e29b-41d4-a716-446655440000";

  const breakdown = await client.get("/api/stats/breakdown");
  const breakdownFiltered = await client.get(`/api/stats/breakdown?project_id=${projectId}`);
  const storage = await client.get("/api/storage/stats");
  const storageFiltered = await client.get(`/api/storage/stats?project_id=${projectId}`);
  const actors = await client.get("/api/audit/actors");
  const actorsLimited = await client.get("/api/audit/actors?limit=10");

  assert.deepEqual(breakdown, {});
  assert.deepEqual(breakdownFiltered, {});
  assert.deepEqual(storage, {});
  assert.deepEqual(storageFiltered, {});
  assert.deepEqual(actors, {});
  assert.deepEqual(actorsLimited, {});
});

test("AegisApiClient allows stuck studies GET path with query params", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  const plain = await client.get("/api/studies/stuck");
  const filtered = await client.get("/api/studies/stuck?minutes=120&project_id=550e8400-e29b-41d4-a716-446655440000");

  assert.deepEqual(plain, {});
  assert.deepEqual(filtered, {});
});

test("AegisApiClient allows study series, labels, and subjects GET paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const series = await client.get(`/api/studies/${uuid}/series`);
  const labels = await client.get(`/api/studies/${uuid}/labels`);
  const subjects = await client.get("/api/subjects");
  const subjectsFiltered = await client.get(`/api/subjects?project_id=${uuid}`);

  assert.deepEqual(series, {});
  assert.deepEqual(labels, {});
  assert.deepEqual(subjects, {});
  assert.deepEqual(subjectsFiltered, {});
});

test("AegisApiClient allows export analytics, webhook subscriptions, and timeline GET paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const analytics = await client.get("/api/export-analytics");
  const webhooks = await client.get("/api/webhook-subscriptions");
  const deliveries = await client.get(`/api/webhook-subscriptions/${uuid}/deliveries`);
  const timeline = await client.get("/api/stats/timeline");
  const timelineFiltered = await client.get(`/api/stats/timeline?days=14&project_id=${uuid}`);

  assert.deepEqual(analytics, {});
  assert.deepEqual(webhooks, {});
  assert.deepEqual(deliveries, {});
  assert.deepEqual(timeline, {});
  assert.deepEqual(timelineFiltered, {});
});

test("AegisApiClient allows reset-pipeline-step and add-label POST paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const reset = await client.post(`/api/studies/${uuid}/reset-pipeline-step`, { step: "deface" });
  const addLabel = await client.post(`/api/studies/${uuid}/labels`, { label: "cohort-A" });

  assert.deepEqual(reset, {});
  assert.deepEqual(addLabel, {});
});

test("AegisApiClient allows DELETE study label path", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const studyId = "550e8400-e29b-41d4-a716-446655440000";
  const labelId = "660e8400-e29b-41d4-a716-446655440001";

  const result = await client.delete(`/api/studies/${studyId}/labels/${labelId}`);
  assert.deepEqual(result, {});
});

test("AegisApiClient allows PUT study reassign and subject paths", async () => {
  globalThis.fetch = (async () => new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");
  const uuid = "550e8400-e29b-41d4-a716-446655440000";

  const reassign = await client.put(`/api/studies/${uuid}/project`, { project_id: uuid });
  const subject = await client.put(`/api/studies/${uuid}/subject`, { subject_id: "SUB-001" });

  assert.deepEqual(reassign, {});
  assert.deepEqual(subject, {});
});

test("AegisApiClient rejects disallowed PUT paths", async () => {
  let fetchCalls = 0;
  globalThis.fetch = (async () => {
    fetchCalls += 1;
    return new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;

  const client = new AegisApiClient("http://example.internal", "token");

  await assert.rejects(client.put("/api/admin/users/some-id", {}), DisallowedPathError);
  await assert.rejects(client.put("/api/studies/1.2.3/project", {}), DisallowedPathError); // DICOM UID, not UUID
  assert.equal(fetchCalls, 0);
});

test.after(() => {
  globalThis.fetch = originalFetch;
});
