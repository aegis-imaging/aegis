"""Tests for the FSL analytics backend."""

import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock, call

import pytest

from app.backends.fsl import FSLBackend


class TestFSLBackend:
    def setup_method(self):
        self.backend = FSLBackend()

    def test_name(self):
        assert self.backend.name == "fsl"

    def test_not_available_without_binary(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False

    def test_available_with_binary(self):
        with patch("shutil.which", return_value="/usr/local/fsl/bin/bet"):
            assert self.backend.available() is True

    def test_no_t1w_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w" in result.error

    def test_bet_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "BET error"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, "/tmp/fsl-out", "1.2.3")
                assert result.success is False
                assert "BET failed" in result.error

    def test_full_pipeline_success(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            fsl_out = os.path.join(out_dir, "fsl")
            os.makedirs(fsl_out, exist_ok=True)

            mock_result = MagicMock()
            mock_result.returncode = 0

            # BET creates output files
            brain_nii = os.path.join(fsl_out, "brain.nii.gz")

            def run_side_effect(cmd, **kwargs):
                if cmd[0] == "bet":
                    with open(brain_nii, "wb") as f:
                        f.write(b"\x00" * 50)
                return mock_result

            with patch("subprocess.run", side_effect=run_side_effect):
                with patch("os.path.isfile") as mock_isfile:
                    # brain.nii.gz exists, MNI template does not
                    mock_isfile.side_effect = lambda p: p == brain_nii
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "fsl"

    def test_run_cmd_passes_env(self):
        env = {"PATH": "/usr/bin", "FSLDIR": "/opt/fsl"}
        mock_result = MagicMock()
        mock_result.returncode = 0

        with patch("subprocess.run", return_value=mock_result) as mock_run:
            self.backend._run_cmd(["bet", "in", "out"], env, "BET")
            mock_run.assert_called_once()
            _, kwargs = mock_run.call_args
            assert kwargs["env"] == env
