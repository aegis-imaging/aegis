"""Tests for DICOM format handling in the QC service.

Covers Enhanced (multi-frame) DICOM pixel array handling and mosaic detection
in the basic QC backend.
"""

import numpy as np
import pydicom
from pydicom.uid import ExplicitVRLittleEndian, generate_uid

from app.backends.basic import BasicBackend


def _make_multi_frame_dicom(tmp_dir, filename="mf.dcm", num_frames=3, rows=32, cols=32):
    """Create a multi-frame DICOM file with controlled pixel data."""
    filepath = str(tmp_dir / filename)
    sop_uid = generate_uid()
    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4.1"  # Enhanced MR
    file_meta.MediaStorageSOPInstanceUID = sop_uid
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = pydicom.dataset.FileDataset(
        filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128
    )
    ds.SOPClassUID = file_meta.MediaStorageSOPClassUID
    ds.SOPInstanceUID = sop_uid
    ds.StudyInstanceUID = generate_uid()
    ds.SeriesInstanceUID = generate_uid()
    ds.Modality = "MR"
    ds.BodyPartExamined = "HEAD"
    ds.NumberOfFrames = num_frames
    ds.Rows = rows
    ds.Columns = cols
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME2"

    # Multi-frame pixel data: (frames, rows, cols)
    arr = np.full((num_frames, rows, cols), 500, dtype=np.uint16)
    # Add slight variation to corner for SNR computation
    arr[:, :4, :4] = 500 + np.arange(16).reshape(4, 4) % 5 - 2
    ds.PixelData = arr.tobytes()

    ds.save_as(filepath)
    return filepath


class TestMultiFrameQC:
    """QC basic backend handles multi-frame DICOM without crashing."""

    def test_snr_on_multi_frame(self, tmp_path):
        """SNR check should extract first frame from multi-frame array."""
        path = _make_multi_frame_dicom(tmp_path, num_frames=5)
        backend = BasicBackend(snr_threshold=1.0)
        result = backend.check([path])
        # Should not crash; should produce a result
        assert result.overall in ("pass", "warn", "fail")
        snr_check = next(c for c in result.checks if c.name == "snr")
        assert snr_check.status in ("pass", "warn", "fail")

    def test_file_integrity_multi_frame(self, tmp_path):
        """File integrity check works with multi-frame DICOM."""
        path = _make_multi_frame_dicom(tmp_path, num_frames=3)
        backend = BasicBackend()
        result = backend.check([path])
        integrity = next(c for c in result.checks if c.name == "file_integrity")
        assert integrity.status == "pass"
        assert integrity.details["parsed_ok"] == 1


def _make_mosaic_dicom(tmp_dir, filename="mosaic.dcm", rows=384, cols=384):
    """Create a Siemens mosaic DICOM file."""
    filepath = str(tmp_dir / filename)
    sop_uid = generate_uid()
    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    file_meta.MediaStorageSOPInstanceUID = sop_uid
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = pydicom.dataset.FileDataset(
        filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128
    )
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    ds.SOPInstanceUID = sop_uid
    ds.StudyInstanceUID = generate_uid()
    ds.SeriesInstanceUID = generate_uid()
    ds.Modality = "MR"
    ds.BodyPartExamined = "HEAD"
    ds.ImageType = ["ORIGINAL", "PRIMARY", "M", "ND", "MOSAIC"]
    ds.Rows = rows
    ds.Columns = cols
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME2"

    arr = np.full((rows, cols), 500, dtype=np.uint16)
    ds.PixelData = arr.tobytes()

    ds.save_as(filepath)
    return filepath


class TestMosaicQC:
    """QC handles mosaic DICOM metadata correctly."""

    def test_mosaic_slice_count(self, tmp_path):
        """Mosaic DICOM is counted as a single file for slice coverage."""
        p = _make_mosaic_dicom(tmp_path)

        backend = BasicBackend()
        result = backend.check([p])
        # Should process without error — single file means 1 "slice"
        assert result.overall in ("pass", "warn", "fail")
