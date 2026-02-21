"""
Utilities for converting between NIfTI and DICOM for the mri_deface/mri_reface backends.

- run_dcm2niix: converts a DICOM series directory to NIfTI
- nifti_to_dicom: injects NIfTI pixel data back into original DICOM files
"""

import logging
import subprocess
from pathlib import Path

import nibabel as nib
import numpy as np
import pydicom
from pydicom.uid import ExplicitVRLittleEndian

log = logging.getLogger(__name__)


def run_dcm2niix(dcm2niix_bin: str, series_dir: str, output_dir: str) -> Path:
    """
    Convert a DICOM series directory to NIfTI using dcm2niix.
    Returns the path to the primary .nii.gz output file.
    """
    cmd = [
        dcm2niix_bin,
        "-z", "y",       # gzip output
        "-f", "output",  # filename prefix
        "-o", output_dir,
        series_dir,
    ]
    log.info("Running dcm2niix: %s", " ".join(cmd))
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError(f"dcm2niix failed:\n{result.stderr}")

    # Find the output NIfTI file
    nifti_files = list(Path(output_dir).glob("*.nii.gz"))
    if not nifti_files:
        nifti_files = list(Path(output_dir).glob("*.nii"))
    if not nifti_files:
        raise RuntimeError(f"dcm2niix produced no NIfTI output in {output_dir}")

    # Prefer the first non-json file (dcm2niix may produce multiple for multi-echo)
    return nifti_files[0]


def nifti_to_dicom(nifti_path: str, original_series_dir: str, output_dir: str) -> list[str]:
    """
    Inject defaced NIfTI pixel data back into the original DICOM files.

    For each DICOM slice:
      1. Find its patient-space position (ImagePositionPatient).
      2. Map that position to a voxel coordinate in the NIfTI affine.
      3. Extract the 2D slice from the NIfTI volume at that voxel index.
      4. Replace PixelData in the DICOM with the defaced slice.

    This preserves all DICOM metadata (UIDs, scanner parameters, etc.)
    while replacing only the pixel content.
    """
    # Load defaced NIfTI
    nii = nib.load(nifti_path)
    nii_data = np.asarray(nii.dataobj)  # shape: (X, Y, Z) in voxel space
    affine = nii.affine                 # voxel → patient (mm) transform
    inv_affine = np.linalg.inv(affine)  # patient → voxel transform

    # Read original DICOM files
    dcm_paths = sorted(Path(original_series_dir).glob("*.dcm"))
    if not dcm_paths:
        raise RuntimeError(f"No DICOM files found in {original_series_dir}")

    Path(output_dir).mkdir(parents=True, exist_ok=True)
    output_paths: list[str] = []

    for dcm_path in dcm_paths:
        out_path = str(Path(output_dir) / dcm_path.name)
        try:
            ds = pydicom.dcmread(str(dcm_path))
            defaced_slice = _extract_nifti_slice(ds, nii_data, inv_affine)
            _replace_pixel_data(ds, defaced_slice, out_path)
            output_paths.append(out_path)
        except Exception as e:
            log.warning("Failed to deface slice %s: %s — using original", dcm_path.name, e)
            import shutil
            shutil.copy2(str(dcm_path), out_path)
            output_paths.append(out_path)

    return output_paths


def _extract_nifti_slice(
    ds: pydicom.Dataset,
    nii_data: np.ndarray,
    inv_affine: np.ndarray,
) -> np.ndarray:
    """
    Extract the 2D slice from the NIfTI volume that corresponds to a DICOM slice.

    Uses ImagePositionPatient (patient mm coordinates of the first voxel) to
    find the NIfTI voxel coordinate, then extracts the appropriate 2D plane.
    """
    rows = int(ds.Rows)
    cols = int(ds.Columns)

    # Get the slice origin in patient space (mm)
    pos = getattr(ds, "ImagePositionPatient", None)
    if pos is None:
        raise ValueError("Dataset missing ImagePositionPatient")
    origin_mm = np.array([float(x) for x in pos] + [1.0])  # homogeneous

    # Transform to NIfTI voxel coordinates
    voxel = inv_affine @ origin_mm  # shape (4,)
    vx, vy, vz = int(round(voxel[0])), int(round(voxel[1])), int(round(voxel[2]))

    # Determine which voxel axis is the slice normal
    # (the one that changes between slices, i.e., least extent in-plane)
    # Compare NIfTI dimensions to DICOM rows/cols
    nx, ny, nz = nii_data.shape[:3]

    # Determine orientation from IOP to figure out which NIfTI axis is the slice axis
    try:
        iop = [float(x) for x in ds.ImageOrientationPatient]
        row_cos = np.array(iop[:3])
        col_cos = np.array(iop[3:])
        normal = np.cross(row_cos, col_cos)
    except Exception:
        normal = np.array([0.0, 0.0, 1.0])

    # Map normal direction to NIfTI axis (which axis index varies with slice)
    # The NIfTI affine columns are the voxel-to-mm directions for each axis
    affine_3x3 = np.linalg.inv(inv_affine[:3, :3])
    dots = [abs(np.dot(affine_3x3[:, i] / np.linalg.norm(affine_3x3[:, i]), normal))
            for i in range(3)]
    slice_axis = int(np.argmax(dots))

    # Extract the 2D slice along the slice axis
    if slice_axis == 0:
        k = max(0, min(vx, nx - 1))
        plane = nii_data[k, :, :]
    elif slice_axis == 1:
        k = max(0, min(vy, ny - 1))
        plane = nii_data[:, k, :]
    else:
        k = max(0, min(vz, nz - 1))
        plane = nii_data[:, :, k]

    # Resize to match DICOM dimensions if needed (shouldn't normally differ)
    if plane.shape != (rows, cols):
        from scipy.ndimage import zoom
        scale = (rows / plane.shape[0], cols / plane.shape[1])
        plane = zoom(plane, scale, order=1)

    return plane.astype(np.int16)


def _replace_pixel_data(ds: pydicom.Dataset, pixel_data: np.ndarray, out_path: str) -> None:
    """Write a modified pixel array back to a DICOM file."""
    orig_dtype = ds.pixel_array.dtype
    new_arr = pixel_data.astype(orig_dtype)

    if hasattr(ds, "file_meta"):
        ds.file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds.PixelData = new_arr.tobytes()
    ds.BitsAllocated = new_arr.itemsize * 8
    ds.BitsStored = new_arr.itemsize * 8
    ds.HighBit = ds.BitsStored - 1
    ds.is_implicit_VR = False
    ds.is_little_endian = True
    ds.save_as(out_path, write_like_original=False)
