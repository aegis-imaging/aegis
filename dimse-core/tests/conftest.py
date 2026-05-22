"""Shared test fixtures for dimse-core."""
from __future__ import annotations

import io
from pathlib import Path

import pytest
from pydicom.dataset import Dataset, FileMetaDataset
from pydicom.filewriter import dcmwrite
from pydicom.uid import ExplicitVRLittleEndian, generate_uid


@pytest.fixture
def tmp_data_dir(tmp_path: Path) -> str:
    return str(tmp_path / "data")


def make_dicom_bytes(
    study_uid: str | None = None,
    series_uid: str | None = None,
    modality: str = "MR",
    body_part: str = "HEAD",
) -> bytes:
    """Build a tiny valid DICOM file in memory."""
    ds = Dataset()
    ds.StudyInstanceUID = study_uid or generate_uid()
    ds.SeriesInstanceUID = series_uid or generate_uid()
    ds.SOPInstanceUID = generate_uid()
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"  # MR Image Storage
    ds.PatientName = "Test^Patient"
    ds.PatientID = "TEST123"
    ds.Modality = modality
    ds.BodyPartExamined = body_part
    ds.StudyDescription = "Test"
    ds.StudyDate = "20260301"
    ds.is_little_endian = True
    ds.is_implicit_VR = False
    fm = FileMetaDataset()
    fm.MediaStorageSOPClassUID = ds.SOPClassUID
    fm.MediaStorageSOPInstanceUID = ds.SOPInstanceUID
    fm.TransferSyntaxUID = ExplicitVRLittleEndian
    ds.file_meta = fm
    buf = io.BytesIO()
    dcmwrite(buf, ds, write_like_original=False)
    return buf.getvalue()
