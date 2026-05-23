"""Tests for app.storage_backend.

The receiver delegates the local-filesystem path to
`dimse_core.storage.StudyLayout`. These tests cover the public surface
(`write_dicom`, `next_file_index`) end-to-end against a tmp data dir
without going through the DIMSE SCP, so a regression in storage logic
shows up here independently of test_scp.
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock

import pytest

from app import storage_backend


def test_write_dicom_local_writes_expected_path(tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.STORAGE_MODE", "local")
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))

    key = storage_backend.write_dicom("1.2.3.local", 0, b"payload-bytes")

    assert key == "dicom/raw/1.2.3.local/0.dcm"
    expected = tmp_path / "dicom" / "raw" / "1.2.3.local" / "0.dcm"
    assert expected.exists()
    assert expected.read_bytes() == b"payload-bytes"


def test_write_dicom_local_does_not_leave_tmp_file(tmp_path, monkeypatch):
    """StudyLayout writes via a `.tmp` rename. The final state must be clean."""
    monkeypatch.setattr("app.config.STORAGE_MODE", "local")
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))

    storage_backend.write_dicom("1.2.3.atomic", 0, b"x")

    study_dir = tmp_path / "dicom" / "raw" / "1.2.3.atomic"
    survivors = sorted(p.name for p in study_dir.iterdir())
    assert survivors == ["0.dcm"]


def test_next_file_index_counts_existing_files_only(tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.STORAGE_MODE", "local")
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))

    assert storage_backend.next_file_index("missing-study") == 0

    storage_backend.write_dicom("counted", 0, b"a")
    storage_backend.write_dicom("counted", 1, b"b")
    # Sibling non-DICOM files don't count.
    (tmp_path / "dicom" / "raw" / "counted" / "notes.txt").write_text("x")

    assert storage_backend.next_file_index("counted") == 2


def test_write_dicom_s3_uses_boto3(monkeypatch):
    """S3 mode is unchanged by the dimse-core migration."""
    monkeypatch.setattr("app.config.STORAGE_MODE", "s3")
    monkeypatch.setattr("app.config.S3_BUCKET", "test-bucket")
    monkeypatch.setattr("app.config.S3_REGION", "us-east-1")

    mock_s3 = MagicMock()
    mock_boto3 = MagicMock()
    mock_boto3.client.return_value = mock_s3

    with pytest.MonkeyPatch.context() as mp:
        mp.setitem(__import__("sys").modules, "boto3", mock_boto3)
        key = storage_backend.write_dicom("1.2.3.s3", 0, b"payload")

    assert key == "dicom/raw/1.2.3.s3/0.dcm"
    put_call = mock_s3.put_object.call_args
    assert put_call.kwargs["Bucket"] == "test-bucket"
    assert put_call.kwargs["Key"] == "dicom/raw/1.2.3.s3/0.dcm"
    assert put_call.kwargs["Body"] == b"payload"


def test_next_file_index_s3_counts_objects(monkeypatch):
    monkeypatch.setattr("app.config.STORAGE_MODE", "s3")
    monkeypatch.setattr("app.config.S3_BUCKET", "test-bucket")
    monkeypatch.setattr("app.config.S3_REGION", "us-east-1")

    mock_s3 = MagicMock()
    mock_s3.list_objects_v2.return_value = {"Contents": [{"Key": "a"}, {"Key": "b"}]}
    mock_boto3 = MagicMock()
    mock_boto3.client.return_value = mock_s3

    with pytest.MonkeyPatch.context() as mp:
        mp.setitem(__import__("sys").modules, "boto3", mock_boto3)
        assert storage_backend.next_file_index("1.2.3.s3") == 2

    list_call = mock_s3.list_objects_v2.call_args
    assert list_call.kwargs["Bucket"] == "test-bucket"
    assert list_call.kwargs["Prefix"] == "dicom/raw/1.2.3.s3/"
