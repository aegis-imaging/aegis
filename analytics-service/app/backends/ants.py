"""ANTs (Advanced Normalization Tools) analytics backend.

Runs cortical thickness estimation, brain extraction, tissue segmentation,
and diffeomorphic registration using ANTs tools.

Requires: ANTs binaries installed (antsCorticalThickness.sh, N4BiasFieldCorrection, etc.).
License: Apache 2.0 (fully open).
Duration: antsCorticalThickness ~4-8 hours; registration alone ~30 min-2 hours.
"""

from __future__ import annotations

import logging
import os
import shutil
import subprocess
import time
from glob import glob

from app import config
from .base import AnalyticsBackend, AnalyticsResult

log = logging.getLogger(__name__)


class ANTsBackend(AnalyticsBackend):
    """ANTs cortical thickness and registration backend."""

    @property
    def name(self) -> str:
        return "ants"

    def available(self) -> bool:
        ants_bin = os.path.join(config.ANTSPATH, "antsCorticalThickness.sh")
        return os.path.isfile(ants_bin) or shutil.which("antsCorticalThickness.sh") is not None

    def analyze(self, bids_dir: str, output_dir: str, study_uid: str) -> AnalyticsResult:
        start = time.time()
        ants_out = os.path.join(output_dir, "ants")
        os.makedirs(ants_out, exist_ok=True)

        env = os.environ.copy()
        env["ANTSPATH"] = config.ANTSPATH
        env["PATH"] = config.ANTSPATH + ":" + env.get("PATH", "")

        # Find T1w NIfTI
        t1w_files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
        if not t1w_files:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        t1w = t1w_files[0]
        prefix = os.path.join(ants_out, "ants_")

        # Find template (OASIS or MNI)
        template_dir = os.getenv("ANTS_TEMPLATE_DIR", "/opt/ants/templates/OASIS-30_Atropos_template")
        brain_template = os.path.join(template_dir, "T_template0.nii.gz")
        brain_prior = os.path.join(template_dir, "T_template0_BrainCerebellumProbabilityMask.nii.gz")
        reg_mask = os.path.join(template_dir, "T_template0_BrainCerebellumRegistrationMask.nii.gz")
        prior_pattern = os.path.join(template_dir, "Priors2", "priors%d.nii.gz")

        if not os.path.isfile(brain_template):
            # Fall back to N4 bias correction + brain extraction only
            log.warning("ANTs template not found at %s — running N4 + extraction only", template_dir)
            return self._run_n4_only(t1w, ants_out, env, start)

        log.info("Running antsCorticalThickness.sh on %s", t1w)

        try:
            result = subprocess.run(
                [
                    "antsCorticalThickness.sh",
                    "-d", "3",
                    "-a", t1w,
                    "-e", brain_template,
                    "-m", brain_prior,
                    "-f", reg_mask,
                    "-p", prior_pattern,
                    "-o", prefix,
                ],
                capture_output=True,
                text=True,
                timeout=config.ANALYTICS_TIMEOUT,
                env=env,
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"antsCorticalThickness timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        duration = time.time() - start

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=duration,
                error=result.stderr[-2000:] if result.stderr else "antsCorticalThickness failed",
            )

        outputs = glob(f"{prefix}*")
        log.info("ANTs complete (%.1fs, %d outputs)", duration, len(outputs))

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
        )

    def _run_n4_only(self, t1w: str, out_dir: str, env: dict, start: float) -> AnalyticsResult:
        """Fallback: run N4 bias field correction only."""
        n4_out = os.path.join(out_dir, "t1w_n4.nii.gz")
        try:
            result = subprocess.run(
                ["N4BiasFieldCorrection", "-d", "3", "-i", t1w, "-o", n4_out],
                capture_output=True,
                text=True,
                timeout=3600,
                env=env,
            )
        except (subprocess.TimeoutExpired, FileNotFoundError) as e:
            return AnalyticsResult(
                tool=self.name, success=False,
                duration_seconds=time.time() - start,
                error=str(e),
            )

        duration = time.time() - start
        outputs = [n4_out] if os.path.isfile(n4_out) else []

        return AnalyticsResult(
            tool=self.name,
            success=result.returncode == 0,
            duration_seconds=duration,
            outputs=outputs,
            error=result.stderr[-500:] if result.returncode != 0 else "",
        )
