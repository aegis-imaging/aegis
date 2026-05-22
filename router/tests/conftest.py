"""Shared fixtures."""

from __future__ import annotations

import os
import shutil
from pathlib import Path

import pytest
from pydicom.dataset import Dataset, FileMetaDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid


@pytest.fixture
def tmp_data_dir(tmp_path: Path) -> str:
    return str(tmp_path / "data")


@pytest.fixture
def tmp_quarantine_dir(tmp_path: Path) -> str:
    qdir = tmp_path / "quarantine"
    qdir.mkdir()
    return str(qdir)


@pytest.fixture
def base_env(tmp_data_dir: str, tmp_quarantine_dir: str, monkeypatch: pytest.MonkeyPatch) -> dict[str, str]:
    env = {
        "ROUTER_SITE_NAME": "test-site",
        "ROUTER_SITE_ID": "test-site-id",
        "ROUTER_DATA_DIR": tmp_data_dir,
        "ROUTER_QUARANTINE_DIR": tmp_quarantine_dir,
        "DIMSE_AE_TITLE": "TEST_AE",
        "DIMSE_PORT": "0",
        "MIDI_B_SALT": "test-salt",
        "OPERATOR_API_KEY": "",
    }
    for k, v in env.items():
        monkeypatch.setenv(k, v)
    # Wipe anything that may bleed in from the developer's shell.
    for k in (
        "DEFACING_SERVICE_URL",
        "PHI_DETECTION_SERVICE_URL",
        "QC_SERVICE_URL",
        "CLASSIFICATION_SERVICE_URL",
        "PROTOCOL_SERVICE_URL",
        "BIDS_SERVICE_URL",
        "ANALYTICS_SERVICE_URL",
        "SCT_SERVICE_URL",
        "SYNTH_SERVICE_URL",
        "CLOUD_RECEIVER_URL",
        "CLOUD_CLIENT_CERT",
        "CLOUD_CLIENT_KEY",
        "CLOUD_CA_CERT",
        "QUARANTINE_ALERT_URL",
    ):
        monkeypatch.delenv(k, raising=False)
    return env


def make_dicom(study_uid: str | None = None, series_uid: str | None = None, modality: str = "MR") -> bytes:
    """Build a tiny valid DICOM byte sequence for tests."""
    ds = Dataset()
    ds.StudyInstanceUID = study_uid or generate_uid()
    ds.SeriesInstanceUID = series_uid or generate_uid()
    ds.SOPInstanceUID = generate_uid()
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"  # MR Image Storage
    ds.PatientName = "Test^Patient"
    ds.PatientID = "TEST123"
    ds.Modality = modality
    ds.BodyPartExamined = "HEAD"
    ds.StudyDescription = "Brain"
    ds.StudyDate = "20260301"
    ds.is_little_endian = True
    ds.is_implicit_VR = False

    fm = FileMetaDataset()
    fm.MediaStorageSOPClassUID = ds.SOPClassUID
    fm.MediaStorageSOPInstanceUID = ds.SOPInstanceUID
    fm.TransferSyntaxUID = ExplicitVRLittleEndian
    ds.file_meta = fm

    import io

    from pydicom.filewriter import dcmwrite

    buf = io.BytesIO()
    dcmwrite(buf, ds, write_like_original=False)
    return buf.getvalue()
