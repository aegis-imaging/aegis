"""
DeepDefacer backend (deep learning, TensorFlow/Keras 3D U-Net).

DeepDefacer uses a modified 3D U-Net (92% fewer parameters than original)
trained on IXI and ICBM brain data to segment and remove facial features.
Significantly faster than registration-based tools (~1-2 min per volume).

Modalities: MRI only.
License: Research-friendly (no commercial restrictions documented).
Docker size: ~500 MB additional (TensorFlow).

Installation:
    pip install deepdefacer          # CPU only
    pip install deepdefacer[gpu]     # GPU support (requires CUDA)

Reference:
    Khazane A, Hoachuck J, Gorgolewski KJ, Poldrack RA.
    "DeepDefacer: Automatic Removal of Facial Features via U-Net Image
    Segmentation." arXiv:2205.15536 (2022).
    https://doi.org/10.48550/arXiv.2205.15536
"""

import logging
import shutil
import subprocess
import tempfile
from pathlib import Path

from .base import DefacingBackend
from .nifti_dicom import nifti_to_dicom, run_dcm2niix

log = logging.getLogger(__name__)


class DeepDefacerBackend(DefacingBackend):
    def __init__(self, dcm2niix_bin: str):
        self._dcm2niix = dcm2niix_bin

    @property
    def name(self) -> str:
        return "deepdefacer"

    def available(self) -> bool:
        try:
            import deepdefacer  # noqa: F401
        except ImportError:
            return False
        return shutil.which(self._dcm2niix) is not None

    def deface(self, series_dir: str, output_dir: str) -> list[str]:
        with tempfile.TemporaryDirectory(prefix="aegis_deepdefacer_") as tmp:
            # 1. Convert DICOM -> NIfTI
            nifti_path = run_dcm2niix(self._dcm2niix, series_dir, tmp)

            # 2. Run deepdefacer via CLI
            defaced_path = Path(tmp) / "defaced.nii.gz"
            cmd = [
                "deepdefacer",
                str(nifti_path),
                "--defaced_output_path", str(defaced_path),
            ]
            log.info("Running deepdefacer: %s", " ".join(cmd))
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
            if result.returncode != 0:
                raise RuntimeError(f"deepdefacer failed: {result.stderr}")

            # DeepDefacer may write to a slightly different path
            if not defaced_path.exists():
                candidates = list(Path(tmp).glob("*defaced*"))
                if candidates:
                    defaced_path = candidates[0]
                else:
                    raise RuntimeError("deepdefacer produced no output file")

            # 3. Inject defaced NIfTI pixel data back into DICOM
            return nifti_to_dicom(str(defaced_path), series_dir, output_dir)
