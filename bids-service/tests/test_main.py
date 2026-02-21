"""Endpoint tests for the bids-service FastAPI service.

Uses httpx + TestClient. Mocks the backend at the module level to avoid
requiring dcm2niix to be installed in the test environment.
"""

from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.backends.base import BidsBackend, ConversionResult


# ---------------------------------------------------------------------------
# Helper: build a mock backend
# ---------------------------------------------------------------------------

def _make_mock_backend(name: str = "dcm2niix", available: bool = True) -> MagicMock:
    backend = MagicMock(spec=BidsBackend)
    backend.name = name
    backend.available.return_value = available
    backend.convert.return_value = ConversionResult()
    return backend


# ---------------------------------------------------------------------------
# /healthz tests
# ---------------------------------------------------------------------------

class TestHealthz:
    def test_healthz_ok(self):
        """When a backend is available, /healthz returns status=ok."""
        mock_backend = _make_mock_backend()

        with patch("app.main._backend", mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.get("/healthz")

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "ok"
        assert body["backend"] == "dcm2niix"
        assert body["available"] is True

    def test_healthz_degraded(self):
        """When no backend is available, /healthz returns status=degraded."""
        with patch("app.main._backend", None):
            from app.main import app
            client = TestClient(app)
            resp = client.get("/healthz")

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "degraded"
        assert body["available"] is False
        assert "error" in body


# ---------------------------------------------------------------------------
# /convert tests
# ---------------------------------------------------------------------------

class TestConvert:
    def test_convert_no_backend(self):
        """When _backend is None, /convert returns status=failed."""
        with patch("app.main._backend", None):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/convert", json={
                "study_uid": "1.2.3",
                "input_dir": "/tmp/in",
                "output_dir": "/tmp/out",
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert body["study_uid"] == "1.2.3"
        assert "No BIDS conversion backend" in body["error"]

    def test_convert_success_mock(self, tmp_path):
        """When backend.convert succeeds with NIfTI output, response has status=complete."""
        mock_backend = _make_mock_backend()
        mock_backend.convert.return_value = ConversionResult(
            output_files=[
                "/tmp/out/sub-abc12345/anat/sub-abc12345_T1w.nii.gz",
                "/tmp/out/sub-abc12345/anat/sub-abc12345_T1w.json",
                "/tmp/out/dataset_description.json",
                "/tmp/out/participants.tsv",
            ],
            warnings=[],
        )

        with patch("app.main._backend", mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/convert", json={
                "study_uid": "1.2.3.4.5",
                "input_dir": str(tmp_path / "in"),
                "output_dir": str(tmp_path / "out"),
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["study_uid"] == "1.2.3.4.5"
        assert body["tool_used"] == "dcm2niix"
        assert len(body["output_files"]) == 4
        assert any(f.endswith(".nii.gz") for f in body["output_files"])
        assert body["error"] is None

    def test_convert_no_nifti_output(self, tmp_path):
        """When backend produces no NIfTI files, response has status=failed."""
        mock_backend = _make_mock_backend()
        mock_backend.convert.return_value = ConversionResult(
            output_files=[
                "/tmp/out/dataset_description.json",
                "/tmp/out/participants.tsv",
            ],
            warnings=["dcm2niix produced no output for series 1"],
        )

        with patch("app.main._backend", mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/convert", json={
                "study_uid": "1.2.3.4.5",
                "input_dir": str(tmp_path / "in"),
                "output_dir": str(tmp_path / "out"),
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert "No NIfTI files produced" in body["error"]

    def test_convert_backend_exception(self, tmp_path):
        """When backend.convert raises, response has status=failed with error."""
        mock_backend = _make_mock_backend()
        mock_backend.convert.side_effect = RuntimeError("dcm2niix crashed")

        with patch("app.main._backend", mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/convert", json={
                "study_uid": "1.2.3.4.5",
                "input_dir": str(tmp_path / "in"),
                "output_dir": str(tmp_path / "out"),
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert "dcm2niix crashed" in body["error"]
