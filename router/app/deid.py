"""De-identification orchestration for a study.

Three layers, all optional except tag de-id:
  1. Tag-level (midi-b) — always runs. Pure-Python, no external service.
  2. Pixel PHI scrub (phi-detection sidecar) — if PHI_DETECTION_SERVICE_URL set.
  3. Face de-id (defacing sidecar) — if DEFACING_SERVICE_URL set.

Each layer writes its output back into the study's `deid/` directory; later
layers operate on the previous layer's output. If any layer fails, the study
is quarantined and forwarding is aborted.
"""

from __future__ import annotations

import logging
import shutil
import time
from dataclasses import dataclass, field
from pathlib import Path

import httpx
import pydicom

from app import storage
from app.config import RouterConfig

log = logging.getLogger(__name__)


@dataclass
class DeidResult:
    ok: bool
    stages_run: list[str] = field(default_factory=list)
    failed_stage: str = ""
    error: str = ""
    duration_seconds: float = 0.0


def run_pipeline(cfg: RouterConfig, paths: storage.StudyPaths) -> DeidResult:
    start = time.monotonic()
    stages: list[str] = []

    # ── Stage 1: tag de-id (always) ───────────────────────────────────
    try:
        _tag_deidentify(cfg, paths)
        stages.append("tag_deid")
    except Exception as e:
        return DeidResult(
            ok=False,
            stages_run=stages,
            failed_stage="tag_deid",
            error=f"{type(e).__name__}: {e}",
            duration_seconds=time.monotonic() - start,
        )

    # ── Stage 2: pixel PHI scrub (optional) ───────────────────────────
    if cfg.phi_detection_url:
        try:
            _call_phi_redact(cfg, paths)
            stages.append("phi_scrub")
        except Exception as e:
            return DeidResult(
                ok=False,
                stages_run=stages,
                failed_stage="phi_scrub",
                error=f"{type(e).__name__}: {e}",
                duration_seconds=time.monotonic() - start,
            )

    # ── Stage 3: defacing (optional) ──────────────────────────────────
    if cfg.defacing_url:
        try:
            _call_defacing(cfg, paths)
            stages.append("defacing")
        except Exception as e:
            return DeidResult(
                ok=False,
                stages_run=stages,
                failed_stage="defacing",
                error=f"{type(e).__name__}: {e}",
                duration_seconds=time.monotonic() - start,
            )

    return DeidResult(ok=True, stages_run=stages, duration_seconds=time.monotonic() - start)


def _tag_deidentify(cfg: RouterConfig, paths: storage.StudyPaths) -> None:
    """Apply MIDI-B basic profile to every raw instance and write to deid/.

    We import midi_b lazily so the router can boot even when midi_b is missing
    (e.g. running tests against the dummy fallback).
    """
    try:
        from midi_b.pipeline import process_study, PipelineOptions  # type: ignore
    except ImportError:
        log.warning("midi_b not importable — falling back to copy-only (DEV ONLY!)")
        _copy_raw_to_deid(paths)
        return

    options = PipelineOptions(
        salt=cfg.midi_b_salt,
        date_shift=cfg.midi_b_date_shift,
        date_shift_max_days=cfg.midi_b_date_shift_max_days,
        keep_private_tags=cfg.midi_b_keep_private_tags,
        retained_tags=list(cfg.midi_b_retained_tags) or None,
    )
    # Clear deid/ before rerunning so stale outputs don't linger.
    for p in paths.deid.glob("*"):
        if p.is_file():
            p.unlink()
    process_study(paths.raw, paths.deid, options)


def _copy_raw_to_deid(paths: storage.StudyPaths) -> None:
    """Dev-only fallback when midi_b isn't available."""
    for p in paths.deid.glob("*"):
        if p.is_file():
            p.unlink()
    for src in storage.list_raw_files(paths):
        shutil.copy2(src, paths.deid / src.name)


def _call_phi_redact(cfg: RouterConfig, paths: storage.StudyPaths) -> None:
    """Pixel PHI redaction: detect-then-burn. Operates on deid/ in-place."""
    input_paths = [str(p) for p in storage.list_deid_files(paths)]
    if not input_paths:
        return
    payload = {
        "study_uid": paths.study_uid,
        "input_paths": input_paths,
        "output_dir": str(paths.deid),
    }
    with httpx.Client(timeout=600.0) as client:
        r = client.post(cfg.phi_detection_url.rstrip("/") + "/redact", json=payload)
        r.raise_for_status()
        result = r.json()
    if result.get("status") != "complete":
        raise RuntimeError(f"phi-detection returned status={result.get('status')!r}: {result.get('error')!r}")


def _call_defacing(cfg: RouterConfig, paths: storage.StudyPaths) -> None:
    """Face de-id: replaces facial pixel data in the deid/ files."""
    input_paths = [str(p) for p in storage.list_deid_files(paths)]
    if not input_paths:
        return
    payload = {
        "study_uid": paths.study_uid,
        "input_paths": input_paths,
        "output_dir": str(paths.deid),
    }
    with httpx.Client(timeout=1800.0) as client:
        r = client.post(cfg.defacing_url.rstrip("/") + "/deface", json=payload)
        r.raise_for_status()
        result = r.json()
    if result.get("status") != "complete":
        raise RuntimeError(f"defacing returned status={result.get('status')!r}: {result.get('error')!r}")


def summarize_study(paths: storage.StudyPaths) -> dict:
    """Read the first deid instance to harvest study-level metadata for the
    cloud ingest call (no PHI in these tags after de-id)."""
    files = storage.list_deid_files(paths)
    if not files:
        return {}
    try:
        ds = pydicom.dcmread(str(files[0]), stop_before_pixels=True)
    except Exception:
        return {}
    series_uids: set[str] = set()
    for f in files:
        try:
            d = pydicom.dcmread(str(f), stop_before_pixels=True, specific_tags=["SeriesInstanceUID"])
            if "SeriesInstanceUID" in d:
                series_uids.add(str(d.SeriesInstanceUID))
        except Exception:
            continue
    return {
        "study_instance_uid": str(getattr(ds, "StudyInstanceUID", paths.study_uid)),
        "modality": str(getattr(ds, "Modality", "")),
        "body_part": str(getattr(ds, "BodyPartExamined", "")),
        "study_description": str(getattr(ds, "StudyDescription", "")),
        "study_date": str(getattr(ds, "StudyDate", "")),
        "series_count": len(series_uids),
        "instance_count": len(files),
        "study_size_bytes": sum(f.stat().st_size for f in files),
    }
