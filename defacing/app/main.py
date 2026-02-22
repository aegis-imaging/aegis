"""
AEGIS Defacing Service

FastAPI service that applies face de-identification to DICOM head/brain imaging.
Receives DICOM file paths from the Go API and writes defaced DICOM files to an
output directory. Pixel data never leaves the local storage volume.

Endpoints:
  GET  /healthz         — liveness + backend availability
  POST /deface          — deface a study (synchronous)

Environment variables:
  DEFACE_TOOL           — "auto" | "mri_reface" | "deepdefacer" | "mri_deface" | "nibabel"
  MRI_DEFACE_BIN        — path to mri_deface binary
  MRI_DEFACE_BRAIN      — path to talairach_mixed_with_skull.gca atlas
  MRI_DEFACE_FACE       — path to face.gca atlas
  MRI_REFACE_BIN        — path to mri_reface binary
  DCM2NIIX_BIN          — path to dcm2niix binary (default: dcm2niix)
"""

import logging
import time
from pathlib import Path

import numpy as np
import pydicom
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import cfg
from .pipeline import run_pipeline
from .backends.base import DefacingBackend
from .backends.nibabel_fallback import NibabelFallbackBackend
from .backends.mri_deface import MriDefaceBackend
from .backends.mri_reface import MriRefaceBackend
from .backends.deepdefacer_backend import DeepDefacerBackend

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# SSIM visual QA
# ---------------------------------------------------------------------------

def _compute_ssim_score(input_paths: list[str], output_paths: list[str]) -> float | None:
    """
    Compute mean Structural Similarity Index (SSIM) between paired raw and defaced
    DICOM slices. Samples up to 10 slice pairs for speed.

    Returns a float in [0, 1] (higher = more structurally similar, defacing preserved
    brain anatomy). Returns None if scikit-image is not installed or computation fails.
    """
    try:
        from skimage.metrics import structural_similarity as ssim
    except ImportError:
        log.debug("scikit-image not available — skipping SSIM computation")
        return None

    try:
        # Build a lookup: filename → raw path
        raw_by_name = {Path(p).name: p for p in input_paths}

        # Match output files to their raw originals by filename
        pairs: list[tuple[str, str]] = []
        for out_path in output_paths:
            name = Path(out_path).name
            if name in raw_by_name:
                pairs.append((raw_by_name[name], out_path))

        if not pairs:
            return None

        # Sample up to 10 pairs
        step = max(1, len(pairs) // 10)
        sampled = pairs[::step][:10]

        scores: list[float] = []
        for raw_path, def_path in sampled:
            try:
                raw_ds = pydicom.dcmread(raw_path)
                def_ds = pydicom.dcmread(def_path)
                raw_arr = raw_ds.pixel_array.astype(np.float32)
                def_arr = def_ds.pixel_array.astype(np.float32)
                if raw_arr.shape != def_arr.shape:
                    continue
                data_range = float(raw_arr.max() - raw_arr.min()) or 1.0
                s = ssim(raw_arr, def_arr, data_range=data_range)
                scores.append(float(s))
            except Exception:
                continue

        return round(float(np.mean(scores)), 4) if scores else None
    except Exception as e:
        log.warning("SSIM computation failed: %s", e)
        return None


# ---------------------------------------------------------------------------
# Backend selection
# ---------------------------------------------------------------------------

def _select_backend() -> DefacingBackend:
    """Choose the best available backend based on DEFACE_TOOL config."""
    candidates: list[DefacingBackend]

    mri_reface = MriRefaceBackend(cfg.mri_reface_bin, cfg.dcm2niix_bin)
    deepdefacer = DeepDefacerBackend(cfg.dcm2niix_bin)
    mri_deface = MriDefaceBackend(
        cfg.mri_deface_bin, cfg.mri_deface_brain, cfg.mri_deface_face, cfg.dcm2niix_bin
    )
    nibabel = NibabelFallbackBackend()

    if cfg.deface_tool == "mri_reface":
        candidates = [mri_reface, nibabel]
    elif cfg.deface_tool == "deepdefacer":
        candidates = [deepdefacer, nibabel]
    elif cfg.deface_tool == "mri_deface":
        candidates = [mri_deface, nibabel]
    elif cfg.deface_tool == "nibabel":
        candidates = [nibabel]
    else:  # "auto"
        # Priority: mri_reface (best coverage) → deepdefacer (fast DL) → mri_deface → nibabel
        candidates = [mri_reface, deepdefacer, mri_deface, nibabel]

    for backend in candidates:
        if backend.available():
            log.info("Defacing backend selected: %s", backend.name)
            return backend

    raise RuntimeError("No defacing backend is available. Install mri_reface, deepdefacer, mri_deface, or ensure pydicom/numpy are installed.")


_backend: DefacingBackend | None = None


def get_backend() -> DefacingBackend:
    global _backend
    if _backend is None:
        _backend = _select_backend()
    return _backend


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(title="AEGIS Defacing Service", version="0.1.0")


class DefaceRequest(BaseModel):
    study_uid: str
    input_paths: list[str]   # absolute paths to raw DICOM files
    output_dir: str          # absolute path for defaced output


class DefaceResponse(BaseModel):
    study_uid: str
    status: str              # "complete" | "failed"
    output_paths: list[str]
    tool_used: str
    duration_seconds: float
    ssim_score: float | None = None   # mean SSIM across defaced slices (None if unavailable)
    error: str | None = None


@app.get("/health")
@app.get("/healthz")
def healthz() -> dict:
    try:
        backend = get_backend()
        return {
            "status": "ok",
            "backend": backend.name,
            "available": backend.available(),
        }
    except Exception as e:
        return {"status": "degraded", "error": str(e)}


@app.post("/deface", response_model=DefaceResponse)
def deface(req: DefaceRequest) -> DefaceResponse:
    if not req.input_paths:
        raise HTTPException(status_code=400, detail="input_paths is required")

    # Validate all input paths exist
    missing = [p for p in req.input_paths if not Path(p).exists()]
    if missing:
        raise HTTPException(
            status_code=400,
            detail=f"Input files not found: {missing[:5]}"
        )

    backend = get_backend()
    log.info(
        "Defacing study %s: %d files → %s (backend=%s)",
        req.study_uid, len(req.input_paths), req.output_dir, backend.name,
    )

    start = time.monotonic()
    try:
        output_paths = run_pipeline(req.input_paths, req.output_dir, backend)
        duration = time.monotonic() - start
        ssim_score = _compute_ssim_score(req.input_paths, output_paths)
        log.info(
            "Defacing complete for %s: %d output files in %.1fs (SSIM=%s)",
            req.study_uid, len(output_paths), duration,
            f"{ssim_score:.4f}" if ssim_score is not None else "n/a",
        )
        return DefaceResponse(
            study_uid=req.study_uid,
            status="complete",
            output_paths=output_paths,
            tool_used=backend.name,
            duration_seconds=duration,
            ssim_score=ssim_score,
        )
    except Exception as e:
        duration = time.monotonic() - start
        log.error("Defacing failed for %s: %s", req.study_uid, e, exc_info=True)
        return DefaceResponse(
            study_uid=req.study_uid,
            status="failed",
            output_paths=[],
            tool_used=backend.name,
            duration_seconds=duration,
            error=str(e),
        )
