"""Heuristic classification backend — DICOM tag analysis + pattern matching.

Classification strategy (priority order):
1. Direct DICOM tags — Modality (0008,0060) + BodyPartExamined (0018,0015)
2. SOP Class UID (0008,0016) → modality mapping
3. SeriesDescription / ProtocolName → body part regex
4. StudyDescription → body part regex (lower confidence)
5. Fallback → empty (inconclusive)
"""

import logging
import os
import re

import pydicom
from pydicom.errors import InvalidDicomError

from .base import ClassificationBackend, ClassificationResult

log = logging.getLogger(__name__)

# ── SOP Class UID → Modality mapping ────────────────────────────────────────

_SOP_MODALITY_MAP: list[tuple[str, str]] = [
    ("1.2.840.10008.5.1.4.1.1.2",   "CT"),   # CT Image Storage
    ("1.2.840.10008.5.1.4.1.1.4",   "MR"),   # MR Image Storage
    ("1.2.840.10008.5.1.4.1.1.128", "PT"),   # PET Image Storage
    ("1.2.840.10008.5.1.4.1.1.6",   "US"),   # Ultrasound Image Storage
    ("1.2.840.10008.5.1.4.1.1.1",   "CR"),   # CR / DX Image Storage
    ("1.2.840.10008.5.1.4.1.1.7",   "SC"),   # Secondary Capture
    ("1.2.840.10008.5.1.4.1.1.20",  "NM"),   # Nuclear Medicine
    ("1.2.840.10008.5.1.4.1.1.12",  "XA"),   # X-Ray Angiographic
]

# ── Body part patterns ──────────────────────────────────────────────────────

_BODY_PART_PATTERNS: list[tuple[re.Pattern, str]] = [
    (re.compile(r"brain|head|cranial|neuro|skull|crani", re.IGNORECASE), "HEAD"),
    (re.compile(r"chest|thorax|lung|cardiac|heart|pulmon", re.IGNORECASE), "CHEST"),
    (re.compile(r"abdom|pelvis|liver|kidney|renal|hepat|pancrea", re.IGNORECASE), "ABDOMEN"),
    (re.compile(r"spine|cervical|thoracic|lumbar|sacr|vertebr", re.IGNORECASE), "SPINE"),
    (re.compile(r"knee|hip|shoulder|ankle|wrist|extremit|femur|tibia|humer|elbow|foot|hand", re.IGNORECASE), "EXTREMITY"),
    (re.compile(r"neck|thyroid|pharyn|laryn", re.IGNORECASE), "NECK"),
]


def _match_body_part(text: str) -> str | None:
    """Return the first matching body part for a text string, or None."""
    for pattern, part in _BODY_PART_PATTERNS:
        if pattern.search(text):
            return part
    return None


def _read_first_valid_dicom(paths: list[str]) -> pydicom.Dataset | None:
    """Try to read the first valid DICOM file from a list of paths."""
    for p in paths:
        if not os.path.isfile(p):
            continue
        try:
            ds = pydicom.dcmread(p, stop_before_pixels=True, force=True)
            return ds
        except (InvalidDicomError, Exception) as exc:
            log.debug("Skip non-DICOM or unreadable file %s: %s", p, exc)
            continue
    return None


class HeuristicBackend(ClassificationBackend):
    """Local heuristic backend — DICOM tag analysis, no external services."""

    @property
    def name(self) -> str:
        return "heuristic"

    def available(self) -> bool:
        return True  # Always available — pure Python, no dependencies beyond pydicom

    def classify(self, input_paths: list[str]) -> ClassificationResult:
        ds = _read_first_valid_dicom(input_paths)
        if ds is None:
            log.warning("No valid DICOM files found in %d paths", len(input_paths))
            return ClassificationResult()

        # Strategy 1: Direct DICOM tags
        modality = str(getattr(ds, "Modality", "") or "").strip()
        body_part = str(getattr(ds, "BodyPartExamined", "") or "").strip().upper()
        if modality and body_part:
            return ClassificationResult(
                modality=modality,
                body_part=body_part,
                confidence=0.95,
                method="dicom_tags",
            )

        # Strategy 2: SOP Class UID → modality
        sop_uid = str(getattr(ds, "SOPClassUID", "") or "")
        if not modality and sop_uid:
            for prefix, mod in _SOP_MODALITY_MAP:
                if sop_uid.startswith(prefix):
                    modality = mod
                    break

        # Strategy 3: SeriesDescription / ProtocolName → body part
        if not body_part:
            for field_name in ("SeriesDescription", "ProtocolName"):
                val = str(getattr(ds, field_name, "") or "").strip()
                if val:
                    match = _match_body_part(val)
                    if match:
                        body_part = match
                        if modality:
                            return ClassificationResult(
                                modality=modality,
                                body_part=body_part,
                                confidence=0.75 if modality else 0.65,
                                method="series_description",
                            )
                        break

        # Strategy 4: StudyDescription → body part (lower confidence)
        if not body_part:
            study_desc = str(getattr(ds, "StudyDescription", "") or "").strip()
            if study_desc:
                match = _match_body_part(study_desc)
                if match:
                    body_part = match

        # Determine best confidence/method based on what we found
        if modality and body_part:
            # We had to combine multiple strategies
            if sop_uid and any(sop_uid.startswith(p) for p, _ in _SOP_MODALITY_MAP):
                return ClassificationResult(modality=modality, body_part=body_part,
                                            confidence=0.80, method="sop_class+description")
            return ClassificationResult(modality=modality, body_part=body_part,
                                        confidence=0.65, method="study_description")

        if modality and not body_part:
            return ClassificationResult(modality=modality, body_part="",
                                        confidence=0.60, method="sop_class" if sop_uid else "dicom_tags")

        if body_part and not modality:
            return ClassificationResult(modality="", body_part=body_part,
                                        confidence=0.50, method="description_only")

        # Fallback: inconclusive
        return ClassificationResult()
