"""Generic retry queue with dead-letter and durable persistence.

Lifted from `dimse-receiver/app/ingest.py` and made generic:
  - caller passes a `process_fn` that takes a payload and returns success
  - caller decides what payload shape to use (must be JSON-serialisable)
  - exponential backoff with configurable base / multiplier / max interval
  - dedup by a caller-provided key (e.g. StudyInstanceUID)
  - dead-letter after N attempts
  - optional on-disk JSON persistence so a restart restores state
  - operator controls: process now, process all, replay dead-letter, clear

The same library backs the cloud receiver's "post to API ingest" retries and
the spoke router's "ship to cloud" retries. Each caller supplies their own
process_fn.
"""

from __future__ import annotations

import json
import logging
import threading
import time
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any, Callable

log = logging.getLogger(__name__)


@dataclass
class RetryConfig:
    """Knobs that control queue behavior."""

    base_interval_seconds: float = 15.0
    multiplier: float = 2.0
    max_interval_seconds: float = 300.0
    max_attempts: int = 5
    queue_max: int = 1000
    durable_path: str | None = None  # if set, persist state to this JSON file


@dataclass
class RetryItem:
    """One pending retry. `payload` is opaque to dimse-core."""

    key: str
    payload: dict[str, Any] = field(default_factory=dict)
    attempts: int = 0
    next_attempt_at: float = 0.0
    last_error: str = ""
    queued_at: float = 0.0
    dead_lettered_at: float = 0.0


# What the process_fn returns.
@dataclass
class RetryResult:
    ok: bool
    error: str = ""


_STATE_VERSION = 1


