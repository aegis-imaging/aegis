"""BrainSuite backend — cortical surface extraction and tissue classification.

BrainSuite provides a sequential pipeline of command-line tools for MRI
analysis: brain surface extraction (BSE), bias field correction (BFC),
partial volume classification (PVC), and cerebrum labeling (Cerebro).

Requires:
  - ``bse`` binary (BrainSuite, free for academic use)

References:
  Shattuck DW, Leahy RM. "BrainSuite: An Automated Cortical Surface
  Identification Tool." Medical Image Analysis, 2002.
  DOI: 10.1016/S1361-8415(02)00054-3
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
from .seg_utils import compute_label_volumes, compute_label_stats, find_t1w_nifti

log = logging.getLogger(__name__)

# BrainSuite cerebro atlas label subset (representative brain structures).
_BRAINSUITE_LABELS: dict[int, str] = {
    2: "left_cerebral_white_matter",
    3: "left_cerebral_cortex",
    4: "left_lateral_ventricle",
    5: "left_inferior_lateral_ventricle",
    7: "left_cerebellum_white_matter",
    8: "left_cerebellum_cortex",
    10: "left_thalamus",
    11: "left_caudate",
    12: "left_putamen",
    13: "left_pallidum",
    17: "left_hippocampus",
    18: "left_amygdala",
    26: "left_accumbens",
    41: "right_cerebral_white_matter",
    42: "right_cerebral_cortex",
    43: "right_lateral_ventricle",
    44: "right_inferior_lateral_ventricle",
    46: "right_cerebellum_white_matter",
    47: "right_cerebellum_cortex",
    49: "right_thalamus",
    50: "right_caudate",
    51: "right_putamen",
    52: "right_pallidum",
    53: "right_hippocampus",
    54: "right_amygdala",
    58: "right_accumbens",
    16: "brain_stem",
}


class BrainSuiteBackend(AnalyticsBackend):
    """Cortical surface extraction and tissue classification via BrainSuite."""

    @property
    def name(self) -> str:
        return "brainsuite"

    def available(self) -> bool:
        bse = getattr(config, "BSE_BIN", "bse")
        return shutil.which(bse) is not None

    def _run_cmd(
        self, cmd: list[str], timeout: int | None = None,
    ) -> subprocess.CompletedProcess:
        """Run a subprocess with standard error handling."""
        return subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout or config.ANALYTICS_TIMEOUT,
        )

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "brainsuite")
        os.makedirs(out_dir, exist_ok=True)

        nifti_path = find_t1w_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        log.info("BrainSuite: using %s", nifti_path)

        bse = getattr(config, "BSE_BIN", "bse")
        brain_path = os.path.join(out_dir, "brain.nii.gz")
        bfc_path = os.path.join(out_dir, "brain.bfc.nii.gz")
        pvc_label_path = os.path.join(out_dir, "brain.pvc.label.nii.gz")

        # Step 1: Brain Surface Extraction
        try:
            result = self._run_cmd([bse, "-i", nifti_path, "-o", brain_path, "--auto"])
        except FileNotFoundError:
            return AnalyticsResult(
                tool=self.name, success=False,
                duration_seconds=time.time() - start,
                error="bse binary not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name, success=False,
                duration_seconds=time.time() - start,
                error=f"BrainSuite timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name, success=False,
                duration_seconds=time.time() - start,
                error=result.stderr[-500:] if result.stderr else "bse failed",
            )

        outputs = [brain_path]
        partial_metrics: dict = {}

        # Step 2: Bias Field Correction (optional — continue on failure)
        if shutil.which("bfc"):
            try:
                r2 = self._run_cmd(["bfc", "-i", brain_path, "-o", bfc_path])
                if r2.returncode == 0 and os.path.isfile(bfc_path):
                    outputs.append(bfc_path)
                    log.info("BrainSuite: BFC completed")
            except (subprocess.TimeoutExpired, FileNotFoundError):
                log.warning("BrainSuite: BFC step skipped")

        # Step 3: Partial Volume Classifier (optional — continue on failure)
        input_for_pvc = bfc_path if os.path.isfile(bfc_path) else brain_path
        if shutil.which("pvc"):
            try:
                r3 = self._run_cmd(["pvc", "-i", input_for_pvc])
                if r3.returncode == 0:
                    # PVC writes label file adjacent to input
                    pvc_candidates = glob(os.path.join(out_dir, "*.pvc.label.nii.gz"))
                    if pvc_candidates:
                        pvc_label_path = pvc_candidates[0]
                        outputs.append(pvc_label_path)
                        log.info("BrainSuite: PVC completed")
            except (subprocess.TimeoutExpired, FileNotFoundError):
                log.warning("BrainSuite: PVC step skipped")

        # Step 4: Cerebro labeling (optional — continue on failure)
        cerebro_label_path = os.path.join(out_dir, "brain.cerebro.label.nii.gz")
        if shutil.which("cerebro"):
            try:
                r4 = self._run_cmd([
                    "cerebro", "--input", input_for_pvc, "-o", cerebro_label_path,
                ])
                if r4.returncode == 0 and os.path.isfile(cerebro_label_path):
                    outputs.append(cerebro_label_path)
                    log.info("BrainSuite: Cerebro labeling completed")
            except (subprocess.TimeoutExpired, FileNotFoundError):
                log.warning("BrainSuite: Cerebro step skipped")

        # Extract volumes from whichever label output we have
        label_file = None
        for candidate in [cerebro_label_path, pvc_label_path]:
            if os.path.isfile(candidate):
                label_file = candidate
                break

        roi_volumes: dict[str, float] = {}
        roi_stats: list[dict] = []
        if label_file:
            roi_volumes = compute_label_volumes(label_file, _BRAINSUITE_LABELS)
            roi_stats = compute_label_stats(label_file, nifti_path, _BRAINSUITE_LABELS)

        metrics: dict = {
            "atlas": "brainsuite",
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
            "roi_stats": roi_stats,
        }

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)
        outputs.append(summary_path)

        duration = time.time() - start
        log.info(
            "BrainSuite complete for %s: %d ROIs in %.1fs",
            study_uid, len(roi_volumes), duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )
