"""Tests for the volBrain backend."""

import json
import os
import tempfile
from unittest.mock import MagicMock, patch

import pytest

from app.backends.volbrain import VolBrainBackend
from app.backends.base import AnalyticsResult


class TestVolBrainAvailability:
    def setup_method(self):
        self.backend = VolBrainBackend()

    def test_name(self):
        assert self.backend.name == "volbrain"

    def test_available_with_url(self):
        with patch.object(self.backend, "available") as mock_avail:
            mock_avail.return_value = True
            assert self.backend.available() is True

    def test_not_available_no_url(self):
        with patch("app.backends.volbrain.config") as mock_config:
            mock_config.VOLBRAIN_API_URL = ""
            backend = VolBrainBackend()
            assert backend.available() is False

    def test_not_available_unreachable(self):
        import urllib.error

        with patch("app.backends.volbrain.config") as mock_config:
            mock_config.VOLBRAIN_API_URL = "https://volbrain.example.com"
            with patch(
                "urllib.request.urlopen",
                side_effect=urllib.error.URLError("unreachable"),
            ):
                backend = VolBrainBackend()
                assert backend.available() is False


class TestVolBrainAnalyze:
    def setup_method(self):
        self.backend = VolBrainBackend()

    def test_no_t1w_nifti(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w NIfTI" in result.error

    def test_submit_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.volbrain.config") as mock_config:
                mock_config.VOLBRAIN_API_URL = "https://volbrain.example.com"
                mock_config.VOLBRAIN_API_KEY = ""
                mock_config.VOLBRAIN_POLL_INTERVAL = 1
                mock_config.VOLBRAIN_TIMEOUT = 5

                with patch(
                    "app.backends.volbrain._submit_job",
                    side_effect=RuntimeError("connection refused"),
                ):
                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                    assert result.success is False
                    assert "submission failed" in result.error

    def test_poll_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.volbrain.config") as mock_config:
                mock_config.VOLBRAIN_API_URL = "https://volbrain.example.com"
                mock_config.VOLBRAIN_API_KEY = ""
                mock_config.VOLBRAIN_POLL_INTERVAL = 1
                mock_config.VOLBRAIN_TIMEOUT = 5

                with patch("app.backends.volbrain._submit_job", return_value="job-123"):
                    with patch(
                        "app.backends.volbrain._poll_result",
                        side_effect=TimeoutError("timed out"),
                    ):
                        result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                        assert result.success is False
                        assert "timed out" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            api_result = {
                "status": "complete",
                "volumes": {
                    "left_hippocampus": 3456.2,
                    "right_hippocampus": 3400.1,
                    "left_amygdala": 1200.0,
                },
                "icv": 1450000.0,
                "tissue_volumes": {
                    "gm": 700000.0,
                    "wm": 450000.0,
                    "csf": 300000.0,
                },
            }

            with patch("app.backends.volbrain.config") as mock_config:
                mock_config.VOLBRAIN_API_URL = "https://volbrain.example.com"
                mock_config.VOLBRAIN_API_KEY = "key123"
                mock_config.VOLBRAIN_POLL_INTERVAL = 1
                mock_config.VOLBRAIN_TIMEOUT = 60

                with patch("app.backends.volbrain._submit_job", return_value="job-123"):
                    with patch("app.backends.volbrain._poll_result", return_value=api_result):
                        result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "volbrain"
            assert result.metrics["atlas"] == "volbrain"
            assert result.metrics["roi_count"] == 3
            assert result.metrics["icv"] == 1450000.0
            assert "left_hippocampus" in result.metrics["roi_volumes"]
            assert result.metrics["roi_volumes"]["left_hippocampus"] == 3456.2

    def test_api_job_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.volbrain.config") as mock_config:
                mock_config.VOLBRAIN_API_URL = "https://volbrain.example.com"
                mock_config.VOLBRAIN_API_KEY = ""
                mock_config.VOLBRAIN_POLL_INTERVAL = 1
                mock_config.VOLBRAIN_TIMEOUT = 5

                with patch("app.backends.volbrain._submit_job", return_value="job-123"):
                    with patch(
                        "app.backends.volbrain._poll_result",
                        side_effect=RuntimeError("volBrain job failed: segmentation error"),
                    ):
                        result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                        assert result.success is False
                        assert "polling failed" in result.error
