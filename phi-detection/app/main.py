"""
AEGIS Burned-in PHI Detection Service

FastAPI service that scans DICOM pixel data for burned-in text (patient names,
dates of birth, accession numbers) using OCR. Receives DICOM file paths from
the Go API and returns structured detection results.

Endpoints:
  GET  /healthz         — liveness + backend availability
  POST /detect          — scan a study for burned-in PHI (synchronous)
  POST /redact          — detect + black out burned-in PHI pixels
  POST /scrub-text      — LLM-based free-text PHI scrubbing

Environment variables:
  PHI_TOOL                  — "auto" | "tesseract" | "google_vision" | "azure_vision" | "aws_textract"
  PHI_CONFIDENCE_THRESHOLD  — minimum OCR confidence 0.0–1.0 (default: 0.4)
  PHI_MIN_TEXT_LENGTH       — minimum text length to report (default: 3)
  TEXT_SCRUB_TOOL            — "auto" | "gemini" | "openai" | "anthropic" | "regex"
"""

import logging
import os
import shutil
import time
from dataclasses import asdict
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .config import cfg
from .backends.base import PHIDetectionBackend
from .backends.tesseract import TesseractBackend
from .backends.google_vision import GoogleVisionBackend
from .backends.azure_vision import AzureVisionBackend
from .backends.aws_textract import AWSTextractBackend
from .backends.gemini import GeminiBackend
from .backends.pixel_redact import redact_dicom_pixels
from .backends.text_scrub import TextScrubBackend, select_text_scrub_backend

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
    google_vision = GoogleVisionBackend(
        confidence_threshold=cfg.confidence_threshold,
        min_text_length=cfg.min_text_length,
    )
    azure_vision = AzureVisionBackend(
        confidence_threshold=cfg.confidence_threshold,
        min_text_length=cfg.min_text_length,
    )
    aws_textract = AWSTextractBackend(
        confidence_threshold=cfg.confidence_threshold,
        min_text_length=cfg.min_text_length,
    )
    gemini = GeminiBackend(
        confidence_threshold=cfg.confidence_threshold,
        min_text_length=cfg.min_text_length,
    )

    if cfg.phi_tool == "tesseract":
        candidates = [tesseract]
    elif cfg.phi_tool == "google_vision":
        candidates = [google_vision]
    elif cfg.phi_tool == "azure_vision":
        candidates = [azure_vision]
    elif cfg.phi_tool == "aws_textract":
        candidates = [aws_textract]
    elif cfg.phi_tool == "gemini":
        candidates = [gemini]
    else:  # "auto"
        # Priority: Gemini highest (best accuracy), then Cloud Vision, then
        # Azure Vision, then AWS Textract, then offline Tesseract fallback.
        candidates = [gemini, google_vision, azure_vision, aws_textract, tesseract]

    for backend in candidates:
        if backend.available():
            log.info("PHI detection backend selected: %s", backend.name)
            return backend

    raise RuntimeError(
        "No PHI detection backend is available. "
        "Install google-genai (Gemini), google-cloud-vision, "
        "azure-ai-vision-imageanalysis (Azure), "
        "boto3 (AWS Textract), or tesseract-ocr+pytesseract."
    )


_backend: PHIDetectionBackend | None = None
_text_scrub_backend: TextScrubBackend | None = None


def get_backend() -> PHIDetectionBackend:
    global _backend
    if _backend is None:
        _backend = _select_backend()
    return _backend


def get_text_scrub_backend() -> TextScrubBackend:
    global _text_scrub_backend
    if _text_scrub_backend is None:
        tool = os.environ.get("TEXT_SCRUB_TOOL", "auto")
        _text_scrub_backend = select_text_scrub_backend(tool)
    return _text_scrub_backend


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


