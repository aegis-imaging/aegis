"""Tests for the TBM-SyN longitudinal backend."""

import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock

import numpy as np
import pytest

from app.backends.tbm_syn import TBMSyNBackend, AD_SIGNATURE_ROIS


class TestTBMSyNBackend:
    def setup_method(self):
        self.backend = TBMSyNBackend()

    def test_name(self):
        assert self.backend.name == "tbm_syn"

    def test_not_available_without_binaries(self):
        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", return_value=None):
                assert self.backend.available() is False

    def test_available_with_both_binaries(self):
        def which_side(name):
            if name in ("antsRegistrationSyN.sh", "CreateJacobianDeterminantImage"):
                return f"/usr/bin/{name}"
            return None

        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", side_effect=which_side):
                assert self.backend.available() is True

    def test_missing_one_binary(self):
        def which_side(name):
            if name == "antsRegistrationSyN.sh":
                return "/usr/bin/antsRegistrationSyN.sh"
            return None

        with patch("os.path.isfile", return_value=False):
            with patch("shutil.which", side_effect=which_side):
                assert self.backend.available() is False

    def test_no_baseline_t1w(self):
        with tempfile.TemporaryDirectory() as bl, tempfile.TemporaryDirectory() as fu:
            # Create follow-up T1w but not baseline
            anat = os.path.join(fu, "sub-01", "anat")
            os.makedirs(anat)
            with open(os.path.join(anat, "sub-01_T1w.nii.gz"), "wb") as f:
                f.write(b"\x00" * 100)

            result = self.backend.analyze_longitudinal(
                bl, fu, "/tmp/out", "1.2.3", "1.2.4", 365.0,
            )
            assert result.success is False
            assert "baseline" in result.error.lower()

    def test_no_followup_t1w(self):
        with tempfile.TemporaryDirectory() as bl, tempfile.TemporaryDirectory() as fu:
            anat = os.path.join(bl, "sub-01", "anat")
            os.makedirs(anat)
            with open(os.path.join(anat, "sub-01_T1w.nii.gz"), "wb") as f:
                f.write(b"\x00" * 100)

            result = self.backend.analyze_longitudinal(
                bl, fu, "/tmp/out", "1.2.3", "1.2.4", 365.0,
            )
            assert result.success is False
            assert "follow-up" in result.error.lower()

    def test_registration_failure(self):
        with tempfile.TemporaryDirectory() as bl, \
             tempfile.TemporaryDirectory() as fu, \
             tempfile.TemporaryDirectory() as out:
            for d in (bl, fu):
                anat = os.path.join(d, "sub-01", "anat")
                os.makedirs(anat)
                with open(os.path.join(anat, "sub-01_T1w.nii.gz"), "wb") as f:
                    f.write(b"\x00" * 100)

            with patch("app.backends.tbm_syn.run_n4_bias_correction", return_value=(True, "")):
                with patch("app.backends.tbm_syn.run_brain_extraction", return_value=(True, "/brain.nii.gz", "")):
                    mock_result = MagicMock()
                    mock_result.returncode = 1
                    mock_result.stderr = "syn registration error"
                    with patch("subprocess.run", return_value=mock_result):
                        result = self.backend.analyze_longitudinal(
                            bl, fu, out, "1.2.3", "1.2.4", 365.0,
                        )
                        assert result.success is False
                        assert "registration" in result.error.lower()


class TestAnnualizeJacobian:
    def setup_method(self):
        self.backend = TBMSyNBackend()

    def test_annualize_one_year(self):
        import nibabel as nib

        data = np.full((3, 3, 3), 0.1, dtype=np.float64)
        img = nib.Nifti1Image(data, np.eye(4))

        with tempfile.TemporaryDirectory() as d:
            in_path = os.path.join(d, "jac.nii.gz")
            out_path = os.path.join(d, "jac_ann.nii.gz")
            nib.save(img, in_path)

            self.backend._annualize_jacobian(in_path, out_path, 365.25)

            result = nib.load(out_path)
            assert np.allclose(result.get_fdata(), 0.1, atol=1e-6)

    def test_annualize_half_year(self):
        import nibabel as nib

        data = np.full((3, 3, 3), 0.05, dtype=np.float64)
        img = nib.Nifti1Image(data, np.eye(4))

        with tempfile.TemporaryDirectory() as d:
            in_path = os.path.join(d, "jac.nii.gz")
            out_path = os.path.join(d, "jac_ann.nii.gz")
            nib.save(img, in_path)

            # 6 months = ~182.625 days
            self.backend._annualize_jacobian(in_path, out_path, 182.625)

            result = nib.load(out_path)
            # 0.05 / 0.5 = 0.1
            assert np.allclose(result.get_fdata(), 0.1, atol=1e-6)


class TestADComposite:
    def setup_method(self):
        self.backend = TBMSyNBackend()

    def test_composite_calculation(self):
        roi_stats = [
            {"roi_id": 41, "roi_name": "Hippocampus_L", "mean": -0.02, "voxel_count": 1000,
             "volume_mm3": 8000, "median": -0.019, "std": 0.01},
            {"roi_id": 42, "roi_name": "Hippocampus_R", "mean": -0.03, "voxel_count": 1000,
             "volume_mm3": 8000, "median": -0.029, "std": 0.01},
            {"roi_id": 1, "roi_name": "Precentral_L", "mean": -0.005, "voxel_count": 2000,
             "volume_mm3": 16000, "median": -0.004, "std": 0.008},
        ]

        composite = self.backend._compute_ad_composite(roi_stats)

        # Only ROI 41 and 42 are AD signature
        assert composite["contributing_roi_count"] == 2
        assert composite["total_voxels"] == 2000

        # Weighted mean: (1000 * -0.02 + 1000 * -0.03) / 2000 = -0.025
        assert composite["composite_mean"] == pytest.approx(-0.025, abs=1e-6)

    def test_empty_stats(self):
        composite = self.backend._compute_ad_composite([])
        assert composite["contributing_roi_count"] == 0
        assert composite["composite_mean"] == 0.0

    def test_no_ad_rois(self):
        roi_stats = [
            {"roi_id": 1, "roi_name": "Precentral_L", "mean": 100.0, "voxel_count": 500,
             "volume_mm3": 4000, "median": 99, "std": 10},
        ]
        composite = self.backend._compute_ad_composite(roi_stats)
        assert composite["contributing_roi_count"] == 0
        assert composite["composite_mean"] == 0.0


class TestADSignatureROIs:
    def test_ad_rois_are_bilateral(self):
        """Most AD ROIs should have both L and R."""
        left_ids = [k for k in AD_SIGNATURE_ROIS if k % 2 == 1 and k < 95]
        right_ids = [k for k in AD_SIGNATURE_ROIS if k % 2 == 0 and k < 95]
        # Should have reasonable bilateral coverage
        assert len(left_ids) >= 10
        assert len(right_ids) >= 10

    def test_hippocampus_included(self):
        assert 41 in AD_SIGNATURE_ROIS
        assert 42 in AD_SIGNATURE_ROIS

    def test_temporal_regions_included(self):
        assert 93 in AD_SIGNATURE_ROIS  # Temporal_Inf_L
        assert 94 in AD_SIGNATURE_ROIS  # Temporal_Inf_R
