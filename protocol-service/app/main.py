"""
AEGIS MRI Protocol Compliance Service

FastAPI service that checks DICOM acquisition parameters against protocol
templates. Supports both Classic and Enhanced DICOM formats. Extracts device
identity (manufacturer, model, software version) and compares parameters
against per-rule tolerance and severity levels.

Endpoints:
  GET  /healthz         -- liveness + backend availability
  POST /check           -- run protocol compliance check (synchronous)

Environment variables:
  PROTOCOL_TOOL              -- "auto" | "basic" (default: auto)
  PROTOCOL_DEFAULT_TOLERANCE -- default % tolerance for numeric comparisons (default: 5.0)
"""

import logging
import time
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .backends.base import ProtocolBackend
from .backends.basic import BasicBackend
from .config import cfg
from .extractor import extract_from_series

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)


# ── Backend selection ────────────────────────────────────────────────────────


def _select_backend() -> ProtocolBackend:
    basic = BasicBackend(default_tolerance=cfg.default_tolerance)

    if cfg.protocol_tool == "basic":
        candidates = [basic]
    else:  # "auto"
        candidates = [basic]

    for backend in candidates:
        if backend.available():
            log.info("Protocol backend selected: %s", backend.name)
            return backend

    raise RuntimeError("No protocol backend is available. Install pydicom.")


_backend: ProtocolBackend | None = None


def get_backend() -> ProtocolBackend:
    global _backend
    if _backend is None:
        _backend = _select_backend()
    return _backend


# ── FastAPI app ──────────────────────────────────────────────────────────────

app = FastAPI(title="AEGIS MRI Protocol Compliance Service", version="0.1.0")


class ParameterRule(BaseModel):
    tag_keyword: str
    target: float | str | list | None = None
    tolerance_pct: float = 5.0
    tolerance_abs: float | None = None
    match_type: str = "numeric"  # "numeric", "exact", "contains_all", "range"
    severity: str = "warning"  # "critical", "warning", "info"
    note: str = ""


class CheckRequest(BaseModel):
    study_uid: str
    input_paths: list[str]
    rules: list[ParameterRule]


class DeviceInfoModel(BaseModel):
    manufacturer: str = ""
    model: str = ""
    software_version: str = ""


class FindingModel(BaseModel):
    tag_keyword: str
    severity: str
    status: str  # "compliant", "deviated", "missing"
    message: str
    expected: str
    actual: str
    deviation_pct: float | None = None


class CheckResponse(BaseModel):
    study_uid: str
    status: str  # "complete" | "failed"
    overall_compliance: str  # "compliant", "minor_deviations", "non_compliant"
    device_info: DeviceInfoModel
    dicom_format: str  # "classic" | "enhanced"
    findings: list[FindingModel]
    tool_used: str
    duration_seconds: float
    error: str | None = None


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
    if not req.rules:
        raise HTTPException(status_code=400, detail="rules is required")

    missing = [p for p in req.input_paths if not Path(p).exists()]
    if missing:
        raise HTTPException(
            status_code=400,
            detail=f"Input files not found: {missing[:5]}",
        )

    backend = get_backend()
    log.info(
        "Protocol check study %s: %d files, %d rules (backend=%s)",
        req.study_uid,
        len(req.input_paths),
        len(req.rules),
        backend.name,
    )

    start = time.monotonic()
    try:
        # Extract parameters from DICOM files
        extraction = extract_from_series(req.input_paths)

        # Compare against rules
        rules_dicts = [r.model_dump() for r in req.rules]
        result = backend.check(extraction.params, rules_dicts)

        duration = time.monotonic() - start

        findings = [
            FindingModel(
                tag_keyword=f.tag_keyword,
                severity=f.severity,
                status=f.status,
                message=f.message,
                expected=f.expected,
                actual=f.actual,
                deviation_pct=f.deviation_pct,
            )
            for f in result.findings
        ]

        log.info(
            "Protocol check complete for %s: overall=%s, %d findings in %.1fs "
            "(format=%s, device=%s %s %s)",
            req.study_uid,
            result.overall,
            len(findings),
            duration,
            extraction.dicom_format,
            extraction.device.manufacturer,
            extraction.device.model,
            extraction.device.software_version,
        )

        return CheckResponse(
            study_uid=req.study_uid,
            status="complete",
            overall_compliance=result.overall,
            device_info=DeviceInfoModel(
                manufacturer=extraction.device.manufacturer,
                model=extraction.device.model,
                software_version=extraction.device.software_version,
            ),
            dicom_format=extraction.dicom_format,
            findings=findings,
            tool_used=backend.name,
            duration_seconds=duration,
        )
    except Exception as e:
        duration = time.monotonic() - start
        log.error(
            "Protocol check failed for %s: %s", req.study_uid, e, exc_info=True
        )
        return CheckResponse(
            study_uid=req.study_uid,
            status="failed",
            overall_compliance="non_compliant",
            device_info=DeviceInfoModel(),
            dicom_format="unknown",
            findings=[],
            tool_used=backend.name,
            duration_seconds=duration,
            error=str(e),
        )
