"""Synthetic brain MRI phantom generator.

CPU path: Shepp-Logan phantom + T1w contrast remapping + Rician noise.
GPU path: MONAI BraTS LDM (optional; see monai_backend.py).

Both paths produce lists of (pixel_array, study_uid, series_uid, instance_num, slice_z)
tuples ready for packaging into DICOM files by write_dicom_series().
"""

import datetime
from pathlib import Path

import numpy as np
import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid

# ── Shepp-Logan phantom ───────────────────────────────────────────────────────
# 10 ellipses: (semi-axis-a, semi-axis-b, center-x, center-y, rotation-deg, density)
# Based on the classic Shepp & Logan 1974 definition.
_ELLIPSES = [
    (0.6900, 0.9200,  0.00,  0.0000,  0.0,  1.0),
    (0.6624, 0.8740,  0.00, -0.0184,  0.0, -0.8),
    (0.1100, 0.3100,  0.22,  0.0000, -18.0, -0.2),
    (0.1600, 0.4100, -0.22,  0.0000,  18.0, -0.2),
    (0.2100, 0.2500,  0.00,  0.3500,   0.0,  0.1),
    (0.0460, 0.0460,  0.00,  0.1000,   0.0,  0.1),
    (0.0460, 0.0460,  0.00, -0.1000,   0.0,  0.1),
    (0.0460, 0.0230, -0.08, -0.6050,   0.0,  0.1),
    (0.0230, 0.0230,  0.00, -0.6060,   0.0,  0.1),
    (0.0230, 0.0460,  0.06, -0.6050,   0.0,  0.1),
]


def _shepp_logan_phantom(n: int) -> np.ndarray:
    """Build an n×n Shepp-Logan phantom normalised to [0, 1]."""
    x = np.linspace(-1, 1, n)
    y = np.linspace(-1, 1, n)
    X, Y = np.meshgrid(x, y)
    p = np.zeros((n, n), dtype=float)
    for a, b, x0, y0, phi, rho in _ELLIPSES:
        phi_r = np.deg2rad(phi)
        xr = (X - x0) * np.cos(phi_r) + (Y - y0) * np.sin(phi_r)
        yr = -(X - x0) * np.sin(phi_r) + (Y - y0) * np.cos(phi_r)
        mask = (xr / a) ** 2 + (yr / b) ** 2 <= 1.0
        p[mask] += rho
    p = np.clip(p, 0, None)
    if p.max() > 0:
        p /= p.max()
    return p


def _t1w_remap(p: np.ndarray) -> np.ndarray:
    """Remap Shepp-Logan intensities to approximate T1-weighted MRI contrast.

    Mapping:
      - White matter  (phantom ~0.8–1.0) → bright  (T1 ~0.80–1.00)
      - Gray matter   (phantom ~0.15–0.7) → mid     (T1 ~0.35–0.70)
      - CSF / sulci   (phantom ~0.01–0.15)→ dark    (T1 ~0.05–0.15)
      - Background    (phantom 0)         → zero
    """
    out = np.zeros_like(p)
    out = np.where(p > 0.70, 0.80 + 0.20 * (p - 0.70) / 0.30, out)
    out = np.where((p > 0.15) & (p <= 0.70), 0.35 + 0.35 * (p - 0.15) / 0.55, out)
    out = np.where((p > 0.01) & (p <= 0.15), 0.05 + 0.10 * (p - 0.01) / 0.14, out)
    return out


def _add_cortical_noise(img: np.ndarray, rng: np.random.Generator) -> np.ndarray:
    """Add Rician noise and subtle sulcal texture to a float image."""
    sigma = 0.025
    noise_r = rng.normal(0, sigma, img.shape)
    noise_i = rng.normal(0, sigma, img.shape)
    img_noisy = np.sqrt((img + noise_r) ** 2 + noise_i ** 2)
    # Subtle sulcal texture in the cortical gray-matter band.
    cortex_mask = (img > 0.30) & (img < 0.60)
    texture = rng.normal(0, 0.012, img.shape) * cortex_mask
    return np.clip(img_noisy + texture, 0, 1)


