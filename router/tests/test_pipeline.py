import sys

import pytest

from app import deid, quarantine, shipper, storage
from app.audit import AuditLog
from app.config import RouterConfig
from app.pipeline import Orchestrator
from tests.conftest import make_dicom


def _seed(tmp_data_dir, study_uid):
    paths = storage.layout(tmp_data_dir, study_uid)
    for i in range(2):
        storage.write_raw_bytes(paths, i, make_dicom(study_uid=study_uid))
    return paths


def _orchestrator(base_env, tmp_quarantine_dir, tmp_path):
    cfg = RouterConfig.load()
    store = quarantine.QuarantineStore(quarantine.default_db_path(tmp_quarantine_dir))
    audit = AuditLog(str(tmp_path / "audit.log"))
    return cfg, store, audit, Orchestrator(cfg, store, audit)


def test_run_sync_local_only_when_no_cloud(base_env, monkeypatch, tmp_data_dir, tmp_quarantine_dir, tmp_path):
    monkeypatch.setitem(sys.modules, "midi_b", None)
    _seed(tmp_data_dir, "1.2.10")
    _, store, _, orch = _orchestrator(base_env, tmp_quarantine_dir, tmp_path)
    result = orch.run_sync("1.2.10", file_count=2, source="test")
    assert result.ok
    assert "tag_deid" in result.stages_run
    assert result.quarantined is False
    assert store.get("1.2.10") is None  # not quarantined


def test_quarantine_on_phi_failure(base_env, monkeypatch, tmp_data_dir, tmp_quarantine_dir, tmp_path):
    monkeypatch.setitem(sys.modules, "midi_b", None)
    monkeypatch.setenv("PHI_DETECTION_SERVICE_URL", "http://phi:8080")
    _seed(tmp_data_dir, "1.2.11")

    class _R:
        status_code = 500
        text = "boom"

        def raise_for_status(self):
            raise RuntimeError("HTTP 500: boom")

        def json(self):
            return {}

    monkeypatch.setattr("httpx.Client.post", lambda *a, **k: _R())

    _, store, _, orch = _orchestrator(base_env, tmp_quarantine_dir, tmp_path)
    result = orch.run_sync("1.2.11")
    assert not result.ok
    assert result.quarantined
    entry = store.get("1.2.11")
    assert entry is not None
    assert entry.failed_stage == "phi_scrub"


def test_quarantine_on_ship_failure(base_env, monkeypatch, tmp_data_dir, tmp_quarantine_dir, tmp_path):
    cert = tmp_path / "c"
    key = tmp_path / "k"
    cert.write_text("x")
    key.write_text("x")
    monkeypatch.setenv("CLOUD_RECEIVER_URL", "https://cloud.example.com")
    monkeypatch.setenv("CLOUD_CLIENT_CERT", str(cert))
    monkeypatch.setenv("CLOUD_CLIENT_KEY", str(key))
    monkeypatch.setitem(sys.modules, "midi_b", None)
    _seed(tmp_data_dir, "1.2.12")

    def fake_ship(cfg, paths, metadata):
        raise shipper.ShipperError("cloud unreachable")

    monkeypatch.setattr("app.pipeline.shipper.ship_study", fake_ship)

    _, store, _, orch = _orchestrator(base_env, tmp_quarantine_dir, tmp_path)
    result = orch.run_sync("1.2.12")
    assert not result.ok
    assert result.quarantined
    entry = store.get("1.2.12")
    assert entry is not None
    assert entry.failed_stage == "ship"
    assert "cloud unreachable" in entry.error


def test_happy_ship(base_env, monkeypatch, tmp_data_dir, tmp_quarantine_dir, tmp_path):
    cert = tmp_path / "c"
    key = tmp_path / "k"
    cert.write_text("x")
    key.write_text("x")
    monkeypatch.setenv("CLOUD_RECEIVER_URL", "https://cloud.example.com")
    monkeypatch.setenv("CLOUD_CLIENT_CERT", str(cert))
    monkeypatch.setenv("CLOUD_CLIENT_KEY", str(key))
    monkeypatch.setitem(sys.modules, "midi_b", None)
    _seed(tmp_data_dir, "1.2.13")

    def fake_ship(cfg, paths, metadata):
        return shipper.ShipResult(cloud_session_id="s1", cloud_study_id="cs-9", files_uploaded=2)

    monkeypatch.setattr("app.pipeline.shipper.ship_study", fake_ship)

    _, store, _, orch = _orchestrator(base_env, tmp_quarantine_dir, tmp_path)
    result = orch.run_sync("1.2.13")
    assert result.ok
    assert result.cloud_study_id == "cs-9"
    assert "ship" in result.stages_run
    assert store.get("1.2.13") is None
