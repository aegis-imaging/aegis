"""Generic threshold-driven alert engine.

The dimse-receiver has retry-queue alerts (dead-letter nonzero, age
thresholds); the router has quarantine alerts; a future receiver might have
something else again. Same shape: take a "snapshot dict" of counters, declare
a list of (condition, predicate, message_fmt), emit an event when a
predicate fires (with optional cooldown so a still-failing condition doesn't
spam).

Bring your own webhook delivery — we just emit a structured event and call
your `webhook_fn(event)` if you provide one.

Example:

    eng = AlertEngine(AlertConfig(
        cooldown_seconds=300,
        max_retained=200,
        webhook_fn=post_to_slack,
    ))
    eng.add_condition(
        "dead_letter_nonzero",
        lambda snap: snap.get("dead_letter", 0) > 0,
        "Dead-letter queue is non-zero ({dead_letter} items)",
    )
    eng.add_condition(
        "pending_age",
        lambda snap: snap.get("pending_oldest_age_seconds", 0) >= 600,
        "Oldest pending retry age {pending_oldest_age_seconds}s exceeds 600s",
    )
    eng.evaluate(snapshot)
"""

from __future__ import annotations

import logging
import threading
import time
from dataclasses import dataclass, field
from typing import Any, Callable

log = logging.getLogger(__name__)


@dataclass
class AlertConfig:
    cooldown_seconds: int = 300
    max_retained: int = 200
    webhook_fn: Callable[[dict[str, Any]], None] | None = None


@dataclass
class AlertEvent:
    timestamp: int
    condition: str
    message: str
    snapshot: dict[str, Any] = field(default_factory=dict)

    def to_json(self) -> dict[str, Any]:
        return {
            "timestamp": self.timestamp,
            "condition": self.condition,
            "message": self.message,
            "snapshot": self.snapshot,
        }


@dataclass
class _Condition:
    name: str
    predicate: Callable[[dict[str, Any]], bool]
    # Exactly one of these is used. message_fn takes precedence when set —
    # use it when the message needs values that aren't in the snapshot
    # (e.g., the threshold the predicate was comparing against).
    message_fmt: str = ""  # str.format-compatible against the snapshot
    message_fn: Callable[[dict[str, Any]], str] | None = None


class AlertEngine:
    """Evaluate a list of conditions against a snapshot and emit events."""

    def __init__(self, cfg: AlertConfig) -> None:
        self.cfg = cfg
        self._lock = threading.Lock()
        self._conditions: list[_Condition] = []
        self._events: list[AlertEvent] = []
        self._last_emitted_at: dict[str, int] = {}

    def add_condition(self, name: str, predicate: Callable[[dict[str, Any]], bool], message_fmt: str) -> None:
        with self._lock:
            self._conditions.append(_Condition(name=name, predicate=predicate, message_fmt=message_fmt))

    def add_condition_fn(
        self,
        name: str,
        predicate: Callable[[dict[str, Any]], bool],
        message_fn: Callable[[dict[str, Any]], str],
    ) -> None:
        """Like add_condition, but the message comes from a callback.

        Use when the message needs values that aren't in the snapshot dict —
        for example, the threshold the predicate was comparing against — so
        `message_fmt.format(**snapshot)` can't express it.
        """
        with self._lock:
            self._conditions.append(_Condition(name=name, predicate=predicate, message_fn=message_fn))

    def clear_conditions(self) -> None:
        """Remove all registered conditions.

        Cooldown state (`_last_emitted_at`) and stored events are *not*
        cleared — re-adding a condition by the same name keeps its cooldown
        history. Use this when the set of active conditions is driven by
        live config that may change between evaluations.
        """
        with self._lock:
            self._conditions.clear()

    def evaluate(self, snapshot: dict[str, Any], now: float | None = None) -> list[AlertEvent]:
        """Run all conditions; emit events for newly-firing ones (respects cooldown).

        Returns the list of newly emitted events.
        """
        current = int(time.time() if now is None else now)
        cooldown = max(0, int(self.cfg.cooldown_seconds))
        newly_emitted: list[AlertEvent] = []

        with self._lock:
            for cond in self._conditions:
                try:
                    if not cond.predicate(snapshot):
                        continue
                except Exception:
                    log.exception("alert condition %s predicate failed", cond.name)
                    continue
                last = self._last_emitted_at.get(cond.name, 0)
                if cooldown > 0 and last > 0 and (current - last) < cooldown:
                    continue
                if cond.message_fn is not None:
                    try:
                        message = cond.message_fn(snapshot)
                    except Exception:
                        log.exception("alert message_fn for %s failed", cond.name)
                        message = cond.name
                else:
                    try:
                        message = cond.message_fmt.format(**snapshot)
                    except Exception:
                        message = cond.message_fmt
                event = AlertEvent(
                    timestamp=current,
                    condition=cond.name,
                    message=message,
                    snapshot=dict(snapshot),
                )
                self._events.append(event)
                self._last_emitted_at[cond.name] = current
                newly_emitted.append(event)

            # Trim retained events to max_retained.
            max_keep = max(1, int(self.cfg.max_retained))
            if len(self._events) > max_keep:
                overflow = len(self._events) - max_keep
                del self._events[:overflow]

        # Webhook delivery happens outside the lock so a slow webhook can't
        # block other producers.
        if self.cfg.webhook_fn is not None:
            for ev in newly_emitted:
                try:
                    self.cfg.webhook_fn(ev.to_json())
                except Exception:
                    log.exception("alert webhook failed for %s", ev.condition)

        for ev in newly_emitted:
            log.warning("alert: %s — %s", ev.condition, ev.message)

        return newly_emitted

    def recent(self, limit: int = 100, condition: str | None = None) -> list[dict[str, Any]]:
        with self._lock:
            source = self._events
            if condition:
                source = [e for e in self._events if e.condition == condition]
            return [e.to_json() for e in reversed(source[-limit:])]

    def reset(self) -> None:
        with self._lock:
            self._events.clear()
            self._last_emitted_at.clear()
