"""MONAI Label backend — AI-powered medical image segmentation.

MONAI Label is an open-source framework for interactive and automatic
segmentation of medical images, built on top of PyTorch and MONAI.
It supports pre-trained models for brain, organ, and tumor segmentation.

Requires:
  - ``monailabel`` pip package (Apache 2.0)
  - Pre-trained model files in ``MONAI_MODEL_DIR``

References:
  Diaz-Pinto A et al. "MONAI Label: A framework for AI-assisted Interactive
  Labeling of 3D Medical Images." Medical Image Analysis, 2024.
  DOI: 10.1016/j.media.2024.103207
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
from .seg_utils import compute_label_volumes, compute_label_stats, find_any_nifti

log = logging.getLogger(__name__)


class MONAILabelBackend(AnalyticsBackend):
    """AI-based automatic segmentation via MONAI Label."""

    @property
    def name(self) -> str:
        return "monai_label"

    def available(self) -> bool:
        try:
            import monailabel  # noqa: F401
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
        out_dir = os.path.join(output_dir, "monai_label")
        os.makedirs(out_dir, exist_ok=True)

        # MONAI handles multiple contrasts
        nifti_path = find_any_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No NIfTI files found in BIDS directory",
            )

        log.info("MONAI Label: using %s", nifti_path)

        seg_path = os.path.join(out_dir, "monai_seg.nii.gz")
        model_dir = getattr(config, "MONAI_MODEL_DIR", "/opt/monai/models")
        model_name = getattr(config, "MONAI_MODEL_NAME", "segmentation")

        try:
            label_map = _run_monai_inference(
                nifti_path, seg_path, model_dir, model_name,
            )
        except ImportError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="monailabel package not installed",
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"MONAI Label inference failed: {e}",
            )

        if not os.path.isfile(seg_path):
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error="MONAI Label did not produce output segmentation",
            )

        # Extract volumes and stats
        roi_volumes = compute_label_volumes(seg_path, label_map)
        roi_stats = compute_label_stats(seg_path, nifti_path, label_map)

        metrics: dict = {
            "atlas": "monai_label",
            "model": model_name,
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
            "roi_stats": roi_stats,
        }

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [seg_path, summary_path]

        log.info(
            "MONAI Label complete for %s: %d ROIs in %.1fs (model=%s)",
            study_uid, len(roi_volumes), duration, model_name,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _run_monai_inference(
    input_path: str,
    output_path: str,
    model_dir: str,
    model_name: str,
) -> dict[int, str]:
    """Run MONAI Label inference and return label map.

    Uses the MONAI Label inference engine directly.  Falls back to
    numeric labels if no label map is available from the model.
    """
    from monailabel.interfaces.app import MONAILabelApp  # type: ignore[import-untyped]

    app = MONAILabelApp(app_dir=model_dir, studies="")
    request = {"image": input_path, "model": model_name}
    result = app.infer(request=request)

    # The inference result typically includes a label field
    if isinstance(result, dict):
        # Save the prediction to output_path
        if "prediction" in result:
            pred = result["prediction"]
            if hasattr(pred, "save"):
                pred.save(output_path)
            elif isinstance(pred, str) and os.path.isfile(pred):
                import shutil
                shutil.copy2(pred, output_path)

        # Try to extract label map from model config
        label_map = result.get("label_names", {})
        if label_map and isinstance(label_map, dict):
            # Convert string keys to int if needed
            return {int(k): v for k, v in label_map.items() if str(k).isdigit()}

    # Fall back to inferring labels from segmentation output
    return _infer_label_map(output_path)


def _infer_label_map(seg_path: str) -> dict[int, str]:
    """Infer a label map from a segmentation NIfTI with numeric labels."""
    if not os.path.isfile(seg_path):
        return {}

    img = nib.load(seg_path)
    data = np.asarray(img.dataobj, dtype=np.int32)
    unique_labels = np.unique(data)

    return {
        int(label_id): f"structure_{label_id}"
        for label_id in unique_labels
        if label_id != 0
    }
