"""
AEGIS Burned-in PHI Detection Service

FastAPI service that scans DICOM pixel data for burned-in text (patient names,
dates of birth, accession numbers) using OCR. Receives DICOM file paths from
the Go API and returns structured detection results.

Endpoints:
  GET  /healthz         — liveness + backend availability
  POST /detect          — scan a study for burned-in PHI (synchronous)

Environment variables:
  PHI_TOOL                  — "auto" | "tesseract" (default: auto)
  PHI_CONFIDENCE_THRESHOLD  — minimum OCR confidence 0.0–1.0 (default: 0.4)
  PHI_MIN_TEXT_LENGTH       — minimum text length to report (default: 3)
"""

import logging
import time
from dataclasses import asdict
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import cfg
from .backends.base import PHIDetectionBackend
from .backends.tesseract import TesseractBackend

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Backend selection
# ---------------------------------------------------------------------------

def _select_backend() -> PHIDetectionBackend:
    """Choose the best available backend based on PHI_TOOL config."""
    tesseract = TesseractBackend(
        confidence_threshold=cfg.confidence_threshold,
        min_text_length=cfg.min_text_length,
    )

    if cfg.phi_tool == "tesseract":
        candidates = [tesseract]
    else:  # "auto"
        # Future: add VertexAI backend before tesseract in priority order.
        candidates = [tesseract]

    for backend in candidates:
        if backend.available():
            log.info("PHI detection backend selected: %s", backend.name)
            return backend

    raise RuntimeError(
        "No PHI detection backend is available. "
        "Install tesseract-ocr and pytesseract."
    )


_backend: PHIDetectionBackend | None = None


def get_backend() -> PHIDetectionBackend:
    global _backend
    if _backend is None:
        _backend = _select_backend()
    return _backend


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(title="AEGIS Burned-in PHI Detection Service", version="0.1.0")


class DetectRequest(BaseModel):
    study_uid: str
    input_paths: list[str]


class RegionModel(BaseModel):
    text: str
    confidence: float
    bbox: list[int]


class FindingModel(BaseModel):
    file: str
    regions: list[RegionModel]


class DetectResponse(BaseModel):
    study_uid: str
    status: str  # "complete" | "failed"
    phi_detected: bool
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


@app.post("/detect", response_model=DetectResponse)
def detect(req: DetectRequest) -> DetectResponse:
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
        "Scanning study %s: %d files (backend=%s)",
        req.study_uid, len(req.input_paths), backend.name,
    )

    start = time.monotonic()
    try:
        file_findings = backend.detect(req.input_paths)
        duration = time.monotonic() - start

        # Convert dataclass findings to Pydantic models.
        findings = [
            FindingModel(
                file=f.file,
                regions=[
                    RegionModel(text=r.text, confidence=r.confidence, bbox=r.bbox)
                    for r in f.regions
                ],
            )
            for f in file_findings
        ]

        phi_detected = len(findings) > 0

        log.info(
            "Scan complete for %s: phi_detected=%s, %d files with findings in %.1fs",
            req.study_uid, phi_detected, len(findings), duration,
        )
        return DetectResponse(
            study_uid=req.study_uid,
            status="complete",
            phi_detected=phi_detected,
            findings=findings,
            tool_used=backend.name,
            duration_seconds=duration,
        )
    except Exception as e:
        duration = time.monotonic() - start
        log.error("Scan failed for %s: %s", req.study_uid, e, exc_info=True)
        return DetectResponse(
            study_uid=req.study_uid,
            status="failed",
            phi_detected=False,
            findings=[],
            tool_used=backend.name,
            duration_seconds=duration,
            error=str(e),
        )
