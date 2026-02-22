const MAX_ENTRIES = 2048;

type Entry<T> = { expiresAtMs: number; value: T };

export class InMemoryIdempotencyCache<T> {
  private readonly entries = new Map<string, Entry<T>>();

  get(key: string, now = Date.now()): T | null {
    const entry = this.entries.get(key);
    if (!entry) {
      return null;
    }
    if (entry.expiresAtMs <= now) {
      this.entries.delete(key);
      return null;
    }
    return entry.value;
  }

  set(key: string, value: T, ttlSeconds: number, now = Date.now()): void {
    if (ttlSeconds <= 0) {
      return;
    }

    this.entries.set(key, { value, expiresAtMs: now + ttlSeconds * 1000 });
    this.prune(now);
  }

  private prune(now: number): void {
    if (this.entries.size <= MAX_ENTRIES) {
      return;
    }

    for (const [key, entry] of this.entries.entries()) {
      if (entry.expiresAtMs <= now) {
        this.entries.delete(key);
      }
    }
  }
}
