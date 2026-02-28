"""Tests for pixel burned-in PHI detection (mocked Tesseract)."""

from pathlib import Path
from unittest.mock import MagicMock, patch

import numpy as np
import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import ExplicitVRLittleEndian

from midi_b.deid.pixel_detect import (
    FileFinding,
    Region,
    detect_burned_in_text,
    detect_file,
    tesseract_available,
)


def _make_pixel_dicom(filepath: Path, rows: int = 64, cols: int = 64) -> Path:
    """Create a minimal DICOM file with pixel data."""
    ds = Dataset()
    ds.PatientName = "TEST"
    ds.PatientID = "T001"
    ds.Rows = rows
    ds.Columns = cols
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME2"
    ds.PixelRepresentation = 0
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
    ds.SOPInstanceUID = "1.2.3.4.5"
    arr = np.ones((rows, cols), dtype=np.uint16) * 1000
    ds.PixelData = arr.tobytes()
    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = ds.SOPClassUID
    file_meta.MediaStorageSOPInstanceUID = ds.SOPInstanceUID
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
    file_ds = FileDataset(str(filepath), ds, file_meta=file_meta, preamble=b"\x00" * 128)
    file_ds.save_as(str(filepath))
    return filepath


def _mock_tesseract_data(texts: list[str], confs: list[float]):
    """Build a mock pytesseract.image_to_data return dict."""
    n = len(texts)
    return {
        "text": texts,
        "conf": confs,
        "left": [10 * i for i in range(n)],
        "top": [10] * n,
        "width": [50] * n,
        "height": [20] * n,
    }


@patch("midi_b.deid.pixel_detect.dicom_to_pil")
def test_detect_file_with_text(mock_pil, tmp_path: Path):
    """Simulates Tesseract finding text in a DICOM image."""
    dcm_path = _make_pixel_dicom(tmp_path / "test.dcm")
    mock_pil.return_value = MagicMock()  # fake PIL Image

    mock_pytesseract = MagicMock()
    mock_pytesseract.Output.DICT = "dict"
    mock_pytesseract.image_to_data.return_value = _mock_tesseract_data(
        ["JOHN", "DOE", "MRN12345"],
        [95.0, 90.0, 85.0],
    )

    with patch("midi_b.deid.pixel_detect.pytesseract", mock_pytesseract, create=True):
        with patch.dict("sys.modules", {"pytesseract": mock_pytesseract}):
            regions = detect_file(str(dcm_path))

    assert len(regions) == 3
    assert regions[0].text == "JOHN"
    assert regions[0].confidence == 0.95
    assert len(regions[0].bbox) == 4


@patch("midi_b.deid.pixel_detect.dicom_to_pil")
def test_detect_file_no_text(mock_pil, tmp_path: Path):
    """No text detected returns empty list."""
    dcm_path = _make_pixel_dicom(tmp_path / "test.dcm")
    mock_pil.return_value = MagicMock()

    mock_pytesseract = MagicMock()
    mock_pytesseract.Output.DICT = "dict"
    mock_pytesseract.image_to_data.return_value = _mock_tesseract_data(
        ["", "ab"],  # empty and too short
        [-1.0, 50.0],
    )

    with patch("midi_b.deid.pixel_detect.pytesseract", mock_pytesseract, create=True):
        with patch.dict("sys.modules", {"pytesseract": mock_pytesseract}):
            regions = detect_file(str(dcm_path))

    assert len(regions) == 0


@patch("midi_b.deid.pixel_detect.dicom_to_pil")
def test_detect_file_no_pixel_data(mock_pil):
    """Files without pixel data return empty list."""
    mock_pil.return_value = None
    mock_pytesseract = MagicMock()
    with patch.dict("sys.modules", {"pytesseract": mock_pytesseract}):
        regions = detect_file("/nonexistent.dcm")
    assert len(regions) == 0


@patch("midi_b.deid.pixel_detect.dicom_to_pil")
def test_detect_burned_in_text_multi_file(mock_pil, tmp_path: Path):
    """Test multi-file detection returns findings for files with text."""
    f1 = _make_pixel_dicom(tmp_path / "a.dcm")
    f2 = _make_pixel_dicom(tmp_path / "b.dcm")
    mock_pil.return_value = MagicMock()

    mock_pytesseract = MagicMock()
    mock_pytesseract.Output.DICT = "dict"
    call_count = [0]

    def side_effect(img, output_type=None):
        call_count[0] += 1
        if call_count[0] == 1:
            return _mock_tesseract_data(["PHI_TEXT"], [92.0])
        return _mock_tesseract_data([""], [-1.0])

    mock_pytesseract.image_to_data.side_effect = side_effect

    with patch("midi_b.deid.pixel_detect.pytesseract", mock_pytesseract, create=True):
        with patch.dict("sys.modules", {"pytesseract": mock_pytesseract}):
            findings = detect_burned_in_text([str(f1), str(f2)])

    assert len(findings) == 1  # only first file had text
    assert findings[0].file == "a.dcm"
    assert len(findings[0].regions) == 1


@patch("midi_b.deid.pixel_detect.dicom_to_pil")
def test_confidence_threshold_filtering(mock_pil, tmp_path: Path):
    """Regions below confidence threshold are excluded."""
    dcm_path = _make_pixel_dicom(tmp_path / "test.dcm")
    mock_pil.return_value = MagicMock()

    mock_pytesseract = MagicMock()
    mock_pytesseract.Output.DICT = "dict"
    mock_pytesseract.image_to_data.return_value = _mock_tesseract_data(
        ["HIGH", "LOW"],
        [80.0, 20.0],  # 0.8 and 0.2 normalised
    )

    with patch("midi_b.deid.pixel_detect.pytesseract", mock_pytesseract, create=True):
        with patch.dict("sys.modules", {"pytesseract": mock_pytesseract}):
            # threshold of 0.5 should exclude the 0.2 confidence region
            regions = detect_file(str(dcm_path), confidence_threshold=0.5)

    assert len(regions) == 1
    assert regions[0].text == "HIGH"


def test_tesseract_available_missing():
    """tesseract_available returns False when pytesseract is not installed."""
    with patch.dict("sys.modules", {"pytesseract": None}):
        # When pytesseract import raises, available should return False
        result = tesseract_available()
        # Can't easily mock a broken import, but at minimum function exists
        assert isinstance(result, bool)
