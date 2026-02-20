"""Tests for classification-service pixel_utils.dicom_to_pil."""

import numpy as np
import pytest

from app.backends.pixel_utils import dicom_to_pil, _apply_windowing, _to_uint8


class TestDicomToPil:
    def test_returns_pil_image(self, tmp_path, make_dicom_file):
        path = make_dicom_file(tmp_dir=tmp_path, filename="pil_test.dcm")
        img = dicom_to_pil(path)
        assert img is not None
        assert img.mode == "L"  # grayscale

    def test_nonexistent_file_returns_none(self):
        assert dicom_to_pil("/nonexistent/path.dcm") is None


class TestToUint8:
    def test_normal_range(self):
        arr = np.array([[0, 128, 255]], dtype=np.float64)
        result = _to_uint8(arr)
        assert result.dtype == np.uint8
        assert result[0, 0] == 0
        assert result[0, 2] == 255

    def test_uniform_array(self):
        arr = np.array([[42, 42]], dtype=np.float64)
        result = _to_uint8(arr)
        np.testing.assert_array_equal(result, np.zeros((1, 2), dtype=np.uint8))
