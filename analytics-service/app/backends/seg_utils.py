"""Shared segmentation utilities for backends that produce integer-label masks.

Used by SynthSeg, nnU-Net, TotalSegmentator, TotalSpineSeg, SPINEPS, and
MedSAM2 backends.
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


def find_t2w_nifti(bids_dir: str) -> str | None:
    """Find a T2w NIfTI in a BIDS directory."""
    candidates = glob(os.path.join(bids_dir, "**", "*T2w*.nii*"), recursive=True)
    return sorted(candidates)[0] if candidates else None


def find_spine_nifti(bids_dir: str) -> str | None:
    """Find a spine NIfTI in a BIDS directory.

    Priority order: files with "spine"/"spinal" in name → T2w → T1w → any NIfTI.
    T2w sagittal is the most common contrast for spine MRI analysis.
    """
    all_niis = sorted(glob(os.path.join(bids_dir, "**", "*.nii*"), recursive=True))
    if not all_niis:
        return None

    # Prefer files explicitly named for spine
    for path in all_niis:
        base = os.path.basename(path).lower()
        if "spine" in base or "spinal" in base:
            return path

    # Fall back to T2w (most useful for spine analysis)
    for path in all_niis:
        if "T2w" in os.path.basename(path):
            return path

    # Fall back to T1w
    for path in all_niis:
        if "T1w" in os.path.basename(path):
            return path

    # Last resort: any NIfTI
    return all_niis[0]


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


# ── Spine label maps ──────────────────────────────────────────────────────

# TotalSpineSeg label map (NeuroPoly, LGPL-3.0)
# https://github.com/neuropoly/totalspineseg
# Labels verified from the tool's output specification.
TOTALSPINESEG_LABELS: dict[int, str] = {
    # Vertebrae (1-25)
    1: "C1", 2: "C2", 3: "C3", 4: "C4", 5: "C5", 6: "C6", 7: "C7",
    8: "T1", 9: "T2", 10: "T3", 11: "T4", 12: "T5", 13: "T6",
    14: "T7", 15: "T8", 16: "T9", 17: "T10", 18: "T11", 19: "T12",
    20: "L1", 21: "L2", 22: "L3", 23: "L4", 24: "L5",
    25: "sacrum",
    # Intervertebral discs (40-62)
    40: "IVD_C2-C3", 41: "IVD_C3-C4", 42: "IVD_C4-C5",
    43: "IVD_C5-C6", 44: "IVD_C6-C7", 45: "IVD_C7-T1",
    46: "IVD_T1-T2", 47: "IVD_T2-T3", 48: "IVD_T3-T4",
    49: "IVD_T4-T5", 50: "IVD_T5-T6", 51: "IVD_T6-T7",
    52: "IVD_T7-T8", 53: "IVD_T8-T9", 54: "IVD_T9-T10",
    55: "IVD_T10-T11", 56: "IVD_T11-T12",
    57: "IVD_T12-L1", 58: "IVD_L1-L2", 59: "IVD_L2-L3",
    60: "IVD_L3-L4", 61: "IVD_L4-L5", 62: "IVD_L5-S1",
    # Spinal cord and canal (200-201)
    200: "spinal_cord",
    201: "spinal_canal",
}

# SPINEPS semantic label map (14 classes, Apache 2.0)
# Greve et al. "SPINEPS — Automatic Whole Spine Segmentation of T2-weighted
# MR images using a Two-Phase Approach." European Radiology, 2025.
# DOI: 10.1007/s00330-024-11155-y
SPINEPS_SEMANTIC_LABELS: dict[int, str] = {
    1: "vertebral_corpus",
    2: "vertebral_arch",
    3: "spinous_process",
    4: "transverse_process_left",
    5: "transverse_process_right",
    6: "costovertebral_joint_left",
    7: "costovertebral_joint_right",
    8: "superior_articular_facet",
    9: "inferior_articular_facet",
    10: "superior_endplate",
    11: "inferior_endplate",
    12: "intervertebral_disc",
    13: "spinal_cord",
    14: "spinal_canal",
}
