"""Gemini multimodal backend for burned-in PHI detection.

Uses the unified google-genai SDK with Gemini (default: gemini-2.5-flash) to
visually inspect DICOM images and identify any burned-in text that could be
patient PHI (names, dates, accession numbers, study IDs, etc.).

Supports two authentication modes:
  - Vertex AI (default): Application Default Credentials / Workload Identity
  - AI Studio: API key via GEMINI_API_KEY env var

Auto-selection priority: HIGHEST — before google_vision, aws_textract, tesseract.

Environment variables:
    GEMINI_API_KEY       — AI Studio API key (if set, uses AI Studio instead of Vertex AI)
    GEMINI_PROJECT_ID    — GCP project for Vertex AI (default: read from ADC metadata)
    GEMINI_LOCATION      — Vertex AI location (default: us-central1)
    GEMINI_MODEL         — Gemini model to use (default: gemini-2.5-flash)
"""

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
    """Gemini Vision PHI detection backend using the unified google-genai SDK."""

    def __init__(
        self,
        confidence_threshold: float = 0.4,
        min_text_length: int = 3,
        project_id: str | None = None,
        location: str | None = None,
        model: str | None = None,
        api_key: str | None = None,
    ):
        self._confidence_threshold = confidence_threshold
        self._min_text_length = min_text_length
        self._project_id = project_id or os.environ.get("GEMINI_PROJECT_ID", "")
        self._location = location or os.environ.get("GEMINI_LOCATION", "us-central1")
        self._model_name = model or os.environ.get("GEMINI_MODEL", "gemini-2.5-flash")
        self._api_key = api_key or os.environ.get("GEMINI_API_KEY", "")
        self._client = None  # lazy-initialised

    @property
    def name(self) -> str:
        return "gemini"

    def available(self) -> bool:
        try:
            from google import genai  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            from google import genai

            if self._api_key:
                # AI Studio mode — API key auth.
                self._client = genai.Client(api_key=self._api_key)
            else:
                # Vertex AI mode — ADC / Workload Identity.
                init_kwargs: dict = {
                    "vertexai": True,
                    "location": self._location,
                }
                if self._project_id:
                    init_kwargs["project"] = self._project_id
                self._client = genai.Client(**init_kwargs)
        return self._client

    def detect(self, dicom_paths: list[str]) -> list[FileFinding]:
        from google.genai import types
        from pathlib import Path

        findings: list[FileFinding] = []
        client = self._get_client()

        for path in dicom_paths:
            try:
                img = dicom_to_pil(path)
                if img is None:
                    continue

                # Encode image as JPEG bytes for the multimodal API.
                buf = io.BytesIO()
                img.save(buf, format="JPEG", quality=85)
                image_bytes = buf.getvalue()

                image_part = types.Part.from_bytes(
                    data=image_bytes,
                    mime_type="image/jpeg",
                )

                response = client.models.generate_content(
                    model=self._model_name,
                    contents=[
                        image_part,
                        "Identify all burned-in PHI text in this image.",
                    ],
                    config=types.GenerateContentConfig(
                        system_instruction=_SYSTEM_PROMPT,
                        response_mime_type="application/json",
                    ),
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

        # With response_mime_type="application/json" the SDK should return
        # clean JSON, but handle code fences as a safety net.
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
