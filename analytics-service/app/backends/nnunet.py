"""nnU-Net v2 backend — self-configuring deep learning segmentation.

nnU-Net automatically adapts its preprocessing, architecture, and training
to any given medical image segmentation task. It won the BraTS 2020 brain
tumor segmentation challenge and excels at arbitrary segmentation tasks when
provided with a pre-trained model.

Requires:
  - ``nnunetv2`` pip package (Apache 2.0)
  - Pre-trained model in ``NNUNET_MODEL_DIR``
  - PyTorch with CUDA support (recommended) or CPU-only

References:
  Isensee et al. "nnU-Net: a self-configuring method for deep learning-based
  biomedical image segmentation." Nature Methods, 2021.
  DOI: 10.1038/s41592-020-01008-z
"""

from __future__ import annotations

import json
import logging
import os
import shutil
import time

import nibabel as nib
import numpy as np

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import compute_label_volumes, compute_label_stats, find_t1w_nifti

log = logging.getLogger(__name__)

# Default label map for BraTS brain tumor segmentation
_BRATS_LABELS: dict[int, str] = {
    1: "necrotic_core",
    2: "peritumoral_edema",
    3: "enhancing_tumor",
}


class NNUNetBackend(AnalyticsBackend):
    """nnU-Net deep learning segmentation."""

    @property
    def name(self) -> str:
        return "nnunet"

    def available(self) -> bool:
        try:
            from nnunetv2.inference.predict_from_raw_data import nnUNetPredictor  # noqa: F401
            return True
        except ImportError:
            return False

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "nnunet")
        os.makedirs(out_dir, exist_ok=True)

        # Find T1w NIfTI (nnU-Net typically requires T1w for brain tasks)
        nifti_path = find_t1w_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        log.info("nnU-Net: using %s", nifti_path)

        # Configuration from env vars
        model_dir = getattr(config, "NNUNET_MODEL_DIR", "/opt/nnunet/models")
        dataset_id = getattr(config, "NNUNET_DATASET_ID", "Dataset001_BrainTumour")
        folds_str = getattr(config, "NNUNET_FOLDS", "all")
        use_gpu = getattr(config, "NNUNET_USE_GPU", True)

        # Parse folds
        if folds_str == "all":
            folds: tuple | str = "all"
        else:
            folds = tuple(int(f.strip()) for f in folds_str.split(","))

        # Load label map (look for labels.json in model directory)
        label_map = _load_label_map(model_dir, dataset_id)

        # Run prediction
        seg_path = os.path.join(out_dir, "segmentation.nii.gz")
        try:
            _run_prediction(
                nifti_path=nifti_path,
                output_path=seg_path,
                model_dir=model_dir,
                dataset_id=dataset_id,
                folds=folds,
                use_gpu=use_gpu,
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"nnU-Net prediction failed: {e}",
            )

        if not os.path.isfile(seg_path):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="nnU-Net did not produce output segmentation",
            )

        # Compute per-label volumes
        roi_volumes = compute_label_volumes(seg_path, label_map)

        # Compute per-label intensity stats from the original image
        roi_stats = compute_label_stats(seg_path, nifti_path, label_map)

        # Build metrics
        metrics: dict = {
            "atlas": "nnunet",
            "model": dataset_id,
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
            "roi_stats": roi_stats,
        }

        # Write summary JSON
        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, summary_path]

        log.info(
            "nnU-Net complete for %s: %d ROIs in %.1fs (model=%s)",
            study_uid,
            len(roi_volumes),
            duration,
            dataset_id,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _load_label_map(model_dir: str, dataset_id: str) -> dict[int, str]:
    """Load label mapping from the nnU-Net model directory.

    Looks for ``dataset.json`` or ``plans.json`` in the model folder,
    which contain ``labels`` mapping. Falls back to BraTS defaults.
    """
    # Try dataset.json in the model directory
    for subdir in ["", dataset_id]:
        dataset_json = os.path.join(model_dir, subdir, "dataset.json")
        if os.path.isfile(dataset_json):
            try:
                with open(dataset_json, encoding="utf-8") as f:
                    data = json.load(f)
                if "labels" in data:
                    labels = data["labels"]
                    # nnU-Net format: {"background": 0, "tumor": 1, ...}
                    result: dict[int, str] = {}
                    for label_name, label_id in labels.items():
                        if label_name.lower() == "background":
                            continue
                        result[int(label_id)] = label_name
                    if result:
                        log.info("Loaded %d labels from %s", len(result), dataset_json)
                        return result
            except Exception as e:
                log.warning("Failed to parse %s: %s", dataset_json, e)

    log.info("Using default BraTS label map")
    return _BRATS_LABELS.copy()


def _run_prediction(
    nifti_path: str,
    output_path: str,
    model_dir: str,
    dataset_id: str,
    folds: tuple | str,
    use_gpu: bool,
) -> None:
    """Run nnU-Net inference using the Python API."""
    from nnunetv2.inference.predict_from_raw_data import nnUNetPredictor

    predictor = nnUNetPredictor(
        perform_everything_on_device=use_gpu,
        device=_get_device(use_gpu),
    )

    # Locate model checkpoint
    model_folder = _find_model_folder(model_dir, dataset_id)
    predictor.initialize_from_trained_model_folder(
        model_folder,
        use_folds=folds if isinstance(folds, tuple) else None,
    )

    # nnU-Net expects input in a specific directory structure
    # Use predict_from_files with a temporary input directory
    import tempfile

    with tempfile.TemporaryDirectory() as tmp_in, tempfile.TemporaryDirectory() as tmp_out:
        # nnU-Net requires _0000 suffix for single-channel input
        input_name = "case_0000.nii.gz"
        os.symlink(os.path.abspath(nifti_path), os.path.join(tmp_in, input_name))

        predictor.predict_from_files(
            list_of_lists_or_source_folder=tmp_in,
            output_folder_or_list_of_truncated_output_files=tmp_out,
            save_probabilities=False,
        )

        # Move output to expected location
        out_files = [f for f in os.listdir(tmp_out) if f.endswith(".nii.gz")]
        if out_files:
            shutil.move(os.path.join(tmp_out, out_files[0]), output_path)


def _find_model_folder(model_dir: str, dataset_id: str) -> str:
    """Locate the nnU-Net model folder.

    Searches for common directory structures:
      - {model_dir}/{dataset_id}/nnUNetTrainer__nnUNetPlans__3d_fullres/
      - {model_dir}/{dataset_id}/
      - {model_dir}/
    """
    # Standard nnU-Net v2 structure
    for trainer in ["nnUNetTrainer__nnUNetPlans__3d_fullres",
                     "nnUNetTrainer__nnUNetPlans__3d_lowres",
                     "nnUNetTrainer__nnUNetPlans__2d"]:
        candidate = os.path.join(model_dir, dataset_id, trainer)
        if os.path.isdir(candidate):
            return candidate

    # Direct dataset directory
    candidate = os.path.join(model_dir, dataset_id)
    if os.path.isdir(candidate):
        return candidate

    # Model dir itself
    if os.path.isdir(model_dir):
        return model_dir

    raise FileNotFoundError(
        f"nnU-Net model not found in {model_dir} for {dataset_id}"
    )


def _get_device(use_gpu: bool):
    """Get the PyTorch device for inference."""
    import torch

    if use_gpu and torch.cuda.is_available():
        return torch.device("cuda")
    return torch.device("cpu")
