"""
AEGIS Neuroimaging Analytics Service

FastAPI service that runs neuroimaging analysis tools on BIDS-converted NIfTI
data. Called asynchronously from the Go API after BIDS conversion completes.

Endpoints:
  GET  /healthz                  -- liveness + backend availability
  POST /analyze                  -- run single-study analytics (synchronous)
  POST /analyze-longitudinal     -- run paired longitudinal analytics (synchronous)

Environment variables:
  ANALYTICS_TOOL        -- "auto" | "freesurfer" | "fsl" | "ants" | "spm" |
                           "atlas_roi" | "synthseg" | "nnunet" |
                           "totalsegmentator"
  ANALYTICS_TIMEOUT     -- max seconds per study (default 86400 = 24h)
  FS_LICENSE            -- FreeSurfer license file path
  FSL_DIR               -- FSL installation directory
  ANTSPATH              -- ANTs binary directory
  SPM_BIN               -- SPM standalone binary
  MCR_DIR               -- MATLAB Compiler Runtime directory
  ATLAS_DIR             -- base directory for atlas volumes + labels
  DEFAULT_ATLAS         -- default atlas name (default "aal3")
  ANTS_TEMPLATE_DIR     -- ANTs template directory for registration
"""

import logging
import os
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from . import config
from .backends.base import AnalyticsBackend, AnalyticsResult, LongitudinalBackend
from .backends.freesurfer import FreeSurferBackend
from .backends.fsl import FSLBackend
from .backends.ants import ANTsBackend
from .backends.spm import SPMBackend
from .backends.atlas_roi import AtlasROIBackend
from .backends.tbm_syn import TBMSyNBackend
from .backends.freesurfer_long import FreeSurferLongBackend
from .backends.synthseg import SynthSegBackend
from .backends.nnunet import NNUNetBackend
from .backends.totalsegmentator import TotalSegmentatorBackend

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)

ALL_BACKENDS: list[AnalyticsBackend] = [
    FreeSurferBackend(),
    FSLBackend(),
    ANTsBackend(),
    SPMBackend(),
    AtlasROIBackend(),
    SynthSegBackend(),
    NNUNetBackend(),
    TotalSegmentatorBackend(),
]

LONGITUDINAL_BACKENDS: list[LongitudinalBackend] = [
    TBMSyNBackend(),
    FreeSurferLongBackend(),
]


def _select_backends(tool: str) -> list[AnalyticsBackend]:
    """Select backends based on ANALYTICS_TOOL config."""
    if tool == "auto":
        available = [b for b in ALL_BACKENDS if b.available()]
        if available:
            return available
        log.warning("No analytics backends available")
        return []

    name_map = {b.name: b for b in ALL_BACKENDS}
    if tool not in name_map:
        raise ValueError(f"Unknown analytics tool: {tool}")
    backend = name_map[tool]
    if not backend.available():
        log.warning("Requested backend %s is not available", tool)
        return []
    return [backend]


def _select_longitudinal_backends(
    tool: str | None,
) -> list[LongitudinalBackend]:
    """Select longitudinal backends by name or return all available."""
    if tool:
        name_map = {b.name: b for b in LONGITUDINAL_BACKENDS}
        if tool in name_map and name_map[tool].available():
            return [name_map[tool]]
        return []

    return [b for b in LONGITUDINAL_BACKENDS if b.available()]


# ── FastAPI app ──────────────────────────────────────────────────────────────

app = FastAPI(title="AEGIS Neuroimaging Analytics Service", version="0.2.0")


class AnalyzeRequest(BaseModel):
    study_uid: str
    bids_dir: str
    output_dir: str
    tools: list[str] | None = None  # Override ANALYTICS_TOOL; null = use config
    atlas: str | None = None  # Override DEFAULT_ATLAS (e.g. "aal3", "mcalt")


class AnalyzeLongitudinalRequest(BaseModel):
    baseline_study_uid: str
    followup_study_uid: str
    baseline_bids_dir: str
    followup_bids_dir: str
    output_dir: str
    scan_interval_days: float
    tools: list[str] | None = None
    atlas: str | None = None


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

    longitudinal_info = {}
    for b in LONGITUDINAL_BACKENDS:
        try:
            longitudinal_info[b.name] = b.available()
        except Exception:
            longitudinal_info[b.name] = False

    any_available = any(backends_info.values()) or any(longitudinal_info.values())
    return {
        "status": "ok" if any_available else "degraded",
        "backends": backends_info,
        "longitudinal_backends": longitudinal_info,
    }


def _run_backends(
    backends: list[AnalyticsBackend],
    req: AnalyzeRequest,
    output_dir: str,
) -> list[ToolResultModel]:
    """Run single-study backends and collect results."""
    results: list[ToolResultModel] = []
    kwargs = {}
    if req.atlas:
        kwargs["atlas"] = req.atlas

    for backend in backends:
        log.info("Starting %s for %s", backend.name, req.study_uid)
        try:
            result = backend.analyze(req.bids_dir, output_dir, req.study_uid, **kwargs)
            results.append(
                ToolResultModel(
                    tool=result.tool,
                    success=result.success,
                    duration_seconds=result.duration_seconds,
                    outputs=result.outputs,
                    metrics=result.metrics,
                    error=result.error,
                )
            )
            log.info(
                "%s %s for %s (%.1fs)",
                backend.name,
                "succeeded" if result.success else "failed",
                req.study_uid,
                result.duration_seconds,
            )
        except Exception as e:
            log.error(
                "%s crashed for %s: %s", backend.name, req.study_uid, e, exc_info=True
            )
            results.append(
                ToolResultModel(
                    tool=backend.name,
                    success=False,
                    error=str(e),
                )
            )
    return results


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
        config.DATA_DIR, "analytics", req.study_uid
    )
    os.makedirs(output_dir, exist_ok=True)

    # Select backends
    if req.tools:
        backends = []
        name_map = {b.name: b for b in ALL_BACKENDS}
        for tool_name in req.tools:
            if tool_name in name_map and name_map[tool_name].available():
                backends.append(name_map[tool_name])
            else:
                log.warning("Requested tool %s not available, skipping", tool_name)
    else:
        backends = _select_backends(config.ANALYTICS_TOOL)

    if not backends:
        return AnalyzeResponse(
            study_uid=req.study_uid,
            status="failed",
            results=[],
            duration_seconds=0.0,
            error="No analytics backends available",
        )

    log.info(
        "Running analytics for %s: %s",
        req.study_uid,
        [b.name for b in backends],
    )

    start = time.time()
    results = _run_backends(backends, req, output_dir)
    duration = time.time() - start
    successes = sum(1 for r in results if r.success)

    if successes == len(results):
        status = "complete"
    elif successes > 0:
        status = "partial"
    else:
        status = "failed"

    log.info(
        "Analytics %s for %s: %d/%d succeeded (%.1fs)",
        status,
        req.study_uid,
        successes,
        len(results),
        duration,
    )

    return AnalyzeResponse(
        study_uid=req.study_uid,
        status=status,
        results=results,
        duration_seconds=duration,
    )


@app.post("/analyze-longitudinal", response_model=AnalyzeResponse)
def analyze_longitudinal(req: AnalyzeLongitudinalRequest) -> AnalyzeResponse:
    """Run longitudinal (paired) analytics on baseline + follow-up BIDS data."""
    if not req.baseline_bids_dir or not req.followup_bids_dir:
        raise HTTPException(
            status_code=400,
            detail="baseline_bids_dir and followup_bids_dir are required",
        )

    for label, path in [
        ("baseline", req.baseline_bids_dir),
        ("follow-up", req.followup_bids_dir),
    ]:
        if not os.path.isdir(path):
            raise HTTPException(
                status_code=400,
                detail=f"{label} BIDS directory not found: {path}",
            )

    if req.scan_interval_days <= 0:
        raise HTTPException(
            status_code=400,
            detail="scan_interval_days must be positive",
        )

    output_dir = req.output_dir or os.path.join(
        config.DATA_DIR, "analytics", req.followup_study_uid
    )
    os.makedirs(output_dir, exist_ok=True)

    # Select longitudinal backends
    if req.tools:
        backends = []
        name_map = {b.name: b for b in LONGITUDINAL_BACKENDS}
        for tool_name in req.tools:
            if tool_name in name_map and name_map[tool_name].available():
                backends.append(name_map[tool_name])
            else:
                log.warning("Requested longitudinal tool %s not available", tool_name)
    else:
        backends = _select_longitudinal_backends(None)

    if not backends:
        return AnalyzeResponse(
            study_uid=req.followup_study_uid,
            status="failed",
            results=[],
            duration_seconds=0.0,
            error="No longitudinal backends available",
        )

    log.info(
        "Running longitudinal analytics: baseline=%s, followup=%s, backends=%s",
        req.baseline_study_uid,
        req.followup_study_uid,
        [b.name for b in backends],
    )

    start = time.time()
    results: list[ToolResultModel] = []
    kwargs = {}
    if req.atlas:
        kwargs["atlas"] = req.atlas

    for backend in backends:
        log.info("Starting %s for %s", backend.name, req.followup_study_uid)
        try:
            result = backend.analyze_longitudinal(
                baseline_bids_dir=req.baseline_bids_dir,
                followup_bids_dir=req.followup_bids_dir,
                output_dir=output_dir,
                baseline_study_uid=req.baseline_study_uid,
                followup_study_uid=req.followup_study_uid,
                scan_interval_days=req.scan_interval_days,
                **kwargs,
            )
            results.append(
                ToolResultModel(
                    tool=result.tool,
                    success=result.success,
                    duration_seconds=result.duration_seconds,
                    outputs=result.outputs,
                    metrics=result.metrics,
                    error=result.error,
                )
            )
        except Exception as e:
            log.error(
                "%s crashed for %s: %s",
                backend.name,
                req.followup_study_uid,
                e,
                exc_info=True,
            )
            results.append(
                ToolResultModel(tool=backend.name, success=False, error=str(e))
            )

    duration = time.time() - start
    successes = sum(1 for r in results if r.success)

    if successes == len(results):
        status = "complete"
    elif successes > 0:
        status = "partial"
    else:
        status = "failed"

    log.info(
        "Longitudinal analytics %s for %s: %d/%d succeeded (%.1fs)",
        status,
        req.followup_study_uid,
        successes,
        len(results),
        duration,
    )

    return AnalyzeResponse(
        study_uid=req.followup_study_uid,
        status=status,
        results=results,
        duration_seconds=duration,
    )
