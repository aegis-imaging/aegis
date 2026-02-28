"""Tests for the nnU-Net backend."""

import json
import os
import tempfile
from unittest.mock import patch, MagicMock

import nibabel as nib
import numpy as np
import pytest

from app.backends.nnunet import (
    NNUNetBackend,
    _BRATS_LABELS,
    _find_model_folder,
    _load_label_map,
)
from app.backends.base import AnalyticsResult


class TestNNUNetAvailability:
    def setup_method(self):
        self.backend = NNUNetBackend()

    def test_name(self):
        assert self.backend.name == "nnunet"

    def test_available_with_package(self):
        mock_module = MagicMock()
        mock_pred = MagicMock()
        with patch.dict("sys.modules", {
            "nnunetv2": mock_module,
            "nnunetv2.inference": mock_module,
            "nnunetv2.inference.predict_from_raw_data": mock_pred,
        }):
            mock_pred.nnUNetPredictor = MagicMock
            assert self.backend.available() is True

    def test_not_available_without_package(self):
        with patch("builtins.__import__", side_effect=ImportError):
            assert self.backend.available() is False


class TestNNUNetAnalyze:
    def setup_method(self):
        self.backend = NNUNetBackend()

    def test_no_t1w_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w" in result.error

    def test_prediction_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.nnunet._load_label_map", return_value=_BRATS_LABELS.copy()):
                with patch("app.backends.nnunet._run_prediction", side_effect=RuntimeError("CUDA OOM")):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is False
            assert "CUDA OOM" in result.error

    def test_no_output_produced(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.nnunet._load_label_map", return_value=_BRATS_LABELS.copy()):
                with patch("app.backends.nnunet._run_prediction"):
                    # _run_prediction succeeds but produces no file
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is False
            assert "did not produce" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)

            # Create real T1w NIfTI
            t1w_data = np.random.rand(4, 4, 4).astype(np.float32) * 100
            t1w_path = os.path.join(anat, "sub-01_T1w.nii.gz")
            affine = np.diag([1.0, 1.0, 1.0, 1.0])
            nib.save(nib.Nifti1Image(t1w_data, affine), t1w_path)

            # Create fake segmentation output
            seg_data = np.zeros((4, 4, 4), dtype=np.int32)
            seg_data[0:2, 0:2, 0:2] = 1  # necrotic_core: 8 voxels
            seg_data[2:4, 0:2, 0:2] = 2  # peritumoral_edema: 8 voxels

            def fake_prediction(nifti_path, output_path, **kwargs):
                nib.save(nib.Nifti1Image(seg_data, affine), output_path)

            with patch("app.backends.nnunet._load_label_map", return_value=_BRATS_LABELS.copy()):
                with patch("app.backends.nnunet._run_prediction", side_effect=fake_prediction):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "nnunet"
            assert result.metrics["atlas"] == "nnunet"
            assert result.metrics["roi_count"] == 2
            assert "necrotic_core" in result.metrics["roi_volumes"]
            assert "peritumoral_edema" in result.metrics["roi_volumes"]
            # 8 voxels × 1mm³ = 8.0 mm³
            assert result.metrics["roi_volumes"]["necrotic_core"] == 8.0
            assert len(result.metrics["roi_stats"]) == 2

            # Check outputs
            nnunet_dir = os.path.join(out_dir, "nnunet")
            assert os.path.isfile(os.path.join(nnunet_dir, "segmentation.nii.gz"))
            assert os.path.isfile(os.path.join(nnunet_dir, "summary.json"))


class TestLoadLabelMap:
    def test_loads_from_dataset_json(self):
        with tempfile.TemporaryDirectory() as model_dir:
            dataset_json = os.path.join(model_dir, "dataset.json")
            labels = {
                "labels": {
                    "background": 0,
                    "necrotic": 1,
                    "edema": 2,
                    "enhancing": 3,
                }
            }
            with open(dataset_json, "w") as f:
                json.dump(labels, f)

            result = _load_label_map(model_dir, "Dataset001")

        assert len(result) == 3
        assert result[1] == "necrotic"
        assert result[2] == "edema"
        assert result[3] == "enhancing"
        assert 0 not in result  # background excluded

    def test_loads_from_subdirectory(self):
        with tempfile.TemporaryDirectory() as model_dir:
            sub = os.path.join(model_dir, "Dataset001")
            os.makedirs(sub)
            dataset_json = os.path.join(sub, "dataset.json")
            labels = {"labels": {"background": 0, "tumor": 1}}
            with open(dataset_json, "w") as f:
                json.dump(labels, f)

            result = _load_label_map(model_dir, "Dataset001")

        assert result[1] == "tumor"

    def test_falls_back_to_brats_defaults(self):
        with tempfile.TemporaryDirectory() as model_dir:
            result = _load_label_map(model_dir, "Dataset001")

        assert result == _BRATS_LABELS

    def test_falls_back_on_invalid_json(self):
        with tempfile.TemporaryDirectory() as model_dir:
            dataset_json = os.path.join(model_dir, "dataset.json")
            with open(dataset_json, "w") as f:
                f.write("not valid json")

            result = _load_label_map(model_dir, "Dataset001")

        assert result == _BRATS_LABELS


class TestFindModelFolder:
    def test_standard_structure(self):
        with tempfile.TemporaryDirectory() as model_dir:
            path = os.path.join(model_dir, "Dataset001", "nnUNetTrainer__nnUNetPlans__3d_fullres")
            os.makedirs(path)

            result = _find_model_folder(model_dir, "Dataset001")
            assert result == path

    def test_lowres_trainer(self):
        with tempfile.TemporaryDirectory() as model_dir:
            path = os.path.join(model_dir, "Dataset001", "nnUNetTrainer__nnUNetPlans__3d_lowres")
            os.makedirs(path)

            result = _find_model_folder(model_dir, "Dataset001")
            assert result == path

    def test_direct_dataset_dir(self):
        with tempfile.TemporaryDirectory() as model_dir:
            path = os.path.join(model_dir, "Dataset001")
            os.makedirs(path)

            result = _find_model_folder(model_dir, "Dataset001")
            assert result == path

    def test_model_dir_itself(self):
        with tempfile.TemporaryDirectory() as model_dir:
            result = _find_model_folder(model_dir, "Nonexistent")
            assert result == model_dir

    def test_raises_when_model_dir_missing(self):
        with pytest.raises(FileNotFoundError, match="nnU-Net model not found"):
            _find_model_folder("/nonexistent/path", "Dataset001")
