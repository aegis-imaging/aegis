"""Tests for pixel_utils.dicom_to_pil."""

import numpy as np
import pytest

from app.backends.pixel_utils import dicom_to_pil, _apply_windowing, _to_uint8


class TestDicomToPil:
    def test_returns_pil_image(self, make_dicom_file):
        path = make_dicom_file(filename="pil_test.dcm", pixel_value=128)
        img = dicom_to_pil(path)
        assert img is not None
        assert img.mode == "L"  # grayscale
        assert img.size == (64, 64)

    def test_no_pixel_data_returns_none(self, make_dicom_file):
        path = make_dicom_file(filename="no_pix.dcm", include_pixel_data=False)
        assert dicom_to_pil(path) is None

    def test_nonexistent_file_returns_none(self):
        assert dicom_to_pil("/nonexistent/path.dcm") is None

    def test_windowing_applied(self, make_dicom_file):
        """With windowing, resulting image pixels should be within 0-255."""
        path = make_dicom_file(
            filename="windowed.dcm",
            pixel_value=1000,
            window_center=500,
            window_width=1000,
        )
        img = dicom_to_pil(path)
        assert img is not None
        arr = np.array(img)
        assert arr.min() >= 0
        assert arr.max() <= 255


class TestApplyWindowingStandalone:
    def test_no_window_tags_passthrough(self):
        from pydicom.dataset import Dataset
        ds = Dataset()
        arr = np.array([[10, 20], [30, 40]], dtype=np.float64)
        result = _apply_windowing(ds, arr)
        np.testing.assert_array_equal(result, arr)

    def test_basic_windowing(self):
        from pydicom.dataset import Dataset
        ds = Dataset()
        ds.WindowCenter = 100
        ds.WindowWidth = 100
        arr = np.array([[0, 100, 200]], dtype=np.float64)
        result = _apply_windowing(ds, arr)
        assert result[0, 0] == 50.0   # clipped from 0 to low=50
        assert result[0, 1] == 100.0  # within range
        assert result[0, 2] == 150.0  # clipped from 200 to high=150
