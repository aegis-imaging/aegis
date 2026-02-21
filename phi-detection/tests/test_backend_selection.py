"""Tests for backend selection logic (_select_backend, get_backend) in app.main.

These functions are the critical dispatch layer that picks which OCR backend
runs. They were previously completely untested.
"""

import pytest
from unittest.mock import patch

from app.backends.tesseract import TesseractBackend
from app.backends.google_vision import GoogleVisionBackend
from app.backends.aws_textract import AWSTextractBackend


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _run_select_backend(phi_tool="auto", gv_avail=False, at_avail=False, ts_avail=False):
    """Invoke _select_backend with controlled backend availability."""
    from app.main import _select_backend

    with patch("app.main.cfg") as mock_cfg, \
         patch.object(GoogleVisionBackend, "available", return_value=gv_avail), \
         patch.object(AWSTextractBackend, "available", return_value=at_avail), \
         patch.object(TesseractBackend, "available", return_value=ts_avail):
        mock_cfg.phi_tool = phi_tool
        mock_cfg.confidence_threshold = 0.4
        mock_cfg.min_text_length = 3
        return _select_backend()


# ---------------------------------------------------------------------------
# _select_backend: auto priority order
# ---------------------------------------------------------------------------

class TestSelectBackendAuto:
    def test_auto_selects_google_vision_first(self):
        """With all backends available, auto picks google_vision."""
        backend = _run_select_backend(phi_tool="auto", gv_avail=True, at_avail=True, ts_avail=True)
        assert backend.name == "google_vision"

    def test_auto_selects_textract_when_vision_unavailable(self):
        """When google_vision is unavailable, auto picks aws_textract."""
        backend = _run_select_backend(phi_tool="auto", gv_avail=False, at_avail=True, ts_avail=True)
        assert backend.name == "aws_textract"

    def test_auto_selects_tesseract_as_last_resort(self):
        """When both cloud backends are unavailable, auto falls back to tesseract."""
        backend = _run_select_backend(phi_tool="auto", gv_avail=False, at_avail=False, ts_avail=True)
        assert backend.name == "tesseract"

    def test_auto_raises_when_all_unavailable(self):
        """RuntimeError when no backend is available."""
        with pytest.raises(RuntimeError, match="No PHI detection backend"):
            _run_select_backend(phi_tool="auto", gv_avail=False, at_avail=False, ts_avail=False)


# ---------------------------------------------------------------------------
# _select_backend: explicit PHI_TOOL overrides
# ---------------------------------------------------------------------------

class TestSelectBackendExplicit:
    def test_phi_tool_forces_tesseract(self):
        """PHI_TOOL=tesseract returns TesseractBackend even when others are available."""
        backend = _run_select_backend(phi_tool="tesseract", gv_avail=True, at_avail=True, ts_avail=True)
        assert backend.name == "tesseract"

    def test_phi_tool_forces_google_vision(self):
        """PHI_TOOL=google_vision returns GoogleVisionBackend."""
        backend = _run_select_backend(phi_tool="google_vision", gv_avail=True, ts_avail=True)
        assert backend.name == "google_vision"

    def test_phi_tool_forces_aws_textract(self):
        """PHI_TOOL=aws_textract returns AWSTextractBackend."""
        backend = _run_select_backend(phi_tool="aws_textract", at_avail=True, ts_avail=True)
        assert backend.name == "aws_textract"

    def test_phi_tool_forced_backend_unavailable_raises(self):
        """RuntimeError when the forced backend is not available."""
        with pytest.raises(RuntimeError):
            _run_select_backend(phi_tool="tesseract", ts_avail=False)


# ---------------------------------------------------------------------------
# get_backend: lazy caching
# ---------------------------------------------------------------------------

class TestGetBackend:
    def test_get_backend_caches_result(self):
        """get_backend() returns the same object on repeated calls (lazy singleton)."""
        import app.main

        original = app.main._backend
        app.main._backend = None  # force re-init
        try:
            from app.main import get_backend

            with patch.object(TesseractBackend, "available", return_value=True), \
                 patch.object(GoogleVisionBackend, "available", return_value=False), \
                 patch.object(AWSTextractBackend, "available", return_value=False), \
                 patch("app.main.cfg") as mock_cfg:
                mock_cfg.phi_tool = "auto"
                mock_cfg.confidence_threshold = 0.4
                mock_cfg.min_text_length = 3

                b1 = get_backend()
                b2 = get_backend()
                assert b1 is b2
        finally:
            app.main._backend = original  # restore previous state
