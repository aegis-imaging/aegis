"""Tests for the SynthSeg backend."""

import csv
import os
import tempfile
from unittest.mock import patch, MagicMock

import pytest

from app.backends.synthseg import (
    SynthSegBackend,
    _parse_qc_csv,
    _parse_volumes_csv,
)
from app.backends.base import AnalyticsResult


class TestSynthSegAvailability:
    def setup_method(self):
        self.backend = SynthSegBackend()

    def test_name(self):
        assert self.backend.name == "synthseg"

    def test_available_with_mri_synthseg(self):
        with patch("shutil.which", return_value="/usr/local/bin/mri_synthseg"):
            assert self.backend.available() is True

    def test_available_with_python_package(self):
        with patch("shutil.which", return_value=None):
            mock_module = MagicMock()
            with patch.dict("sys.modules", {"SynthSeg": mock_module}):
                assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            with patch("builtins.__import__", side_effect=ImportError):
                assert self.backend.available() is False


class TestSynthSegAnalyze:
    def setup_method(self):
        self.backend = SynthSegBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_synthseg_binary_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("shutil.which", return_value=None):
                with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "not found" in result.error

    def test_synthseg_timeout(self):
        import subprocess

        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("shutil.which", return_value="/usr/bin/mri_synthseg"):
                with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 60)):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "timed out" in result.error

    def test_synthseg_nonzero_exit(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "segfault"

            with patch("shutil.which", return_value="/usr/bin/mri_synthseg"):
                with patch("subprocess.run", return_value=mock_result):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "segfault" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            def fake_run(cmd, **kwargs):
                # Create fake output files
                synthseg_dir = os.path.join(out_dir, "synthseg")
                os.makedirs(synthseg_dir, exist_ok=True)

                seg_path = os.path.join(synthseg_dir, "synthseg_seg.nii.gz")
                with open(seg_path, "wb") as f:
                    f.write(b"\x00" * 10)

                vol_path = os.path.join(synthseg_dir, "synthseg_volumes.csv")
                with open(vol_path, "w", newline="") as f:
                    writer = csv.writer(f)
                    writer.writerow(["subject", "Left-Hippocampus", "Right-Hippocampus"])
                    writer.writerow([nifti, "3456.2", "3400.1"])

                qc_path = os.path.join(synthseg_dir, "synthseg_qc.csv")
                with open(qc_path, "w", newline="") as f:
                    writer = csv.writer(f)
                    writer.writerow(["subject", "qc_score"])
                    writer.writerow([nifti, "0.95"])

                mock = MagicMock()
                mock.returncode = 0
                mock.stderr = ""
                return mock

            with patch("shutil.which", return_value="/usr/bin/mri_synthseg"):
                with patch("subprocess.run", side_effect=fake_run):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "synthseg"
            assert result.metrics["atlas"] == "synthseg"
            assert result.metrics["roi_count"] == 2
            assert result.metrics["qc_score"] == 0.95
            assert "Left-Hippocampus" in result.metrics["roi_volumes"]
            assert result.metrics["roi_volumes"]["Left-Hippocampus"] == 3456.2

    def test_uses_freesurfer_path(self):
        """When mri_synthseg is available, uses it over standalone Python."""
        backend = SynthSegBackend()
        with patch("shutil.which", return_value="/usr/bin/mri_synthseg"):
            assert backend._use_freesurfer() is True

        with patch("shutil.which", return_value=None):
            assert backend._use_freesurfer() is False


class TestParseVolumesCSV:
    def test_basic_parsing(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["subject", "Left-Hippocampus", "Right-Hippocampus", "ctx-lh-precentral"])
            writer.writerow(["/path/to/input.nii.gz", "3456.2", "3400.1", "12500.5"])
            f.flush()

            volumes = _parse_volumes_csv(f.name)

        os.unlink(f.name)
        assert len(volumes) == 3
        assert volumes["Left-Hippocampus"] == 3456.2
        assert volumes["Right-Hippocampus"] == 3400.1
        assert volumes["ctx-lh-precentral"] == 12500.5

    def test_missing_file(self):
        assert _parse_volumes_csv("/nonexistent/path.csv") == {}

    def test_skips_subject_column(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["subject", "Hippocampus"])
            writer.writerow(["/path/scan.nii.gz", "3000.0"])
            f.flush()

            volumes = _parse_volumes_csv(f.name)

        os.unlink(f.name)
        assert "subject" not in volumes
        assert "Hippocampus" in volumes

    def test_handles_non_numeric_values(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["subject", "Region1", "Region2"])
            writer.writerow(["/path", "not_a_number", "1500.0"])
            f.flush()

            volumes = _parse_volumes_csv(f.name)

        os.unlink(f.name)
        assert "Region1" not in volumes
        assert volumes["Region2"] == 1500.0


class TestParseQCCSV:
    def test_basic_parsing(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["subject", "qc_score"])
            writer.writerow(["/path/scan.nii.gz", "0.9523"])
            f.flush()

            score = _parse_qc_csv(f.name)

        os.unlink(f.name)
        assert score == 0.9523

    def test_missing_file(self):
        assert _parse_qc_csv("/nonexistent/path.csv") is None

    def test_empty_file(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            f.write("")
            f.flush()

            score = _parse_qc_csv(f.name)

        os.unlink(f.name)
        assert score is None
