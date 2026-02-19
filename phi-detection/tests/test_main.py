"""Endpoint tests for the phi-detection FastAPI service.

Uses httpx + TestClient. Mocks the backend at the module level to avoid
requiring Tesseract to be installed in the test environment.
"""

from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.backends.base import PHIDetectionBackend, FileFinding, Region


# ---------------------------------------------------------------------------
# Helper: build a mock backend
# ---------------------------------------------------------------------------

def _make_mock_backend(name: str = "tesseract", available: bool = True) -> MagicMock:
    backend = MagicMock(spec=PHIDetectionBackend)
    backend.name = name
    backend.available.return_value = available
    backend.detect.return_value = []
    return backend


# ---------------------------------------------------------------------------
# /healthz tests
# ---------------------------------------------------------------------------

class TestHealthz:
    def test_healthz_ok(self):
        """When a backend is available, /healthz returns status=ok."""
        mock_backend = _make_mock_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.get("/healthz")

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "ok"
        assert body["backend"] == "tesseract"
        assert body["available"] is True

    def test_healthz_degraded(self):
        """When no backend is available, /healthz returns status=degraded."""
        with patch("app.main._backend", None), \
             patch("app.main.get_backend", side_effect=RuntimeError("No backend")):
            from app.main import app
            client = TestClient(app)
            resp = client.get("/healthz")

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "degraded"
        assert "error" in body


# ---------------------------------------------------------------------------
# /detect tests
# ---------------------------------------------------------------------------

class TestDetect:
    def test_detect_empty_paths(self):
        """Empty input_paths returns 400."""
        mock_backend = _make_mock_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/detect", json={
                "study_uid": "1.2.3",
                "input_paths": [],
            })

        assert resp.status_code == 400

    def test_detect_success_clean(self, tmp_path):
        """When backend finds no PHI, response has phi_detected=False."""
        # Create a dummy file so path-existence check passes
        dummy = tmp_path / "slice.dcm"
        dummy.write_bytes(b"\x00" * 128)

        mock_backend = _make_mock_backend()
        mock_backend.detect.return_value = []

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/detect", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(dummy)],
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["phi_detected"] is False
        assert body["findings"] == []
        assert body["tool_used"] == "tesseract"
        assert body["study_uid"] == "1.2.3.4.5"

    def test_detect_success_flagged(self, tmp_path):
        """When backend finds PHI, response has phi_detected=True and findings."""
        dummy = tmp_path / "slice.dcm"
        dummy.write_bytes(b"\x00" * 128)

        mock_backend = _make_mock_backend()
        mock_backend.detect.return_value = [
            FileFinding(
                file="slice.dcm",
                regions=[
                    Region(text="DOE JOHN", confidence=0.92, bbox=[10, 10, 80, 12]),
                    Region(text="DOB 01/01/1970", confidence=0.85, bbox=[10, 30, 100, 12]),
                ],
            )
        ]

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/detect", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(dummy)],
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["phi_detected"] is True
        assert len(body["findings"]) == 1
        assert body["findings"][0]["file"] == "slice.dcm"
        assert len(body["findings"][0]["regions"]) == 2
        assert body["findings"][0]["regions"][0]["text"] == "DOE JOHN"
        assert body["findings"][0]["regions"][0]["confidence"] == 0.92
        assert body["findings"][0]["regions"][1]["text"] == "DOB 01/01/1970"
