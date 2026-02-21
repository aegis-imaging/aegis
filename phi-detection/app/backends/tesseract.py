"""Tesseract OCR backend for burned-in PHI detection.

Uses pytesseract + pydicom to extract pixel arrays from DICOM files and
run OCR to detect any overlaid text. Works offline with no cloud dependencies.
"""

import logging
from pathlib import Path

from .base import PHIDetectionBackend, FileFinding, Region
from .pixel_utils import dicom_to_pil

log = logging.getLogger(__name__)


class TesseractBackend(PHIDetectionBackend):
    def __init__(self, confidence_threshold: float = 0.4, min_text_length: int = 3):
        self._confidence_threshold = confidence_threshold
        self._min_text_length = min_text_length

    @property
    def name(self) -> str:
        return "tesseract"

    def available(self) -> bool:
        try:
            import pytesseract
            pytesseract.get_tesseract_version()
            return True
        except Exception:
            return False

    def detect(self, dicom_paths: list[str]) -> list[FileFinding]:
        import pytesseract

        findings: list[FileFinding] = []

        for path in dicom_paths:
            try:
                regions = self._scan_file(path, pytesseract)
                if regions:
                    findings.append(FileFinding(
                        file=Path(path).name,
                        regions=regions,
                    ))
            except Exception as e:
                log.warning("Failed to scan %s: %s", path, e)
                continue

        return findings

    def _scan_file(self, path: str, pytesseract) -> list[Region]:
        """Extract pixel data from a DICOM file and run OCR."""
        img = dicom_to_pil(path)
        if img is None:
            return []

        # Run Tesseract with detailed output (bounding boxes + confidence).
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

            # Tesseract returns -1 for non-text blocks.
            if conf < 0:
                continue

            # Normalise confidence to 0.0-1.0.
            conf_norm = conf / 100.0

            if conf_norm < self._confidence_threshold:
                continue
            if len(text) < self._min_text_length:
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
