"""SynthSeg backend — contrast-agnostic brain MRI segmentation.

SynthSeg segments brain MRI scans regardless of acquisition parameters
(T1w, T2w, FLAIR, any resolution). With ``--parc``, it produces 97 ROIs
(68 cortical + 29 subcortical) following the FreeSurfer Desikan-Killiany atlas.

Requires either:
  - FreeSurfer 7.3.2+ (``mri_synthseg`` binary)
  - Standalone SynthSeg Python package (Apache 2.0)

References:
  Billot et al. "SynthSeg: Segmentation of brain MRI scans of any contrast
  and resolution without retraining." Medical Image Analysis, 2023.
  DOI: 10.1016/j.media.2023.102789
"""

from __future__ import annotations

import csv
import json
import logging
import os
import shutil
import subprocess
import time

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import find_any_nifti

log = logging.getLogger(__name__)


class SynthSegBackend(AnalyticsBackend):
    """Contrast-agnostic brain segmentation via SynthSeg."""

    @property
    def name(self) -> str:
        return "synthseg"

    def available(self) -> bool:
        # Check for FreeSurfer's mri_synthseg binary
        if shutil.which("mri_synthseg"):
            return True
        # Check for standalone SynthSeg Python package
        try:
            import SynthSeg  # noqa: F401
            return True
        except ImportError:
            pass
        return False

    def _use_freesurfer(self) -> bool:
        """Return True if mri_synthseg is available (preferred path)."""
        return shutil.which("mri_synthseg") is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "synthseg")
        os.makedirs(out_dir, exist_ok=True)

        # Find any NIfTI (SynthSeg is contrast-agnostic)
        nifti_path = find_any_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("SynthSeg: using %s", nifti_path)

        # Output paths
        seg_path = os.path.join(out_dir, "synthseg_seg.nii.gz")
        vol_path = os.path.join(out_dir, "synthseg_volumes.csv")
        qc_path = os.path.join(out_dir, "synthseg_qc.csv")

        # Build command
        use_parc = getattr(config, "SYNTHSEG_PARC", "true").lower() == "true"
        use_robust = getattr(config, "SYNTHSEG_ROBUST", "false").lower() == "true"
        use_qc = getattr(config, "SYNTHSEG_QC", "true").lower() == "true"

        if self._use_freesurfer():
            cmd = ["mri_synthseg", "--i", nifti_path, "--o", seg_path]
        else:
            cmd = [
                "python", "-m", "SynthSeg.predict",
                "--i", nifti_path, "--o", seg_path,
            ]

        cmd.extend(["--vol", vol_path])
        if use_parc:
            cmd.append("--parc")
        if use_robust:
            cmd.append("--robust")
        if use_qc:
            cmd.extend(["--qc", qc_path])

        # Run SynthSeg
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
                error="SynthSeg binary/module not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"SynthSeg timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=result.stderr[-500:] if result.stderr else "SynthSeg failed",
            )

        # Parse volumetric CSV
        roi_volumes = _parse_volumes_csv(vol_path)

        # Parse QC score
        qc_score = _parse_qc_csv(qc_path) if use_qc and os.path.isfile(qc_path) else None

        # Build metrics
        metrics: dict = {
            "atlas": "synthseg",
            "roi_count": len(roi_volumes),
            "parcellation": use_parc,
            "roi_volumes": roi_volumes,
        }
        if qc_score is not None:
            metrics["qc_score"] = qc_score

        # Write summary JSON
        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, vol_path, summary_path]
        if use_qc and os.path.isfile(qc_path):
            outputs.append(qc_path)

        log.info(
            "SynthSeg complete for %s: %d ROIs in %.1fs (QC=%.3f)",
            study_uid,
            len(roi_volumes),
            duration,
            qc_score if qc_score is not None else -1,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _parse_volumes_csv(csv_path: str) -> dict[str, float]:
    """Parse SynthSeg volumetric CSV output into {roi_name: volume_mm3}.

    SynthSeg CSV format (mri_synthseg --vol):
      subject,Left-Hippocampus,Right-Hippocampus,...
      /path/to/input.nii.gz,3456.2,3400.1,...
    """
    if not os.path.isfile(csv_path):
        return {}

    volumes: dict[str, float] = {}
    try:
        with open(csv_path, encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                for key, val in row.items():
                    if key in ("subject", ""):
                        continue
                    try:
                        volumes[key] = round(float(val), 2)
                    except (ValueError, TypeError):
                        continue
                break  # Only first data row
    except Exception as e:
        log.warning("Failed to parse SynthSeg volumes CSV: %s", e)

    return volumes


def _parse_qc_csv(csv_path: str) -> float | None:
    """Parse SynthSeg QC CSV and return the quality score (0-1)."""
    if not os.path.isfile(csv_path):
        return None

    try:
        with open(csv_path, encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                for key, val in row.items():
                    if key in ("subject", ""):
                        continue
                    return round(float(val), 4)
    except Exception as e:
        log.warning("Failed to parse SynthSeg QC CSV: %s", e)

    return None
