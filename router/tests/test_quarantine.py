from app.quarantine import QuarantineStore, default_db_path


def test_quarantine_and_list(tmp_quarantine_dir):
    store = QuarantineStore(default_db_path(tmp_quarantine_dir))
    store.quarantine(
        study_uid="1.2.3",
        reason="de-id failed",
        failed_stage="defacing",
        error="model crashed",
        detail={"stages_run": ["tag_deid"]},
        file_count=42,
    )
    items = store.list()
    assert len(items) == 1
    e = items[0]
    assert e.study_uid == "1.2.3"
    assert e.failed_stage == "defacing"
    assert e.detail == {"stages_run": ["tag_deid"]}
    assert e.file_count == 42
    assert e.attempts == 0


def test_quarantine_upsert_preserves_quarantined_at(tmp_quarantine_dir):
    store = QuarantineStore(default_db_path(tmp_quarantine_dir))
    store.quarantine("1.2", "r1", "stage1", "e1", file_count=10)
    first = store.get("1.2")
    assert first is not None

    store.quarantine("1.2", "r2", "stage2", "e2", file_count=10)
    second = store.get("1.2")
    assert second is not None
    assert second.quarantined_at == first.quarantined_at
    assert second.failed_stage == "stage2"


def test_clear_removes_entry(tmp_quarantine_dir):
    store = QuarantineStore(default_db_path(tmp_quarantine_dir))
    store.quarantine("1.2", "r", "s", "e")
    assert store.clear("1.2") is True
    assert store.get("1.2") is None
    assert store.clear("1.2") is False


def test_mark_attempted_increments_count(tmp_quarantine_dir):
    store = QuarantineStore(default_db_path(tmp_quarantine_dir))
    store.quarantine("1.2", "r", "s", "e")
    store.mark_attempted("1.2")
    store.mark_attempted("1.2")
    entry = store.get("1.2")
    assert entry is not None
    assert entry.attempts == 2
    assert entry.last_attempt_at is not None
