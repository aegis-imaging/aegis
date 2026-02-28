"""
AEGIS SCT (Spinal Cord Toolbox) Service

FastAPI service that runs spinal cord analysis tools on BIDS-converted NIfTI
data. Called asynchronously from the Go API after BIDS conversion completes.

Endpoints:
  GET  /healthz   -- liveness + backend availability
  POST /analyze   -- run SCT pipeline (synchronous)

Environment variables:
  SCT_TOOL      -- "auto" | "sct" (default "auto")
  SCT_CONTRAST  -- default contrast: t1, t2, t2s, dwi (default "t2")
  SCT_DATA_DIR  -- shared data directory (default "/app/data")
  SCT_TIMEOUT   -- max seconds per SCT CLI step (default 3600)
"""

import logging
import os
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import cfg
from .backends.base import SctBackend, SctResult
from .backends.sct import SpinalCordToolboxBackend

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)

ALL_BACKENDS: list[SctBackend] = [
    SpinalCordToolboxBackend(),
]


def _select_backend(tool: str) -> SctBackend | None:
    """Select SCT backend based on config."""
    if tool == "auto":
        for b in ALL_BACKENDS:
            if b.available():
                return b
        log.warning("No SCT backends available")
        return None

    name_map = {b.name: b for b in ALL_BACKENDS}
    if tool not in name_map:
        log.warning("Unknown SCT tool: %s", tool)
        return None
    backend = name_map[tool]
    if not backend.available():
        log.warning("Requested backend %s is not available", tool)
        return None
    return backend


# ── FastAPI app ──────────────────────────────────────────────────────────────

app = FastAPI(title="AEGIS SCT Service", version="0.1.0")


class AnalyzeRequest(BaseModel):
    study_uid: str
    bids_dir: str
    output_dir: str
    contrast: str | None = None  # Override SCT_CONTRAST


class ToolResultModel(BaseModel):
    tool: str
    success: bool
    duration_seconds: float = 0.0
    outputs: list[str] = []
    metrics: dict = {}
    error: str = ""


class AnalyzeResponse(BaseModel):
    study_uid: str
    status: str  # "complete" | "partial" | "failed"
    results: list[ToolResultModel]
    duration_seconds: float
    error: str | None = None


@app.get("/health")
@app.get("/healthz")
def healthz() -> dict:
    backends_info = {}
    for b in ALL_BACKENDS:
        try:
            backends_info[b.name] = b.available()
        except Exception:
            backends_info[b.name] = False

    any_available = any(backends_info.values())
    return {
        "status": "ok" if any_available else "degraded",
        "backends": backends_info,
    }


@app.post("/analyze", response_model=AnalyzeResponse)
def analyze(req: AnalyzeRequest) -> AnalyzeResponse:
    if not req.bids_dir:
        raise HTTPException(status_code=400, detail="bids_dir is required")

    if not os.path.isdir(req.bids_dir):
        raise HTTPException(
            status_code=400,
            detail=f"BIDS directory not found: {req.bids_dir}",
        )

    # Determine output directory
    output_dir = req.output_dir or os.path.join(
        cfg.data_dir, "sct", req.study_uid
    )
    os.makedirs(output_dir, exist_ok=True)

    # Select backend
    backend = _select_backend(cfg.tool)
    if not backend:
        return AnalyzeResponse(
            study_uid=req.study_uid,
            status="failed",
            results=[],
            duration_seconds=0.0,
            error="No SCT backends available",
        )

    log.info("Running SCT for %s using %s", req.study_uid, backend.name)

    start = time.time()
    kwargs = {}
    if req.contrast:
        kwargs["contrast"] = req.contrast

    try:
        result = backend.analyze(req.bids_dir, output_dir, req.study_uid, **kwargs)
        tool_result = ToolResultModel(
            tool=result.tool,
            success=result.success,
            duration_seconds=result.duration_seconds,
            outputs=result.outputs,
            metrics=result.metrics,
            error=result.error,
        )
    except Exception as e:
        log.error("SCT crashed for %s: %s", req.study_uid, e, exc_info=True)
        tool_result = ToolResultModel(
            tool=backend.name,
            success=False,
            error=str(e),
        )

    duration = time.time() - start

    if tool_result.success:
        # Check if all steps completed vs partial
        steps = tool_result.metrics.get("steps_completed", [])
        status = "complete" if len(steps) >= 3 else "partial"
    else:
        status = "failed"

    log.info(
        "SCT %s for %s (%.1fs)", status, req.study_uid, duration,
    )

    return AnalyzeResponse(
        study_uid=req.study_uid,
        status=status,
        results=[tool_result],
        duration_seconds=duration,
    )
