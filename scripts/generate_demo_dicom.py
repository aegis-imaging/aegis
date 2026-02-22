#!/usr/bin/env python3
"""
generate_demo_dicom.py
======================
Generate a synthetic brain MRI DICOM series for the AEGIS landing page demo.

Two generation modes (auto-selected):
  GPU / MONAI LDM  -- Latent Diffusion Model pre-trained on BraTS 2016/2017 data.
                      Produces photorealistic T1w axial slices.
                      Requires: torch, monai-generative, huggingface_hub

  CPU fallback     -- Shepp-Logan phantom + anatomical noise + MRI-like contrast.
                      Pure numpy/scipy, no GPU, no downloads, always works.

Output:
  A DICOM series with synthetic (fake) PHI tags that demonstrate AEGIS
  de-identification. The tags are invented and contain no real patient data.

  The generated brain images are also realistic enough to be passed through
  the AEGIS defacing service (mri_deface / DeepDefacer) for demonstration.

Usage:
  # CPU fallback only (fast, no downloads):
  python scripts/generate_demo_dicom.py

  # With MONAI LDM (GPU machine, recommended for defacing demo):
  pip install torch torchvision
  pip install monai monai-generative huggingface_hub
  python scripts/generate_demo_dicom.py --gpu

  # Custom output directory:
  python scripts/generate_demo_dicom.py --output frontend/landing/public/demo

GPU setup:
  pip install torch torchvision --index-url https://download.pytorch.org/whl/cu118
  pip install monai monai-generative huggingface_hub pydicom numpy scipy pillow
"""

import argparse
import os
import sys

import numpy as np
import pydicom
from pydicom.dataset import Dataset, FileMetaDataset
from pydicom.uid import generate_uid, ExplicitVRLittleEndian

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)


# ---------------------------------------------------------------------------
# Synthetic PHI -- invented, NOT real patient data.
# These tags exist solely to showcase AEGIS de-identification in the demo.
# ---------------------------------------------------------------------------
SYNTHETIC_PHI = {
    "PatientName":               "DEMO^PATIENT^MRI",
    "PatientID":                 "PAT-2025-001337",
    "PatientBirthDate":          "19780314",
    "PatientSex":                "M",
    "PatientAge":                "047Y",
    "PatientWeight":             "82.5",
    "PatientAddress":            "123 Cortex Lane, Brainsville, CA 90210",
    "PatientTelephoneNumbers":   "555-867-5309",
    "ReferringPhysicianName":    "SMITH^JOHN^A",
    "InstitutionName":           "Neuroscience Medical Center",
    "InstitutionAddress":        "456 Gray Matter Ave, Synapse City, NY 10001",
    "StudyDate":                 "20250214",
    "StudyTime":                 "143022",
    "AccessionNumber":           "ACC-20250214-0042",
    "StudyID":                   "STUDY-0042",
    "StudyDescription":          "BRAIN MRI W/O CONTRAST T1W AXIAL",
    "RequestingPhysician":       "JONES^SARAH^B",
    "RequestedProcedureDescription": "MRI BRAIN ROUTINE",
    "PerformingPhysicianName":   "LEE^MICHAEL^C",
    "OperatorsName":             "TECH^MRI^OPERATOR",
    "StationName":               "MRI-3T-01",
    "DeviceSerialNumber":        "SN-7734-SIEMENS",
}

