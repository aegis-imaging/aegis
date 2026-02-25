"""DIMSE C-STORE SCU sender utilities.

This module is used when Go routing forwards a study to a DIMSE destination.
It reads stored DICOM files from shared storage and sends them via C-STORE.
Also provides send_echo() for C-ECHO connectivity checks.
"""

from __future__ import annotations

import time
from pathlib import Path
import logging

import pydicom

from app import config

log = logging.getLogger(__name__)


def _load_pynetdicom_ae():
    """Import pynetdicom lazily so module import works without optional deps."""
    from pynetdicom import AE

    return AE


def _status_ok(status) -> bool:
    """Return True if C-STORE response status is success or warning."""
    if status is None:
        return False
    code = getattr(status, "Status", None)
    if code is None:
        return False
    # Success (0x0000) or warning class (0xBxxx).
    return code == 0x0000 or (code & 0xF000) == 0xB000


def _sorted_dicom_files(study_uid: str, dicom_store: str) -> list[Path]:
    """Return study DICOM files sorted by numeric filename when possible."""
    study_dir = Path(config.DIMSE_DATA_DIR) / "dicom" / dicom_store / study_uid
    files = list(study_dir.glob("*.dcm"))

    def sort_key(path: Path):
        stem = path.stem
        if stem.isdigit():
            return (0, int(stem))
        return (1, stem)

    files.sort(key=sort_key)
    return files


def forward_study(
    study_uid: str,
    dicom_store: str,
    host: str,
    port: int,
    ae_title: str,
) -> dict[str, int]:
    """Send all DICOM files from a study to a remote DIMSE destination.

    Raises:
        FileNotFoundError: no DICOM files found for the study.
        RuntimeError: association or one/more C-STORE requests failed.
    """
    files = _sorted_dicom_files(study_uid, dicom_store)
    if not files:
        raise FileNotFoundError(
            f"No DICOM files for study {study_uid} in store {dicom_store}"
        )

    AE = _load_pynetdicom_ae()
    ae = AE(ae_title=config.DIMSE_AE_TITLE)

    datasets = []
    added_contexts: set[tuple[str, str]] = set()
    for path in files:
        ds = pydicom.dcmread(path, force=True)
        datasets.append(ds)
        sop_uid = str(getattr(ds, "SOPClassUID", ""))
        ts_uid = str(getattr(getattr(ds, "file_meta", None), "TransferSyntaxUID", ""))
        if not sop_uid:
            continue
        ctx_key = (sop_uid, ts_uid)
        if ctx_key in added_contexts:
            continue
        if ts_uid:
            ae.add_requested_context(sop_uid, ts_uid)
        else:
            ae.add_requested_context(sop_uid)
        added_contexts.add(ctx_key)

    assoc = ae.associate(host, int(port), ae_title=ae_title)
    if not assoc.is_established:
        raise RuntimeError(
            f"DIMSE association failed to {host}:{port} (AE={ae_title})"
        )

    sent = 0
    failed = 0
    try:
        for ds in datasets:
            status = assoc.send_c_store(ds)
            if _status_ok(status):
                sent += 1
            else:
                failed += 1
    finally:
        assoc.release()

    if failed > 0:
        raise RuntimeError(
            f"C-STORE completed with failures (sent={sent}, failed={failed})"
        )

    log.info(
        "DIMSE forward complete for %s: sent=%d failed=%d host=%s port=%d ae=%s",
        study_uid,
        sent,
        failed,
        host,
        port,
        ae_title,
    )
    return {"files_total": len(datasets), "files_sent": sent, "files_failed": failed}


