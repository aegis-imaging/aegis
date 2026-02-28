"""Shared atlas utilities for ROI labeling and statistics extraction.

Used by both AtlasROIBackend (single-study) and TBMSyNBackend (longitudinal).
"""

from __future__ import annotations

import csv
import logging
import os
import subprocess
from importlib import resources

import nibabel as nib
import numpy as np

from app import config

log = logging.getLogger(__name__)


def load_atlas_labels(atlas_name: str) -> dict[int, str]:
    """Load ROI ID → name mapping from a bundled CSV file.

    Looks for the CSV in the ``app.atlases`` package first (bundled data),
    then falls back to ``{ATLAS_DIR}/{atlas_name}/{atlas_name}_labels.csv``.
    """
    # Try bundled data first
    try:
        ref = resources.files("app.atlases").joinpath(f"{atlas_name}_labels.csv")
        text = ref.read_text(encoding="utf-8")
        reader = csv.DictReader(text.splitlines())
        return {int(row["roi_id"]): row["roi_name"] for row in reader}
    except (FileNotFoundError, KeyError, TypeError):
        pass

    # Fall back to filesystem
    csv_path = os.path.join(config.ATLAS_DIR, atlas_name, f"{atlas_name}_labels.csv")
    if not os.path.isfile(csv_path):
        raise FileNotFoundError(f"Atlas labels not found: {csv_path}")

    labels: dict[int, str] = {}
    with open(csv_path, encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            labels[int(row["roi_id"])] = row["roi_name"]
    return labels


def find_atlas_nifti(atlas_name: str) -> str:
    """Locate the atlas NIfTI file on disk.

    Looks in ``{ATLAS_DIR}/{atlas_name}/`` for a file named
    ``{atlas_name}*.nii*`` (e.g. ``aal3v1_1mm.nii.gz``).
    """
    atlas_dir = os.path.join(config.ATLAS_DIR, atlas_name)
    if not os.path.isdir(atlas_dir):
        raise FileNotFoundError(f"Atlas directory not found: {atlas_dir}")

    from glob import glob as _glob

    candidates = _glob(os.path.join(atlas_dir, f"{atlas_name}*.nii*"))
    if not candidates:
        # Try any NIfTI in the directory
        candidates = _glob(os.path.join(atlas_dir, "*.nii*"))

    if not candidates:
        raise FileNotFoundError(f"No NIfTI atlas file in {atlas_dir}")

    return candidates[0]


def get_ants_env() -> dict[str, str]:
    """Build environment dict with ANTs paths configured."""
    env = os.environ.copy()
    env["ANTSPATH"] = config.ANTSPATH
    env["PATH"] = config.ANTSPATH + ":" + env.get("PATH", "")
    return env


def run_n4_bias_correction(
    input_path: str,
    output_path: str,
    env: dict[str, str] | None = None,
    timeout: int = 3600,
) -> tuple[bool, str]:
    """Run N4 bias field correction on a NIfTI image.

    Returns (success, error_message).
    """
    if env is None:
        env = get_ants_env()

    try:
        result = subprocess.run(
            ["N4BiasFieldCorrection", "-d", "3", "-i", input_path, "-o", output_path],
            capture_output=True,
            text=True,
            timeout=timeout,
            env=env,
        )
        if result.returncode != 0:
            return False, result.stderr[-500:] if result.stderr else "N4 failed"
        return True, ""
    except FileNotFoundError:
        return False, "N4BiasFieldCorrection not found"
    except subprocess.TimeoutExpired:
        return False, f"N4BiasFieldCorrection timed out after {timeout}s"


def run_brain_extraction(
    input_path: str,
    output_prefix: str,
    template_dir: str | None = None,
    env: dict[str, str] | None = None,
    timeout: int = 7200,
) -> tuple[bool, str, str]:
    """Run brain extraction using antsBrainExtraction.sh.

    Returns (success, brain_path, error_message).
    The extracted brain is at ``{output_prefix}BrainExtractionBrain.nii.gz``.
    """
    if env is None:
        env = get_ants_env()
    if template_dir is None:
        template_dir = config.ANTS_TEMPLATE_DIR

    brain_template = os.path.join(template_dir, "T_template0.nii.gz")
    brain_prior = os.path.join(
        template_dir, "T_template0_BrainCerebellumProbabilityMask.nii.gz"
    )

    if not os.path.isfile(brain_template) or not os.path.isfile(brain_prior):
        return False, "", f"Brain extraction templates not found in {template_dir}"

    try:
        result = subprocess.run(
            [
                "antsBrainExtraction.sh",
                "-d", "3",
                "-a", input_path,
                "-e", brain_template,
                "-m", brain_prior,
                "-o", output_prefix,
            ],
            capture_output=True,
            text=True,
            timeout=timeout,
            env=env,
        )
    except FileNotFoundError:
        return False, "", "antsBrainExtraction.sh not found"
    except subprocess.TimeoutExpired:
        return False, "", f"Brain extraction timed out after {timeout}s"

    if result.returncode != 0:
        return False, "", result.stderr[-500:] if result.stderr else "Brain extraction failed"

    brain_path = f"{output_prefix}BrainExtractionBrain.nii.gz"
    if not os.path.isfile(brain_path):
        return False, "", f"Expected brain output not found: {brain_path}"

    return True, brain_path, ""


def warp_atlas_to_subject(
    atlas_path: str,
    template_brain: str,
    subject_brain: str,
    output_path: str,
    env: dict[str, str] | None = None,
    timeout: int = 7200,
) -> tuple[bool, str]:
    """Register template → subject via ANTs SyN and warp atlas labels.

    Steps:
      1. ``antsRegistrationSyN.sh`` to register template to subject space
      2. ``antsApplyTransforms`` to warp atlas labels using NearestNeighbor

    Returns (success, error_message).
    """
    if env is None:
        env = get_ants_env()

    work_dir = os.path.dirname(output_path)
    prefix = os.path.join(work_dir, "atlas_reg_")

    # Step 1: Register template → subject
    try:
        result = subprocess.run(
            [
                "antsRegistrationSyN.sh",
                "-d", "3",
                "-f", subject_brain,
                "-m", template_brain,
                "-t", "s",
                "-o", prefix,
            ],
            capture_output=True,
            text=True,
            timeout=timeout,
            env=env,
        )
    except FileNotFoundError:
        return False, "antsRegistrationSyN.sh not found"
    except subprocess.TimeoutExpired:
        return False, f"Registration timed out after {timeout}s"

    if result.returncode != 0:
        return False, result.stderr[-500:] if result.stderr else "Registration failed"

    warp = f"{prefix}1Warp.nii.gz"
    affine = f"{prefix}0GenericAffine.mat"

    if not os.path.isfile(warp) or not os.path.isfile(affine):
        return False, "Registration outputs not found"

    # Step 2: Apply transforms to atlas labels (NearestNeighbor for discrete labels)
    try:
        result = subprocess.run(
            [
                "antsApplyTransforms",
                "-d", "3",
                "-i", atlas_path,
                "-r", subject_brain,
                "-t", warp, affine,
                "-n", "NearestNeighbor",
                "-o", output_path,
            ],
            capture_output=True,
            text=True,
            timeout=600,
            env=env,
        )
    except FileNotFoundError:
        return False, "antsApplyTransforms not found"
    except subprocess.TimeoutExpired:
        return False, "Atlas warp timed out"

    if result.returncode != 0:
        return False, result.stderr[-500:] if result.stderr else "Atlas warp failed"

    if not os.path.isfile(output_path):
        return False, f"Warped atlas not found: {output_path}"

    return True, ""


def extract_roi_stats(
    warped_atlas_path: str,
    data_img_path: str,
    label_map: dict[int, str],
) -> list[dict]:
    """Extract per-ROI statistics from a data image using warped atlas labels.

    For each ROI label present in both the atlas and ``label_map``, computes:
    voxel_count, volume_mm3, mean, median, std.

    Args:
        warped_atlas_path: NIfTI with integer ROI labels in subject space.
        data_img_path: NIfTI data image (e.g. T1w, Jacobian, PET) in same space.
        label_map: ROI ID → name mapping.

    Returns:
        List of dicts with per-ROI statistics.
    """
    atlas_img = nib.load(warped_atlas_path)
    data_img = nib.load(data_img_path)

    atlas_data = np.asarray(atlas_img.dataobj, dtype=np.int32)
    img_data = np.asarray(data_img.dataobj, dtype=np.float64)

    # Compute voxel volume from the data image affine
    voxel_dims = data_img.header.get_zooms()[:3]
    voxel_vol_mm3 = float(np.prod(voxel_dims))

    results = []
    unique_labels = np.unique(atlas_data)

    for roi_id in unique_labels:
        if roi_id == 0:
            continue  # skip background
        if roi_id not in label_map:
            continue

        mask = atlas_data == roi_id
        roi_voxels = img_data[mask]

        if roi_voxels.size == 0:
            continue

        results.append({
            "roi_id": int(roi_id),
            "roi_name": label_map[roi_id],
            "voxel_count": int(roi_voxels.size),
            "volume_mm3": round(roi_voxels.size * voxel_vol_mm3, 2),
            "mean": round(float(np.mean(roi_voxels)), 6),
            "median": round(float(np.median(roi_voxels)), 6),
            "std": round(float(np.std(roi_voxels)), 6),
        })

    results.sort(key=lambda r: r["roi_id"])
    return results
