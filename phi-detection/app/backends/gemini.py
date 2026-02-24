"""Gemini multimodal backend for burned-in PHI detection.

Uses Vertex AI Gemini (gemini-2.0-flash) to visually inspect DICOM images
and identify any burned-in text that could be patient PHI (names, dates,
accession numbers, study IDs, etc.).

This backend uses Application Default Credentials (ADC) — no API key
required when running on GCP Cloud Run with Workload Identity.

Auto-selection priority: HIGHEST — before google_vision, aws_textract, tesseract.

Environment variables:
    GEMINI_PROJECT_ID    — GCP project for Vertex AI (default: read from ADC metadata)
    GEMINI_LOCATION      — Vertex AI location (default: us-central1)
    GEMINI_MODEL         — Gemini model to use (default: gemini-2.0-flash)
"""

import base64
import io
import json
import logging
import os
import re

from .base import PHIDetectionBackend, FileFinding, Region
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)

_SYSTEM_PROMPT = """You are a medical imaging HIPAA compliance assistant.
Your task is to identify any burned-in text visible in medical images that
could be Protected Health Information (PHI): patient names, dates of birth,
medical record numbers, accession numbers, study dates, facility names,
physician names, or any other personally identifiable information.

Respond ONLY with a valid JSON array. Each element must have:
  - "text": the exact text string found
  - "confidence": a float from 0.0 to 1.0

If no PHI text is found, respond with an empty JSON array: []

Do not include any explanation — only the JSON array."""


class GeminiBackend(PHIDetectionBackend):
    """Gemini Vision PHI detection backend using Vertex AI."""

    def __init__(
        self,
        confidence_threshold: float = 0.4,
        min_text_length: int = 3,
        project_id: str | None = None,
        location: str | None = None,
        model: str | None = None,
    ):
        self._confidence_threshold = confidence_threshold
        self._min_text_length = min_text_length
        self._project_id = project_id or os.environ.get("GEMINI_PROJECT_ID", "")
        self._location = location or os.environ.get("GEMINI_LOCATION", "us-central1")
        self._model_name = model or os.environ.get("GEMINI_MODEL", "gemini-2.0-flash")
        self._model = None  # lazy-initialised

    @property
    def name(self) -> str:
        return "gemini"

    def available(self) -> bool:
        try:
            import vertexai  # noqa: F401
            from vertexai.generative_models import GenerativeModel  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_model(self):
        if self._model is None:
            import vertexai
            from vertexai.generative_models import GenerativeModel

            init_kwargs: dict = {"location": self._location}
            if self._project_id:
                init_kwargs["project"] = self._project_id
            vertexai.init(**init_kwargs)
            self._model = GenerativeModel(
                self._model_name,
                system_instruction=_SYSTEM_PROMPT,
            )
        return self._model

    def detect(self, dicom_paths: list[str]) -> list[FileFinding]:
        from vertexai.generative_models import Part
        from pathlib import Path

        findings: list[FileFinding] = []
        model = self._get_model()

        for path in dicom_paths:
            try:
                img = dicom_to_pil(path)
                if img is None:
                    continue

                # Encode image as JPEG bytes for the multimodal API.
                buf = io.BytesIO()
                img.save(buf, format="JPEG", quality=85)
                image_bytes = buf.getvalue()

                image_part = Part.from_data(
                    data=image_bytes,
                    mime_type="image/jpeg",
                )

                response = model.generate_content(
                    [image_part, "Identify all burned-in PHI text in this image."]
                )

                regions = self._parse_response(response.text)
                if regions:
                    findings.append(FileFinding(file=Path(path).name, regions=regions))

            except Exception as e:
                log.warning("Gemini PHI detection failed on %s: %s", path, e)
                continue

        return findings

    def _parse_response(self, text: str) -> list[Region]:
        """Extract structured regions from Gemini's JSON response."""
        if not text:
            return []

        # Gemini sometimes wraps JSON in markdown code fences — strip them.
        cleaned = re.sub(r"```(?:json)?\s*", "", text).strip().rstrip("`").strip()

        # Extract the first JSON array from the text.
        match = re.search(r"\[.*\]", cleaned, re.DOTALL)
        if not match:
            return []

        try:
            items = json.loads(match.group())
        except (json.JSONDecodeError, ValueError):
            log.debug("Gemini returned non-JSON: %s", cleaned[:200])
            return []

        regions: list[Region] = []
        for item in items:
            if not isinstance(item, dict):
                continue
            text_val = str(item.get("text", "")).strip()
            if len(text_val) < self._min_text_length:
                continue
            confidence = float(item.get("confidence", 1.0))
            if confidence < self._confidence_threshold:
                continue
            regions.append(Region(text=text_val, confidence=confidence, bbox=[0, 0, 0, 0]))

        return regions
