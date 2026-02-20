"""Tests for app.scp C-STORE/C-ECHO/EVT_RELEASED handlers."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import pytest
from pydicom.uid import generate_uid

from app.ingest import StudyAccumulator
from app.scp import (
    _association_state,
    _get_file_index,
    create_scp,
    handle_echo,
    handle_release,
    handle_store,
)


class _Requestor:
    def __init__(self, ae_title: str = "PACS_AE"):
        self.ae_title = ae_title


class _Assoc:
    def __init__(self, ae_title: str = "PACS_AE"):
        self.requestor = _Requestor(ae_title=ae_title)


class _Event:
    def __init__(self, assoc, dataset=None, file_meta=None):
        self.assoc = assoc
        self.dataset = dataset
        self.file_meta = file_meta


def test_get_file_index_empty(tmp_path):
    assert _get_file_index(tmp_path / "missing") == 0


def test_get_file_index_counts_dcm_only(tmp_path):
    d = tmp_path / "study"
    d.mkdir()
    (d / "0.dcm").write_bytes(b"a")
    (d / "1.dcm").write_bytes(b"b")
    (d / "notes.txt").write_text("x")
    assert _get_file_index(d) == 2


def test_handle_echo_success():
    assoc = _Assoc()
    status = handle_echo(_Event(assoc=assoc))
    assert status == 0x0000


def test_handle_store_missing_study_uid(make_dicom_dataset, tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    ds = make_dicom_dataset()
    del ds.StudyInstanceUID

    assoc = _Assoc()
    status = handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta))

    assert status == 0xC000


def test_handle_store_writes_file_and_tracks_state(make_dicom_dataset, tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    ds = make_dicom_dataset(study_uid="1.2.3")

    assoc = _Assoc()
    status = handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta))

    assert status == 0x0000
    out = tmp_path / "dicom" / "raw" / "1.2.3" / "0.dcm"
    assert out.exists()

    studies = _association_state[id(assoc)]
    assert "1.2.3" in studies
    assert studies["1.2.3"].file_count == 1
    assert len(studies["1.2.3"].series_uids) == 1


def test_handle_store_increments_index(make_dicom_dataset, tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    study_uid = "1.2.840.1"
    series_uid_1 = generate_uid()
    series_uid_2 = generate_uid()

    assoc = _Assoc()
    ds1 = make_dicom_dataset(study_uid=study_uid, series_uid=series_uid_1)
    ds2 = make_dicom_dataset(study_uid=study_uid, series_uid=series_uid_2)

    assert handle_store(_Event(assoc=assoc, dataset=ds1, file_meta=ds1.file_meta)) == 0x0000
    assert handle_store(_Event(assoc=assoc, dataset=ds2, file_meta=ds2.file_meta)) == 0x0000

    study_dir = tmp_path / "dicom" / "raw" / study_uid
    assert (study_dir / "0.dcm").exists()
    assert (study_dir / "1.dcm").exists()

    acc = _association_state[id(assoc)][study_uid]
    assert acc.file_count == 2
    assert len(acc.series_uids) == 2


def test_handle_release_triggers_ingest_per_study():
    assoc = _Assoc()
    assoc_id = id(assoc)
    _association_state[assoc_id] = {
        "study1": StudyAccumulator(study_instance_uid="study1", file_count=2, series_uids={"s1"}),
        "study2": StudyAccumulator(study_instance_uid="study2", file_count=1, series_uids={"s2"}),
    }

    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_release(_Event(assoc=assoc))

    assert mock_trigger.call_count == 2
    assert assoc_id not in _association_state


def test_handle_release_empty_state():
    assoc = _Assoc()
    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_release(_Event(assoc=assoc))
    mock_trigger.assert_not_called()


def test_create_scp_uses_config(monkeypatch):
    pytest.importorskip("pynetdicom")
    monkeypatch.setattr("app.config.DIMSE_AE_TITLE", "AEGIS")
    monkeypatch.setattr("app.config.DIMSE_MAX_ASSOCIATIONS", 7)
    ae = create_scp()

    assert ae.maximum_associations == 7
    assert str(ae.ae_title).strip() == "AEGIS"
    assert len(ae.supported_contexts) > 0
