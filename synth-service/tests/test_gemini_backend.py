"""Tests for the Vertex AI Imagen gemini_backend module.

All google-genai / PIL calls are mocked — no cloud credentials required.
"""

import io
import sys
from unittest.mock import MagicMock, patch

import numpy as np
import pytest


# ---------------------------------------------------------------------------
# Helper: build fake sys.modules entries for google.genai
# ---------------------------------------------------------------------------

def _make_genai_modules():
    """Build fake sys.modules entries for google.genai."""
    mock_types = MagicMock()
    mock_genai = MagicMock()
    mock_genai.types = mock_types

    mock_google = MagicMock()
    mock_google.genai = mock_genai

    return {
        "google": mock_google,
        "google.genai": mock_genai,
        "google.genai.types": mock_types,
    }, mock_genai


# ---------------------------------------------------------------------------
# _imagen_available
# ---------------------------------------------------------------------------

class TestImagenAvailable:
    def test_available_when_genai_importable(self):
        """Returns True when google-genai SDK can be imported."""
        mods, _ = _make_genai_modules()
        with patch.dict("sys.modules", mods):
            from app.gemini_backend import _imagen_available
            result = _imagen_available()
        assert result is True

    def test_unavailable_when_import_fails(self):
        """Returns False when google-genai cannot be imported."""
        with patch.dict("sys.modules", {"google": None, "google.genai": None}):
            from app.gemini_backend import _imagen_available
            result = _imagen_available()
        assert result is False


# ---------------------------------------------------------------------------
# generate_gemini_slices
# ---------------------------------------------------------------------------

def _make_mock_png(size: int = 64) -> bytes:
    """Create a minimal PNG image as bytes using PIL."""
    from PIL import Image
    img = Image.fromarray(np.zeros((size, size), dtype=np.uint8), mode="L")
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    return buf.getvalue()


class TestGenerateGeminiSlices:
    def _run_generate(self, n_slices=3, size=32, seed=0, png_bytes=None):
        """Run generate_gemini_slices with all google-genai calls mocked."""
        if png_bytes is None:
            png_bytes = _make_mock_png(64)

        # Build mock response matching google-genai SDK structure
        mock_image_obj = MagicMock()
        mock_image_obj.image_bytes = png_bytes

        mock_generated_image = MagicMock()
        mock_generated_image.image = mock_image_obj

        mock_response = MagicMock()
        mock_response.generated_images = [mock_generated_image]

        mock_client = MagicMock()
        mock_client.models.generate_images.return_value = mock_response

        mods, mock_genai = _make_genai_modules()
        mock_genai.Client.return_value = mock_client

        with patch.dict("sys.modules", mods), \
             patch("app.gemini_backend.generate_uid", side_effect=[
                 "1.2.3.4.5.6.7.8.9",  # study_uid
                 "1.2.3.4.5.6.7.8.0",  # series_uid
             ]):
            from app.gemini_backend import generate_gemini_slices
            slices = generate_gemini_slices(n_slices=n_slices, size=size, seed=seed)

        return slices, mock_client

    def test_returns_correct_number_of_slices(self):
        slices, _ = self._run_generate(n_slices=3, size=32)
        assert len(slices) == 3

    def test_slice_tuple_format(self):
        """Each tuple is (pixel_array, study_uid, series_uid, instance_num, slice_z)."""
        slices, _ = self._run_generate(n_slices=2, size=32)
        arr, study_uid, series_uid, instance_num, slice_z = slices[0]
        assert isinstance(arr, np.ndarray)
        assert arr.shape == (32, 32)
        assert arr.dtype == np.float32
        assert 0.0 <= arr.min() and arr.max() <= 1.0
        assert isinstance(study_uid, str) and len(study_uid) > 0
        assert isinstance(series_uid, str) and len(series_uid) > 0
        assert instance_num == 1
        assert isinstance(slice_z, float)

    def test_all_slices_share_same_study_and_series_uid(self):
        slices, _ = self._run_generate(n_slices=4, size=32)
        study_uids = {s[1] for s in slices}
        series_uids = {s[2] for s in slices}
        assert len(study_uids) == 1
        assert len(series_uids) == 1

    def test_instance_numbers_are_sequential(self):
        slices, _ = self._run_generate(n_slices=3, size=32)
        nums = [s[3] for s in slices]
        assert nums == [1, 2, 3]

    def test_generate_images_called_per_slice(self):
        n = 3
        _, mock_client = self._run_generate(n_slices=n, size=32)
        assert mock_client.models.generate_images.call_count == n

    def test_prompt_varies_by_position(self):
        """Prompt should vary for inferior vs superior slices."""
        _, mock_client = self._run_generate(n_slices=4, size=32)
        calls = mock_client.models.generate_images.call_args_list
        prompts = [c[1]["prompt"] for c in calls]
        # Not all prompts should be identical (position varies)
        assert len(set(prompts)) > 1

    def test_output_array_resized_to_requested_size(self):
        """Output arrays must be size×size regardless of Imagen output size."""
        png_bytes = _make_mock_png(size=512)  # Imagen returns 512px
        slices, _ = self._run_generate(n_slices=1, size=64, png_bytes=png_bytes)
        arr = slices[0][0]
        assert arr.shape == (64, 64)

    def test_compatible_with_write_dicom_series(self, tmp_path):
        """Slices produced by generate_gemini_slices can be written as DICOM."""
        from app.phantom import write_dicom_series

        slices, _ = self._run_generate(n_slices=2, size=32)
        output_dir = str(tmp_path / "gemini_test")
        paths = write_dicom_series(slices, output_dir, size=32)
        assert len(paths) == 2
        for p in paths:
            import os
            assert os.path.exists(p)


# ---------------------------------------------------------------------------
# /generate endpoint with use_gemini=true
# ---------------------------------------------------------------------------

class TestGenerateEndpointGemini:
    def test_generate_uses_gemini_when_enabled(self, tmp_path):
        """POST /generate with use_gemini=true calls generate_gemini_slices."""
        from app.phantom import generate_phantom_slices
        mock_slices = generate_phantom_slices(n_slices=2, size=32)

        captured = {}

        def _mock_gemini(n_slices, size, seed):
            captured["called"] = True
            captured["n_slices"] = n_slices
            return mock_slices

        with patch("app.gemini_backend.generate_gemini_slices", side_effect=_mock_gemini), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.use_gemini = True
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            from fastapi.testclient import TestClient
            client = TestClient(app)
            resp = client.post("/generate", json={"slices": 2, "size": 32, "use_gemini": True})

        assert resp.status_code == 200
        body = resp.json()
        assert body["tool_used"] == "gemini"
        assert captured.get("called") is True

    def test_gemini_disabled_in_config_uses_phantom(self, tmp_path):
        """When cfg.use_gemini=false, gemini path is not taken even if use_gemini=true in request."""
        captured = {}

        def _mock_phantom(n_slices, size, seed, with_face):
            captured["called"] = True
            from app.phantom import generate_phantom_slices as _gps
            return _gps(n_slices=n_slices, size=size, seed=seed)

        with patch("app.main.generate_phantom_slices", side_effect=_mock_phantom), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.use_gemini = False  # disabled at server level
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            from fastapi.testclient import TestClient
            client = TestClient(app)
            resp = client.post("/generate", json={"slices": 2, "size": 32, "use_gemini": True})

        assert resp.status_code == 200
        body = resp.json()
        assert body["tool_used"] == "phantom"
        assert captured.get("called") is True
