"""Shared fixtures for qc-service tests.

Generates synthetic DICOM files with controlled pixel data for deterministic
QC testing (especially SNR). Uses pydicom + numpy.
"""

import numpy as np
import pydicom
import pytest
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid


@pytest.fixture
def make_dicom_file():
    """Factory fixture — creates a minimal DICOM file with controlled pixel data.

    Parameters:
        pixel_array: numpy uint16 array (rows × cols). If None, generates uniform 500.
        rows, cols: image dimensions (used only if pixel_array is None).
        modality, body_part: DICOM tags.
        pixel_spacing: [row_spacing, col_spacing] or None.
        image_position: [x, y, z] or None.
        slice_location: float or None.
        instance_number: integer.
        include_pixel_data: if False, omits PixelData entirely.
    """

    def _make(
        tmp_dir,
        filename="test.dcm",
        rows=64,
        cols=64,
        pixel_array=None,
        modality="MR",
        body_part="HEAD",
        pixel_spacing=None,
        image_position=None,
        slice_location=None,
        instance_number=1,
        study_uid=None,
        series_uid=None,
        include_pixel_data=True,
    ):
        filepath = str(tmp_dir / filename)
        sop_uid = generate_uid()
        file_meta = pydicom.Dataset()
        file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
        file_meta.MediaStorageSOPInstanceUID = sop_uid
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)
        ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
        ds.SOPInstanceUID = sop_uid
        ds.StudyInstanceUID = study_uid or generate_uid()
        ds.SeriesInstanceUID = series_uid or generate_uid()
        ds.Modality = modality
        ds.BodyPartExamined = body_part
        ds.InstanceNumber = instance_number
        ds.Rows = rows
        ds.Columns = cols
        ds.BitsAllocated = 16
        ds.BitsStored = 16
        ds.HighBit = 15
        ds.PixelRepresentation = 0
        ds.SamplesPerPixel = 1
        ds.PhotometricInterpretation = "MONOCHROME2"

        if pixel_spacing is not None:
            ds.PixelSpacing = pixel_spacing

        if image_position is not None:
            ds.ImagePositionPatient = image_position

        if slice_location is not None:
            ds.SliceLocation = slice_location

        if include_pixel_data:
            if pixel_array is not None:
                ds.PixelData = pixel_array.astype(np.uint16).tobytes()
                ds.Rows = pixel_array.shape[0]
                ds.Columns = pixel_array.shape[1]
            else:
                arr = np.full((rows, cols), 500, dtype=np.uint16)
                ds.PixelData = arr.tobytes()

        ds.save_as(filepath)
        return filepath

    return _make


@pytest.fixture
def dicom_study(tmp_path, make_dicom_file):
    """Creates a 5-slice MR EXTREMITY study with high SNR and uniform 1mm spacing.

    Uses EXTREMITY body part (not in expected_counts map) so coverage always passes.
    """
    study_uid = generate_uid()
    series_uid = generate_uid()
    paths = []
    for i in range(5):
        # High-signal uniform image — corner and body both ~500
        # Add tiny corner variation so SNR is computable (std > 0)
        arr = np.full((64, 64), 500, dtype=np.uint16)
        # Corner region (6×6) gets small variation: 498-502
        arr[:6, :6] = 500 + np.arange(36).reshape(6, 6) % 5 - 2  # values 498–502
        p = make_dicom_file(
            tmp_path,
            filename=f"slice_{i}.dcm",
            pixel_array=arr,
            body_part="EXTREMITY",
            pixel_spacing=[1.0, 1.0],
            image_position=[0.0, 0.0, float(i)],
            slice_location=float(i),
            instance_number=i + 1,
            study_uid=study_uid,
            series_uid=series_uid,
        )
        paths.append(p)
    return paths