@app.get("/health")
@app.get("/healthz")
def healthz() -> dict:
    try:
        backend = get_backend()
        text_scrub = get_text_scrub_backend()
        return {
            "status": "ok",
            "backend": backend.name,
            "available": backend.available(),
            "text_scrub_backend": text_scrub.name,
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


# ---------------------------------------------------------------------------
# POST /redact — detect + black out burned-in PHI
# ---------------------------------------------------------------------------

class RedactRequest(BaseModel):
    study_uid: str
    input_paths: list[str]
    output_dir: str


class RedactResponse(BaseModel):
    study_uid: str
    status: str  # "complete" | "failed"
    files_processed: int
    files_redacted: int  # files that had pixels blacked out
    findings: list[FindingModel]
    tool_used: str
    duration_seconds: float
    error: str | None = None


@app.post("/redact", response_model=RedactResponse)
def redact(req: RedactRequest) -> RedactResponse:
    if not req.input_paths:
        raise HTTPException(status_code=400, detail="input_paths is required")

    missing = [p for p in req.input_paths if not Path(p).exists()]
    if missing:
        raise HTTPException(
            status_code=400,
            detail=f"Input files not found: {missing[:5]}",
        )

    output_dir = Path(req.output_dir)
    if not output_dir.exists():
        try:
            output_dir.mkdir(parents=True, exist_ok=True)
        except Exception as e:
            raise HTTPException(
                status_code=400, detail=f"Cannot create output_dir: {e}"
            )

    backend = get_backend()
    log.info(
        "Redacting study %s: %d files (backend=%s)",
        req.study_uid, len(req.input_paths), backend.name,
    )

    start = time.monotonic()
    try:
        # Step 1: detect burned-in PHI
        file_findings = backend.detect(req.input_paths)

        # Build a lookup: filename → regions
        findings_by_file: dict[str, list[dict]] = {}
        for f in file_findings:
            findings_by_file[f.file] = [
                {"text": r.text, "confidence": r.confidence, "bbox": r.bbox}
                for r in f.regions
            ]

        # Step 2: redact or copy each file
        files_redacted = 0
        for input_path in req.input_paths:
            filename = Path(input_path).name
            output_path = str(output_dir / filename)
            regions = findings_by_file.get(filename, [])

            if regions:
                did_redact = redact_dicom_pixels(input_path, output_path, regions)
                if did_redact:
                    files_redacted += 1
            else:
                shutil.copy2(input_path, output_path)

        duration = time.monotonic() - start

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

        log.info(
            "Redaction complete for %s: %d/%d files redacted in %.1fs",
            req.study_uid, files_redacted, len(req.input_paths), duration,
        )
        return RedactResponse(
            study_uid=req.study_uid,
            status="complete",
            files_processed=len(req.input_paths),
            files_redacted=files_redacted,
            findings=findings,
            tool_used=backend.name,
            duration_seconds=duration,
        )
    except Exception as e:
        duration = time.monotonic() - start
        log.error("Redaction failed for %s: %s", req.study_uid, e, exc_info=True)
        return RedactResponse(
            study_uid=req.study_uid,
            status="failed",
            files_processed=0,
            files_redacted=0,
            findings=[],
            tool_used=backend.name,
            duration_seconds=duration,
            error=str(e),
        )


# ---------------------------------------------------------------------------
# POST /scrub-text — LLM-based free-text PHI scrubbing
# ---------------------------------------------------------------------------

class ScrubTextRequest(BaseModel):
    """Request to scrub PHI from free-text fields."""
    texts: list[str]
    context: dict | None = None


class ScrubTextItem(BaseModel):
    """Result for a single text field."""
    original: str
    scrubbed: str
    phi_found: bool


class ScrubTextResponse(BaseModel):
    """Response from /scrub-text."""
    status: str  # "complete" | "failed"
    results: list[ScrubTextItem]
    tool_used: str
    duration_seconds: float
    error: str | None = None


@app.post("/scrub-text", response_model=ScrubTextResponse)
def scrub_text(req: ScrubTextRequest) -> ScrubTextResponse:
    if not req.texts:
        raise HTTPException(status_code=400, detail="texts is required")

    backend = get_text_scrub_backend()
    log.info(
        "Scrubbing %d text fields (backend=%s)",
        len(req.texts), backend.name,
    )

    start = time.monotonic()
    try:
        results = []
        for text in req.texts:
            result = backend.scrub(text, req.context)
            results.append(ScrubTextItem(
                original=text,
                scrubbed=result["scrubbed"],
                phi_found=result["phi_found"],
            ))

        duration = time.monotonic() - start
        log.info(
            "Text scrub complete: %d fields, %d with PHI in %.1fs",
            len(results),
            sum(1 for r in results if r.phi_found),
            duration,
        )
        return ScrubTextResponse(
            status="complete",
            results=results,
            tool_used=backend.name,
            duration_seconds=duration,
        )
    except Exception as e:
        duration = time.monotonic() - start
        log.error("Text scrub failed: %s", e, exc_info=True)
        return ScrubTextResponse(
            status="failed",
            results=[],
            tool_used=backend.name,
            duration_seconds=duration,
            error=str(e),
        )
