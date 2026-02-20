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
    queued_at: float = 0.0
    dead_lettered_at: float = 0.0


_retry_queue: list[QueuedIngest] = []
_dead_letter: list[QueuedIngest] = []
_retry_lock = threading.Lock()
_metrics = {
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


def _retry_delay_seconds(attempts: int) -> int:
    """Return retry delay using bounded exponential backoff."""
    base = max(1, int(config.DIMSE_INGEST_RETRY_INTERVAL))
    multiplier = max(1.0, float(config.DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER))
    max_interval = max(base, int(config.DIMSE_INGEST_RETRY_MAX_INTERVAL))

    exponent = max(0, attempts - 1)
    delay = int(base * (multiplier ** exponent))
    return min(delay, max_interval)


def _merge_accumulator(existing: StudyAccumulator, incoming: StudyAccumulator) -> None:
    """Merge incoming study metadata into an existing queued accumulator."""
    existing.file_count = max(existing.file_count, incoming.file_count)
    existing.series_uids.update(incoming.series_uids)
    if not existing.modality and incoming.modality:
        existing.modality = incoming.modality
    if not existing.body_part and incoming.body_part:
        existing.body_part = incoming.body_part
    if not existing.study_description and incoming.study_description:
        existing.study_description = incoming.study_description
    if not existing.calling_ae_title and incoming.calling_ae_title:
        existing.calling_ae_title = incoming.calling_ae_title


def _upsert_dead_letter(item: QueuedIngest) -> None:
    """Insert or merge a dead-letter item by StudyInstanceUID."""
    for existing in _dead_letter:
        if existing.acc.study_instance_uid == item.acc.study_instance_uid:
            _merge_accumulator(existing.acc, item.acc)
            existing.attempts = max(existing.attempts, item.attempts)
            existing.last_error = item.last_error
            if item.queued_at > 0 and (existing.queued_at <= 0 or item.queued_at < existing.queued_at):
                existing.queued_at = item.queued_at
            if item.dead_lettered_at > 0 and (
                existing.dead_lettered_at <= 0 or item.dead_lettered_at < existing.dead_lettered_at
            ):
                existing.dead_lettered_at = item.dead_lettered_at
            _metrics["dead_letter_deduped_total"] += 1
            return

    _dead_letter.append(item)
    _metrics["dead_letter_total"] += 1


def _enqueue_retry(acc: StudyAccumulator, reason: str) -> bool:
    """Queue a study for retry if queue capacity permits."""
    current = time.time()
    with _retry_lock:
        for item in _retry_queue:
            if item.acc.study_instance_uid == acc.study_instance_uid:
                _merge_accumulator(item.acc, acc)
                item.last_error = f"deduped: {reason}"
                _metrics["deduped_total"] += 1
                return True

        if len(_retry_queue) >= config.DIMSE_INGEST_QUEUE_MAX:
            log.error(
                "Retry queue full (%d), dropping study %s",
                config.DIMSE_INGEST_QUEUE_MAX,
                acc.study_instance_uid,
            )
            _upsert_dead_letter(
                QueuedIngest(
                    acc=acc,
                    attempts=1,
                    next_attempt_at=current,
                    last_error=f"queue_full: {reason}",
                    queued_at=current,
                    dead_lettered_at=current,
                )
            )
            return False

        _retry_queue.append(
            QueuedIngest(
                acc=acc,
                attempts=1,
                next_attempt_at=current + _retry_delay_seconds(1),
                last_error=reason,
                queued_at=current,
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
        _process_retry_item(item, current)

    return processed


def _process_retry_item(item: QueuedIngest, current: float) -> str:
    """Process one retry item. Returns 'ok', 'requeued', or 'dead_letter'."""
    if item.queued_at <= 0:
        item.queued_at = current

    _metrics["retried_total"] += 1
    ok = trigger_ingest(item.acc)
    if ok:
        _metrics["retried_ok_total"] += 1
        return "ok"

    item.attempts += 1
    if item.attempts > config.DIMSE_INGEST_MAX_ATTEMPTS:
        item.last_error = "max_attempts_exceeded"
        item.dead_lettered_at = current
        with _retry_lock:
            _upsert_dead_letter(item)
        log.error(
            "Ingest dead-letter for %s after %d attempts",
            item.acc.study_instance_uid,
            item.attempts - 1,
        )
        return "dead_letter"

    item.next_attempt_at = current + _retry_delay_seconds(item.attempts)
    item.last_error = "retry_failed"
    with _retry_lock:
        _retry_queue.append(item)
    return "requeued"


def process_retry_study(study_instance_uid: str, now: float | None = None) -> dict[str, object]:
    """Immediately process one pending retry item by StudyInstanceUID."""
    current = time.time() if now is None else now
    item = None
    with _retry_lock:
        idx = next(
            (i for i, item in enumerate(_retry_queue) if item.acc.study_instance_uid == study_instance_uid),
            -1,
        )
        if idx >= 0:
            item = _retry_queue.pop(idx)

    if item is None:
        return {
            "snapshot": retry_snapshot(),
            "study_instance_uid": study_instance_uid,
            "found": False,
            "attempted": False,
            "result": "not_found",
        }

    result = _process_retry_item(item, current)
    return {
        "snapshot": retry_snapshot(),
        "study_instance_uid": study_instance_uid,
        "found": True,
        "attempted": True,
        "result": result,
    }


def process_retry_all(limit: int = 10000, now: float | None = None) -> dict[str, object]:
    """Immediately process up to `limit` pending retry items regardless of schedule."""
    current = time.time() if now is None else now
    with _retry_lock:
        ordered = sorted(_retry_queue, key=lambda item: item.next_attempt_at)
        selected = ordered[:limit]
        selected_ids = {id(item) for item in selected}
        _retry_queue[:] = [item for item in _retry_queue if id(item) not in selected_ids]

    counts = {"ok": 0, "requeued": 0, "dead_letter": 0}
    for item in selected:
        result = _process_retry_item(item, current)
        if result in counts:
            counts[result] += 1

    return {
        "snapshot": retry_snapshot(),
        "processed": len(selected),
        "ok": counts["ok"],
        "requeued": counts["requeued"],
        "dead_letter": counts["dead_letter"],
    }


def _oldest_age_seconds(items: list[QueuedIngest], current: float, timestamp_attr: str) -> int:
    """Return whole-second age of the oldest item in a queue-like collection."""
    oldest = min(
        (int(getattr(item, timestamp_attr)) for item in items if getattr(item, timestamp_attr) > 0),
        default=0,
    )
    if oldest <= 0:
        return 0
    return max(0, int(current) - oldest)


def _pending_next_attempt(items: list[QueuedIngest], current: float) -> tuple[int, int]:
    """Return epoch seconds for next due pending retry and seconds until due."""
    next_due = min((int(item.next_attempt_at) for item in items), default=0)
    if next_due <= 0:
        return 0, 0
    return next_due, max(0, next_due - int(current))


def retry_snapshot(now: float | None = None) -> dict[str, int]:
    """Return queue/dead-letter counters for health/status endpoints."""
    current = time.time() if now is None else now
    with _retry_lock:
        pending_oldest_age_seconds = _oldest_age_seconds(_retry_queue, current, "queued_at")
        dead_letter_oldest_age_seconds = _oldest_age_seconds(_dead_letter, current, "dead_lettered_at")
        pending_next_attempt_at, pending_next_attempt_in_seconds = _pending_next_attempt(_retry_queue, current)
        return {
            "pending": len(_retry_queue),
            "dead_letter": len(_dead_letter),
            "queued_total": _metrics["queued_total"],
            "deduped_total": _metrics["deduped_total"],
            "retried_total": _metrics["retried_total"],
            "retried_ok_total": _metrics["retried_ok_total"],
            "dead_letter_total": _metrics["dead_letter_total"],
            "dead_letter_deduped_total": _metrics["dead_letter_deduped_total"],
            "replayed_total": _metrics["replayed_total"],
            "cleared_dead_letter_total": _metrics["cleared_dead_letter_total"],
            "cleared_pending_total": _metrics["cleared_pending_total"],
            "pending_oldest_age_seconds": pending_oldest_age_seconds,
            "dead_letter_oldest_age_seconds": dead_letter_oldest_age_seconds,
            "pending_next_attempt_at": pending_next_attempt_at,
            "pending_next_attempt_in_seconds": pending_next_attempt_in_seconds,
        }


def retry_details(
    limit: int = 100,
    now: float | None = None,
    study_instance_uid: str | None = None,
    sort: str = "next_attempt",
) -> dict[str, object]:
    """Return queue/dead-letter details for operational debugging."""
    current = time.time() if now is None else now
    sort_mode = sort if sort in {"next_attempt", "age_desc"} else "next_attempt"

    with _retry_lock:
        pending_source = _retry_queue
        dead_source = _dead_letter
        if study_instance_uid:
            pending_source = [item for item in _retry_queue if item.acc.study_instance_uid == study_instance_uid]
            dead_source = [item for item in _dead_letter if item.acc.study_instance_uid == study_instance_uid]
        pending_total = len(pending_source)
        dead_letter_total = len(dead_source)
        if sort_mode == "age_desc":
            pending = sorted(pending_source, key=lambda item: item.queued_at or current)[:limit]
            dead = sorted(dead_source, key=lambda item: item.dead_lettered_at or current)[:limit]
        else:
            pending = sorted(pending_source, key=lambda item: item.next_attempt_at)[:limit]
            dead = sorted(dead_source, key=lambda item: item.dead_lettered_at or 0)[:limit]

    pending_items = [
        {
            "study_instance_uid": item.acc.study_instance_uid,
            "attempts": item.attempts,
            "next_attempt_at": int(item.next_attempt_at),
            "seconds_until_next_attempt": max(0, int(item.next_attempt_at - current)),
            "queued_at": int(item.queued_at) if item.queued_at > 0 else 0,
            "age_seconds": max(0, int(current - item.queued_at)) if item.queued_at > 0 else 0,
            "file_count": item.acc.file_count,
            "series_count": len(item.acc.series_uids),
            "last_error": item.last_error,
        }
        for item in pending
    ]
    dead_items = [
        {
            "study_instance_uid": item.acc.study_instance_uid,
            "attempts": item.attempts,
            "queued_at": int(item.queued_at) if item.queued_at > 0 else 0,
            "dead_lettered_at": int(item.dead_lettered_at) if item.dead_lettered_at > 0 else 0,
            "dead_letter_age_seconds": (
                max(0, int(current - item.dead_lettered_at)) if item.dead_lettered_at > 0 else 0
            ),
            "file_count": item.acc.file_count,
            "series_count": len(item.acc.series_uids),
            "last_error": item.last_error,
        }
        for item in dead
    ]

    return {
        "snapshot": retry_snapshot(),
        "now": int(current),
        "limit": limit,
        "study_instance_uid": study_instance_uid or "",
        "sort": sort_mode,
        "pending_total": pending_total,
        "dead_letter_total": dead_letter_total,
        "pending_returned": len(pending_items),
        "dead_letter_returned": len(dead_items),
        "pending_truncated": len(pending_items) < pending_total,
        "dead_letter_truncated": len(dead_items) < dead_letter_total,
        "pending_items": pending_items,
        "dead_letter_items": dead_items,
    }


def replay_dead_letter(limit: int = 100, now: float | None = None) -> dict[str, int]:
    """Move dead-letter items back into the retry queue.

    Items are re-queued for immediate processing. Returns counters including moved count.
    """
    current = time.time() if now is None else now
    moved = 0
    before_pending = 0
    before_dead_letter = 0
    after_pending = 0
    after_dead_letter = 0
    blocked_by_queue_full = False

    with _retry_lock:
        before_pending = len(_retry_queue)
        before_dead_letter = len(_dead_letter)
        while moved < limit and _dead_letter and len(_retry_queue) < config.DIMSE_INGEST_QUEUE_MAX:
            item = _dead_letter.pop(0)
            item.next_attempt_at = current
            item.last_error = "replayed"
            item.queued_at = current
            item.dead_lettered_at = 0.0
            _retry_queue.append(item)
            moved += 1
            _metrics["replayed_total"] += 1
        blocked_by_queue_full = bool(_dead_letter and len(_retry_queue) >= config.DIMSE_INGEST_QUEUE_MAX and moved < limit)
        after_pending = len(_retry_queue)
        after_dead_letter = len(_dead_letter)

    snapshot = retry_snapshot()
    snapshot["replayed_now"] = moved
    snapshot["before_pending"] = before_pending
    snapshot["before_dead_letter"] = before_dead_letter
    snapshot["after_pending"] = after_pending
    snapshot["after_dead_letter"] = after_dead_letter
    snapshot["blocked_by_queue_full"] = blocked_by_queue_full
    return snapshot


def replay_dead_letter_study(study_instance_uid: str, now: float | None = None) -> dict[str, object]:
    """Replay one dead-letter entry matching a specific StudyInstanceUID."""
    current = time.time() if now is None else now
    found = False
    moved = 0
    blocked_by_queue_full = False
    before_pending = 0
    before_dead_letter = 0
    after_pending = 0
    after_dead_letter = 0

    with _retry_lock:
        before_pending = len(_retry_queue)
        before_dead_letter = len(_dead_letter)
        idx = next(
            (i for i, item in enumerate(_dead_letter) if item.acc.study_instance_uid == study_instance_uid),
            -1,
        )
        if idx >= 0:
            found = True
            if len(_retry_queue) >= config.DIMSE_INGEST_QUEUE_MAX:
                blocked_by_queue_full = True
            else:
                item = _dead_letter.pop(idx)
                item.next_attempt_at = current
                item.last_error = "replayed_targeted"
                item.queued_at = current
                item.dead_lettered_at = 0.0
                _retry_queue.append(item)
                moved = 1
                _metrics["replayed_total"] += 1
        after_pending = len(_retry_queue)
        after_dead_letter = len(_dead_letter)

    snapshot = retry_snapshot()
    return {
        "snapshot": snapshot,
        "study_instance_uid": study_instance_uid,
        "found": found,
        "moved": moved,
        "blocked_by_queue_full": blocked_by_queue_full,
        "before_pending": before_pending,
        "before_dead_letter": before_dead_letter,
        "after_pending": after_pending,
        "after_dead_letter": after_dead_letter,
    }


def clear_dead_letter(limit: int = 10000) -> dict[str, int]:
    """Clear up to `limit` dead-letter items and return counters."""
    cleared = 0
    before_dead_letter = 0
    after_dead_letter = 0
    with _retry_lock:
        before_dead_letter = len(_dead_letter)
        while cleared < limit and _dead_letter:
            _dead_letter.pop(0)
            cleared += 1
        _metrics["cleared_dead_letter_total"] += cleared
        after_dead_letter = len(_dead_letter)

    snapshot = retry_snapshot()
    snapshot["cleared_now"] = cleared
    snapshot["before_dead_letter"] = before_dead_letter
    snapshot["after_dead_letter"] = after_dead_letter
    return snapshot


def clear_dead_letter_study(study_instance_uid: str) -> dict[str, object]:
    """Clear one dead-letter entry by StudyInstanceUID."""
    cleared = 0
    found = False
    before_dead_letter = 0
    after_dead_letter = 0
    with _retry_lock:
        before_dead_letter = len(_dead_letter)
        idx = next(
            (i for i, item in enumerate(_dead_letter) if item.acc.study_instance_uid == study_instance_uid),
            -1,
        )
        if idx >= 0:
            found = True
            _dead_letter.pop(idx)
            cleared = 1
            _metrics["cleared_dead_letter_total"] += 1
        after_dead_letter = len(_dead_letter)

    return {
        "snapshot": retry_snapshot(),
        "study_instance_uid": study_instance_uid,
        "found": found,
        "cleared": cleared,
        "before_dead_letter": before_dead_letter,
        "after_dead_letter": after_dead_letter,
    }


def clear_pending(limit: int = 10000) -> dict[str, int]:
    """Clear up to `limit` pending retry queue items and return counters."""
    cleared = 0
    before_pending = 0
    after_pending = 0
    with _retry_lock:
        before_pending = len(_retry_queue)
        while cleared < limit and _retry_queue:
            _retry_queue.pop(0)
            cleared += 1
        _metrics["cleared_pending_total"] += cleared
        after_pending = len(_retry_queue)

    snapshot = retry_snapshot()
    snapshot["cleared_now"] = cleared
    snapshot["before_pending"] = before_pending
    snapshot["after_pending"] = after_pending
    return snapshot


def clear_pending_study(study_instance_uid: str) -> dict[str, object]:
    """Clear one pending retry entry by StudyInstanceUID."""
    cleared = 0
    found = False
    before_pending = 0
    after_pending = 0
    with _retry_lock:
        before_pending = len(_retry_queue)
        idx = next(
            (i for i, item in enumerate(_retry_queue) if item.acc.study_instance_uid == study_instance_uid),
            -1,
        )
        if idx >= 0:
            found = True
            _retry_queue.pop(idx)
            cleared = 1
            _metrics["cleared_pending_total"] += 1
        after_pending = len(_retry_queue)

    return {
        "snapshot": retry_snapshot(),
        "study_instance_uid": study_instance_uid,
        "found": found,
        "cleared": cleared,
        "before_pending": before_pending,
        "after_pending": after_pending,
    }


def reset_retry_state() -> None:
    """Reset in-memory retry state (tests only)."""
    with _retry_lock:
        _retry_queue.clear()
        _dead_letter.clear()
        _metrics["queued_total"] = 0
        _metrics["deduped_total"] = 0
        _metrics["retried_total"] = 0
        _metrics["retried_ok_total"] = 0
        _metrics["dead_letter_total"] = 0
        _metrics["dead_letter_deduped_total"] = 0
        _metrics["replayed_total"] = 0
        _metrics["cleared_dead_letter_total"] = 0
        _metrics["cleared_pending_total"] = 0
