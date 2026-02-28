"""Endpoint tests for the phi-detection FastAPI service.

Uses httpx + TestClient. Mocks the backend at the module level to avoid
requiring Tesseract to be installed in the test environment.
"""

import shutil
from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.backends.base import PHIDetectionBackend, FileFinding, Region
from app.backends.text_scrub import TextScrubBackend


# ---------------------------------------------------------------------------
# Helper: build a mock backend
# ---------------------------------------------------------------------------

def _make_mock_backend(name: str = "tesseract", available: bool = True) -> MagicMock:
    backend = MagicMock(spec=PHIDetectionBackend)
    backend.name = name
    backend.available.return_value = available
    backend.detect.return_value = []
    return backend


def _make_mock_text_scrub_backend(name: str = "regex") -> MagicMock:
    backend = MagicMock(spec=TextScrubBackend)
    backend.name = name
    backend.available.return_value = True
    backend.scrub.return_value = {"scrubbed": "", "phi_found": False}
    return backend


# ---------------------------------------------------------------------------
# /healthz tests
# ---------------------------------------------------------------------------

class TestHealthz:
    def test_healthz_ok(self):
        """When a backend is available, /healthz returns status=ok."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
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

    def test_detect_missing_path_returns_400(self):
        """Non-existent input path returns 400 before calling the backend."""
        mock_backend = _make_mock_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/detect", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": ["/nonexistent/does/not/exist.dcm"],
            })

        assert resp.status_code == 400
        mock_backend.detect.assert_not_called()

    def test_detect_returns_failed_on_backend_exception(self, tmp_path):
        """When the backend raises unexpectedly, /detect returns status=failed (200)."""
        dummy = tmp_path / "slice.dcm"
        dummy.write_bytes(b"\x00" * 128)

        mock_backend = _make_mock_backend()
        mock_backend.detect.side_effect = RuntimeError("backend exploded")

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
        assert body["status"] == "failed"
        assert body["phi_detected"] is False
        assert body["error"] is not None
        assert "backend exploded" in body["error"]

    def test_detect_response_includes_duration(self, tmp_path):
        """Successful /detect response includes a non-negative duration_seconds."""
        dummy = tmp_path / "slice.dcm"
        dummy.write_bytes(b"\x00" * 128)

        mock_backend = _make_mock_backend()

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
        assert "duration_seconds" in body
        assert body["duration_seconds"] >= 0.0


# ---------------------------------------------------------------------------
# /redact tests
# ---------------------------------------------------------------------------

class TestRedact:
    def test_redact_empty_paths(self):
        """Empty input_paths returns 400."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/redact", json={
                "study_uid": "1.2.3",
                "input_paths": [],
                "output_dir": "/tmp/out",
            })

        assert resp.status_code == 400

    def test_redact_no_phi_copies_files(self, tmp_path):
        """When no PHI detected, files are copied unchanged."""
        dummy = tmp_path / "input" / "slice.dcm"
        dummy.parent.mkdir(parents=True)
        dummy.write_bytes(b"\x00" * 128)
        output_dir = tmp_path / "output"

        mock_backend = _make_mock_backend()
        mock_backend.detect.return_value = []
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/redact", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(dummy)],
                "output_dir": str(output_dir),
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["files_processed"] == 1
        assert body["files_redacted"] == 0
        assert (output_dir / "slice.dcm").exists()

    def test_redact_with_phi_applies_redaction(self, tmp_path):
        """When PHI detected, response shows files_redacted > 0."""
        dummy = tmp_path / "input" / "slice.dcm"
        dummy.parent.mkdir(parents=True)
        dummy.write_bytes(b"\x00" * 128)
        output_dir = tmp_path / "output"

        mock_backend = _make_mock_backend()
        mock_backend.detect.return_value = [
            FileFinding(
                file="slice.dcm",
                regions=[Region(text="PHI TEXT", confidence=0.9, bbox=[10, 10, 80, 12])],
            )
        ]
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub), \
             patch("app.main.redact_dicom_pixels", return_value=True) as mock_redact:
            from app.main import app
            client = TestClient(app)
            resp = client.post("/redact", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(dummy)],
                "output_dir": str(output_dir),
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["files_redacted"] == 1
        assert len(body["findings"]) == 1
        mock_redact.assert_called_once()

    def test_redact_missing_input_returns_400(self):
        """Non-existent input path returns 400."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/redact", json={
                "study_uid": "1.2.3",
                "input_paths": ["/nonexistent/path.dcm"],
                "output_dir": "/tmp/out",
            })

        assert resp.status_code == 400

    def test_redact_returns_failed_on_exception(self, tmp_path):
        """When backend raises, /redact returns status=failed."""
        dummy = tmp_path / "slice.dcm"
        dummy.write_bytes(b"\x00" * 128)

        mock_backend = _make_mock_backend()
        mock_backend.detect.side_effect = RuntimeError("OCR crash")
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/redact", json={
                "study_uid": "1.2.3.4.5",
                "input_paths": [str(dummy)],
                "output_dir": str(tmp_path / "out"),
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert "OCR crash" in body["error"]


# ---------------------------------------------------------------------------
# /scrub-text tests
# ---------------------------------------------------------------------------

class TestScrubText:
    def test_scrub_text_empty_texts_returns_400(self):
        """Empty texts list returns 400."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/scrub-text", json={"texts": []})

        assert resp.status_code == 400

    def test_scrub_text_no_phi(self):
        """When backend finds no PHI, results have phi_found=False."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()
        mock_text_scrub.scrub.return_value = {"scrubbed": "CT CHEST", "phi_found": False}

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/scrub-text", json={
                "texts": ["CT CHEST"],
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert len(body["results"]) == 1
        assert body["results"][0]["original"] == "CT CHEST"
        assert body["results"][0]["scrubbed"] == "CT CHEST"
        assert body["results"][0]["phi_found"] is False

    def test_scrub_text_with_phi(self):
        """When backend finds PHI, results show scrubbed text."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()
        mock_text_scrub.scrub.return_value = {
            "scrubbed": "CT for [REMOVED]",
            "phi_found": True,
        }

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/scrub-text", json={
                "texts": ["CT for John Smith"],
                "context": {"patient_name": "SMITH^JOHN"},
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "complete"
        assert body["results"][0]["phi_found"] is True
        assert body["results"][0]["scrubbed"] == "CT for [REMOVED]"
        assert body["tool_used"] == "regex"

    def test_scrub_text_multiple_fields(self):
        """Multiple text fields are all processed."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()
        mock_text_scrub.scrub.side_effect = [
            {"scrubbed": "CT CHEST", "phi_found": False},
            {"scrubbed": "[REMOVED] scan", "phi_found": True},
        ]

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/scrub-text", json={
                "texts": ["CT CHEST", "John Smith scan"],
            })

        assert resp.status_code == 200
        body = resp.json()
        assert len(body["results"]) == 2
        assert body["results"][0]["phi_found"] is False
        assert body["results"][1]["phi_found"] is True

    def test_scrub_text_includes_duration(self):
        """Response includes duration_seconds."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()
        mock_text_scrub.scrub.return_value = {"scrubbed": "test", "phi_found": False}

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/scrub-text", json={
                "texts": ["test"],
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["duration_seconds"] >= 0.0

    def test_scrub_text_returns_failed_on_exception(self):
        """When backend raises, /scrub-text returns status=failed."""
        mock_backend = _make_mock_backend()
        mock_text_scrub = _make_mock_text_scrub_backend()
        mock_text_scrub.scrub.side_effect = RuntimeError("LLM timeout")

        with patch("app.main._backend", mock_backend), \
             patch("app.main.get_backend", return_value=mock_backend), \
             patch("app.main._text_scrub_backend", mock_text_scrub), \
             patch("app.main.get_text_scrub_backend", return_value=mock_text_scrub):
            from app.main import app
            client = TestClient(app)
            resp = client.post("/scrub-text", json={
                "texts": ["some text"],
            })

        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "failed"
        assert "LLM timeout" in body["error"]
