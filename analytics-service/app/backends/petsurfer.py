"""PETSurfer backend — integrated PET-MRI analysis within FreeSurfer.

PETSurfer provides tools for end-to-end PET analysis including motion
correction, PET-MRI registration, partial volume correction (PVC),
reference region kinetic modeling, and ROI-based quantification using
the FreeSurfer segmentation atlas (aparc+aseg).

Requires:
  - ``mri_gtmpvc`` binary (included in FreeSurfer 7.x+)
  - FreeSurfer license file

References:
  Greve DN et al. "Cortical surface-based analysis reduces bias and
  variance in kinetic modeling of brain PET data." NeuroImage, 2014.
  DOI: 10.1016/j.neuroimage.2013.12.021

  https://surfer.nmr.mgh.harvard.edu/fswiki/PetSurfer
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
from .seg_utils import find_pet_nifti, find_t1w_nifti

log = logging.getLogger(__name__)

# Default FreeSurfer label IDs for reference regions (cerebellum cortex).
_DEFAULT_REF_LABELS = "8 47"

# GTM output ROI names follow FreeSurfer's aparc+aseg nomenclature.


class PETSurferBackend(AnalyticsBackend):
    """PET quantification with partial volume correction via PETSurfer."""

    @property
    def name(self) -> str:
        return "petsurfer"

    def available(self) -> bool:
        if not shutil.which("mri_gtmpvc"):
            return False
        # PETSurfer is part of FreeSurfer — check license
        license_path = getattr(config, "FS_LICENSE", "/app/secrets/freesurfer.license")
        return os.path.isfile(license_path)

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "petsurfer")
        os.makedirs(out_dir, exist_ok=True)

        # Find PET NIfTI
        pet_path = find_pet_nifti(bids_dir)
        if not pet_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No PET NIfTI files found in BIDS directory",
            )

        log.info("PETSurfer: using PET=%s", pet_path)

        # Check for a FreeSurfer recon-all output (aparc+aseg)
        seg_path = _find_aparc_aseg(output_dir)

        # PSF FWHM and reference region config
        psf_fwhm = getattr(config, "PETSURFER_PSF_FWHM", "6")
        km_ref = getattr(config, "PETSURFER_KM_REF", _DEFAULT_REF_LABELS)

        # Build mri_gtmpvc command
        cmd = [
            "mri_gtmpvc",
            "--i", pet_path,
            "--psf", psf_fwhm,
            "--default-seg-merge",
            "--auto-mask", "1", "0.01",
            "--km-ref", *km_ref.split(),
            "--o", out_dir,
        ]
        if seg_path:
            cmd.extend(["--seg", seg_path])

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
                error="mri_gtmpvc binary not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"PETSurfer timed out after {config.ANALYTICS_TIMEOUT}s",
            )

        if result.returncode != 0:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=result.stderr[-500:] if result.stderr else "mri_gtmpvc failed",
            )

        # Parse GTM stats
        gtm_stats_path = os.path.join(out_dir, "gtm.stats.dat")
        roi_suvr = _parse_gtm_stats(gtm_stats_path)

        metrics: dict = {
            "atlas": "gtm",
            "roi_count": len(roi_suvr),
            "roi_suvr": roi_suvr,
            "psf_fwhm": psf_fwhm,
        }

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [summary_path]
        if os.path.isfile(gtm_stats_path):
            outputs.append(gtm_stats_path)

        log.info(
            "PETSurfer complete for %s: %d ROIs in %.1fs",
            study_uid, len(roi_suvr), duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _find_aparc_aseg(output_dir: str) -> str | None:
    """Look for an existing FreeSurfer aparc+aseg segmentation."""
    # Check common locations relative to the analytics output root
    parent = os.path.dirname(output_dir)
    for candidate in [
        os.path.join(parent, "freesurfer", "mri", "aparc+aseg.mgz"),
        os.path.join(parent, "freesurfer", "mri", "aparc+aseg.nii.gz"),
    ]:
        if os.path.isfile(candidate):
            return candidate
    return None


def _parse_gtm_stats(stats_path: str) -> dict[str, float]:
    """Parse mri_gtmpvc gtm.stats.dat output into {roi_name: suvr}.

    The file format is whitespace-delimited with columns:
      ROI_Index  ROI_Name  SUVR  ...
    """
    roi_suvr: dict[str, float] = {}
    if not os.path.isfile(stats_path):
        return roi_suvr

    try:
        with open(stats_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                parts = line.split()
                if len(parts) >= 3:
                    roi_name = parts[1]
                    try:
                        roi_suvr[roi_name] = round(float(parts[2]), 6)
                    except ValueError:
                        continue
    except Exception as e:
        log.warning("Failed to parse GTM stats: %s", e)

    return roi_suvr
