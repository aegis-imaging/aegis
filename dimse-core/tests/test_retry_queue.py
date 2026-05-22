"""Tests for the generic retry queue.

The cloud receiver and the spoke router both rely on this — bugs here would
manifest as silent data loss, so the coverage here is deliberately wide.
"""
from __future__ import annotations

import time

from dimse_core.retry_queue import RetryConfig, RetryQueue, RetryResult


def _cfg(**kw):
    return RetryConfig(**{
        "base_interval_seconds": 1.0,
        "multiplier": 2.0,
        "max_interval_seconds": 60.0,
        "max_attempts": 3,
        "queue_max": 10,
        **kw,
    })


def test_enqueue_and_process_success():
    processed = []

    def proc(item):
        processed.append(item.key)
        return RetryResult(ok=True)

    q = RetryQueue(_cfg(), proc)
    q.enqueue("study-1", {"x": 1})
    n = q.process_all()
    assert n == 1
    assert processed == ["study-1"]
    snap = q.snapshot()
    assert snap["pending"] == 0
    assert snap["retried_ok_total"] == 1


def test_failure_schedules_retry():
    fails = [0]

    def proc(item):
        fails[0] += 1
        return RetryResult(ok=False, error="nope")

    q = RetryQueue(_cfg(max_attempts=5), proc)
    q.enqueue("s", {})
    q.process_all()
    snap = q.snapshot()
    assert snap["pending"] == 1, "item should still be pending after one failure"
    assert snap["dead_letter"] == 0


def test_max_attempts_moves_to_dead_letter():
    def proc(item):
        return RetryResult(ok=False, error="permanent")

    q = RetryQueue(_cfg(max_attempts=2), proc)
    q.enqueue("s", {})
    # First attempt: failure → still pending.
    q.process_all()
    assert q.snapshot()["pending"] == 1
    # Second attempt: failure → dead-letter.
    q.process_all()
    snap = q.snapshot()
    assert snap["pending"] == 0
    assert snap["dead_letter"] == 1


def test_dedup_merges_payload():
    q = RetryQueue(_cfg(), lambda it: RetryResult(ok=True))
    q.enqueue("s", {"a": 1})
    q.enqueue("s", {"b": 2})
    snap = q.snapshot()
    assert snap["pending"] == 1
    assert snap["deduped_total"] == 1
    details = q.details()
    assert details["pending"][0]["payload"] == {"a": 1, "b": 2}


def test_dedup_dead_letter():
    q = RetryQueue(_cfg(max_attempts=1), lambda it: RetryResult(ok=False, error="fail"))
    q.enqueue("s", {"a": 1})
    q.process_all()
    assert q.snapshot()["dead_letter"] == 1
    q.enqueue("s", {"b": 2})  # Should merge into the dead-lettered item.
    snap = q.snapshot()
    assert snap["dead_letter"] == 1
    assert snap["dead_letter_deduped_total"] == 1


def test_replay_dead_letter_moves_back_to_pending():
    q = RetryQueue(_cfg(max_attempts=1), lambda it: RetryResult(ok=False, error="x"))
    q.enqueue("s", {})
    q.process_all()
    assert q.snapshot()["dead_letter"] == 1
    moved = q.replay_dead_letter()
    assert moved == 1
    snap = q.snapshot()
    assert snap["pending"] == 1
    assert snap["dead_letter"] == 0


def test_replay_dead_letter_one_by_key():
    q = RetryQueue(_cfg(max_attempts=1), lambda it: RetryResult(ok=False, error="x"))
    q.enqueue("a", {})
    q.enqueue("b", {})
    q.process_all()
    assert q.snapshot()["dead_letter"] == 2
    assert q.replay_dead_letter_one("a") is True
    snap = q.snapshot()
    assert snap["dead_letter"] == 1
    assert snap["pending"] == 1


def test_clear_dead_letter():
    q = RetryQueue(_cfg(max_attempts=1), lambda it: RetryResult(ok=False, error="x"))
    q.enqueue("a", {})
    q.enqueue("b", {})
    q.process_all()
    assert q.clear_dead_letter() == 2
    assert q.snapshot()["dead_letter"] == 0


def test_clear_pending():
    q = RetryQueue(_cfg(), lambda it: RetryResult(ok=True))
    q.enqueue("a", {})
    q.enqueue("b", {})
    assert q.clear_pending() == 2
    assert q.snapshot()["pending"] == 0


def test_queue_full_drops_to_dead_letter():
    q = RetryQueue(_cfg(queue_max=2), lambda it: RetryResult(ok=True))
    q.enqueue("a", {})
    q.enqueue("b", {})
    q.enqueue("c", {})
    snap = q.snapshot()
    assert snap["pending"] == 2
    assert snap["dead_letter"] == 1


def test_backoff_schedules_future_attempt():
    q = RetryQueue(_cfg(base_interval_seconds=10.0, max_attempts=5), lambda it: RetryResult(ok=False, error="x"))
    q.enqueue("s", {})
    now0 = time.time()
    q.process_all()
    details = q.details()
    item = details["pending"][0]
    assert item["next_attempt_at"] > now0 + 5, "next attempt should be scheduled in the future"


def test_process_due_respects_schedule():
    """An item scheduled in the future is NOT picked up by process_due()."""
    q = RetryQueue(_cfg(base_interval_seconds=100.0, max_attempts=5),
                   lambda it: RetryResult(ok=False, error="x"))
    q.enqueue("s", {})
    q.process_all()  # first failure, now scheduled 100s out
    n = q.process_due()
    assert n == 0


def test_persist_and_reload(tmp_path):
    path = tmp_path / "state.json"
    q1 = RetryQueue(_cfg(max_attempts=1, durable_path=str(path)),
                    lambda it: RetryResult(ok=False, error="x"))
    q1.enqueue("a", {"v": 1})
    q1.process_all()  # → dead-letter
    q1.enqueue("b", {"v": 2})  # still pending
    snap_before = q1.snapshot()
    assert snap_before["pending"] == 1
    assert snap_before["dead_letter"] == 1

    q2 = RetryQueue(_cfg(max_attempts=1, durable_path=str(path)),
                    lambda it: RetryResult(ok=True))
    snap_after = q2.snapshot()
    assert snap_after["pending"] == 1
    assert snap_after["dead_letter"] == 1


def test_process_one_by_key():
    q = RetryQueue(_cfg(), lambda it: RetryResult(ok=True))
    q.enqueue("a", {})
    q.enqueue("b", {})
    result = q.process_one("a")
    assert result is not None and result.ok
    assert q.snapshot()["pending"] == 1


def test_process_one_unknown_returns_none():
    q = RetryQueue(_cfg(), lambda it: RetryResult(ok=True))
    assert q.process_one("never-enqueued") is None
