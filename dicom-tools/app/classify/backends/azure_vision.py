"""Azure Computer Vision backend for DICOM classification.

Inherits HeuristicBackend. Same pattern as GoogleVisionClassificationBackend:
heuristic runs first; if confidence < threshold, falls back to Azure Computer
Vision image tagging on the first rendered DICOM frame.

Requires azure-ai-vision-imageanalysis, azure-identity, Pillow, and numpy
installed. Credentials via DefaultAzureCredential (managed identity in
Container Apps, env vars for local dev).

Environment variables:
    AZURE_VISION_ENDPOINT  — Azure Computer Vision resource endpoint
                             (e.g. https://aegis-prod-vision.cognitiveservices.azure.com)
"""

import io
import logging
import os

from .heuristic import HeuristicBackend
from .base import ClassificationResult
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)

# -- Azure Vision tag -> body_part mapping ------------------------------------

_AZ_BODY_PART_MAP: list[tuple[str, str]] = [
    # HEAD/BRAIN
    ("brain", "HEAD"),
    ("skull", "HEAD"),
    ("head", "HEAD"),
    ("cranium", "HEAD"),
    ("cerebral", "HEAD"),
    ("temporal lobe", "HEAD"),
    ("frontal lobe", "HEAD"),
    ("meninges", "HEAD"),
    # CHEST
    ("lung", "CHEST"),
    ("chest", "CHEST"),
    ("thorax", "CHEST"),
    ("heart", "CHEST"),
    ("cardiac", "CHEST"),
    ("rib", "CHEST"),
    ("pleural", "CHEST"),
    ("mediastinum", "CHEST"),
    # ABDOMEN
    ("abdomen", "ABDOMEN"),
    ("liver", "ABDOMEN"),
    ("kidney", "ABDOMEN"),
    ("pancreas", "ABDOMEN"),
    ("pelvis", "ABDOMEN"),
    ("colon", "ABDOMEN"),
    ("bladder", "ABDOMEN"),
    # SPINE
    ("spine", "SPINE"),
    ("vertebra", "SPINE"),
    ("lumbar", "SPINE"),
    ("cervical", "SPINE"),
    ("disc", "SPINE"),
    ("sacrum", "SPINE"),
    # EXTREMITY
    ("knee", "EXTREMITY"),
    ("hip", "EXTREMITY"),
    ("shoulder", "EXTREMITY"),
    ("femur", "EXTREMITY"),
    ("tibia", "EXTREMITY"),
    ("extremity", "EXTREMITY"),
    ("limb", "EXTREMITY"),
    ("ankle", "EXTREMITY"),
    ("wrist", "EXTREMITY"),
    ("elbow", "EXTREMITY"),
    # NECK
    ("neck", "NECK"),
    ("thyroid", "NECK"),
    ("larynx", "NECK"),
    ("pharynx", "NECK"),
]

# -- Azure Vision tag -> modality mapping -------------------------------------

_AZ_MODALITY_MAP: list[tuple[str, str]] = [
    ("magnetic resonance", "MR"),
    ("mri", "MR"),
    ("computed tomography", "CT"),
    (" ct ", "CT"),
    ("x-ray", "CR"),
    ("radiograph", "CR"),
    ("ultrasound", "US"),
    ("sonogram", "US"),
    ("pet scan", "PT"),
    ("positron", "PT"),
    ("nuclear medicine", "NM"),
    ("fluoroscopy", "XA"),
    ("angiogram", "XA"),
]


def _az_tags_to_body_part(tags: list[str]) -> str | None:
    combined = " ".join(tags).lower()
    for phrase, part in _AZ_BODY_PART_MAP:
        if phrase.lower() in combined:
            return part
    return None


def _az_tags_to_modality(tags: list[str]) -> str | None:
    combined = " ".join(tags).lower()
    for phrase, mod in _AZ_MODALITY_MAP:
        if phrase.lower() in combined:
            return mod
    return None


class AzureVisionClassificationBackend(HeuristicBackend):
    """Heuristic backend augmented with Azure Computer Vision tag inference."""

    def __init__(self, confidence_threshold: float = 0.5):
        self._confidence_threshold = confidence_threshold
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

    def classify(self, input_paths: list[str]) -> ClassificationResult:
        # Run heuristic strategies 1-4 first (free, instant).
        result = super().classify(input_paths)

        if result.confidence >= self._confidence_threshold:
            return result

        log.info(
            "Heuristic confidence %.2f < threshold; trying Azure Computer Vision tags",
            result.confidence,
        )
        az_result = self._classify_with_vision(input_paths, result)
        return az_result if az_result.confidence > result.confidence else result

    def _classify_with_vision(
        self, input_paths: list[str], heuristic: ClassificationResult
    ) -> ClassificationResult:
        """Run Azure Vision image tagging on the first renderable DICOM frame."""
        from azure.ai.vision.imageanalysis.models import VisualFeatures

        client = self._get_client()

        for path in input_paths:
            img = dicom_to_pil(path)
            if img is None:
                continue

            try:
                buf = io.BytesIO()
                img.save(buf, format="JPEG")
                result = client.analyze(
                    image_data=buf.getvalue(),
                    visual_features=[VisualFeatures.TAGS],
                )

                if not result.tags or not result.tags.values:
                    continue

                tags = [
                    t.name for t in result.tags.values
                    if t.confidence >= 0.5
                ]
                log.debug("Azure Vision tags: %s", tags)

                body_part = _az_tags_to_body_part(tags) or heuristic.body_part
                modality = _az_tags_to_modality(tags) or heuristic.modality

                if body_part or modality:
                    return ClassificationResult(
                        modality=modality or "",
                        body_part=body_part or "",
                        confidence=0.70,
                        method="azure_vision_tags",
                    )

            except Exception as e:
                log.warning("Azure Vision tagging failed on %s: %s", path, e)
                continue

        return heuristic
