"""Local Tesseract OCR detection for burned-in PHI in DICOM pixel data.

Simplified standalone version of phi-detection/app/backends/tesseract.py.
Requires pytesseract and Tesseract OCR binary to be installed.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)


@dataclass
class Region:
    """A detected text region in a DICOM image."""

    text: str
    confidence: float
    bbox: list[int]  # [x, y, w, h]


@dataclass
class FileFinding:
    """OCR findings for a single DICOM file."""

    file: str
    regions: list[Region] = field(default_factory=list)


def tesseract_available() -> bool:
    """Check whether Tesseract OCR is installed and accessible."""
    try:
        import pytesseract
        pytesseract.get_tesseract_version()
        return True
    except Exception:
        return False


def detect_burned_in_text(
    dicom_paths: list[str],
    confidence_threshold: float = 0.4,
    min_text_length: int = 3,
) -> list[FileFinding]:
    """Scan DICOM files for burned-in text using Tesseract OCR.

    Args:
        dicom_paths: List of paths to DICOM files.
        confidence_threshold: Minimum OCR confidence (0.0-1.0).
        min_text_length: Minimum text length to report.

    Returns:
        List of FileFinding objects, one per file that had detections.
    """
    import pytesseract

    findings: list[FileFinding] = []

    for path in dicom_paths:
        try:
            regions = _scan_file(path, pytesseract, confidence_threshold, min_text_length)
            if regions:
                findings.append(FileFinding(
                    file=Path(path).name,
                    regions=regions,
                ))
        except Exception as e:
            log.warning("Failed to scan %s: %s", path, e)
            continue

    return findings


def detect_file(
    path: str,
    confidence_threshold: float = 0.4,
    min_text_length: int = 3,
) -> list[Region]:
    """Scan a single DICOM file for burned-in text.

    Returns a list of detected regions (empty if no text found or no pixel data).
    """
    import pytesseract
    return _scan_file(path, pytesseract, confidence_threshold, min_text_length)


def _scan_file(
    path: str,
    pytesseract: object,
    confidence_threshold: float,
    min_text_length: int,
) -> list[Region]:
    """Extract pixel data from a DICOM file and run OCR."""
    img = dicom_to_pil(path)
    if img is None:
        return []

    try:
        data = pytesseract.image_to_data(img, output_type=pytesseract.Output.DICT)
    except Exception as e:
        log.debug("Tesseract failed on %s: %s", path, e)
        return []

    regions: list[Region] = []
    n = len(data["text"])
    for i in range(n):
        text = data["text"][i].strip()
        conf = float(data["conf"][i])

        # Tesseract returns -1 for non-text blocks
        if conf < 0:
            continue

        conf_norm = conf / 100.0

        if conf_norm < confidence_threshold:
            continue
        if len(text) < min_text_length:
            continue

        regions.append(Region(
            text=text,
            confidence=round(conf_norm, 3),
            bbox=[
                data["left"][i],
                data["top"][i],
                data["width"][i],
                data["height"][i],
            ],
        ))

    return regions
