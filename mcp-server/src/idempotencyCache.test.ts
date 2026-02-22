import assert from "node:assert/strict";
import test from "node:test";
import { InMemoryIdempotencyCache } from "./idempotencyCache.js";

test("InMemoryIdempotencyCache: returns null for unknown key", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  assert.equal(cache.get("missing", 0), null);
});

test("InMemoryIdempotencyCache: stores and retrieves a value", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  cache.set("k", "hello", 60, 0);
  assert.equal(cache.get("k", 1), "hello");
});

test("InMemoryIdempotencyCache: returns null after TTL expires", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  cache.set("k", "value", 10, 0); // expires at t=10_000 ms
  assert.equal(cache.get("k", 9_999), "value"); // 1 ms before expiry — still valid
  assert.equal(cache.get("k", 10_000), null); // exactly at expiry — expired
});

test("InMemoryIdempotencyCache: ignores set with ttlSeconds=0", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  cache.set("k", "value", 0, 0);
  assert.equal(cache.get("k", 0), null);
});

test("InMemoryIdempotencyCache: ignores set with negative ttl", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  cache.set("k", "value", -1, 0);
  assert.equal(cache.get("k", 0), null);
});

test("InMemoryIdempotencyCache: second set overwrites earlier value", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  cache.set("k", "first", 60, 0);
  cache.set("k", "second", 60, 0);
  assert.equal(cache.get("k", 1), "second");
});

test("InMemoryIdempotencyCache: independent keys do not interfere", () => {
  const cache = new InMemoryIdempotencyCache<number>();
  cache.set("a", 1, 60, 0);
  cache.set("b", 2, 60, 0);
  assert.equal(cache.get("a", 1), 1);
  assert.equal(cache.get("b", 1), 2);
});

test("InMemoryIdempotencyCache: works with object values", () => {
  const cache = new InMemoryIdempotencyCache<{ ok: boolean }>();
  const obj = { ok: true };
  cache.set("k", obj, 60, 0);
  const result = cache.get("k", 1);
  assert.deepEqual(result, { ok: true });
});

test("InMemoryIdempotencyCache: get does not affect other keys after expiry cleanup", () => {
  const cache = new InMemoryIdempotencyCache<string>();
  cache.set("a", "alive", 60, 0);
  cache.set("b", "expired", 5, 0); // expires at t=5_000 ms
  cache.get("b", 10_000); // trigger expiry of "b"
  assert.equal(cache.get("a", 10_000), "alive");
});
