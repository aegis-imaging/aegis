"""TBM-SyN (Tensor-Based Morphometry with Symmetric Normalization) backend.

Measures longitudinal brain atrophy between paired T1w MRI scans (baseline +
follow-up) using ANTs symmetric diffeomorphic registration, log-Jacobian
determinant computation, and atlas-based ROI atrophy extraction.

References:
  - Vemuri P, Senjem ML, Gunter JL, et al. NeuroImage 2015;113:61-69
  - Avants BB et al. Medical Image Analysis 2008;12(1):26-41

Requires: ANTs binaries (antsRegistrationSyN.sh, CreateJacobianDeterminantImage,
N4BiasFieldCorrection, antsBrainExtraction.sh, antsApplyTransforms).
"""

from __future__ import annotations

import csv
import json
import logging
import os
import shutil
import subprocess
import time
from glob import glob

import nibabel as nib
import numpy as np

from app import config
from .base import LongitudinalBackend, AnalyticsResult
from .atlas_utils import (
    extract_roi_stats,
    find_atlas_nifti,
    get_ants_env,
    load_atlas_labels,
    run_brain_extraction,
    run_n4_bias_correction,
    warp_atlas_to_subject,
)

log = logging.getLogger(__name__)

# 31 AD-signature ROIs from AAL3 — regions characteristically affected in
# Alzheimer's disease. Used to compute the composite atrophy score.
# Bilateral pairs: entorhinal (not in AAL3 — use parahippocampal as proxy),
# hippocampus, amygdala, parahippocampal, fusiform, inferior temporal,
# middle temporal, superior temporal, temporal poles, angular gyrus,
# precuneus, posterior cingulate, inferior parietal, lateral occipital.
AD_SIGNATURE_ROIS: dict[int, str] = {
    41: "Hippocampus_L",
    42: "Hippocampus_R",
    43: "ParaHippocampal_L",
    44: "ParaHippocampal_R",
    45: "Amygdala_L",
    46: "Amygdala_R",
    55: "Occipital_Mid_L",
    56: "Occipital_Mid_R",
    59: "Fusiform_L",
    60: "Fusiform_R",
    65: "Parietal_Inf_L",
    66: "Parietal_Inf_R",
    69: "Angular_L",
    70: "Angular_R",
    71: "Precuneus_L",
    72: "Precuneus_R",
    39: "Cingulate_Post_L",
    40: "Cingulate_Post_R",
    85: "Temporal_Sup_L",
    86: "Temporal_Sup_R",
    87: "Temporal_Pole_Sup_L",
    88: "Temporal_Pole_Sup_R",
    89: "Temporal_Mid_L",
    90: "Temporal_Mid_R",
    91: "Temporal_Pole_Mid_L",
    92: "Temporal_Pole_Mid_R",
    93: "Temporal_Inf_L",
    94: "Temporal_Inf_R",
    67: "SupraMarginal_L",
    68: "SupraMarginal_R",
    35: "Cingulate_Ant_L",
}


def _find_t1w(bids_dir: str) -> str | None:
    """Find first T1w NIfTI in a BIDS directory."""
    files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
    return files[0] if files else None


