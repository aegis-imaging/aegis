"""
mri_reface backend.

mri_reface replaces face regions with an average face (rather than zeroing them),
which improves compatibility with downstream neuroimaging analysis tools that
expect an intact head shape.

Supports: T1/T2/FLAIR/T2*/ASL MRI, Amyloid/Tau/FDG PET, CT.

License: Non-commercial research use only.
         Contact authors for commercial licensing.
         See: https://www.nitrc.org/projects/mri_reface

Dependencies: MATLAB Runtime (free), ANTs, NiftyReg.
Docker size: ~2-4 GB.

Installation (Dockerfile):
    # Download compiled executable from NITRC
    # https://www.nitrc.org/projects/mri_reface
    # Install MATLAB Runtime v912 (R2021b), ANTs, NiftyReg
    # See project page for full instructions.
"""

import logging
import shutil
import subprocess
import tempfile
from pathlib import Path

from .base import DefacingBackend
from .nifti_dicom import nifti_to_dicom, run_dcm2niix

log = logging.getLogger(__name__)


class MriRefaceBackend(DefacingBackend):
    def __init__(self, bin_path: str, dcm2niix_bin: str):
        self._bin = bin_path
        self._dcm2niix = dcm2niix_bin

    @property
    def name(self) -> str:
        return "mri_reface"

    def available(self) -> bool:
        return (
            shutil.which(self._bin) is not None
            and shutil.which(self._dcm2niix) is not None
        )

    def deface(self, series_dir: str, output_dir: str) -> list[str]:
        with tempfile.TemporaryDirectory(prefix="aegis_mri_reface_") as tmp:
            # 1. Convert DICOM → NIfTI
            nifti_path = run_dcm2niix(self._dcm2niix, series_dir, tmp)

            # 2. Run mri_reface
            defaced_path = Path(tmp) / "refaced.nii.gz"
            cmd = [
                self._bin,
                "-i", str(nifti_path),
                "-o", str(defaced_path),
            ]
            log.info("Running mri_reface: %s", " ".join(cmd))
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
            if result.returncode != 0:
                raise RuntimeError(f"mri_reface failed: {result.stderr}")

            # 3. Inject refaced NIfTI pixel data back into DICOM
            return nifti_to_dicom(str(defaced_path), series_dir, output_dir)
