from dimse_core.audit import AuditLog


def test_record_and_recent():
    log = AuditLog(max_entries=10)
    log.record("a", x=1)
    log.record("b", y=2)
    entries = log.recent()
    assert entries[0]["event"] == "b"
    assert entries[1]["event"] == "a"


def test_event_filter():
    log = AuditLog(max_entries=10)
    log.record("a")
    log.record("b")
    log.record("a")
    a_only = log.recent(event="a")
    assert len(a_only) == 2
    assert all(e["event"] == "a" for e in a_only)


def test_bounded_ring():
    log = AuditLog(max_entries=3)
    for i in range(5):
        log.record("e", i=i)
    assert log.size() == 3
    entries = log.recent()
    assert [e["i"] for e in entries] == [4, 3, 2]


def test_persist_and_reload(tmp_path):
    path = tmp_path / "audit.log"
    log1 = AuditLog(max_entries=10, persist_path=str(path))
    log1.record("alpha")
    log1.record("beta")
    log2 = AuditLog(max_entries=10, persist_path=str(path))
    assert [e["event"] for e in log2.recent()] == ["beta", "alpha"]


def test_clear():
    log = AuditLog(max_entries=10)
    log.record("a")
    log.clear()
    assert log.size() == 0
