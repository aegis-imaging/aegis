"""ITK-SNAP / Convert3D backend — threshold-based segmentation.

ITK-SNAP is a popular interactive segmentation tool.  Its companion CLI
``c3d`` (Convert3D) enables headless threshold + connected-component
segmentation pipelines.  ``itksnap-wt`` (workspace tool) is an
alternative entry point.

Requires either:
  - ``c3d`` binary (Convert3D, BSD license)
  - ``itksnap-wt`` binary (ITK-SNAP workspace tool)

References:
  Yushkevich PA et al. "User-guided 3D active contour segmentation of
  anatomical structures: significantly improved efficiency and reliability."
  NeuroImage, 2006.  DOI: 10.1016/j.neuroimage.2006.01.015
"""

from __future__ import annotations

import json
import logging
import os
import shutil
import subprocess
import time

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import compute_label_volumes, compute_label_stats, find_t1w_nifti

log = logging.getLogger(__name__)

# Default tissue labels for a simple 3-class threshold segmentation.
_DEFAULT_LABELS: dict[int, str] = {
    1: "CSF",
    2: "gray_matter",
    3: "white_matter",
}


class ITKSnapBackend(AnalyticsBackend):
    """Threshold-based segmentation via Convert3D (c3d)."""

    @property
    def name(self) -> str:
        return "itksnap"

    def available(self) -> bool:
        c3d = getattr(config, "C3D_BIN", "c3d")
        itksnap_wt = getattr(config, "ITKSNAP_WT_BIN", "itksnap-wt")
        return shutil.which(c3d) is not None or shutil.which(itksnap_wt) is not None

    def _use_c3d(self) -> bool:
        c3d = getattr(config, "C3D_BIN", "c3d")
        return shutil.which(c3d) is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "itksnap")
        os.makedirs(out_dir, exist_ok=True)

        nifti_path = find_t1w_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        log.info("ITK-SNAP: using %s", nifti_path)

        seg_path = os.path.join(out_dir, "itksnap_seg.nii.gz")

        try:
            if self._use_c3d():
                c3d = getattr(config, "C3D_BIN", "c3d")
                # Otsu 3-class segmentation via c3d
                cmd = [
                    c3d, nifti_path,
                    "-otsu-thresh", "3",
                    "-o", seg_path,
                ]
            else:
                itksnap_wt = getattr(config, "ITKSNAP_WT_BIN", "itksnap-wt")
                cmd = [
                    itksnap_wt,
                    "-i", nifti_path,
                    "-o", seg_path,
                ]

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
                error="c3d / itksnap-wt binary not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"ITK-SNAP timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=result.stderr[-500:] if result.stderr else "c3d/itksnap-wt failed",
            )

        if not os.path.isfile(seg_path):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="Segmentation output not produced",
            )

        # Extract volumes and stats
        roi_volumes = compute_label_volumes(seg_path, _DEFAULT_LABELS)
        roi_stats = compute_label_stats(seg_path, nifti_path, _DEFAULT_LABELS)

        metrics: dict = {
            "atlas": "itksnap",
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
            "roi_stats": roi_stats,
        }

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, summary_path]

        log.info(
            "ITK-SNAP complete for %s: %d ROIs in %.1fs",
            study_uid, len(roi_volumes), duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )
