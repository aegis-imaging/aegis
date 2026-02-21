"""Tests for app.operator_audit."""

from __future__ import annotations

from app.operator_audit import get_actions, record_action, reset_actions


def setup_function():
    reset_actions()


def test_record_action_and_order():
    record_action("a1", value=1)
    record_action("a2", value=2)
    out = get_actions(limit=10)
    assert out["total"] == 2
    assert out["items"][0]["action"] == "a2"
    assert out["items"][1]["action"] == "a1"


def test_get_actions_limit():
    for i in range(5):
        record_action(f"a{i}", i=i)
    out = get_actions(limit=2)
    assert out["total"] == 5
    assert len(out["items"]) == 2
    assert out["items"][0]["action"] == "a4"
    assert out["items"][1]["action"] == "a3"


def test_get_actions_filters_by_action_name():
    record_action("retry_process", value=1)
    record_action("retry_replay_bulk", value=2)
    record_action("retry_process", value=3)

    out = get_actions(limit=10, action="retry_process")
    assert out["total"] == 2
    assert [x["action"] for x in out["items"]] == ["retry_process", "retry_process"]


def test_operator_audit_max_retention(monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_OPERATOR_AUDIT_MAX", 3)
    for i in range(5):
        record_action(f"a{i}", i=i)
    out = get_actions(limit=10)
    assert out["total"] == 3
    assert [x["action"] for x in out["items"]] == ["a4", "a3", "a2"]