MRI_PARAMS = {
    "Modality":                       "MR",
    "Manufacturer":                   "SIEMENS",
    "ManufacturerModelName":          "MAGNETOM Prisma",
    "MagneticFieldStrength":          "3.0",
    "RepetitionTime":                 "2300.0",
    "EchoTime":                       "2.98",
    "FlipAngle":                      "9.0",
    "SliceThickness":                 "1.0",
    "PixelSpacing":                   [1.0, 1.0],
    "BodyPartExamined":               "HEAD",
    "ScanningSequence":               "GR",
    "SequenceVariant":                "SP",
    "ImageType":                      ["ORIGINAL", "PRIMARY", "M", "ND", "NORM"],
    "PhotometricInterpretation":      "MONOCHROME2",
    "BitsAllocated":                  16,
    "BitsStored":                     16,
    "HighBit":                        15,
    "PixelRepresentation":            0,
    "SamplesPerPixel":                1,
    "Rows":                           256,
    "Columns":                        256,
    "InstanceCreationDate":           "20250214",
    "InstanceCreationTime":           "143022",
    "ProtocolName":                   "T1w_MPRAGE_1mm_ISO",
    "SeriesDescription":              "T1w MPRAGE 1mm isotropic",
    "NumberOfAverages":               "1",
    "ImagingFrequency":               "123.254",
    "ImagedNucleus":                  "1H",
    "SpacingBetweenSlices":           "1.0",
    "PercentSampling":                "100",
    "PercentPhaseFieldOfView":        "100",
    "PixelBandwidth":                 "240",
    "SoftwareVersions":               "syngo MR E11",
    "TransmitCoilName":               "Body",
    "AcquisitionMatrix":              [0, 256, 256, 0],
    "InPlanePhaseEncodingDirection":  "COL",
    "SAR":                            "0.0487",
    "PositionReferenceIndicator":     "IC",
}


# ---------------------------------------------------------------------------
# CPU path: Shepp-Logan phantom with T1w MRI contrast + anatomical layering
# ---------------------------------------------------------------------------

def _shepp_logan_phantom(n):
    """2D Shepp-Logan phantom of size (n, n), values in [0, 1]."""
    # Standard 10-ellipse parameterisation
    # [intensity, semi-axis a, semi-axis b, centre cx, centre cy, rotation deg]
    ellipses = [
        [1.00,  .6900, .9200,  0.00,  0.000,   0],
        [-.98,  .6624, .8740,  0.00, -.0184,   0],
        [-.02,  .1100, .3100,  .220,  0.000, -18],
        [-.02,  .1600, .4100, -.220,  0.000,  18],
        [ .01,  .2100, .2500,  0.00,  .3500,   0],
        [ .01,  .0460, .0460,  0.00,  .1000,   0],
        [ .01,  .0460, .0460,  0.00, -.1000,   0],
        [ .01,  .0460, .0230, -.080, -.6050,   0],
        [ .01,  .0230, .0230,  0.00, -.6060,   0],
        [ .01,  .0230, .0460,  .060, -.6050,   0],
    ]
    phantom = np.zeros((n, n), dtype=np.float32)
    xs = np.linspace(-1, 1, n)
    xx, yy = np.meshgrid(xs, xs)
    for val, a, b, cx, cy, phi in ellipses:
        pr = np.deg2rad(phi)
        xr = (xx - cx) * np.cos(pr) + (yy - cy) * np.sin(pr)
        yr = -(xx - cx) * np.sin(pr) + (yy - cy) * np.cos(pr)
        phantom[(xr / a) ** 2 + (yr / b) ** 2 <= 1.0] += val
    return np.clip(phantom, 0, 1)


def _t1w_remap(p):
    """Remap Shepp-Logan intensities to approximate T1-weighted MRI."""
    from scipy.ndimage import gaussian_filter
    out = p.copy()
    # White matter: brightest
    out[p > 0.72] = p[p > 0.72] * 1.10
    # CSF-like cavities: dark on T1
    csf = (p > 0.03) & (p < 0.22)
    out[csf] = p[csf] * 0.25
    # Scalp/skull ring (outer edge of head ellipse)
    out[p > 0.93] = 0.55
    return np.clip(out, 0, 1)


def _add_cortical_noise(img, rng):
    """Add Rician + cortical sulcal texture."""
    from scipy.ndimage import gaussian_filter
    coarse = gaussian_filter(rng.normal(0, 0.025, img.shape).astype(np.float32), sigma=4)
    fine   = gaussian_filter(rng.normal(0, 0.012, img.shape).astype(np.float32), sigma=1.5)
    gm = (img > 0.30) & (img < 0.60)
    result = img.copy()
    result[gm] += (coarse + fine)[gm]
    # Rician noise everywhere
    real = result + rng.normal(0, 0.010, img.shape).astype(np.float32)
    imag = rng.normal(0, 0.010, img.shape).astype(np.float32)
    return np.clip(np.sqrt(real**2 + imag**2), 0, 1)


