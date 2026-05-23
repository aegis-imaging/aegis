import time

from dimse_core.alerts import AlertConfig, AlertEngine


def test_condition_fires_on_truthy_predicate():
    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition("dl", lambda s: s.get("dead_letter", 0) > 0, "dead-letter has {dead_letter}")
    events = eng.evaluate({"dead_letter": 3})
    assert len(events) == 1
    assert events[0].condition == "dl"
    assert "3" in events[0].message


def test_cooldown_blocks_repeated_emission():
    eng = AlertEngine(AlertConfig(cooldown_seconds=60))
    eng.add_condition("dl", lambda s: s["dl"] > 0, "x")
    first = eng.evaluate({"dl": 1}, now=1000)
    second = eng.evaluate({"dl": 1}, now=1030)  # within cooldown
    assert len(first) == 1
    assert len(second) == 0


def test_cooldown_expires():
    eng = AlertEngine(AlertConfig(cooldown_seconds=60))
    eng.add_condition("dl", lambda s: s["dl"] > 0, "x")
    eng.evaluate({"dl": 1}, now=1000)
    later = eng.evaluate({"dl": 1}, now=1090)  # past cooldown
    assert len(later) == 1


def test_multiple_conditions_independent():
    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition("a", lambda s: s.get("a", 0) > 0, "a fires")
    eng.add_condition("b", lambda s: s.get("b", 0) > 0, "b fires")
    events = eng.evaluate({"a": 1, "b": 0})
    assert [e.condition for e in events] == ["a"]
    events = eng.evaluate({"a": 0, "b": 1})
    assert [e.condition for e in events] == ["b"]


def test_webhook_called_per_event():
    calls = []
    eng = AlertEngine(AlertConfig(cooldown_seconds=0, webhook_fn=calls.append))
    eng.add_condition("x", lambda s: True, "x")
    eng.add_condition("y", lambda s: True, "y")
    eng.evaluate({})
    assert len(calls) == 2
    assert {c["condition"] for c in calls} == {"x", "y"}


def test_recent_returns_newest_first():
    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition("a", lambda s: True, "a")
    eng.evaluate({}, now=1)
    eng.evaluate({}, now=2)
    recent = eng.recent()
    assert recent[0]["timestamp"] == 2


def test_max_retained_caps_history():
    eng = AlertEngine(AlertConfig(cooldown_seconds=0, max_retained=2))
    eng.add_condition("a", lambda s: True, "a")
    for i in range(5):
        eng.evaluate({}, now=i)
    assert len(eng.recent()) == 2


def test_predicate_exception_skipped():
    def explode(s):
        raise RuntimeError("boom")

    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition("explode", explode, "should not fire")
    eng.add_condition("ok", lambda s: True, "ok")
    events = eng.evaluate({})
    assert [e.condition for e in events] == ["ok"]


def test_message_fn_builds_dynamic_message_with_external_values():
    """message_fn closes over values that aren't in the snapshot.

    The receiver's retry alerts use this to include the configured
    threshold in the alert message (the threshold lives in env vars, not
    the snapshot dict).
    """
    threshold = 600
    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition_fn(
        "pending_age",
        lambda s: s.get("pending_oldest_age_seconds", 0) >= threshold,
        lambda s: f"Oldest pending {s['pending_oldest_age_seconds']}s exceeds threshold {threshold}s",
    )
    events = eng.evaluate({"pending_oldest_age_seconds": 800})
    assert len(events) == 1
    assert events[0].message == "Oldest pending 800s exceeds threshold 600s"


def test_message_fn_takes_precedence_over_message_fmt():
    """If a condition somehow has both, message_fn wins (its presence is the signal)."""
    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition_fn(
        "x",
        lambda s: True,
        lambda s: "from fn",
    )
    events = eng.evaluate({})
    assert events[0].message == "from fn"


def test_message_fn_exception_falls_back_to_condition_name():
    def boom(s):
        raise RuntimeError("explode")

    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition_fn("flaky", lambda s: True, boom)
    events = eng.evaluate({})
    assert len(events) == 1
    assert events[0].message == "flaky"  # falls back to condition name


def test_clear_conditions_preserves_cooldown_state():
    """Re-adding a condition by the same name keeps its cooldown history."""
    eng = AlertEngine(AlertConfig(cooldown_seconds=60))
    eng.add_condition("dl", lambda s: True, "x")
    first = eng.evaluate({}, now=1000)
    assert len(first) == 1

    # Wipe and re-add the same-named condition (e.g., config-driven rebuild).
    eng.clear_conditions()
    eng.add_condition("dl", lambda s: True, "x")
    blocked = eng.evaluate({}, now=1030)  # within cooldown of first
    assert len(blocked) == 0


def test_clear_conditions_does_not_clear_events():
    """Stored alert history survives a conditions rebuild."""
    eng = AlertEngine(AlertConfig(cooldown_seconds=0))
    eng.add_condition("a", lambda s: True, "a")
    eng.evaluate({}, now=1)
    eng.clear_conditions()
    assert len(eng.recent()) == 1
