"""Shared fixtures for bids-service tests.

Creates minimal DICOM files with configurable metadata for testing
the BIDS conversion pipeline. Uses struct.pack for pixel data (no numpy).
"""

import os
import struct
import tempfile

import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid
import pytest


@pytest.fixture()
def tmp_dir():
    """Provide a temporary directory that is cleaned up after the test."""
    with tempfile.TemporaryDirectory() as d:
        yield d


@pytest.fixture()
def make_dicom_file(tmp_dir):
    """Factory fixture that creates a minimal DICOM file.

    Parameters
    ----------
    filename : str
        Name of the .dcm file (written inside *tmp_dir*).
    modality : str
        DICOM Modality tag (default "MR").
    series_description : str
        SeriesDescription tag (default "").
    protocol_name : str
        ProtocolName tag (default "").
    series_uid : str | None
        SeriesInstanceUID (auto-generated if None).
    study_uid : str | None
        StudyInstanceUID (auto-generated if None).
    rows, cols : int
        Image dimensions (default 64x64).
    pixel_value : int
        Fill value for every pixel (default 0).

    Returns
    -------
    str
        Absolute path to the created DICOM file.
    """

    def _factory(
        filename: str = "test.dcm",
        modality: str = "MR",
        series_description: str = "",
        protocol_name: str = "",
        series_uid: str | None = None,
        study_uid: str | None = None,
        rows: int = 64,
        cols: int = 64,
        pixel_value: int = 0,
    ) -> str:
        filepath = os.path.join(tmp_dir, filename)

        file_meta = pydicom.dataset.FileMetaDataset()
        file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"  # MR Image Storage
        file_meta.MediaStorageSOPInstanceUID = generate_uid()
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)

        ds.SOPClassUID = file_meta.MediaStorageSOPClassUID
        ds.SOPInstanceUID = file_meta.MediaStorageSOPInstanceUID
        ds.StudyInstanceUID = study_uid or generate_uid()
        ds.SeriesInstanceUID = series_uid or generate_uid()
        ds.Modality = modality
        ds.SeriesDescription = series_description
        ds.ProtocolName = protocol_name
        ds.Rows = rows
        ds.Columns = cols
        ds.BitsAllocated = 16
        ds.BitsStored = 16
        ds.HighBit = 15
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"

        # Build pixel data with struct.pack (no numpy needed)
        pixel_count = rows * cols
        ds.PixelData = struct.pack(f"<{pixel_count}H", *([pixel_value] * pixel_count))

        ds.save_as(filepath)
        return filepath

    return _factory


@pytest.fixture()
def dicom_study(make_dicom_file):
    """Create a 5-slice DICOM study with a shared StudyInstanceUID."""
    study_uid = generate_uid()
    series_uid = generate_uid()
    paths = []
    for i in range(5):
        p = make_dicom_file(
            filename=f"slice_{i:03d}.dcm",
            modality="MR",
            series_description="T1_SAG_MPRAGE",
            protocol_name="T1w",
            series_uid=series_uid,
            study_uid=study_uid,
            pixel_value=i * 50,
        )
        paths.append(p)
    return paths
