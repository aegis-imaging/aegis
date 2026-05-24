"""Pixel PHI redaction utility.

Blacks out detected text regions in DICOM pixel data by filling bounding
box areas with zero (black). Used by the /redact endpoint after detection.
"""

import logging
import shutil
from pathlib import Path

import numpy as np

log = logging.getLogger(__name__)


def redact_dicom_pixels(
    input_path: str,
    output_path: str,
    regions: list[dict],
    padding_px: int = 5,
) -> bool:
    """Black out detected text regions in DICOM pixel data.

    Args:
        input_path: Path to the source DICOM file.
        output_path: Path to write the redacted DICOM file.
        regions: List of region dicts with 'bbox' key ([x, y, w, h]).
        padding_px: Pixels of padding around each bbox.

    Returns:
        True if any redaction was applied, False if file was copied unchanged.
    """
    import pydicom

    if not regions:
        shutil.copy2(input_path, output_path)
        return False

    try:
        ds = pydicom.dcmread(input_path)
    except Exception as e:
        log.error("Cannot read DICOM %s: %s", input_path, e)
        raise

    if not hasattr(ds, "PixelData"):
        shutil.copy2(input_path, output_path)
        return False

    try:
        arr = ds.pixel_array.copy()
    except Exception as e:
        log.error("Cannot decode pixel_array from %s: %s", input_path, e)
        raise

    original_dtype = arr.dtype
    is_multi_frame = arr.ndim >= 3 and arr.shape[0] > 1 and (arr.ndim < 4 or arr.shape[-1] != 3)

    # Determine fill value based on photometric interpretation
    photometric = getattr(ds, "PhotometricInterpretation", "MONOCHROME2")
    if photometric == "MONOCHROME1":
        # MONOCHROME1: higher values = darker, so fill with max to black out
        fill_value = np.iinfo(original_dtype).max if np.issubdtype(original_dtype, np.integer) else 0
    else:
        fill_value = 0

    redacted = False
    for region in regions:
        bbox = region.get("bbox", [0, 0, 0, 0])
        if len(bbox) != 4:
            continue
        x, y, w, h = bbox
        if w <= 0 or h <= 0:
            continue

        # Apply padding and clamp to image bounds
        if is_multi_frame:
            frame_h, frame_w = arr.shape[1], arr.shape[2]
        elif arr.ndim == 3 and arr.shape[2] == 3:
            # RGB image
            frame_h, frame_w = arr.shape[0], arr.shape[1]
        else:
            frame_h, frame_w = arr.shape[0], arr.shape[1]

        x0 = max(0, x - padding_px)
        y0 = max(0, y - padding_px)
        x1 = min(frame_w, x + w + padding_px)
        y1 = min(frame_h, y + h + padding_px)

        if is_multi_frame:
            arr[:, y0:y1, x0:x1] = fill_value
        else:
            arr[y0:y1, x0:x1] = fill_value

        redacted = True

    if not redacted:
        shutil.copy2(input_path, output_path)
        return False

    # Write modified pixels back
    ds.PixelData = arr.astype(original_dtype).tobytes()

    # Use uncompressed transfer syntax for modified data
    ds.file_meta.TransferSyntaxUID = "1.2.840.10008.1.2.1"  # Explicit VR Little Endian

    Path(output_path).parent.mkdir(parents=True, exist_ok=True)
    ds.save_as(output_path, write_like_original=False)

    log.info("Redacted %d regions in %s → %s", len(regions), input_path, output_path)
    return True
