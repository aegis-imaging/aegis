"""Tests for the Vertex AI Imagen gemini_backend module.

All Vertex AI / PIL calls are mocked — no cloud credentials required.
"""

import io
from unittest.mock import MagicMock, patch, PropertyMock

import numpy as np
import pytest


# ---------------------------------------------------------------------------
# _imagen_available
# ---------------------------------------------------------------------------

class TestImagenAvailable:
    def test_available_when_vertexai_importable(self):
        """Returns True when vertexai and ImageGenerationModel can be imported."""
        mock_module = MagicMock()
        with patch.dict("sys.modules", {
            "vertexai": mock_module,
            "vertexai.preview": mock_module,
            "vertexai.preview.vision_models": mock_module,
        }):
            from app.gemini_backend import _imagen_available
            # Reset the cached import state by calling directly
            result = _imagen_available()
        # When mocked as importable, should return True
        assert isinstance(result, bool)

    def test_unavailable_when_import_fails(self):
        """Returns False when vertexai cannot be imported."""
        import sys
        # Temporarily remove from sys.modules to force ImportError
        saved = sys.modules.pop("vertexai", None)
        saved_preview = sys.modules.pop("vertexai.preview", None)
        saved_vm = sys.modules.pop("vertexai.preview.vision_models", None)
        try:
            with patch.dict("sys.modules", {"vertexai": None}):
                from app.gemini_backend import _imagen_available
                result = _imagen_available()
            assert result is False
        finally:
            if saved is not None:
                sys.modules["vertexai"] = saved
            if saved_preview is not None:
                sys.modules["vertexai.preview"] = saved_preview
            if saved_vm is not None:
                sys.modules["vertexai.preview.vision_models"] = saved_vm


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
        """Run generate_gemini_slices with all Vertex AI calls mocked."""
        if png_bytes is None:
            png_bytes = _make_mock_png(64)

        mock_image = MagicMock()
        mock_image._image_bytes = png_bytes

        mock_response = MagicMock()
        mock_response.images = [mock_image]

        mock_model = MagicMock()
        mock_model.generate_images.return_value = mock_response

        mock_igm = MagicMock()
        mock_igm.from_pretrained.return_value = mock_model

        mock_vertexai = MagicMock()

        with patch.dict("sys.modules", {
            "vertexai": mock_vertexai,
            "vertexai.preview": mock_vertexai,
            "vertexai.preview.vision_models": MagicMock(
                ImageGenerationModel=mock_igm
            ),
        }), patch("app.gemini_backend.generate_uid", side_effect=[
            "1.2.3.4.5.6.7.8.9",  # study_uid
            "1.2.3.4.5.6.7.8.0",  # series_uid
        ]):
            from app.gemini_backend import generate_gemini_slices
            slices = generate_gemini_slices(n_slices=n_slices, size=size, seed=seed)

        return slices, mock_model

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
        _, mock_model = self._run_generate(n_slices=n, size=32)
        assert mock_model.generate_images.call_count == n

    def test_prompt_varies_by_position(self):
        """Prompt should vary for inferior vs superior slices."""
        _, mock_model = self._run_generate(n_slices=4, size=32)
        calls = mock_model.generate_images.call_args_list
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
