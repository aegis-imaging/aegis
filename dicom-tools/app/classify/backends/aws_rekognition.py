"""AWS Rekognition backend for DICOM classification.

Inherits HeuristicBackend. Same pattern as GoogleVisionClassificationBackend:
heuristic runs first; if confidence < threshold, falls back to Rekognition
detect_labels on the first rendered DICOM frame.

Requires boto3, Pillow, and numpy installed. Credentials via IAM role (ECS)
or AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_DEFAULT_REGION env vars.
"""

import io
import logging

from .heuristic import HeuristicBackend
from .base import ClassificationResult
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)

# -- Rekognition label -> body_part mapping ------------------------------------

_REKOG_BODY_PART_MAP: list[tuple[str, str]] = [
    # HEAD/BRAIN
    ("brain", "HEAD"),
    ("skull", "HEAD"),
    ("head", "HEAD"),
    ("cranium", "HEAD"),
    ("neural", "HEAD"),
    # CHEST
    ("lung", "CHEST"),
    ("chest", "CHEST"),
    ("thorax", "CHEST"),
    ("heart", "CHEST"),
    ("cardiac", "CHEST"),
    ("rib cage", "CHEST"),
    # ABDOMEN
    ("abdomen", "ABDOMEN"),
    ("liver", "ABDOMEN"),
    ("kidney", "ABDOMEN"),
    ("pancreas", "ABDOMEN"),
    ("pelvis", "ABDOMEN"),
    ("colon", "ABDOMEN"),
    # SPINE
    ("spine", "SPINE"),
    ("vertebra", "SPINE"),
    ("lumbar", "SPINE"),
    ("cervical", "SPINE"),
    ("disc", "SPINE"),
    # EXTREMITY
    ("knee", "EXTREMITY"),
    ("hip", "EXTREMITY"),
    ("shoulder", "EXTREMITY"),
    ("femur", "EXTREMITY"),
    ("tibia", "EXTREMITY"),
    ("extremity", "EXTREMITY"),
    ("limb", "EXTREMITY"),
    # NECK
    ("neck", "NECK"),
    ("thyroid", "NECK"),
]

# -- Rekognition label -> modality mapping -------------------------------------

_REKOG_MODALITY_MAP: list[tuple[str, str]] = [
    ("mri", "MR"),
    ("magnetic", "MR"),
    ("ct scan", "CT"),
    ("computed tom", "CT"),
    ("x-ray", "CR"),
    ("radiograph", "CR"),
    ("ultrasound", "US"),
    ("pet", "PT"),
    ("nuclear", "NM"),
    ("fluoroscopy", "XA"),
]


def _rekog_labels_to_body_part(labels: list[str]) -> str | None:
    """Return first matching body part from a list of Rekognition labels."""
    combined = " ".join(labels).lower()
    for phrase, part in _REKOG_BODY_PART_MAP:
        if phrase.lower() in combined:
            return part
    return None


def _rekog_labels_to_modality(labels: list[str]) -> str | None:
    """Return first matching modality from a list of Rekognition labels."""
    combined = " ".join(labels).lower()
    for phrase, mod in _REKOG_MODALITY_MAP:
        if phrase.lower() in combined:
            return mod
    return None


class AWSRekognitionClassificationBackend(HeuristicBackend):
    """Heuristic backend augmented with AWS Rekognition label inference."""

    def __init__(self, confidence_threshold: float = 0.5):
        self._confidence_threshold = confidence_threshold
        self._client = None

    @property
    def name(self) -> str:
        return "aws_rekognition"

    def available(self) -> bool:
        try:
            import boto3  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            import boto3
            self._client = boto3.client("rekognition")
        return self._client

    def classify(self, input_paths: list[str]) -> ClassificationResult:
        # Run heuristic strategies 1-4 first (free, instant).
        result = super().classify(input_paths)

        if result.confidence >= self._confidence_threshold:
            return result

        log.info(
            "Heuristic confidence %.2f < threshold; trying AWS Rekognition detect_labels",
            result.confidence,
        )
        rekog_result = self._classify_with_rekognition(input_paths, result)
        return rekog_result if rekog_result.confidence > result.confidence else result

    def _classify_with_rekognition(
        self, input_paths: list[str], heuristic: ClassificationResult
    ) -> ClassificationResult:
        """Run Rekognition detect_labels on the first renderable DICOM frame."""
        client = self._get_client()

        for path in input_paths:
            img = dicom_to_pil(path)
            if img is None:
                continue

            try:
                buf = io.BytesIO()
                img.save(buf, format="JPEG")
                response = client.detect_labels(
                    Image={"Bytes": buf.getvalue()},
                    MaxLabels=20,
                    MinConfidence=50,
                )

                labels = [lbl["Name"] for lbl in response.get("Labels", [])]
                log.debug("Rekognition labels: %s", labels)

                body_part = _rekog_labels_to_body_part(labels) or heuristic.body_part
                modality = _rekog_labels_to_modality(labels) or heuristic.modality

                if body_part or modality:
                    return ClassificationResult(
                        modality=modality or "",
                        body_part=body_part or "",
                        confidence=0.70,
                        method="aws_rekognition_labels",
                    )

            except Exception as e:
                log.warning("Rekognition detect_labels failed on %s: %s", path, e)
                continue

        return heuristic
