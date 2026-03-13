"""
Pure-Python defacing backend using pydicom + nibabel.

Operates directly on DICOM pixel data — no external tools required.
This is the development fallback and is suitable for testing the pipeline.

Algorithm:
  1. Read all DICOM slices and sort by slice position.
  2. Stack into a 3D numpy volume.
  3. Determine the anterior-posterior axis from ImageOrientationPatient.
  4. Zero out voxels in the anterior 30% of that axis (the face region).
  5. Write back modified pixel data into new DICOM files.

Limitations:
  - The face fraction (30%) is approximate; may over-zero in unusual orientations.
  - Only handles uncompressed pixel data (Explicit VR Little Endian).
  - Use mri_deface or mri_reface in production.
"""

import logging
import shutil
from pathlib import Path

import numpy as np
import pydicom
from pydicom.uid import ExplicitVRLittleEndian

from .base import DefacingBackend

log = logging.getLogger(__name__)

# Fraction of the A/P axis to zero out as the face region.
FACE_FRACTION = 0.30


class NibabelFallbackBackend(DefacingBackend):
    @property
    def name(self) -> str:
        return "nibabel-fallback"

    def available(self) -> bool:
        try:
            import pydicom  # noqa: F401
            import numpy  # noqa: F401
            return True
        except ImportError:
            return False

    def deface(self, series_dir: str, output_dir: str) -> list[str]:
        dcm_paths = sorted(Path(series_dir).glob("*.dcm"))
        if not dcm_paths:
            raise RuntimeError(f"No DICOM files found in {series_dir}")

        # --- Load all slices ---
        datasets: list[pydicom.Dataset] = []
        for p in dcm_paths:
            try:
                ds = pydicom.dcmread(str(p))
                datasets.append(ds)
            except Exception as e:
                log.warning("Skipping %s: %s", p, e)

        if not datasets:
            raise RuntimeError("Could not read any DICOM files")

        # --- Sort by slice position (ImagePositionPatient[2] = z in patient space) ---
        def slice_position(ds: pydicom.Dataset) -> float:
            try:
                pos = getattr(ds, "ImagePositionPatient", None)
                if pos:
                    return float(pos[2])
            except Exception:
                pass
            return float(getattr(ds, "InstanceNumber", 0))

        datasets.sort(key=slice_position)

        # --- Stack into 3D volume ---
        first = datasets[0]
        try:
            rows = int(first.Rows)
            cols = int(first.Columns)
        except AttributeError as e:
            raise RuntimeError(f"Cannot determine image dimensions: {e}")

        volume = np.zeros((len(datasets), rows, cols), dtype=np.int16)
        for i, ds in enumerate(datasets):
            try:
                arr = ds.pixel_array
                volume[i] = arr.astype(np.int16)
            except Exception as e:
                log.warning("Slice %d pixel_array failed: %s", i, e)

        # --- Determine anterior-posterior axis ---
        ap_dim, face_at_high_end = self._find_ap_axis(first, volume.shape)
        log.info("A/P axis dimension: %d, face at high end: %s", ap_dim, face_at_high_end)

        # --- Zero out face region ---
        size = volume.shape[ap_dim]
        face_size = max(1, int(size * FACE_FRACTION))
        volume = self._zero_face(volume, ap_dim, face_at_high_end, face_size)
        log.info("Zeroed %d/%d voxels along axis %d", face_size, size, ap_dim)

        # --- Write defaced DICOM files ---
        Path(output_dir).mkdir(parents=True, exist_ok=True)
        output_paths: list[str] = []

        for i, ds in enumerate(datasets):
            out_path = str(Path(output_dir) / dcm_paths[i].name)
            try:
                self._write_slice(ds, volume[i], out_path)
                output_paths.append(out_path)
            except Exception as e:
                log.warning("Could not write defaced slice %d (%s): %s — copying original", i, dcm_paths[i].name, e)
                shutil.copy2(str(dcm_paths[i]), out_path)
                output_paths.append(out_path)

        return output_paths

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _find_ap_axis(self, ds: pydicom.Dataset, shape: tuple) -> tuple[int, bool]:
        """
        Return (axis_index, face_at_high_end) where axis_index is the dimension
        (0=slices, 1=rows, 2=cols) most aligned with the anterior direction.
        """
        ap_vector = np.array([0.0, -1.0, 0.0])  # -Y = Anterior in LPS (DICOM)

        try:
            iop = [float(x) for x in ds.ImageOrientationPatient]
            row_cos = np.array(iop[:3])   # direction along columns
            col_cos = np.array(iop[3:])   # direction along rows
            normal = np.cross(row_cos, col_cos)  # direction along slices (k)
        except Exception:
            # Fallback: assume standard axial orientation
            row_cos = np.array([1.0, 0.0, 0.0])
            col_cos = np.array([0.0, 1.0, 0.0])
            normal = np.array([0.0, 0.0, 1.0])

        # Which of (cols direction, rows direction, slices direction) is most A/P?
        # axes: 2 = cols, 1 = rows, 0 = slices
        axes = [
            (normal, 0),    # slices axis  → dim 0 in volume
            (col_cos, 1),   # rows axis    → dim 1 in volume
            (row_cos, 2),   # cols axis    → dim 2 in volume
        ]
        best_dot = -1.0
        best_dim = 0
        face_at_high = True
        for vec, dim in axes:
            dot = np.dot(vec, ap_vector)
            if abs(dot) > best_dot:
                best_dot = abs(dot)
                best_dim = dim
                face_at_high = dot > 0  # face (anterior) is at high index if dot > 0

        return best_dim, face_at_high

    def _zero_face(
        self,
        volume: np.ndarray,
        ap_dim: int,
        face_at_high_end: bool,
        face_size: int,
    ) -> np.ndarray:
        """Zero out the face region along the given axis."""
        size = volume.shape[ap_dim]
        if face_at_high_end:
            start = size - face_size
            end = size
        else:
            start = 0
            end = face_size

        slices: list = [slice(None)] * volume.ndim
        slices[ap_dim] = slice(start, end)
        volume[tuple(slices)] = 0
        return volume

    def _write_slice(self, ds: pydicom.Dataset, pixel_data: np.ndarray, out_path: str) -> None:
        """Write a single de-faced slice back to DICOM."""
        orig_dtype = ds.pixel_array.dtype
        new_arr = pixel_data.astype(orig_dtype)

        # Ensure uncompressed transfer syntax so we can write raw bytes
        if hasattr(ds, "file_meta"):
            ds.file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

        ds.PixelData = new_arr.tobytes()
        ds.BitsAllocated = new_arr.itemsize * 8
        ds.BitsStored = new_arr.itemsize * 8
        ds.HighBit = ds.BitsStored - 1
        ds.is_implicit_VR = False
        ds.is_little_endian = True
        ds.save_as(out_path, write_like_original=False)
