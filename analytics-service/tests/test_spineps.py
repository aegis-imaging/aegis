"""Tests for the SPINEPS backend."""

import json
import os
import tempfile
from unittest.mock import MagicMock, patch

import nibabel as nib
import numpy as np
import pytest

from app.backends.spineps import SPINEPSBackend, _find_seg_file, _infer_instance_labels, _load_centroids
from app.backends.seg_utils import SPINEPS_SEMANTIC_LABELS


class TestSPINEPSAvailability:
    def setup_method(self):
        self.backend = SPINEPSBackend()

    def test_name(self):
        assert self.backend.name == "spineps"

    def test_available_with_package(self):
        mock_module = MagicMock()
        with patch.dict("sys.modules", {"spineps": mock_module, "spineps.seg_run": mock_module}):
            mock_module.process_img_nii = MagicMock()
            assert self.backend.available() is True

    def test_not_available(self):
        with patch.dict("sys.modules", {"spineps": None, "spineps.seg_run": None}):
            assert self.backend.available() is False


class TestSPINEPSAnalyze:
    def setup_method(self):
        self.backend = SPINEPSBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_import_error(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch(
                "app.backends.spineps.SPINEPSBackend.analyze",
                wraps=self.backend.analyze,
            ):
                # Simulate ImportError during analysis
                with patch("builtins.__import__", side_effect=ImportError("no spineps")):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False

    def test_no_output_produced(self):
        """SPINEPS runs but produces no output segmentation files."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_process = MagicMock()
            mock_model = MagicMock()

            with patch("app.backends.spineps.SPINEPSBackend.available", return_value=True):
                with patch.dict("sys.modules", {
                    "spineps": MagicMock(),
                    "spineps.seg_model": MagicMock(get_segmentation_model=MagicMock(return_value=mock_model)),
                    "spineps.seg_run": MagicMock(process_img_nii=mock_process),
                }):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "did not produce" in result.error

    def test_successful_analysis(self):
        """Mock SPINEPS producing both semantic and instance segmentations."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            # Create input NIfTI
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            data = np.ones((10, 10, 10), dtype=np.float32) * 80
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            def fake_process(img_path, output_dir, model_semantic, model_instance, **kwargs):
                # Create semantic segmentation
                sem_data = np.zeros((10, 10, 10), dtype=np.int32)
                sem_data[0:3, :, :] = 1   # vertebral_corpus
                sem_data[3:5, :, :] = 12  # intervertebral_disc
                sem_data[5:6, :, :] = 13  # spinal_cord

                sem_path = os.path.join(output_dir, "sub-01_seg-spine.nii.gz")
                nib.save(nib.Nifti1Image(sem_data, np.eye(4)), sem_path)

                # Create instance segmentation
                inst_data = np.zeros((10, 10, 10), dtype=np.int32)
                inst_data[0:3, :, :] = 20  # L1
                inst_data[3:5, :, :] = 21  # L2
                inst_data[7:9, :, :] = 22  # L3

                inst_path = os.path.join(output_dir, "sub-01_seg-vert.nii.gz")
                nib.save(nib.Nifti1Image(inst_data, np.eye(4)), inst_path)

            mock_model = MagicMock()
            mock_get_model = MagicMock(return_value=mock_model)

            with patch.dict("sys.modules", {
                "spineps": MagicMock(),
                "spineps.seg_model": MagicMock(get_segmentation_model=mock_get_model),
                "spineps.seg_run": MagicMock(process_img_nii=fake_process),
            }):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "spineps"
            assert result.metrics["atlas"] == "spineps"
            assert "vertebral_corpus" in result.metrics["semantic_volumes"]
            assert "intervertebral_disc" in result.metrics["semantic_volumes"]
            assert len(result.metrics["instance_vertebrae"]) > 0

    def test_model_selection(self):
        """When SPINEPS_MODEL=t1w, should request t1w model."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            called_models = []

            def fake_get_model(name):
                called_models.append(name)
                return MagicMock()

            with patch("app.config.SPINEPS_MODEL", "t1w"):
                with patch.dict("sys.modules", {
                    "spineps": MagicMock(),
                    "spineps.seg_model": MagicMock(get_segmentation_model=fake_get_model),
                    "spineps.seg_run": MagicMock(process_img_nii=MagicMock()),
                }):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            # Model name should be t1w (first call) and instance (second call)
            assert "t1w" in called_models


class TestSPINEPSSemanticLabels:
    def test_all_14_labels(self):
        assert len(SPINEPS_SEMANTIC_LABELS) == 14

    def test_key_structures(self):
        names = set(SPINEPS_SEMANTIC_LABELS.values())
        assert "vertebral_corpus" in names
        assert "vertebral_arch" in names
        assert "intervertebral_disc" in names
        assert "spinal_cord" in names
        assert "spinal_canal" in names
        assert "spinous_process" in names


class TestFindSegFile:
    def test_finds_matching_file(self):
        with tempfile.TemporaryDirectory() as d:
            seg = os.path.join(d, "sub-01_seg-spine.nii.gz")
            with open(seg, "wb") as f:
                f.write(b"\x00" * 10)
            assert _find_seg_file(d, "seg-spine") == seg

    def test_returns_none_no_match(self):
        with tempfile.TemporaryDirectory() as d:
            assert _find_seg_file(d, "seg-spine") is None


class TestInferInstanceLabels:
    def test_maps_known_vertebrae(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((3, 1, 1), dtype=np.int32)
            data[0] = 1   # C1
            data[1] = 20  # L1
            data[2] = 25  # sacrum
            path = os.path.join(d, "seg.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), path)

            labels = _infer_instance_labels(path)
            assert labels[1] == "C1"
            assert labels[20] == "L1"
            assert labels[25] == "sacrum"

    def test_unknown_label_gets_fallback_name(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 1, 1), dtype=np.int32)
            data[0] = 99
            path = os.path.join(d, "seg.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), path)

            labels = _infer_instance_labels(path)
            assert labels[99] == "vertebra_99"


class TestLoadCentroids:
    def test_loads_valid_json_list(self):
        with tempfile.TemporaryDirectory() as d:
            centroids = [{"label": "C1", "x": 10, "y": 20, "z": 30}]
            path = os.path.join(d, "centroids.json")
            with open(path, "w") as f:
                json.dump(centroids, f)

            result = _load_centroids(d)
            assert len(result) == 1
            assert result[0]["label"] == "C1"

    def test_returns_empty_no_file(self):
        with tempfile.TemporaryDirectory() as d:
            assert _load_centroids(d) == []

    def test_handles_invalid_json(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "centroids.json")
            with open(path, "w") as f:
                f.write("not json{{{")

            assert _load_centroids(d) == []
