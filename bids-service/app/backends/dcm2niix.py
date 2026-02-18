"""
BIDS conversion backend using dcm2niix.

Converts DICOM files to NIfTI with BIDS-compliant directory structure
and JSON sidecar metadata.
"""

import hashlib
import json
import logging
import os
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

import pydicom
from pydicom.errors import InvalidDicomError

from .base import BidsBackend, ConversionResult
from ..config import cfg

log = logging.getLogger(__name__)

# Map DICOM series info to BIDS datatype and suffix.
# Keys are lowercase patterns matched against ProtocolName or SeriesDescription.
_BIDS_SUFFIX_MAP: list[tuple[str, str, str]] = [
    # (pattern, datatype, suffix)
    # Anatomical
    (r"t1w|t1\b|mprage|spgr|bravo|ir-?fspgr", "anat", "T1w"),
    (r"t2w|t2\b|t2_tse|t2_fse", "anat", "T2w"),
    (r"flair", "anat", "FLAIR"),
    (r"t2star|t2\*|swi", "anat", "T2star"),
    (r"pd\b|proton.?density", "anat", "PD"),
    # Functional
    (r"bold|fmri|func|resting.?state|task", "func", "bold"),
    # Diffusion
    (r"dti|dwi|diff|tensor", "dwi", "dwi"),
    # Perfusion
    (r"asl|perfusion|pcasl|pasl", "perf", "asl"),
    # CT
    (r"\bct\b|computed.?tomography", "ct", "CT"),
    # PET
    (r"\bpet\b|fdg|amyloid|tau|psma", "pet", "pet"),
]


def _subject_label(study_uid: str) -> str:
    """Generate a BIDS subject label from StudyInstanceUID (first 8 chars of SHA-256)."""
    return hashlib.sha256(study_uid.encode()).hexdigest()[:8]


def _classify_series(protocol: str, description: str, modality: str) -> tuple[str, str]:
    """
    Determine BIDS datatype and suffix from DICOM series metadata.
    Returns (datatype, suffix) or falls back to ("anat", "unknown").
    """
    text = f"{protocol} {description}".lower()

    for pattern, datatype, suffix in _BIDS_SUFFIX_MAP:
        if re.search(pattern, text):
            return datatype, suffix

    # Fallback by modality
    mod = modality.upper()
    if mod in ("CT",):
        return "ct", "CT"
    if mod in ("PT", "PET"):
        return "pet", "pet"
    if mod in ("MR", "MRI"):
        return "anat", "T1w"

    return "anat", "unknown"


def _sanitize_filename(s: str) -> str:
    """Remove characters invalid in BIDS filenames."""
    return re.sub(r"[^a-zA-Z0-9]", "", s)