class TBMSyNBackend(LongitudinalBackend):
    """TBM-SyN longitudinal atrophy measurement."""

    @property
    def name(self) -> str:
        return "tbm_syn"

    def available(self) -> bool:
        # Needs ANTs registration + Jacobian tools
        ants_reg = os.path.join(config.ANTSPATH, "antsRegistrationSyN.sh")
        jacobian = os.path.join(config.ANTSPATH, "CreateJacobianDeterminantImage")
        has_reg = os.path.isfile(ants_reg) or shutil.which("antsRegistrationSyN.sh") is not None
        has_jac = os.path.isfile(jacobian) or shutil.which("CreateJacobianDeterminantImage") is not None
        return has_reg and has_jac

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
        atlas_name = kwargs.get("atlas") or config.DEFAULT_ATLAS

        tbm_out = os.path.join(output_dir, "tbm_syn")
        os.makedirs(tbm_out, exist_ok=True)
        env = get_ants_env()

        # 1. Find T1w NIfTI in both BIDS directories
        baseline_t1w = _find_t1w(baseline_bids_dir)
        followup_t1w = _find_t1w(followup_bids_dir)

        if not baseline_t1w:
            return AnalyticsResult(
                tool=self.name, success=False,
                error="No T1w NIfTI found in baseline BIDS directory",
            )
        if not followup_t1w:
            return AnalyticsResult(
                tool=self.name, success=False,
                error="No T1w NIfTI found in follow-up BIDS directory",
            )

        log.info(
            "TBM-SyN: baseline=%s, followup=%s, interval=%.1f days",
            baseline_t1w, followup_t1w, scan_interval_days,
        )

        # 2. N4 bias correction on both
        bl_n4 = os.path.join(tbm_out, "baseline_n4.nii.gz")
        fu_n4 = os.path.join(tbm_out, "followup_n4.nii.gz")

        ok, err = run_n4_bias_correction(baseline_t1w, bl_n4, env=env)
        if not ok:
            log.warning("N4 failed on baseline (%s), using uncorrected", err)
            bl_n4 = baseline_t1w

        ok, err = run_n4_bias_correction(followup_t1w, fu_n4, env=env)
        if not ok:
            log.warning("N4 failed on follow-up (%s), using uncorrected", err)
            fu_n4 = followup_t1w

        # 3. Brain extraction on both
        bl_brain_prefix = os.path.join(tbm_out, "bl_brain_")
        fu_brain_prefix = os.path.join(tbm_out, "fu_brain_")

        ok, bl_brain, err = run_brain_extraction(bl_n4, bl_brain_prefix, env=env)
        if not ok:
            log.warning("Brain extraction failed on baseline (%s), using N4", err)
            bl_brain = bl_n4

        ok, fu_brain, err = run_brain_extraction(fu_n4, fu_brain_prefix, env=env)
        if not ok:
            log.warning("Brain extraction failed on follow-up (%s), using N4", err)
            fu_brain = fu_n4

        # 4. Symmetric diffeomorphic registration (baseline ← follow-up)
        reg_prefix = os.path.join(tbm_out, "syn_reg_")
        ok, err = self._run_syn_registration(
            bl_brain, fu_brain, reg_prefix, env, config.ANALYTICS_TIMEOUT
        )
        if not ok:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"SyN registration failed: {err}",
            )

        # 5. Compute log-Jacobian determinant
        warp_field = f"{reg_prefix}1Warp.nii.gz"
        jacobian_path = os.path.join(tbm_out, "log_jacobian.nii.gz")
        ok, err = self._compute_log_jacobian(warp_field, jacobian_path, env)
        if not ok:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"Jacobian computation failed: {err}",
            )

        # 6. Annualize the Jacobian
        annualized_path = os.path.join(tbm_out, "log_jacobian_annualized.nii.gz")
        self._annualize_jacobian(jacobian_path, annualized_path, scan_interval_days)

        # 7. Warp atlas to baseline subject space
        try:
            atlas_labels = load_atlas_labels(atlas_name)
            atlas_nifti = find_atlas_nifti(atlas_name)
        except FileNotFoundError as e:
            log.warning("Atlas not available (%s), skipping ROI extraction", e)
            duration = time.time() - start
            return AnalyticsResult(
                tool=self.name,
                success=True,
                duration_seconds=duration,
                outputs=[jacobian_path, annualized_path],
                metrics={
                    "scan_interval_days": scan_interval_days,
                    "atlas": None,
                    "note": "Jacobian computed but atlas unavailable for ROI extraction",
                },
            )

        template_brain = os.path.join(config.ANTS_TEMPLATE_DIR, "T_template0.nii.gz")
        warped_atlas = os.path.join(tbm_out, "warped_atlas.nii.gz")

        if os.path.isfile(template_brain):
            ok, err = warp_atlas_to_subject(
                atlas_path=atlas_nifti,
                template_brain=template_brain,
                subject_brain=bl_brain,
                output_path=warped_atlas,
                env=env,
            )
            if not ok:
                log.warning("Atlas warp failed (%s), skipping ROI extraction", err)
                duration = time.time() - start
                return AnalyticsResult(
                    tool=self.name,
                    success=True,
                    duration_seconds=duration,
                    outputs=[jacobian_path, annualized_path],
                    metrics={"scan_interval_days": scan_interval_days},
                )
        else:
            log.warning("Template not found, skipping atlas warp")
            duration = time.time() - start
            return AnalyticsResult(
                tool=self.name,
                success=True,
                duration_seconds=duration,
                outputs=[jacobian_path, annualized_path],
                metrics={"scan_interval_days": scan_interval_days},
            )

        # 8. Extract per-ROI mean annualized log-Jacobian
        roi_stats = extract_roi_stats(warped_atlas, annualized_path, atlas_labels)

        # Write atrophy CSV
        csv_path = os.path.join(tbm_out, "roi_atrophy.csv")
        if roi_stats:
            with open(csv_path, "w", newline="", encoding="utf-8") as f:
                writer = csv.DictWriter(f, fieldnames=roi_stats[0].keys())
                writer.writeheader()
                writer.writerows(roi_stats)

        # 9. Compute AD-signature composite
        ad_composite = self._compute_ad_composite(roi_stats)

        composite_path = os.path.join(tbm_out, "ad_composite.json")
        with open(composite_path, "w", encoding="utf-8") as f:
            json.dump(ad_composite, f, indent=2)

        duration = time.time() - start
        outputs = [
            jacobian_path,
            annualized_path,
            warped_atlas,
            csv_path,
            composite_path,
        ]

        # Per-ROI atrophy stats for structured DB storage
        roi_atrophy_list = [
            {
                "roi_name": r["roi_name"],
                "roi_id": r.get("roi_id", 0),
                "mean": r.get("mean", 0),
                "median": r.get("median", 0),
                "std": r.get("std", 0),
                "voxel_count": r.get("voxel_count", 0),
            }
            for r in roi_stats
        ]
        metrics = {
            "scan_interval_days": scan_interval_days,
            "atlas": atlas_name,
            "roi_count": len(roi_stats),
            "ad_composite_mean": ad_composite.get("composite_mean"),
            "ad_composite_roi_count": ad_composite.get("contributing_roi_count"),
            "roi_atrophy": roi_atrophy_list,
        }

        log.info(
            "TBM-SyN complete: %d ROIs, AD composite=%.6f, %.1fs",
            len(roi_stats),
            ad_composite.get("composite_mean", 0),
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )

    def _run_syn_registration(
        self,
        fixed: str,
        moving: str,
        prefix: str,
        env: dict[str, str],
        timeout: int,
    ) -> tuple[bool, str]:
        """Run ANTs SyN registration between baseline (fixed) and follow-up (moving)."""
        try:
            result = subprocess.run(
                [
                    "antsRegistrationSyN.sh",
                    "-d", "3",
                    "-f", fixed,
                    "-m", moving,
                    "-t", "so",  # SyN only (no rigid/affine pre-alignment)
                    "-o", prefix,
                ],
                capture_output=True,
                text=True,
                timeout=timeout,
                env=env,
            )
        except FileNotFoundError:
            return False, "antsRegistrationSyN.sh not found"
        except subprocess.TimeoutExpired:
            return False, f"SyN registration timed out after {timeout}s"

        if result.returncode != 0:
            return False, result.stderr[-500:] if result.stderr else "Registration failed"

        warp = f"{prefix}1Warp.nii.gz"
        if not os.path.isfile(warp):
            return False, f"Warp field not produced: {warp}"

        return True, ""

    def _compute_log_jacobian(
        self,
        warp_field: str,
        output_path: str,
        env: dict[str, str],
    ) -> tuple[bool, str]:
        """Compute log-Jacobian determinant from warp field."""
        try:
            result = subprocess.run(
                [
                    "CreateJacobianDeterminantImage",
                    "3",           # dimension
                    warp_field,    # input warp
                    output_path,   # output
                    "1",           # doLogJacobian = true
                    "1",           # useGeometric = true
                ],
                capture_output=True,
                text=True,
                timeout=600,
                env=env,
            )
        except FileNotFoundError:
            return False, "CreateJacobianDeterminantImage not found"
        except subprocess.TimeoutExpired:
            return False, "Jacobian computation timed out"

        if result.returncode != 0:
            return False, result.stderr[-500:] if result.stderr else "Jacobian failed"

        if not os.path.isfile(output_path):
            return False, f"Jacobian output not found: {output_path}"

        return True, ""

    def _annualize_jacobian(
        self,
        jacobian_path: str,
        output_path: str,
        scan_interval_days: float,
    ) -> None:
        """Annualize log-Jacobian by dividing by scan interval in years."""
        img = nib.load(jacobian_path)
        data = np.asarray(img.dataobj, dtype=np.float64)

        years = scan_interval_days / 365.25
        if years > 0:
            annualized = data / years
        else:
            log.warning("Scan interval <= 0 days, skipping annualization")
            annualized = data

        out_img = nib.Nifti1Image(annualized, img.affine, img.header)
        nib.save(out_img, output_path)

    def _compute_ad_composite(self, roi_stats: list[dict]) -> dict:
        """Compute voxel-weighted mean atrophy across AD-signature ROIs."""
        total_voxels = 0
        weighted_sum = 0.0
        contributing = []

        for roi in roi_stats:
            if roi["roi_id"] in AD_SIGNATURE_ROIS:
                voxels = roi["voxel_count"]
                total_voxels += voxels
                weighted_sum += roi["mean"] * voxels
                contributing.append({
                    "roi_id": roi["roi_id"],
                    "roi_name": roi["roi_name"],
                    "mean_atrophy": roi["mean"],
                    "voxel_count": voxels,
                })

        composite_mean = weighted_sum / total_voxels if total_voxels > 0 else 0.0

        return {
            "composite_mean": round(composite_mean, 6),
            "contributing_roi_count": len(contributing),
            "total_voxels": total_voxels,
            "contributing_rois": contributing,
            "description": "Voxel-weighted mean annualized log-Jacobian across AD-signature ROIs",
        }
