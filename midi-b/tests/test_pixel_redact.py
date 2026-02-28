"""Tests for pixel PHI redaction."""

import shutil
from pathlib import Path
from unittest.mock import patch

import numpy as np
import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import ExplicitVRLittleEndian

from midi_b.deid.pixel_redact import redact_dicom_pixels


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

    # Create a non-zero pixel array
    arr = np.ones((rows, cols), dtype=np.uint16) * 1000
    ds.PixelData = arr.tobytes()

    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = ds.SOPClassUID
    file_meta.MediaStorageSOPInstanceUID = ds.SOPInstanceUID
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
    file_ds = FileDataset(str(filepath), ds, file_meta=file_meta, preamble=b"\x00" * 128)
    file_ds.save_as(str(filepath))
    return filepath


def test_redact_zeros_region(tmp_path: Path):
    src = _make_pixel_dicom(tmp_path / "src.dcm")
    dst = tmp_path / "dst.dcm"
    regions = [{"bbox": [10, 10, 20, 10]}]

    result = redact_dicom_pixels(str(src), str(dst), regions)

    assert result is True
    assert dst.exists()
    ds = pydicom.dcmread(str(dst))
    arr = ds.pixel_array
    # The region [10:20, 5:35] (with 5px padding) should be zeroed
    assert arr[15, 20] == 0  # center of region should be zero
    assert arr[0, 0] == 1000  # corner should be unchanged


def test_redact_no_regions_copies(tmp_path: Path):
    src = _make_pixel_dicom(tmp_path / "src.dcm")
    dst = tmp_path / "dst.dcm"

    result = redact_dicom_pixels(str(src), str(dst), [])

    assert result is False
    assert dst.exists()
    ds = pydicom.dcmread(str(dst))
    arr = ds.pixel_array
    assert arr[0, 0] == 1000  # unchanged


def test_redact_no_pixel_data_copies(tmp_path: Path):
    """DICOM without pixel data should be copied unchanged."""
    src = tmp_path / "nopixel.dcm"
    ds = Dataset()
    ds.PatientName = "TEST"
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
    ds.SOPInstanceUID = "1.2.3.4.5"
    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = ds.SOPClassUID
    file_meta.MediaStorageSOPInstanceUID = ds.SOPInstanceUID
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
    file_ds = FileDataset(str(src), ds, file_meta=file_meta, preamble=b"\x00" * 128)
    file_ds.save_as(str(src))

    dst = tmp_path / "dst.dcm"
    result = redact_dicom_pixels(str(src), str(dst), [{"bbox": [0, 0, 10, 10]}])
    assert result is False
    assert dst.exists()


def test_redact_zero_dimension_bbox_ignored(tmp_path: Path):
    src = _make_pixel_dicom(tmp_path / "src.dcm")
    dst = tmp_path / "dst.dcm"
    regions = [{"bbox": [10, 10, 0, 10]}]  # width=0 should be skipped

    result = redact_dicom_pixels(str(src), str(dst), regions)
    assert result is False  # no actual redaction applied


def test_redact_clamps_to_image_bounds(tmp_path: Path):
    src = _make_pixel_dicom(tmp_path / "src.dcm", rows=32, cols=32)
    dst = tmp_path / "dst.dcm"
    # Region extends beyond image bounds
    regions = [{"bbox": [25, 25, 20, 20]}]

    result = redact_dicom_pixels(str(src), str(dst), regions)
    assert result is True
    ds = pydicom.dcmread(str(dst))
    arr = ds.pixel_array
    # Top-left corner should be unchanged
    assert arr[0, 0] == 1000


def test_redact_monochrome1_fills_max(tmp_path: Path):
    """MONOCHROME1 images should fill with max value (white = black in M1)."""
    src = tmp_path / "m1.dcm"
    ds = Dataset()
    ds.PatientName = "TEST"
    ds.PatientID = "T001"
    ds.Rows = 32
    ds.Columns = 32
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME1"
    ds.PixelRepresentation = 0
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
    ds.SOPInstanceUID = "1.2.3.4.5"
    arr = np.ones((32, 32), dtype=np.uint16) * 500
    ds.PixelData = arr.tobytes()
    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = ds.SOPClassUID
    file_meta.MediaStorageSOPInstanceUID = ds.SOPInstanceUID
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
    file_ds = FileDataset(str(src), ds, file_meta=file_meta, preamble=b"\x00" * 128)
    file_ds.save_as(str(src))

    dst = tmp_path / "dst.dcm"
    regions = [{"bbox": [5, 5, 10, 10]}]
    result = redact_dicom_pixels(str(src), str(dst), regions)
    assert result is True
    ds2 = pydicom.dcmread(str(dst))
    arr2 = ds2.pixel_array
    assert arr2[10, 10] == np.iinfo(np.uint16).max  # filled with max
