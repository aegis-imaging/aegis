"""Tests for app.main FastAPI health endpoint and lifespan."""

from __future__ import annotations

from fastapi.testclient import TestClient

from app.ingest import reset_retry_state
from app.main import app


class _DummyAE:
    def __init__(self, active_associations):
        self.active_associations = active_associations
        self.shutdown_called = False

    def shutdown(self):
        self.shutdown_called = True


def setup_function():
    reset_retry_state()


def test_healthz_ok_with_running_scp():
    dummy = _DummyAE(active_associations=[])

    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=dummy), patch("app.main.start_scp", return_value=None):
        with TestClient(app) as client:
            resp = client.get("/healthz")
            assert resp.status_code == 200
            data = resp.json()
            assert data["status"] == "ok"
            assert data["scp"] == "running"
            assert "ingest_retry" in data

    assert dummy.shutdown_called is True


def test_healthz_degraded_when_scp_not_running():
    dummy = _DummyAE(active_associations=None)

    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=dummy), patch("app.main.start_scp", return_value=None):
        with TestClient(app) as client:
            resp = client.get("/healthz")
            assert resp.status_code == 200
            data = resp.json()
            assert data["status"] == "degraded"
            assert data["scp"] == "not_running"
            assert "ingest_retry" in data


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


def test_ingest_retry_status_endpoint():
    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ):
        with TestClient(app) as client:
            resp = client.get("/ingest/retry")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert "ingest_retry" in body


def test_ingest_retry_details_endpoint():
    from unittest.mock import patch

    details = {"snapshot": {"pending": 0, "dead_letter": 0}, "pending_items": [], "dead_letter_items": []}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.retry_details", return_value=details) as mock_details:
        with TestClient(app) as client:
            resp = client.get("/ingest/retry/details?limit=10")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": details}
    mock_details.assert_called_once_with(limit=10)


def test_ingest_retry_process_endpoint():
    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.process_retry_queue", return_value=2):
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/process")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert body["processed"] == 2
    assert "ingest_retry" in body


def test_ingest_retry_replay_endpoint():
    from unittest.mock import patch

    snap = {"pending": 1, "dead_letter": 0, "replayed_now": 1}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.replay_dead_letter", return_value=snap) as mock_replay:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/replay?limit=5")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": snap}
    mock_replay.assert_called_once_with(limit=5)


def test_ingest_retry_replay_study_endpoint():
    from unittest.mock import patch

    result = {"study_instance_uid": "1.2.3.4", "found": True, "moved": 1, "blocked_by_queue_full": False}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.replay_dead_letter_study", return_value=result) as mock_replay:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/replay/1.2.3.4")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": result}
    mock_replay.assert_called_once_with(study_instance_uid="1.2.3.4")


def test_ingest_retry_clear_dead_letter_endpoint():
    from unittest.mock import patch

    snap = {"dead_letter": 0, "cleared_now": 3}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.clear_dead_letter", return_value=snap) as mock_clear:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/clear-dead-letter?limit=3")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": snap}
    mock_clear.assert_called_once_with(limit=3)
