"""Unit tests for the Tesseract PHI detection backend.

Tests the pure utility functions (_apply_windowing, _to_uint8) directly and
mocks pytesseract for the OCR-dependent code paths.
"""

import numpy as np
import pytest

from app.backends.tesseract import TesseractBackend
from app.backends.base import Region, FileFinding


# ---------------------------------------------------------------------------
# _apply_windowing tests
# ---------------------------------------------------------------------------

class TestApplyWindowing:
    """Tests for TesseractBackend._apply_windowing (static method)."""

    def test_apply_windowing_basic(self):
        """Window center=500, width=1000 clips to [0, 1000]."""
        from pydicom.dataset import Dataset

        ds = Dataset()
        ds.WindowCenter = 500
        ds.WindowWidth = 1000

        arr = np.array([[-100, 0, 500], [1000, 1200, 2000]], dtype=np.float64)
        result = TesseractBackend._apply_windowing(ds, arr, np)

        # low = 500 - 500 = 0, high = 500 + 500 = 1000
        assert result[0, 0] == 0.0      # clipped from -100
        assert result[0, 1] == 0.0      # clipped from 0 (at boundary)
        assert result[0, 2] == 500.0    # within range
        assert result[1, 0] == 1000.0   # at boundary
        assert result[1, 1] == 1000.0   # clipped from 1200
        assert result[1, 2] == 1000.0   # clipped from 2000

    def test_apply_windowing_no_window(self):
        """Without WC/WW, array is returned unchanged."""
        from pydicom.dataset import Dataset

        ds = Dataset()
        # No WindowCenter or WindowWidth set

        arr = np.array([[10, 20], [30, 40]], dtype=np.float64)
        result = TesseractBackend._apply_windowing(ds, arr, np)

        np.testing.assert_array_equal(result, arr)

    def test_apply_windowing_multivalue(self):
        """Multi-value WC/WW takes the first element."""
        from pydicom.dataset import Dataset
        from pydicom.multival import MultiValue

        ds = Dataset()
        ds.WindowCenter = MultiValue(float, [200, 500])
        ds.WindowWidth = MultiValue(float, [400, 800])

        arr = np.array([[0, 200, 500]], dtype=np.float64)
        result = TesseractBackend._apply_windowing(ds, arr, np)

        # Should use WC=200, WW=400 → low=0, high=400
        assert result[0, 0] == 0.0
        assert result[0, 1] == 200.0
        assert result[0, 2] == 400.0  # clipped from 500

    def test_apply_windowing_zero_width(self):
        """Width=0 returns the array unchanged (guard against division by zero)."""
        from pydicom.dataset import Dataset

        ds = Dataset()
        ds.WindowCenter = 100
        ds.WindowWidth = 0

        arr = np.array([[50, 100, 150]], dtype=np.float64)
        result = TesseractBackend._apply_windowing(ds, arr, np)

        np.testing.assert_array_equal(result, arr)


# ---------------------------------------------------------------------------
# _to_uint8 tests
# ---------------------------------------------------------------------------

class TestToUint8:
    """Tests for TesseractBackend._to_uint8 (static method)."""

    def test_to_uint8_normal(self):
        """Range 0-4095 maps to 0-255."""
        arr = np.array([[0, 2048, 4095]], dtype=np.float64)
        result = TesseractBackend._to_uint8(arr, np)

        assert result.dtype == np.uint8
        assert result[0, 0] == 0
        assert result[0, 2] == 255
        # 2048/4095 * 255 ~ 127
        assert 126 <= result[0, 1] <= 128

    def test_to_uint8_uniform(self):
        """All same value maps to all zeros (no contrast)."""
        arr = np.array([[42, 42], [42, 42]], dtype=np.float64)
        result = TesseractBackend._to_uint8(arr, np)

        assert result.dtype == np.uint8
        np.testing.assert_array_equal(result, np.zeros((2, 2), dtype=np.uint8))


