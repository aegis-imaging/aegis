"""
AEGIS QC Automation Service

FastAPI service that runs automated image quality checks on DICOM studies.
Receives DICOM file paths from the Go API and returns structured QC findings.

Endpoints:
  GET  /healthz         -- liveness + backend availability
  POST /check           -- run QC checks on a study (synchronous)

Environment variables:
  QC_TOOL               -- "auto" | "basic" (default: auto)
  QC_SNR_THRESHOLD      -- minimum acceptable SNR (default: 10.0)
  QC_GAP_RATIO          -- gap ratio threshold for missing slices (default: 2.0)
"""

import logging
import time
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .backends.base import QCBackend
from .backends.basic import BasicBackend
from .config import cfg

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)


# ── Backend selection ────────────────────────────────────────────────────────


def _select_backend() -> QCBackend:
    basic = BasicBackend(
        snr_threshold=cfg.snr_threshold,
        gap_ratio_threshold=cfg.gap_ratio_threshold,
    )

    if cfg.qc_tool == "basic":
        candidates = [basic]
    else:  # "auto"
        # Future: add VertexAI backend before basic in priority order.
        candidates = [basic]

    for backend in candidates:
        if backend.available():
            log.info("QC backend selected: %s", backend.name)
            return backend

    raise RuntimeError("No QC backend is available. Install pydicom and numpy.")


_backend: QCBackend | None = None


def get_backend() -> QCBackend:
    global _backend
    if _backend is None:
        _backend = _select_backend()
    return _backend


# ── FastAPI app ──────────────────────────────────────────────────────────────

app = FastAPI(title="AEGIS QC Automation Service", version="0.1.0")


class CheckRequest(BaseModel):
    study_uid: str
    input_paths: list[str]


class CheckDetailModel(BaseModel):
    name: str
    status: str
    message: str
    details: dict = {}


class CheckResponse(BaseModel):
    study_uid: str
    status: str  # "complete" | "failed"
    overall_quality: str  # "pass" | "warn" | "fail"
    quality_issues: list[CheckDetailModel]
    tool_used: str
    duration_seconds: float
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


@app.post("/check", response_model=CheckResponse)
def check(req: CheckRequest) -> CheckResponse:
    if not req.input_paths:
        raise HTTPException(status_code=400, detail="input_paths is required")

    missing = [p for p in req.input_paths if not Path(p).exists()]
    if missing:
        raise HTTPException(
            status_code=400,
            detail=f"Input files not found: {missing[:5]}",
        )

    backend = get_backend()
    log.info(
        "QC check study %s: %d files (backend=%s)",
        req.study_uid,
        len(req.input_paths),
        backend.name,
    )

    start = time.monotonic()
    try:
        result = backend.check(req.input_paths)
        duration = time.monotonic() - start

        quality_issues = [
            CheckDetailModel(
                name=c.name, status=c.status, message=c.message, details=c.details
            )
            for c in result.checks
        ]

        log.info(
            "QC complete for %s: overall=%s, %d checks in %.1fs",
            req.study_uid,
            result.overall,
            len(quality_issues),
            duration,
        )
        return CheckResponse(
            study_uid=req.study_uid,
            status="complete",
            overall_quality=result.overall,
            quality_issues=quality_issues,
            tool_used=backend.name,
            duration_seconds=duration,
        )
    except Exception as e:
        duration = time.monotonic() - start
        log.error("QC failed for %s: %s", req.study_uid, e, exc_info=True)
        return CheckResponse(
            study_uid=req.study_uid,
            status="failed",
            overall_quality="fail",
            quality_issues=[],
            tool_used=backend.name,
            duration_seconds=duration,
            error=str(e),
        )
