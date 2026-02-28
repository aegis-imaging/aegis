"""
Defacing pipeline orchestration.

Groups DICOM files by SeriesInstanceUID, then runs the configured
defacing backend on each unique series separately.
"""

import logging
import tempfile
from pathlib import Path

import pydicom

from .backends.base import DefacingBackend

log = logging.getLogger(__name__)


def group_by_series(dicom_paths: list[str]) -> dict[str, list[str]]:
    """Group DICOM file paths by SeriesInstanceUID."""
    groups: dict[str, list[str]] = {}
    for path in dicom_paths:
        try:
            ds = pydicom.dcmread(path, stop_before_pixels=True)
            uid = getattr(ds, "SeriesInstanceUID", "unknown")
        except Exception as e:
            log.warning("Cannot read DICOM metadata from %s: %s", path, e)
            uid = "unknown"
        groups.setdefault(str(uid), []).append(path)
    return groups


def _is_mosaic(ds) -> bool:
    """Detect Siemens mosaic format from ImageType tag."""
    image_type = getattr(ds, "ImageType", [])
    if isinstance(image_type, (list, pydicom.multival.MultiValue)):
        return "MOSAIC" in [str(v).upper() for v in image_type]
    return False


def should_deface_series(dicom_paths: list[str]) -> bool:
    """
    Return True if this series needs defacing.

    Only 3D volumetric head/brain series are defaced. 2D localizers,
    secondary captures, structured reports, Enhanced (multi-frame) DICOM,
    and mosaic DICOM are skipped — dcm2niix handles mosaic unwrapping
    during BIDS conversion.

    Minimum slice count for a 3D volume is 20 (below this, likely a localizer).
    """
    if len(dicom_paths) < 20:
        log.info("Skipping series with %d slices (likely localizer)", len(dicom_paths))
        return False

    try:
        ds = pydicom.dcmread(dicom_paths[0], stop_before_pixels=True)
        sop_class = str(getattr(ds, "SOPClassUID", ""))
        # Skip secondary capture, structured reports, presentation states,
        # and Enhanced (multi-frame) SOP classes.
        skip_classes = {
            "1.2.840.10008.5.1.4.1.1.7",      # Secondary Capture
            "1.2.840.10008.5.1.4.1.1.88.22",   # Enhanced SR
            "1.2.840.10008.5.1.4.1.1.11.1",    # Grayscale Softcopy Presentation State
            "1.2.840.10008.5.1.4.1.1.4.1",     # Enhanced MR Image Storage
            "1.2.840.10008.5.1.4.1.1.2.1",     # Enhanced CT Image Storage
            "1.2.840.10008.5.1.4.1.1.128.1",   # Enhanced PET Image Storage
        }
        if sop_class in skip_classes:
            log.info("Skipping SOPClass %s (enhanced/unsupported)", sop_class)
            return False

        # Skip multi-frame DICOM (Enhanced DICOM with NumberOfFrames > 1)
        num_frames = getattr(ds, "NumberOfFrames", None)
        if num_frames is not None and int(num_frames) > 1:
            log.info("Skipping multi-frame DICOM (%d frames)", int(num_frames))
            return False

        # Skip Siemens mosaic DICOM — dcm2niix handles unwrapping
        if _is_mosaic(ds):
            log.info("Skipping mosaic DICOM (Siemens mosaic format)")
            return False
    except Exception:
        pass

    return True


def run_pipeline(
    dicom_paths: list[str],
    output_dir: str,
    backend: DefacingBackend,
) -> list[str]:
    """
    Run the defacing pipeline on a list of DICOM files.

    - Groups files by series
    - Skips non-3D or non-head series
    - Runs the backend on each qualifying series
    - Returns all output file paths (defaced + passthrough)
    """
    Path(output_dir).mkdir(parents=True, exist_ok=True)

    series_groups = group_by_series(dicom_paths)
    log.info("Found %d series across %d files", len(series_groups), len(dicom_paths))

    all_output_paths: list[str] = []

    for series_uid, paths in series_groups.items():
        # Write all series output directly into output_dir (no per-series subdir).
        # The API importer names files sequentially (0.dcm, 1.dcm, ...) across all
        # series, so there are no filename collisions at the study level, and the
        # DICOMweb WADO-RS handler expects files at dicom/{store}/{studyUID}/{n}.dcm.
        series_out = output_dir
        Path(series_out).mkdir(parents=True, exist_ok=True)

        if should_deface_series(paths):
            log.info("Defacing series %s (%d files) using %s", series_uid, len(paths), backend.name)
            try:
                with tempfile.TemporaryDirectory(prefix="aegis_series_") as series_tmp:
                    # Copy series files into a flat temp dir for the backend
                    import shutil
                    for p in paths:
                        shutil.copy2(p, series_tmp)
                    out_paths = backend.deface(series_tmp, series_out)
                    all_output_paths.extend(out_paths)
            except Exception as e:
                log.error("Defacing failed for series %s: %s — copying originals", series_uid, e)
                # Passthrough: copy originals so the study is not lost
                import shutil
                for p in paths:
                    dst = str(Path(series_out) / Path(p).name)
                    shutil.copy2(p, dst)
                    all_output_paths.append(dst)
        else:
            # Passthrough: copy non-3D or non-head series as-is
            import shutil
            log.info("Passing through series %s (%d files)", series_uid, len(paths))
            for p in paths:
                dst = str(Path(series_out) / Path(p).name)
                shutil.copy2(p, dst)
                all_output_paths.append(dst)

    return all_output_paths
