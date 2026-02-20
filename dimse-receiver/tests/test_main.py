"""Tests for app.main FastAPI health endpoint and lifespan."""

from __future__ import annotations

from fastapi.testclient import TestClient

from app.main import app


class _DummyAE:
    def __init__(self, active_associations):
        self.active_associations = active_associations
        self.shutdown_called = False

    def shutdown(self):
        self.shutdown_called = True


def test_healthz_ok_with_running_scp():
    dummy = _DummyAE(active_associations=[])

    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=dummy), patch("app.main.start_scp", return_value=None):
        with TestClient(app) as client:
            resp = client.get("/healthz")
            assert resp.status_code == 200
            assert resp.json() == {"status": "ok", "scp": "running"}

    assert dummy.shutdown_called is True


def test_healthz_degraded_when_scp_not_running():
    dummy = _DummyAE(active_associations=None)

    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=dummy), patch("app.main.start_scp", return_value=None):
        with TestClient(app) as client:
            resp = client.get("/healthz")
            assert resp.status_code == 200
            assert resp.json() == {"status": "degraded", "scp": "not_running"}


def test_forward_success():
    from unittest.mock import patch

    payload = {
        "study_instance_uid": "1.2.3",
        "dicom_store": "raw",
        "destination": {"ae_title": "REMOTE_AE", "host": "10.0.0.8", "port": 104},
    }
    result = {"files_total": 2, "files_sent": 2, "files_failed": 0}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.forward_study", return_value=result) as mock_forward:
        with TestClient(app) as client:
            resp = client.post("/forward", json=payload)

    assert resp.status_code == 200
    assert resp.json() == {"status": "complete", **result}
    mock_forward.assert_called_once_with(
        study_uid="1.2.3",
        dicom_store="raw",
        host="10.0.0.8",
        port=104,
        ae_title="REMOTE_AE",
    )


def test_forward_file_not_found_maps_to_404():
    from unittest.mock import patch

    payload = {
        "study_instance_uid": "1.2.3",
        "destination": {"ae_title": "REMOTE_AE", "host": "10.0.0.8", "port": 104},
    }
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.forward_study", side_effect=FileNotFoundError("missing study")):
        with TestClient(app) as client:
            resp = client.post("/forward", json=payload)

    assert resp.status_code == 404
    assert "missing study" in resp.json()["detail"]


def test_forward_runtime_error_maps_to_502():
    from unittest.mock import patch

    payload = {
        "study_instance_uid": "1.2.3",
        "destination": {"ae_title": "REMOTE_AE", "host": "10.0.0.8", "port": 104},
    }
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.forward_study", side_effect=RuntimeError("association failed")):
        with TestClient(app) as client:
            resp = client.post("/forward", json=payload)

    assert resp.status_code == 502
    assert "association failed" in resp.json()["detail"]
