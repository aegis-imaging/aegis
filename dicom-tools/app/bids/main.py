"""
AEGIS BIDS Conversion Service

Converts DICOM files to NIfTI format with BIDS-compliant directory structure
and sidecar JSON metadata. Uses dcm2niix for the conversion.

Endpoints:
    GET  /healthz   — liveness/readiness probe
    POST /convert   — convert a study from DICOM to BIDS/NIfTI
"""

import logging
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import cfg
from .backends.base import BidsBackend
from .backends.dcm2niix import Dcm2niixBackend

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
log = logging.getLogger(__name__)

app = FastAPI(title="AEGIS BIDS Conversion Service")

# ── Backend selection ────────────────────────────────────────────────────────

_BACKENDS: list[BidsBackend] = [
    Dcm2niixBackend(),
]


def _select_backend() -> BidsBackend | None:
    """Select the best available backend (or a specific one via BIDS_TOOL)."""
    if cfg.tool != "auto":
        for b in _BACKENDS:
            if b.name == cfg.tool:
                return b if b.available() else None
        return None

    for b in _BACKENDS:
        if b.available():
            return b
    return None


_backend = _select_backend()


# ── Health check ─────────────────────────────────────────────────────────────

@app.get("/health")
@app.get("/healthz")
def healthz() -> dict:
    if _backend is None:
        return {
            "status": "degraded",
            "backend": None,
            "available": False,
            "error": "No BIDS conversion backend available. Install dcm2niix.",
        }
    return {
        "status": "ok",
        "backend": _backend.name,
        "available": True,
    }


# ── Conversion endpoint ─────────────────────────────────────────────────────

class ConvertRequest(BaseModel):
    study_uid: str
    input_dir: str
    output_dir: str


class ConvertResponse(BaseModel):
    study_uid: str
    status: str  # "complete" | "failed"
    output_files: list[str] = []
    warnings: list[str] = []
    tool_used: str = ""
    duration_seconds: float = 0.0
    error: str | None = None


@app.post("/convert", response_model=ConvertResponse)
def convert(req: ConvertRequest) -> ConvertResponse:
    if _backend is None:
        return ConvertResponse(
            study_uid=req.study_uid,
            status="failed",
            error="No BIDS conversion backend available",
        )

    log.info("Starting BIDS conversion for study %s (backend=%s)", req.study_uid, _backend.name)
    t0 = time.time()

    try:
        result = _backend.convert(req.input_dir, req.output_dir, req.study_uid)
    except Exception as e:
        elapsed = time.time() - t0
        log.error("BIDS conversion failed for %s: %s", req.study_uid, e)
        return ConvertResponse(
            study_uid=req.study_uid,
            status="failed",
            tool_used=_backend.name,
            duration_seconds=round(elapsed, 2),
            error=str(e),
        )

    elapsed = time.time() - t0

    # If no NIfTI files were produced, report failure
    nifti_files = [f for f in result.output_files if f.endswith((".nii.gz", ".nii"))]
    if not nifti_files:
        log.warning("BIDS conversion produced no NIfTI files for %s", req.study_uid)
        return ConvertResponse(
            study_uid=req.study_uid,
            status="failed",
            output_files=result.output_files,
            warnings=result.warnings,
            tool_used=_backend.name,
            duration_seconds=round(elapsed, 2),
            error="No NIfTI files produced — DICOM data may be unsupported",
        )

    log.info(
        "BIDS conversion complete for %s: %d files, %d warnings in %.1fs",
        req.study_uid, len(result.output_files), len(result.warnings), elapsed,
    )

    return ConvertResponse(
        study_uid=req.study_uid,
        status="complete",
        output_files=result.output_files,
        warnings=result.warnings,
        tool_used=_backend.name,
        duration_seconds=round(elapsed, 2),
    )
