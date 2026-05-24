"""Gemini multimodal backend for DICOM classification.

Inherits HeuristicBackend. Runs heuristic strategies 1-4 first (free, instant).
If confidence < threshold, renders DICOM pixels and calls Gemini multimodal to
classify modality and body part from the image content.

Uses the unified google-genai SDK. Supports two authentication modes:
  - Vertex AI (default): Application Default Credentials / Workload Identity
  - AI Studio: API key via GEMINI_API_KEY env var

Requires google-genai, Pillow, and numpy installed.

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

from .heuristic import HeuristicBackend
from .base import ClassificationResult
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)

_SYSTEM_PROMPT = """You are a medical imaging classification assistant.
Your task is to identify the imaging modality and body part from a DICOM medical image.

For modality, use standard DICOM codes: MR, CT, PT, US, CR, NM, XA, SC, DX.
For body part, use: HEAD, CHEST, ABDOMEN, SPINE, EXTREMITY, NECK, or empty string if unknown.

Respond ONLY with a valid JSON object with these fields:
  - "modality": the DICOM modality code (string, or empty if unknown)
  - "body_part": the body part (string, or empty if unknown)

Do not include any explanation — only the JSON object."""


class GeminiClassificationBackend(HeuristicBackend):
    """Heuristic backend augmented with Gemini multimodal classification."""

    def __init__(
        self,
        confidence_threshold: float = 0.5,
        project_id: str | None = None,
        location: str | None = None,
        model: str | None = None,
        api_key: str | None = None,
    ):
        self._confidence_threshold = confidence_threshold
        self._project_id = project_id or os.environ.get("GEMINI_PROJECT_ID", "")
        self._location = location or os.environ.get("GEMINI_LOCATION", "us-central1")
        self._model_name = model or os.environ.get("GEMINI_MODEL", "gemini-2.5-flash")
        self._api_key = api_key or os.environ.get("GEMINI_API_KEY", "")
        self._client = None

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
                self._client = genai.Client(api_key=self._api_key)
            else:
                init_kwargs: dict = {
                    "vertexai": True,
                    "location": self._location,
                }
                if self._project_id:
                    init_kwargs["project"] = self._project_id
                self._client = genai.Client(**init_kwargs)
        return self._client

    def classify(self, input_paths: list[str]) -> ClassificationResult:
        # Run heuristic strategies 1-4 first (free, instant).
        result = super().classify(input_paths)

        if result.confidence >= self._confidence_threshold:
            log.debug(
                "Heuristic confidence %.2f >= threshold %.2f; skipping Gemini API",
                result.confidence, self._confidence_threshold,
            )
            return result

        # Heuristic inconclusive — augment with Gemini multimodal.
        log.info(
            "Heuristic confidence %.2f < threshold; trying Gemini classification",
            result.confidence,
        )
        gemini_result = self._classify_with_gemini(input_paths, result)
        return gemini_result if gemini_result.confidence > result.confidence else result

    def _classify_with_gemini(
        self, input_paths: list[str], heuristic: ClassificationResult
    ) -> ClassificationResult:
        """Run Gemini multimodal classification on the first renderable DICOM frame."""
        from google.genai import types

        client = self._get_client()

        for path in input_paths:
            img = dicom_to_pil(path)
            if img is None:
                continue

            try:
                buf = io.BytesIO()
                img.save(buf, format="JPEG", quality=85)

                image_part = types.Part.from_bytes(
                    data=buf.getvalue(),
                    mime_type="image/jpeg",
                )

                response = client.models.generate_content(
                    model=self._model_name,
                    contents=[
                        image_part,
                        "Classify this medical image: what is the imaging modality and body part?",
                    ],
                    config=types.GenerateContentConfig(
                        system_instruction=_SYSTEM_PROMPT,
                        response_mime_type="application/json",
                    ),
                )

                modality, body_part = self._parse_response(response.text, heuristic)

                if modality or body_part:
                    return ClassificationResult(
                        modality=modality or "",
                        body_part=body_part or "",
                        confidence=0.75,
                        method="gemini_multimodal",
                    )

            except Exception as e:
                log.warning("Gemini classification failed on %s: %s", path, e)
                continue

        return heuristic

    def _parse_response(
        self, text: str, heuristic: ClassificationResult
    ) -> tuple[str, str]:
        """Extract modality and body_part from Gemini's JSON response."""
        if not text:
            return "", ""

        # Strip markdown code fences as a safety net.
        cleaned = re.sub(r"```(?:json)?\s*", "", text).strip().rstrip("`").strip()

        try:
            data = json.loads(cleaned)
        except (json.JSONDecodeError, ValueError):
            log.debug("Gemini returned non-JSON: %s", cleaned[:200])
            return "", ""

        if not isinstance(data, dict):
            return "", ""

        modality = str(data.get("modality", "") or "").strip().upper()
        body_part = str(data.get("body_part", "") or "").strip().upper()

        # Validate modality is a known DICOM code.
        valid_modalities = {"MR", "CT", "PT", "US", "CR", "NM", "XA", "SC", "DX"}
        if modality not in valid_modalities:
            modality = heuristic.modality

        # Validate body_part is a known value.
        valid_body_parts = {"HEAD", "CHEST", "ABDOMEN", "SPINE", "EXTREMITY", "NECK"}
        if body_part not in valid_body_parts:
            body_part = heuristic.body_part

        return modality, body_part
