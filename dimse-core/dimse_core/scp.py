"""DICOM C-STORE SCP factory.

Both the cloud `dimse-receiver` and the spoke `router` want exactly the same
behavior at the wire: listen for C-STORE on a port, write every incoming
instance to disk per-study, accumulate study-level metadata during the
association, and notify the caller when the association releases. The
difference is purely what each caller does with the notification — cloud
ingests into the API, router hands off to a local pipeline. We give the
caller a single callback (`on_study_complete`) and stay out of their way.

Usage:
    cfg = SCPConfig(ae_title="AEGIS", port=11112, data_dir="/var/aegis/data")

    def on_complete(acc: StudyAccumulator) -> None:
        # do whatever the caller wants — enqueue, ingest, etc.
        ...

    ae = create_ae(cfg, on_complete)
    start_listening(ae, cfg)  # blocks
"""

from __future__ import annotations

import io
import logging
import threading
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Iterable

log = logging.getLogger(__name__)


# All common transfer syntaxes — including compressed formats — so PACS can
# send JPEG2000 / JPEG-LS / RLE without transcoding first.
ACCEPTED_TRANSFER_SYNTAXES: tuple[str, ...] = (
    "1.2.840.10008.1.2",        # Implicit VR Little Endian
    "1.2.840.10008.1.2.1",      # Explicit VR Little Endian
    "1.2.840.10008.1.2.2",      # Explicit VR Big Endian
    "1.2.840.10008.1.2.4.50",   # JPEG Baseline
    "1.2.840.10008.1.2.4.51",   # JPEG Extended
    "1.2.840.10008.1.2.4.57",   # JPEG Lossless
    "1.2.840.10008.1.2.4.70",   # JPEG Lossless SV1
    "1.2.840.10008.1.2.4.80",   # JPEG-LS Lossless
    "1.2.840.10008.1.2.4.81",   # JPEG-LS Near-Lossless
    "1.2.840.10008.1.2.4.90",   # JPEG 2000 Lossless
    "1.2.840.10008.1.2.4.91",   # JPEG 2000
    "1.2.840.10008.1.2.5",      # RLE Lossless
)


@dataclass
class StudyAccumulator:
    """Per-study state collected during a single DICOM association.

    The SCP populates this incrementally as C-STOREs arrive; on EVT_RELEASED
    the caller's `on_study_complete` callback is invoked with it.
    """

    study_instance_uid: str
    modality: str = ""
    body_part: str = ""
    study_description: str = ""
    study_date: str = ""
    series_uids: set[str] = field(default_factory=set)
    file_count: int = 0
    calling_ae_title: str = ""
    storage_keys: list[str] = field(default_factory=list)


@dataclass
class SCPConfig:
    """Everything the SCP needs to know.

    `write_dicom_fn` lets the caller plug in custom storage (e.g. S3 instead
    of local disk). If None, the SCP writes to `data_dir/dicom/raw/<study>/<i>.dcm`.
    """

    ae_title: str
    port: int = 11112
    bind_host: str = "0.0.0.0"
    data_dir: str = "/var/lib/aegis"
    max_associations: int = 10
    accepted_transfer_syntaxes: tuple[str, ...] = ACCEPTED_TRANSFER_SYNTAXES
    write_dicom_fn: Callable[["SCPConfig", str, int, bytes], str] | None = None


def default_write_dicom(cfg: SCPConfig, study_uid: str, file_index: int, data: bytes) -> str:
    """Default storage: write to `{data_dir}/dicom/raw/<study_uid>/<index>.dcm`.

    Returns the relative storage key. Callers wanting S3 / GCS / arbitrary
    layouts can swap this via SCPConfig.write_dicom_fn.
    """
    key = f"dicom/raw/{study_uid}/{file_index}.dcm"
    path = Path(cfg.data_dir) / key
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(".tmp")
    tmp.write_bytes(data)
    tmp.replace(path)
    return key


def default_next_index(cfg: SCPConfig, study_uid: str) -> int:
    """Count existing .dcm files under default storage layout."""
    study_dir = Path(cfg.data_dir) / "dicom" / "raw" / study_uid
    if not study_dir.exists():
        return 0
    return sum(1 for _ in study_dir.glob("*.dcm"))


# ── Internal state ─────────────────────────────────────────────────────────

# Maps association id → {study_uid: StudyAccumulator}
_assoc_state: dict[int, dict[str, "StudyAccumulator"]] = {}
_state_lock = threading.Lock()


def _handle_echo(event: Any) -> int:
    log.debug("C-ECHO from %s", _safe_ae_title(event))
    return 0x0000


