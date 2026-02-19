"""
Tests for app.main — FastAPI endpoint tests using TestClient.

Module-level globals (app.main._backend, app.main.run_pipeline) are patched
so tests do not require any external defacing tools.
"""

from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import app


# ---------------------------------------------------------------------------
# /healthz
# ---------------------------------------------------------------------------

class TestHealthz:
    """Tests for GET /healthz."""

    def test_healthz_ok(self):
        """When a backend is available, /healthz returns status=ok."""
        mock_backend = MagicMock()
        mock_backend.name = "mock-backend"
        mock_backend.available.return_value = True

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            client = TestClient(app)
            resp = client.get("/healthz")

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "ok"
        assert body["backend"] == "mock-backend"
        assert body["available"] is True

    def test_healthz_degraded(self):
        """When no backend is available, /healthz returns status=degraded."""
        with patch("app.main._backend", None), \
             patch("app.main.get_backend", side_effect=RuntimeError("no backend")):
            client = TestClient(app)
            resp = client.get("/healthz")

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "degraded"
        assert "error" in body


# ---------------------------------------------------------------------------
# POST /deface
# ---------------------------------------------------------------------------

class TestDeface:
    """Tests for POST /deface."""

    def test_deface_success(self, tmp_path):
        """
        Successful deface returns status=complete with output_paths and tool_used.
        """
        # Create dummy input files so the path-existence check passes
        input_files = []
        for i in range(3):
            p = tmp_path / f"{i:04d}.dcm"
            p.write_bytes(b"\x00" * 128)
            input_files.append(str(p))

        output_dir = str(tmp_path / "output")

        mock_backend = MagicMock()
        mock_backend.name = "mock-backend"
        mock_backend.available.return_value = True

        mock_output_paths = [f"{output_dir}/{i:04d}.dcm" for i in range(3)]

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main.run_pipeline", return_value=mock_output_paths) as mock_pipeline:

            client = TestClient(app)
            resp = client.post("/deface", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": input_files,
                "output_dir": output_dir,
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["study_uid"] == "1.2.3.4.5"
        assert body["tool_used"] == "mock-backend"
        assert body["output_paths"] == mock_output_paths
        assert body["duration_seconds"] >= 0
        assert body["error"] is None

        # Verify run_pipeline was called with correct args
        mock_pipeline.assert_called_once_with(input_files, output_dir, mock_backend)

    def test_deface_empty_input_paths(self):
        """POST with empty input_paths returns HTTP 400."""
        mock_backend = MagicMock()
        mock_backend.name = "mock-backend"

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):

            client = TestClient(app)
            resp = client.post("/deface", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [],
                "output_dir": "/tmp/output",
            })

        assert resp.status_code == 400
        body = resp.json()
        assert "input_paths" in body["detail"].lower()

    def test_deface_missing_files(self, tmp_path):
        """POST with input paths that don't exist returns HTTP 400."""
        mock_backend = MagicMock()
        mock_backend.name = "mock-backend"

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):

            client = TestClient(app)
            resp = client.post("/deface", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": ["/nonexistent/path/file.dcm"],
                "output_dir": str(tmp_path / "output"),
            })

        assert resp.status_code == 400
        body = resp.json()
        assert "not found" in body["detail"].lower()

    def test_deface_no_backend(self, tmp_path):
        """
        When no backend is available (get_backend raises), the endpoint
        returns HTTP 500 because the RuntimeError is unhandled.
        """
        # Create a dummy input file so path validation passes
        input_file = tmp_path / "test.dcm"
        input_file.write_bytes(b"\x00" * 128)

        with patch("app.main._backend", None), \
             patch("app.main.get_backend", side_effect=RuntimeError("No defacing backend")):

            # raise_server_exceptions=False so TestClient returns the 500
            # instead of re-raising the RuntimeError in the test process.
            client = TestClient(app, raise_server_exceptions=False)
            resp = client.post("/deface", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(input_file)],
                "output_dir": str(tmp_path / "output"),
            })

        assert resp.status_code == 500

    def test_deface_pipeline_failure(self, tmp_path):
        """
        When run_pipeline raises an exception, the endpoint returns
        status=failed with the error message (not HTTP 500).
        """
        input_file = tmp_path / "test.dcm"
        input_file.write_bytes(b"\x00" * 128)
        output_dir = str(tmp_path / "output")

        mock_backend = MagicMock()
        mock_backend.name = "mock-backend"
        mock_backend.available.return_value = True

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main.run_pipeline", side_effect=RuntimeError("pipeline exploded")):

            client = TestClient(app)
            resp = client.post("/deface", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(input_file)],
                "output_dir": output_dir,
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert body["error"] == "pipeline exploded"
        assert body["output_paths"] == []
        assert body["tool_used"] == "mock-backend"