# ---------------------------------------------------------------------------
# _scan_file tests
# ---------------------------------------------------------------------------

class TestScanFile:
    """Tests for TesseractBackend._scan_file with mocked dependencies."""

    def test_scan_file_no_pixel_data(self, make_dicom_file):
        """DICOM without PixelData returns empty list."""
        path = make_dicom_file(filename="no_pixels.dcm", include_pixel_data=False)

        import pydicom
        from PIL import Image

        backend = TesseractBackend(confidence_threshold=0.4, min_text_length=3)
        # pytesseract won't be called, but pass a dummy
        regions = backend._scan_file(path, pydicom, None, Image, np)
        assert regions == []

    def test_scan_file_with_mock_tesseract(self, make_dicom_file):
        """Mock pytesseract.image_to_data and verify Region filtering."""
        path = make_dicom_file(filename="with_text.dcm", pixel_value=100)

        import pydicom
        from PIL import Image
        from unittest.mock import MagicMock

        mock_tesseract = MagicMock()
        mock_tesseract.Output.DICT = "dict"
        mock_tesseract.image_to_data.return_value = {
            "text": ["PATIENT", "ab", "NAME", "12345", ""],
            "conf": [85.0, 90.0, 60.0, 30.0, -1.0],
            "left": [10, 20, 30, 40, 50],
            "top": [10, 20, 30, 40, 50],
            "width": [50, 10, 40, 60, 0],
            "height": [12, 8, 12, 12, 0],
        }

        backend = TesseractBackend(confidence_threshold=0.4, min_text_length=3)
        regions = backend._scan_file(path, pydicom, mock_tesseract, Image, np)

        # "PATIENT": conf=0.85 >= 0.4, len=7 >= 3 → included
        # "ab":      conf=0.90 >= 0.4, len=2 < 3  → excluded (too short)
        # "NAME":    conf=0.60 >= 0.4, len=4 >= 3 → included
        # "12345":   conf=0.30 < 0.4              → excluded (low confidence)
        # "":        conf=-1.0 < 0                 → excluded (non-text block)
        assert len(regions) == 2
        assert regions[0].text == "PATIENT"
        assert regions[0].confidence == 0.85
        assert regions[0].bbox == [10, 10, 50, 12]
        assert regions[1].text == "NAME"
        assert regions[1].confidence == 0.6
        assert regions[1].bbox == [30, 30, 40, 12]


# ---------------------------------------------------------------------------
# detect (multi-file) tests
# ---------------------------------------------------------------------------

class TestDetectMultipleFiles:
    """Test the high-level detect method with multiple files."""

    def test_detect_multiple_files(self, make_dicom_file):
        """detect() processes multiple files and aggregates findings."""
        from unittest.mock import patch, MagicMock

        path1 = make_dicom_file(filename="file1.dcm", pixel_value=100)
        path2 = make_dicom_file(filename="file2.dcm", pixel_value=200)
        path3 = make_dicom_file(filename="file3.dcm", pixel_value=0)

        backend = TesseractBackend(confidence_threshold=0.4, min_text_length=3)

        # Mock _scan_file to return controlled results
        def mock_scan(path, *args):
            if "file1" in path:
                return [Region(text="DOE JOHN", confidence=0.92, bbox=[10, 10, 80, 12])]
            elif "file2" in path:
                return [Region(text="MRN-001", confidence=0.88, bbox=[5, 5, 50, 10])]
            else:
                return []  # file3 is clean

        with patch.object(backend, "_scan_file", side_effect=mock_scan):
            findings = backend.detect([path1, path2, path3])

        # file1 and file2 have findings; file3 is clean (omitted)
        assert len(findings) == 2
        assert findings[0].file == "file1.dcm"
        assert len(findings[0].regions) == 1
        assert findings[0].regions[0].text == "DOE JOHN"
        assert findings[1].file == "file2.dcm"
        assert len(findings[1].regions) == 1
        assert findings[1].regions[0].text == "MRN-001"