class Dcm2niixBackend(BidsBackend):
    """Local dev backend: dcm2niix for DICOM→NIfTI + BIDS structure generation."""

    @property
    def name(self) -> str:
        return "dcm2niix"

    def available(self) -> bool:
        try:
            result = subprocess.run(
                [cfg.dcm2niix_bin, "--version"],
                capture_output=True, text=True, timeout=10,
            )
            return result.returncode == 0
        except (FileNotFoundError, subprocess.TimeoutExpired):
            return False

    def convert(self, input_dir: str, output_dir: str, study_uid: str) -> ConversionResult:
        result = ConversionResult()
        sub_label = _subject_label(study_uid)
        sub_dir = f"sub-{sub_label}"

        # Group DICOM files by SeriesInstanceUID
        series_groups = self._group_by_series(input_dir, result)
        if not series_groups:
            result.warnings.append("No valid DICOM files found in input directory")
            return result

        os.makedirs(output_dir, exist_ok=True)

        # Track suffixes to avoid collisions (e.g. two T1w series → T1w, run-2_T1w)
        suffix_counts: dict[str, int] = {}

        for series_uid, info in series_groups.items():
            datatype, suffix = _classify_series(
                info["protocol"], info["description"], info["modality"],
            )

            # Handle duplicate suffixes with run- entity
            key = f"{datatype}/{suffix}"
            suffix_counts[key] = suffix_counts.get(key, 0) + 1
            run_label = ""
            if suffix_counts[key] > 1:
                run_label = f"run-{suffix_counts[key]:02d}_"

            # Create output directory: output_dir/sub-<hash>/<datatype>/
            out_datatype_dir = os.path.join(output_dir, sub_dir, datatype)
            os.makedirs(out_datatype_dir, exist_ok=True)

            # Run dcm2niix on a temporary directory containing only this series
            bids_prefix = f"sub-{sub_label}_{run_label}{suffix}"
            self._convert_series(
                info["files"], out_datatype_dir, bids_prefix, result,
            )

        # Generate BIDS metadata files
        self._write_dataset_description(output_dir)
        result.output_files.append(os.path.join(output_dir, "dataset_description.json"))

        self._write_participants_tsv(output_dir, sub_label, series_groups)
        result.output_files.append(os.path.join(output_dir, "participants.tsv"))

        return result

    def _group_by_series(
        self, input_dir: str, result: ConversionResult,
    ) -> dict[str, dict]:
        """Group DICOM files by SeriesInstanceUID and extract metadata."""
        groups: dict[str, dict] = {}

        for filename in sorted(os.listdir(input_dir)):
            filepath = os.path.join(input_dir, filename)
            if not os.path.isfile(filepath):
                continue
            if not filename.lower().endswith(".dcm"):
                continue

            try:
                ds = pydicom.dcmread(filepath, stop_before_pixels=True)
            except (InvalidDicomError, Exception) as e:
                result.warnings.append(f"Skipping {filename}: {e}")
                continue

            series_uid = getattr(ds, "SeriesInstanceUID", "unknown")
            if series_uid not in groups:
                groups[series_uid] = {
                    "protocol": str(getattr(ds, "ProtocolName", "")),
                    "description": str(getattr(ds, "SeriesDescription", "")),
                    "modality": str(getattr(ds, "Modality", "")),
                    "patient_age": str(getattr(ds, "PatientAge", "")),
                    "patient_sex": str(getattr(ds, "PatientSex", "")),
                    "files": [],
                }
            groups[series_uid]["files"].append(filepath)

        return groups

    def _convert_series(
        self,
        dicom_files: list[str],
        out_dir: str,
        bids_prefix: str,
        result: ConversionResult,
    ) -> None:
        """Run dcm2niix on a set of DICOM files belonging to one series."""
        with tempfile.TemporaryDirectory() as tmp_in:
            # Symlink or copy files into a temp dir for dcm2niix
            for f in dicom_files:
                dst = os.path.join(tmp_in, os.path.basename(f))
                try:
                    os.symlink(f, dst)
                except OSError:
                    shutil.copy2(f, dst)

            cmd = [
                cfg.dcm2niix_bin,
                "-z", "y",       # gzip output
                "-ba", "y",      # BIDS sidecar JSON
                "-f", bids_prefix,
                "-o", out_dir,
                tmp_in,
            ]
            log.info("Running: %s", " ".join(cmd))

            try:
                proc = subprocess.run(
                    cmd, capture_output=True, text=True, timeout=300,
                )
            except subprocess.TimeoutExpired:
                result.warnings.append(f"dcm2niix timed out for {bids_prefix}")
                return

            if proc.returncode != 0:
                result.warnings.append(
                    f"dcm2niix failed for {bids_prefix}: {proc.stderr.strip()}"
                )
                return

            # Collect output files
            for entry in os.listdir(out_dir):
                full = os.path.join(out_dir, entry)
                if entry.startswith(bids_prefix) and os.path.isfile(full):
                    result.output_files.append(full)

    def _write_dataset_description(self, output_dir: str) -> None:
        """Write BIDS dataset_description.json."""
        desc = {
            "Name": "AEGIS Export",
            "BIDSVersion": "1.9.0",
            "GeneratedBy": [
                {
                    "Name": "AEGIS BIDS Conversion Service",
                    "Version": "1.0.0",
                    "Description": "Automated DICOM to NIfTI/BIDS conversion",
                }
            ],
            "License": "",
        }
        path = os.path.join(output_dir, "dataset_description.json")
        with open(path, "w") as f:
            json.dump(desc, f, indent=2)

    def _write_participants_tsv(
        self, output_dir: str, sub_label: str, series_groups: dict,
    ) -> None:
        """Write BIDS participants.tsv with minimal demographics (if available)."""
        age = ""
        sex = ""
        for info in series_groups.values():
            if info.get("patient_age") and not age:
                age = info["patient_age"]
            if info.get("patient_sex") and not sex:
                sex = info["patient_sex"]
            if age and sex:
                break

        path = os.path.join(output_dir, "participants.tsv")
        with open(path, "w") as f:
            f.write("participant_id\tage\tsex\n")
            f.write(f"sub-{sub_label}\t{age}\t{sex}\n")
