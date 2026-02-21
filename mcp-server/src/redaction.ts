const REDACTED = "[REDACTED]";
const CIRCULAR = "[CIRCULAR]";

const sensitiveKeyPattern = /password|secret|token|authorization|api[_-]?key|email|reason|note|requested_by|credential/i;

function shouldRedactKey(key: string): boolean {
  return sensitiveKeyPattern.test(key);
}

function redactValue(value: unknown, key: string | null, seen: WeakSet<object>): unknown {
  if (key && shouldRedactKey(key)) {
    return REDACTED;
  }

  if (Array.isArray(value)) {
    return value.map((item) => redactValue(item, null, seen));
  }

  if (value && typeof value === "object") {
    if (seen.has(value as object)) {
      return CIRCULAR;
    }
    seen.add(value as object);

    const output: Record<string, unknown> = {};
    for (const [childKey, childValue] of Object.entries(value as Record<string, unknown>)) {
      output[childKey] = redactValue(childValue, childKey, seen);
    }
    return output;
  }

  return value;
}

export function redactToolArgs(args: Record<string, unknown>): Record<string, unknown> {
  const seen = new WeakSet<object>();
  const redacted = redactValue(args, null, seen);
  if (!redacted || typeof redacted !== "object" || Array.isArray(redacted)) {
    return {};
  }
  return redacted as Record<string, unknown>;
}
