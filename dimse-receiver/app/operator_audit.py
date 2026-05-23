"""Operator action audit log — thin shim over dimse_core.audit.AuditLog.

The receiver's public surface (`record_action`, `get_actions`, `reset_actions`)
predates dimse-core. To avoid touching every call site and risking regressions
in the receiver's mature test suite, we keep the same module-level functions
here but back them with `dimse_core.audit.AuditLog`. The shim translates the
receiver's `{timestamp, action, details}` entry shape to/from the library's
`{ts, event, **fields}` shape.

Behavior remains exactly what the existing tests expect:
  - record_action("name", k=v, ...)   appends to a bounded ring buffer
  - get_actions(limit=N, action=...)  returns newest first, optional filter
  - reset_actions()                   clears the buffer
  - DIMSE_OPERATOR_AUDIT_MAX env var  caps the in-memory retention
  - monkeypatching app.config.DIMSE_OPERATOR_AUDIT_MAX rebuilds the buffer
"""

from __future__ import annotations

import threading
from typing import Any

from dimse_core.audit import AuditLog

from app import config

_lock = threading.Lock()
_log: AuditLog | None = None
_log_max: int = 0


def _get_log() -> AuditLog:
    """Return the singleton AuditLog, rebuilding if the configured max changed.

    The rebuild-on-config-change path is needed because the existing tests
    monkeypatch app.config.DIMSE_OPERATOR_AUDIT_MAX after the singleton has
    been initialised; the test then expects the new cap to be honored.
    """
    global _log, _log_max
    with _lock:
        expected_max = max(1, int(config.DIMSE_OPERATOR_AUDIT_MAX))
        if _log is None or _log_max != expected_max:
            new_log = AuditLog(max_entries=expected_max)
            if _log is not None:
                # Preserve as many existing entries as we can. recent() is
                # newest-first; replay oldest-first so the new ring's
                # insertion order matches the old one.
                for entry in reversed(_log.recent(limit=expected_max)):
                    fields = {k: v for k, v in entry.items() if k not in ("ts", "event")}
                    new_log.record(entry["event"], **fields)
            _log = new_log
            _log_max = expected_max
        return _log


def record_action(action: str, **details: Any) -> None:
    """Record an operator action with bounded in-memory retention."""
    log = _get_log()
    log.record(action, details=details)


def get_actions(limit: int = 100, action: str | None = None) -> dict[str, Any]:
    """Return most-recent action entries, newest first; optional name filter."""
    log = _get_log()
    pulled = log.recent(limit=10_000, event=action) if action else log.recent(limit=10_000)
    items = [
        {
            "timestamp": int(e.get("ts", 0)),
            "action": str(e.get("event", "")),
            "details": e.get("details") or {},
        }
        for e in pulled
    ]
    return {"total": len(items), "items": items[: max(0, int(limit))]}


def reset_actions() -> None:
    """Clear the audit log (tests only)."""
    global _log, _log_max
    with _lock:
        _log = None
        _log_max = 0
