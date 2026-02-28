"""TotalSpineSeg backend — automatic spine MRI segmentation.

TotalSpineSeg provides instance segmentation of vertebrae, intervertebral
discs (IVDs), the spinal cord, and the spinal canal from MRI. It is robust
to various MRI contrasts (T1w, T2w, FLAIR) and uses cascaded nnU-Net models.

Requires:
  - ``totalspineseg`` pip package (LGPL-3.0)
  - PyTorch with CUDA support (recommended) or CPU-only
  - Models auto-downloaded on first run

References:
  Houde JC et al. TotalSpineSeg: automatic segmentation of the spine in
  MRI using cascaded nnU-Net models. NeuroPoly lab, ISMRM 2024.
  https://github.com/neuropoly/totalspineseg
"""

from __future__ import annotations

import json
import logging
import os
import shutil
import subprocess
import time
from glob import glob

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import (
    TOTALSPINESEG_LABELS,
    compute_label_stats,
    compute_label_volumes,
    find_any_nifti,
    find_spine_nifti,
)

log = logging.getLogger(__name__)


class TotalSpineSegBackend(AnalyticsBackend):
    """Spine MRI segmentation via TotalSpineSeg (cascaded nnU-Net)."""

    @property
    def name(self) -> str:
        return "totalspineseg"

    def available(self) -> bool:
        return shutil.which("totalspineseg") is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "totalspineseg")
        os.makedirs(out_dir, exist_ok=True)

        # Find input NIfTI — prefer spine/T2w, fall back to any contrast
        nifti_path = find_spine_nifti(bids_dir)
        if not nifti_path:
            nifti_path = find_any_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("TotalSpineSeg: using %s", nifti_path)

        # Build CLI command
        cmd = ["totalspineseg", nifti_path, out_dir]
        if getattr(config, "TOTALSPINESEG_STEP1_ONLY", False):
            cmd.append("--step1")
        if getattr(config, "TOTALSPINESEG_ISO", False):
            cmd.append("--iso")

        # Run CLI
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=config.ANALYTICS_TIMEOUT,
            )
        except FileNotFoundError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="totalspineseg binary not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"TotalSpineSeg timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=result.stderr[-500:] if result.stderr else "TotalSpineSeg failed",
            )

        # Locate output segmentation — check step2 output first, then step1
        seg_path = _find_output_segmentation(out_dir)
        if not seg_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="TotalSpineSeg did not produce output segmentation",
            )

        # Compute per-label volumes and intensity stats
        roi_volumes = compute_label_volumes(seg_path, TOTALSPINESEG_LABELS)
        roi_stats = compute_label_stats(seg_path, nifti_path, TOTALSPINESEG_LABELS)

        # Categorize detected structures
        vertebrae = [n for n in roi_volumes if not n.startswith("IVD_")
                     and n not in ("spinal_cord", "spinal_canal")]
        ivds = [n for n in roi_volumes if n.startswith("IVD_")]

        metrics: dict = {
            "atlas": "totalspineseg",
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
            "roi_stats": roi_stats,
            "vertebrae_detected": sorted(vertebrae),
            "ivd_detected": sorted(ivds),
            "has_spinal_cord": "spinal_cord" in roi_volumes,
            "has_spinal_canal": "spinal_canal" in roi_volumes,
        }

        # Write summary JSON
        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, summary_path]

        log.info(
            "TotalSpineSeg complete for %s: %d ROIs (%d vertebrae, %d IVDs) in %.1fs",
            study_uid,
            len(roi_volumes),
            len(vertebrae),
            len(ivds),
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _find_output_segmentation(out_dir: str) -> str | None:
    """Find the output segmentation NIfTI from TotalSpineSeg output.

    Searches step2 output first (full pipeline), then step1 (coarse), then
    any NIfTI in the output directory.
    """
    # Step 2 output (full instance segmentation)
    for subdir in ["step2_output", "step1_output"]:
        seg_dir = os.path.join(out_dir, subdir)
        if os.path.isdir(seg_dir):
            niis = sorted(glob(os.path.join(seg_dir, "*.nii*")))
            if niis:
                return niis[0]

    # Any NIfTI in the output directory
    niis = sorted(glob(os.path.join(out_dir, "**", "*.nii*"), recursive=True))
    return niis[0] if niis else None
