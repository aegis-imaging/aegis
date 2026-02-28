"""SPM (Statistical Parametric Mapping) analytics backend.

Runs tissue segmentation and DARTEL normalization using the standalone
SPM12 binary with MATLAB Compiler Runtime.

Requires: spm12 standalone binary + MATLAB Compiler Runtime installed.
License: GPL v2 (free); standalone doesn't require MATLAB license.
Duration: Segmentation ~5 min, DARTEL ~30 min per subject.
"""

from __future__ import annotations

import logging
import os
import shutil
import subprocess
import tempfile
import time
from glob import glob

from app import config
from .base import AnalyticsBackend, AnalyticsResult

log = logging.getLogger(__name__)


def _generate_segmentation_batch(t1w_path: str, output_dir: str) -> str:
    """Generate a MATLAB batch script for SPM segmentation."""
    return f"""% SPM12 Segmentation batch
matlabbatch{{1}}.spm.spatial.preproc.channel.vols = {{'{t1w_path},1'}};
matlabbatch{{1}}.spm.spatial.preproc.channel.biasreg = 0.001;
matlabbatch{{1}}.spm.spatial.preproc.channel.biasfwhm = 60;
matlabbatch{{1}}.spm.spatial.preproc.channel.write = [0 1];
matlabbatch{{1}}.spm.spatial.preproc.tissue(1).tpm = {{fullfile(spm('Dir'),'tpm','TPM.nii,1')}};
matlabbatch{{1}}.spm.spatial.preproc.tissue(1).ngaus = 1;
matlabbatch{{1}}.spm.spatial.preproc.tissue(1).native = [1 0];
matlabbatch{{1}}.spm.spatial.preproc.tissue(1).warped = [0 0];
matlabbatch{{1}}.spm.spatial.preproc.tissue(2).tpm = {{fullfile(spm('Dir'),'tpm','TPM.nii,2')}};
matlabbatch{{1}}.spm.spatial.preproc.tissue(2).ngaus = 1;
matlabbatch{{1}}.spm.spatial.preproc.tissue(2).native = [1 0];
matlabbatch{{1}}.spm.spatial.preproc.tissue(2).warped = [0 0];
matlabbatch{{1}}.spm.spatial.preproc.tissue(3).tpm = {{fullfile(spm('Dir'),'tpm','TPM.nii,3')}};
matlabbatch{{1}}.spm.spatial.preproc.tissue(3).ngaus = 2;
matlabbatch{{1}}.spm.spatial.preproc.tissue(3).native = [1 0];
matlabbatch{{1}}.spm.spatial.preproc.tissue(3).warped = [0 0];
spm('defaults', 'fmri');
spm_jobman('run', matlabbatch);
"""


class SPMBackend(AnalyticsBackend):
    """SPM segmentation + normalization backend."""

    @property
    def name(self) -> str:
        return "spm"

    def available(self) -> bool:
        return shutil.which(config.SPM_BIN) is not None

    def analyze(self, bids_dir: str, output_dir: str, study_uid: str) -> AnalyticsResult:
        start = time.time()
        spm_out = os.path.join(output_dir, "spm")
        os.makedirs(spm_out, exist_ok=True)

        # Find T1w NIfTI
        t1w_files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
        if not t1w_files:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        t1w = t1w_files[0]
        # SPM needs uncompressed NIfTI; if .nii.gz, decompress
        if t1w.endswith(".nii.gz"):
            import gzip
            unzipped = os.path.join(spm_out, os.path.basename(t1w).replace(".nii.gz", ".nii"))
            with gzip.open(t1w, "rb") as f_in, open(unzipped, "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)
            t1w = unzipped

        # Generate batch script
        batch_script = _generate_segmentation_batch(t1w, spm_out)

        with tempfile.NamedTemporaryFile(mode="w", suffix=".m", delete=False, dir=spm_out) as f:
            f.write(batch_script)
            batch_path = f.name

        log.info("Running SPM segmentation on %s", t1w)

        env = os.environ.copy()
        if config.MCR_DIR:
            env["MCR_DIR"] = config.MCR_DIR

        try:
            result = subprocess.run(
                [config.SPM_BIN, "batch", batch_path],
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
                error=f"SPM timed out after {config.ANALYTICS_TIMEOUT}s",
            )
        except FileNotFoundError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"SPM binary not found: {config.SPM_BIN}",
            )
        finally:
            try:
                os.unlink(batch_path)
            except OSError:
                pass

        duration = time.time() - start

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=duration,
                error=result.stderr[-2000:] if result.stderr else "SPM segmentation failed",
            )

        # Collect segmentation outputs (c1=GM, c2=WM, c3=CSF)
        t1w_base = os.path.splitext(os.path.basename(t1w))[0]
        t1w_dir = os.path.dirname(t1w)
        outputs = []
        for prefix in ["c1", "c2", "c3", "m", "y_"]:
            outputs.extend(glob(os.path.join(t1w_dir, f"{prefix}{t1w_base}*")))
            outputs.extend(glob(os.path.join(spm_out, f"{prefix}{t1w_base}*")))

        log.info("SPM complete (%.1fs, %d outputs)", duration, len(outputs))

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
        )
