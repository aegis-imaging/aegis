"""Tests for shared atlas utility functions."""

import csv
import os
import subprocess
import tempfile
from unittest.mock import patch, MagicMock

import numpy as np
import pytest

from app.backends.atlas_utils import (
    extract_roi_stats,
    find_atlas_nifti,
    get_ants_env,
    load_atlas_labels,
    run_brain_extraction,
    run_n4_bias_correction,
    warp_atlas_to_subject,
)


class TestLoadAtlasLabels:
    def test_loads_bundled_aal3(self):
        labels = load_atlas_labels("aal3")
        assert len(labels) == 166
        assert labels[1] == "Precentral_L"
        assert labels[41] == "Hippocampus_L"
        assert labels[166] == "MnR_R"

    def test_filesystem_fallback(self):
        with tempfile.TemporaryDirectory() as d:
            atlas_dir = os.path.join(d, "test_atlas")
            os.makedirs(atlas_dir)
            csv_path = os.path.join(atlas_dir, "test_atlas_labels.csv")
            with open(csv_path, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(["roi_id", "roi_name"])
                writer.writerow([1, "Region_A"])
                writer.writerow([2, "Region_B"])

            with patch("app.backends.atlas_utils.config") as mock_cfg:
                mock_cfg.ATLAS_DIR = d
                labels = load_atlas_labels("test_atlas")
                assert labels == {1: "Region_A", 2: "Region_B"}

    def test_not_found_raises(self):
        with patch("app.backends.atlas_utils.config") as mock_cfg:
            mock_cfg.ATLAS_DIR = "/nonexistent"
            with pytest.raises(FileNotFoundError):
                load_atlas_labels("nonexistent_atlas")


class TestFindAtlasNifti:
    def test_finds_nifti(self):
        with tempfile.TemporaryDirectory() as d:
            atlas_dir = os.path.join(d, "myatlas")
            os.makedirs(atlas_dir)
            nifti = os.path.join(atlas_dir, "myatlas_1mm.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 10)

            with patch("app.backends.atlas_utils.config") as mock_cfg:
                mock_cfg.ATLAS_DIR = d
                result = find_atlas_nifti("myatlas")
                assert result == nifti

    def test_not_found(self):
        with patch("app.backends.atlas_utils.config") as mock_cfg:
            mock_cfg.ATLAS_DIR = "/nonexistent"
            with pytest.raises(FileNotFoundError):
                find_atlas_nifti("missing")


class TestGetAntsEnv:
    def test_includes_antspath(self):
        with patch("app.backends.atlas_utils.config") as mock_cfg:
            mock_cfg.ANTSPATH = "/opt/ants/bin"
            env = get_ants_env()
            assert env["ANTSPATH"] == "/opt/ants/bin"
            assert env["PATH"].startswith("/opt/ants/bin:")


class TestRunN4BiasCorrection:
    def test_success(self):
        mock_result = MagicMock()
        mock_result.returncode = 0
        with patch("subprocess.run", return_value=mock_result):
            ok, err = run_n4_bias_correction("/in.nii.gz", "/out.nii.gz", env={})
            assert ok is True
            assert err == ""

    def test_failure(self):
        mock_result = MagicMock()
        mock_result.returncode = 1
        mock_result.stderr = "N4 error"
        with patch("subprocess.run", return_value=mock_result):
            ok, err = run_n4_bias_correction("/in.nii.gz", "/out.nii.gz", env={})
            assert ok is False
            assert "N4 error" in err

    def test_not_found(self):
        with patch("subprocess.run", side_effect=FileNotFoundError):
            ok, err = run_n4_bias_correction("/in.nii.gz", "/out.nii.gz", env={})
            assert ok is False
            assert "not found" in err

    def test_timeout(self):
        with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("N4", 60)):
            ok, err = run_n4_bias_correction("/in.nii.gz", "/out.nii.gz", env={}, timeout=60)
            assert ok is False
            assert "timed out" in err


class TestRunBrainExtraction:
    def test_success(self):
        mock_result = MagicMock()
        mock_result.returncode = 0

        with tempfile.TemporaryDirectory() as d:
            prefix = os.path.join(d, "brain_")
            brain_out = f"{prefix}BrainExtractionBrain.nii.gz"
            with open(brain_out, "wb") as f:
                f.write(b"\x00" * 10)

            template_dir = os.path.join(d, "templates")
            os.makedirs(template_dir)
            for name in ["T_template0.nii.gz", "T_template0_BrainCerebellumProbabilityMask.nii.gz"]:
                with open(os.path.join(template_dir, name), "wb") as f:
                    f.write(b"\x00" * 10)

            with patch("subprocess.run", return_value=mock_result):
                ok, path, err = run_brain_extraction("/in.nii.gz", prefix, template_dir=template_dir, env={})
                assert ok is True
                assert path == brain_out
                assert err == ""

    def test_missing_templates(self):
        ok, path, err = run_brain_extraction("/in.nii.gz", "/tmp/out_", template_dir="/nonexistent", env={})
        assert ok is False
        assert "not found" in err


class TestWarpAtlasToSubject:
    def test_success(self):
        with tempfile.TemporaryDirectory() as d:
            output_path = os.path.join(d, "warped.nii.gz")
            prefix = os.path.join(d, "atlas_reg_")

            def run_side_effect(cmd, **kwargs):
                # Create expected outputs
                if "antsRegistrationSyN" in str(cmd):
                    for suffix in ["1Warp.nii.gz", "0GenericAffine.mat"]:
                        with open(f"{prefix}{suffix}", "wb") as f:
                            f.write(b"\x00" * 10)
                elif "antsApplyTransforms" in str(cmd):
                    with open(output_path, "wb") as f:
                        f.write(b"\x00" * 10)
                mock = MagicMock()
                mock.returncode = 0
                return mock

            with patch("subprocess.run", side_effect=run_side_effect):
                ok, err = warp_atlas_to_subject(
                    "/atlas.nii.gz", "/template.nii.gz", "/subject.nii.gz",
                    output_path, env={},
                )
                assert ok is True
                assert err == ""

    def test_registration_failure(self):
        mock_result = MagicMock()
        mock_result.returncode = 1
        mock_result.stderr = "reg failed"
        with patch("subprocess.run", return_value=mock_result):
            ok, err = warp_atlas_to_subject(
                "/a.nii", "/t.nii", "/s.nii", "/out.nii", env={},
            )
            assert ok is False
            assert "reg failed" in err

    def test_binary_not_found(self):
        with patch("subprocess.run", side_effect=FileNotFoundError):
            ok, err = warp_atlas_to_subject(
                "/a.nii", "/t.nii", "/s.nii", "/out.nii", env={},
            )
            assert ok is False
            assert "not found" in err


class TestExtractROIStats:
    def _make_nifti(self, data, affine=None, path=None):
        """Create a temporary NIfTI file with given data array."""
        import nibabel as nib

        if affine is None:
            affine = np.diag([2.0, 2.0, 2.0, 1.0])  # 2mm isotropic
        img = nib.Nifti1Image(data.astype(np.float32), affine)
        if path is None:
            fd, path = tempfile.mkstemp(suffix=".nii.gz")
            os.close(fd)
        nib.save(img, path)
        return path

    def test_basic_extraction(self):
        # Atlas: 3x3x3 with ROI 1 (top half) and ROI 2 (bottom half)
        atlas = np.zeros((4, 4, 4), dtype=np.int32)
        atlas[:2, :, :] = 1
        atlas[2:, :, :] = 2

        # Data: uniform value per region
        data = np.zeros((4, 4, 4), dtype=np.float32)
        data[:2, :, :] = 100.0
        data[2:, :, :] = 200.0

        labels = {1: "Region_A", 2: "Region_B"}

        atlas_path = self._make_nifti(atlas)
        data_path = self._make_nifti(data)

        try:
            stats = extract_roi_stats(atlas_path, data_path, labels)
            assert len(stats) == 2

            a = next(s for s in stats if s["roi_id"] == 1)
            assert a["roi_name"] == "Region_A"
            assert a["voxel_count"] == 32  # 2*4*4
            assert a["mean"] == pytest.approx(100.0)
            assert a["volume_mm3"] == pytest.approx(32 * 8.0)  # 2mm^3 = 8mm^3

            b = next(s for s in stats if s["roi_id"] == 2)
            assert b["mean"] == pytest.approx(200.0)
        finally:
            os.unlink(atlas_path)
            os.unlink(data_path)

    def test_skips_background(self):
        atlas = np.zeros((3, 3, 3), dtype=np.int32)
        atlas[0, 0, 0] = 1
        data = np.ones((3, 3, 3), dtype=np.float32)
        labels = {0: "Background", 1: "ROI"}

        atlas_path = self._make_nifti(atlas)
        data_path = self._make_nifti(data)

        try:
            stats = extract_roi_stats(atlas_path, data_path, labels)
            assert len(stats) == 1
            assert stats[0]["roi_id"] == 1
        finally:
            os.unlink(atlas_path)
            os.unlink(data_path)

    def test_skips_unlabeled_rois(self):
        atlas = np.zeros((3, 3, 3), dtype=np.int32)
        atlas[0, 0, 0] = 1
        atlas[1, 1, 1] = 99  # Not in label_map
        data = np.ones((3, 3, 3), dtype=np.float32)
        labels = {1: "Known"}

        atlas_path = self._make_nifti(atlas)
        data_path = self._make_nifti(data)

        try:
            stats = extract_roi_stats(atlas_path, data_path, labels)
            assert len(stats) == 1
            assert stats[0]["roi_name"] == "Known"
        finally:
            os.unlink(atlas_path)
            os.unlink(data_path)

    def test_sorted_by_roi_id(self):
        atlas = np.zeros((3, 3, 3), dtype=np.int32)
        atlas[0, :, :] = 5
        atlas[1, :, :] = 2
        atlas[2, :, :] = 10
        data = np.ones((3, 3, 3), dtype=np.float32)
        labels = {2: "B", 5: "A", 10: "C"}

        atlas_path = self._make_nifti(atlas)
        data_path = self._make_nifti(data)

        try:
            stats = extract_roi_stats(atlas_path, data_path, labels)
            ids = [s["roi_id"] for s in stats]
            assert ids == [2, 5, 10]
        finally:
            os.unlink(atlas_path)
            os.unlink(data_path)
