import json
from pathlib import Path

from app.audit import AuditLog


def test_record_and_recent(tmp_path):
    log = AuditLog(str(tmp_path / "audit.log"))
    log.record("pipeline.start", study_uid="1.2", source="dimse")
    log.record("pipeline.shipped", study_uid="1.2", cloud_study_id="cs-1")
    entries = log.recent(limit=10)
    # Newest first.
    assert entries[0]["event"] == "pipeline.shipped"
    assert entries[1]["event"] == "pipeline.start"
    assert entries[0]["cloud_study_id"] == "cs-1"


def test_recent_event_filter(tmp_path):
    log = AuditLog(str(tmp_path / "audit.log"))
    log.record("pipeline.start", study_uid="1.2")
    log.record("pipeline.quarantined", study_uid="1.2")
    log.record("pipeline.start", study_uid="3.4")
    starts = log.recent(event="pipeline.start")
    assert len(starts) == 2
    assert all(e["event"] == "pipeline.start" for e in starts)


def test_persisted_to_disk(tmp_path):
    path = tmp_path / "audit.log"
    log1 = AuditLog(str(path))
    log1.record("router.start", site="example")
    # Re-open with a fresh instance; it should rehydrate from disk.
    log2 = AuditLog(str(path))
    entries = log2.recent()
    assert entries[0]["event"] == "router.start"
    assert entries[0]["site"] == "example"


def test_disk_lines_are_valid_jsonl(tmp_path):
    path = tmp_path / "audit.log"
    log = AuditLog(str(path))
    log.record("a", x=1)
    log.record("b", y=2)
    raw = Path(path).read_text().splitlines()
    assert len(raw) == 2
    decoded = [json.loads(line) for line in raw]
    assert decoded[0]["event"] == "a"
    assert decoded[1]["event"] == "b"
