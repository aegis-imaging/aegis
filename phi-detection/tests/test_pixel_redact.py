"""Tests for pixel PHI redaction utility."""

import io
import struct

import numpy as np
import pytest


def _make_synthetic_dicom(tmp_path, rows=64, cols=64, pixel_value=1000,
                          photometric="MONOCHROME2", multi_frame=False):
    """Create a minimal synthetic DICOM file with controlled pixel values."""
    import pydicom
    from pydicom.dataset import Dataset, FileDataset
    from pydicom.uid import ExplicitVRLittleEndian
    from pydicom.sequence import Sequence

    path = str(tmp_path / "test.dcm")

    file_meta = Dataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
    file_meta.MediaStorageSOPInstanceUID = "1.2.3.4.5.6"
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = FileDataset(path, {}, file_meta=file_meta, preamble=b"\x00" * 128)
    ds.Rows = rows
    ds.Columns = cols
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = photometric

    if multi_frame:
        ds.NumberOfFrames = 3
        frames = np.full((3, rows, cols), pixel_value, dtype=np.uint16)
        ds.PixelData = frames.tobytes()
    else:
        arr = np.full((rows, cols), pixel_value, dtype=np.uint16)
        ds.PixelData = arr.tobytes()

    ds.save_as(path, write_like_original=False)
    return path


class TestRedactDicomPixels:
    def test_no_regions_copies_unchanged(self, tmp_path):
        """File is copied unchanged when no regions are provided."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=500)
        output_path = str(tmp_path / "out.dcm")

        result = redact_dicom_pixels(input_path, output_path, [])
        assert result is False

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array
        assert np.all(arr == 500)

    def test_region_blacked_out(self, tmp_path):
        """Pixels within the region bbox are set to 0."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=1000)
        output_path = str(tmp_path / "out.dcm")

        regions = [{"bbox": [10, 10, 20, 10]}]  # x=10, y=10, w=20, h=10
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=0)
        assert result is True

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array

        # Redacted area should be 0
        assert np.all(arr[10:20, 10:30] == 0)

        # Outside the region should still be original value
        assert arr[0, 0] == 1000
        assert arr[50, 50] == 1000

    def test_padding_expands_region(self, tmp_path):
        """Padding expands the blacked-out area beyond the bbox."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=800)
        output_path = str(tmp_path / "out.dcm")

        regions = [{"bbox": [20, 20, 10, 10]}]
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=5)
        assert result is True

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array

        # With 5px padding: x 15-35, y 15-35
        assert arr[15, 15] == 0
        assert arr[34, 34] == 0

        # Outside the padded area
        assert arr[14, 14] == 800
        assert arr[36, 36] == 800

    def test_monochrome1_uses_max_value(self, tmp_path):
        """MONOCHROME1 fills with max value (to appear black in that polarity)."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=500,
                                            photometric="MONOCHROME1")
        output_path = str(tmp_path / "out.dcm")

        regions = [{"bbox": [10, 10, 20, 10]}]
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=0)
        assert result is True

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array

        # MONOCHROME1: should be filled with max uint16 value
        assert np.all(arr[10:20, 10:30] == 65535)

    def test_multi_frame_redaction(self, tmp_path):
        """All frames are redacted when multi-frame DICOM."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=700, multi_frame=True)
        output_path = str(tmp_path / "out.dcm")

        regions = [{"bbox": [5, 5, 10, 10]}]
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=0)
        assert result is True

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array

        # All 3 frames should be redacted in the same region
        for frame_idx in range(3):
            assert np.all(arr[frame_idx, 5:15, 5:15] == 0)
            assert arr[frame_idx, 0, 0] == 700

    def test_clamps_to_image_bounds(self, tmp_path):
        """Region extending beyond image bounds is clamped."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, rows=32, cols=32, pixel_value=999)
        output_path = str(tmp_path / "out.dcm")

        # Region that extends past the 32x32 image
        regions = [{"bbox": [20, 20, 100, 100]}]
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=0)
        assert result is True

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array

        assert np.all(arr[20:32, 20:32] == 0)
        assert arr[0, 0] == 999

    def test_multiple_regions(self, tmp_path):
        """Multiple regions are all redacted."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=600)
        output_path = str(tmp_path / "out.dcm")

        regions = [
            {"bbox": [5, 5, 10, 5]},
            {"bbox": [40, 40, 10, 5]},
        ]
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=0)
        assert result is True

        ds = pydicom.dcmread(output_path)
        arr = ds.pixel_array

        assert np.all(arr[5:10, 5:15] == 0)
        assert np.all(arr[40:45, 40:50] == 0)
        assert arr[25, 25] == 600

    def test_invalid_bbox_skipped(self, tmp_path):
        """Regions with zero or negative dimensions are skipped."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=400)
        output_path = str(tmp_path / "out.dcm")

        regions = [{"bbox": [10, 10, 0, 0]}]  # zero-size bbox
        result = redact_dicom_pixels(input_path, output_path, regions, padding_px=0)
        assert result is False  # No actual redaction performed

    def test_output_uses_uncompressed_transfer_syntax(self, tmp_path):
        """Redacted file uses Explicit VR Little Endian transfer syntax."""
        from app.backends.pixel_redact import redact_dicom_pixels
        import pydicom

        input_path = _make_synthetic_dicom(tmp_path, pixel_value=300)
        output_path = str(tmp_path / "out.dcm")

        regions = [{"bbox": [10, 10, 10, 10]}]
        redact_dicom_pixels(input_path, output_path, regions, padding_px=0)

        ds = pydicom.dcmread(output_path)
        assert ds.file_meta.TransferSyntaxUID == "1.2.840.10008.1.2.1"
