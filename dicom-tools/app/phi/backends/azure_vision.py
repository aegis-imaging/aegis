"""Azure Computer Vision backend for burned-in PHI detection.

Uses the Azure AI Vision Read API (OCR) on DICOM pixel data. Requires
azure-ai-vision-imageanalysis and azure-identity installed. Credentials via
DefaultAzureCredential (managed identity in Container Apps, env vars locally).

Environment variables:
    AZURE_VISION_ENDPOINT  — Azure Computer Vision resource endpoint
                             (e.g. https://aegis-prod-vision.cognitiveservices.azure.com)

Auto-selection priority: between gemini/google_vision and aws_textract.
"""

import io
import logging
import os
from pathlib import Path

from .base import PHIDetectionBackend, FileFinding, Region
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)


class AzureVisionBackend(PHIDetectionBackend):
    def __init__(self, confidence_threshold: float = 0.4, min_text_length: int = 3):
        self._confidence_threshold = confidence_threshold
        self._min_text_length = min_text_length
        self._client = None
        self._endpoint = os.environ.get("AZURE_VISION_ENDPOINT", "")

    @property
    def name(self) -> str:
        return "azure_vision"

    def available(self) -> bool:
        if not self._endpoint:
            return False
        try:
            from azure.ai.vision.imageanalysis import ImageAnalysisClient  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            from azure.ai.vision.imageanalysis import ImageAnalysisClient
            from azure.identity import DefaultAzureCredential

            self._client = ImageAnalysisClient(
                endpoint=self._endpoint,
                credential=DefaultAzureCredential(),
            )
        return self._client

    def detect(self, dicom_paths: list[str]) -> list[FileFinding]:
        from azure.ai.vision.imageanalysis.models import VisualFeatures

        findings: list[FileFinding] = []
        client = self._get_client()

        for path in dicom_paths:
            try:
                img = dicom_to_pil(path)
                if img is None:
                    continue

                buf = io.BytesIO()
                img.save(buf, format="JPEG")

                result = client.analyze(
                    image_data=buf.getvalue(),
                    visual_features=[VisualFeatures.READ],
                )

                regions = self._parse_read_result(result)
                if regions:
                    findings.append(FileFinding(file=Path(path).name, regions=regions))

            except Exception as e:
                log.warning("AzureVision failed on %s: %s", path, e)
                continue

        return findings

    def _parse_read_result(self, result) -> list[Region]:
        """Convert Azure Vision Read result to Region dataclasses.

        The Read API returns blocks -> lines -> words with bounding polygons
        and per-word confidence scores (0.0-1.0).
        """
        regions: list[Region] = []

        if not result.read or not result.read.blocks:
            return regions

        for block in result.read.blocks:
            for line in block.lines:
                for word in line.words:
                    text = (word.text or "").strip()
                    confidence = word.confidence or 0.0

                    if confidence < self._confidence_threshold:
                        continue
                    if len(text) < self._min_text_length:
                        continue

                    # Convert bounding polygon to [x, y, w, h].
                    bbox = [0, 0, 0, 0]
                    if word.bounding_polygon and len(word.bounding_polygon) >= 4:
                        xs = [p.x for p in word.bounding_polygon]
                        ys = [p.y for p in word.bounding_polygon]
                        x, y = min(xs), min(ys)
                        w = max(xs) - x
                        h = max(ys) - y
                        bbox = [int(x), int(y), int(w), int(h)]

                    regions.append(Region(
                        text=text,
                        confidence=round(confidence, 3),
                        bbox=bbox,
                    ))

        return regions