def _add_face_region(canvas, z_norm, size):
    """
    Add a synthetic face/skull region to inferior slices (z_norm < 0).

    In axial MRI the face appears in inferior slices as an anterior protrusion
    containing orbital air cells (dark), nasal cavity (dark), and soft tissue
    (bright). Defacing tools (mri_deface, DeepDefacer) target exactly this
    region and mask it; the before/after difference is what the demo shows.

    z_norm: normalised z position in [-1, 1]; face visible when < 0.
    """
    from scipy.ndimage import gaussian_filter

    if z_norm >= 0:
        return canvas

    # Face prominence scales with depth below mid-brain
    strength = min(1.0, (-z_norm) * 1.8)  # 0 at mid-brain, 1 near chin

    xs = np.linspace(-1, 1, size)
    xx, yy = np.meshgrid(xs, xs)

    # --- Anterior soft-tissue face ellipse (frontal face / forehead) ---
    # Shifted anteriorly (positive y in our grid = anterior)
    face_cy = 0.55 * strength        # shifts forward as we go inferior
    face_a  = 0.35 * strength + 0.1  # lateral width
    face_b  = 0.30 * strength + 0.1  # AP depth
    face_mask = ((xx / face_a) ** 2 + ((yy - face_cy) / face_b) ** 2) <= 1.0
    canvas[face_mask] = np.clip(canvas[face_mask] + 0.45 * strength, 0, 1)

    # Smooth the face boundary
    from scipy.ndimage import gaussian_filter
    canvas = gaussian_filter(canvas, sigma=1.5)

    # --- Orbital air cells (dark, bilateral) ---
    for sign in (-1, 1):
        orb_cx = sign * 0.22
        orb_cy = 0.38 * strength
        orb_r  = 0.07 + 0.03 * strength
        orb_mask = ((xx - orb_cx) ** 2 + (yy - orb_cy) ** 2) <= orb_r ** 2
        canvas[orb_mask] = 0.03  # nearly black -- air

    # --- Nasal septum / cavity (dark midline) ---
    nasal_b  = 0.09 * strength
    nasal_a  = 0.05
    nasal_cy = 0.25 * strength
    nasal_mask = ((xx / nasal_a) ** 2 + ((yy - nasal_cy) / nasal_b) ** 2) <= 1.0
    canvas[nasal_mask] = 0.05

    # --- Maxillary sinuses (dark, lateral to nose) ---
    for sign in (-1, 1):
        sx_cx = sign * 0.20
        sx_cy = 0.20 * strength
        sx_a  = 0.09 * strength
        sx_b  = 0.07 * strength
        sx_mask = ((xx - sx_cx) / max(sx_a, 0.01)) ** 2 + \
                  ((yy - sx_cy) / max(sx_b, 0.01)) ** 2 <= 1.0
        canvas[sx_mask] = 0.04

    return np.clip(canvas, 0, 1)


def generate_phantom_slices(n_slices, size=256, seed=42, with_face=False):
    """
    Generate n_slices axial T1w brain phantom slices.

    with_face=True:
      Adds anatomically plausible face/orbital/sinus regions to inferior
      slices. Required for a meaningful defacing demonstration -- without it,
      defacing tools have nothing to remove.

    The series is ordered inferior→superior (slice 0 = base of skull,
    slice n-1 = crown), matching conventional scanner convention and the
    orientation expected by mri_deface / DeepDefacer.

    Returns ndarray of shape (n_slices, size, size) in [0, 1].
    """
    from scipy.ndimage import zoom

    rng  = np.random.default_rng(seed)
    base = _shepp_logan_phantom(size)
    slices = []

    for i in range(n_slices):
        # z in [-1, 1]: -1 = inferior (chin), +1 = superior (crown)
        z = (i / max(n_slices - 1, 1)) * 2 - 1
        # Taper brain cross-section toward crown and base
        scale = max(0.4, 1.0 - 0.5 * z**2)
        scaled = zoom(base, scale, order=1)
        sh, sw = scaled.shape

        # Centre-crop / zero-pad back to target size
        out = np.zeros((size, size), dtype=np.float32)
        rh = min(sh, size)
        rw = min(sw, size)
        yr = (size - rh) // 2
        xr = (size - rw) // 2
        sy = (sh - rh) // 2
        sx = (sw - rw) // 2
        out[yr:yr+rh, xr:xr+rw] = scaled[sy:sy+rh, sx:sx+rw]

        slc = _t1w_remap(out)

        # Optionally add face anatomy for defacing demo
        if with_face:
            slc = _add_face_region(slc, z, size)

        slc = _add_cortical_noise(slc, rng)
        slices.append(slc.astype(np.float32))

    return np.stack(slices, axis=0)