def _add_face_region(canvas: np.ndarray, z_norm: float, size: int) -> np.ndarray:
    """Inject anatomically plausible face structures into inferior slices.

    z_norm = 0 → most inferior slice, z_norm = 1 → most superior.
    Only modifies slices where z_norm < 0.35 (inferior 35% = face region).
    Added structures (bilateral orbits, nasal cavity, maxillary sinuses, fat rim)
    give defacing tools something to detect and remove — used for demo purposes.
    """
    if z_norm >= 0.35:
        return canvas

    ctr = size // 2
    fade = 1.0 - (z_norm / 0.35)  # strongest at inferior pole, fades toward brain
    result = canvas.copy()

    # Orbital air cells (bilateral) — hypointense on T1w
    for side in (-1, 1):
        ox = int(ctr + side * size * 0.13)
        oy = int(ctr - size * 0.12)
        r = int(size * 0.055)
        yy, xx = np.ogrid[:size, :size]
        orbit_mask = (xx - ox) ** 2 + (yy - oy) ** 2 <= r**2
        result[orbit_mask] = 0.02 * (1 - fade * 0.5)

    # Nasal cavity — hypointense midline channel
    nc_x0 = max(0, int(ctr - size * 0.06))
    nc_x1 = min(size, int(ctr + size * 0.06))
    nc_y0 = max(0, int(ctr - size * 0.04))
    nc_y1 = min(size, int(ctr + size * 0.18))
    result[nc_y0:nc_y1, nc_x0:nc_x1] *= 1 - fade * 0.70

    # Maxillary sinuses (bilateral) — hypointense ellipses
    for side in (-1, 1):
        mx = int(ctr + side * size * 0.17)
        my = int(ctr + size * 0.10)
        ry = int(size * 0.07)
        rx = int(size * 0.09)
        yy, xx = np.ogrid[:size, :size]
        sinus_mask = ((xx - mx) / rx) ** 2 + ((yy - my) / ry) ** 2 <= 1.0
        result[sinus_mask] *= 1 - fade * 0.75

    # Subcutaneous fat rim around the face — hyperintense on T1w
    face_outer = int(size * 0.42)
    face_inner = int(size * 0.36)
    yy, xx = np.ogrid[:size, :size]
    d = np.sqrt((xx - ctr) ** 2 + (yy - ctr) ** 2)
    fat_mask = (d >= face_inner) & (d <= face_outer) & (result < 0.10)
    result[fat_mask] += 0.55 * fade

    return np.clip(result, 0, 1)


# ── Synthetic PHI ─────────────────────────────────────────────────────────────
# All fields are fabricated — no real patient data.
_SYNTHETIC_PHI: dict = {
    "PatientName": "DEMO^PATIENT^MRI",
    "PatientID": "PAT-SYNTH-001",
    "PatientBirthDate": "19780314",
    "PatientSex": "M",
    "PatientAge": "047Y",
    "PatientWeight": "72.5",
    "PatientSize": "1.75",
    "InstitutionName": "AEGIS Demo Center",
    "InstitutionAddress": "1 Demo Ave, Springfield",
    "ReferringPhysicianName": "DEMO^REFERRING^PHYSICIAN",
    "OperatorsName": "AEGIS^AUTO^GEN",
}


# ── DICOM packaging ───────────────────────────────────────────────────────────


