"""Tests for the PETSurfer backend."""

import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import pytest

from app.backends.petsurfer import PETSurferBackend, _parse_gtm_stats
from app.backends.base import AnalyticsResult


class TestPETSurferAvailability:
    def setup_method(self):
        self.backend = PETSurferBackend()

    def test_name(self):
        assert self.backend.name == "petsurfer"

    def test_available_with_binary_and_license(self):
        with patch("shutil.which", return_value="/usr/local/bin/mri_gtmpvc"):
            with patch("os.path.isfile", return_value=True):
                assert self.backend.available() is True

    def test_not_available_no_binary(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False

    def test_not_available_no_license(self):
        with patch("shutil.which", return_value="/usr/local/bin/mri_gtmpvc"):
            with patch("os.path.isfile", return_value=False):
                assert self.backend.available() is False


class TestPETSurferAnalyze:
    def setup_method(self):
        self.backend = PETSurferBackend()

    def test_no_pet_nifti(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No PET NIfTI" in result.error

    def test_binary_not_found(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            pet_dir = os.path.join(bids_dir, "sub-01", "pet")
            os.makedirs(pet_dir)
            pet = os.path.join(pet_dir, "sub-01_pet.nii.gz")
            with open(pet, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "not found" in result.error

    def test_timeout(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            pet_dir = os.path.join(bids_dir, "sub-01", "pet")
            os.makedirs(pet_dir)
            pet = os.path.join(pet_dir, "sub-01_pet.nii.gz")
            with open(pet, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 60)):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "timed out" in result.error

    def test_nonzero_exit(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            pet_dir = os.path.join(bids_dir, "sub-01", "pet")
            os.makedirs(pet_dir)
            pet = os.path.join(pet_dir, "sub-01_pet.nii.gz")
            with open(pet, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "error"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            pet_dir = os.path.join(bids_dir, "sub-01", "pet")
            os.makedirs(pet_dir)
            pet = os.path.join(pet_dir, "sub-01_pet.nii.gz")
            with open(pet, "wb") as f:
                f.write(b"\x00" * 100)

            def fake_run(cmd, **kwargs):
                ps_dir = os.path.join(out_dir, "petsurfer")
                os.makedirs(ps_dir, exist_ok=True)
                stats = os.path.join(ps_dir, "gtm.stats.dat")
                with open(stats, "w") as f:
                    f.write("# GTM stats\n")
                    f.write("1 Left-Hippocampus 1.234 0.1\n")
                    f.write("2 Right-Hippocampus 1.198 0.12\n")

                mock = MagicMock()
                mock.returncode = 0
                mock.stderr = ""
                return mock

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "petsurfer"
            assert result.metrics["atlas"] == "gtm"
            assert result.metrics["roi_count"] == 2
            assert "Left-Hippocampus" in result.metrics["roi_suvr"]
            assert result.metrics["roi_suvr"]["Left-Hippocampus"] == 1.234


class TestParseGTMStats:
    def test_basic_parsing(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".dat", delete=False) as f:
            f.write("# comment line\n")
            f.write("1 Left-Caudate 1.45 0.05\n")
            f.write("2 Right-Caudate 1.50 0.06\n")
            f.flush()

            stats = _parse_gtm_stats(f.name)

        os.unlink(f.name)
        assert len(stats) == 2
        assert stats["Left-Caudate"] == 1.45
        assert stats["Right-Caudate"] == 1.5

    def test_missing_file(self):
        assert _parse_gtm_stats("/nonexistent.dat") == {}

    def test_skips_comments(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".dat", delete=False) as f:
            f.write("# header\n")
            f.write("# another comment\n")
            f.write("1 Region 2.5 0.1\n")
            f.flush()

            stats = _parse_gtm_stats(f.name)

        os.unlink(f.name)
        assert len(stats) == 1
