"""Tests for app.main FastAPI health endpoint and lifespan."""

from __future__ import annotations

from fastapi.testclient import TestClient

from app.ingest import reset_retry_state
from app.main import app
from app.operator_audit import reset_actions


class _DummyAE:
    def __init__(self, active_associations):
        self.active_associations = active_associations
        self.shutdown_called = False

    def shutdown(self):
        self.shutdown_called = True


def setup_function():
    reset_retry_state()
    reset_actions()
    import app.config as cfg

    cfg.DIMSE_OPERATOR_API_KEY = ""


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


def test_retry_endpoints_require_operator_key_when_configured(monkeypatch):
    from unittest.mock import patch

    monkeypatch.setattr("app.config.DIMSE_OPERATOR_API_KEY", "secret")
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ):
        with TestClient(app) as client:
            resp = client.get("/ingest/retry")

    assert resp.status_code == 401
    assert "operator auth required" in resp.json()["detail"]


def test_retry_endpoints_accept_operator_key_header(monkeypatch):
    from unittest.mock import patch

    monkeypatch.setattr("app.config.DIMSE_OPERATOR_API_KEY", "secret")
    actions = {"total": 1, "items": [{"action": "retry_process"}]}
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.get_actions", return_value=actions):
        with TestClient(app) as client:
            resp = client.get(
                "/ingest/retry/actions?limit=1",
                headers={"x-aegis-operator-key": "secret"},
            )

    assert resp.status_code == 200
    assert resp.json() == {"status": "ok", "actions": actions}


def test_retry_endpoints_accept_operator_bearer(monkeypatch):
    from unittest.mock import patch

    monkeypatch.setattr("app.config.DIMSE_OPERATOR_API_KEY", "secret")
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ):
        with TestClient(app) as client:
            resp = client.get(
                "/ingest/retry",
                headers={"authorization": "Bearer secret"},
            )

    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"


def test_ingest_retry_actions_endpoint():
    from unittest.mock import patch

    actions = {"total": 1, "items": [{"action": "retry_process"}]}
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.get_actions", return_value=actions) as mock_get:
        with TestClient(app) as client:
            resp = client.get("/ingest/retry/actions?limit=5")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "actions": actions}
    mock_get.assert_called_once_with(limit=5)


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

    snap = {"pending": 0, "dead_letter": 0}
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.process_retry_queue", return_value=2), patch(
        "app.main.retry_snapshot", return_value=snap
    ), patch("app.main.record_action") as mock_record:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/process")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert body["processed"] == 2
    assert "ingest_retry" in body
    mock_record.assert_called_once()


def test_ingest_retry_process_study_endpoint():
    from unittest.mock import patch

    result = {"study_instance_uid": "1.2.3.4", "found": True, "attempted": True, "result": "ok"}
    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.process_retry_study", return_value=result) as mock_process:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/process/1.2.3.4")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": result}
    mock_process.assert_called_once_with(study_instance_uid="1.2.3.4")


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


def test_ingest_retry_clear_pending_endpoint():
    from unittest.mock import patch

    snap = {"pending": 0, "cleared_now": 3}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.clear_pending", return_value=snap) as mock_clear:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/clear-pending?limit=3")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": snap}
    mock_clear.assert_called_once_with(limit=3)


def test_ingest_retry_clear_pending_study_endpoint():
    from unittest.mock import patch

    result = {"study_instance_uid": "1.2.3.4", "found": True, "cleared": 1}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.clear_pending_study", return_value=result) as mock_clear:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/clear-pending/1.2.3.4")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": result}
    mock_clear.assert_called_once_with(study_instance_uid="1.2.3.4")


def test_ingest_retry_clear_dead_letter_study_endpoint():
    from unittest.mock import patch

    result = {"study_instance_uid": "1.2.3.4", "found": True, "cleared": 1}

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ), patch("app.main.clear_dead_letter_study", return_value=result) as mock_clear:
        with TestClient(app) as client:
            resp = client.post("/ingest/retry/clear-dead-letter/1.2.3.4")

    assert resp.status_code == 200
    body = resp.json()
    assert body == {"status": "ok", "ingest_retry": result}
    mock_clear.assert_called_once_with(study_instance_uid="1.2.3.4")


def test_retry_control_endpoints_emit_action_records():
    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=_DummyAE(active_associations=[])), patch(
        "app.main.start_scp", return_value=None
    ):
        with TestClient(app) as client:
            client.post("/ingest/retry/process")
            client.post("/ingest/retry/process/abc")
            client.post("/ingest/retry/replay?limit=1")
            client.post("/ingest/retry/replay/abc")
            client.post("/ingest/retry/clear-dead-letter?limit=1")
            client.post("/ingest/retry/clear-dead-letter/abc")
            client.post("/ingest/retry/clear-pending?limit=1")
            client.post("/ingest/retry/clear-pending/abc")
            actions_resp = client.get("/ingest/retry/actions?limit=10")

    assert actions_resp.status_code == 200
    items = actions_resp.json()["actions"]["items"]
    names = [x["action"] for x in items]
    assert "retry_process" in names
    assert "retry_process_study" in names
    assert "retry_replay_bulk" in names
    assert "retry_replay_study" in names
    assert "retry_clear_dead_letter" in names
    assert "retry_clear_dead_letter_study" in names
    assert "retry_clear_pending" in names
    assert "retry_clear_pending_study" in names
