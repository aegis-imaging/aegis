"""SPINEPS backend — automatic whole-spine MRI segmentation.

SPINEPS provides both semantic segmentation (14 vertebral subregion classes)
and instance segmentation (individually labeled vertebrae) from T2w or T1w
sagittal MRI.  It is the only tool that segments vertebral substructures
(body, arch, processes, facet joints, endplates).

Requires:
  - ``spineps`` pip package (Apache 2.0)
  - PyTorch with CUDA support (recommended) or CPU-only
  - Models auto-downloaded on first run

References:
  Greve H et al. "SPINEPS -- Automatic Whole Spine Segmentation of
  T2-weighted MR images using a Two-Phase Approach to Multi-class Semantic
  and Instance Segmentation." European Radiology, 2025.
  DOI: 10.1007/s00330-024-11155-y
"""

from __future__ import annotations

import json
import logging
import os
import time
from glob import glob

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import (
    SPINEPS_SEMANTIC_LABELS,
    compute_label_stats,
    compute_label_volumes,
    find_spine_nifti,
)

log = logging.getLogger(__name__)


class SPINEPSBackend(AnalyticsBackend):
    """Whole-spine MRI segmentation via SPINEPS (semantic + instance)."""

    @property
    def name(self) -> str:
        return "spineps"

    def available(self) -> bool:
        try:
            from spineps.seg_run import process_img_nii  # noqa: F401
            return True
        except ImportError:
            return False

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "spineps")
        os.makedirs(out_dir, exist_ok=True)

        # Find input NIfTI — SPINEPS prefers T2w sagittal
        nifti_path = find_spine_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("SPINEPS: using %s", nifti_path)

        model_name = getattr(config, "SPINEPS_MODEL", "t2w")

        # Run SPINEPS via Python API
        try:
            from spineps.seg_model import get_segmentation_model
            from spineps.seg_run import process_img_nii

            model_semantic = get_segmentation_model(model_name)
            model_instance = get_segmentation_model("instance")

            process_img_nii(
                img_path=nifti_path,
                output_dir=out_dir,
                model_semantic=model_semantic,
                model_instance=model_instance,
            )
        except ImportError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="spineps package not installed",
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"SPINEPS failed: {e}",
            )

        # Locate output segmentations
        semantic_path = _find_seg_file(out_dir, "seg-spine")
        instance_path = _find_seg_file(out_dir, "seg-vert")

        if not semantic_path and not instance_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="SPINEPS did not produce output segmentation",
            )

        # Compute semantic volumes (14 subregion classes)
        semantic_volumes: dict[str, float] = {}
        semantic_stats: list[dict] = []
        if semantic_path:
            semantic_volumes = compute_label_volumes(
                semantic_path, SPINEPS_SEMANTIC_LABELS,
            )
            semantic_stats = compute_label_stats(
                semantic_path, nifti_path, SPINEPS_SEMANTIC_LABELS,
            )

        # Compute instance vertebrae volumes (auto-discovered labels)
        instance_vertebrae: dict[str, float] = {}
        instance_stats: list[dict] = []
        if instance_path:
            instance_label_map = _infer_instance_labels(instance_path)
            instance_vertebrae = compute_label_volumes(
                instance_path, instance_label_map,
            )
            instance_stats = compute_label_stats(
                instance_path, nifti_path, instance_label_map,
            )

        # Parse centroids JSON if present
        centroids = _load_centroids(out_dir)

        metrics: dict = {
            "atlas": "spineps",
            "model": model_name,
            "roi_count": len(semantic_volumes) + len(instance_vertebrae),
            "semantic_volumes": semantic_volumes,
            "instance_vertebrae": instance_vertebrae,
            "centroids": centroids,
            "semantic_stats": semantic_stats,
            "instance_stats": instance_stats,
        }

        # Write summary JSON
        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [p for p in [semantic_path, instance_path, summary_path] if p]

        log.info(
            "SPINEPS complete for %s: %d semantic + %d instance ROIs in %.1fs",
            study_uid,
            len(semantic_volumes),
            len(instance_vertebrae),
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _find_seg_file(out_dir: str, prefix: str) -> str | None:
    """Find a segmentation NIfTI by filename prefix in the output directory."""
    candidates = sorted(glob(os.path.join(out_dir, "**", f"*{prefix}*.nii*"), recursive=True))
    return candidates[0] if candidates else None


def _infer_instance_labels(seg_path: str) -> dict[int, str]:
    """Infer vertebra names from SPINEPS instance label IDs.

    SPINEPS uses a convention where instance labels encode the vertebral
    level (e.g. 1=C1, 2=C2, ..., 8=T1, ..., 20=L1, ..., 25=sacrum).
    """
    import nibabel as nib
    import numpy as np

    img = nib.load(seg_path)
    data = np.asarray(img.dataobj, dtype=np.int32)
    unique = sorted(int(x) for x in np.unique(data) if x != 0)

    # Map label IDs to vertebra names using the same scheme as TotalSpineSeg
    _VERT_NAMES = {
        1: "C1", 2: "C2", 3: "C3", 4: "C4", 5: "C5", 6: "C6", 7: "C7",
        8: "T1", 9: "T2", 10: "T3", 11: "T4", 12: "T5", 13: "T6",
        14: "T7", 15: "T8", 16: "T9", 17: "T10", 18: "T11", 19: "T12",
        20: "L1", 21: "L2", 22: "L3", 23: "L4", 24: "L5",
        25: "sacrum",
    }

    return {
        label_id: _VERT_NAMES.get(label_id, f"vertebra_{label_id}")
        for label_id in unique
    }


def _load_centroids(out_dir: str) -> list[dict]:
    """Load vertebral centroid coordinates from SPINEPS JSON output."""
    json_files = sorted(glob(os.path.join(out_dir, "**", "*centroid*.json"), recursive=True))
    if not json_files:
        return []

    try:
        with open(json_files[0], encoding="utf-8") as f:
            data = json.load(f)
        if isinstance(data, list):
            return data
        if isinstance(data, dict):
            return list(data.values()) if data else []
    except (json.JSONDecodeError, OSError):
        log.warning("Failed to parse SPINEPS centroids JSON")

    return []
