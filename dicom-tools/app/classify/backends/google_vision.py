"""Google Cloud Vision backend for DICOM classification.

Inherits HeuristicBackend. Runs heuristic strategies 1-4 first (free, instant).
If confidence < threshold, renders DICOM pixels and runs Vision API
label_detection to extract medical imaging labels for body_part and modality.

Requires google-cloud-vision, Pillow, and numpy installed, plus Application
Default Credentials (or GOOGLE_APPLICATION_CREDENTIALS env var).
"""

import io
import logging

from .heuristic import HeuristicBackend
from .base import ClassificationResult
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)

# -- Label -> body_part mapping ------------------------------------------------
# Cloud Vision label_detection returns general-purpose English phrases.
# These are case-insensitive substring matches against the combined label text.

_VISION_BODY_PART_MAP: list[tuple[str, str]] = [
    # HEAD/BRAIN
    ("brain", "HEAD"),
    ("skull", "HEAD"),
    ("cranial", "HEAD"),
    ("head", "HEAD"),
    ("cerebral", "HEAD"),
    ("temporal lobe", "HEAD"),
    ("frontal lobe", "HEAD"),
    ("meninges", "HEAD"),
    # CHEST
    ("lung", "CHEST"),
    ("chest", "CHEST"),
    ("thorax", "CHEST"),
    ("cardiac", "CHEST"),
    ("heart", "CHEST"),
    ("rib", "CHEST"),
    ("pleural", "CHEST"),
    ("mediastinum", "CHEST"),
    # ABDOMEN
    ("abdomen", "ABDOMEN"),
    ("liver", "ABDOMEN"),
    ("kidney", "ABDOMEN"),
    ("pancreas", "ABDOMEN"),
    ("bowel", "ABDOMEN"),
    ("colon", "ABDOMEN"),
    ("pelvis", "ABDOMEN"),
    ("bladder", "ABDOMEN"),
    ("uterus", "ABDOMEN"),
    # SPINE
    ("spine", "SPINE"),
    ("vertebr", "SPINE"),
    ("lumbar", "SPINE"),
    ("cervical", "SPINE"),
    ("thoracic", "SPINE"),
    ("sacrum", "SPINE"),
    ("disc", "SPINE"),
    # EXTREMITY
    ("knee", "EXTREMITY"),
    ("hip", "EXTREMITY"),
    ("shoulder", "EXTREMITY"),
    ("femur", "EXTREMITY"),
    ("tibia", "EXTREMITY"),
    ("fibula", "EXTREMITY"),
    ("humerus", "EXTREMITY"),
    ("ankle", "EXTREMITY"),
    ("wrist", "EXTREMITY"),
    ("hand", "EXTREMITY"),
    ("foot", "EXTREMITY"),
    ("elbow", "EXTREMITY"),
    # NECK
    ("neck", "NECK"),
    ("thyroid", "NECK"),
    ("larynx", "NECK"),
    ("pharynx", "NECK"),
]

# -- Label -> modality mapping -------------------------------------------------

_VISION_MODALITY_MAP: list[tuple[str, str]] = [
    ("magnetic resonance", "MR"),
    ("mri", "MR"),
    ("computed tomography", "CT"),
    (" ct ", "CT"),
    ("x-ray", "CR"),
    ("radiograph", "CR"),
    ("ultrasound", "US"),
    ("sonogram", "US"),
    ("echocardiogram", "US"),
    ("pet scan", "PT"),
    ("positron", "PT"),
    ("nuclear medicine", "NM"),
    ("scintigraphy", "NM"),
    ("fluoroscopy", "XA"),
    ("angiogram", "XA"),
]


def _labels_to_body_part(labels: list[str]) -> str | None:
    """Return first matching body part from a list of label strings."""
    combined = " ".join(labels).lower()
    for phrase, part in _VISION_BODY_PART_MAP:
        if phrase.lower() in combined:
            return part
    return None


def _labels_to_modality(labels: list[str]) -> str | None:
    """Return first matching modality from a list of label strings."""
    combined = " ".join(labels).lower()
    for phrase, mod in _VISION_MODALITY_MAP:
        if phrase.lower() in combined:
            return mod
    return None


class GoogleVisionClassificationBackend(HeuristicBackend):
    """Heuristic backend augmented with Google Cloud Vision label inference."""

    def __init__(self, confidence_threshold: float = 0.5):
        self._confidence_threshold = confidence_threshold
        self._client = None

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

    def classify(self, input_paths: list[str]) -> ClassificationResult:
        # Run heuristic strategies 1-4 first (free, instant).
        result = super().classify(input_paths)

        if result.confidence >= self._confidence_threshold:
            log.debug(
                "Heuristic confidence %.2f >= threshold %.2f; skipping Vision API",
                result.confidence, self._confidence_threshold,
            )
            return result

        # Heuristic inconclusive -- augment with Vision label inference.
        log.info(
            "Heuristic confidence %.2f < threshold; trying Google Vision label_detection",
            result.confidence,
        )
        vision_result = self._classify_with_vision(input_paths, result)
        return vision_result if vision_result.confidence > result.confidence else result

    def _classify_with_vision(
        self, input_paths: list[str], heuristic: ClassificationResult
    ) -> ClassificationResult:
        """Run Vision API label_detection on the first renderable DICOM frame."""
        client = self._get_client()

        for path in input_paths:
            img = dicom_to_pil(path)
            if img is None:
                continue

            try:
                buf = io.BytesIO()
                img.save(buf, format="JPEG")
                # Pass image content as dict — the SDK accepts both proto
                # objects and dicts, avoids importing vision at method level.
                response = client.label_detection(
                    image={"content": buf.getvalue()}, max_results=20
                )
                if response.error.message:
                    log.warning("Vision API error: %s", response.error.message)
                    continue

                labels = [ann.description for ann in response.label_annotations]
                log.debug("Vision labels: %s", labels)

                body_part = _labels_to_body_part(labels) or heuristic.body_part
                modality = _labels_to_modality(labels) or heuristic.modality

                if body_part or modality:
                    return ClassificationResult(
                        modality=modality or "",
                        body_part=body_part or "",
                        confidence=0.70,
                        method="google_vision_labels",
                    )

            except Exception as e:
                log.warning("Vision label_detection failed on %s: %s", path, e)
                continue

        return heuristic
