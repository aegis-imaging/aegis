"""Shared fixtures for classification-service tests."""

import struct
from pathlib import Path

import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import generate_uid, ExplicitVRLittleEndian
import pytest


@pytest.fixture
def make_dicom_file():
    """Factory fixture: creates a synthetic DICOM file on disk."""

    def _make(
        tmp_dir: Path,
        filename: str = "test.dcm",
        rows: int = 16,
        cols: int = 16,
        modality: str = "MR",
        body_part: str = "HEAD",
        series_uid: str | None = None,
        study_uid: str | None = None,
        sop_class_uid: str = "1.2.840.10008.5.1.4.1.1.4",  # MR Image Storage
        instance_number: int = 1,
        series_description: str = "T1 SAGITTAL",
        study_description: str = "BRAIN MRI W/O CONTRAST",
        protocol_name: str = "T1_SAG_MPRAGE",
        # Set to None to omit the tag entirely
        modality_value=...,
        body_part_value=...,
    ) -> str:
        filepath = str(tmp_dir / filename)

        file_meta = Dataset()
        file_meta.MediaStorageSOPClassUID = sop_class_uid
        file_meta.MediaStorageSOPInstanceUID = generate_uid()
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)
        ds.SOPClassUID = sop_class_uid
        ds.SOPInstanceUID = generate_uid()
        ds.StudyInstanceUID = study_uid or generate_uid()
        ds.SeriesInstanceUID = series_uid or generate_uid()
        ds.Rows = rows
        ds.Columns = cols
        ds.BitsAllocated = 16
        ds.BitsStored = 12
        ds.HighBit = 11
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"
        ds.InstanceNumber = instance_number

        # Use sentinel (...) to mean "use default"; None means "omit tag"
        mod = modality if modality_value is ... else modality_value
        bp = body_part if body_part_value is ... else body_part_value
        if mod is not None:
            ds.Modality = mod
        if bp is not None:
            ds.BodyPartExamined = bp

        if series_description is not None:
            ds.SeriesDescription = series_description
        if study_description is not None:
            ds.StudyDescription = study_description
        if protocol_name is not None:
            ds.ProtocolName = protocol_name

        ds.PixelData = struct.pack(f"<{rows * cols}H", *([500] * (rows * cols)))
        ds.save_as(filepath)
        return filepath

    return _make


@pytest.fixture
def dicom_study(tmp_path, make_dicom_file):
    """Creates a 5-slice synthetic MR HEAD study. Returns list of file paths."""
    study_uid = generate_uid()
    series_uid = generate_uid()
    paths = []
    for i in range(1, 6):
        p = make_dicom_file(
            tmp_dir=tmp_path,
            filename=f"slice_{i:04d}.dcm",
            study_uid=study_uid,
            series_uid=series_uid,
            instance_number=i,
        )
        paths.append(p)
    return paths
