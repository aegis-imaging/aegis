"""SCP handler unit tests.

Tests exercise the handler functions directly with mocked pynetdicom events,
exactly the pattern dimse-receiver/tests/test_scp.py uses. We don't actually
bind a TCP socket here — that's covered by the existing dimse-receiver
integration tests.
"""
from __future__ import annotations

import io
from dataclasses import dataclass
from pathlib import Path

import pydicom

from dimse_core import scp
from dimse_core.scp import SCPConfig, StudyAccumulator
from tests.conftest import make_dicom_bytes


@dataclass
class _Requestor:
    ae_title: str = "TEST_SCU"


@dataclass
class _Assoc:
    requestor: _Requestor


@dataclass
class _Event:
    dataset: pydicom.Dataset
    file_meta: pydicom.dataset.FileMetaDataset
    assoc: _Assoc


def _make_event(study_uid: str | None = None, calling_ae: str = "TEST_SCU"):
    raw = make_dicom_bytes(study_uid=study_uid)
    ds = pydicom.dcmread(io.BytesIO(raw))
    file_meta = ds.file_meta
    return _Event(dataset=ds, file_meta=file_meta, assoc=_Assoc(_Requestor(calling_ae)))


def test_handle_store_writes_file_and_updates_accumulator(tmp_data_dir):
    scp.reset_state_for_tests()
    cfg = SCPConfig(ae_title="TEST", data_dir=tmp_data_dir)
    event = _make_event(study_uid="1.2.3.4")
    rc = scp._handle_store(event, cfg, scp.default_next_index)
    assert rc == 0x0000
    # File written under default layout
    files = list((Path(tmp_data_dir) / "dicom" / "raw" / "1.2.3.4").glob("*.dcm"))
    assert len(files) == 1


def test_handle_store_rejects_missing_study_uid(tmp_data_dir):
    scp.reset_state_for_tests()
    cfg = SCPConfig(ae_title="TEST", data_dir=tmp_data_dir)
    event = _make_event(study_uid=None)
    # Manually strip StudyInstanceUID to simulate a malformed C-STORE
    if "StudyInstanceUID" in event.dataset:
        del event.dataset.StudyInstanceUID
    rc = scp._handle_store(event, cfg, scp.default_next_index)
    assert rc == 0xC000


def test_release_invokes_callback_per_study(tmp_data_dir):
    scp.reset_state_for_tests()
    cfg = SCPConfig(ae_title="TEST", data_dir=tmp_data_dir)
    completed: list[StudyAccumulator] = []

    def on_complete(acc):
        completed.append(acc)

    event = _make_event(study_uid="1.2.3.4")
    scp._handle_store(event, cfg, scp.default_next_index)
    # Two instances in the same association → one accumulator, one callback.
    event2 = _make_event(study_uid="1.2.3.4")
    event2.assoc = event.assoc  # same association
    scp._handle_store(event2, cfg, scp.default_next_index)

    scp._handle_released(event, on_complete)
    assert len(completed) == 1
    assert completed[0].file_count == 2
    assert completed[0].study_instance_uid == "1.2.3.4"


def test_release_clears_assoc_state(tmp_data_dir):
    scp.reset_state_for_tests()
    cfg = SCPConfig(ae_title="TEST", data_dir=tmp_data_dir)
    event = _make_event()
    scp._handle_store(event, cfg, scp.default_next_index)
    scp._handle_released(event, lambda acc: None)
    # A second release for the same assoc should be a no-op (no callback).
    called = []
    scp._handle_released(event, lambda acc: called.append(acc))
    assert called == []


def test_custom_write_fn_called(tmp_data_dir):
    scp.reset_state_for_tests()
    written = []

    def my_write(cfg, study_uid, idx, body):
        written.append((study_uid, idx, len(body)))
        return f"custom://{study_uid}/{idx}"

    cfg = SCPConfig(ae_title="TEST", data_dir=tmp_data_dir, write_dicom_fn=my_write)
    event = _make_event(study_uid="abc")
    scp._handle_store(event, cfg, lambda c, s: 0)
    assert len(written) == 1
    assert written[0][0] == "abc"


def test_metadata_extracted_from_dataset(tmp_data_dir):
    scp.reset_state_for_tests()
    cfg = SCPConfig(ae_title="TEST", data_dir=tmp_data_dir)
    event = _make_event(study_uid="metadata-test")
    completed: list[StudyAccumulator] = []
    scp._handle_store(event, cfg, scp.default_next_index)
    scp._handle_released(event, completed.append)
    assert completed[0].modality == "MR"
    assert completed[0].body_part == "HEAD"
    assert completed[0].calling_ae_title == "TEST_SCU"
