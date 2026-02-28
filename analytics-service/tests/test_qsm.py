"""Tests for the QSM backend."""

import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import nibabel as nib
import numpy as np
import pytest

from app.backends.qsm import QSMBackend, _compute_global_stats, _tkd_inversion
from app.backends.base import AnalyticsResult


class TestQSMAvailability:
    def setup_method(self):
        self.backend = QSMBackend()

    def test_name(self):
        assert self.backend.name == "qsm"

    def test_available_with_tgv_qsm(self):
        with patch("shutil.which", return_value="/usr/local/bin/tgv_qsm"):
            assert self.backend.available() is True

    def test_available_with_scipy(self):
        with patch("shutil.which", return_value=None):
            mock_scipy = MagicMock()
            with patch.dict("sys.modules", {"scipy": mock_scipy}):
                assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            with patch("builtins.__import__", side_effect=ImportError):
                assert self.backend.available() is False


class TestQSMAnalyze:
    def setup_method(self):
        self.backend = QSMBackend()

    def test_no_phase_nifti(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "phase" in result.error.lower()

    def test_tgv_binary_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            fmap = os.path.join(bids_dir, "sub-01", "fmap")
            os.makedirs(fmap)
            phase = os.path.join(fmap, "sub-01_phasediff.nii.gz")
            data = np.random.rand(4, 4, 4).astype(np.float32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), phase)

            with patch("shutil.which", return_value="/usr/bin/tgv_qsm"):
                with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False

    def test_pipeline_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            fmap = os.path.join(bids_dir, "sub-01", "fmap")
            os.makedirs(fmap)
            phase = os.path.join(fmap, "sub-01_phasediff.nii.gz")
            data = np.random.rand(4, 4, 4).astype(np.float32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), phase)

            with patch("shutil.which", return_value=None):
                with patch(
                    "app.backends.qsm._run_python_qsm",
                    side_effect=RuntimeError("qsm failed"),
                ):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "qsm failed" in result.error

    def test_successful_python_pipeline(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            fmap = os.path.join(bids_dir, "sub-01", "fmap")
            os.makedirs(fmap)
            phase = os.path.join(fmap, "sub-01_phasediff.nii.gz")
            mag = os.path.join(fmap, "sub-01_magnitude1.nii.gz")

            phase_data = np.random.rand(4, 4, 4).astype(np.float32) * 2 - 1
            mag_data = np.ones((4, 4, 4), dtype=np.float32) * 100
            affine = np.eye(4)
            nib.save(nib.Nifti1Image(phase_data, affine), phase)
            nib.save(nib.Nifti1Image(mag_data, affine), mag)

            def fake_python_qsm(phase_path, mag_path, output_path):
                qsm_data = np.random.rand(4, 4, 4).astype(np.float32) * 0.1
                nib.save(nib.Nifti1Image(qsm_data, np.eye(4)), output_path)

            with patch("shutil.which", return_value=None):
                with patch(
                    "app.backends.qsm._run_python_qsm",
                    side_effect=fake_python_qsm,
                ):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "qsm"
            assert result.metrics["atlas"] == "qsm"
            assert "mean" in result.metrics["global_susceptibility"]

    def test_uses_tgv_preference(self):
        backend = QSMBackend()
        with patch("shutil.which", return_value="/usr/bin/tgv_qsm"):
            assert backend._use_tgv() is True
        with patch("shutil.which", return_value=None):
            assert backend._use_tgv() is False


class TestTKDInversion:
    def test_basic_inversion(self):
        shape = (8, 8, 8)
        local_field = np.random.rand(*shape)
        mask = np.ones(shape)
        voxel_dims = (1.0, 1.0, 1.0)

        result = _tkd_inversion(local_field, mask, voxel_dims)
        assert result.shape == shape
        assert not np.all(result == 0)


class TestComputeGlobalStats:
    def test_basic_stats(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.array([[[0.1, 0.2], [0.3, 0.0]]]).astype(np.float32)
            path = os.path.join(d, "qsm.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), path)

            stats = _compute_global_stats(path)
            assert "mean" in stats
            assert "median" in stats
            assert "std" in stats
            assert stats["mean"] > 0

    def test_all_zeros(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 2, 2), dtype=np.float32)
            path = os.path.join(d, "qsm.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), path)

            stats = _compute_global_stats(path)
            assert stats["mean"] == 0.0
