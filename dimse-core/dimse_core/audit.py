"""Bounded in-memory operator/event audit log with optional JSONL persistence.

Identical in spirit to what `dimse-receiver/app/operator_audit.py` does, but
generalised:
  - takes a configurable max size at construction (not from module globals)
  - optionally appends each entry to a JSONL file so a restart doesn't lose
    recent history (the router needs this; the receiver doesn't, but it's
    cheap to leave enabled)
  - supports arbitrary event-type filtering on read
"""

from __future__ import annotations

import json
import logging
import os
import threading
import time
from collections import deque
from pathlib import Path
from typing import Any, Iterable

log = logging.getLogger(__name__)


class AuditLog:
    """Thread-safe ring buffer of {ts, event, ...fields} entries.

    Parameters
    ----------
    max_entries : maximum entries to keep in memory (bounded ring)
    persist_path: optional path to a JSONL file we append to and rehydrate from
    """

    def __init__(self, max_entries: int = 500, persist_path: str | None = None) -> None:
        self._max = max(1, int(max_entries))
        self._lock = threading.Lock()
        self._buf: deque[dict[str, Any]] = deque(maxlen=self._max)
        self._path = persist_path
        if self._path:
            Path(self._path).parent.mkdir(parents=True, exist_ok=True)
            self._load_from_disk()

    def record(self, event: str, **fields: Any) -> None:
        """Record an event. `fields` are JSON-encoded into the entry."""
        entry = {"ts": time.time(), "event": event, **fields}
        with self._lock:
            self._buf.append(entry)
            if self._path:
                self._append_disk_locked(entry)

    def recent(self, limit: int = 100, event: str | None = None) -> list[dict[str, Any]]:
        """Return up to `limit` most recent entries, newest first.

        Optionally filter to a single event type.
        """
        with self._lock:
            items: Iterable[dict[str, Any]] = reversed(self._buf)
            if event:
                items = (e for e in items if e.get("event") == event)
            out: list[dict[str, Any]] = []
            for e in items:
                out.append(e)
                if len(out) >= limit:
                    break
            return out

    def size(self) -> int:
        with self._lock:
            return len(self._buf)

    def clear(self) -> None:
        with self._lock:
            self._buf.clear()

    # ── disk persistence ─────────────────────────────────────────────────

    def _append_disk_locked(self, entry: dict[str, Any]) -> None:
        try:
            with open(self._path, "a", encoding="utf-8") as f:
                f.write(json.dumps(entry, default=str) + "\n")
        except OSError:
            log.warning("audit append failed: %s", self._path)

    def _load_from_disk(self) -> None:
        if not self._path or not os.path.exists(self._path):
            return
        try:
            with open(self._path, "r", encoding="utf-8") as f:
                lines = f.readlines()
        except OSError:
            return
        # Keep only the most recent _max lines so re-hydration is bounded.
        for line in lines[-self._max :]:
            try:
                self._buf.append(json.loads(line))
            except json.JSONDecodeError:
                continue
