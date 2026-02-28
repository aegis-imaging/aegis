"""TotalSegmentator backend — whole-body CT/MRI segmentation.

TotalSegmentator uses nnU-Net to segment 117 anatomical structures from CT
and MRI scans. It serves as the server-side replacement for 3D Slicer's
SlicerTotalSegmentator extension (same underlying models, no GUI needed).

Requires:
  - ``TotalSegmentator`` pip package (Apache 2.0)
  - PyTorch with CUDA support (recommended) or CPU-only
  - Models auto-downloaded on first run (~1.6 GB)

References:
  Wasserthal et al. "TotalSegmentator: Robust Segmentation of 104 Anatomic
  Structures in CT Images." Radiology: AI, 2023.
  DOI: 10.1148/ryai.230024
"""

from __future__ import annotations

import json
import logging
import os
import time

import nibabel as nib
import numpy as np

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import find_any_nifti

log = logging.getLogger(__name__)

# Subset of TotalSegmentator brain-related structures (v2)
_BRAIN_STRUCTURES = {
    "brain", "skull", "eyes", "optic_chiasm",
    "brainstem", "cerebellum",
}

# Full label list is loaded dynamically from TotalSegmentator's map_to_binary.json
# or inferred from the output segmentation mask.


class TotalSegmentatorBackend(AnalyticsBackend):
    """Whole-body CT/MRI segmentation via TotalSegmentator."""

    @property
    def name(self) -> str:
        return "totalsegmentator"

    def available(self) -> bool:
        try:
            import totalsegmentator  # noqa: F401
            return True
        except ImportError:
            pass
        # Also check CLI
        import shutil
        return shutil.which("TotalSegmentator") is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "totalsegmentator")
        os.makedirs(out_dir, exist_ok=True)

        # Find any NIfTI (TotalSegmentator handles CT and MRI)
        nifti_path = find_any_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("TotalSegmentator: using %s", nifti_path)

        # Configuration
        task = getattr(config, "TOTALSEG_TASK", "total")
        fast = getattr(config, "TOTALSEG_FAST", False)
        use_gpu = getattr(config, "TOTALSEG_USE_GPU", True)

        # Run TotalSegmentator
        seg_path = os.path.join(out_dir, "segmentation.nii.gz")
        try:
            label_map = _run_totalsegmentator(
                nifti_path, out_dir, seg_path, task, fast, use_gpu,
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"TotalSegmentator failed: {e}",
            )

        if not os.path.isfile(seg_path):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="TotalSegmentator did not produce output segmentation",
            )

        # Compute per-label volumes from the combined segmentation
        roi_volumes = _compute_volumes(seg_path, label_map)

        # Compute per-label intensity stats
        roi_stats = _compute_stats(seg_path, nifti_path, label_map)

        # Build metrics
        metrics: dict = {
            "atlas": "totalsegmentator",
            "task": task,
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
            "roi_stats": roi_stats,
        }

        # Write summary JSON
        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, summary_path]

        log.info(
            "TotalSegmentator complete for %s: %d ROIs in %.1fs (task=%s)",
            study_uid,
            len(roi_volumes),
            duration,
            task,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _run_totalsegmentator(
    input_path: str,
    output_dir: str,
    combined_output: str,
    task: str,
    fast: bool,
    use_gpu: bool,
) -> dict[int, str]:
    """Run TotalSegmentator and return the label map.

    Tries the Python API first, falls back to CLI.
    Returns a label_id → name mapping extracted from TotalSegmentator.
    """
    label_map: dict[int, str] = {}

    try:
        label_map = _run_python_api(input_path, output_dir, combined_output, task, fast, use_gpu)
    except ImportError:
        log.info("TotalSegmentator Python API not available, trying CLI")
        _run_cli(input_path, output_dir, combined_output, task, fast)
        label_map = _infer_label_map_from_output(combined_output)

    return label_map


def _run_python_api(
    input_path: str,
    output_dir: str,
    combined_output: str,
    task: str,
    fast: bool,
    use_gpu: bool,
) -> dict[int, str]:
    """Run via TotalSegmentator Python API."""
    from totalsegmentator.python_api import totalsegmentator

    # Run segmentation — outputs individual structure NIfTIs + combined
    totalsegmentator(
        input=input_path,
        output=output_dir,
        task=task,
        fast=fast,
        device="gpu" if use_gpu else "cpu",
        ml=True,  # Output multilabel (combined) segmentation
    )

    # The combined output is typically at output_dir/total.nii.gz or similar
    _find_and_rename_combined(output_dir, combined_output)

    # Try to load label map from TotalSegmentator
    return _load_totalseg_label_map(task)


def _run_cli(
    input_path: str,
    output_dir: str,
    combined_output: str,
    task: str,
    fast: bool,
) -> None:
    """Run via TotalSegmentator CLI."""
    import subprocess

    cmd = [
        "TotalSegmentator",
        "-i", input_path,
        "-o", output_dir,
        "--task", task,
        "--ml",  # multilabel output
    ]
    if fast:
        cmd.append("--fast")

    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=config.ANALYTICS_TIMEOUT,
    )

    if result.returncode != 0:
        raise RuntimeError(
            result.stderr[-500:] if result.stderr else "TotalSegmentator CLI failed"
        )

    _find_and_rename_combined(output_dir, combined_output)


