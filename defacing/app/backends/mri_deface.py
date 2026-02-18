"""
mri_deface backend (FreeSurfer standalone).

mri_deface is a small standalone tool from FreeSurfer (~0.5 GB with atlas files).
It processes NIfTI volumes and is suitable for brain MRI defacing in production.

Modalities: MRI only.
License: Free, no restrictions.
Docker size: ~0.5 GB.

Installation (Dockerfile):
    ARG MRI_DEFACE_VERSION=1.22
    RUN curl -fsSL \
        https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/${MRI_DEFACE_VERSION}/linux/mri_deface \
        -o /usr/local/bin/mri_deface && chmod +x /usr/local/bin/mri_deface
    RUN curl -fsSL \
        https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/${MRI_DEFACE_VERSION}/talairach_mixed_with_skull.gca \
        -o /opt/mri_deface/talairach_mixed_with_skull.gca
    RUN curl -fsSL \
        https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/${MRI_DEFACE_VERSION}/face.gca \
        -o /opt/mri_deface/face.gca
"""

import logging
import shutil
import subprocess
import tempfile
from pathlib import Path

from .base import DefacingBackend
from .nifti_dicom import nifti_to_dicom, run_dcm2niix

log = logging.getLogger(__name__)


class MriDefaceBackend(DefacingBackend):
    def __init__(self, bin_path: str, brain_atlas: str, face_atlas: str, dcm2niix_bin: str):
        self._bin = bin_path
        self._brain_atlas = brain_atlas
        self._face_atlas = face_atlas
        self._dcm2niix = dcm2niix_bin

    @property
    def name(self) -> str:
        return "mri_deface"

    def available(self) -> bool:
        return (
            shutil.which(self._bin) is not None
            and Path(self._brain_atlas).exists()
            and Path(self._face_atlas).exists()
            and shutil.which(self._dcm2niix) is not None
        )

    def deface(self, series_dir: str, output_dir: str) -> list[str]:
        with tempfile.TemporaryDirectory(prefix="aegis_mri_deface_") as tmp:
            # 1. Convert DICOM → NIfTI
            nifti_path = run_dcm2niix(self._dcm2niix, series_dir, tmp)

            # 2. Run mri_deface
            defaced_path = Path(tmp) / "defaced.nii.gz"
            cmd = [
                self._bin,
                str(nifti_path),
                self._brain_atlas,
                self._face_atlas,
                str(defaced_path),
            ]
            log.info("Running mri_deface: %s", " ".join(cmd))
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
            if result.returncode != 0:
                raise RuntimeError(f"mri_deface failed: {result.stderr}")

            # 3. Inject defaced NIfTI pixel data back into DICOM
            return nifti_to_dicom(str(defaced_path), series_dir, output_dir)
