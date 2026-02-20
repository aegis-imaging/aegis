"""Shared DICOM pixel extraction utilities.

Converts a DICOM file's pixel array to a PIL Image suitable for OCR
or image-label inference. Used by all PHI detection backends.
"""

import logging
from typing import Optional

import numpy as np

log = logging.getLogger(__name__)


def dicom_to_pil(path: str) -> Optional["PIL.Image.Image"]:
    """Read a DICOM file and return a grayscale PIL Image.

    Applies DICOM windowing if WindowCenter/WindowWidth are present.
    Returns None if the file has no pixel data or cannot be decoded.

    Args:
        path: Absolute path to a .dcm file.

    Returns:
        A PIL.Image.Image in mode 'L' (8-bit grayscale), or None.
    """
    import pydicom
    from PIL import Image

    try:
        ds = pydicom.dcmread(path)
    except Exception as e:
        log.debug("Cannot read DICOM %s: %s", path, e)
        return None

    if not hasattr(ds, "PixelData"):
        return None

    try:
        arr = ds.pixel_array
    except Exception as e:
        log.debug("Cannot decode pixel_array from %s: %s", path, e)
        return None

    # Multi-frame: use first frame only (burned-in text is typically
    # consistent across frames).
    if arr.ndim == 3 and arr.shape[0] > 1 and arr.shape[2] != 3:
        arr = arr[0]

    arr = _apply_windowing(ds, arr)
    arr = _to_uint8(arr)
    return Image.fromarray(arr)


def _apply_windowing(ds, arr: np.ndarray) -> np.ndarray:
    """Apply DICOM window center/width for better contrast."""
    wc = getattr(ds, "WindowCenter", None)
    ww = getattr(ds, "WindowWidth", None)
    if wc is None or ww is None:
        return arr

    # Handle multi-value window center/width (take first).
    if hasattr(wc, "__iter__"):
        wc = wc[0] if len(wc) > 0 else wc
    if hasattr(ww, "__iter__"):
        ww = ww[0] if len(ww) > 0 else ww

    wc, ww = float(wc), float(ww)
    if ww <= 0:
        return arr

    low = wc - ww / 2
    high = wc + ww / 2
    arr = arr.astype(np.float64)
    arr = np.clip(arr, low, high)
    return arr


def _to_uint8(arr: np.ndarray) -> np.ndarray:
    """Normalise array to 0-255 uint8."""
    arr = arr.astype(np.float64)
    mn, mx = arr.min(), arr.max()
    if mx - mn == 0:
        return np.zeros(arr.shape[:2], dtype=np.uint8)
    return ((arr - mn) / (mx - mn) * 255.0).astype(np.uint8)
