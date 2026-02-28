"""DICOM C-STORE SCP handler using pynetdicom."""

from __future__ import annotations

import io
import logging
import threading
from pathlib import Path
from typing import Any

from app import config
from app.ingest import StudyAccumulator, submit_ingest
from app.storage_backend import next_file_index, write_dicom

log = logging.getLogger(__name__)

# Per-association state: maps association id -> {study_uid -> StudyAccumulator}
_association_state: dict[int, dict[str, StudyAccumulator]] = {}
_state_lock = threading.Lock()


def _load_pynetdicom():
    """Import pynetdicom lazily so module import works without optional deps."""
    from pynetdicom import (
        AE,
        AllStoragePresentationContexts,
        VerificationPresentationContexts,
        evt,
    )
    from pynetdicom.presentation import build_context

    return AE, AllStoragePresentationContexts, VerificationPresentationContexts, evt, build_context


# All transfer syntaxes we accept — includes compressed formats so PACS can
# send JPEG2000, JPEG-LS, and RLE without transcoding first.
ACCEPTED_TRANSFER_SYNTAXES = [
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
]


def _get_file_index(study_dir: Path) -> int:
    """Return the next 0-based file index for a study directory.

    Counts existing .dcm files so re-sends don't overwrite.
    """
    if not study_dir.exists():
        return 0
    return len(list(study_dir.glob("*.dcm")))


def handle_echo(event: Any) -> int:
    """Handle C-ECHO (verification) requests."""
    log.debug("C-ECHO from %s", event.assoc.requestor.ae_title)
    return 0x0000  # Success


def handle_store(event: Any) -> int:
    """Handle C-STORE requests — save DICOM file and accumulate metadata."""
    ds = event.dataset
    ds.file_meta = event.file_meta

    study_uid = getattr(ds, "StudyInstanceUID", None)
    if not study_uid:
        log.warning("C-STORE: missing StudyInstanceUID, rejecting")
        return 0xC000  # Failure

    study_uid = str(study_uid)

    # Extract metadata
    modality = str(getattr(ds, "Modality", ""))
    body_part = str(getattr(ds, "BodyPartExamined", ""))
    study_desc = str(getattr(ds, "StudyDescription", ""))
    series_uid = str(getattr(ds, "SeriesInstanceUID", ""))
    calling_ae = str(getattr(event.assoc.requestor, "ae_title", ""))

    # Write file to configured storage backend (local filesystem or S3).
    try:
        file_index = next_file_index(study_uid)
        buf = io.BytesIO()
        ds.save_as(buf, write_like_original=False)
        write_dicom(study_uid, file_index, buf.getvalue())
    except Exception as e:
        log.error("Failed to save DICOM file for study %s: %s", study_uid, e)
        return 0xC000  # Failure

    log.debug("Saved file %d for study %s", file_index, study_uid)

    # Update per-association accumulator
    assoc_id = id(event.assoc)
    with _state_lock:
        if assoc_id not in _association_state:
            _association_state[assoc_id] = {}
        studies = _association_state[assoc_id]
        if study_uid not in studies:
            studies[study_uid] = StudyAccumulator(
                study_instance_uid=study_uid,
                modality=modality,
                body_part=body_part,
                study_description=study_desc,
                calling_ae_title=calling_ae,
            )
        acc = studies[study_uid]
        acc.file_count += 1
        if series_uid:
            acc.series_uids.add(series_uid)
        # Update metadata if we get better info from later files
        if not acc.modality and modality:
            acc.modality = modality
        if not acc.body_part and body_part:
            acc.body_part = body_part

    return 0x0000  # Success


def handle_release(event: Any) -> None:
    """Handle association release — trigger ingest for each received study."""
    assoc_id = id(event.assoc)

    with _state_lock:
        studies = _association_state.pop(assoc_id, {})

    if not studies:
        log.debug("Association released with no stored files")
        return

    for study_uid, acc in studies.items():
        log.info(
            "Association released — ingesting study %s (%d files, %d series, AE: %s)",
            study_uid,
            acc.file_count,
            len(acc.series_uids),
            acc.calling_ae_title,
        )
        try:
            submit_ingest(acc)
        except Exception as e:
            log.error("Ingest trigger failed for %s: %s", study_uid, e)


def create_scp():
    """Create and configure the DICOM Application Entity.

    Accepts all storage SOP classes with all common transfer syntaxes
    including compressed formats (JPEG2000, JPEG-LS, RLE). This allows
    PACS systems to send compressed DICOM without transcoding first.
    """
    AE, all_storage_contexts, verification_contexts, _, build_context = _load_pynetdicom()

    ae = AE(ae_title=config.DIMSE_AE_TITLE)

    # Build contexts with all transfer syntaxes for each SOP class
    contexts = []
    for ctx in all_storage_contexts:
        contexts.append(build_context(ctx.abstract_syntax, ACCEPTED_TRANSFER_SYNTAXES))

    ae.supported_contexts = contexts + verification_contexts

    # Limit concurrent associations
    ae.maximum_associations = config.DIMSE_MAX_ASSOCIATIONS

    return ae


def _get_scp_handlers() -> list[tuple[Any, Any]]:
    """Build pynetdicom event handler mapping lazily."""
    _, _, _, evt, _ = _load_pynetdicom()
    return [
        (evt.EVT_C_ECHO, handle_echo),
        (evt.EVT_C_STORE, handle_store),
        (evt.EVT_RELEASED, handle_release),
    ]


def start_scp(ae: Any) -> None:
    """Start the SCP in a blocking call (run in a thread)."""
    log.info(
        "Starting DICOM SCP on port %d (AE Title: %s, max assoc: %d)",
        config.DIMSE_PORT,
        config.DIMSE_AE_TITLE,
        config.DIMSE_MAX_ASSOCIATIONS,
    )
    ae.start_server(
        ("0.0.0.0", config.DIMSE_PORT),
        evt_handlers=_get_scp_handlers(),
        block=True,
    )
