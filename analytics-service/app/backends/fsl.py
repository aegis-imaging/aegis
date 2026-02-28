"""FSL analytics backend.

Runs brain extraction (BET), tissue segmentation (FAST), and linear
registration to MNI152 (FLIRT) on T1w NIfTI data. Optionally runs
DTI fitting (dtifit) on DWI data.

Requires: FSL binaries installed (bet, fast, flirt, dtifit, eddy).
Duration: BET ~1 min, FAST ~5 min, FLIRT ~2 min, DTI pipeline ~30 min.
"""

from __future__ import annotations

import logging
import os
import shutil
import subprocess
import time
from glob import glob
from pathlib import Path

from app import config
from .base import AnalyticsBackend, AnalyticsResult

log = logging.getLogger(__name__)


class FSLBackend(AnalyticsBackend):
    """FSL BET + FAST + FLIRT + DTIFIT analytics backend."""

    @property
    def name(self) -> str:
        return "fsl"

    def available(self) -> bool:
        return shutil.which("bet") is not None

    def _run_cmd(self, cmd: list[str], env: dict, label: str) -> subprocess.CompletedProcess:
        """Run a subprocess command with FSL environment."""
        log.info("Running %s: %s", label, " ".join(cmd))
        return subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=config.ANALYTICS_TIMEOUT,
            env=env,
        )

    def analyze(self, bids_dir: str, output_dir: str, study_uid: str) -> AnalyticsResult:
        start = time.time()
        fsl_out = os.path.join(output_dir, "fsl")
        os.makedirs(fsl_out, exist_ok=True)

        env = os.environ.copy()
        env["FSLDIR"] = config.FSL_DIR
        env["FSLOUTPUTTYPE"] = "NIFTI_GZ"
        fsl_bin = os.path.join(config.FSL_DIR, "bin")
        if os.path.isdir(fsl_bin):
            env["PATH"] = fsl_bin + ":" + env.get("PATH", "")

        # Find T1w NIfTI
        t1w_files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
        if not t1w_files:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        t1w = t1w_files[0]
        outputs = []
        errors = []

        # BET: brain extraction
        brain = os.path.join(fsl_out, "brain")
        result = self._run_cmd(["bet", t1w, brain, "-R", "-m"], env, "BET")
        if result.returncode != 0:
            errors.append(f"BET failed: {result.stderr[-500:]}")
        else:
            outputs.extend(glob(f"{brain}*"))

        # FAST: tissue segmentation (only if BET succeeded)
        brain_nii = f"{brain}.nii.gz"
        if os.path.isfile(brain_nii):
            result = self._run_cmd(["fast", "-t", "1", "-n", "3", "-o", brain, brain_nii], env, "FAST")
            if result.returncode != 0:
                errors.append(f"FAST failed: {result.stderr[-500:]}")
            else:
                outputs.extend(glob(f"{brain}_seg*") + glob(f"{brain}_pve*"))

        # FLIRT: registration to MNI152
        mni_template = os.path.join(config.FSL_DIR, "data", "standard", "MNI152_T1_2mm_brain.nii.gz")
        if os.path.isfile(brain_nii) and os.path.isfile(mni_template):
            to_mni = os.path.join(fsl_out, "brain_to_mni")
            mat_out = os.path.join(fsl_out, "brain_to_mni.mat")
            result = self._run_cmd(
                ["flirt", "-in", brain_nii, "-ref", mni_template, "-out", to_mni, "-omat", mat_out],
                env, "FLIRT",
            )
            if result.returncode != 0:
                errors.append(f"FLIRT failed: {result.stderr[-500:]}")
            else:
                outputs.extend([f for f in [f"{to_mni}.nii.gz", mat_out] if os.path.isfile(f)])

        # DTI: if DWI data exists
        dwi_files = glob(os.path.join(bids_dir, "**", "*dwi*.nii*"), recursive=True)
        if dwi_files and shutil.which("dtifit"):
            dwi = dwi_files[0]
            dwi_dir = os.path.dirname(dwi)
            bval = glob(os.path.join(dwi_dir, "*.bval"))
            bvec = glob(os.path.join(dwi_dir, "*.bvec"))
            mask = f"{brain}_mask.nii.gz"
            if bval and bvec and os.path.isfile(mask):
                dti_out = os.path.join(fsl_out, "dti")
                result = self._run_cmd(
                    ["dtifit", "-k", dwi, "-o", dti_out, "-m", mask, "-r", bvec[0], "-b", bval[0]],
                    env, "DTIFIT",
                )
                if result.returncode != 0:
                    errors.append(f"DTIFIT failed: {result.stderr[-500:]}")
                else:
                    outputs.extend(glob(f"{dti_out}_*"))

        duration = time.time() - start
        success = len(errors) == 0

        return AnalyticsResult(
            tool=self.name,
            success=success,
            duration_seconds=duration,
            outputs=outputs,
            error="; ".join(errors) if errors else "",
        )
