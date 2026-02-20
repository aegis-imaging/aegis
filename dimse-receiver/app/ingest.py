"""HTTP client and retry queue for Go API ingest calls."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
import threading
import time

import httpx

from app import config

log = logging.getLogger(__name__)


@dataclass
class StudyAccumulator:
    """Tracks metadata for a single study during a DICOM association."""

    study_instance_uid: str = ""
    modality: str = ""
    body_part: str = ""
    study_description: str = ""
    series_uids: set = field(default_factory=set)
    file_count: int = 0
    calling_ae_title: str = ""


@dataclass
class QueuedIngest:
    """In-memory ingest retry record."""

    acc: StudyAccumulator
    attempts: int = 1
    next_attempt_at: float = 0.0
    last_error: str = ""


_retry_queue: list[QueuedIngest] = []
_dead_letter: list[QueuedIngest] = []
_retry_lock = threading.Lock()
_metrics = {
    "queued_total": 0,
    "retried_total": 0,
    "retried_ok_total": 0,
    "dead_letter_total": 0,
}


def _enqueue_retry(acc: StudyAccumulator, reason: str) -> bool:
    """Queue a study for retry if queue capacity permits."""
    with _retry_lock:
        if len(_retry_queue) >= config.DIMSE_INGEST_QUEUE_MAX:
            log.error(
                "Retry queue full (%d), dropping study %s",
                config.DIMSE_INGEST_QUEUE_MAX,
                acc.study_instance_uid,
            )
            _metrics["dead_letter_total"] += 1
            _dead_letter.append(
                QueuedIngest(
                    acc=acc,
                    attempts=1,
                    next_attempt_at=time.time(),
                    last_error=f"queue_full: {reason}",
                )
            )
            return False

        _retry_queue.append(
            QueuedIngest(
                acc=acc,
                attempts=1,
                next_attempt_at=time.time() + config.DIMSE_INGEST_RETRY_INTERVAL,
                last_error=reason,
            )
        )
        _metrics["queued_total"] += 1
    return True


def trigger_ingest(acc: StudyAccumulator) -> bool:
    """POST study metadata to the Go API ingest endpoint.

    Returns True on success (HTTP 2xx), False otherwise.
    """
    url = f"{config.API_URL}/api/ingest"
    payload = {
        "project_slug": config.DIMSE_PROJECT_SLUG,
        "study_metadata": {
            "study_instance_uid": acc.study_instance_uid,
            "modality": acc.modality,
            "body_part": acc.body_part,
            "study_description": acc.study_description,
            "series_count": len(acc.series_uids),
            "instance_count": acc.file_count,
        },
    }
    if config.DIMSE_INSTITUTION_ID:
        payload["institution_id"] = config.DIMSE_INSTITUTION_ID
    elif config.DIMSE_INSTITUTION_SLUG:
        payload["institution_slug"] = config.DIMSE_INSTITUTION_SLUG
    if acc.calling_ae_title:
        payload["institution_ae_title"] = acc.calling_ae_title

    try:
        with httpx.Client(timeout=config.DIMSE_INGEST_TIMEOUT) as client:
            resp = client.post(url, json=payload)
        if resp.status_code < 300:
            log.info(
                "Ingest OK for %s (%d files, %d series)",
                acc.study_instance_uid,
                acc.file_count,
                len(acc.series_uids),
            )
            return True
        log.error(
            "Ingest failed for %s: HTTP %d — %s",
            acc.study_instance_uid,
            resp.status_code,
            resp.text[:200],
        )
        return False
    except Exception as e:
        log.error("Ingest request failed for %s: %s", acc.study_instance_uid, e)
        return False


def submit_ingest(acc: StudyAccumulator) -> bool:
    """Try ingest now, enqueue retry on failure.

    Returns True when initial send succeeds. Returns False when queued or dropped.
    """
    ok = trigger_ingest(acc)
    if ok:
        return True
    queued = _enqueue_retry(acc, reason="initial_attempt_failed")
    if queued:
        log.warning(
            "Queued ingest retry for %s (queue size now %d)",
            acc.study_instance_uid,
            retry_snapshot()["pending"],
        )
    return False


def process_retry_queue(now: float | None = None) -> int:
    """Process due retry items once. Returns count processed."""
    current = time.time() if now is None else now
    processed = 0

    with _retry_lock:
        due = [item for item in _retry_queue if item.next_attempt_at <= current]
        _retry_queue[:] = [item for item in _retry_queue if item.next_attempt_at > current]

    for item in due:
        processed += 1
        _metrics["retried_total"] += 1
        ok = trigger_ingest(item.acc)
        if ok:
            _metrics["retried_ok_total"] += 1
            continue

        item.attempts += 1
        if item.attempts > config.DIMSE_INGEST_MAX_ATTEMPTS:
            item.last_error = "max_attempts_exceeded"
            with _retry_lock:
                _dead_letter.append(item)
                _metrics["dead_letter_total"] += 1
            log.error(
                "Ingest dead-letter for %s after %d attempts",
                item.acc.study_instance_uid,
                item.attempts - 1,
            )
            continue

        item.next_attempt_at = current + config.DIMSE_INGEST_RETRY_INTERVAL
        item.last_error = "retry_failed"
        with _retry_lock:
            _retry_queue.append(item)

    return processed


def retry_snapshot() -> dict[str, int]:
    """Return queue/dead-letter counters for health/status endpoints."""
    with _retry_lock:
        return {
            "pending": len(_retry_queue),
            "dead_letter": len(_dead_letter),
            "queued_total": _metrics["queued_total"],
            "retried_total": _metrics["retried_total"],
            "retried_ok_total": _metrics["retried_ok_total"],
            "dead_letter_total": _metrics["dead_letter_total"],
        }


def reset_retry_state() -> None:
    """Reset in-memory retry state (tests only)."""
    with _retry_lock:
        _retry_queue.clear()
        _dead_letter.clear()
        _metrics["queued_total"] = 0
        _metrics["retried_total"] = 0
        _metrics["retried_ok_total"] = 0
        _metrics["dead_letter_total"] = 0
