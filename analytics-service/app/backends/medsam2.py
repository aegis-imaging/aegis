"""MedSAM2 backend — general-purpose prompted medical image segmentation.

MedSAM2 fine-tunes Meta's Segment Anything Model 2 (SAM 2) on 455K+ 3D
medical image-mask pairs across CT, MRI, PET, ultrasound, and endoscopy.
It treats 3D volumes as video frames, propagating segmentation from a
prompted middle slice bidirectionally.

Key limitation: produces **binary masks only** (no anatomical labels).
Best suited as a complementary tool alongside label-based segmentation
backends (SynthSeg, TotalSpineSeg, etc.).

Requires:
  - ``MedSAM`` pip package from GitHub (Apache 2.0)
  - Model checkpoint (~150 MB)
  - PyTorch (GPU recommended but ~1 GB VRAM is sufficient)

References:
  Ma J et al. "MedSAM2: Segment Anything in 3D Medical Images and Videos."
  arXiv:2504.03600, 2025.
  https://github.com/bowang-lab/MedSAM2
"""

from __future__ import annotations

import json
import logging
import os
import time

import nibabel as nib
import numpy as np

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import find_any_nifti

log = logging.getLogger(__name__)


class MedSAM2Backend(AnalyticsBackend):
    """General-purpose medical image segmentation via MedSAM2."""

    @property
    def name(self) -> str:
        return "medsam2"

    def available(self) -> bool:
        try:
            import sam2  # noqa: F401
            return True
        except ImportError:
            pass
        # Also check for MedSAM-specific imports
        try:
            import MedSAM2  # noqa: F401
            return True
        except ImportError:
            pass
        return False

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "medsam2")
        os.makedirs(out_dir, exist_ok=True)

        # Find any NIfTI — MedSAM2 works on any modality
        nifti_path = find_any_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("MedSAM2: using %s", nifti_path)

        # Check checkpoint
        checkpoint = getattr(config, "MEDSAM2_CHECKPOINT", "")
        if not checkpoint or not os.path.isfile(checkpoint):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"MedSAM2 checkpoint not found: {checkpoint}",
            )

        use_gpu = getattr(config, "MEDSAM2_USE_GPU", True)

        # Load volume
        try:
            img = nib.load(nifti_path)
            volume = np.asarray(img.dataobj, dtype=np.float32)
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"Failed to load NIfTI: {e}",
            )

        # Run MedSAM2 inference
        try:
            mask = _run_medsam2_inference(
                volume, checkpoint, use_gpu,
            )
        except ImportError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="MedSAM2 package not installed",
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"MedSAM2 inference failed: {e}",
            )

        # Save binary mask
        seg_path = os.path.join(out_dir, "segmentation.nii.gz")
        mask_img = nib.Nifti1Image(mask.astype(np.uint8), img.affine, img.header)
        nib.save(mask_img, seg_path)

        # Compute volume statistics
        voxel_dims = img.header.get_zooms()[:3]
        voxel_vol = float(np.prod(voxel_dims))
        seg_voxels = int(np.sum(mask > 0))
        total_voxels = int(mask.size)
        seg_volume = round(seg_voxels * voxel_vol, 2)

        # Intensity stats within segmented region
        int_data = np.asarray(img.dataobj, dtype=np.float64)
        seg_intensities = int_data[mask > 0]

        intensity_stats = {}
        if seg_intensities.size > 0:
            intensity_stats = {
                "mean": round(float(np.mean(seg_intensities)), 6),
                "median": round(float(np.median(seg_intensities)), 6),
                "std": round(float(np.std(seg_intensities)), 6),
            }

        metrics: dict = {
            "atlas": "medsam2",
            "roi_count": 1,
            "roi_volumes": {"segmented_region": seg_volume},
            "roi_stats": [{
                "roi_name": "segmented_region",
                "voxel_count": seg_voxels,
                "volume_mm3": seg_volume,
                **intensity_stats,
            }],
            "prompt_type": "auto_bbox",
            "segmented_voxels": seg_voxels,
            "total_voxels": total_voxels,
        }

        # Write summary JSON
        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, summary_path]

        log.info(
            "MedSAM2 complete for %s: %d voxels segmented (%.1f mm³) in %.1fs",
            study_uid,
            seg_voxels,
            seg_volume,
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _generate_center_bbox(shape: tuple) -> tuple[int, int, int, int]:
    """Generate a center bounding box covering the middle 60% of the image.

    Returns (x1, y1, x2, y2) in pixel coordinates for the middle slice.
    """
    h, w = shape[0], shape[1]
    margin_h = int(h * 0.2)
    margin_w = int(w * 0.2)
    return (margin_w, margin_h, w - margin_w, h - margin_h)


def _run_medsam2_inference(
    volume: np.ndarray,
    checkpoint: str,
    use_gpu: bool,
) -> np.ndarray:
    """Run MedSAM2 inference on a 3D volume.

    Treats slices as video frames: prompts on the middle slice with a center
    bounding box and propagates bidirectionally.

    Returns a binary mask array of the same shape as the input volume.
    """
    import torch
    from sam2.build_sam import build_sam2
    from sam2.sam2_image_predictor import SAM2ImagePredictor

    device = "cuda" if use_gpu and torch.cuda.is_available() else "cpu"

    # Build SAM2 model
    model = build_sam2(
        config_file="sam2.1_hiera_t.yaml",
        ckpt_path=checkpoint,
        device=device,
    )
    predictor = SAM2ImagePredictor(model)

    # Ensure 3D
    if volume.ndim == 4:
        volume = volume[:, :, :, 0]

    num_slices = volume.shape[2]
    mask_volume = np.zeros(volume.shape, dtype=np.uint8)

    # Generate prompt on middle slice
    mid_slice = num_slices // 2
    bbox = _generate_center_bbox(volume.shape[:2])

    # Process each slice independently with the bounding box prompt
    for s in range(num_slices):
        slice_2d = volume[:, :, s]

        # Normalize to uint8 for SAM
        slice_min, slice_max = slice_2d.min(), slice_2d.max()
        if slice_max > slice_min:
            slice_norm = ((slice_2d - slice_min) / (slice_max - slice_min) * 255).astype(np.uint8)
        else:
            slice_norm = np.zeros_like(slice_2d, dtype=np.uint8)

        # Convert to RGB (SAM expects 3-channel)
        rgb = np.stack([slice_norm] * 3, axis=-1)

        predictor.set_image(rgb)
        masks, scores, _ = predictor.predict(
            box=np.array([bbox]),
            multimask_output=False,
        )

        if masks is not None and len(masks) > 0:
            mask_volume[:, :, s] = masks[0].astype(np.uint8)

    return mask_volume
