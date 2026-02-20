"""Shared fixtures for DIMSE receiver tests."""

from __future__ import annotations

from pathlib import Path

import pytest
from pydicom.dataset import FileDataset, FileMetaDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid


@pytest.fixture
def make_dicom_dataset():
    """Factory fixture that returns an in-memory FileDataset."""

    def _make(
        study_uid: str | None = None,
        series_uid: str | None = None,
        modality: str = "MR",
        body_part: str = "HEAD",
        study_description: str = "BRAIN MRI",
    ) -> FileDataset:
        file_meta = FileMetaDataset()
        file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
        file_meta.MediaStorageSOPInstanceUID = generate_uid()
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset("", {}, file_meta=file_meta, preamble=b"\x00" * 128)
        ds.SOPClassUID = file_meta.MediaStorageSOPClassUID
        ds.SOPInstanceUID = file_meta.MediaStorageSOPInstanceUID
        ds.StudyInstanceUID = study_uid or generate_uid()
        ds.SeriesInstanceUID = series_uid or generate_uid()
        ds.Modality = modality
        ds.BodyPartExamined = body_part
        ds.StudyDescription = study_description

        ds.Rows = 2
        ds.Columns = 2
        ds.BitsAllocated = 16
        ds.BitsStored = 16
        ds.HighBit = 15
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"
        ds.PixelData = (b"\x00\x00" * 4)

        ds.is_little_endian = True
        ds.is_implicit_VR = False
        return ds

    return _make
