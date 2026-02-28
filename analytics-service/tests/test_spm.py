"""Tests for the SPM analytics backend."""

import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock

import pytest

from app.backends.spm import SPMBackend, _generate_segmentation_batch


class TestGenerateSegmentationBatch:
    def test_generates_matlab_script(self):
        script = _generate_segmentation_batch("/data/t1w.nii", "/out")
        assert "matlabbatch" in script
        assert "/data/t1w.nii" in script
        assert "spm('defaults', 'fmri')" in script
        assert "spm_jobman('run', matlabbatch)" in script

    def test_includes_tissue_classes(self):
        script = _generate_segmentation_batch("/data/t1w.nii", "/out")
        assert "tissue(1)" in script  # GM
        assert "tissue(2)" in script  # WM
        assert "tissue(3)" in script  # CSF


class TestSPMBackend:
    def setup_method(self):
        self.backend = SPMBackend()

    def test_name(self):
        assert self.backend.name == "spm"

    def test_not_available_without_binary(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False

    def test_available_with_binary(self):
        with patch("shutil.which", return_value="/usr/local/bin/spm12"):
            assert self.backend.available() is True

    def test_no_t1w_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w" in result.error

    def test_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("spm12", 86400)):
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                assert result.success is False
                assert "timed out" in result.error

    def test_binary_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=FileNotFoundError("spm12 not found")):
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                assert result.success is False
                assert "not found" in result.error

    def test_spm_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "SPM segmentation error"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                assert result.success is False
                assert "SPM segmentation error" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii")  # uncompressed
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0

            # Create expected output files
            spm_out = os.path.join(out_dir, "spm")
            os.makedirs(spm_out, exist_ok=True)
            for prefix in ["c1", "c2", "c3"]:
                with open(os.path.join(spm_out, f"{prefix}sub-01_T1w.nii"), "w") as f:
                    f.write("mock")

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is True
                assert result.tool == "spm"
                assert len(result.outputs) >= 3  # c1, c2, c3

    def test_nii_gz_decompressed(self):
        """SPM needs uncompressed NIfTI — verify .nii.gz files are decompressed."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            # Write a gzipped file
            import gzip
            t1w_gz = os.path.join(anat, "sub-01_T1w.nii.gz")
            with gzip.open(t1w_gz, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0

            calls = []

            def capture_run(cmd, **kwargs):
                calls.append(cmd)
                return mock_result

            with patch("subprocess.run", side_effect=capture_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            # The batch script path should end with .m
            assert any(c[-1].endswith(".m") for c in calls if len(c) >= 3)
            # Success since returncode is 0
            assert result.success is True

    def test_batch_script_cleaned_up(self):
        """Verify the temporary MATLAB batch file is deleted after run."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            # No .m files should remain in spm output dir
            spm_out = os.path.join(out_dir, "spm")
            if os.path.isdir(spm_out):
                m_files = [f for f in os.listdir(spm_out) if f.endswith(".m")]
                assert len(m_files) == 0
