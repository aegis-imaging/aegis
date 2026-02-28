"""FreeSurfer recon-all analytics backend.

Runs cortical reconstruction and volumetric segmentation on T1w NIfTI data.
Outputs: subcortical volumetrics (aseg.stats), cortical parcellation (aparc.stats),
cortical thickness maps, skull-stripped brain, and segmentation volumes.

Requires: FreeSurfer binaries installed, license file at FS_LICENSE path.
Duration: ~2.5 hours (v8.0) to ~8 hours (v7.x).
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


def _parse_stats_file(stats_path: str) -> dict:
    """Parse a FreeSurfer .stats file into a dict of region -> volume_mm3."""
    volumes = {}
    try:
        with open(stats_path) as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                parts = line.split()
                if len(parts) >= 5:
                    region = parts[4] if len(parts) > 4 else parts[0]
                    try:
                        volume = float(parts[3])
                        volumes[region] = volume
                    except (ValueError, IndexError):
                        continue
    except FileNotFoundError:
        log.warning("Stats file not found: %s", stats_path)
    return volumes


class FreeSurferBackend(AnalyticsBackend):
    """FreeSurfer recon-all analytics backend."""

    @property
    def name(self) -> str:
        return "freesurfer"

    def available(self) -> bool:
        if not shutil.which("recon-all"):
            return False
        if not os.path.isfile(config.FS_LICENSE):
            log.warning("FreeSurfer license not found at %s", config.FS_LICENSE)
            return False
        return True

    def analyze(self, bids_dir: str, output_dir: str, study_uid: str) -> AnalyticsResult:
        start = time.time()
        subject_id = f"sub-{study_uid[:8]}"
        subjects_dir = os.path.join(output_dir, "freesurfer")
        os.makedirs(subjects_dir, exist_ok=True)

        # Find T1w NIfTI
        t1w_files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
        if not t1w_files:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        t1w = t1w_files[0]
        log.info("Running recon-all on %s (subject: %s)", t1w, subject_id)

        env = os.environ.copy()
        env["SUBJECTS_DIR"] = subjects_dir
        env["FS_LICENSE"] = config.FS_LICENSE

        try:
            result = subprocess.run(
                [
                    "recon-all",
                    "-s", subject_id,
                    "-i", t1w,
                    "-all",
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
                error=f"recon-all timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        duration = time.time() - start

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=duration,
                error=result.stderr[-2000:] if result.stderr else "recon-all failed",
            )

        # Collect outputs
        subj_dir = os.path.join(subjects_dir, subject_id)
        outputs = []
        for pattern in ["stats/*.stats", "surf/lh.thickness", "surf/rh.thickness",
                        "mri/brain.mgz", "mri/aseg.mgz"]:
            outputs.extend(glob(os.path.join(subj_dir, pattern)))

        # Parse volumetric stats
        metrics = {}
        aseg_path = os.path.join(subj_dir, "stats", "aseg.stats")
        if os.path.isfile(aseg_path):
            metrics["subcortical_volumes"] = _parse_stats_file(aseg_path)

        for hemi in ("lh", "rh"):
            aparc_path = os.path.join(subj_dir, "stats", f"{hemi}.aparc.stats")
            if os.path.isfile(aparc_path):
                metrics[f"{hemi}_cortical_parcellation"] = _parse_stats_file(aparc_path)

        log.info("recon-all complete for %s (%.1fs, %d outputs)", subject_id, duration, len(outputs))

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )
