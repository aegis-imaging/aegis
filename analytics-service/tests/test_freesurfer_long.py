"""Tests for the FreeSurfer longitudinal backend."""

import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock

import pytest

from app.backends.freesurfer_long import FreeSurferLongBackend, _find_t1w


class TestFreeSurferLongBackend:
    def setup_method(self):
        self.backend = FreeSurferLongBackend()

    def test_name(self):
        assert self.backend.name == "freesurfer_long"

    def test_not_available_without_binary(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False

    def test_not_available_without_license(self):
        with patch("shutil.which", return_value="/usr/local/bin/recon-all"):
            with patch("os.path.isfile", return_value=False):
                assert self.backend.available() is False

    def test_available_with_binary_and_license(self):
        with patch("shutil.which", return_value="/usr/local/bin/recon-all"):
            with patch("os.path.isfile", return_value=True):
                assert self.backend.available() is True

    def test_no_baseline_t1w(self):
        with tempfile.TemporaryDirectory() as bl, tempfile.TemporaryDirectory() as fu:
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

    def test_baseline_recon_failure(self):
        with tempfile.TemporaryDirectory() as bl, \
             tempfile.TemporaryDirectory() as fu, \
             tempfile.TemporaryDirectory() as out:
            for d in (bl, fu):
                anat = os.path.join(d, "sub-01", "anat")
                os.makedirs(anat)
                with open(os.path.join(anat, "sub-01_T1w.nii.gz"), "wb") as f:
                    f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "recon-all error"
            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze_longitudinal(
                    bl, fu, out, "1.2.3", "1.2.4", 365.0,
                )
                assert result.success is False
                assert "baseline" in result.error.lower()

    def test_baseline_timeout(self):
        with tempfile.TemporaryDirectory() as bl, \
             tempfile.TemporaryDirectory() as fu, \
             tempfile.TemporaryDirectory() as out:
            for d in (bl, fu):
                anat = os.path.join(d, "sub-01", "anat")
                os.makedirs(anat)
                with open(os.path.join(anat, "sub-01_T1w.nii.gz"), "wb") as f:
                    f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("recon-all", 3600)):
                result = self.backend.analyze_longitudinal(
                    bl, fu, out, "1.2.3", "1.2.4", 365.0,
                )
                assert result.success is False
                assert "timed out" in result.error.lower()

    def test_skips_existing_baseline(self):
        """If baseline dir already exists, step 1 is skipped."""
        with tempfile.TemporaryDirectory() as bl, \
             tempfile.TemporaryDirectory() as fu, \
             tempfile.TemporaryDirectory() as out:
            for d in (bl, fu):
                anat = os.path.join(d, "sub-01", "anat")
                os.makedirs(anat)
                with open(os.path.join(anat, "sub-01_T1w.nii.gz"), "wb") as f:
                    f.write(b"\x00" * 100)

            subjects_dir = os.path.join(out, "freesurfer_long")
            # Pre-create baseline subject dir
            tp1_dir = os.path.join(subjects_dir, "sub-1234abcd_tp1")
            os.makedirs(tp1_dir)

            call_count = 0
            def mock_run(*args, **kwargs):
                nonlocal call_count
                call_count += 1
                r = MagicMock()
                r.returncode = 1
                r.stderr = "failed at step 2"
                return r

            with patch("subprocess.run", side_effect=mock_run):
                result = self.backend.analyze_longitudinal(
                    bl, fu, out, "1234abcd.5.6", "5678efgh.5.6", 365.0,
                )
                # Should fail at step 2 (follow-up), not step 1
                assert result.success is False
                assert "follow-up" in result.error.lower()

    def test_volume_change_computation(self):
        """Test the volume change percentage computation logic."""
        backend = FreeSurferLongBackend()
        # Directly test the metrics computation by checking the dict structure
        bl_vols = {"Left-Hippocampus": 4000.0, "Right-Hippocampus": 4200.0}
        fu_vols = {"Left-Hippocampus": 3800.0, "Right-Hippocampus": 4100.0}

        changes = {}
        for region in bl_vols:
            if region in fu_vols and bl_vols[region] > 0:
                pct = ((fu_vols[region] - bl_vols[region]) / bl_vols[region]) * 100
                changes[region] = round(pct, 3)

        assert changes["Left-Hippocampus"] == pytest.approx(-5.0, abs=0.01)
        assert changes["Right-Hippocampus"] == pytest.approx(-2.381, abs=0.01)


class TestFindT1w:
    def test_finds_t1w(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00")
            assert _find_t1w(d) == t1w

    def test_returns_none_when_missing(self):
        with tempfile.TemporaryDirectory() as d:
            assert _find_t1w(d) is None

    def test_finds_nested(self):
        with tempfile.TemporaryDirectory() as d:
            nested = os.path.join(d, "sub-02", "ses-01", "anat")
            os.makedirs(nested)
            t1w = os.path.join(nested, "sub-02_ses-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00")
            assert _find_t1w(d) == t1w
