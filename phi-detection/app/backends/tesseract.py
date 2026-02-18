"""Tesseract OCR backend for burned-in PHI detection.

Uses pytesseract + pydicom to extract pixel arrays from DICOM files and
run OCR to detect any overlaid text. Works offline with no cloud dependencies.
"""

import logging
from pathlib import Path

import numpy as np

from .base import PHIDetectionBackend, FileFinding, Region

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
        import pydicom
        import pytesseract
        from PIL import Image

        findings: list[FileFinding] = []

        for path in dicom_paths:
            try:
                regions = self._scan_file(path, pydicom, pytesseract, Image, np)
                if regions:
                    findings.append(FileFinding(
                        file=Path(path).name,
                        regions=regions,
                    ))
            except Exception as e:
                log.warning("Failed to scan %s: %s", path, e)
                continue

        return findings

    def _scan_file(self, path: str, pydicom, pytesseract, Image, np) -> list[Region]:
        """Extract pixel data from a DICOM file and run OCR."""
        ds = pydicom.dcmread(path)

        if not hasattr(ds, "PixelData"):
            return []

        try:
            arr = ds.pixel_array
        except Exception:
            return []

        # Handle multi-frame: scan the first frame only (burned-in text is
        # typically consistent across frames).
        if arr.ndim == 3 and arr.shape[0] > 1 and arr.shape[2] != 3:
            arr = arr[0]

        # Apply window/level if available (makes text more visible for OCR).
        arr = self._apply_windowing(ds, arr, np)

        # Convert to 8-bit grayscale for OCR.
        arr = self._to_uint8(arr, np)

        img = Image.fromarray(arr)

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

            # Normalise confidence to 0.0–1.0.
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

    @staticmethod
    def _apply_windowing(ds, arr: np.ndarray, np) -> np.ndarray:
        """Apply DICOM window center/width for better OCR contrast."""
        wc = getattr(ds, "WindowCenter", None)
        ww = getattr(ds, "WindowWidth", None)
        if wc is None or ww is None:
            return arr

        # Handle multi-value window center/width (take first).
        if hasattr(wc, "__iter__"):
            wc = wc[0] if len(wc) > 0 else wc
        if hasattr(ww, "__iter__"):
            ww = ww[0] if len(ww) > 0 else ww

        wc = float(wc)
        ww = float(ww)
        if ww <= 0:
            return arr

        low = wc - ww / 2
        high = wc + ww / 2
        arr = arr.astype(np.float64)
        arr = np.clip(arr, low, high)
        return arr

    @staticmethod
    def _to_uint8(arr: np.ndarray, np) -> np.ndarray:
        """Normalise array to 0–255 uint8."""
        arr = arr.astype(np.float64)
        mn, mx = arr.min(), arr.max()
        if mx - mn == 0:
            return np.zeros(arr.shape[:2], dtype=np.uint8)
        arr = (arr - mn) / (mx - mn) * 255.0
        return arr.astype(np.uint8)
