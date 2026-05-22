"""Ring-buffer audit log of pipeline events and operator actions.

Lives in-memory and gets persisted to JSONL on disk so a restart doesn't lose
recent history. Bounded so it can't fill the disk on a chatty PACS.
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
    def __init__(self, path: str, max_entries: int = 2000) -> None:
        self._path = path
        self._max = max_entries
        self._lock = threading.Lock()
        self._buf: deque[dict[str, Any]] = deque(maxlen=max_entries)
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        self._load()

    def record(self, event: str, **fields: Any) -> None:
        entry = {"ts": time.time(), "event": event, **fields}
        with self._lock:
            self._buf.append(entry)
            self._append_disk(entry)

    def recent(self, limit: int = 100, event: str = "") -> list[dict[str, Any]]:
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

    def _append_disk(self, entry: dict[str, Any]) -> None:
        try:
            with open(self._path, "a", encoding="utf-8") as f:
                f.write(json.dumps(entry, default=str) + "\n")
        except OSError:
            log.warning("could not persist audit entry to %s", self._path)

    def _load(self) -> None:
        if not os.path.exists(self._path):
            return
        try:
            with open(self._path, "r", encoding="utf-8") as f:
                lines = f.readlines()
        except OSError:
            return
        for line in lines[-self._max :]:
            try:
                self._buf.append(json.loads(line))
            except json.JSONDecodeError:
                continue
