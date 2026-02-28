"""Tests for the MedSAM2 backend."""

import os
import tempfile
from unittest.mock import MagicMock, patch

import nibabel as nib
import numpy as np
import pytest

from app.backends.medsam2 import MedSAM2Backend, _generate_center_bbox


class TestMedSAM2Availability:
    def setup_method(self):
        self.backend = MedSAM2Backend()

    def test_name(self):
        assert self.backend.name == "medsam2"

    def test_available_with_sam2(self):
        with patch.dict("sys.modules", {"sam2": MagicMock()}):
            assert self.backend.available() is True

    def test_available_with_medsam2(self):
        with patch.dict("sys.modules", {"sam2": None, "MedSAM2": MagicMock()}):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch.dict("sys.modules", {"sam2": None, "MedSAM2": None}):
            assert self.backend.available() is False


class TestMedSAM2Analyze:
    def setup_method(self):
        self.backend = MedSAM2Backend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_checkpoint_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            data = np.ones((4, 4, 4), dtype=np.float32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            with patch("app.config.MEDSAM2_CHECKPOINT", "/nonexistent/checkpoint.pth"):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "checkpoint not found" in result.error

    def test_successful_analysis(self):
        """Mock MedSAM2 inference producing a binary mask."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            # Create input NIfTI
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            data = np.ones((8, 8, 8), dtype=np.float32) * 100
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            # Create a fake checkpoint file
            ckpt = os.path.join(out_dir, "checkpoint.pth")
            with open(ckpt, "wb") as f:
                f.write(b"\x00" * 100)

            # Mock inference to return a binary mask
            def fake_inference(volume, checkpoint, use_gpu):
                mask = np.zeros(volume.shape[:3], dtype=np.uint8)
                mask[2:6, 2:6, 2:6] = 1  # Center region
                return mask

            with patch("app.config.MEDSAM2_CHECKPOINT", ckpt):
                with patch(
                    "app.backends.medsam2._run_medsam2_inference",
                    side_effect=fake_inference,
                ):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "medsam2"
            assert result.metrics["atlas"] == "medsam2"
            assert result.metrics["roi_count"] == 1
            assert "segmented_region" in result.metrics["roi_volumes"]
            assert result.metrics["prompt_type"] == "auto_bbox"
            assert result.metrics["segmented_voxels"] > 0
            assert result.metrics["total_voxels"] == 8 * 8 * 8

            # Verify output files exist
            seg_path = os.path.join(out_dir, "medsam2", "segmentation.nii.gz")
            assert os.path.isfile(seg_path)

            summary_path = os.path.join(out_dir, "medsam2", "summary.json")
            assert os.path.isfile(summary_path)

    def test_import_error(self):
        """MedSAM2 package not installed — inference fails."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            data = np.ones((4, 4, 4), dtype=np.float32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            ckpt = os.path.join(out_dir, "checkpoint.pth")
            with open(ckpt, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.config.MEDSAM2_CHECKPOINT", ckpt):
                with patch(
                    "app.backends.medsam2._run_medsam2_inference",
                    side_effect=ImportError("No module named 'sam2'"),
                ):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "not installed" in result.error

    def test_inference_failure(self):
        """Runtime error during inference."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            data = np.ones((4, 4, 4), dtype=np.float32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            ckpt = os.path.join(out_dir, "checkpoint.pth")
            with open(ckpt, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.config.MEDSAM2_CHECKPOINT", ckpt):
                with patch(
                    "app.backends.medsam2._run_medsam2_inference",
                    side_effect=RuntimeError("CUDA out of memory"),
                ):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "inference failed" in result.error


class TestGenerateCenterBbox:
    def test_center_bbox_dimensions(self):
        x1, y1, x2, y2 = _generate_center_bbox((100, 200))
        # 20% margin on width=200 → x1=40, x2=160
        assert x1 == 40
        assert x2 == 160
        # 20% margin on height=100 → y1=20, y2=80
        assert y1 == 20
        assert y2 == 80

    def test_small_image(self):
        x1, y1, x2, y2 = _generate_center_bbox((10, 10))
        assert x1 == 2
        assert y1 == 2
        assert x2 == 8
        assert y2 == 8

    def test_covers_sixty_percent(self):
        """The bbox should cover approximately 60% of each dimension."""
        h, w = 100, 100
        x1, y1, x2, y2 = _generate_center_bbox((h, w))
        bbox_h = y2 - y1
        bbox_w = x2 - x1
        assert bbox_h == 60
        assert bbox_w == 60
