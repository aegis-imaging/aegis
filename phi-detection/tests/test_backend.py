"""Unit tests for the Tesseract PHI detection backend.

Tests the pixel_utils functions (windowing, uint8) and mocks pytesseract
for the OCR-dependent code paths.
"""

import numpy as np
import pytest

from app.backends.tesseract import TesseractBackend
from app.backends.pixel_utils import _apply_windowing, _to_uint8
from app.backends.base import Region, FileFinding


# ---------------------------------------------------------------------------
# _apply_windowing tests
# ---------------------------------------------------------------------------

class TestApplyWindowing:
    """Tests for pixel_utils._apply_windowing."""

    def test_apply_windowing_basic(self):
        """Window center=500, width=1000 clips to [0, 1000]."""
        from pydicom.dataset import Dataset

        ds = Dataset()
        ds.WindowCenter = 500
        ds.WindowWidth = 1000

        arr = np.array([[-100, 0, 500], [1000, 1200, 2000]], dtype=np.float64)
        result = _apply_windowing(ds, arr)

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
        result = _apply_windowing(ds, arr)

        np.testing.assert_array_equal(result, arr)

    def test_apply_windowing_multivalue(self):
        """Multi-value WC/WW takes the first element."""
        from pydicom.dataset import Dataset
        from pydicom.multival import MultiValue

        ds = Dataset()
        ds.WindowCenter = MultiValue(float, [200, 500])
        ds.WindowWidth = MultiValue(float, [400, 800])

        arr = np.array([[0, 200, 500]], dtype=np.float64)
        result = _apply_windowing(ds, arr)

        # Should use WC=200, WW=400 -> low=0, high=400
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
        result = _apply_windowing(ds, arr)

        np.testing.assert_array_equal(result, arr)


# ---------------------------------------------------------------------------
# _to_uint8 tests
# ---------------------------------------------------------------------------

class TestToUint8:
    """Tests for pixel_utils._to_uint8."""

    def test_to_uint8_normal(self):
        """Range 0-4095 maps to 0-255."""
        arr = np.array([[0, 2048, 4095]], dtype=np.float64)
        result = _to_uint8(arr)

        assert result.dtype == np.uint8
        assert result[0, 0] == 0
        assert result[0, 2] == 255
        # 2048/4095 * 255 ~ 127
        assert 126 <= result[0, 1] <= 128

    def test_to_uint8_uniform(self):
        """All same value maps to all zeros (no contrast)."""
        arr = np.array([[42, 42], [42, 42]], dtype=np.float64)
        result = _to_uint8(arr)

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

        backend = TesseractBackend(confidence_threshold=0.4, min_text_length=3)
        regions = backend._scan_file(path, None)  # pytesseract won't be called
        assert regions == []

    def test_scan_file_with_mock_tesseract(self, make_dicom_file):
        """Mock pytesseract.image_to_data and verify Region filtering."""
        path = make_dicom_file(filename="with_text.dcm", pixel_value=100)

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
        regions = backend._scan_file(path, mock_tesseract)

        # "PATIENT": conf=0.85 >= 0.4, len=7 >= 3 -> included
        # "ab":      conf=0.90 >= 0.4, len=2 < 3  -> excluded (too short)
        # "NAME":    conf=0.60 >= 0.4, len=4 >= 3 -> included
        # "12345":   conf=0.30 < 0.4              -> excluded (low confidence)
        # "":        conf=-1.0 < 0                 -> excluded (non-text block)
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


# ---------------------------------------------------------------------------
# TesseractBackend.available() — unit-level
# ---------------------------------------------------------------------------

class TestTesseractAvailable:
    """TesseractBackend.available() inspects the tesseract binary via pytesseract."""

    def test_returns_true_when_version_succeeds(self):
        from unittest.mock import patch

        backend = TesseractBackend()
        with patch("pytesseract.get_tesseract_version", return_value="4.1.1"):
            assert backend.available() is True

    def test_returns_false_when_version_raises(self):
        from unittest.mock import patch

        backend = TesseractBackend()
        with patch("pytesseract.get_tesseract_version", side_effect=Exception("binary not found")):
            assert backend.available() is False


# ---------------------------------------------------------------------------
# TesseractBackend.detect() — exception recovery
# ---------------------------------------------------------------------------

class TestTesseractDetectExceptionRecovery:
    def test_detect_continues_on_scan_file_exception(self, make_dicom_file):
        """If _scan_file raises for one file, detect() skips it and continues."""
        from unittest.mock import patch

        path1 = make_dicom_file(filename="ok1.dcm", pixel_value=100)
        path2 = make_dicom_file(filename="err.dcm", pixel_value=50)
        path3 = make_dicom_file(filename="ok3.dcm", pixel_value=200)

        backend = TesseractBackend(confidence_threshold=0.4, min_text_length=3)

        def mock_scan(path, *args):
            if "err.dcm" in path:
                raise RuntimeError("OCR process failed")
            elif "ok1.dcm" in path:
                return [Region(text="PATIENT", confidence=0.9, bbox=[0, 0, 50, 10])]
            return []

        with patch.object(backend, "_scan_file", side_effect=mock_scan):
            findings = backend.detect([path1, path2, path3])

        # Only ok1 produced findings; err.dcm is skipped; ok3 is clean.
        assert len(findings) == 1
        assert findings[0].file == "ok1.dcm"
        assert findings[0].regions[0].text == "PATIENT"


# ---------------------------------------------------------------------------
# Backend .name properties
# ---------------------------------------------------------------------------

class TestBackendNameProperties:
    def test_backend_name_properties(self):
        from app.backends.google_vision import GoogleVisionBackend
        from app.backends.aws_textract import AWSTextractBackend

        assert TesseractBackend().name == "tesseract"
        assert GoogleVisionBackend().name == "google_vision"
        assert AWSTextractBackend().name == "aws_textract"