def send_cfind(
    host: str,
    port: int,
    ae_title: str,
    query_level: str,
    query_params: dict,
) -> list[dict]:
    """Send C-FIND to a remote DIMSE AE and return matching datasets.

    Args:
        host: Remote AE host.
        port: Remote AE port.
        ae_title: Remote AE title.
        query_level: DICOM query/retrieve level — ``PATIENT``, ``STUDY``,
            ``SERIES``, or ``IMAGE``.
        query_params: Dict mapping DICOM keyword → value (e.g.
            ``{"PatientID": "12345", "StudyDate": "20240101-20240201"}``).
            Empty-string values are valid (wildcard match).

    Returns:
        List of matching datasets serialised as ``{keyword: value}`` dicts.

    Raises:
        RuntimeError: if association fails or the C-FIND response is an error.
    """
    from pynetdicom.sop_class import (  # type: ignore[import]
        StudyRootQueryRetrieveInformationModelFind,
        PatientRootQueryRetrieveInformationModelFind,
    )

    AE = _load_pynetdicom_ae()
    ae = AE(ae_title=config.DIMSE_AE_TITLE)

    level_upper = query_level.upper()
    if level_upper == "PATIENT":
        sop_class = PatientRootQueryRetrieveInformationModelFind
    else:
        sop_class = StudyRootQueryRetrieveInformationModelFind

    ae.add_requested_context(sop_class)

    assoc = ae.associate(host, int(port), ae_title=ae_title)
    if not assoc.is_established:
        raise RuntimeError(
            f"DIMSE association failed to {host}:{port} (AE={ae_title})"
        )

    import pydicom
    from pydicom.uid import generate_uid

    ds = pydicom.dataset.Dataset()
    ds.QueryRetrieveLevel = level_upper
    # Apply caller-supplied key/value pairs.
    for keyword, value in query_params.items():
        try:
            setattr(ds, keyword, value)
        except AttributeError:
            log.warning("C-FIND: unknown DICOM keyword %r — skipping", keyword)

    results: list[dict] = []
    try:
        responses = assoc.send_c_find(ds, sop_class)
        for status, identifier in responses:
            if status is None:
                raise RuntimeError("C-FIND connection reset (no status received)")
            code = getattr(status, "Status", None)
            # 0xFF00 = Pending; 0xFF01 = Pending with warnings; 0x0000 = Success
            if code in (0xFF00, 0xFF01) and identifier is not None:
                row: dict = {}
                for elem in identifier:
                    if elem.keyword:
                        try:
                            row[elem.keyword] = str(elem.value)
                        except Exception:
                            row[elem.keyword] = ""
                results.append(row)
            elif code == 0x0000:
                break  # success, no more pending results
            elif code is not None and code != 0x0000:
                raise RuntimeError(f"C-FIND returned error status: {code:#06x}")
    finally:
        assoc.release()

    log.info(
        "C-FIND complete: host=%s port=%d ae=%s level=%s results=%d",
        host,
        port,
        ae_title,
        level_upper,
        len(results),
    )
    return results


def send_echo(host: str, port: int, ae_title: str) -> dict:
    """Send C-ECHO to a remote DIMSE destination and return latency.

    Returns:
        dict with keys: success (bool), latency_ms (float)

    Raises:
        RuntimeError: if association fails or C-ECHO response is not success.
    """
    from pynetdicom.sop_class import Verification  # type: ignore[import]

    AE = _load_pynetdicom_ae()
    ae = AE(ae_title=config.DIMSE_AE_TITLE)
    ae.add_requested_context(Verification)

    t0 = time.monotonic()
    assoc = ae.associate(host, int(port), ae_title=ae_title)
    if not assoc.is_established:
        raise RuntimeError(
            f"DIMSE association failed to {host}:{port} (AE={ae_title})"
        )

    try:
        status = assoc.send_c_echo()
        latency_ms = round((time.monotonic() - t0) * 1000, 1)
        code = getattr(status, "Status", None) if status is not None else None
        if code != 0x0000:
            raise RuntimeError(f"C-ECHO returned non-success status: {code!r}")
    finally:
        assoc.release()

    log.info("C-ECHO success: host=%s port=%d ae=%s latency_ms=%.1f", host, port, ae_title, latency_ms)
    return {"success": True, "latency_ms": latency_ms}

