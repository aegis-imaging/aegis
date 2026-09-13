"""Tests for the MONAI Label backend."""

import os
import tempfile
from unittest.mock import MagicMock, patch

import pytest

from app.backends.monai_label import MONAILabelBackend, _infer_label_map
from app.backends.base import AnalyticsResult


class TestMONAIAvailability:
    def setup_method(self):
        self.backend = MONAILabelBackend()

    def test_name(self):
        assert self.backend.name == "monai_label"

    def test_available_with_package(self):
        mock_module = MagicMock()
        with patch.dict("sys.modules", {"monailabel": mock_module}):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch("builtins.__import__", side_effect=ImportError):
            assert self.backend.available() is False


class TestMONAIAnalyze:
    def setup_method(self):
        self.backend = MONAILabelBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_import_error(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch(
                "app.backends.monai_label._run_monai_inference",
                side_effect=ImportError("monailabel not installed"),
            ):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "not installed" in result.error

    def test_inference_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch(
                "app.backends.monai_label._run_monai_inference",
                side_effect=RuntimeError("model crashed"),
            ):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "model crashed" in result.error

    def test_no_output_produced(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch(
                "app.backends.monai_label._run_monai_inference",
                return_value={},
            ):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "not produce" in result.error

    def test_successful_analysis(self):
        import nibabel as nib
        import numpy as np

        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            data = np.ones((4, 4, 4), dtype=np.float32) * 100
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            def fake_inference(input_path, output_path, model_dir, model_name):
                seg_data = np.zeros((4, 4, 4), dtype=np.int32)
                seg_data[0, 0, 0] = 1
                seg_data[1, 0, 0] = 2
                nib.save(nib.Nifti1Image(seg_data, np.eye(4)), output_path)
                return {1: "tumor", 2: "edema"}

            with patch(
                "app.backends.monai_label._run_monai_inference",
                side_effect=fake_inference,
            ):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "monai_label"
            assert result.metrics["atlas"] == "monai_label"
            assert result.metrics["roi_count"] == 2
            assert "tumor" in result.metrics["roi_volumes"]


class TestInferLabelMap:
    def test_infers_from_nifti(self):
        import nibabel as nib
        import numpy as np

        with tempfile.TemporaryDirectory() as d:
            seg_data = np.zeros((3, 3, 3), dtype=np.int32)
            seg_data[0, 0, 0] = 1
            seg_data[1, 0, 0] = 5
            seg_path = os.path.join(d, "seg.nii.gz")
            nib.save(nib.Nifti1Image(seg_data, np.eye(4)), seg_path)

            label_map = _infer_label_map(seg_path)
            assert 1 in label_map
            assert 5 in label_map
            assert 0 not in label_map

    def test_missing_file(self):
        assert _infer_label_map("/nonexistent.nii.gz") == {}
