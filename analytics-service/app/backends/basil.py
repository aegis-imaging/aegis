"""BASIL / oxford_asl backend — ASL perfusion quantification.

BASIL (Bayesian Inference for Arterial Spin Labelling) is part of FSL
and provides tools for quantitative analysis of Arterial Spin Labeling
(ASL) MRI data.  ``oxford_asl`` is the main pipeline entry point,
producing calibrated cerebral blood flow (CBF) maps.

Requires:
  - ``oxford_asl`` binary (part of FSL, Oxford University Innovation)

References:
  Chappell MA et al. "Variational Bayesian Inference for a Nonlinear
  Forward Model." IEEE Transactions on Signal Processing, 2009.
  DOI: 10.1109/TSP.2008.2005752

  https://asl-docs.readthedocs.io/en/latest/
"""

from __future__ import annotations

import json
import logging
import os
import shutil
import subprocess
import time

import nibabel as nib
import numpy as np

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import find_asl_nifti, find_t1w_nifti

log = logging.getLogger(__name__)


class BASILBackend(AnalyticsBackend):
    """ASL perfusion quantification via BASIL / oxford_asl."""

    @property
    def name(self) -> str:
        return "basil"

    def available(self) -> bool:
        return shutil.which("oxford_asl") is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "basil")
        os.makedirs(out_dir, exist_ok=True)

        # Find ASL NIfTI
        asl_path = find_asl_nifti(bids_dir)
        if not asl_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No ASL NIfTI files found in BIDS directory",
            )

        log.info("BASIL: using ASL=%s", asl_path)

        # Optional structural T1w for calibration and registration
        t1w_path = find_t1w_nifti(bids_dir)

        # Configuration
        bolus = getattr(config, "BASIL_BOLUS_DURATION", "1.8")
        tis = getattr(config, "BASIL_TIS", "3.6")
        calib_method = getattr(config, "BASIL_CALIB_METHOD", "voxel")

        # Build oxford_asl command
        cmd = [
            "oxford_asl",
            "-i", asl_path,
            "-o", out_dir,
            "--bolus", bolus,
            "--tis", tis,
        ]
        if t1w_path:
            cmd.extend(["--struct", t1w_path])

        # Run FSL environment setup
        fsl_dir = getattr(config, "FSL_DIR", "/usr/local/fsl")
        env = os.environ.copy()
        env["FSLDIR"] = fsl_dir
        env["PATH"] = os.path.join(fsl_dir, "bin") + ":" + env.get("PATH", "")
        fsl_out = os.path.join(fsl_dir, "data", "standard", "MNI152_T1_2mm.nii.gz")
        env.setdefault("FSLOUTPUTTYPE", "NIFTI_GZ")

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=config.ANALYTICS_TIMEOUT,
                env=env,
            )
        except FileNotFoundError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="oxford_asl binary not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"BASIL timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=result.stderr[-500:] if result.stderr else "oxford_asl failed",
            )

        # Find the calibrated CBF output
        cbf_path = _find_cbf_output(out_dir)

        # Compute CBF statistics
        global_cbf: dict[str, float] = {}
        if cbf_path:
            global_cbf = _compute_cbf_stats(cbf_path)

        metrics: dict = {
            "atlas": "basil",
            "global_cbf": global_cbf,
            "roi_count": 0,
            "roi_cbf": [],
            "bolus_duration": bolus,
            "post_label_delay": tis,
        }

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [summary_path]
        if cbf_path:
            outputs.append(cbf_path)

        log.info(
            "BASIL complete for %s: mean CBF=%.1f ml/100g/min in %.1fs",
            study_uid,
            global_cbf.get("mean", 0),
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _find_cbf_output(output_dir: str) -> str | None:
    """Find the calibrated CBF map from oxford_asl output."""
    from glob import glob

    # oxford_asl outputs various CBF maps; prefer calibrated
    for pattern in [
        "native_space/perfusion_calib.nii.gz",
        "native_space/perfusion.nii.gz",
        "perfusion_calib.nii.gz",
        "perfusion.nii.gz",
    ]:
        candidates = glob(os.path.join(output_dir, "**", pattern), recursive=True)
        if candidates:
            return candidates[0]
    return None


def _compute_cbf_stats(cbf_path: str) -> dict[str, float]:
    """Compute global CBF statistics from a perfusion map."""
    img = nib.load(cbf_path)
    data = np.asarray(img.dataobj, dtype=np.float64)

    # Mask out zero/negative values (non-brain)
    valid = data[data > 0]
    if valid.size == 0:
        return {"mean": 0.0, "median": 0.0, "std": 0.0}

    return {
        "mean": round(float(np.mean(valid)), 2),
        "median": round(float(np.median(valid)), 2),
        "std": round(float(np.std(valid)), 2),
    }
