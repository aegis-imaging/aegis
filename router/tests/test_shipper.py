import sys

import pytest

from app import deid, shipper, storage
from app.config import RouterConfig
from tests.conftest import make_dicom


def _prep_study(tmp_data_dir, study_uid="1.2.9"):
    paths = storage.layout(tmp_data_dir, study_uid)
    for i in range(2):
        storage.write_raw_bytes(paths, i, make_dicom(study_uid=study_uid))
    return paths


def test_ship_requires_cloud_config(base_env, tmp_data_dir):
    cfg = RouterConfig.load()
    paths = _prep_study(tmp_data_dir)
    with pytest.raises(shipper.ShipperError, match="cloud forwarding not configured"):
        shipper.ship_study(cfg, paths, {"study_instance_uid": "1.2.9"})


def test_ship_happy_path(base_env, monkeypatch, tmp_data_dir, tmp_path):
    cert = tmp_path / "client.crt"
    key = tmp_path / "client.key"
    cert.write_text("x")
    key.write_text("x")
    monkeypatch.setenv("CLOUD_RECEIVER_URL", "https://cloud.example.com")
    monkeypatch.setenv("CLOUD_CLIENT_CERT", str(cert))
    monkeypatch.setenv("CLOUD_CLIENT_KEY", str(key))
    monkeypatch.setenv("CLOUD_PROJECT_SLUG", "alpha")
    monkeypatch.setenv("CLOUD_INSTITUTION_SLUG", "sample-hospital")

    cfg = RouterConfig.load()
    monkeypatch.setitem(sys.modules, "midi_b", None)
    paths = _prep_study(tmp_data_dir)
    deid.run_pipeline(cfg, paths)  # populate deid/

    calls = {"init": 0, "puts": 0, "complete": 0}

    class _R:
        def __init__(self, status_code=200, body=None):
            self.status_code = status_code
            self._body = body or {}
            self.text = ""

        def json(self):
            return self._body

    def fake_post(self, url, json=None, **kw):
        if url.endswith("/api/upload/init"):
            calls["init"] += 1
            return _R(body={"session_id": "sess1", "upload_urls": ["/u/0", "/u/1"]})
        if url.endswith("/api/upload/complete"):
            calls["complete"] += 1
            return _R(body={"study_id": "study-xyz"})
        raise AssertionError(f"unexpected POST: {url}")

    def fake_put(self, url, content=None, headers=None, **kw):
        calls["puts"] += 1
        assert url.startswith("/u/")
        return _R()

    monkeypatch.setattr("httpx.Client.post", fake_post)
    monkeypatch.setattr("httpx.Client.put", fake_put)

    result = shipper.ship_study(cfg, paths, {"study_instance_uid": "1.2.9", "modality": "MR"})
    assert result.cloud_session_id == "sess1"
    assert result.cloud_study_id == "study-xyz"
    assert result.files_uploaded == 2
    assert calls == {"init": 1, "puts": 2, "complete": 1}


def test_ship_init_error_raises(base_env, monkeypatch, tmp_data_dir, tmp_path):
    cert = tmp_path / "c"
    key = tmp_path / "k"
    cert.write_text("x")
    key.write_text("x")
    monkeypatch.setenv("CLOUD_RECEIVER_URL", "https://cloud.example.com")
    monkeypatch.setenv("CLOUD_CLIENT_CERT", str(cert))
    monkeypatch.setenv("CLOUD_CLIENT_KEY", str(key))
    cfg = RouterConfig.load()
    monkeypatch.setitem(sys.modules, "midi_b", None)
    paths = _prep_study(tmp_data_dir)
    deid.run_pipeline(cfg, paths)

    class _R:
        status_code = 403
        text = "unauthorized spoke"

        def json(self):
            return {}

    monkeypatch.setattr("httpx.Client.post", lambda *a, **k: _R())
    with pytest.raises(shipper.ShipperError, match="upload/init failed: HTTP 403"):
        shipper.ship_study(cfg, paths, {"study_instance_uid": "1.2.9"})
