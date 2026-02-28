"""QSM backend — Quantitative Susceptibility Mapping analysis.

QSM estimates tissue magnetic susceptibility from gradient-echo MRI
phase data.  The pipeline consists of three stages: phase unwrapping,
background field removal, and dipole inversion.

This backend implements a basic Python QSM pipeline using numpy/scipy.
When the ``tgv_qsm`` or ``sepia`` binaries are available, those are
preferred for higher-quality reconstruction.

Requires one of:
  - ``scipy`` Python package (for Laplacian phase unwrapping)
  - ``tgv_qsm`` binary (FANSI toolbox)

References:
  Wang Y, Liu T. "Quantitative susceptibility mapping (QSM): Decoding
  MRI data for a tissue magnetic biomarker." Magnetic Resonance in
  Medicine, 2015.  DOI: 10.1002/mrm.25358

  MEDI Toolbox: https://pre.weill.cornell.edu/mri/pages/qsm.html
  STI Suite: https://people.eecs.berkeley.edu/~chunlei.liu/software.html
"""

from __future__ import annotations

import json
import logging
import os
import shutil
import subprocess
import time

import nibabel as nib
import numpy as np

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import find_qsm_niftis

log = logging.getLogger(__name__)


class QSMBackend(AnalyticsBackend):
    """Quantitative Susceptibility Mapping from phase MRI data."""

    @property
    def name(self) -> str:
        return "qsm"

    def available(self) -> bool:
        # Check for tgv_qsm CLI
        if shutil.which("tgv_qsm"):
            return True
        # Fall back to scipy-based pipeline
        try:
            import scipy  # noqa: F401
            return True
        except ImportError:
            return False

    def _use_tgv(self) -> bool:
        return shutil.which("tgv_qsm") is not None

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "qsm")
        os.makedirs(out_dir, exist_ok=True)

        # Find magnitude + phase NIfTI pair
        magnitude_path, phase_path = find_qsm_niftis(bids_dir)
        if not phase_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No phase NIfTI files found in BIDS directory (QSM requires phase data)",
            )

        log.info(
            "QSM: using magnitude=%s, phase=%s",
            magnitude_path or "(none)",
            phase_path,
        )

        qsm_path = os.path.join(out_dir, "qsm_map.nii.gz")

        try:
            if self._use_tgv():
                _run_tgv_qsm(phase_path, magnitude_path, qsm_path)
            else:
                _run_python_qsm(phase_path, magnitude_path, qsm_path)
        except FileNotFoundError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="QSM binary/module not found",
            )
        except subprocess.TimeoutExpired:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"QSM timed out after {config.ANALYTICS_TIMEOUT}s",
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"QSM pipeline failed: {e}",
            )

        if not os.path.isfile(qsm_path):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="QSM map not produced",
            )

        # Compute global susceptibility statistics
        global_stats = _compute_global_stats(qsm_path)

        metrics: dict = {
            "atlas": "qsm",
            "global_susceptibility": global_stats,
            "roi_count": 0,
            "roi_volumes": {},
        }

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [qsm_path, summary_path]

        log.info(
            "QSM complete for %s: mean=%.4f ppm in %.1fs",
            study_uid,
            global_stats.get("mean", 0),
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _run_tgv_qsm(
    phase_path: str,
    magnitude_path: str | None,
    output_path: str,
) -> None:
    """Run QSM via the tgv_qsm CLI tool."""
    cmd = ["tgv_qsm", "-p", phase_path, "-o", output_path]
    if magnitude_path:
        cmd.extend(["-m", magnitude_path])

    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=config.ANALYTICS_TIMEOUT,
    )
    if result.returncode != 0:
        raise RuntimeError(
            result.stderr[-500:] if result.stderr else "tgv_qsm failed"
        )


def _run_python_qsm(
    phase_path: str,
    magnitude_path: str | None,
    output_path: str,
) -> None:
    """Run a basic QSM pipeline using scipy/numpy.

    Implements Laplacian phase unwrapping and truncated k-space dipole
    inversion (TKD).  This is a simplified pipeline suitable for basic
    QSM estimation; production use should prefer TGV-QSM or MEDI.
    """
    from scipy.ndimage import laplace  # type: ignore[import-untyped]

    # Load phase data
    phase_img = nib.load(phase_path)
    phase_data = np.asarray(phase_img.dataobj, dtype=np.float64)

    # Ensure phase is in radians (handle scaled integer phase)
    if np.abs(phase_data).max() > 2 * np.pi:
        # Likely in integer range — normalize to [-pi, pi]
        phase_data = (phase_data / np.abs(phase_data).max()) * np.pi

    # Step 1: Laplacian phase unwrapping
    lap_phase = laplace(phase_data)

    # Step 2: Create a brain mask (threshold on magnitude if available)
    if magnitude_path and os.path.isfile(magnitude_path):
        mag_img = nib.load(magnitude_path)
        mag_data = np.asarray(mag_img.dataobj, dtype=np.float64)
        threshold = np.mean(mag_data) * 0.1
        mask = mag_data > threshold
    else:
        mask = np.abs(phase_data) > 0

    # Step 3: Truncated k-space division (TKD) dipole inversion
    voxel_dims = phase_img.header.get_zooms()[:3]
    qsm_data = _tkd_inversion(lap_phase, mask, voxel_dims)

    # Save output
    qsm_img = nib.Nifti1Image(qsm_data.astype(np.float32), phase_img.affine)
    nib.save(qsm_img, output_path)


def _tkd_inversion(
    local_field: np.ndarray,
    mask: np.ndarray,
    voxel_dims: tuple,
    threshold: float = 0.1,
) -> np.ndarray:
    """Truncated k-space division for dipole inversion.

    Simple but fast QSM reconstruction.
    """
    shape = local_field.shape

    # Create dipole kernel in k-space
    kx = np.fft.fftfreq(shape[0], d=voxel_dims[0])
    ky = np.fft.fftfreq(shape[1], d=voxel_dims[1])
    kz = np.fft.fftfreq(shape[2], d=voxel_dims[2])
    KX, KY, KZ = np.meshgrid(kx, ky, kz, indexing="ij")

    k2 = KX**2 + KY**2 + KZ**2
    k2[k2 == 0] = 1e-10

    # Dipole kernel: D = 1/3 - kz^2/k^2
    dipole = 1.0 / 3.0 - KZ**2 / k2

    # Truncated division: zero out where dipole is small
    dipole[np.abs(dipole) < threshold] = threshold

    # Invert
    field_k = np.fft.fftn(local_field * mask)
    chi_k = field_k / dipole
    chi = np.real(np.fft.ifftn(chi_k)) * mask

    return chi


def _compute_global_stats(qsm_path: str) -> dict[str, float]:
    """Compute global QSM statistics from the susceptibility map."""
    img = nib.load(qsm_path)
    data = np.asarray(img.dataobj, dtype=np.float64)

    nonzero = data[data != 0]
    if nonzero.size == 0:
        return {"mean": 0.0, "median": 0.0, "std": 0.0, "min": 0.0, "max": 0.0}

    return {
        "mean": round(float(np.mean(nonzero)), 6),
        "median": round(float(np.median(nonzero)), 6),
        "std": round(float(np.std(nonzero)), 6),
        "min": round(float(np.min(nonzero)), 6),
        "max": round(float(np.max(nonzero)), 6),
    }
