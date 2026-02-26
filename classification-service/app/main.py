"""
AEGIS Metadata Classification Service

Classifies DICOM study modality and body part by reading DICOM headers.
Uses heuristic analysis (DICOM tags, SOP Class UID, description patterns)
for local dev; Google Cloud Vision and AWS Rekognition backends available
for production (augment heuristics with image-based label inference).

Endpoints:
    GET  /healthz   — liveness/readiness probe
    POST /classify  — classify a study's modality and body part
"""

import logging
import time

from fastapi import FastAPI
from pydantic import BaseModel

from .config import cfg
from .backends.base import ClassificationBackend
from .backends.heuristic import HeuristicBackend
from .backends.gemini import GeminiClassificationBackend
from .backends.google_vision import GoogleVisionClassificationBackend
from .backends.aws_rekognition import AWSRekognitionClassificationBackend

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
log = logging.getLogger(__name__)

app = FastAPI(title="AEGIS Metadata Classification Service")

# ── Backend selection ────────────────────────────────────────────────────────

_BACKENDS: list[ClassificationBackend] = [
    # Priority order: Gemini highest, then cloud vision/rekognition, then offline fallback.
    GeminiClassificationBackend(confidence_threshold=cfg.confidence_threshold),
    GoogleVisionClassificationBackend(confidence_threshold=cfg.confidence_threshold),
    AWSRekognitionClassificationBackend(confidence_threshold=cfg.confidence_threshold),
    HeuristicBackend(),
]


def _select_backend() -> ClassificationBackend | None:
    """Select the best available backend (or a specific one via CLASSIFY_TOOL)."""
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
            "error": "No classification backend available.",
        }
    return {
        "status": "ok",
        "backend": _backend.name,
        "available": True,
    }


# ── Classification endpoint ─────────────────────────────────────────────────

class ClassifyRequest(BaseModel):
    study_uid: str
    input_paths: list[str]


class ClassifyResponse(BaseModel):
    status: str  # "complete" | "failed"
    modality: str = ""
    body_part: str = ""
    confidence: float = 0.0
    method: str = ""
    tool_used: str = ""
    duration_seconds: float = 0.0
    error: str | None = None


@app.post("/classify", response_model=ClassifyResponse)
def classify(req: ClassifyRequest) -> ClassifyResponse:
    if _backend is None:
        return ClassifyResponse(
            status="failed",
            error="No classification backend available",
        )

    log.info("Starting classification for study %s (backend=%s, %d files)",
             req.study_uid, _backend.name, len(req.input_paths))
    t0 = time.time()

    try:
        result = _backend.classify(req.input_paths)
    except Exception as e:
        elapsed = time.time() - t0
        log.error("Classification failed for %s: %s", req.study_uid, e)
        return ClassifyResponse(
            status="failed",
            tool_used=_backend.name,
            duration_seconds=round(elapsed, 2),
            error=str(e),
        )

    elapsed = time.time() - t0

    log.info(
        "Classification complete for %s: modality=%s body_part=%s confidence=%.2f method=%s in %.1fs",
        req.study_uid, result.modality, result.body_part, result.confidence, result.method, elapsed,
    )

    return ClassifyResponse(
        status="complete",
        modality=result.modality,
        body_part=result.body_part,
        confidence=result.confidence,
        method=result.method,
        tool_used=_backend.name,
        duration_seconds=round(elapsed, 2),
    )