def _build_dicom_slice(
    pixel_array: np.ndarray,
    study_uid: str,
    series_uid: str,
    instance_num: int,
    slice_location: float,
    size: int,
    output_path: str,
) -> FileDataset:
    """Package a 2D float array [0, 1] as a 16-bit DICOM MR image slice."""
    sop_uid = generate_uid()

    file_meta = pydicom.Dataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    file_meta.MediaStorageSOPInstanceUID = sop_uid
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
    # FileMetaInformationVersion is required for a complete File Meta Information
    # group; pydicom 3.x omits MetaElementGroupLength (0002,0000) unless the
    # file meta is complete and enforce_file_format=True is passed on save.
    file_meta.FileMetaInformationVersion = b"\x00\x01"

    ds = FileDataset(output_path, {}, file_meta=file_meta, preamble=b"\x00" * 128)
    ds.is_implicit_VR = False
    ds.is_little_endian = True

    # SOP
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    ds.SOPInstanceUID = sop_uid

    # Synthetic PHI tags
    for attr, val in _SYNTHETIC_PHI.items():
        setattr(ds, attr, val)

    # Study / Series / Instance hierarchy
    ds.StudyInstanceUID = study_uid
    ds.SeriesInstanceUID = series_uid
    ds.InstanceNumber = instance_num

    # Study metadata
    now = datetime.datetime.now(datetime.timezone.utc)
    ds.StudyDate = now.strftime("%Y%m%d")
    ds.StudyTime = now.strftime("%H%M%S.%f")[:13]
    ds.StudyDescription = "Synthetic Brain MRI T1w"
    ds.SeriesDescription = "T1w MPRAGE Synthetic"

    # Modality
    ds.Modality = "MR"
    ds.BodyPartExamined = "HEAD"

    # Acquisition parameters (3T MPRAGE-style)
    ds.RepetitionTime = "2300.0"
    ds.EchoTime = "2.98"
    ds.InversionTime = "900.0"
    ds.FlipAngle = "9.0"
    ds.SliceThickness = "1.0"
    ds.PixelSpacing = [1.0, 1.0]
    ds.ImageOrientationPatient = [1.0, 0.0, 0.0, 0.0, 1.0, 0.0]
    ds.ImagePositionPatient = [-size / 2.0, -size / 2.0, slice_location]
    ds.SliceLocation = slice_location
    ds.SpacingBetweenSlices = "1.0"
    ds.MagneticFieldStrength = "3"
    ds.StationName = "MRI-3T-01"
    ds.SequenceName = "tfl3d1_16ns"
    ds.ScanningSequence = "GR"
    ds.SequenceVariant = "SP"

    # Image geometry
    ds.Rows = size
    ds.Columns = size
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME2"
    # Window covers the meaningful tissue range (roughly 0–1000 out of 0–4095).
    # WC=500/WW=1000 matches the working demo files and renders brain tissue
    # at 50–100% brightness in OHIF instead of the near-black 20% produced
    # by the old WC=2048/WW=4096 setting.
    ds.WindowWidth = 1000
    ds.WindowCenter = 500

    # Pixel data — scale float [0, 1] → uint16 (12-bit range: 0–4095)
    arr_u16 = (np.clip(pixel_array, 0, 1) * 4095).astype(np.uint16)
    ds.PixelData = arr_u16.tobytes()

    return ds


# Slice tuple type: (pixel_array, study_uid, series_uid, instance_num, slice_z)
SliceTuple = tuple[np.ndarray, str, str, int, float]


def generate_phantom_slices(
    n_slices: int = 20,
    size: int = 256,
    seed: int = 42,
    with_face: bool = False,
) -> list[SliceTuple]:
    """Generate n_slices synthetic T1w brain MRI slices using the Shepp-Logan phantom.

    Returns a list of SliceTuple — all slices share the same study_uid / series_uid.
    """
    rng = np.random.default_rng(seed)
    base = _shepp_logan_phantom(size)
    base_t1w = _t1w_remap(base)

    study_uid = generate_uid()
    series_uid = generate_uid()

    slices: list[SliceTuple] = []
    for i in range(n_slices):
        z_norm = i / max(n_slices - 1, 1)
        # Taper signal at the poles so the brain has natural superior/inferior fade.
        taper = np.sin(np.pi * z_norm) ** 0.5
        img = base_t1w * taper
        if with_face:
            img = _add_face_region(img, z_norm, size)
        img = _add_cortical_noise(img, rng)
        slice_z = float(i) - n_slices / 2.0
        slices.append((img, study_uid, series_uid, i + 1, slice_z))

    return slices


def write_dicom_series(
    slices: list[SliceTuple],
    output_dir: str,
    size: int = 256,
) -> list[str]:
    """Write a list of SliceTuples to DICOM files under output_dir.

    Returns the list of file paths written.
    """
    out_path = Path(output_dir)
    out_path.mkdir(parents=True, exist_ok=True)

    paths: list[str] = []
    for pixel_array, study_uid, series_uid, instance_num, slice_z in slices:
        fname = out_path / f"{instance_num:04d}.dcm"
        ds = _build_dicom_slice(
            pixel_array,
            study_uid=study_uid,
            series_uid=series_uid,
            instance_num=instance_num,
            slice_location=slice_z,
            size=size,
            output_path=str(fname),
        )
        # enforce_file_format=True (pydicom ≥ 3.0) ensures MetaElementGroupLength
        # (0002,0000) is written, making the file valid for strict DICOM parsers.
        try:
            ds.save_as(str(fname), enforce_file_format=True)
        except TypeError:
            # pydicom < 3.0 — save_as writes MetaElementGroupLength automatically.
            ds.save_as(str(fname))
        paths.append(str(fname))

    return paths
