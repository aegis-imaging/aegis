"""Tests for the ANTs analytics backend."""

import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock

import pytest

from app.backends.ants import ANTsBackend


class TestANTsBackend:
    def setup_method(self):
        self.backend = ANTsBackend()

    def test_name(self):
        assert self.backend.name == "ants"

    def test_not_available_without_binary(self):
        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", return_value=None):
                assert self.backend.available() is False

    def test_available_with_binary_in_path(self):
        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", return_value="/usr/bin/antsCorticalThickness.sh"):
                assert self.backend.available() is True

    def test_available_with_binary_in_antspath(self):
        with patch("os.path.isfile", return_value=True):
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
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("os.path.isfile", return_value=True):
                with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("ants", 86400)):
                    result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                    assert result.success is False
                    assert "timed out" in result.error

    def test_n4_fallback_when_no_template(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0

            # Template does not exist -> N4 fallback
            real_isfile = os.path.isfile

            def isfile_side_effect(p):
                if "T_template0" in p:
                    return False
                return real_isfile(p)

            with patch("os.path.isfile", side_effect=isfile_side_effect):
                with patch("subprocess.run", return_value=mock_result):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.tool == "ants"
                    # N4 returns success based on returncode
                    assert result.success is True

    def test_cortical_thickness_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "ANTs error occurred"

            with patch("os.path.isfile", return_value=True):
                with patch("subprocess.run", return_value=mock_result):
                    result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                    assert result.success is False
                    assert "ANTs error" in result.error


class TestN4Only:
    def setup_method(self):
        self.backend = ANTsBackend()

    def test_n4_success(self):
        with tempfile.TemporaryDirectory() as out_dir:
            mock_result = MagicMock()
            mock_result.returncode = 0

            n4_out = os.path.join(out_dir, "t1w_n4.nii.gz")

            def run_side_effect(cmd, **kwargs):
                # Create output file when N4 runs
                with open(n4_out, "wb") as f:
                    f.write(b"\x00" * 50)
                return mock_result

            import time
            with patch("subprocess.run", side_effect=run_side_effect):
                result = self.backend._run_n4_only("/tmp/t1w.nii.gz", out_dir, {}, time.time())
                assert result.success is True
                assert n4_out in result.outputs

    def test_n4_file_not_found(self):
        import time
        with patch("subprocess.run", side_effect=FileNotFoundError("N4 not found")):
            result = self.backend._run_n4_only("/tmp/t1w.nii.gz", "/tmp/out", {}, time.time())
            assert result.success is False
            assert "N4 not found" in result.error
