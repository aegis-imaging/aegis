"""Tests for retry alerting module."""

from __future__ import annotations

from app.retry_alerts import evaluate_retry_alerts, get_alerts, reset_alerts


def setup_function():
    reset_alerts()
    import app.config as cfg

    cfg.DIMSE_RETRY_ALERTS_ENABLED = False
    cfg.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS = 300
    cfg.DIMSE_RETRY_ALERT_MAX_EVENTS = 200
    cfg.DIMSE_RETRY_ALERT_PENDING_AGE_SECONDS = 0
    cfg.DIMSE_RETRY_ALERT_DEAD_LETTER_AGE_SECONDS = 0
    cfg.DIMSE_RETRY_ALERT_DEAD_LETTER_NONZERO = True
    cfg.DIMSE_RETRY_ALERT_WEBHOOK_URL = ""
    cfg.DIMSE_INGEST_PENDING_AGE_WARN_SECONDS = 0
    cfg.DIMSE_DEAD_LETTER_AGE_WARN_SECONDS = 0


def test_evaluate_retry_alerts_disabled_no_events():
    snapshot = {"pending": 0, "dead_letter": 3}
    result = evaluate_retry_alerts(snapshot, now=100.0)
    assert result == {"active": 0, "emitted": 0}
    assert get_alerts()["total"] == 0


def test_dead_letter_nonzero_alert_respects_cooldown(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERTS_ENABLED", True)
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS", 60)
    snapshot = {"pending": 0, "dead_letter": 2}

    result1 = evaluate_retry_alerts(snapshot, now=100.0)
    result2 = evaluate_retry_alerts(snapshot, now=120.0)
    result3 = evaluate_retry_alerts(snapshot, now=161.0)

    assert result1 == {"active": 1, "emitted": 1}
    assert result2 == {"active": 1, "emitted": 0}
    assert result3 == {"active": 1, "emitted": 1}

    alerts = get_alerts()
    assert alerts["total"] == 2
    assert alerts["items"][0]["condition"] == "dead_letter_nonzero"
    assert alerts["items"][1]["condition"] == "dead_letter_nonzero"


def test_age_threshold_alerts_use_warn_threshold_fallback(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERTS_ENABLED", True)
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS", 0)
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERT_DEAD_LETTER_NONZERO", False)
    monkeypatch.setattr("app.config.DIMSE_INGEST_PENDING_AGE_WARN_SECONDS", 30)
    monkeypatch.setattr("app.config.DIMSE_DEAD_LETTER_AGE_WARN_SECONDS", 60)

    snapshot = {
        "pending": 1,
        "dead_letter": 1,
        "pending_oldest_age_seconds": 35,
        "dead_letter_oldest_age_seconds": 80,
    }
    result = evaluate_retry_alerts(snapshot, now=200.0)
    assert result == {"active": 2, "emitted": 2}

    all_alerts = get_alerts()
    assert all_alerts["total"] == 2

    pending_only = get_alerts(condition="pending_age_threshold_exceeded")
    assert pending_only["total"] == 1
    assert pending_only["items"][0]["condition"] == "pending_age_threshold_exceeded"

    dead_only = get_alerts(condition="dead_letter_age_threshold_exceeded")
    assert dead_only["total"] == 1
    assert dead_only["items"][0]["condition"] == "dead_letter_age_threshold_exceeded"


def test_alert_max_events_retention(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERTS_ENABLED", True)
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERT_COOLDOWN_SECONDS", 0)
    monkeypatch.setattr("app.config.DIMSE_RETRY_ALERT_MAX_EVENTS", 2)

    snapshot = {"pending": 0, "dead_letter": 1}
    evaluate_retry_alerts(snapshot, now=1.0)
    evaluate_retry_alerts(snapshot, now=2.0)
    evaluate_retry_alerts(snapshot, now=3.0)

    alerts = get_alerts()
    assert alerts["total"] == 2
    assert alerts["items"][0]["timestamp"] == 3
    assert alerts["items"][1]["timestamp"] == 2

