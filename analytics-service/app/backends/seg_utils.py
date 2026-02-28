"""Shared segmentation utilities for backends that produce integer-label masks.

Used by SynthSeg, nnU-Net, and TotalSegmentator backends.
"""

from __future__ import annotations

import logging
import os
from glob import glob

import nibabel as nib
import numpy as np

log = logging.getLogger(__name__)

# Preferred NIfTI suffix order for contrast-agnostic backends
_CONTRAST_PRIORITY = ["T1w", "T2w", "FLAIR", "PD", "bold", "dwi"]


def find_any_nifti(bids_dir: str) -> str | None:
    """Find any NIfTI in a BIDS directory, preferring T1w > T2w > FLAIR > etc.

    Falls back to the first NIfTI found if no known contrast suffix matches.
    """
    all_niis = sorted(glob(os.path.join(bids_dir, "**", "*.nii*"), recursive=True))
    if not all_niis:
        return None

    for suffix in _CONTRAST_PRIORITY:
        for path in all_niis:
            if suffix in os.path.basename(path):
                return path

    return all_niis[0]


def find_t1w_nifti(bids_dir: str) -> str | None:
    """Find a T1w NIfTI in a BIDS directory."""
    candidates = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
    return candidates[0] if candidates else None


def find_pet_nifti(bids_dir: str) -> str | None:
    """Find a PET NIfTI in a BIDS directory.

    Searches for ``*pet*`` filename patterns and the BIDS ``pet/`` subdirectory.
    """
    for pattern in ["*pet*.nii*", "*PET*.nii*"]:
        candidates = sorted(glob(os.path.join(bids_dir, "**", pattern), recursive=True))
        if candidates:
            return candidates[0]
    # Fall back to pet/ subdirectory in BIDS layout
    pet_dirs = sorted(glob(os.path.join(bids_dir, "**", "pet"), recursive=True))
    for pet_dir in pet_dirs:
        niis = sorted(glob(os.path.join(pet_dir, "*.nii*")))
        if niis:
            return niis[0]
    return None


def find_asl_nifti(bids_dir: str) -> str | None:
    """Find an ASL/perfusion NIfTI in a BIDS directory.

    Searches for ``*asl*`` or ``*perf*`` filename patterns and the BIDS
    ``perf/`` subdirectory.
    """
    for pattern in ["*asl*.nii*", "*ASL*.nii*", "*perf*.nii*"]:
        candidates = sorted(glob(os.path.join(bids_dir, "**", pattern), recursive=True))
        if candidates:
            return candidates[0]
    # Fall back to perf/ subdirectory
    perf_dirs = sorted(glob(os.path.join(bids_dir, "**", "perf"), recursive=True))
    for perf_dir in perf_dirs:
        niis = sorted(glob(os.path.join(perf_dir, "*.nii*")))
        if niis:
            return niis[0]
    return None


def find_qsm_niftis(bids_dir: str) -> tuple[str | None, str | None]:
    """Find magnitude and phase NIfTI pair for QSM analysis.

    Returns ``(magnitude_path, phase_path)``.  Either may be ``None``.
    """
    all_niis = sorted(glob(os.path.join(bids_dir, "**", "*.nii*"), recursive=True))
    magnitude: str | None = None
    phase: str | None = None
    for path in all_niis:
        base = os.path.basename(path).lower()
        if "phase" in base or "phasediff" in base:
            if phase is None:
                phase = path
        elif "magnitude" in base or "mag" in base:
            if magnitude is None:
                magnitude = path
    # Fall back: if echo files exist, use first two
    if magnitude is None and phase is None:
        echo_files = [p for p in all_niis if "echo" in os.path.basename(p).lower()]
        if len(echo_files) >= 2:
            magnitude = echo_files[0]
            phase = echo_files[1]
    return magnitude, phase


def compute_label_volumes(
    seg_path: str,
    label_map: dict[int, str],
) -> dict[str, float]:
    """Compute per-label volumes (mm³) from an integer-label segmentation NIfTI.

    Args:
        seg_path: Path to segmentation NIfTI with integer labels.
        label_map: Label ID → name mapping.

    Returns:
        Dict of {roi_name: volume_mm3}.
    """
    img = nib.load(seg_path)
    data = np.asarray(img.dataobj, dtype=np.int32)
    voxel_dims = img.header.get_zooms()[:3]
    voxel_vol = float(np.prod(voxel_dims))

    volumes: dict[str, float] = {}
    for label_id in np.unique(data):
        if label_id == 0:
            continue
        if label_id not in label_map:
            continue
        count = int(np.sum(data == label_id))
        volumes[label_map[label_id]] = round(count * voxel_vol, 2)

    return volumes


def compute_label_stats(
    seg_path: str,
    intensity_path: str,
    label_map: dict[int, str],
) -> list[dict]:
    """Compute per-label intensity statistics from a segmentation + data image pair.

    Args:
        seg_path: Path to segmentation NIfTI with integer labels.
        intensity_path: Path to intensity image (e.g. T1w) in the same space.
        label_map: Label ID → name mapping.

    Returns:
        List of dicts with roi_name, mean, median, std, voxel_count, volume_mm3.
    """
    seg_img = nib.load(seg_path)
    data_img = nib.load(intensity_path)

    seg_data = np.asarray(seg_img.dataobj, dtype=np.int32)
    int_data = np.asarray(data_img.dataobj, dtype=np.float64)

    voxel_dims = data_img.header.get_zooms()[:3]
    voxel_vol = float(np.prod(voxel_dims))

    stats: list[dict] = []
    for label_id in np.unique(seg_data):
        if label_id == 0:
            continue
        if label_id not in label_map:
            continue

        mask = seg_data == label_id
        voxels = int_data[mask]
        if voxels.size == 0:
            continue

        stats.append({
            "roi_name": label_map[label_id],
            "voxel_count": int(voxels.size),
            "volume_mm3": round(voxels.size * voxel_vol, 2),
            "mean": round(float(np.mean(voxels)), 6),
            "median": round(float(np.median(voxels)), 6),
            "std": round(float(np.std(voxels)), 6),
        })

    stats.sort(key=lambda r: r["roi_name"])
    return stats
