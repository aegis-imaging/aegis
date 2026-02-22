const WINDOW_MS = 60_000;
const MAX_COUNTERS = 512;

type Window = { windowStart: number; count: number };

export type RateLimitResult = { allowed: boolean; retryAfterSeconds: number };

export class InMemoryRateLimiter {
  private readonly counters = new Map<string, Window>();

  consume(key: string, limit: number, now = Date.now()): RateLimitResult {
    if (limit <= 0) {
      return { allowed: false, retryAfterSeconds: Math.ceil(WINDOW_MS / 1000) };
    }

    const windowStart = now - (now % WINDOW_MS);
    const current = this.counters.get(key);

    if (!current || current.windowStart !== windowStart) {
      this.counters.set(key, { windowStart, count: 1 });
      this.prune(windowStart);
      return { allowed: true, retryAfterSeconds: 0 };
    }

    if (current.count >= limit) {
      const retryAfterSeconds = Math.max(1, Math.ceil((windowStart + WINDOW_MS - now) / 1000));
      return { allowed: false, retryAfterSeconds };
    }

    current.count += 1;
    return { allowed: true, retryAfterSeconds: 0 };
  }

  private prune(activeWindowStart: number): void {
    if (this.counters.size <= MAX_COUNTERS) {
      return;
    }

    for (const [key, value] of this.counters.entries()) {
      if (value.windowStart !== activeWindowStart) {
        this.counters.delete(key);
      }
    }
  }
}
