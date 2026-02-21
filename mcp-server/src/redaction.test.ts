import assert from "node:assert/strict";
import test from "node:test";
import { redactToolArgs } from "./redaction.js";

test("redactToolArgs redacts sensitive fields recursively", () => {
  const input = {
    request_id: "req_12345678",
    study_uid: "1.2.840.10008.1.2.1",
    reason: "contains sensitive details",
    requested_by: "operator@example.org",
    nested: {
      api_token: "abc123",
      uploader_email: "uploader@example.org",
      note_text: "private"
    },
    tags: [{ token: "hidden" }, { safe: true }]
  };

  const result = redactToolArgs(input);

  assert.equal(result.request_id, "req_12345678");
  assert.equal(result.study_uid, "1.2.840.10008.1.2.1");
  assert.equal(result.reason, "[REDACTED]");
  assert.equal(result.requested_by, "[REDACTED]");
  assert.deepEqual(result.nested, {
    api_token: "[REDACTED]",
    uploader_email: "[REDACTED]",
    note_text: "[REDACTED]"
  });
  assert.deepEqual(result.tags, [{ token: "[REDACTED]" }, { safe: true }]);
});

test("redactToolArgs handles circular objects", () => {
  const circular: Record<string, unknown> = { request_id: "req_12345678" };
  circular.self = circular;

  const result = redactToolArgs(circular);

  assert.equal(result.request_id, "req_12345678");
  assert.equal(result.self, "[CIRCULAR]");
});
