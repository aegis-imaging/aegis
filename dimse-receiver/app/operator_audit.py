"""In-memory operator action audit log for retry-control endpoints."""

from __future__ import annotations

import threading
import time
from typing import Any

from app import config

_lock = threading.Lock()
_actions: list[dict[str, Any]] = []


def record_action(action: str, **details: Any) -> None:
    """Record an operator action with bounded in-memory retention."""
    event = {
        "timestamp": int(time.time()),
        "action": action,
        "details": details,
    }
    with _lock:
        _actions.append(event)
        if len(_actions) > config.DIMSE_OPERATOR_AUDIT_MAX:
            overflow = len(_actions) - config.DIMSE_OPERATOR_AUDIT_MAX
            del _actions[:overflow]


def get_actions(limit: int = 100, action: str | None = None) -> dict[str, Any]:
    """Return most recent action entries first, optionally filtered by action."""
    with _lock:
        source = _actions
        if action:
            source = [event for event in _actions if event.get("action") == action]
        total = len(source)
        items = list(reversed(source[-limit:]))
    return {"total": total, "items": items}


def reset_actions() -> None:
    """Clear action log (tests only)."""
    with _lock:
        _actions.clear()
