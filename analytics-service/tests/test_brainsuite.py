"""Tests for the BrainSuite backend."""

import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import pytest

from app.backends.brainsuite import BrainSuiteBackend
from app.backends.base import AnalyticsResult


class TestBrainSuiteAvailability:
    def setup_method(self):
        self.backend = BrainSuiteBackend()

    def test_name(self):
        assert self.backend.name == "brainsuite"

    def test_available_with_bse(self):
        with patch("shutil.which", return_value="/opt/BrainSuite/bin/bse"):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False


class TestBrainSuiteAnalyze:
    def setup_method(self):
        self.backend = BrainSuiteBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w NIfTI" in result.error

    def test_bse_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("shutil.which", return_value="/opt/bse"):
                with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "not found" in result.error

    def test_bse_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("shutil.which", return_value="/opt/bse"):
                with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 60)):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "timed out" in result.error

    def test_bse_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "segfault"

            with patch("shutil.which", return_value="/opt/bse"):
                with patch("subprocess.run", return_value=mock_result):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False

    def test_bse_only_success(self):
        """BSE succeeds but no further tools available — still succeeds."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0
            mock_result.stderr = ""

            def fake_run(cmd, **kwargs):
                # Create brain output file for BSE
                brain_path = os.path.join(out_dir, "brainsuite", "brain.nii.gz")
                os.makedirs(os.path.dirname(brain_path), exist_ok=True)
                with open(brain_path, "wb") as f:
                    f.write(b"\x00" * 10)
                return mock_result

            def which_only_bse(cmd):
                if cmd == "bse":
                    return "/opt/bse"
                return None

            with patch("shutil.which", side_effect=which_only_bse):
                with patch("subprocess.run", side_effect=fake_run):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "brainsuite"
            assert result.metrics["atlas"] == "brainsuite"

    def test_full_pipeline_success(self):
        """All 4 steps succeed with cerebro producing labels."""
        import nibabel as nib
        import numpy as np

        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            data = np.ones((4, 4, 4), dtype=np.float32) * 100
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            call_count = [0]

            def fake_run(cmd, **kwargs):
                call_count[0] += 1
                bs_dir = os.path.join(out_dir, "brainsuite")
                os.makedirs(bs_dir, exist_ok=True)

                if "bse" in cmd[0]:
                    with open(os.path.join(bs_dir, "brain.nii.gz"), "wb") as f:
                        f.write(b"\x00" * 10)
                elif "bfc" in cmd[0]:
                    with open(os.path.join(bs_dir, "brain.bfc.nii.gz"), "wb") as f:
                        f.write(b"\x00" * 10)
                elif "cerebro" in cmd[0]:
                    # Create a label NIfTI with known labels
                    label_data = np.zeros((4, 4, 4), dtype=np.int32)
                    label_data[0, 0, 0] = 17  # left_hippocampus
                    label_data[1, 0, 0] = 53  # right_hippocampus
                    label_path = os.path.join(bs_dir, "brain.cerebro.label.nii.gz")
                    nib.save(nib.Nifti1Image(label_data, np.eye(4)), label_path)

                mock = MagicMock()
                mock.returncode = 0
                mock.stderr = ""
                return mock

            with patch("shutil.which", return_value="/opt/bin"):
                with patch("subprocess.run", side_effect=fake_run):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.metrics["roi_count"] > 0
            assert "left_hippocampus" in result.metrics["roi_volumes"]
