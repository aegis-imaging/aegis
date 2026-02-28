"""Tests for the ITK-SNAP / Convert3D backend."""

import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import pytest

from app.backends.itksnap import ITKSnapBackend
from app.backends.base import AnalyticsResult


class TestITKSnapAvailability:
    def setup_method(self):
        self.backend = ITKSnapBackend()

    def test_name(self):
        assert self.backend.name == "itksnap"

    def test_available_with_c3d(self):
        with patch("shutil.which", return_value="/usr/local/bin/c3d"):
            assert self.backend.available() is True

    def test_available_with_itksnap_wt(self):
        def which_side_effect(cmd):
            if cmd == "itksnap-wt":
                return "/usr/local/bin/itksnap-wt"
            return None

        with patch("shutil.which", side_effect=which_side_effect):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False


class TestITKSnapAnalyze:
    def setup_method(self):
        self.backend = ITKSnapBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w NIfTI" in result.error

    def test_binary_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("shutil.which", return_value="/usr/bin/c3d"):
                with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "not found" in result.error

    def test_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("shutil.which", return_value="/usr/bin/c3d"):
                with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 60)):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "timed out" in result.error

    def test_nonzero_exit(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "error"

            with patch("shutil.which", return_value="/usr/bin/c3d"):
                with patch("subprocess.run", return_value=mock_result):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False

    def test_no_output_produced(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0
            mock_result.stderr = ""

            with patch("shutil.which", return_value="/usr/bin/c3d"):
                with patch("subprocess.run", return_value=mock_result):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "not produced" in result.error

    def test_successful_analysis(self):
        import nibabel as nib
        import numpy as np

        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            # Create a real NIfTI for compute_label_volumes
            data = np.zeros((4, 4, 4), dtype=np.float32)
            data[0:2, 0:2, 0:2] = 100
            affine = np.eye(4)
            nib.save(nib.Nifti1Image(data, affine), nifti)

            def fake_run(cmd, **kwargs):
                seg_dir = os.path.join(out_dir, "itksnap")
                os.makedirs(seg_dir, exist_ok=True)
                seg_path = os.path.join(seg_dir, "itksnap_seg.nii.gz")
                # Create seg with labels 1, 2
                seg_data = np.zeros((4, 4, 4), dtype=np.int32)
                seg_data[0, 0, 0] = 1
                seg_data[1, 0, 0] = 2
                nib.save(nib.Nifti1Image(seg_data, affine), seg_path)

                mock = MagicMock()
                mock.returncode = 0
                mock.stderr = ""
                return mock

            with patch("shutil.which", return_value="/usr/bin/c3d"):
                with patch("subprocess.run", side_effect=fake_run):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "itksnap"
            assert result.metrics["atlas"] == "itksnap"
            assert result.metrics["roi_count"] > 0

    def test_uses_c3d_preference(self):
        backend = ITKSnapBackend()
        with patch("shutil.which", return_value="/usr/bin/c3d"):
            assert backend._use_c3d() is True
        with patch("shutil.which", return_value=None):
            assert backend._use_c3d() is False
