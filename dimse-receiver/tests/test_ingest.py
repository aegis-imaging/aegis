"""Tests for app.ingest."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import app.ingest as ingest_module
from app.ingest import (
    _retry_delay_seconds,
    clear_dead_letter,
    clear_dead_letter_study,
    clear_pending,
    clear_pending_study,
    replay_dead_letter,
    replay_dead_letter_study,
    retry_details,
    StudyAccumulator,
    process_retry_queue,
    process_retry_all,
    process_retry_study,
    reset_retry_state,
    retry_snapshot,
    submit_ingest,
    trigger_ingest,
)


def _acc() -> StudyAccumulator:
    return StudyAccumulator(
        study_instance_uid="1.2.3.4",
        modality="MR",
        body_part="HEAD",
        study_description="BRAIN MRI",
        series_uids={"1", "2"},
        file_count=5,
        calling_ae_title="PACS_AE",
    )


def setup_function():
    reset_retry_state()


def test_trigger_ingest_success(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_PROJECT_SLUG", "default")
    monkeypatch.setattr("app.config.DIMSE_INGEST_TIMEOUT", 30)
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_ID", "")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_SLUG", "")

    mock_resp = MagicMock(status_code=200, text="ok")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is True
    mock_client.post.assert_called_once()
    called_url = mock_client.post.call_args[0][0]
    called_json = mock_client.post.call_args.kwargs["json"]
    assert called_url == "http://api:8080/api/ingest"
    assert called_json["project_slug"] == "default"
    assert called_json["study_metadata"]["study_instance_uid"] == "1.2.3.4"
    assert called_json["study_metadata"]["series_count"] == 2
    assert called_json["study_metadata"]["instance_count"] == 5
    assert called_json["institution_ae_title"] == "PACS_AE"
    assert "institution_id" not in called_json


def test_trigger_ingest_http_error(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_ID", "")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_SLUG", "")

    mock_resp = MagicMock(status_code=500, text="boom")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is False


def test_trigger_ingest_exception(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_ID", "")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_SLUG", "")

    with patch("app.ingest.httpx.Client", side_effect=RuntimeError("network down")):
        ok = trigger_ingest(_acc())

    assert ok is False


def test_trigger_ingest_includes_config_institution_id(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_PROJECT_SLUG", "default")
    monkeypatch.setattr("app.config.DIMSE_INGEST_TIMEOUT", 30)
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_ID", "inst-123")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_SLUG", "")

    mock_resp = MagicMock(status_code=201, text="created")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is True
    called_json = mock_client.post.call_args.kwargs["json"]
    assert called_json["institution_id"] == "inst-123"


def test_trigger_ingest_includes_config_institution_slug(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_PROJECT_SLUG", "default")
    monkeypatch.setattr("app.config.DIMSE_INGEST_TIMEOUT", 30)
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_ID", "")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_SLUG", "hospital-bravo")

    mock_resp = MagicMock(status_code=201, text="created")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is True
    called_json = mock_client.post.call_args.kwargs["json"]
    assert called_json["institution_slug"] == "hospital-bravo"
    assert "institution_id" not in called_json


def test_trigger_ingest_prefers_institution_id_over_slug(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_PROJECT_SLUG", "default")
    monkeypatch.setattr("app.config.DIMSE_INGEST_TIMEOUT", 30)
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_ID", "inst-123")
    monkeypatch.setattr("app.config.DIMSE_INSTITUTION_SLUG", "hospital-bravo")

    mock_resp = MagicMock(status_code=201, text="created")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is True
    called_json = mock_client.post.call_args.kwargs["json"]
    assert called_json["institution_id"] == "inst-123"
    assert "institution_slug" not in called_json


def test_submit_ingest_queues_on_failure(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    with patch("app.ingest.trigger_ingest", return_value=False):
        ok = submit_ingest(_acc())

    assert ok is False
    snap = retry_snapshot()
    assert snap["pending"] == 1
    assert snap["queued_total"] == 1
    assert snap["deduped_total"] == 0


def test_submit_ingest_dedupes_same_study_uid(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    acc_a = _acc()
    acc_a.file_count = 2
    acc_a.series_uids = {"1"}

    acc_b = _acc()
    acc_b.file_count = 7
    acc_b.series_uids = {"1", "2", "3"}

    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(acc_a)
        submit_ingest(acc_b)

    snap = retry_snapshot()
    assert snap["pending"] == 1
    assert snap["queued_total"] == 1
    assert snap["deduped_total"] == 1

    queued = ingest_module._retry_queue[0]
    assert queued.acc.file_count == 7
    assert queued.acc.series_uids == {"1", "2", "3"}


def test_retry_snapshot_reports_oldest_queue_ages(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=100.0):
        submit_ingest(_acc())

    acc2 = _acc()
    acc2.study_instance_uid = "9.9.9.9"
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=120.0):
        submit_ingest(acc2)

    snap = retry_snapshot(now=160.0)
    assert snap["pending"] == 1
    assert snap["dead_letter"] == 1
    assert snap["pending_oldest_age_seconds"] == 60
    assert snap["dead_letter_oldest_age_seconds"] == 40


def test_retry_snapshot_reports_next_pending_attempt(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    acc_a = _acc()
    acc_b = _acc()
    acc_b.study_instance_uid = "2.2.2.2"

    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=100.0):
        submit_ingest(acc_a)  # next due = 115
    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=120.0):
        submit_ingest(acc_b)  # next due = 135

    snap = retry_snapshot(now=110.0)
    assert snap["pending"] == 2
    assert snap["pending_next_attempt_at"] == 115
    assert snap["pending_next_attempt_in_seconds"] == 5


def test_retry_snapshot_reports_no_next_pending_attempt_when_empty():
    snap = retry_snapshot(now=100.0)
    assert snap["pending"] == 0
    assert snap["pending_next_attempt_at"] == 0
    assert snap["pending_next_attempt_in_seconds"] == 0


def test_retry_delay_seconds_exponential_and_capped(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 10)
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER", 2.0)
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_MAX_INTERVAL", 25)

    assert _retry_delay_seconds(1) == 10
    assert _retry_delay_seconds(2) == 20
    assert _retry_delay_seconds(3) == 25  # capped from 40


def test_process_retry_queue_retries_and_succeeds(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())

    queued = ingest_module._retry_queue[0]
    with patch("app.ingest.trigger_ingest", return_value=True):
        processed = process_retry_queue(now=queued.next_attempt_at + 1)

    assert processed == 1
    snap = retry_snapshot()
    assert snap["pending"] == 0
    assert snap["retried_total"] == 1
    assert snap["retried_ok_total"] == 1
    assert snap["dead_letter"] == 0


def test_process_retry_queue_moves_to_dead_letter_after_max_attempts(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 1)
    monkeypatch.setattr("app.config.DIMSE_INGEST_MAX_ATTEMPTS", 2)

    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        queued = ingest_module._retry_queue[0]
        process_retry_queue(now=queued.next_attempt_at + 1)  # attempt 2 fails, requeue
        queued2 = ingest_module._retry_queue[0]
        process_retry_queue(now=queued2.next_attempt_at + 1)  # attempt 3 -> dead letter

    snap = retry_snapshot()
    assert snap["pending"] == 0
    assert snap["dead_letter"] == 1
    assert snap["dead_letter_total"] == 1


def test_process_retry_queue_applies_backoff(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 10)
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER", 2.0)
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_MAX_INTERVAL", 300)
    monkeypatch.setattr("app.config.DIMSE_INGEST_MAX_ATTEMPTS", 10)

    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        first_due = ingest_module._retry_queue[0].next_attempt_at
        process_retry_queue(now=first_due + 1)
        second_due = ingest_module._retry_queue[0].next_attempt_at
        process_retry_queue(now=second_due + 1)
        third_due = ingest_module._retry_queue[0].next_attempt_at

    # First retry delay: 20s, second retry delay: 40s
    assert int(second_due - (first_due + 1)) == 20
    assert int(third_due - (second_due + 1)) == 40


def test_process_retry_study_success(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())

    with patch("app.ingest.trigger_ingest", return_value=True):
        result = process_retry_study("1.2.3.4", now=123.0)

    assert result["found"] is True
    assert result["attempted"] is True
    assert result["result"] == "ok"
    assert result["snapshot"]["pending"] == 0


def test_process_retry_study_requeued(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 10)
    monkeypatch.setattr("app.config.DIMSE_INGEST_MAX_ATTEMPTS", 5)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        result = process_retry_study("1.2.3.4", now=100.0)

    assert result["found"] is True
    assert result["result"] == "requeued"
    assert result["snapshot"]["pending"] == 1


def test_process_retry_study_dead_letter(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 10)
    monkeypatch.setattr("app.config.DIMSE_INGEST_MAX_ATTEMPTS", 1)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        result = process_retry_study("1.2.3.4", now=100.0)

    assert result["found"] is True
    assert result["result"] == "dead_letter"
    assert result["snapshot"]["pending"] == 0
    assert result["snapshot"]["dead_letter"] == 1


def test_process_retry_study_not_found():
    result = process_retry_study("missing-study")
    assert result["found"] is False
    assert result["attempted"] is False
    assert result["result"] == "not_found"


def test_process_retry_all_requeues_failures(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 10)
    monkeypatch.setattr("app.config.DIMSE_INGEST_MAX_ATTEMPTS", 5)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        acc2 = _acc()
        acc2.study_instance_uid = "2.2.2.2"
        submit_ingest(acc2)
        result = process_retry_all(limit=10, now=100.0)

    assert result["processed"] == 2
    assert result["ok"] == 0
    assert result["requeued"] == 2
    assert result["dead_letter"] == 0
    assert result["snapshot"]["pending"] == 2


def test_process_retry_all_respects_limit(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 10)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        acc2 = _acc()
        acc2.study_instance_uid = "2.2.2.2"
        submit_ingest(acc2)

    with patch("app.ingest.trigger_ingest", return_value=True):
        result = process_retry_all(limit=1, now=100.0)

    assert result["processed"] == 1
    assert result["ok"] == 1
    assert result["snapshot"]["pending"] == 1


def test_submit_ingest_dead_letters_when_queue_full(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    with patch("app.ingest.trigger_ingest", return_value=False):
        ok = submit_ingest(_acc())
    assert ok is False
    snap = retry_snapshot()
    assert snap["pending"] == 0
    assert snap["dead_letter"] == 1


def test_submit_ingest_dead_letter_dedupes_same_study(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        submit_ingest(_acc())

    snap = retry_snapshot()
    assert snap["dead_letter"] == 1
    assert snap["dead_letter_total"] == 1
    assert snap["dead_letter_deduped_total"] == 1


def test_replay_dead_letter_moves_items_to_pending(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 10)

    with patch("app.ingest.trigger_ingest", return_value=False):
        monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
        submit_ingest(_acc())  # force dead-letter via queue-full
        monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 10)

    before = retry_snapshot()
    assert before["dead_letter"] == 1
    assert before["pending"] == 0

    after = replay_dead_letter(limit=10, now=123.0)
    assert after["replayed_now"] == 1
    assert after["dead_letter"] == 0
    assert after["pending"] == 1
    assert after["replayed_total"] == 1


def test_replay_dead_letter_resets_pending_age(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=100.0):
        submit_ingest(_acc())

    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 10)
    with patch("app.ingest.time.time", return_value=130.0):
        replay_dead_letter(limit=1)

    snap = retry_snapshot(now=160.0)
    assert snap["pending"] == 1
    assert snap["dead_letter"] == 0
    assert snap["pending_oldest_age_seconds"] == 30
    assert snap["dead_letter_oldest_age_seconds"] == 0


def test_replay_dead_letter_respects_limit(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 10)
    acc2 = _acc()
    acc2.study_instance_uid = "2.2.2.2"
    with patch("app.ingest.trigger_ingest", return_value=False):
        monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
        submit_ingest(_acc())
        submit_ingest(acc2)
        monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 10)

    after = replay_dead_letter(limit=1, now=123.0)
    assert after["replayed_now"] == 1
    assert after["dead_letter"] == 1
    assert after["pending"] == 1


def test_retry_details_returns_pending_and_dead_letter(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 1)
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    acc2 = _acc()
    acc2.study_instance_uid = "9.9.9.9"

    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())  # queued
        submit_ingest(acc2)  # dead-letter (queue full, different study)

    details = retry_details(limit=10, now=100.0)
    assert details["limit"] == 10
    assert details["snapshot"]["pending"] == 1
    assert details["snapshot"]["dead_letter"] == 1
    assert len(details["pending_items"]) == 1
    assert len(details["dead_letter_items"]) == 1
    assert details["pending_items"][0]["study_instance_uid"] == "1.2.3.4"
    assert "seconds_until_next_attempt" in details["pending_items"][0]
    assert "queued_at" in details["pending_items"][0]
    assert "age_seconds" in details["pending_items"][0]
    assert "dead_lettered_at" in details["dead_letter_items"][0]
    assert "dead_letter_age_seconds" in details["dead_letter_items"][0]


def test_retry_details_reports_item_ages(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 1)
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)

    acc2 = _acc()
    acc2.study_instance_uid = "9.9.9.9"

    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=100.0):
        submit_ingest(_acc())
    with patch("app.ingest.trigger_ingest", return_value=False), patch("app.ingest.time.time", return_value=120.0):
        submit_ingest(acc2)

    details = retry_details(limit=10, now=160.0)
    pending_item = details["pending_items"][0]
    dead_item = details["dead_letter_items"][0]
    assert pending_item["queued_at"] == 100
    assert pending_item["age_seconds"] == 60
    assert dead_item["dead_lettered_at"] == 120
    assert dead_item["dead_letter_age_seconds"] == 40


def test_clear_dead_letter_removes_items(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    acc2 = _acc()
    acc2.study_instance_uid = "2.2.2.2"
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        submit_ingest(acc2)

    before = retry_snapshot()
    assert before["dead_letter"] == 2

    after = clear_dead_letter(limit=1)
    assert after["cleared_now"] == 1
    assert after["dead_letter"] == 1

    after2 = clear_dead_letter(limit=10)
    assert after2["cleared_now"] == 1
    assert after2["dead_letter"] == 0
    assert after2["cleared_dead_letter_total"] == 2


def test_replay_dead_letter_study_moves_matching_item(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())

    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 10)
    result = replay_dead_letter_study("1.2.3.4", now=456.0)
    assert result["found"] is True
    assert result["moved"] == 1
    assert result["blocked_by_queue_full"] is False
    assert result["snapshot"]["pending"] == 1
    assert result["snapshot"]["dead_letter"] == 0


def test_replay_dead_letter_study_not_found():
    result = replay_dead_letter_study("missing-study")
    assert result["found"] is False
    assert result["moved"] == 0
    assert result["blocked_by_queue_full"] is False


def test_clear_dead_letter_study_found(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_QUEUE_MAX", 0)
    acc2 = _acc()
    acc2.study_instance_uid = "2.2.2.2"
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        submit_ingest(acc2)

    result = clear_dead_letter_study("2.2.2.2")
    assert result["found"] is True
    assert result["cleared"] == 1
    assert result["snapshot"]["dead_letter"] == 1


def test_clear_dead_letter_study_not_found():
    result = clear_dead_letter_study("missing-study")
    assert result["found"] is False
    assert result["cleared"] == 0


def test_clear_pending_removes_items(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        acc2 = _acc()
        acc2.study_instance_uid = "2.2.2.2"
        submit_ingest(acc2)

    before = retry_snapshot()
    assert before["pending"] == 2

    after = clear_pending(limit=1)
    assert after["cleared_now"] == 1
    assert after["pending"] == 1

    after2 = clear_pending(limit=10)
    assert after2["cleared_now"] == 1
    assert after2["pending"] == 0
    assert after2["cleared_pending_total"] == 2


def test_clear_pending_study_found(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_INGEST_RETRY_INTERVAL", 15)
    with patch("app.ingest.trigger_ingest", return_value=False):
        submit_ingest(_acc())
        acc2 = _acc()
        acc2.study_instance_uid = "2.2.2.2"
        submit_ingest(acc2)

    result = clear_pending_study("2.2.2.2")
    assert result["found"] is True
    assert result["cleared"] == 1
    assert result["snapshot"]["pending"] == 1


def test_clear_pending_study_not_found():
    result = clear_pending_study("missing-study")
    assert result["found"] is False
    assert result["cleared"] == 0
