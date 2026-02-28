"""Tests for the TotalSegmentator backend."""

import os
import tempfile
from unittest.mock import patch, MagicMock

import nibabel as nib
import numpy as np
import pytest

from app.backends.totalsegmentator import (
    TotalSegmentatorBackend,
    _compute_stats,
    _compute_volumes,
    _find_and_rename_combined,
    _infer_label_map_from_output,
)
from app.backends.base import AnalyticsResult


def _make_nifti(path: str, data: np.ndarray, voxel_dims=(1.0, 1.0, 1.0)):
    affine = np.diag(list(voxel_dims) + [1.0])
    img = nib.Nifti1Image(data, affine)
    nib.save(img, path)
    return path


class TestTotalSegAvailability:
    def setup_method(self):
        self.backend = TotalSegmentatorBackend()

    def test_name(self):
        assert self.backend.name == "totalsegmentator"

    def test_available_with_python_package(self):
        mock_module = MagicMock()
        with patch.dict("sys.modules", {"totalsegmentator": mock_module}):
            assert self.backend.available() is True

    def test_available_with_cli(self):
        """When the Python package is missing but CLI exists, should be available."""
        original_import = __builtins__.__import__ if hasattr(__builtins__, '__import__') else __import__

        def selective_import(name, *args, **kwargs):
            if name == "totalsegmentator":
                raise ImportError("no module")
            return original_import(name, *args, **kwargs)

        with patch("builtins.__import__", side_effect=selective_import):
            with patch("shutil.which", return_value="/usr/local/bin/TotalSegmentator"):
                assert self.backend.available() is True

    def test_not_available(self):
        """When neither the Python package nor CLI exist, should not be available."""
        original_import = __builtins__.__import__ if hasattr(__builtins__, '__import__') else __import__

        def selective_import(name, *args, **kwargs):
            if name == "totalsegmentator":
                raise ImportError("no module")
            return original_import(name, *args, **kwargs)

        with patch("builtins.__import__", side_effect=selective_import):
            with patch("shutil.which", return_value=None):
                assert self.backend.available() is False


class TestTotalSegAnalyze:
    def setup_method(self):
        self.backend = TotalSegmentatorBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_totalseg_exception(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.totalsegmentator._run_totalsegmentator",
                        side_effect=RuntimeError("model download failed")):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is False
            assert "model download failed" in result.error

    def test_no_output_produced(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.totalsegmentator._run_totalsegmentator", return_value={}):
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
            affine = np.diag([2.0, 2.0, 2.0, 1.0])
            nib.save(nib.Nifti1Image(t1w_data, affine), t1w_path)

            # Create segmentation output
            seg_data = np.zeros((4, 4, 4), dtype=np.int32)
            seg_data[0:2, :, :] = 1  # brain: 32 voxels
            seg_data[2:4, :, :] = 2  # skull: 32 voxels

            label_map = {1: "brain", 2: "skull"}

            def fake_run(input_path, output_dir, combined_output, task, fast, use_gpu):
                nib.save(nib.Nifti1Image(seg_data, affine), combined_output)
                return label_map

            with patch("app.backends.totalsegmentator._run_totalsegmentator", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "totalsegmentator"
            assert result.metrics["atlas"] == "totalsegmentator"
            assert result.metrics["roi_count"] == 2
            assert "brain" in result.metrics["roi_volumes"]
            assert "skull" in result.metrics["roi_volumes"]
            # 32 voxels × 8mm³ = 256.0
            assert result.metrics["roi_volumes"]["brain"] == 256.0
            assert len(result.metrics["roi_stats"]) == 2

            # Check outputs
            ts_dir = os.path.join(out_dir, "totalsegmentator")
            assert os.path.isfile(os.path.join(ts_dir, "segmentation.nii.gz"))
            assert os.path.isfile(os.path.join(ts_dir, "summary.json"))


class TestInferLabelMap:
    def test_basic_inference(self):
        with tempfile.TemporaryDirectory() as d:
            seg_data = np.zeros((3, 3, 3), dtype=np.int32)
            seg_data[0, 0, 0] = 1
            seg_data[1, 1, 1] = 5
            seg_data[2, 2, 2] = 10

            path = os.path.join(d, "seg.nii.gz")
            _make_nifti(path, seg_data)

            label_map = _infer_label_map_from_output(path)

        assert len(label_map) == 3
        assert label_map[1] == "structure_1"
        assert label_map[5] == "structure_5"
        assert label_map[10] == "structure_10"
        assert 0 not in label_map

    def test_missing_file(self):
        assert _infer_label_map_from_output("/nonexistent/file.nii.gz") == {}


class TestComputeVolumes:
    def test_with_label_map(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 2, 2), dtype=np.int32)
            data[0, 0, 0] = 1
            data[0, 0, 1] = 1
            data[1, 0, 0] = 2

            path = os.path.join(d, "seg.nii.gz")
            _make_nifti(path, data, voxel_dims=(1.0, 1.0, 1.0))

            label_map = {1: "brain", 2: "skull"}
            volumes = _compute_volumes(path, label_map)

        assert volumes["brain"] == 2.0
        assert volumes["skull"] == 1.0

    def test_without_label_map_infers(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 2, 2), dtype=np.int32)
            data[0, 0, 0] = 3

            path = os.path.join(d, "seg.nii.gz")
            _make_nifti(path, data)

            volumes = _compute_volumes(path, {})

        assert "structure_3" in volumes


class TestComputeStats:
    def test_basic_stats(self):
        with tempfile.TemporaryDirectory() as d:
            seg_data = np.zeros((3, 3, 3), dtype=np.int32)
            seg_data[0, :, :] = 1  # 9 voxels

            int_data = np.ones((3, 3, 3), dtype=np.float64) * 50.0
            int_data[0, 0, 0] = 100.0  # One bright voxel in region 1

            seg_path = os.path.join(d, "seg.nii.gz")
            int_path = os.path.join(d, "t1w.nii.gz")
            _make_nifti(seg_path, seg_data)
            _make_nifti(int_path, int_data)

            label_map = {1: "brain"}
            stats = _compute_stats(seg_path, int_path, label_map)

        assert len(stats) == 1
        s = stats[0]
        assert s["roi_name"] == "brain"
        assert s["voxel_count"] == 9
        assert s["mean"] > 50.0  # Mix of 50 and 100


class TestFindAndRenameCombined:
    def test_finds_total_named_file(self):
        with tempfile.TemporaryDirectory() as d:
            total = os.path.join(d, "total.nii.gz")
            other = os.path.join(d, "brain.nii.gz")
            for f in [total, other]:
                with open(f, "wb") as fp:
                    fp.write(b"\x00" * 10)

            target = os.path.join(d, "segmentation.nii.gz")
            _find_and_rename_combined(d, target)

            assert os.path.isfile(target)

    def test_falls_back_to_largest(self):
        with tempfile.TemporaryDirectory() as d:
            small = os.path.join(d, "small.nii.gz")
            large = os.path.join(d, "large.nii.gz")
            with open(small, "wb") as f:
                f.write(b"\x00" * 10)
            with open(large, "wb") as f:
                f.write(b"\x00" * 1000)

            target = os.path.join(d, "segmentation.nii.gz")
            _find_and_rename_combined(d, target)

            assert os.path.isfile(target)
            assert os.path.getsize(target) == 1000