class RetryQueue:
    """Thread-safe bounded retry queue with dead-letter and persistence.

    `process_fn` is called with one RetryItem at a time; return
    RetryResult(ok=True) for success or RetryResult(ok=False, error="...")
    for failure.
    """

    def __init__(
        self,
        cfg: RetryConfig,
        process_fn: Callable[[RetryItem], RetryResult],
        *,
        merge_payload_fn: Callable[[dict[str, Any], dict[str, Any]], dict[str, Any]] | None = None,
    ) -> None:
        self.cfg = cfg
        self._process = process_fn
        self._merge = merge_payload_fn or (lambda existing, new: {**existing, **new})
        self._lock = threading.Lock()
        self._pending: list[RetryItem] = []
        self._dead_letter: list[RetryItem] = []
        self._metrics: dict[str, int] = {
            "queued_total": 0,
            "deduped_total": 0,
            "retried_total": 0,
            "retried_ok_total": 0,
            "dead_letter_total": 0,
            "dead_letter_deduped_total": 0,
            "replayed_total": 0,
            "cleared_dead_letter_total": 0,
            "cleared_pending_total": 0,
        }
        if self.cfg.durable_path:
            self._load_disk()

    # ── Enqueue ────────────────────────────────────────────────────────

    def enqueue(self, key: str, payload: dict[str, Any]) -> None:
        """Add or update a pending item. If `key` already exists, payload is
        merged (caller controls how — see merge_payload_fn).
        """
        now = time.time()
        with self._lock:
            existing = self._find_by_key_locked(self._pending, key)
            if existing is not None:
                existing.payload = self._merge(existing.payload, payload)
                self._metrics["deduped_total"] += 1
                self._persist_locked()
                return

            existing_dl = self._find_by_key_locked(self._dead_letter, key)
            if existing_dl is not None:
                # A study already in dead-letter is getting more data — merge
                # payload and leave it dead-lettered (operator must replay).
                existing_dl.payload = self._merge(existing_dl.payload, payload)
                self._metrics["dead_letter_deduped_total"] += 1
                self._persist_locked()
                return

            if len(self._pending) >= self.cfg.queue_max:
                # Queue full — push directly to dead-letter so we don't drop.
                self._dead_letter.append(
                    RetryItem(
                        key=key,
                        payload=payload,
                        attempts=0,
                        last_error="queue full at enqueue",
                        queued_at=now,
                        dead_lettered_at=now,
                    )
                )
                self._metrics["dead_letter_total"] += 1
                self._persist_locked()
                return

            self._pending.append(
                RetryItem(
                    key=key,
                    payload=payload,
                    attempts=0,
                    next_attempt_at=now,  # ready immediately for first attempt
                    queued_at=now,
                )
            )
            self._metrics["queued_total"] += 1
            self._persist_locked()

    # ── Worker loop ────────────────────────────────────────────────────

    def process_due(self, now: float | None = None, limit: int | None = None) -> int:
        """Process every item whose next_attempt_at has passed. Returns count attempted."""
        return self._process_internal(now=now, due_only=True, limit=limit)

    def process_all(self, limit: int | None = None) -> int:
        """Process every pending item right now, ignoring schedule."""
        return self._process_internal(now=time.time(), due_only=False, limit=limit)

    def process_one(self, key: str) -> RetryResult | None:
        """Process one pending item by key, ignoring schedule. None if not found."""
        with self._lock:
            item = self._find_by_key_locked(self._pending, key)
            if item is None:
                return None
        return self._attempt_item(item)

    def _process_internal(self, *, now: float | None, due_only: bool, limit: int | None) -> int:
        current = time.time() if now is None else now
        with self._lock:
            candidates: list[RetryItem] = [
                it for it in self._pending
                if (not due_only) or it.next_attempt_at <= current
            ]
        attempted = 0
        for item in candidates:
            if limit is not None and attempted >= limit:
                break
            self._attempt_item(item)
            attempted += 1
        return attempted

    def _attempt_item(self, item: RetryItem) -> RetryResult:
        item.attempts += 1
        with self._lock:
            self._metrics["retried_total"] += 1
        try:
            result = self._process(item)
        except Exception as e:
            result = RetryResult(ok=False, error=f"{type(e).__name__}: {e}")

        if result.ok:
            with self._lock:
                self._pending = [it for it in self._pending if it.key != item.key]
                self._metrics["retried_ok_total"] += 1
                self._persist_locked()
            return result

        # Failure: schedule next attempt or dead-letter.
        item.last_error = result.error
        if item.attempts >= self.cfg.max_attempts:
            with self._lock:
                self._pending = [it for it in self._pending if it.key != item.key]
                item.dead_lettered_at = time.time()
                self._dead_letter.append(item)
                self._metrics["dead_letter_total"] += 1
                self._persist_locked()
        else:
            delay = self._backoff_delay(item.attempts)
            item.next_attempt_at = time.time() + delay
            with self._lock:
                self._persist_locked()
        return result

    def _backoff_delay(self, attempts: int) -> float:
        base = max(0.1, float(self.cfg.base_interval_seconds))
        mult = max(1.0, float(self.cfg.multiplier))
        cap = max(base, float(self.cfg.max_interval_seconds))
        # attempts is 1-indexed at this point (already incremented).
        return min(cap, base * (mult ** (attempts - 1)))

    # ── Operator controls ─────────────────────────────────────────────

    def replay_dead_letter(self, limit: int | None = None) -> int:
        """Move up to `limit` items from dead-letter back into the pending queue."""
        moved = 0
        with self._lock:
            cap = self.cfg.queue_max - len(self._pending)
            if cap <= 0:
                return 0
            keep: list[RetryItem] = []
            for item in self._dead_letter:
                if moved < cap and (limit is None or moved < limit):
                    item.attempts = 0
                    item.next_attempt_at = time.time()
                    item.dead_lettered_at = 0.0
                    self._pending.append(item)
                    moved += 1
                    self._metrics["replayed_total"] += 1
                else:
                    keep.append(item)
            self._dead_letter = keep
            self._persist_locked()
        return moved

    def replay_dead_letter_one(self, key: str) -> bool:
        with self._lock:
            for i, item in enumerate(self._dead_letter):
                if item.key == key:
                    if len(self._pending) >= self.cfg.queue_max:
                        return False
                    item.attempts = 0
                    item.next_attempt_at = time.time()
                    item.dead_lettered_at = 0.0
                    self._dead_letter.pop(i)
                    self._pending.append(item)
                    self._metrics["replayed_total"] += 1
                    self._persist_locked()
                    return True
            return False

    def clear_dead_letter(self, limit: int | None = None) -> int:
        with self._lock:
            n = len(self._dead_letter) if limit is None else min(limit, len(self._dead_letter))
            del self._dead_letter[:n]
            self._metrics["cleared_dead_letter_total"] += n
            self._persist_locked()
            return n

    def clear_dead_letter_one(self, key: str) -> bool:
        with self._lock:
            before = len(self._dead_letter)
            self._dead_letter = [it for it in self._dead_letter if it.key != key]
            cleared = before - len(self._dead_letter)
            if cleared:
                self._metrics["cleared_dead_letter_total"] += cleared
                self._persist_locked()
            return cleared > 0

    def clear_pending(self, limit: int | None = None) -> int:
        with self._lock:
            n = len(self._pending) if limit is None else min(limit, len(self._pending))
            del self._pending[:n]
            self._metrics["cleared_pending_total"] += n
            self._persist_locked()
            return n

    def clear_pending_one(self, key: str) -> bool:
        with self._lock:
            before = len(self._pending)
            self._pending = [it for it in self._pending if it.key != key]
            cleared = before - len(self._pending)
            if cleared:
                self._metrics["cleared_pending_total"] += cleared
                self._persist_locked()
            return cleared > 0

    # ── Snapshot / diagnostics ────────────────────────────────────────

    def snapshot(self) -> dict[str, Any]:
        now = time.time()
        with self._lock:
            pending_oldest = min((it.queued_at for it in self._pending), default=0.0)
            pending_next = min((it.next_attempt_at for it in self._pending), default=0.0)
            dl_oldest = min((it.dead_lettered_at for it in self._dead_letter), default=0.0)
            return {
                "pending": len(self._pending),
                "dead_letter": len(self._dead_letter),
                "pending_oldest_age_seconds": int(now - pending_oldest) if pending_oldest else 0,
                "pending_next_attempt_at": pending_next,
                "pending_next_attempt_in_seconds": max(0, int(pending_next - now)) if pending_next else 0,
                "dead_letter_oldest_age_seconds": int(now - dl_oldest) if dl_oldest else 0,
                **self._metrics,
            }

    def details(self, limit: int = 100, key: str | None = None) -> dict[str, Any]:
        with self._lock:
            pending = [asdict(it) for it in self._pending if not key or it.key == key]
            dead = [asdict(it) for it in self._dead_letter if not key or it.key == key]
        pending = pending[:limit]
        dead = dead[:limit]
        return {
            "pending": pending,
            "dead_letter": dead,
            "pending_returned": len(pending),
            "dead_letter_returned": len(dead),
        }

    # ── Persistence ────────────────────────────────────────────────────

    @staticmethod
    def _find_by_key_locked(items: list[RetryItem], key: str) -> RetryItem | None:
        for it in items:
            if it.key == key:
                return it
        return None

    def _persist_locked(self) -> None:
        if not self.cfg.durable_path:
            return
        path = Path(self.cfg.durable_path)
        tmp = path.with_name(f".{path.name}.tmp")
        try:
            path.parent.mkdir(parents=True, exist_ok=True)
            payload = {
                "version": _STATE_VERSION,
                "updated_at": int(time.time()),
                "pending": [asdict(it) for it in self._pending],
                "dead_letter": [asdict(it) for it in self._dead_letter],
                "metrics": dict(self._metrics),
            }
            tmp.write_text(json.dumps(payload, separators=(",", ":"), sort_keys=True))
            tmp.replace(path)
        except Exception:
            log.exception("retry queue persist failed: %s", path)

    def _load_disk(self) -> None:
        path = Path(self.cfg.durable_path) if self.cfg.durable_path else None
        if path is None or not path.exists():
            return
        try:
            data = json.loads(path.read_text())
        except Exception:
            log.exception("retry queue load failed: %s", path)
            return
        with self._lock:
            self._pending = [RetryItem(**it) for it in data.get("pending", [])]
            self._dead_letter = [RetryItem(**it) for it in data.get("dead_letter", [])]
            for k, v in (data.get("metrics") or {}).items():
                if k in self._metrics:
                    self._metrics[k] = int(v)
