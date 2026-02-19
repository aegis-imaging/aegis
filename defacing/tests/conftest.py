"""
Shared fixtures for defacing service tests.

Generates synthetic DICOM files using pydicom + numpy so that tests
are fully self-contained (no external DICOM samples required).
"""

import os
import tempfile
from pathlib import Path

import numpy as np
import pydicom
from pydicom.dataset import Dataset, FileDataset, FileMetaDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid

import pytest


# ---------------------------------------------------------------------------
# Factory fixture
# ---------------------------------------------------------------------------

@pytest.fixture()
def make_dicom_file():
    """
    Factory fixture that creates a synthetic DICOM file on disk.

    Returns a callable with signature::

        make_dicom_file(
            tmp_dir,
            filename="slice.dcm",
            rows=64,
            cols=64,
            modality="MR",
            body_part="HEAD",
            series_uid=None,
            study_uid=None,
            sop_class_uid="1.2.840.10008.5.1.4.1.1.4",   # MR Image Storage
            instance_number=1,
            slice_thickness=1.0,
            image_position=None,       # (x, y, z) tuple
            series_description="T1w",
        ) -> str   # absolute path to the created .dcm file
    """

    def _factory(
        tmp_dir: str,
        filename: str = "slice.dcm",
        rows: int = 64,
        cols: int = 64,
        modality: str = "MR",
        body_part: str = "HEAD",
        series_uid: str | None = None,
        study_uid: str | None = None,
        sop_class_uid: str = "1.2.840.10008.5.1.4.1.1.4",
        instance_number: int = 1,
        slice_thickness: float = 1.0,
        image_position: tuple[float, float, float] | None = None,
        series_description: str = "T1w",
    ) -> str:
        if series_uid is None:
            series_uid = generate_uid()
        if study_uid is None:
            study_uid = generate_uid()
        if image_position is None:
            image_position = (0.0, 0.0, float(instance_number))

        sop_instance_uid = generate_uid()

        file_path = os.path.join(tmp_dir, filename)

        # File meta info
        file_meta = FileMetaDataset()
        file_meta.MediaStorageSOPClassUID = sop_class_uid
        file_meta.MediaStorageSOPInstanceUID = sop_instance_uid
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset(
            file_path,
            {},
            file_meta=file_meta,
            preamble=b"\x00" * 128,
        )

        # Patient / Study / Series level
        ds.PatientID = "TESTPAT"
        ds.PatientName = "Test^Patient"
        ds.StudyInstanceUID = study_uid
        ds.SeriesInstanceUID = series_uid
        ds.SOPInstanceUID = sop_instance_uid
        ds.SOPClassUID = sop_class_uid
        ds.Modality = modality
        ds.BodyPartExamined = body_part
        ds.SeriesDescription = series_description
        ds.InstanceNumber = instance_number
        ds.SliceThickness = str(slice_thickness)

        # Image geometry
        ds.Rows = rows
        ds.Columns = cols
        ds.ImagePositionPatient = list(image_position)
        ds.ImageOrientationPatient = [1.0, 0.0, 0.0, 0.0, 1.0, 0.0]
        ds.PixelSpacing = [1.0, 1.0]

        # Pixel data — random 16-bit values so defacing changes are detectable
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"
        ds.BitsAllocated = 16
        ds.BitsStored = 16
        ds.HighBit = 15
        ds.PixelRepresentation = 1  # signed

        rng = np.random.default_rng(seed=instance_number)
        pixel_data = rng.integers(100, 2000, size=(rows, cols), dtype=np.int16)
        ds.PixelData = pixel_data.tobytes()

        ds.is_implicit_VR = False
        ds.is_little_endian = True

        ds.save_as(file_path, write_like_original=False)
        return file_path

    return _factory


# ---------------------------------------------------------------------------
# Convenience series fixtures
# ---------------------------------------------------------------------------

@pytest.fixture()
def dicom_series_25(make_dicom_file, tmp_path):
    """
    25-slice single-series study in a temp directory.

    25 slices exceeds the ``should_deface_series`` threshold of 20.
    Returns a list of absolute paths to the 25 .dcm files.
    """
    series_uid = generate_uid()
    study_uid = generate_uid()
    series_dir = str(tmp_path / "series_25")
    os.makedirs(series_dir, exist_ok=True)

    paths = []
    for i in range(25):
        p = make_dicom_file(
            series_dir,
            filename=f"{i:04d}.dcm",
            series_uid=series_uid,
            study_uid=study_uid,
            instance_number=i + 1,
            image_position=(0.0, 0.0, float(i)),
        )
        paths.append(p)
    return paths


@pytest.fixture()
def dicom_series_5(make_dicom_file, tmp_path):
    """
    5-slice single-series study (below the 20-slice threshold).

    Returns a list of absolute paths to the 5 .dcm files.
    """
    series_uid = generate_uid()
    study_uid = generate_uid()
    series_dir = str(tmp_path / "series_5")
    os.makedirs(series_dir, exist_ok=True)

    paths = []
    for i in range(5):
        p = make_dicom_file(
            series_dir,
            filename=f"{i:04d}.dcm",
            series_uid=series_uid,
            study_uid=study_uid,
            instance_number=i + 1,
            image_position=(0.0, 0.0, float(i)),
        )
        paths.append(p)
    return paths


@pytest.fixture()
def multi_series_study(make_dicom_file, tmp_path):
    """
    Study with 2 series, each containing 25 slices.

    Returns ``(series_a_paths, series_b_paths)`` where each is a list
    of absolute paths. The two series have different SeriesInstanceUIDs
    but share the same StudyInstanceUID.
    """
    study_uid = generate_uid()
    series_a_uid = generate_uid()
    series_b_uid = generate_uid()

    dir_a = str(tmp_path / "series_a")
    dir_b = str(tmp_path / "series_b")
    os.makedirs(dir_a, exist_ok=True)
    os.makedirs(dir_b, exist_ok=True)

    paths_a = []
    for i in range(25):
        p = make_dicom_file(
            dir_a,
            filename=f"{i:04d}.dcm",
            series_uid=series_a_uid,
            study_uid=study_uid,
            instance_number=i + 1,
            image_position=(0.0, 0.0, float(i)),
            series_description="T1w_MPRAGE",
        )
        paths_a.append(p)

    paths_b = []
    for i in range(25):
        p = make_dicom_file(
            dir_b,
            filename=f"{i:04d}.dcm",
            series_uid=series_b_uid,
            study_uid=study_uid,
            instance_number=i + 1,
            image_position=(0.0, 0.0, float(i)),
            series_description="FLAIR",
        )
        paths_b.append(p)

    return paths_a, paths_b
