"""DICOM C-STORE SCP that receives instances from the spoke's PACS.

Implementation notes:
- Per-association state map keyed by association id; each association may carry
  one or more studies (a PACS may auto-route a series mid-association).
- On EVT_RELEASED, each completed study is handed to the pipeline orchestrator.
  We don't wait for "study complete" semantics — most PACSes don't signal it,
  and the orchestrator is idempotent if more instances arrive later for the
  same StudyInstanceUID (it merges, re-runs de-id on the additional instances,
  and re-attempts shipment).
"""

from __future__ import annotations

import logging
import threading
from dataclasses import dataclass, field
from typing import Callable

from pydicom.dataset import Dataset
from pynetdicom import AE, ALL_TRANSFER_SYNTAXES, evt
from pynetdicom.events import Event
from pynetdicom.sop_class import Verification

from app import storage
from app.config import RouterConfig

log = logging.getLogger(__name__)


@dataclass
class StudyAccumulator:
    study_uid: str
    series_uids: set[str] = field(default_factory=set)
    instance_count: int = 0
    modality: str = ""
    body_part: str = ""
    study_description: str = ""
    study_date: str = ""
    calling_ae_title: str = ""

    def update_from_dataset(self, ds: Dataset) -> None:
        if "SeriesInstanceUID" in ds:
            self.series_uids.add(str(ds.SeriesInstanceUID))
        if not self.modality and "Modality" in ds:
            self.modality = str(ds.Modality)
        if not self.body_part and "BodyPartExamined" in ds:
            self.body_part = str(ds.BodyPartExamined)
        if not self.study_description and "StudyDescription" in ds:
            self.study_description = str(ds.StudyDescription)
        if not self.study_date and "StudyDate" in ds:
            self.study_date = str(ds.StudyDate)
        self.instance_count += 1


# Map association_id -> {study_uid: StudyAccumulator}
_state: dict[int, dict[str, StudyAccumulator]] = {}
_state_lock = threading.Lock()


def _handle_echo(event: Event) -> int:
    return 0x0000  # Success


def _handle_store(event: Event, cfg: RouterConfig) -> int:
    ds = event.dataset
    ds.file_meta = event.file_meta

    if "StudyInstanceUID" not in ds:
        log.warning("C-STORE rejected: missing StudyInstanceUID")
        return 0xC000

    study_uid = str(ds.StudyInstanceUID)
    paths = storage.layout(cfg.data_dir, study_uid)
    assoc_id = id(event.assoc)

    with _state_lock:
        per_assoc = _state.setdefault(assoc_id, {})
        acc = per_assoc.get(study_uid)
        if acc is None:
            acc = StudyAccumulator(
                study_uid=study_uid,
                calling_ae_title=str(event.assoc.requestor.ae_title).strip() if event.assoc else "",
            )
            per_assoc[study_uid] = acc
        acc.update_from_dataset(ds)
        next_idx = storage.next_raw_index(paths)

    try:
        body = bytes(event.encoded_dataset())
        storage.write_raw_bytes(paths, next_idx, body)
    except Exception:
        log.exception("C-STORE failed to persist instance for %s", study_uid)
        return 0xA700  # Out of resources

    return 0x0000


def _handle_release(event: Event, on_study_complete: Callable[[StudyAccumulator], None]) -> None:
    assoc_id = id(event.assoc)
    with _state_lock:
        studies = _state.pop(assoc_id, {})
    for acc in studies.values():
        try:
            on_study_complete(acc)
        except Exception:
            log.exception("study orchestrator failed for %s", acc.study_uid)


def create_ae(cfg: RouterConfig, on_study_complete: Callable[[StudyAccumulator], None]) -> AE:
    ae = AE(ae_title=cfg.dimse_ae_title)
    ae.maximum_associations = cfg.dimse_max_associations

    storage_sop_classes = _all_storage_sop_classes()
    for sop in storage_sop_classes:
        ae.add_supported_context(sop, ALL_TRANSFER_SYNTAXES)
    ae.add_supported_context(Verification, ALL_TRANSFER_SYNTAXES)

    handlers = [
        (evt.EVT_C_ECHO, _handle_echo),
        (evt.EVT_C_STORE, _handle_store, [cfg]),
        (evt.EVT_RELEASED, _handle_release, [on_study_complete]),
    ]
    ae._handlers_to_bind = handlers  # used by start_listening
    return ae


def start_listening(ae: AE) -> None:
    """Block-listening in a daemon thread. Returns when ae.shutdown() is called."""
    handlers = getattr(ae, "_handlers_to_bind", [])
    ae.start_server(("0.0.0.0", _scp_port_for(ae)), block=True, evt_handlers=handlers)


def _scp_port_for(ae: AE) -> int:
    # Read from the same env the rest of the service uses so tests can override.
    from app.config import RouterConfig as _RC

    return _RC.load().dimse_port


def _all_storage_sop_classes() -> list[str]:
    # Pull all Storage SOP classes pynetdicom knows about. That covers MR, CT,
    # PET, US, SC, etc. without us maintaining a list.
    from pynetdicom import sop_class as _sops

    out: list[str] = []
    for name in dir(_sops):
        if not name.endswith("Storage") and not name.endswith("StorageSOPClass"):
            continue
        val = getattr(_sops, name)
        if isinstance(val, str) and val.startswith("1."):
            out.append(val)
    return out
