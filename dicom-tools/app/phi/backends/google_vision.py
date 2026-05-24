"""Google Cloud Vision backend for burned-in PHI detection.

Uses the Vision API text_detection (OCR) on DICOM pixel data. Requires
google-cloud-vision installed and Application Default Credentials (or
GOOGLE_APPLICATION_CREDENTIALS env var pointing to a service account key).

Auto-selection priority: HIGHEST (before aws_textract, before tesseract).
"""

import io
import logging
from pathlib import Path

from .base import PHIDetectionBackend, FileFinding, Region
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)


class GoogleVisionBackend(PHIDetectionBackend):
    def __init__(self, confidence_threshold: float = 0.4, min_text_length: int = 3):
        self._confidence_threshold = confidence_threshold
        self._min_text_length = min_text_length
        self._client = None  # lazy-initialised

    @property
    def name(self) -> str:
        return "google_vision"

    def available(self) -> bool:
        try:
            from google.cloud import vision  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            from google.cloud import vision
            self._client = vision.ImageAnnotatorClient()
        return self._client

    def detect(self, dicom_paths: list[str]) -> list[FileFinding]:
        findings: list[FileFinding] = []
        client = self._get_client()

        for path in dicom_paths:
            try:
                img = dicom_to_pil(path)
                if img is None:
                    continue

                buf = io.BytesIO()
                img.save(buf, format="JPEG")

                # Pass image content as dict — the SDK accepts both proto
                # objects and dicts, and this avoids importing vision at
                # the method level (tests inject _client directly).
                response = client.text_detection(image={"content": buf.getvalue()})
                if response.error.message:
                    log.warning("Vision API error on %s: %s", path, response.error.message)
                    continue

                regions = self._parse_annotations(response.text_annotations)
                if regions:
                    findings.append(FileFinding(file=Path(path).name, regions=regions))

            except Exception as e:
                log.warning("GoogleVision failed on %s: %s", path, e)
                continue

        return findings

    def _parse_annotations(self, annotations) -> list[Region]:
        """Convert Vision API TextAnnotation objects to Region dataclasses.

        The first annotation is the full-page text block; skip it and use
        individual word/symbol annotations which carry bounding polygons.

        Note: Vision API text_detection does not return per-word confidence
        scores. We use 1.0 as a sentinel — the API only returns results it
        is confident about.
        """
        regions: list[Region] = []
        # Skip index 0 (full-page aggregate).
        for ann in annotations[1:]:
            text = (ann.description or "").strip()
            if len(text) < self._min_text_length:
                continue

            confidence = 1.0

            # Convert bounding polygon to [x, y, w, h].
            verts = ann.bounding_poly.vertices if ann.bounding_poly else []
            if len(verts) >= 4:
                xs = [v.x for v in verts]
                ys = [v.y for v in verts]
                x, y = min(xs), min(ys)
                w = max(xs) - x
                h = max(ys) - y
                bbox = [x, y, w, h]
            else:
                bbox = [0, 0, 0, 0]

            regions.append(Region(text=text, confidence=confidence, bbox=bbox))

        return regions