# ---------------------------------------------------------------------------
# GPU path: MONAI pre-trained BraTS Latent Diffusion Model
# Model: monai-test/brats_mri_axial_slices_generative_diffusion (HuggingFace)
# ---------------------------------------------------------------------------

def generate_monai_slices(n_slices, size=256):
    """
    Generate brain MRI slices using the MONAI BraTS LDM.
    Returns ndarray of shape (n_slices, size, size) in [0, 1].
    """
    import torch
    from huggingface_hub import snapshot_download

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print(f"[MONAI] Device: {device}")

    print("[MONAI] Downloading BraTS LDM from HuggingFace (one-time)...")
    model_path = snapshot_download(
        repo_id="monai-test/brats_mri_axial_slices_generative_diffusion"
    )
    print(f"[MONAI] Model cached at: {model_path}")

    try:
        from generative.networks.nets import DiffusionModelUNet, AutoencoderKL
        from generative.networks.schedulers import DDIMScheduler
    except ImportError:
        raise ImportError(
            "monai-generative not installed. Run:\n"
            "  pip install monai-generative"
        )

    unet = DiffusionModelUNet(
        spatial_dims=2,
        in_channels=1,
        out_channels=1,
        num_channels=(256, 512, 768),
        attention_levels=(False, True, True),
        num_res_blocks=2,
        num_head_channels=64,
        with_conditioning=False,
    ).to(device)

    autoencoder = AutoencoderKL(
        spatial_dims=2,
        in_channels=1,
        out_channels=1,
        num_channels=(128, 256, 512),
        latent_channels=4,
        num_res_blocks=2,
        attention_levels=(False, True, True),
        with_encoder_nonlocal_attn=True,
        with_decoder_nonlocal_attn=True,
    ).to(device)

    # Locate checkpoints (HuggingFace repo layout may vary)
    unet_ckpt, ae_ckpt = None, None
    for root, _, files in os.walk(model_path):
        for f in files:
            fp = os.path.join(root, f)
            if "diffusion" in f.lower() and f.endswith((".pt", ".pth")):
                unet_ckpt = fp
            elif "autoencoder" in f.lower() and f.endswith((".pt", ".pth")):
                ae_ckpt = fp

    if not unet_ckpt or not ae_ckpt:
        raise FileNotFoundError(
            "Could not find diffusion_model.pt / autoencoder.pt in downloaded repo.\n"
            f"Contents of {model_path}:\n" +
            "\n".join(
                os.path.join(r, ff)
                for r, _, fs in os.walk(model_path)
                for ff in fs
            )
        )

    unet.load_state_dict(torch.load(unet_ckpt, map_location=device))
    autoencoder.load_state_dict(torch.load(ae_ckpt, map_location=device))
    unet.eval()
    autoencoder.eval()

    scheduler = DDIMScheduler(
        num_train_timesteps=1000,
        beta_start=0.0015,
        beta_end=0.0195,
        schedule="scaled_linear_beta",
        clip_sample=False,
    )
    scheduler.set_timesteps(50)

    scale_factor = 0.3  # BraTS checkpoint latent normalisation

    generated = []
    print(f"[MONAI] Sampling {n_slices} slices (DDIM 50 steps)...")
    with torch.no_grad():
        for i in range(n_slices):
            print(f"  {i + 1}/{n_slices}", end="\r", flush=True)
            latent_sz = size // 4
            z = torch.randn((1, 4, latent_sz, latent_sz), device=device)
            for t in scheduler.timesteps:
                noise_pred = unet(z, timesteps=torch.Tensor([t]).to(device))
                z, _ = scheduler.step(noise_pred, t, z)
            img = autoencoder.decode_stage_2_outputs(z / scale_factor)
            arr = img.squeeze().cpu().numpy()
            arr = (arr - arr.min()) / (arr.max() - arr.min() + 1e-8)
            generated.append(arr.astype(np.float32))

    print()
    return np.stack(generated, axis=0)