def _handle_store(event: Any, cfg: SCPConfig, next_index_fn: Callable[[SCPConfig, str], int]) -> int:
    ds = event.dataset
    ds.file_meta = event.file_meta

    study_uid = getattr(ds, "StudyInstanceUID", None)
    if not study_uid:
        log.warning("C-STORE rejected: missing StudyInstanceUID")
        return 0xC000
    study_uid = str(study_uid)

    modality = str(getattr(ds, "Modality", ""))
    body_part = str(getattr(ds, "BodyPartExamined", ""))
    study_desc = str(getattr(ds, "StudyDescription", ""))
    study_date = str(getattr(ds, "StudyDate", ""))
    series_uid = str(getattr(ds, "SeriesInstanceUID", ""))
    calling_ae = _safe_ae_title(event)

    # Serialize + persist.
    try:
        buf = io.BytesIO()
        ds.save_as(buf, write_like_original=False)
        body = buf.getvalue()
        idx = next_index_fn(cfg, study_uid)
        write = cfg.write_dicom_fn or default_write_dicom
        key = write(cfg, study_uid, idx, body)
    except Exception:
        log.exception("C-STORE failed to persist instance for %s", study_uid)
        return 0xA700  # Out of resources

    # Update per-association accumulator.
    assoc_id = id(event.assoc)
    with _state_lock:
        per_assoc = _assoc_state.setdefault(assoc_id, {})
        acc = per_assoc.get(study_uid)
        if acc is None:
            acc = StudyAccumulator(
                study_instance_uid=study_uid,
                modality=modality,
                body_part=body_part,
                study_description=study_desc,
                study_date=study_date,
                calling_ae_title=calling_ae,
            )
            per_assoc[study_uid] = acc
        acc.file_count += 1
        acc.storage_keys.append(key)
        if series_uid:
            acc.series_uids.add(series_uid)
        # Later instances may carry better metadata than the first.
        if not acc.modality and modality:
            acc.modality = modality
        if not acc.body_part and body_part:
            acc.body_part = body_part
        if not acc.study_description and study_desc:
            acc.study_description = study_desc
        if not acc.study_date and study_date:
            acc.study_date = study_date
    return 0x0000


def _handle_released(event: Any, on_study_complete: Callable[[StudyAccumulator], None]) -> None:
    assoc_id = id(event.assoc)
    with _state_lock:
        per_assoc = _assoc_state.pop(assoc_id, {})
    for acc in per_assoc.values():
        try:
            on_study_complete(acc)
        except Exception:
            log.exception("on_study_complete failed for %s", acc.study_instance_uid)


def _safe_ae_title(event: Any) -> str:
    try:
        return str(event.assoc.requestor.ae_title).strip()
    except Exception:
        return ""


# ── Public factory ─────────────────────────────────────────────────────────

def create_ae(
    cfg: SCPConfig,
    on_study_complete: Callable[[StudyAccumulator], None],
    *,
    next_index_fn: Callable[[SCPConfig, str], int] | None = None,
) -> Any:
    """Build a pynetdicom AE configured to accept all Storage + Verification.

    Returns the AE object; pass it to start_listening() to run.
    """
    AE, all_storage_contexts, verification_contexts, evt, build_context = _load_pynetdicom()

    ae = AE(ae_title=cfg.ae_title)

    contexts = [
        build_context(ctx.abstract_syntax, list(cfg.accepted_transfer_syntaxes))
        for ctx in all_storage_contexts
    ]
    ae.supported_contexts = contexts + verification_contexts
    ae.maximum_associations = cfg.max_associations

    index_fn = next_index_fn or default_next_index
    # Store handlers + cfg so start_listening can bind them.
    ae._dimse_core_handlers = [  # type: ignore[attr-defined]
        (evt.EVT_C_ECHO, _handle_echo),
        (evt.EVT_C_STORE, _handle_store, [cfg, index_fn]),
        (evt.EVT_RELEASED, _handle_released, [on_study_complete]),
    ]
    return ae


def start_listening(ae: Any, cfg: SCPConfig) -> None:
    """Block-listening loop. Run in a daemon thread if you want non-blocking."""
    handlers = getattr(ae, "_dimse_core_handlers", None)
    if handlers is None:
        raise RuntimeError("AE was not built with dimse_core.create_ae; cannot start_listening")
    log.info("dimse-core SCP listening on %s:%d (AE=%s, max_assoc=%d)",
             cfg.bind_host, cfg.port, cfg.ae_title, cfg.max_associations)
    ae.start_server((cfg.bind_host, cfg.port), evt_handlers=handlers, block=True)


def reset_state_for_tests() -> None:
    """Clear association state — for tests only."""
    with _state_lock:
        _assoc_state.clear()


# ── Optional pynetdicom loader (lazy so the package imports without it) ────

def _load_pynetdicom() -> tuple[Any, Iterable[Any], Iterable[Any], Any, Any]:
    from pynetdicom import (
        AE,
        AllStoragePresentationContexts,
        VerificationPresentationContexts,
        evt,
    )
    from pynetdicom.presentation import build_context

    return AE, AllStoragePresentationContexts, VerificationPresentationContexts, evt, build_context
