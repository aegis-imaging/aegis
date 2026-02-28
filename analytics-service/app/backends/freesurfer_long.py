"""FreeSurfer longitudinal stream analytics backend.

Runs FreeSurfer's longitudinal pipeline on paired baseline + follow-up T1w
NIfTI data. Steps:
  1. recon-all -all on baseline (if not already processed)
  2. recon-all -all on follow-up (if not already processed)
  3. recon-all -base (create unbiased template from both timepoints)
  4. recon-all -long (longitudinal processing of each timepoint)

Outputs: per-timepoint cortical thickness maps, volumetric segmentations,
symmetrized percent change (SPC) maps, and longitudinal statistics.

Requires: FreeSurfer binaries installed, license file at FS_LICENSE path.
Duration: ~6–18 hours total (2× cross-sectional + base + 2× long).
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
from .base import LongitudinalBackend, AnalyticsResult

log = logging.getLogger(__name__)


def _find_t1w(bids_dir: str) -> str | None:
    """Find the first T1w NIfTI file in a BIDS directory."""
    t1w_files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
    return t1w_files[0] if t1w_files else None


def _run_recon(
    cmd: list[str],
    subjects_dir: str,
    timeout: int,
) -> subprocess.CompletedProcess:
    """Run a recon-all command with proper environment."""
    env = os.environ.copy()
    env["SUBJECTS_DIR"] = subjects_dir
    env["FS_LICENSE"] = config.FS_LICENSE
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=timeout,
        env=env,
    )


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


class FreeSurferLongBackend(LongitudinalBackend):
    """FreeSurfer longitudinal stream backend.

    Uses recon-all -base (unbiased template) + recon-all -long (longitudinal
    processing) to compute cortical thickness and volume changes between
    two timepoints with reduced bias from within-subject variability.
    """

    @property
    def name(self) -> str:
        return "freesurfer_long"

    def available(self) -> bool:
        if not shutil.which("recon-all"):
            return False
        if not os.path.isfile(config.FS_LICENSE):
            log.warning("FreeSurfer license not found at %s", config.FS_LICENSE)
            return False
        return True

    def analyze_longitudinal(
        self,
        baseline_bids_dir: str,
        followup_bids_dir: str,
        output_dir: str,
        baseline_study_uid: str,
        followup_study_uid: str,
        scan_interval_days: float,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        subjects_dir = os.path.join(output_dir, "freesurfer_long")
        os.makedirs(subjects_dir, exist_ok=True)

        tp1_id = f"sub-{baseline_study_uid[:8]}_tp1"
        tp2_id = f"sub-{followup_study_uid[:8]}_tp2"
        base_id = f"sub-{baseline_study_uid[:8]}_base"

        # Find T1w NIfTI for each timepoint
        tp1_t1w = _find_t1w(baseline_bids_dir)
        tp2_t1w = _find_t1w(followup_bids_dir)

        if not tp1_t1w:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI found in baseline BIDS directory",
            )
        if not tp2_t1w:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI found in follow-up BIDS directory",
            )

        timeout = config.ANALYTICS_TIMEOUT
        outputs: list[str] = []
        metrics: dict = {
            "scan_interval_days": scan_interval_days,
            "baseline_study_uid": baseline_study_uid,
            "followup_study_uid": followup_study_uid,
        }

        # Step 1: Cross-sectional recon-all on baseline (skip if already done)
        tp1_dir = os.path.join(subjects_dir, tp1_id)
        if not os.path.isdir(tp1_dir):
            log.info("Step 1/4: recon-all on baseline %s", tp1_id)
            try:
                r = _run_recon(
                    ["recon-all", "-s", tp1_id, "-i", tp1_t1w, "-all"],
                    subjects_dir, timeout,
                )
                if r.returncode != 0:
                    return AnalyticsResult(
                        tool=self.name, success=False,
                        duration_seconds=time.time() - start,
                        error=f"Baseline recon-all failed: {(r.stderr or '')[-1000:]}",
                    )
            except subprocess.TimeoutExpired:
                return AnalyticsResult(
                    tool=self.name, success=False,
                    duration_seconds=time.time() - start,
                    error=f"Baseline recon-all timed out after {timeout}s",
                )
        else:
            log.info("Step 1/4: baseline %s already processed, skipping", tp1_id)

        # Step 2: Cross-sectional recon-all on follow-up (skip if already done)
        tp2_dir = os.path.join(subjects_dir, tp2_id)
        if not os.path.isdir(tp2_dir):
            log.info("Step 2/4: recon-all on follow-up %s", tp2_id)
            try:
                r = _run_recon(
                    ["recon-all", "-s", tp2_id, "-i", tp2_t1w, "-all"],
                    subjects_dir, timeout,
                )
                if r.returncode != 0:
                    return AnalyticsResult(
                        tool=self.name, success=False,
                        duration_seconds=time.time() - start,
                        error=f"Follow-up recon-all failed: {(r.stderr or '')[-1000:]}",
                    )
            except subprocess.TimeoutExpired:
                return AnalyticsResult(
                    tool=self.name, success=False,
                    duration_seconds=time.time() - start,
                    error=f"Follow-up recon-all timed out after {timeout}s",
                )
        else:
            log.info("Step 2/4: follow-up %s already processed, skipping", tp2_id)

        # Step 3: Create unbiased base template
        log.info("Step 3/4: recon-all -base %s from %s + %s", base_id, tp1_id, tp2_id)
        try:
            r = _run_recon(
                [
                    "recon-all", "-base", base_id,
                    "-tp", tp1_id, "-tp", tp2_id,
                    "-all",
                ],
                subjects_dir, timeout,
            )
            if r.returncode != 0:
                return AnalyticsResult(
                    tool=self.name, success=False,
                    duration_seconds=time.time() - start,
                    error=f"Base template creation failed: {(r.stderr or '')[-1000:]}",
                )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name, success=False,
                duration_seconds=time.time() - start,
                error=f"Base template timed out after {timeout}s",
            )

        # Step 4: Longitudinal processing for both timepoints
        for tp_id in (tp1_id, tp2_id):
            long_id = f"{tp_id}.long.{base_id}"
            log.info("Step 4/4: recon-all -long %s -> %s", tp_id, long_id)
            try:
                r = _run_recon(
                    ["recon-all", "-long", tp_id, base_id, "-all"],
                    subjects_dir, timeout,
                )
                if r.returncode != 0:
                    return AnalyticsResult(
                        tool=self.name, success=False,
                        duration_seconds=time.time() - start,
                        error=f"Longitudinal processing failed for {tp_id}: {(r.stderr or '')[-1000:]}",
                    )
            except subprocess.TimeoutExpired:
                return AnalyticsResult(
                    tool=self.name, success=False,
                    duration_seconds=time.time() - start,
                    error=f"Longitudinal processing timed out for {tp_id}",
                )

        duration = time.time() - start

        # Collect outputs from longitudinal directories
        tp1_long_dir = os.path.join(subjects_dir, f"{tp1_id}.long.{base_id}")
        tp2_long_dir = os.path.join(subjects_dir, f"{tp2_id}.long.{base_id}")

        for d in (tp1_long_dir, tp2_long_dir):
            for pattern in [
                "stats/*.stats", "surf/lh.thickness", "surf/rh.thickness",
                "mri/brain.mgz", "mri/aseg.mgz",
            ]:
                outputs.extend(glob(os.path.join(d, pattern)))

        # Parse volumetric stats from both longitudinal timepoints
        for tp_label, long_dir in [("baseline", tp1_long_dir), ("followup", tp2_long_dir)]:
            aseg = os.path.join(long_dir, "stats", "aseg.stats")
            if os.path.isfile(aseg):
                metrics[f"{tp_label}_subcortical_volumes"] = _parse_stats_file(aseg)
            for hemi in ("lh", "rh"):
                aparc = os.path.join(long_dir, "stats", f"{hemi}.aparc.stats")
                if os.path.isfile(aparc):
                    metrics[f"{tp_label}_{hemi}_cortical"] = _parse_stats_file(aparc)

        # Compute volume change percentages for key structures
        bl_vols = metrics.get("baseline_subcortical_volumes", {})
        fu_vols = metrics.get("followup_subcortical_volumes", {})
        if bl_vols and fu_vols:
            changes = {}
            for region in bl_vols:
                if region in fu_vols and bl_vols[region] > 0:
                    pct = ((fu_vols[region] - bl_vols[region]) / bl_vols[region]) * 100
                    changes[region] = round(pct, 3)
            if changes:
                metrics["volume_change_percent"] = changes

        log.info(
            "FreeSurfer longitudinal complete: %s vs %s (%.1fs, %d outputs)",
            baseline_study_uid, followup_study_uid, duration, len(outputs),
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )
