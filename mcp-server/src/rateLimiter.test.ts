import assert from "node:assert/strict";
import test from "node:test";
import { InMemoryRateLimiter } from "./rateLimiter.js";

test("InMemoryRateLimiter: first call is always allowed", () => {
  const rl = new InMemoryRateLimiter();
  const result = rl.consume("key", 10, 0);
  assert.equal(result.allowed, true);
  assert.equal(result.retryAfterSeconds, 0);
});

test("InMemoryRateLimiter: allows up to the limit within one window", () => {
  const rl = new InMemoryRateLimiter();
  const now = 0;
  for (let i = 0; i < 3; i++) {
    const result = rl.consume("key", 3, now);
    assert.equal(result.allowed, true, `call ${i + 1} should be allowed`);
  }
  const blocked = rl.consume("key", 3, now);
  assert.equal(blocked.allowed, false);
  assert.ok(blocked.retryAfterSeconds > 0);
});

test("InMemoryRateLimiter: rejects immediately when limit=0", () => {
  const rl = new InMemoryRateLimiter();
  const result = rl.consume("key", 0, 0);
  assert.equal(result.allowed, false);
  assert.equal(result.retryAfterSeconds, 60);
});

test("InMemoryRateLimiter: resets counter after window rolls over", () => {
  const rl = new InMemoryRateLimiter();
  // exhaust limit at t=0
  rl.consume("key", 1, 0);
  assert.equal(rl.consume("key", 1, 0).allowed, false);
  // advance 60 s — new window
  const result = rl.consume("key", 1, 60_000);
  assert.equal(result.allowed, true);
});

test("InMemoryRateLimiter: separate keys are independent", () => {
  const rl = new InMemoryRateLimiter();
  const now = 0;
  rl.consume("a", 1, now);
  rl.consume("a", 1, now); // exhausted
  const resultB = rl.consume("b", 1, now);
  assert.equal(resultB.allowed, true);
});

test("InMemoryRateLimiter: retryAfterSeconds is positive and bounded within the window", () => {
  const rl = new InMemoryRateLimiter();
  // window starts at 0, now = 30_000 ms into the window
  const now = 30_000;
  rl.consume("key", 1, now); // allowed
  const blocked = rl.consume("key", 1, now);
  assert.equal(blocked.allowed, false);
  // remaining time in window is 60_000 - 30_000 = 30 s
  assert.ok(blocked.retryAfterSeconds >= 1);
  assert.ok(blocked.retryAfterSeconds <= 60);
});

test("InMemoryRateLimiter: limit=1 allows exactly one call per window", () => {
  const rl = new InMemoryRateLimiter();
  assert.equal(rl.consume("k", 1, 0).allowed, true);
  assert.equal(rl.consume("k", 1, 0).allowed, false);
  assert.equal(rl.consume("k", 1, 60_000).allowed, true);
  assert.equal(rl.consume("k", 1, 60_000).allowed, false);
});
