"""Endpoint tests for qc-service."""

from unittest.mock import patch, MagicMock

from fastapi.testclient import TestClient

from app.backends.base import CheckResult, QCBackend, QCResult


# ---------------------------------------------------------------------------
# Helper: mock backend
# ---------------------------------------------------------------------------


def _make_mock_backend(name: str = "basic", available: bool = True) -> MagicMock:
    backend = MagicMock(spec=QCBackend)
    backend.name = name
    backend.available.return_value = available
    backend.check.return_value = QCResult(
        checks=[
            CheckResult("file_integrity", "pass", "All files OK"),
            CheckResult("slice_consistency", "pass", "Consistent"),
            CheckResult("snr", "pass", "SNR OK"),
            CheckResult("coverage", "pass", "Adequate"),
            CheckResult("missing_slices", "pass", "No gaps"),
        ],
        overall="pass",
    )
    return backend


# ---------------------------------------------------------------------------
# /healthz
# ---------------------------------------------------------------------------


class TestHealthz:
    def test_healthz_ok(self):
        mock_backend = _make_mock_backend()
        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.get("/healthz")
        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "ok"
        assert body["backend"] == "basic"
        assert body["available"] is True

    def test_healthz_degraded(self):
        with patch("app.main._backend", None), \
             patch("app.main.get_backend", side_effect=RuntimeError("No backend")):
            from app.main import app
            client = TestClient(app)
            resp = client.get("/healthz")
        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "degraded"


# ---------------------------------------------------------------------------
# /check
# ---------------------------------------------------------------------------


class TestCheck:
    def test_check_success(self, dicom_study):
        mock_backend = _make_mock_backend()
        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/check", json={
                "study_uid": "1.2.3",
                "input_paths": dicom_study,
            })
        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["overall_quality"] == "pass"
        assert body["tool_used"] == "basic"
        assert len(body["quality_issues"]) == 5

    def test_check_empty_paths(self):
        from app.main import app
        client = TestClient(app)
        resp = client.post("/check", json={
            "study_uid": "1.2.3",
            "input_paths": [],
        })
        assert resp.status_code == 400

    def test_check_missing_files(self):
        from app.main import app
        client = TestClient(app)
        resp = client.post("/check", json={
            "study_uid": "1.2.3",
            "input_paths": ["/nonexistent/file.dcm"],
        })
        assert resp.status_code == 400

    def test_check_backend_error(self, dicom_study):
        mock_backend = _make_mock_backend()
        mock_backend.check.side_effect = RuntimeError("Backend crashed")
        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/check", json={
                "study_uid": "1.2.3",
                "input_paths": dicom_study,
            })
        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert "Backend crashed" in body["error"]
