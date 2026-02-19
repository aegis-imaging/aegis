"""Shared fixtures for phi-detection tests.

Creates minimal DICOM files with configurable pixel data for testing
the OCR-based burned-in PHI detection pipeline.
"""

import os
import struct
import tempfile

import numpy as np
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
    """Factory fixture that creates a minimal DICOM file with pixel data.

    Parameters
    ----------
    filename : str
        Name of the .dcm file (written inside *tmp_dir*).
    rows, cols : int
        Image dimensions (default 64x64).
    pixel_value : int
        Fill value for every pixel (default 0).
    window_center, window_width : float | list[float] | None
        Optional DICOM windowing tags.
    bits_allocated : int
        BitsAllocated (default 16).
    include_pixel_data : bool
        If False, the file has no PixelData element at all.

    Returns
    -------
    str
        Absolute path to the created DICOM file.
    """

    def _factory(
        filename: str = "test.dcm",
        rows: int = 64,
        cols: int = 64,
        pixel_value: int = 0,
        window_center=None,
        window_width=None,
        bits_allocated: int = 16,
        include_pixel_data: bool = True,
    ) -> str:
        filepath = os.path.join(tmp_dir, filename)

        file_meta = pydicom.dataset.FileMetaDataset()
        file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
        file_meta.MediaStorageSOPInstanceUID = generate_uid()
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)

        ds.SOPClassUID = file_meta.MediaStorageSOPClassUID
        ds.SOPInstanceUID = file_meta.MediaStorageSOPInstanceUID
        ds.StudyInstanceUID = generate_uid()
        ds.SeriesInstanceUID = generate_uid()
        ds.Modality = "CT"
        ds.Rows = rows
        ds.Columns = cols
        ds.BitsAllocated = bits_allocated
        ds.BitsStored = bits_allocated
        ds.HighBit = bits_allocated - 1
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"

        if window_center is not None:
            ds.WindowCenter = window_center
        if window_width is not None:
            ds.WindowWidth = window_width

        if include_pixel_data:
            arr = np.full((rows, cols), pixel_value, dtype=np.uint16)
            ds.PixelData = arr.tobytes()

        ds.save_as(filepath)
        return filepath

    return _factory


@pytest.fixture()
def dicom_study(make_dicom_file):
    """Create a 5-slice DICOM study with varied pixel values."""
    paths = []
    for i in range(5):
        p = make_dicom_file(
            filename=f"slice_{i:03d}.dcm",
            pixel_value=i * 50,
        )
        paths.append(p)
    return paths
