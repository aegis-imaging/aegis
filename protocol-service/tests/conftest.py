"""Shared fixtures for protocol-service tests."""

import struct
from pathlib import Path

import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.sequence import Sequence
from pydicom.uid import generate_uid, ExplicitVRLittleEndian
import pytest


@pytest.fixture
def make_dicom_file():
    """Factory fixture: creates a synthetic DICOM file with MR acquisition parameters."""

    def _make(
        tmp_dir: Path,
        filename: str = "test.dcm",
        rows: int = 16,
        cols: int = 16,
        modality: str = "MR",
        body_part: str = "HEAD",
        sop_class_uid: str = "1.2.840.10008.5.1.4.1.1.4",  # MR Image Storage
        study_uid: str | None = None,
        series_uid: str | None = None,
        instance_number: int = 1,
        repetition_time: float = 2000.0,
        echo_time: float = 3.5,
        flip_angle: float = 9.0,
        slice_thickness: float = 1.0,
        pixel_spacing: list[float] | None = None,
        manufacturer: str = "SIEMENS",
        model: str = "MAGNETOM Prisma",
        software_version: str = "syngo MR E11",
        protocol_name: str = "T1_SAG_MPRAGE",
        series_description: str = "T1 SAGITTAL",
        magnetic_field_strength: float = 3.0,
        scanning_sequence: str = "GR",
        sequence_variant: str = "SP",
        mr_acquisition_type: str = "3D",
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
        ds.Modality = modality
        ds.BodyPartExamined = body_part
        ds.Rows = rows
        ds.Columns = cols
        ds.BitsAllocated = 16
        ds.BitsStored = 12
        ds.HighBit = 11
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"
        ds.InstanceNumber = instance_number
        ds.RepetitionTime = repetition_time
        ds.EchoTime = echo_time
        ds.FlipAngle = flip_angle
        ds.SliceThickness = slice_thickness
        ds.PixelSpacing = pixel_spacing or [0.5, 0.5]
        ds.Manufacturer = manufacturer
        ds.ManufacturerModelName = model
        ds.SoftwareVersions = software_version
        ds.ProtocolName = protocol_name
        ds.SeriesDescription = series_description
        ds.MagneticFieldStrength = magnetic_field_strength
        ds.ScanningSequence = scanning_sequence
        ds.SequenceVariant = sequence_variant
        ds.MRAcquisitionType = mr_acquisition_type
        ds.ImagePositionPatient = [0.0, 0.0, float(instance_number)]
        ds.ImageOrientationPatient = [1.0, 0.0, 0.0, 0.0, 1.0, 0.0]

        ds.PixelData = struct.pack(f"<{rows * cols}H", *([500] * (rows * cols)))
        ds.save_as(filepath)
        return filepath

    return _make


@pytest.fixture
def dicom_study(tmp_path, make_dicom_file):
    """Creates a 3-slice synthetic MR study. Returns list of file paths."""
    study_uid = generate_uid()
    series_uid = generate_uid()
    paths = []
    for i in range(1, 4):
        p = make_dicom_file(
            tmp_dir=tmp_path,
            filename=f"slice_{i:04d}.dcm",
            study_uid=study_uid,
            series_uid=series_uid,
            instance_number=i,
        )
        paths.append(p)
    return paths


@pytest.fixture
def make_enhanced_dicom(tmp_path):
    """Factory: creates an Enhanced MR DICOM file with nested functional group sequences."""

    def _make(
        filename: str = "enhanced.dcm",
        repetition_time: float = 2000.0,
        echo_time: float = 3.5,
        flip_angle: float = 9.0,
        slice_thickness: float = 1.0,
        pixel_spacing: list[float] | None = None,
        manufacturer: str = "SIEMENS",
        model: str = "MAGNETOM Vida",
        software_version: str = "VE12U",
        perframe_echo_time: float | None = None,
    ) -> str:
        filepath = str(tmp_path / filename)
        sop_class_uid = "1.2.840.10008.5.1.4.1.1.4.1"  # Enhanced MR

        file_meta = Dataset()
        file_meta.MediaStorageSOPClassUID = sop_class_uid
        file_meta.MediaStorageSOPInstanceUID = generate_uid()
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)
        ds.SOPClassUID = sop_class_uid
        ds.SOPInstanceUID = generate_uid()
        ds.StudyInstanceUID = generate_uid()
        ds.SeriesInstanceUID = generate_uid()
        ds.Modality = "MR"
        ds.Rows = 16
        ds.Columns = 16
        ds.BitsAllocated = 16
        ds.BitsStored = 12
        ds.HighBit = 11
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"
        ds.Manufacturer = manufacturer
        ds.ManufacturerModelName = model
        ds.SoftwareVersions = software_version
        ds.MagneticFieldStrength = 3.0
        ds.ProtocolName = "ep2d_bold"
        ds.SeriesDescription = "BOLD fMRI"

        # Build SharedFunctionalGroupsSequence
        shared = Dataset()

        timing = Dataset()
        timing.RepetitionTime = repetition_time
        timing.FlipAngle = flip_angle
        shared.MRTimingAndRelatedParametersSequence = Sequence([timing])

        echo = Dataset()
        echo.EffectiveEchoTime = echo_time
        shared.MREchoSequence = Sequence([echo])

        pixel_measures = Dataset()
        pixel_measures.SliceThickness = slice_thickness
        pixel_measures.PixelSpacing = pixel_spacing or [0.5, 0.5]
        shared.PixelMeasuresSequence = Sequence([pixel_measures])

        ds.SharedFunctionalGroupsSequence = Sequence([shared])

        # Optionally add PerFrameFunctionalGroupsSequence
        if perframe_echo_time is not None:
            frame0 = Dataset()
            pf_echo = Dataset()
            pf_echo.EffectiveEchoTime = perframe_echo_time
            frame0.MREchoSequence = Sequence([pf_echo])
            ds.PerFrameFunctionalGroupsSequence = Sequence([frame0])

        ds.PixelData = struct.pack("<256H", *([500] * 256))
        ds.save_as(filepath)
        return filepath

    return _make
