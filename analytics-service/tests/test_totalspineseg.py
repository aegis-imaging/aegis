"""Tests for the TotalSpineSeg backend."""

import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import nibabel as nib
import numpy as np
import pytest

from app.backends.totalspineseg import TotalSpineSegBackend, _find_output_segmentation
from app.backends.seg_utils import TOTALSPINESEG_LABELS


class TestTotalSpineSegAvailability:
    def setup_method(self):
        self.backend = TotalSpineSegBackend()

    def test_name(self):
        assert self.backend.name == "totalspineseg"

    def test_available_with_cli(self):
        with patch("shutil.which", return_value="/usr/local/bin/totalspineseg"):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False


class TestTotalSpineSegAnalyze:
    def setup_method(self):
        self.backend = TotalSpineSegBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_cli_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "not found" in result.error

    def test_cli_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 60)):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "timed out" in result.error

    def test_nonzero_exit(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "error occurred"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False

    def test_no_output_produced(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0
            mock_result.stderr = ""

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "did not produce" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            # Create input NIfTI
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            data = np.ones((10, 10, 10), dtype=np.float32) * 100
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            mock_result = MagicMock()
            mock_result.returncode = 0
            mock_result.stderr = ""

            def fake_run(cmd, **kwargs):
                # Create output segmentation with spine labels
                seg_dir = os.path.join(out_dir, "totalspineseg", "step2_output")
                os.makedirs(seg_dir, exist_ok=True)

                seg_data = np.zeros((10, 10, 10), dtype=np.int32)
                seg_data[0:2, :, :] = 1   # C1
                seg_data[2:4, :, :] = 2   # C2
                seg_data[4:5, :, :] = 40  # IVD_C2-C3
                seg_data[5:6, :, :] = 200  # spinal_cord
                seg_data[6:7, :, :] = 201  # spinal_canal

                seg_path = os.path.join(seg_dir, "segmentation.nii.gz")
                nib.save(nib.Nifti1Image(seg_data, np.eye(4)), seg_path)
                return mock_result

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "totalspineseg"
            assert result.metrics["atlas"] == "totalspineseg"
            assert "C1" in result.metrics["roi_volumes"]
            assert "C2" in result.metrics["roi_volumes"]
            assert "IVD_C2-C3" in result.metrics["roi_volumes"]
            assert result.metrics["has_spinal_cord"] is True
            assert result.metrics["has_spinal_canal"] is True
            assert "C1" in result.metrics["vertebrae_detected"]
            assert "IVD_C2-C3" in result.metrics["ivd_detected"]

    def test_prefers_t2w(self):
        """Should use T2w over T1w when both are available."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            t2w = os.path.join(anat, "sub-01_T2w.nii.gz")
            for f in [t1w, t2w]:
                data = np.ones((4, 4, 4), dtype=np.float32)
                nib.save(nib.Nifti1Image(data, np.eye(4)), f)

            mock_result = MagicMock()
            mock_result.returncode = 0
            mock_result.stderr = ""

            called_with = []

            def fake_run(cmd, **kwargs):
                called_with.append(cmd)
                # Create output
                seg_dir = os.path.join(out_dir, "totalspineseg", "step2_output")
                os.makedirs(seg_dir, exist_ok=True)
                seg_data = np.zeros((4, 4, 4), dtype=np.int32)
                seg_data[0, 0, 0] = 1  # C1
                nib.save(
                    nib.Nifti1Image(seg_data, np.eye(4)),
                    os.path.join(seg_dir, "seg.nii.gz"),
                )
                return mock_result

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            # The input file passed to CLI should be T2w
            assert "T2w" in called_with[0][1]


class TestTotalSpineSegLabelMap:
    def test_vertebrae_labels_complete(self):
        """All C1-C7, T1-T12, L1-L5, sacrum should be present."""
        names = set(TOTALSPINESEG_LABELS.values())
        for v in ["C1", "C2", "C3", "C4", "C5", "C6", "C7"]:
            assert v in names
        for v in [f"T{i}" for i in range(1, 13)]:
            assert v in names
        for v in ["L1", "L2", "L3", "L4", "L5"]:
            assert v in names
        assert "sacrum" in names

    def test_ivd_labels_present(self):
        names = set(TOTALSPINESEG_LABELS.values())
        assert "IVD_C2-C3" in names
        assert "IVD_L5-S1" in names

    def test_spinal_cord_and_canal(self):
        assert TOTALSPINESEG_LABELS[200] == "spinal_cord"
        assert TOTALSPINESEG_LABELS[201] == "spinal_canal"


class TestFindOutputSegmentation:
    def test_finds_step2_output(self):
        with tempfile.TemporaryDirectory() as d:
            step2 = os.path.join(d, "step2_output")
            os.makedirs(step2)
            seg = os.path.join(step2, "seg.nii.gz")
            with open(seg, "wb") as f:
                f.write(b"\x00" * 10)

            assert _find_output_segmentation(d) == seg

    def test_falls_back_to_step1(self):
        with tempfile.TemporaryDirectory() as d:
            step1 = os.path.join(d, "step1_output")
            os.makedirs(step1)
            seg = os.path.join(step1, "seg.nii.gz")
            with open(seg, "wb") as f:
                f.write(b"\x00" * 10)

            assert _find_output_segmentation(d) == seg

    def test_returns_none_empty(self):
        with tempfile.TemporaryDirectory() as d:
            assert _find_output_segmentation(d) is None
