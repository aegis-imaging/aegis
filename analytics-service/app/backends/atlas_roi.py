"""Atlas ROI labeling backend — warps an anatomical atlas to subject space
and extracts per-ROI volumetric statistics.

Supports AAL3, MCALT/ADIR122, and any atlas with a NIfTI volume + labels CSV
placed in ``{ATLAS_DIR}/{atlas_name}/``.

Requires: ANTs binaries (antsRegistrationSyN.sh, antsApplyTransforms,
N4BiasFieldCorrection, antsBrainExtraction.sh).
"""

from __future__ import annotations

import csv
import json
import logging
import os
import shutil
import time
from glob import glob

from app import config
from .base import AnalyticsBackend, AnalyticsResult
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


class AtlasROIBackend(AnalyticsBackend):
    """Atlas-based ROI labeling and volumetric statistics."""

    @property
    def name(self) -> str:
        return "atlas_roi"

    def available(self) -> bool:
        # Requires ANTs registration tools
        ants_reg = os.path.join(config.ANTSPATH, "antsRegistrationSyN.sh")
        if os.path.isfile(ants_reg) or shutil.which("antsRegistrationSyN.sh"):
            return True
        return False

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        atlas_name = kwargs.get("atlas") or config.DEFAULT_ATLAS

        roi_out = os.path.join(output_dir, "atlas_roi")
        os.makedirs(roi_out, exist_ok=True)
        env = get_ants_env()

        # 1. Find T1w NIfTI in BIDS directory
        t1w_files = glob(os.path.join(bids_dir, "**", "*T1w*.nii*"), recursive=True)
        if not t1w_files:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )
        t1w = t1w_files[0]
        log.info("Atlas ROI: using T1w %s with atlas %s", t1w, atlas_name)

        # 2. Load atlas labels and locate atlas NIfTI
        try:
            label_map = load_atlas_labels(atlas_name)
        except FileNotFoundError as e:
            return AnalyticsResult(
                tool=self.name, success=False, error=str(e),
            )

        try:
            atlas_nifti = find_atlas_nifti(atlas_name)
        except FileNotFoundError as e:
            return AnalyticsResult(
                tool=self.name, success=False, error=str(e),
            )

        # 3. N4 bias correction
        n4_out = os.path.join(roi_out, "t1w_n4.nii.gz")
        ok, err = run_n4_bias_correction(t1w, n4_out, env=env)
        if not ok:
            log.warning("N4 failed (%s), proceeding with uncorrected T1w", err)
            n4_out = t1w

        # 4. Brain extraction
        brain_prefix = os.path.join(roi_out, "brain_")
        ok, brain_path, err = run_brain_extraction(n4_out, brain_prefix, env=env)
        if not ok:
            log.warning("Brain extraction failed (%s), using N4 output directly", err)
            brain_path = n4_out

        # 5. Template → subject registration + atlas warp
        template_brain = os.path.join(
            config.ANTS_TEMPLATE_DIR, "T_template0.nii.gz"
        )
        if not os.path.isfile(template_brain):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"ANTs template not found: {template_brain}",
            )

        warped_atlas = os.path.join(roi_out, "warped_atlas.nii.gz")
        ok, err = warp_atlas_to_subject(
            atlas_path=atlas_nifti,
            template_brain=template_brain,
            subject_brain=brain_path,
            output_path=warped_atlas,
            env=env,
            timeout=config.ANALYTICS_TIMEOUT,
        )
        if not ok:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"Atlas warp failed: {err}",
            )

        # 6. Extract per-ROI statistics
        roi_stats = extract_roi_stats(warped_atlas, brain_path, label_map)

        # Write CSV output
        csv_path = os.path.join(roi_out, "roi_stats.csv")
        if roi_stats:
            with open(csv_path, "w", newline="", encoding="utf-8") as f:
                writer = csv.DictWriter(f, fieldnames=roi_stats[0].keys())
                writer.writeheader()
                writer.writerows(roi_stats)

        # Write summary JSON
        summary = {
            "atlas": atlas_name,
            "roi_count": len(roi_stats),
            "total_labeled_voxels": sum(r["voxel_count"] for r in roi_stats),
        }
        summary_path = os.path.join(roi_out, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(summary, f, indent=2)

        duration = time.time() - start
        outputs = [warped_atlas, csv_path, summary_path]

        # Build metrics dict with top-level counts and per-ROI volumes
        roi_volumes = {r["roi_name"]: r["volume_mm3"] for r in roi_stats}
        metrics = {
            "atlas": atlas_name,
            "roi_count": len(roi_stats),
            "roi_volumes": roi_volumes,
        }

        log.info(
            "Atlas ROI complete for %s: %d ROIs in %.1fs",
            study_uid,
            len(roi_stats),
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )
