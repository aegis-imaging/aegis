"""Retry queue alerting — thin shim over dimse_core.alerts.AlertEngine.

The receiver's three alert conditions (dead-letter non-zero, pending
age threshold, dead-letter age threshold) are receiver-specific and the
message strings include the configured threshold value (which lives in
env vars, not the retry snapshot dict). That's exactly the case
`AlertEngine.add_condition_fn(...)` was added for: per-condition
`message_fn(snapshot) -> str` callbacks that can close over values
outside the snapshot.

Public API of this module is unchanged:
  - evaluate_retry_alerts(snapshot, now) -> {"active": N, "emitted": M}
  - get_alerts(limit, condition) -> {"total": N, "items": [...]}
  - reset_alerts()

`get_alerts()` continues to return events with a snapshot field
containing only the four counters the receiver historically exposed
(`pending`, `dead_letter`, `pending_oldest_age_seconds`,
`dead_letter_oldest_age_seconds`) — webhook payloads stay byte-stable
for any external consumer that depends on the shape.
"""

from __future__ import annotations

import logging
import threading
from typing import Any

import httpx
from dimse_core.alerts import AlertConfig, AlertEngine

from app import config

log = logging.getLogger(__name__)

# 4-key projection that historical webhook consumers and /ingest/retry/alerts
# clients have always seen. Kept narrow because retry snapshots have lots of
# other counters that are noise from an alerting perspective.
_SNAPSHOT_KEYS = (
    "pending",
    "dead_letter",
    "pending_oldest_age_seconds",
    "dead_letter_oldest_age_seconds",
)

_lock = threading.Lock()
_engine: AlertEngine | None = None


def _project_snapshot(snapshot: dict[str, int]) -> dict[str, int]:
    return {k: max(0, int(snapshot.get(k, 0))) for k in _SNAPSHOT_KEYS}


def _emit_webhook(event: dict[str, Any]) -> None:
    """Webhook delivery — reads `config.DIMSE_RETRY_ALERT_WEBHOOK_URL` fresh on each
    call so tests (and operators) can change it without rebuilding the engine."""
    webhook_url = config.DIMSE_RETRY_ALERT_WEBHOOK_URL.strip()
    if not webhook_url:
        return
    try:
        with httpx.Client(timeout=5.0) as client:
            resp = client.post(webhook_url, json=event)
        if resp.status_code >= 300:
            log.error("Retry alert webhook failed: HTTP %d", resp.status_code)
    except Exception as exc:
        log.error("Retry alert webhook request failed: %s", exc)


def _get_engine() -> AlertEngine:
    """Lazily build the singleton engine.

    Per-call config like cooldown and max_retained is read off `engine.cfg`
    on each `evaluate()`, so we keep that dataclass in sync with the live
    receiver config instead of rebuilding the engine (which would also
    drop cooldown state).
    """
    global _engine
    with _lock:
        if _engine is None:
            _engine = AlertEngine(
                AlertConfig(
                    cooldown_seconds=int(config.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS),
                    max_retained=max(1, int(config.DIMSE_RETRY_ALERT_MAX_EVENTS)),
                    webhook_fn=_emit_webhook,
                )
            )
        # Keep cfg in sync with live config (tests monkeypatch these).
        _engine.cfg.cooldown_seconds = max(0, int(config.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS))
        _engine.cfg.max_retained = max(1, int(config.DIMSE_RETRY_ALERT_MAX_EVENTS))
        return _engine


def _pending_age_threshold() -> int:
    configured = max(0, int(config.DIMSE_RETRY_ALERT_PENDING_AGE_SECONDS))
    if configured > 0:
        return configured
    return max(0, int(config.DIMSE_INGEST_PENDING_AGE_WARN_SECONDS))


def _dead_letter_age_threshold() -> int:
    configured = max(0, int(config.DIMSE_RETRY_ALERT_DEAD_LETTER_AGE_SECONDS))
    if configured > 0:
        return configured
    return max(0, int(config.DIMSE_DEAD_LETTER_AGE_WARN_SECONDS))


def _register_active_conditions(engine: AlertEngine) -> None:
    """Replace the engine's condition list with whichever conditions current
    config has enabled. Cooldown state survives the rebuild because the
    engine keys cooldown by condition name."""
    engine.clear_conditions()

    if config.DIMSE_RETRY_ALERT_DEAD_LETTER_NONZERO:
        engine.add_condition_fn(
            "dead_letter_nonzero",
            lambda s: int(s.get("dead_letter", 0)) > 0,
            lambda s: f"Dead-letter queue is non-zero ({int(s.get('dead_letter', 0))} items)",
        )

    pending_threshold = _pending_age_threshold()
    if pending_threshold > 0:
        engine.add_condition_fn(
            "pending_age_threshold_exceeded",
            lambda s: int(s.get("pending_oldest_age_seconds", 0)) >= pending_threshold,
            lambda s: (
                "Oldest pending retry age "
                f"{int(s.get('pending_oldest_age_seconds', 0))}s exceeds threshold "
                f"{pending_threshold}s"
            ),
        )

    dl_threshold = _dead_letter_age_threshold()
    if dl_threshold > 0:
        engine.add_condition_fn(
            "dead_letter_age_threshold_exceeded",
            lambda s: int(s.get("dead_letter_oldest_age_seconds", 0)) >= dl_threshold,
            lambda s: (
                "Oldest dead-letter age "
                f"{int(s.get('dead_letter_oldest_age_seconds', 0))}s exceeds threshold "
                f"{dl_threshold}s"
            ),
        )


def evaluate_retry_alerts(snapshot: dict[str, int], now: float | None = None) -> dict[str, int]:
    """Evaluate retry/dead-letter thresholds and emit alert events when due.

    Returns `{"active": N, "emitted": M}` where:
      - `active` is the number of currently-firing conditions (regardless
        of cooldown).
      - `emitted` is the number that produced a new event this call
        (cooldown may suppress emission of an actively-firing condition).
    """
    if not config.DIMSE_RETRY_ALERTS_ENABLED:
        return {"active": 0, "emitted": 0}

    engine = _get_engine()
    _register_active_conditions(engine)
    projected = _project_snapshot(snapshot)

    # Active count: how many of the currently-registered conditions' predicates
    # are truthy right now. Done *before* evaluate() so we report it
    # independent of cooldown.
    active = 0
    for cond in list(engine._conditions):  # noqa: SLF001 — read-only inspection
        try:
            if cond.predicate(projected):
                active += 1
        except Exception:
            log.exception("retry alert predicate %s failed", cond.name)

    emitted = engine.evaluate(projected, now=now)
    return {"active": active, "emitted": len(emitted)}


def get_alerts(limit: int = 100, condition: str | None = None) -> dict[str, Any]:
    """Return most recent retry alert events first, optionally filtered."""
    engine = _get_engine()
    items = engine.recent(limit=10_000, condition=condition)
    total = len(items)
    return {"total": total, "items": items[: max(0, int(limit))]}


def reset_alerts() -> None:
    """Reset in-memory alert state (tests only)."""
    global _engine
    with _lock:
        _engine = None
