"""Tests for app.scp C-STORE/C-ECHO/EVT_RELEASED/EVT_ABORTED handlers."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest
from pydicom.uid import generate_uid

from app.ingest import StudyAccumulator
from app.scp import (
    _association_state,
    create_scp,
    handle_abort,
    handle_conn_close,
    handle_echo,
    handle_release,
    handle_store,
)
from app.storage_backend import validate_uid


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


def test_validate_uid_accepts_dicom_uids():
    assert validate_uid("1.2.840.10008.1.2") == "1.2.840.10008.1.2"


@pytest.mark.parametrize("bad", ["", "../../etc/passwd", "1.2.3/evil", "1" * 65])
def test_validate_uid_rejects_unsafe_values(bad):
    with pytest.raises(ValueError):
        validate_uid(bad)


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


def test_handle_store_missing_sop_instance_uid(make_dicom_dataset, tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    ds = make_dicom_dataset(study_uid="1.2.3")
    del ds.SOPInstanceUID

    assoc = _Assoc()
    status = handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta))

    assert status == 0xC000


def test_handle_store_writes_file_and_tracks_state(make_dicom_dataset, tmp_path, monkeypatch):
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    ds = make_dicom_dataset(study_uid="1.2.3")

    assoc = _Assoc()
    status = handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta))

    assert status == 0x0000
    out = tmp_path / "dicom" / "raw" / "1.2.3" / f"{ds.SOPInstanceUID}.dcm"
    assert out.exists()

    studies = _association_state[id(assoc)]
    assert "1.2.3" in studies
    assert studies["1.2.3"].file_count == 1
    assert len(studies["1.2.3"].series_uids) == 1


def test_handle_store_names_files_by_sop_instance_uid(make_dicom_dataset, tmp_path, monkeypatch):
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
    assert (study_dir / f"{ds1.SOPInstanceUID}.dcm").exists()
    assert (study_dir / f"{ds2.SOPInstanceUID}.dcm").exists()
    assert len(list(study_dir.glob("*.dcm"))) == 2

    acc = _association_state[id(assoc)][study_uid]
    assert acc.file_count == 2
    assert len(acc.series_uids) == 2


def test_handle_store_resend_same_instance_is_idempotent(make_dicom_dataset, tmp_path, monkeypatch):
    """Re-sending the same SOP instance overwrites its own file, not another slice."""
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    study_uid = "1.2.840.2"
    ds = make_dicom_dataset(study_uid=study_uid)

    assoc = _Assoc()
    assert handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta)) == 0x0000
    assert handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta)) == 0x0000

    study_dir = tmp_path / "dicom" / "raw" / study_uid
    assert len(list(study_dir.glob("*.dcm"))) == 1
    assert (study_dir / f"{ds.SOPInstanceUID}.dcm").exists()


def test_handle_store_concurrent_associations_do_not_collide(make_dicom_dataset, tmp_path, monkeypatch):
    """Two associations receiving the same study write distinct files."""
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    study_uid = "1.2.840.3"
    ds1 = make_dicom_dataset(study_uid=study_uid)
    ds2 = make_dicom_dataset(study_uid=study_uid)

    assoc1 = _Assoc()
    assoc2 = _Assoc()
    assert handle_store(_Event(assoc=assoc1, dataset=ds1, file_meta=ds1.file_meta)) == 0x0000
    assert handle_store(_Event(assoc=assoc2, dataset=ds2, file_meta=ds2.file_meta)) == 0x0000

    study_dir = tmp_path / "dicom" / "raw" / study_uid
    assert (study_dir / f"{ds1.SOPInstanceUID}.dcm").exists()
    assert (study_dir / f"{ds2.SOPInstanceUID}.dcm").exists()
    assert len(list(study_dir.glob("*.dcm"))) == 2


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
    _association_state.clear()  # prevent id() reuse from leaking prior test state
    assoc = _Assoc()
    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_release(_Event(assoc=assoc))
    mock_trigger.assert_not_called()


def test_handle_abort_triggers_ingest_and_clears_state():
    """A-ABORT after receiving files still ingests them and frees state."""
    assoc = _Assoc()
    assoc_id = id(assoc)
    _association_state[assoc_id] = {
        "study1": StudyAccumulator(study_instance_uid="study1", file_count=3, series_uids={"s1"}),
    }

    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_abort(_Event(assoc=assoc))

    assert mock_trigger.call_count == 1
    assert assoc_id not in _association_state


def test_handle_conn_close_triggers_ingest_and_clears_state():
    """Connection drop after receiving files still ingests them and frees state."""
    assoc = _Assoc()
    assoc_id = id(assoc)
    _association_state[assoc_id] = {
        "study1": StudyAccumulator(study_instance_uid="study1", file_count=2, series_uids={"s1"}),
    }

    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_conn_close(_Event(assoc=assoc))

    assert mock_trigger.call_count == 1
    assert assoc_id not in _association_state


def test_release_then_conn_close_does_not_double_ingest():
    """EVT_RELEASED then EVT_CONN_CLOSE both fire — ingest must happen once."""
    assoc = _Assoc()
    assoc_id = id(assoc)
    _association_state[assoc_id] = {
        "study1": StudyAccumulator(study_instance_uid="study1", file_count=1, series_uids={"s1"}),
    }

    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_release(_Event(assoc=assoc))
        handle_conn_close(_Event(assoc=assoc))

    assert mock_trigger.call_count == 1
    assert assoc_id not in _association_state


def test_handle_abort_empty_state_is_noop():
    _association_state.clear()
    assoc = _Assoc()
    with patch("app.scp.submit_ingest", return_value=True) as mock_trigger:
        handle_abort(_Event(assoc=assoc))
    mock_trigger.assert_not_called()


def test_create_scp_uses_config(monkeypatch):
    pytest.importorskip("pynetdicom")
    monkeypatch.setattr("app.config.DIMSE_AE_TITLE", "AEGIS")
    monkeypatch.setattr("app.config.DIMSE_MAX_ASSOCIATIONS", 7)
    ae = create_scp()

    assert ae.maximum_associations == 7
    assert str(ae.ae_title).strip() == "AEGIS"
    assert len(ae.supported_contexts) > 0


# ── Storage backend integration ──────────────────────────────────────────────


def test_handle_store_local_mode_writes_file(make_dicom_dataset, tmp_path, monkeypatch):
    """STORAGE_MODE=local (default) writes to the filesystem as before."""
    monkeypatch.setattr("app.config.STORAGE_MODE", "local")
    monkeypatch.setattr("app.config.DIMSE_DATA_DIR", str(tmp_path))
    ds = make_dicom_dataset(study_uid="1.2.3.local")

    assoc = _Assoc()
    status = handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta))

    assert status == 0x0000
    out = tmp_path / "dicom" / "raw" / "1.2.3.local" / f"{ds.SOPInstanceUID}.dcm"
    assert out.exists()


def test_handle_store_s3_mode_calls_put_object(make_dicom_dataset, monkeypatch):
    """STORAGE_MODE=s3 writes to S3 via boto3 and does NOT touch the filesystem."""
    monkeypatch.setattr("app.config.STORAGE_MODE", "s3")
    monkeypatch.setattr("app.config.S3_BUCKET", "test-bucket")
    monkeypatch.setattr("app.config.S3_REGION", "us-east-1")

    mock_s3 = MagicMock()
    mock_boto3 = MagicMock()
    mock_boto3.client.return_value = mock_s3

    ds = make_dicom_dataset(study_uid="1.2.3.s3")

    with patch.dict("sys.modules", {"boto3": mock_boto3}):
        # Re-import storage_backend so it picks up the mocked boto3
        import importlib
        import app.storage_backend as sb
        importlib.reload(sb)

        assoc = _Assoc()
        status = handle_store(_Event(assoc=assoc, dataset=ds, file_meta=ds.file_meta))

    assert status == 0x0000
    # SOP-UID naming needs no LIST round-trip before each write.
    mock_s3.list_objects_v2.assert_not_called()
    # Verify S3 put_object was called with the correct key
    put_calls = mock_s3.put_object.call_args_list
    assert len(put_calls) == 1
    call_kwargs = put_calls[0].kwargs
    assert call_kwargs["Bucket"] == "test-bucket"
    assert call_kwargs["Key"] == f"dicom/raw/1.2.3.s3/{ds.SOPInstanceUID}.dcm"
    assert isinstance(call_kwargs["Body"], bytes)
    assert len(call_kwargs["Body"]) > 0