def _find_and_rename_combined(output_dir: str, target_path: str) -> None:
    """Find the combined multilabel NIfTI and move it to the target path."""
    import shutil
    from glob import glob

    # TotalSegmentator outputs the combined file with various names
    candidates = glob(os.path.join(output_dir, "*.nii.gz"))
    # Prefer files named 'total*' or containing 'multilabel'
    for pattern in ["total", "multilabel", "combined"]:
        for c in candidates:
            if pattern in os.path.basename(c).lower():
                if c != target_path:
                    shutil.copy2(c, target_path)
                return

    # Fall back to the largest NIfTI
    if candidates:
        largest = max(candidates, key=os.path.getsize)
        if largest != target_path:
            shutil.copy2(largest, target_path)


def _load_totalseg_label_map(task: str) -> dict[int, str]:
    """Load label map from TotalSegmentator's bundled data."""
    try:
        from totalsegmentator.map_to_binary import class_map
        if task in class_map:
            return {i + 1: name for i, name in enumerate(class_map[task])}
        # Fall back to "total" task
        if "total" in class_map:
            return {i + 1: name for i, name in enumerate(class_map["total"])}
    except (ImportError, AttributeError):
        pass

    # Try alternative import path
    try:
        from totalsegmentator.config import class_map as cm
        if task in cm:
            return {i + 1: name for i, name in enumerate(cm[task])}
    except (ImportError, AttributeError):
        pass

    log.warning("Could not load TotalSegmentator label map, using numeric labels")
    return {}


def _infer_label_map_from_output(seg_path: str) -> dict[int, str]:
    """Infer label names from the segmentation output when no map is available."""
    if not os.path.isfile(seg_path):
        return {}

    img = nib.load(seg_path)
    data = np.asarray(img.dataobj, dtype=np.int32)
    unique_labels = np.unique(data)

    return {
        int(label_id): f"structure_{label_id}"
        for label_id in unique_labels
        if label_id != 0
    }


def _compute_volumes(seg_path: str, label_map: dict[int, str]) -> dict[str, float]:
    """Compute per-label volumes from a segmentation NIfTI."""
    if not label_map:
        label_map = _infer_label_map_from_output(seg_path)

    img = nib.load(seg_path)
    data = np.asarray(img.dataobj, dtype=np.int32)
    voxel_dims = img.header.get_zooms()[:3]
    voxel_vol = float(np.prod(voxel_dims))

    volumes: dict[str, float] = {}
    for label_id in np.unique(data):
        if label_id == 0:
            continue
        name = label_map.get(int(label_id), f"structure_{label_id}")
        count = int(np.sum(data == label_id))
        volumes[name] = round(count * voxel_vol, 2)

    return volumes


def _compute_stats(
    seg_path: str,
    intensity_path: str,
    label_map: dict[int, str],
) -> list[dict]:
    """Compute per-label intensity statistics."""
    if not label_map:
        label_map = _infer_label_map_from_output(seg_path)

    seg_img = nib.load(seg_path)
    int_img = nib.load(intensity_path)

    seg_data = np.asarray(seg_img.dataobj, dtype=np.int32)
    int_data = np.asarray(int_img.dataobj, dtype=np.float64)

    voxel_dims = int_img.header.get_zooms()[:3]
    voxel_vol = float(np.prod(voxel_dims))

    stats: list[dict] = []
    for label_id in np.unique(seg_data):
        if label_id == 0:
            continue
        name = label_map.get(int(label_id), f"structure_{label_id}")
        mask = seg_data == label_id
        voxels = int_data[mask]
        if voxels.size == 0:
            continue

        stats.append({
            "roi_name": name,
            "voxel_count": int(voxels.size),
            "volume_mm3": round(voxels.size * voxel_vol, 2),
            "mean": round(float(np.mean(voxels)), 6),
            "median": round(float(np.median(voxels)), 6),
            "std": round(float(np.std(voxels)), 6),
        })

    stats.sort(key=lambda r: r["roi_name"])
    return stats
