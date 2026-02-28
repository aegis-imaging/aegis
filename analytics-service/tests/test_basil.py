"""Tests for the BASIL / oxford_asl backend."""

import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import nibabel as nib
import numpy as np
import pytest

from app.backends.basil import BASILBackend, _compute_cbf_stats, _find_cbf_output
from app.backends.base import AnalyticsResult


class TestBASILAvailability:
    def setup_method(self):
        self.backend = BASILBackend()

    def test_name(self):
        assert self.backend.name == "basil"

    def test_available_with_oxford_asl(self):
        with patch("shutil.which", return_value="/usr/local/fsl/bin/oxford_asl"):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False


class TestBASILAnalyze:
    def setup_method(self):
        self.backend = BASILBackend()

    def test_no_asl_nifti(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No ASL NIfTI" in result.error

    def test_binary_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            perf = os.path.join(bids_dir, "sub-01", "perf")
            os.makedirs(perf)
            asl = os.path.join(perf, "sub-01_asl.nii.gz")
            with open(asl, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "not found" in result.error

    def test_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            perf = os.path.join(bids_dir, "sub-01", "perf")
            os.makedirs(perf)
            asl = os.path.join(perf, "sub-01_asl.nii.gz")
            with open(asl, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 60)):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "timed out" in result.error

    def test_nonzero_exit(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            perf = os.path.join(bids_dir, "sub-01", "perf")
            os.makedirs(perf)
            asl = os.path.join(perf, "sub-01_asl.nii.gz")
            with open(asl, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "error"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            perf = os.path.join(bids_dir, "sub-01", "perf")
            os.makedirs(perf)
            asl = os.path.join(perf, "sub-01_asl.nii.gz")
            with open(asl, "wb") as f:
                f.write(b"\x00" * 100)

            def fake_run(cmd, **kwargs):
                basil_dir = os.path.join(out_dir, "basil")
                ns_dir = os.path.join(basil_dir, "native_space")
                os.makedirs(ns_dir, exist_ok=True)
                cbf_path = os.path.join(ns_dir, "perfusion_calib.nii.gz")
                cbf_data = np.random.rand(4, 4, 4).astype(np.float32) * 50 + 20
                nib.save(nib.Nifti1Image(cbf_data, np.eye(4)), cbf_path)

                mock = MagicMock()
                mock.returncode = 0
                mock.stderr = ""
                return mock

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "basil"
            assert result.metrics["atlas"] == "basil"
            assert result.metrics["global_cbf"]["mean"] > 0


class TestFindCBFOutput:
    def test_finds_calibrated(self):
        with tempfile.TemporaryDirectory() as d:
            ns_dir = os.path.join(d, "native_space")
            os.makedirs(ns_dir)
            cbf = os.path.join(ns_dir, "perfusion_calib.nii.gz")
            with open(cbf, "wb") as f:
                f.write(b"\x00")
            assert _find_cbf_output(d) == cbf

    def test_falls_back_to_uncalibrated(self):
        with tempfile.TemporaryDirectory() as d:
            ns_dir = os.path.join(d, "native_space")
            os.makedirs(ns_dir)
            cbf = os.path.join(ns_dir, "perfusion.nii.gz")
            with open(cbf, "wb") as f:
                f.write(b"\x00")
            assert _find_cbf_output(d) == cbf

    def test_returns_none_empty(self):
        with tempfile.TemporaryDirectory() as d:
            assert _find_cbf_output(d) is None


class TestComputeCBFStats:
    def test_basic_stats(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.array([[[10.0, 20.0], [30.0, 0.0]]]).astype(np.float32)
            path = os.path.join(d, "cbf.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), path)

            stats = _compute_cbf_stats(path)
            assert stats["mean"] == 20.0
            assert stats["median"] == 20.0

    def test_all_zeros(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 2, 2), dtype=np.float32)
            path = os.path.join(d, "cbf.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), path)

            stats = _compute_cbf_stats(path)
            assert stats["mean"] == 0.0
