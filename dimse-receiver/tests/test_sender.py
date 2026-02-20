"""Tests for app.sender helper functions."""

from __future__ import annotations

from app.sender import _sorted_dicom_files, _status_ok


class _Status:
    def __init__(self, code):
        self.Status = code


def test_status_ok_success():
    assert _status_ok(_Status(0x0000)) is True


def test_status_ok_warning():
    assert _status_ok(_Status(0xB000)) is True


def test_status_ok_failure():
    assert _status_ok(_Status(0xC000)) is False


def test_status_ok_none():
    assert _status_ok(None) is False


def test_sorted_dicom_files_numeric_order(tmp_path, monkeypatch):
    data_dir = tmp_path / "data"
    study_dir = data_dir / "dicom" / "raw" / "1.2.3"
    study_dir.mkdir(parents=True, exist_ok=True)
    (study_dir / "10.dcm").write_bytes(b"x")
    (study_dir / "2.dcm").write_bytes(b"x")
    (study_dir / "a.dcm").write_bytes(b"x")

    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(data_dir))
    files = _sorted_dicom_files("1.2.3", "raw")
    names = [f.name for f in files]

    assert names == ["2.dcm", "10.dcm", "a.dcm"]

