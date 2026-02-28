"""Shared fixtures for MIDI-B tests — synthetic DICOM datasets via pydicom."""

from __future__ import annotations

import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.sequence import Sequence as DicomSequence
from pydicom.uid import ExplicitVRLittleEndian
import pytest
import tempfile
from pathlib import Path


def make_dataset(**overrides: object) -> Dataset:
    """Create a minimal pydicom Dataset that exercises the main action types."""
    ds = Dataset()
    ds.PatientName = "DOE^JOHN"
    ds.PatientID = "MRN-12345"
    ds.PatientBirthDate = "19800115"
    ds.PatientSex = "M"
    ds.ReferringPhysicianName = "SMITH^ALICE"
    ds.InstitutionName = "General Hospital"
    ds.StudyDate = "20240301"
    ds.StudyTime = "143000"
    ds.SeriesDate = "20240301"
    ds.AcquisitionDate = "20240301"
    ds.ContentDate = "20240301"
    ds.AccessionNumber = "ACC-99999"
    ds.StudyInstanceUID = "1.2.840.113619.2.55.3.12345"
    ds.SeriesInstanceUID = "1.2.840.113619.2.55.3.12345.1"
    ds.SOPInstanceUID = "1.2.840.113619.2.55.3.12345.1.1"
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
    ds.Modality = "CT"
    ds.Manufacturer = "SIEMENS"
    ds.StudyDescription = "CT CHEST WITH CONTRAST"
    ds.SeriesDescription = "AXIAL 3MM"
    ds.BodyPartExamined = "CHEST"
    ds.Rows = 512
    ds.Columns = 512

    for k, v in overrides.items():
        setattr(ds, k, v)

    return ds


def make_sr_dataset() -> Dataset:
    """Create a synthetic Structured Report dataset."""
    ds = Dataset()
    ds.PatientName = "DOE^JOHN"
    ds.PatientID = "MRN-12345"
    ds.ReferringPhysicianName = "SMITH^ALICE"
    ds.InstitutionName = "General Hospital"
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.88.33"  # Comprehensive SR
    ds.StudyInstanceUID = "1.2.840.113619.2.55.3.12345"
    ds.SeriesInstanceUID = "1.2.840.113619.2.55.3.12345.1"
    ds.SOPInstanceUID = "1.2.840.113619.2.55.3.12345.1.1"
    ds.Modality = "SR"
    ds.StudyDate = "20240301"
    ds.AccessionNumber = "ACC-99999"

    # Build nested ContentSequence
    nested_item = Dataset()
    nested_item.TextValue = "Referring physician: Dr. Smith, MRN: 12345"
    nested_item.PersonName = "SMITH^ALICE"

    item1 = Dataset()
    item1.TextValue = "Patient John Doe presented with headache"
    item1.ReferencedSOPInstanceUID = "1.2.840.113619.2.55.3.99999"
    item1.ContentSequence = DicomSequence([nested_item])

    item2 = Dataset()
    item2.TextValue = "Normal brain MRI findings"
    item2.UID = "1.2.840.113619.2.55.3.88888"

    ds.ContentSequence = DicomSequence([item1, item2])

    return ds


def make_dicom_file(tmp_path: Path, ds: Dataset, filename: str = "test.dcm") -> Path:
    """Write a Dataset to a DICOM file and return the path."""
    filepath = tmp_path / filename
    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = getattr(ds, "SOPClassUID", "1.2.840.10008.5.1.4.1.1.2")
    file_meta.MediaStorageSOPInstanceUID = getattr(ds, "SOPInstanceUID", "1.2.3.4")
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
    file_ds = FileDataset(str(filepath), ds, file_meta=file_meta, preamble=b"\x00" * 128)
    file_ds.is_little_endian = True
    file_ds.is_implicit_VR = False
    file_ds.save_as(str(filepath), write_like_original=False)
    return filepath
