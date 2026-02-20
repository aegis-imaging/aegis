"""AWS Textract backend for burned-in PHI detection.

Uses Textract detect_document_text on DICOM pixel data. Requires boto3
installed. Credentials via IAM role (ECS) or AWS_ACCESS_KEY_ID /
AWS_SECRET_ACCESS_KEY / AWS_DEFAULT_REGION env vars for local dev.

Auto-selection priority: MEDIUM (after google_vision, before tesseract).
"""

import io
import logging
from pathlib import Path

from .base import PHIDetectionBackend, FileFinding, Region
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)


class AWSTextractBackend(PHIDetectionBackend):
    def __init__(self, confidence_threshold: float = 0.4, min_text_length: int = 3):
        self._confidence_threshold = confidence_threshold
        self._min_text_length = min_text_length
        self._client = None

    @property
    def name(self) -> str:
        return "aws_textract"

    def available(self) -> bool:
        try:
            import boto3  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            import boto3
            self._client = boto3.client("textract")
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

                response = client.detect_document_text(
                    Document={"Bytes": buf.getvalue()}
                )

                regions = self._parse_blocks(response.get("Blocks", []))
                if regions:
                    findings.append(FileFinding(file=Path(path).name, regions=regions))

            except Exception as e:
                log.warning("AWSTextract failed on %s: %s", path, e)
                continue

        return findings

    def _parse_blocks(self, blocks: list) -> list[Region]:
        """Convert Textract Block objects to Region dataclasses.

        Uses WORD blocks for fine-grained bounding boxes.
        Confidence is 0-100 from the API, normalised to 0-1.

        Textract geometry is fractional (0-1) relative to image dimensions.
        We scale to 1000 for a practical integer representation that
        preserves relative positions for audit trail display.
        """
        regions: list[Region] = []
        for block in blocks:
            if block.get("BlockType") != "WORD":
                continue

            text = (block.get("Text") or "").strip()
            conf_raw = float(block.get("Confidence", 0))
            conf = conf_raw / 100.0

            if conf < self._confidence_threshold:
                continue
            if len(text) < self._min_text_length:
                continue

            geo = block.get("Geometry", {}).get("BoundingBox", {})
            x = int(geo.get("Left", 0) * 1000)
            y = int(geo.get("Top", 0) * 1000)
            w = int(geo.get("Width", 0) * 1000)
            h = int(geo.get("Height", 0) * 1000)

            regions.append(Region(
                text=text,
                confidence=round(conf, 3),
                bbox=[x, y, w, h],
            ))

        return regions
