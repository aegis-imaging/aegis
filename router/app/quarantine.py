"""SQLite-backed local quarantine queue.

When a study fails de-id or shipping, it gets parked here. An operator at the
spoke (or via the cloud admin-dashboard once that's wired up) reviews and
either retries or discards. Nothing PHI-bearing leaves the spoke until cleared.
"""

from __future__ import annotations

import json
import logging
import os
import sqlite3
import threading
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import httpx

log = logging.getLogger(__name__)

_SCHEMA = """
CREATE TABLE IF NOT EXISTS quarantine (
    study_uid       TEXT PRIMARY KEY,
    reason          TEXT NOT NULL,
    failed_stage    TEXT NOT NULL,
    error           TEXT NOT NULL,
    detail          TEXT,
    file_count      INTEGER NOT NULL DEFAULT 0,
    quarantined_at  REAL NOT NULL,
    last_attempt_at REAL,
    attempts        INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS retry_queue (
    study_uid       TEXT PRIMARY KEY,
    next_attempt_at REAL NOT NULL,
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',
    queued_at       REAL NOT NULL
);
"""


@dataclass
class QuarantineEntry:
    study_uid: str
    reason: str
    failed_stage: str
    error: str
    detail: dict[str, Any]
    file_count: int
    quarantined_at: float
    last_attempt_at: float | None
    attempts: int

    def to_json(self) -> dict[str, Any]:
        return {
            "study_uid": self.study_uid,
            "reason": self.reason,
            "failed_stage": self.failed_stage,
            "error": self.error,
            "detail": self.detail,
            "file_count": self.file_count,
            "quarantined_at": self.quarantined_at,
            "last_attempt_at": self.last_attempt_at,
            "attempts": self.attempts,
        }


class QuarantineStore:
    def __init__(self, db_path: str) -> None:
        self._db_path = db_path
        self._lock = threading.Lock()
        Path(db_path).parent.mkdir(parents=True, exist_ok=True)
        with self._conn() as c:
            c.executescript(_SCHEMA)

    def _conn(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self._db_path, timeout=30.0)
        conn.execute("PRAGMA journal_mode=WAL")
        return conn

    def quarantine(
        self,
        study_uid: str,
        reason: str,
        failed_stage: str,
        error: str,
        detail: dict[str, Any] | None = None,
        file_count: int = 0,
    ) -> None:
        now = time.time()
        detail_json = json.dumps(detail or {}, sort_keys=True, default=str)
        with self._lock, self._conn() as c:
            c.execute(
                """
                INSERT INTO quarantine
                    (study_uid, reason, failed_stage, error, detail, file_count, quarantined_at, attempts)
                VALUES (?, ?, ?, ?, ?, ?, ?, 0)
                ON CONFLICT(study_uid) DO UPDATE SET
                    reason = excluded.reason,
                    failed_stage = excluded.failed_stage,
                    error = excluded.error,
                    detail = excluded.detail,
                    file_count = excluded.file_count,
                    quarantined_at = CASE
                        WHEN quarantine.quarantined_at IS NULL THEN excluded.quarantined_at
                        ELSE quarantine.quarantined_at
                    END
                """,
                (study_uid, reason, failed_stage, error, detail_json, file_count, now),
            )
            c.commit()
        log.warning(
            "study %s quarantined at stage=%s reason=%s error=%s",
            study_uid,
            failed_stage,
            reason,
            error,
        )

    def list(self, limit: int = 100) -> list[QuarantineEntry]:
        with self._lock, self._conn() as c:
            rows = c.execute(
                """SELECT study_uid, reason, failed_stage, error, detail, file_count,
                          quarantined_at, last_attempt_at, attempts
                   FROM quarantine
                   ORDER BY quarantined_at DESC
                   LIMIT ?""",
                (limit,),
            ).fetchall()
        return [self._row_to_entry(r) for r in rows]

    def get(self, study_uid: str) -> QuarantineEntry | None:
        with self._lock, self._conn() as c:
            row = c.execute(
                """SELECT study_uid, reason, failed_stage, error, detail, file_count,
                          quarantined_at, last_attempt_at, attempts
                   FROM quarantine WHERE study_uid = ?""",
                (study_uid,),
            ).fetchone()
        return self._row_to_entry(row) if row else None

    def clear(self, study_uid: str) -> bool:
        with self._lock, self._conn() as c:
            cur = c.execute("DELETE FROM quarantine WHERE study_uid = ?", (study_uid,))
            c.commit()
            return cur.rowcount > 0

    def mark_attempted(self, study_uid: str) -> None:
        with self._lock, self._conn() as c:
            c.execute(
                """UPDATE quarantine
                   SET last_attempt_at = ?, attempts = attempts + 1
                   WHERE study_uid = ?""",
                (time.time(), study_uid),
            )
            c.commit()

    @staticmethod
    def _row_to_entry(row: tuple) -> QuarantineEntry:
        return QuarantineEntry(
            study_uid=row[0],
            reason=row[1],
            failed_stage=row[2],
            error=row[3],
            detail=json.loads(row[4]) if row[4] else {},
            file_count=row[5] or 0,
            quarantined_at=row[6],
            last_attempt_at=row[7],
            attempts=row[8] or 0,
        )


def notify_cloud(alert_url: str, entry: QuarantineEntry, site_id: str) -> None:
    """Best-effort POST to a cloud control-plane webhook. Never raises — the
    notification is a nice-to-have; the operational truth is the local DB."""
    if not alert_url:
        return
    try:
        with httpx.Client(timeout=10.0) as client:
            client.post(
                alert_url,
                json={
                    "type": "router.quarantine",
                    "site_id": site_id,
                    "study_uid": entry.study_uid,
                    "reason": entry.reason,
                    "failed_stage": entry.failed_stage,
                    "error": entry.error,
                    "quarantined_at": entry.quarantined_at,
                },
            )
    except Exception:
        log.exception("quarantine alert webhook failed; alert dropped")


def default_db_path(quarantine_dir: str) -> str:
    return os.path.join(quarantine_dir, "quarantine.sqlite")
