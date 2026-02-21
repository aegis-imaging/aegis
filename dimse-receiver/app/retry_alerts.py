"""Retry queue alerting for DIMSE ingest operations."""

from __future__ import annotations

import logging
import threading
import time
from typing import Any

import httpx

from app import config

log = logging.getLogger(__name__)

_lock = threading.Lock()
_alerts: list[dict[str, Any]] = []
_last_emitted_at: dict[str, int] = {}


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


def _active_conditions(snapshot: dict[str, int]) -> list[dict[str, Any]]:
    conditions: list[dict[str, Any]] = []

    dead_letter_count = max(0, int(snapshot.get("dead_letter", 0)))
    if config.DIMSE_RETRY_ALERT_DEAD_LETTER_NONZERO and dead_letter_count > 0:
        conditions.append(
            {
                "condition": "dead_letter_nonzero",
                "message": f"Dead-letter queue is non-zero ({dead_letter_count} items)",
            }
        )

    pending_age_threshold = _pending_age_threshold()
    pending_oldest_age = max(0, int(snapshot.get("pending_oldest_age_seconds", 0)))
    if pending_age_threshold > 0 and pending_oldest_age >= pending_age_threshold:
        conditions.append(
            {
                "condition": "pending_age_threshold_exceeded",
                "message": (
                    "Oldest pending retry age "
                    f"{pending_oldest_age}s exceeds threshold {pending_age_threshold}s"
                ),
            }
        )

    dead_letter_age_threshold = _dead_letter_age_threshold()
    dead_letter_oldest_age = max(0, int(snapshot.get("dead_letter_oldest_age_seconds", 0)))
    if dead_letter_age_threshold > 0 and dead_letter_oldest_age >= dead_letter_age_threshold:
        conditions.append(
            {
                "condition": "dead_letter_age_threshold_exceeded",
                "message": (
                    "Oldest dead-letter age "
                    f"{dead_letter_oldest_age}s exceeds threshold {dead_letter_age_threshold}s"
                ),
            }
        )

    return conditions


def _emit_webhook(event: dict[str, Any]) -> None:
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


def evaluate_retry_alerts(snapshot: dict[str, int], now: float | None = None) -> dict[str, int]:
    """Evaluate retry/dead-letter thresholds and emit alert events when due."""
    if not config.DIMSE_RETRY_ALERTS_ENABLED:
        return {"active": 0, "emitted": 0}

    current = int(time.time() if now is None else now)
    cooldown = max(0, int(config.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS))
    active = _active_conditions(snapshot)
    emitted_events: list[dict[str, Any]] = []

    with _lock:
        for condition in active:
            key = condition["condition"]
            last_sent = _last_emitted_at.get(key, 0)
            if cooldown > 0 and last_sent > 0 and (current - last_sent) < cooldown:
                continue

            event = {
                "timestamp": current,
                "condition": key,
                "message": condition["message"],
                "snapshot": {
                    "pending": max(0, int(snapshot.get("pending", 0))),
                    "dead_letter": max(0, int(snapshot.get("dead_letter", 0))),
                    "pending_oldest_age_seconds": max(0, int(snapshot.get("pending_oldest_age_seconds", 0))),
                    "dead_letter_oldest_age_seconds": max(
                        0, int(snapshot.get("dead_letter_oldest_age_seconds", 0))
                    ),
                },
            }
            _alerts.append(event)
            _last_emitted_at[key] = current
            emitted_events.append(event)

        max_events = max(1, int(config.DIMSE_RETRY_ALERT_MAX_EVENTS))
        if len(_alerts) > max_events:
            overflow = len(_alerts) - max_events
            del _alerts[:overflow]

    for event in emitted_events:
        log.warning("DIMSE retry alert: %s", event["message"])
        _emit_webhook(event)

    return {"active": len(active), "emitted": len(emitted_events)}


def get_alerts(limit: int = 100, condition: str | None = None) -> dict[str, Any]:
    """Return most recent retry alert events first, optionally filtered."""
    with _lock:
        source = _alerts
        if condition:
            source = [event for event in _alerts if event.get("condition") == condition]
        total = len(source)
        items = list(reversed(source[-limit:]))
    return {"total": total, "items": items}


def reset_alerts() -> None:
    """Reset in-memory alert state (tests only)."""
    with _lock:
        _alerts.clear()
        _last_emitted_at.clear()

