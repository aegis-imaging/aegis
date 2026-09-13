"""Tests for the Atlas ROI labeling backend."""

import os
import tempfile
from unittest.mock import patch, MagicMock

import pytest

from app.backends.atlas_roi import AtlasROIBackend
from app.backends.base import AnalyticsResult


class TestAtlasROIBackend:
    def setup_method(self):
        self.backend = AtlasROIBackend()

    def test_name(self):
        assert self.backend.name == "atlas_roi"

    def test_not_available_without_ants(self):
        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", return_value=None):
                assert self.backend.available() is False

    def test_available_with_ants_in_path(self):
        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", return_value="/usr/bin/antsRegistrationSyN.sh"):
                assert self.backend.available() is True

    def test_available_with_ants_in_antspath(self):
        with patch("os.path.isfile", return_value=True):
            assert self.backend.available() is True

    def test_no_t1w_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No T1w" in result.error

    def test_atlas_labels_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.atlas_roi.load_atlas_labels", side_effect=FileNotFoundError("not found")):
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3", atlas="nonexistent")
                assert result.success is False
                assert "not found" in result.error

    def test_atlas_nifti_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.atlas_roi.load_atlas_labels", return_value={1: "R"}):
                with patch("app.backends.atlas_roi.find_atlas_nifti", side_effect=FileNotFoundError("no nifti")):
                    result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                    assert result.success is False
                    assert "no nifti" in result.error

    def test_template_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.atlas_roi.load_atlas_labels", return_value={1: "R"}):
                with patch("app.backends.atlas_roi.find_atlas_nifti", return_value="/atlas.nii.gz"):
                    with patch("app.backends.atlas_roi.run_n4_bias_correction", return_value=(True, "")):
                        with patch("app.backends.atlas_roi.run_brain_extraction", return_value=(True, "/brain.nii.gz", "")):
                            with patch("app.backends.atlas_roi.config") as mock_cfg:
                                mock_cfg.ANTSPATH = "/opt/ants"
                                mock_cfg.DEFAULT_ATLAS = "aal3"
                                mock_cfg.ANTS_TEMPLATE_DIR = "/nonexistent"
                                mock_cfg.ANALYTICS_TIMEOUT = 86400
                                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                                assert result.success is False
                                assert "template not found" in result.error.lower()

    def test_warp_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            # Create fake template
            template_dir = os.path.join(out_dir, "templates")
            os.makedirs(template_dir)
            template = os.path.join(template_dir, "T_template0.nii.gz")
            with open(template, "wb") as f:
                f.write(b"\x00" * 10)

            with patch("app.backends.atlas_roi.load_atlas_labels", return_value={1: "R"}):
                with patch("app.backends.atlas_roi.find_atlas_nifti", return_value="/atlas.nii.gz"):
                    with patch("app.backends.atlas_roi.run_n4_bias_correction", return_value=(True, "")):
                        with patch("app.backends.atlas_roi.run_brain_extraction", return_value=(True, "/brain.nii.gz", "")):
                            with patch("app.backends.atlas_roi.warp_atlas_to_subject", return_value=(False, "reg error")):
                                with patch("app.backends.atlas_roi.config") as mock_cfg:
                                    mock_cfg.ANTSPATH = "/opt/ants"
                                    mock_cfg.DEFAULT_ATLAS = "aal3"
                                    mock_cfg.ANTS_TEMPLATE_DIR = template_dir
                                    mock_cfg.ANALYTICS_TIMEOUT = 86400
                                    result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                                    assert result.success is False
                                    assert "reg error" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            template_dir = os.path.join(out_dir, "templates")
            os.makedirs(template_dir)
            template = os.path.join(template_dir, "T_template0.nii.gz")
            with open(template, "wb") as f:
                f.write(b"\x00" * 10)

            mock_stats = [
                {"roi_id": 1, "roi_name": "Precentral_L", "voxel_count": 500,
                 "volume_mm3": 4000.0, "mean": 120.5, "median": 118.0, "std": 15.3},
                {"roi_id": 2, "roi_name": "Precentral_R", "voxel_count": 480,
                 "volume_mm3": 3840.0, "mean": 119.2, "median": 117.0, "std": 14.8},
            ]

            with patch("app.backends.atlas_roi.load_atlas_labels", return_value={1: "Precentral_L", 2: "Precentral_R"}):
                with patch("app.backends.atlas_roi.find_atlas_nifti", return_value="/atlas.nii.gz"):
                    with patch("app.backends.atlas_roi.run_n4_bias_correction", return_value=(True, "")):
                        with patch("app.backends.atlas_roi.run_brain_extraction", return_value=(True, "/brain.nii.gz", "")):
                            with patch("app.backends.atlas_roi.warp_atlas_to_subject") as mock_warp:
                                mock_warp.return_value = (True, "")
                                with patch("app.backends.atlas_roi.extract_roi_stats", return_value=mock_stats):
                                    with patch("app.backends.atlas_roi.config") as mock_cfg:
                                        mock_cfg.ANTSPATH = "/opt/ants"
                                        mock_cfg.DEFAULT_ATLAS = "aal3"
                                        mock_cfg.ANTS_TEMPLATE_DIR = template_dir
                                        mock_cfg.ANALYTICS_TIMEOUT = 86400
                                        result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "atlas_roi"
            assert result.metrics["roi_count"] == 2
            assert result.metrics["atlas"] == "aal3"
            assert "Precentral_L" in result.metrics["roi_volumes"]

            # Check output files were created
            roi_out = os.path.join(out_dir, "atlas_roi")
            assert os.path.isfile(os.path.join(roi_out, "roi_stats.csv"))
            assert os.path.isfile(os.path.join(roi_out, "summary.json"))

    def test_atlas_kwarg_override(self):
        """The atlas kwarg should override DEFAULT_ATLAS."""
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.atlas_roi.load_atlas_labels") as mock_load:
                mock_load.side_effect = FileNotFoundError("mcalt not found")
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3", atlas="mcalt")
                # Verify it tried to load "mcalt" labels, not default
                mock_load.assert_called_once_with("mcalt")

    def test_n4_failure_continues(self):
        """N4 failure should not abort — uses uncorrected T1w."""
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("app.backends.atlas_roi.load_atlas_labels", return_value={1: "R"}):
                with patch("app.backends.atlas_roi.find_atlas_nifti", return_value="/a.nii"):
                    with patch("app.backends.atlas_roi.run_n4_bias_correction", return_value=(False, "N4 failed")):
                        with patch("app.backends.atlas_roi.run_brain_extraction") as mock_be:
                            # Brain extraction should be called with original t1w
                            mock_be.return_value = (False, "", "brain extraction also failed")
                            with patch("app.backends.atlas_roi.config") as mock_cfg:
                                mock_cfg.ANTSPATH = "/opt/ants"
                                mock_cfg.DEFAULT_ATLAS = "aal3"
                                mock_cfg.ANTS_TEMPLATE_DIR = "/nonexistent"
                                mock_cfg.ANALYTICS_TIMEOUT = 86400
                                result = self.backend.analyze(bids_dir, "/tmp/out2", "1.2.3")
                                # Should have called brain extraction with original t1w
                                call_args = mock_be.call_args
                                assert call_args[0][0] == t1w
