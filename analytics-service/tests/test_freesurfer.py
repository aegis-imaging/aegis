"""Tests for the FreeSurfer analytics backend."""

import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock

import pytest

from app.backends.freesurfer import FreeSurferBackend, _parse_stats_file


class TestParseStatsFile:
    def test_parse_aseg_stats(self, tmp_path):
        stats = tmp_path / "aseg.stats"
        stats.write_text(
            "# Table of volumes\n"
            "# ColHeaders  Index SegId NVoxels Volume_mm3 StructName\n"
            "  1   4    3456   3456.0 Left-Lateral-Ventricle\n"
            "  2  43    2890   2890.0 Right-Lateral-Ventricle\n"
            "  3  17    4200   4200.0 Left-Hippocampus\n"
        )
        result = _parse_stats_file(str(stats))
        assert result["Left-Lateral-Ventricle"] == 3456.0
        assert result["Left-Hippocampus"] == 4200.0
        assert len(result) == 3

    def test_skips_comments_and_blanks(self, tmp_path):
        stats = tmp_path / "test.stats"
        stats.write_text(
            "# comment line\n"
            "\n"
            "  1   4    3456   3456.0 Region1\n"
            "# another comment\n"
        )
        result = _parse_stats_file(str(stats))
        assert len(result) == 1

    def test_missing_file(self):
        result = _parse_stats_file("/nonexistent/stats")
        assert result == {}

    def test_invalid_volume(self, tmp_path):
        stats = tmp_path / "bad.stats"
        stats.write_text("  1   4    3456   notanumber Region1\n")
        result = _parse_stats_file(str(stats))
        assert result == {}


class TestFreeSurferBackend:
    def setup_method(self):
        self.backend = FreeSurferBackend()

    def test_name(self):
        assert self.backend.name == "freesurfer"

    def test_not_available_without_binary(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False

    def test_not_available_without_license(self):
        with patch("shutil.which", return_value="/usr/bin/recon-all"):
            with patch("os.path.isfile", return_value=False):
                assert self.backend.available() is False

    def test_available_with_binary_and_license(self):
        with patch("shutil.which", return_value="/usr/bin/recon-all"):
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

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("recon-all", 86400)):
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                assert result.success is False
                assert "timed out" in result.error

    def test_recon_all_failure(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "ERROR: recon-all failed"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
                assert result.success is False
                assert "recon-all failed" in result.error

    def test_successful_analysis(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 0

            # Create expected output structure
            subj_dir = os.path.join(out_dir, "freesurfer", "sub-1234abcd")
            stats_dir = os.path.join(subj_dir, "stats")
            os.makedirs(stats_dir)
            with open(os.path.join(stats_dir, "aseg.stats"), "w") as f:
                f.write("  1   4    3456   3456.0 Left-Hippocampus\n")

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1234abcd-5678")
                assert result.success is True
                assert result.tool == "freesurfer"
                assert result.duration_seconds > 0
