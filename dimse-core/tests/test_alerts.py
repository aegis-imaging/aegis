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
