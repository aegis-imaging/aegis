"""SCT (Spinal Cord Toolbox) backend — comprehensive spinal cord analysis.

Wraps SCT CLI tools in a multi-step pipeline:
  1. sct_deepseg_sc     — spinal cord segmentation
  2. sct_label_vertebrae — automatic vertebral labeling
  3. sct_process_segmentation — CSA per vertebral level
  4. sct_compute_compression  — compression metrics (aMCC, aSCOR)
  5. sct_dmri_compute_dti     — DTI metrics (if DWI data present)

Each step is a subprocess call; failures are logged but non-fatal
(partial results are returned).

Requires:
  - Spinal Cord Toolbox installed (https://spinalcordtoolbox.com/)
  - NIfTI input from BIDS-converted spine MRI

References:
  De Leener B et al. "SCT: Spinal Cord Toolbox, an open-source software
  for processing spinal cord MRI data." NeuroImage, 2017.
  DOI: 10.1016/j.neuroimage.2016.10.009
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
from pathlib import Path

from .base import SctBackend, SctResult
from ..config import cfg

log = logging.getLogger(__name__)


class SpinalCordToolboxBackend(SctBackend):
    """Spinal cord analysis via SCT CLI tools."""

    @property
    def name(self) -> str:
        return "sct"

    def available(self) -> bool:
        return shutil.which("sct_deepseg_sc") is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> SctResult:
        start = time.time()
        os.makedirs(output_dir, exist_ok=True)

        contrast = kwargs.get("contrast", cfg.contrast)

        # Find input NIfTI (prefer spine/T2w)
        nifti_path = _find_spine_nifti(bids_dir)
        if not nifti_path:
            return SctResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("SCT: using %s (contrast=%s)", nifti_path, contrast)

        steps_completed: list[str] = []
        outputs: list[str] = []
        metrics: dict = {
            "atlas": "sct",
            "contrast": contrast,
            "input_file": nifti_path,
        }

        # Step 1: Spinal cord segmentation
        seg_sc = os.path.join(output_dir, "seg_sc.nii.gz")
        ok = _run_sct(
            ["sct_deepseg_sc", "-i", nifti_path, "-c", contrast, "-o", seg_sc],
            "deepseg_sc",
            output_dir,
        )
        if ok and os.path.isfile(seg_sc):
            steps_completed.append("deepseg_sc")
            outputs.append(seg_sc)
        else:
            return SctResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                outputs=outputs,
                metrics=metrics,
                error="sct_deepseg_sc failed — cannot continue pipeline",
            )

        # Step 2: Vertebral labeling
        labels_vert = os.path.join(output_dir, "labels_vert.nii.gz")
        ok = _run_sct(
            [
                "sct_label_vertebrae",
                "-i", nifti_path,
                "-s", seg_sc,
                "-c", contrast,
                "-ofolder", output_dir,
            ],
            "label_vertebrae",
            output_dir,
        )
        # sct_label_vertebrae outputs to <input_basename>_labeled.nii.gz
        labeled_files = glob(os.path.join(output_dir, "*_labeled.nii.gz"))
        if ok and labeled_files:
            labels_vert = labeled_files[0]
            steps_completed.append("label_vertebrae")
            outputs.append(labels_vert)

            # Extract detected vertebral levels
            vert_levels = _parse_vertebral_levels(labels_vert)
            if vert_levels:
                metrics["vertebral_levels_detected"] = vert_levels

        # Step 3: CSA per vertebral level
        csa_csv = os.path.join(output_dir, "csa_perlevel.csv")
        csa_args = [
            "sct_process_segmentation",
            "-i", seg_sc,
            "-o", csa_csv,
            "-perslice", "0",
        ]
        if labeled_files:
            csa_args.extend(["-vertfile", labels_vert, "-perlevel", "1"])

        ok = _run_sct(csa_args, "process_segmentation", output_dir)
        if ok and os.path.isfile(csa_csv):
            steps_completed.append("process_segmentation")
            outputs.append(csa_csv)

            csa_data = _parse_csa_csv(csa_csv)
            if csa_data:
                metrics["csa_per_level"] = csa_data
                values = [v for v in csa_data.values() if isinstance(v, (int, float))]
                if values:
                    metrics["mean_csa_mm2"] = round(sum(values) / len(values), 2)

        # Step 4: Compression metrics (requires vertebral labels)
        if labeled_files:
            compression_csv = os.path.join(output_dir, "compression.csv")
            ok = _run_sct(
                [
                    "sct_compute_compression",
                    "-i", seg_sc,
                    "-vertfile", labels_vert,
                    "-o", compression_csv,
                ],
                "compute_compression",
                output_dir,
            )
            if ok and os.path.isfile(compression_csv):
                steps_completed.append("compute_compression")
                outputs.append(compression_csv)

                compression = _parse_compression_csv(compression_csv)
                if compression:
                    metrics["compression"] = compression

        # Step 5: DTI (optional — only if DWI data present)
        dwi_path = _find_dwi_nifti(bids_dir)
        if dwi_path:
            bval = dwi_path.replace(".nii.gz", ".bval").replace(".nii", ".bval")
            bvec = dwi_path.replace(".nii.gz", ".bvec").replace(".nii", ".bvec")
            if os.path.isfile(bval) and os.path.isfile(bvec):
                dti_dir = os.path.join(output_dir, "dti")
                os.makedirs(dti_dir, exist_ok=True)
                ok = _run_sct(
                    [
                        "sct_dmri_compute_dti",
                        "-i", dwi_path,
                        "-bval", bval,
                        "-bvec", bvec,
                        "-o", dti_dir,
                    ],
                    "dmri_compute_dti",
                    output_dir,
                )
                if ok:
                    steps_completed.append("dmri_compute_dti")
                    dti_files = glob(os.path.join(dti_dir, "*.nii.gz"))
                    outputs.extend(dti_files)
                    metrics["dti_outputs"] = [os.path.basename(f) for f in dti_files]

        metrics["steps_completed"] = steps_completed

        # Write summary JSON
        summary_path = os.path.join(output_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)
        outputs.append(summary_path)

        duration = time.time() - start
        log.info(
            "SCT complete for %s: %d steps in %.1fs",
            study_uid,
            len(steps_completed),
            duration,
        )

        return SctResult(
            tool=self.name,
            success=len(steps_completed) > 0,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _run_sct(cmd: list[str], step_name: str, work_dir: str) -> bool:
    """Run an SCT CLI command. Returns True on success."""
    log.info("SCT step %s: %s", step_name, " ".join(cmd))
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=cfg.timeout,
            cwd=work_dir,
        )
        if result.returncode != 0:
            log.warning(
                "SCT step %s failed (rc=%d): %s",
                step_name,
                result.returncode,
                result.stderr[:500] if result.stderr else "(no stderr)",
            )
            return False
        return True
    except subprocess.TimeoutExpired:
        log.error("SCT step %s timed out after %ds", step_name, cfg.timeout)
        return False
    except FileNotFoundError:
        log.error("SCT step %s: command not found: %s", step_name, cmd[0])
        return False


def _find_spine_nifti(bids_dir: str) -> str | None:
    """Find a spine NIfTI in a BIDS directory.

    Priority: files with "spine"/"spinal" → T2w → T1w → any NIfTI.
    """
    all_niis = sorted(glob(os.path.join(bids_dir, "**", "*.nii*"), recursive=True))
    if not all_niis:
        return None

    for path in all_niis:
        base = os.path.basename(path).lower()
        if "spine" in base or "spinal" in base:
            return path

    for path in all_niis:
        if "T2w" in os.path.basename(path):
            return path

    for path in all_niis:
        if "T1w" in os.path.basename(path):
            return path

    return all_niis[0]


def _find_dwi_nifti(bids_dir: str) -> str | None:
    """Find a DWI NIfTI in a BIDS directory."""
    candidates = sorted(glob(os.path.join(bids_dir, "**", "*dwi*.nii*"), recursive=True))
    if not candidates:
        candidates = sorted(glob(os.path.join(bids_dir, "**/dwi/*.nii*"), recursive=True))
    return candidates[0] if candidates else None


def _parse_csa_csv(csv_path: str) -> dict[str, float]:
    """Parse SCT CSA per-level CSV into a dict of level → CSA (mm2)."""
    result: dict[str, float] = {}
    try:
        with open(csv_path, encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                level = row.get("VertLevel", row.get("Slice (I->S)", ""))
                csa = row.get("MEAN(area)", row.get("CSA (mm^2)", ""))
                if level and csa:
                    try:
                        result[str(level)] = round(float(csa), 2)
                    except ValueError:
                        pass
    except (OSError, csv.Error) as e:
        log.warning("Failed to parse CSA CSV %s: %s", csv_path, e)
    return result


def _parse_compression_csv(csv_path: str) -> dict[str, float]:
    """Parse SCT compression metrics CSV."""
    result: dict[str, float] = {}
    try:
        with open(csv_path, encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                for key in ("aMCC", "aSCOR", "MSCC", "ratio_AP"):
                    if key in row and row[key]:
                        try:
                            result[key] = round(float(row[key]), 4)
                        except ValueError:
                            pass
    except (OSError, csv.Error) as e:
        log.warning("Failed to parse compression CSV %s: %s", csv_path, e)
    return result


def _parse_vertebral_levels(labels_path: str) -> list[str]:
    """Extract vertebral level names from a labeled NIfTI."""
    try:
        import nibabel as nib
        import numpy as np

        img = nib.load(labels_path)
        data = np.asarray(img.dataobj, dtype=np.int32)
        unique = sorted(int(x) for x in np.unique(data) if x != 0)

        _VERT_NAMES = {
            1: "C1", 2: "C2", 3: "C3", 4: "C4", 5: "C5", 6: "C6", 7: "C7",
            8: "T1", 9: "T2", 10: "T3", 11: "T4", 12: "T5", 13: "T6",
            14: "T7", 15: "T8", 16: "T9", 17: "T10", 18: "T11", 19: "T12",
            20: "L1", 21: "L2", 22: "L3", 23: "L4", 24: "L5",
            25: "S1",
        }
        return [_VERT_NAMES.get(v, f"V{v}") for v in unique]
    except Exception as e:
        log.warning("Failed to parse vertebral levels from %s: %s", labels_path, e)
        return []
