"""Tests for DICOM format handling in the defacing pipeline.

Covers Enhanced (multi-frame), Mosaic (Siemens), and compressed DICOM
detection and rejection in should_deface_series().
"""

import os

import pydicom
from pydicom.uid import generate_uid

from app.pipeline import _is_mosaic, should_deface_series


class TestIsMosaic:
    """Tests for _is_mosaic() helper."""

    def test_mosaic_image_type(self):
        ds = pydicom.Dataset()
        ds.ImageType = ["ORIGINAL", "PRIMARY", "M", "ND", "MOSAIC"]
        assert _is_mosaic(ds) is True

    def test_non_mosaic_image_type(self):
        ds = pydicom.Dataset()
        ds.ImageType = ["ORIGINAL", "PRIMARY", "M", "ND"]
        assert _is_mosaic(ds) is False

    def test_empty_image_type(self):
        ds = pydicom.Dataset()
        ds.ImageType = []
        assert _is_mosaic(ds) is False

    def test_no_image_type(self):
        ds = pydicom.Dataset()
        assert _is_mosaic(ds) is False

    def test_case_insensitive(self):
        ds = pydicom.Dataset()
        ds.ImageType = ["ORIGINAL", "PRIMARY", "mosaic"]
        assert _is_mosaic(ds) is True


class TestShouldDefaceSeriesEnhanced:
    """Tests for Enhanced (multi-frame) DICOM rejection."""

    def test_enhanced_mr_sop_class_skipped(self, make_dicom_file, tmp_path):
        """Enhanced MR Image Storage SOP class is skipped."""
        series_uid = generate_uid()
        study_uid = generate_uid()
        d = str(tmp_path / "enhanced_mr")
        os.makedirs(d, exist_ok=True)
        paths = []
        for i in range(25):
            p = make_dicom_file(
                d,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                sop_class_uid="1.2.840.10008.5.1.4.1.1.4.1",  # Enhanced MR
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )
            paths.append(p)
        assert should_deface_series(paths) is False

    def test_enhanced_ct_sop_class_skipped(self, make_dicom_file, tmp_path):
        """Enhanced CT Image Storage SOP class is skipped."""
        series_uid = generate_uid()
        study_uid = generate_uid()
        d = str(tmp_path / "enhanced_ct")
        os.makedirs(d, exist_ok=True)
        paths = []
        for i in range(25):
            p = make_dicom_file(
                d,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                sop_class_uid="1.2.840.10008.5.1.4.1.1.2.1",  # Enhanced CT
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )
            paths.append(p)
        assert should_deface_series(paths) is False

    def test_multi_frame_dicom_skipped(self, make_dicom_file, tmp_path):
        """DICOM with NumberOfFrames > 1 is skipped even with classic SOP class."""
        series_uid = generate_uid()
        study_uid = generate_uid()
        d = str(tmp_path / "multi_frame")
        os.makedirs(d, exist_ok=True)
        paths = []
        for i in range(25):
            p = make_dicom_file(
                d,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )
            # Patch NumberOfFrames into the file
            ds = pydicom.dcmread(p)
            ds.NumberOfFrames = 10
            ds.save_as(p)
            paths.append(p)
        assert should_deface_series(paths) is False


class TestShouldDefaceSeriesMosaic:
    """Tests for Siemens mosaic DICOM rejection."""

    def test_mosaic_dicom_skipped(self, make_dicom_file, tmp_path):
        """Siemens mosaic DICOM (ImageType contains MOSAIC) is skipped."""
        series_uid = generate_uid()
        study_uid = generate_uid()
        d = str(tmp_path / "mosaic")
        os.makedirs(d, exist_ok=True)
        paths = []
        for i in range(25):
            p = make_dicom_file(
                d,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )
            # Patch ImageType to include MOSAIC
            ds = pydicom.dcmread(p)
            ds.ImageType = ["ORIGINAL", "PRIMARY", "M", "ND", "MOSAIC"]
            ds.save_as(p)
            paths.append(p)
        assert should_deface_series(paths) is False

    def test_non_mosaic_not_skipped(self, dicom_series_25):
        """Normal DICOM series without MOSAIC tag is not skipped."""
        assert should_deface_series(dicom_series_25) is True
