"""
AEGIS Synthetic Brain MRI Service

FastAPI service that generates synthetic DICOM brain MRI studies on demand.
Writes files to the shared data volume so the Go API can import them via
the normal study ingest path.

Endpoints:
  GET  /healthz         -- liveness + backend availability
  POST /generate        -- generate a synthetic DICOM study

Environment variables:
  SYNTH_DATA_DIR        -- shared data volume root (default: /app/data)
  SYNTH_USE_GPU         -- enable MONAI LDM GPU backend (default: false)
  SYNTH_USE_GEMINI      -- enable Vertex AI Imagen backend (default: false)
  SYNTH_MAX_SLICES      -- maximum slices per request (default: 200)
  SYNTH_MAX_SIZE        -- maximum image size in pixels (default: 512)
"""

import logging
import time
from pathlib import Path

from fastapi import FastAPI
from pydantic import BaseModel

from .config import cfg
from .phantom import generate_phantom_slices, write_dicom_series

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)


# ── FastAPI app ───────────────────────────────────────────────────────────────

app = FastAPI(title="AEGIS Synthetic MRI Service", version="0.1.0")


class GenerateRequest(BaseModel):
    slices: int = 20
    size: int = 256
    seed: int = 42
    with_face: bool = False
    use_gpu: bool = False
    use_gemini: bool = False
    # Optional override for the output base directory (default: cfg.data_dir).
    output_base_dir: str | None = None


class GenerateResponse(BaseModel):
    study_uid: str
    output_dir: str
    file_count: int
    duration_seconds: float
    tool_used: str
    error: str | None = None


@app.get("/health")
@app.get("/healthz")
def healthz() -> dict:
    try:
        import numpy   # noqa: F401
        import pydicom  # noqa: F401

        tool = "phantom"
        if cfg.use_gemini:
            try:
                from .gemini_backend import _imagen_available
                tool = "gemini" if _imagen_available() else "phantom (gemini unavailable)"
            except ImportError:
                tool = "phantom (gemini unavailable)"
        elif cfg.use_gpu:
            try:
                from .monai_backend import generate_monai_slices  # noqa: F401
                tool = "monai"
            except ImportError:
                tool = "phantom (monai unavailable)"

        return {"status": "ok", "tool": tool, "gpu": cfg.use_gpu, "gemini": cfg.use_gemini}
    except Exception as exc:
        return {"status": "degraded", "error": str(exc)}


@app.post("/generate", response_model=GenerateResponse)
def generate(req: GenerateRequest) -> GenerateResponse:
    # Clamp request values to configured limits.
    n_slices = max(1, min(req.slices, cfg.max_slices))
    size = min(req.size, cfg.max_size)
    base_dir = req.output_base_dir or cfg.data_dir

    start = time.monotonic()
    try:
        use_gemini = req.use_gemini and cfg.use_gemini
        use_gpu = req.use_gpu and cfg.use_gpu and not use_gemini

        if use_gemini:
            from .gemini_backend import generate_gemini_slices
            tool_used = "gemini"
            slice_data = generate_gemini_slices(n_slices=n_slices, size=size, seed=req.seed)
        elif use_gpu:
            from .monai_backend import generate_monai_slices
            tool_used = "monai"
            slice_data = generate_monai_slices(n_slices=n_slices, size=size)
        else:
            tool_used = "phantom"
            slice_data = generate_phantom_slices(
                n_slices=n_slices,
                size=size,
                seed=req.seed,
                with_face=req.with_face,
            )

        # All slices share the same study_uid (embedded in each tuple).
        study_uid = slice_data[0][1] if slice_data else ""
        output_dir = str(Path(base_dir) / "synth" / study_uid)

        paths = write_dicom_series(slice_data, output_dir, size=size)
        duration = time.monotonic() - start

        log.info(
            "generate: study=%s tool=%s slices=%d size=%d with_face=%s in %.2fs",
            study_uid, tool_used, len(paths), size, req.with_face, duration,
        )
        return GenerateResponse(
            study_uid=study_uid,
            output_dir=output_dir,
            file_count=len(paths),
            duration_seconds=duration,
            tool_used=tool_used,
        )

    except Exception as exc:
        duration = time.monotonic() - start
        log.error("generate failed: %s", exc, exc_info=True)
        return GenerateResponse(
            study_uid="",
            output_dir="",
            file_count=0,
            duration_seconds=duration,
            tool_used="phantom",
            error=str(exc),
        )