# ---------------------------------------------------------------------------
# DICOM packaging
# ---------------------------------------------------------------------------

def _build_dicom_slice(pixels_f32, slice_idx, n_slices, study_uid, series_uid, fref_uid):
    """
    Wrap a (H, W) float32 [0,1] array into a DICOM MR Image dataset.
    """
    pixels_u16 = (pixels_f32 * 4095.0).clip(0, 4095).astype(np.uint16)

    file_meta = FileMetaDataset()
    sop_uid = generate_uid()
    file_meta.MediaStorageSOPClassUID    = "1.2.840.10008.5.1.4.1.1.4"
    file_meta.MediaStorageSOPInstanceUID = sop_uid
    file_meta.TransferSyntaxUID          = ExplicitVRLittleEndian

    ds = Dataset()
    ds.file_meta      = file_meta
    ds.is_implicit_VR = False
    ds.is_little_endian = True
    ds.preamble       = b"\x00" * 128

    ds.SOPClassUID       = "1.2.840.10008.5.1.4.1.1.4"
    ds.SOPInstanceUID    = sop_uid
    ds.StudyInstanceUID  = study_uid
    ds.SeriesInstanceUID = series_uid
    ds.FrameOfReferenceUID = fref_uid
    ds.SeriesNumber      = "1"
    ds.InstanceNumber    = str(slice_idx + 1)

    # Synthetic PHI tags (these are what AEGIS will de-identify)
    for tag, val in SYNTHETIC_PHI.items():
        setattr(ds, tag, val)

    # MRI acquisition parameters (retained by AEGIS anonymization profiles)
    for tag, val in MRI_PARAMS.items():
        setattr(ds, tag, val)

    # Slice geometry — 1 mm spacing along Z, FOV centred at origin
    z_pos = (slice_idx - n_slices // 2) * 1.0
    ds.ImagePositionPatient    = ["-128.0", "-128.0", f"{z_pos:.1f}"]
    ds.ImageOrientationPatient = ["1", "0", "0", "0", "1", "0"]
    ds.SliceLocation           = f"{z_pos:.1f}"

    ds.WindowCenter = "512"
    ds.WindowWidth  = "1024"
    ds.Rows         = pixels_u16.shape[0]
    ds.Columns      = pixels_u16.shape[1]
    ds.PixelData    = pixels_u16.tobytes()

    return ds


def write_dicom_series(slices, output_dir):
    """Write ndarray (N, H, W) as a DICOM series. Returns list of file paths."""
    os.makedirs(output_dir, exist_ok=True)
    study_uid = generate_uid()
    series_uid = generate_uid()
    fref_uid = generate_uid()
    n = slices.shape[0]
    paths = []
    for i, slc in enumerate(slices):
        ds   = _build_dicom_slice(slc, i, n, study_uid, series_uid, fref_uid)
        path = os.path.join(output_dir, f"slice_{i + 1:03d}.dcm")
        pydicom.dcmwrite(path, ds, write_like_original=False)
        paths.append(path)
        print(f"  Wrote {path}")
    return paths


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(
        description=(
            "Generate a synthetic brain MRI DICOM series for the AEGIS demo.\n"
            "\n"
            "Modes:\n"
            "  default        -- 20 slices, de-id demo (landing page drop zone)\n"
            "  --defacing-demo -- 180 slices + face anatomy, for defacing before/after\n"
            "  --gpu          -- MONAI BraTS LDM (photorealistic, requires GPU machine)\n"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    ap.add_argument(
        "--output", "-o",
        default=os.path.join(PROJECT_DIR, "frontend", "landing", "public", "demo"),
        help="Output directory (default: frontend/landing/public/demo)",
    )
    ap.add_argument(
        "--slices", "-n", type=int, default=None,
        help="Number of axial slices (default: 20 normal, 180 for --defacing-demo)",
    )
    ap.add_argument(
        "--size", type=int, default=256,
        help="Image size in pixels (default: 256)",
    )
    ap.add_argument(
        "--gpu", action="store_true",
        help="Use MONAI BraTS LDM pretrained model (requires monai-generative + torch)",
    )
    ap.add_argument(
        "--defacing-demo", action="store_true", dest="defacing_demo",
        help=(
            "Generate a full-head volume suitable for mri_deface / DeepDefacer.\n"
            "Adds face/orbital/sinus anatomy to inferior slices (the region\n"
            "defacing tools detect and mask). Use with --gpu for best results.\n"
            "Defaults to 180 slices at 256x256 (1 mm isotropic)."
        ),
    )
    ap.add_argument(
        "--seed", type=int, default=42,
        help="RNG seed for phantom generation (default: 42)",
    )
    args = ap.parse_args()

    # Apply defacing-demo defaults
    if args.defacing_demo:
        n_slices    = args.slices or 180
        with_face   = True
        demo_label  = "Full-head volume for defacing demo"
        output_dir  = args.output if args.slices else os.path.join(
            PROJECT_DIR, "frontend", "landing", "public", "demo-defacing"
        )
    else:
        n_slices    = args.slices or 20
        with_face   = False
        demo_label  = "De-identification demo (drop any .dcm in landing page)"
        output_dir  = args.output

    for pkg in ("numpy", "pydicom", "scipy"):
        try:
            __import__(pkg)
        except ImportError:
            print(f"Missing required package: {pkg}")
            print(f"  pip install {pkg}")
            sys.exit(1)

    print("=" * 60)
    print("AEGIS Synthetic Brain MRI Generator")
    print("=" * 60)
    print(f"Mode     : {demo_label}")
    print(f"Output   : {output_dir}")
    print(f"Slices   : {n_slices}")
    print(f"Size     : {args.size}x{args.size}")
    print(f"Engine   : {'MONAI LDM (GPU)' if args.gpu else 'Shepp-Logan phantom (CPU)'}")
    if args.defacing_demo:
        print()
        print("Face anatomy: ENABLED")
        print("  Inferior slices contain orbital air cells, nasal cavity, and")
        print("  maxillary sinuses -- the exact region defacing tools target.")
        print("  Run this volume through `defacing/` or mri_deface to see the")
        print("  before/after comparison for the landing page demo.")
    print()

    if args.gpu:
        try:
            slices = generate_monai_slices(n_slices, args.size)
            print(f"Generated {len(slices)} slices via MONAI LDM.")
        except Exception as exc:
            print(f"MONAI generation failed ({exc})")
            print("Falling back to Shepp-Logan phantom...")
            slices = generate_phantom_slices(n_slices, args.size, seed=args.seed,
                                             with_face=with_face)
    else:
        slices = generate_phantom_slices(n_slices, args.size, seed=args.seed,
                                         with_face=with_face)
        print(f"Generated {len(slices)} Shepp-Logan phantom slices.")

    paths = write_dicom_series(slices, output_dir)

    print()
    print(f"Done. {len(paths)} DICOM files in: {output_dir}")
    print()
    print("Sample synthetic PHI tags AEGIS will de-identify:")
    for tag in list(SYNTHETIC_PHI.keys())[:8]:
        print(f"  {tag:40s} = {SYNTHETIC_PHI[tag]}")
    print(f"  ... and {len(SYNTHETIC_PHI) - 8} more identifiers")

    if args.defacing_demo:
        print()
        print("Next steps for defacing demo:")
        print(f"  1. Run the AEGIS defacing service against this directory:")
        print(f"       cd defacing && uvicorn app.main:app --port 8081")
        print(f"       # Then trigger via API or pass the DICOM dir directly")
        print(f"  2. Or run mri_deface directly:")
        print(f"       dcm2niix -o /tmp/nii {output_dir}")
        print(f"       mri_deface /tmp/nii/*.nii /path/to/talairach.gca /path/to/face.gca /tmp/defaced.nii")
        print(f"  3. Or use DeepDefacer (GPU):")
        print(f"       pip install deepdefacer")
        print(f"       deepdefacer --input /tmp/nii/*.nii --output /tmp/defaced/")


if __name__ == "__main__":
    main()
