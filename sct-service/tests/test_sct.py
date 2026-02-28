"""Tests for the SCT backend — CLI pipeline with mocked subprocess calls."""

import csv
import json
import os
import subprocess
import tempfile
from unittest.mock import MagicMock, patch

import nibabel as nib
import numpy as np
import pytest

from app.backends.sct import (
    SpinalCordToolboxBackend,
    _find_spine_nifti,
    _find_dwi_nifti,
    _parse_csa_csv,
    _parse_compression_csv,
    _parse_vertebral_levels,
    _run_sct,
)


class TestAvailability:
    def setup_method(self):
        self.backend = SpinalCordToolboxBackend()

    def test_name(self):
        assert self.backend.name == "sct"

    def test_available_with_cli(self):
        with patch("shutil.which", return_value="/opt/sct/bin/sct_deepseg_sc"):
            assert self.backend.available() is True

    def test_not_available(self):
        with patch("shutil.which", return_value=None):
            assert self.backend.available() is False


class TestFindSpineNifti:
    def test_empty_dir(self):
        with tempfile.TemporaryDirectory() as d:
            assert _find_spine_nifti(d) is None

    def test_prefers_spine_keyword(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "T1w.nii.gz")
            spine = os.path.join(anat, "spine_T2w.nii.gz")
            for f in [t1w, spine]:
                with open(f, "wb") as fh:
                    fh.write(b"\x00" * 10)

            result = _find_spine_nifti(d)
            assert "spine" in result.lower()

    def test_prefers_t2w_over_t1w(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            t2w = os.path.join(anat, "sub-01_T2w.nii.gz")
            for f in [t1w, t2w]:
                with open(f, "wb") as fh:
                    fh.write(b"\x00" * 10)

            result = _find_spine_nifti(d)
            assert "T2w" in result

    def test_falls_back_to_any(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "anat")
            os.makedirs(anat)
            generic = os.path.join(anat, "sub-01_FLAIR.nii.gz")
            with open(generic, "wb") as f:
                f.write(b"\x00" * 10)

            result = _find_spine_nifti(d)
            assert result is not None
            assert "FLAIR" in result


class TestFindDwiNifti:
    def test_no_dwi(self):
        with tempfile.TemporaryDirectory() as d:
            assert _find_dwi_nifti(d) is None

    def test_finds_dwi_in_name(self):
        with tempfile.TemporaryDirectory() as d:
            dwi_dir = os.path.join(d, "sub-01", "dwi")
            os.makedirs(dwi_dir)
            nifti = os.path.join(dwi_dir, "sub-01_dwi.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 10)

            result = _find_dwi_nifti(d)
            assert result is not None
            assert "dwi" in result


class TestRunSct:
    def test_successful_command(self):
        mock_result = MagicMock()
        mock_result.returncode = 0

        with patch("subprocess.run", return_value=mock_result):
            assert _run_sct(["echo", "hello"], "test_step", "/tmp") is True

    def test_nonzero_exit(self):
        mock_result = MagicMock()
        mock_result.returncode = 1
        mock_result.stderr = "error"

        with patch("subprocess.run", return_value=mock_result):
            assert _run_sct(["false"], "test_step", "/tmp") is False

    def test_timeout(self):
        with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 3600)):
            assert _run_sct(["sleep", "9999"], "test_step", "/tmp") is False

    def test_command_not_found(self):
        with patch("subprocess.run", side_effect=FileNotFoundError("not found")):
            assert _run_sct(["nonexistent"], "test_step", "/tmp") is False


class TestParseCsaCsv:
    def test_valid_csv(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["VertLevel", "MEAN(area)"])
            writer.writerow(["3", "64.5"])
            writer.writerow(["4", "62.1"])
            f.flush()

            result = _parse_csa_csv(f.name)
            assert result == {"3": 64.5, "4": 62.1}

        os.unlink(f.name)

    def test_alternative_headers(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["Slice (I->S)", "CSA (mm^2)"])
            writer.writerow(["10", "55.3"])
            f.flush()

            result = _parse_csa_csv(f.name)
            assert result == {"10": 55.3}

        os.unlink(f.name)

    def test_empty_csv(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["VertLevel", "MEAN(area)"])
            f.flush()

            result = _parse_csa_csv(f.name)
            assert result == {}

        os.unlink(f.name)

    def test_missing_file(self):
        result = _parse_csa_csv("/nonexistent/file.csv")
        assert result == {}

    def test_bad_float(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["VertLevel", "MEAN(area)"])
            writer.writerow(["3", "not_a_number"])
            f.flush()

            result = _parse_csa_csv(f.name)
            assert result == {}

        os.unlink(f.name)


class TestParseCompressionCsv:
    def test_valid_csv(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["aMCC", "aSCOR", "MSCC", "ratio_AP"])
            writer.writerow(["0.1500", "0.7200", "0.3100", "0.8500"])
            f.flush()

            result = _parse_compression_csv(f.name)
            assert result["aMCC"] == 0.15
            assert result["aSCOR"] == 0.72
            assert result["MSCC"] == 0.31
            assert result["ratio_AP"] == 0.85

        os.unlink(f.name)

    def test_partial_fields(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as f:
            writer = csv.writer(f)
            writer.writerow(["aMCC", "other_col"])
            writer.writerow(["0.22", "abc"])
            f.flush()

            result = _parse_compression_csv(f.name)
            assert result == {"aMCC": 0.22}

        os.unlink(f.name)

    def test_missing_file(self):
        result = _parse_compression_csv("/nonexistent/file.csv")
        assert result == {}


class TestParseVertebralLevels:
    def test_valid_labeled_nifti(self):
        with tempfile.NamedTemporaryFile(suffix=".nii.gz", delete=False) as f:
            data = np.zeros((10, 10, 10), dtype=np.int32)
            data[0:3, :, :] = 3   # C3
            data[3:5, :, :] = 4   # C4
            data[5:8, :, :] = 8   # T1
            nib.save(nib.Nifti1Image(data, np.eye(4)), f.name)

            levels = _parse_vertebral_levels(f.name)
            assert "C3" in levels
            assert "C4" in levels
            assert "T1" in levels

        os.unlink(f.name)

    def test_empty_nifti(self):
        with tempfile.NamedTemporaryFile(suffix=".nii.gz", delete=False) as f:
            data = np.zeros((5, 5, 5), dtype=np.int32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), f.name)

            levels = _parse_vertebral_levels(f.name)
            assert levels == []

        os.unlink(f.name)

    def test_unknown_label(self):
        with tempfile.NamedTemporaryFile(suffix=".nii.gz", delete=False) as f:
            data = np.zeros((5, 5, 5), dtype=np.int32)
            data[0, 0, 0] = 99  # Not in mapping
            nib.save(nib.Nifti1Image(data, np.eye(4)), f.name)

            levels = _parse_vertebral_levels(f.name)
            assert levels == ["V99"]

        os.unlink(f.name)

    def test_invalid_file(self):
        levels = _parse_vertebral_levels("/nonexistent/file.nii.gz")
        assert levels == []


class TestFullPipeline:
    def setup_method(self):
        self.backend = SpinalCordToolboxBackend()

    def test_no_nifti_files(self):
        with tempfile.TemporaryDirectory() as bids_dir:
            result = self.backend.analyze(bids_dir, "/tmp/out", "1.2.3")
            assert result.success is False
            assert "No NIfTI" in result.error

    def test_deepseg_fails(self):
        """If deepseg_sc fails, whole pipeline fails."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            mock_result = MagicMock()
            mock_result.returncode = 1
            mock_result.stderr = "deepseg error"

            with patch("subprocess.run", return_value=mock_result):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "sct_deepseg_sc failed" in result.error

    def test_full_pipeline_success(self):
        """Mock all 4 steps succeeding (no DWI)."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            # Create input NIfTI
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            data = np.ones((10, 10, 10), dtype=np.float32) * 100
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            call_count = [0]

            def fake_run(cmd, **kwargs):
                call_count[0] += 1
                step = cmd[0]
                mock_result = MagicMock()
                mock_result.returncode = 0
                mock_result.stderr = ""

                if step == "sct_deepseg_sc":
                    seg_sc = os.path.join(out_dir, "seg_sc.nii.gz")
                    seg_data = np.ones((10, 10, 10), dtype=np.int32)
                    nib.save(nib.Nifti1Image(seg_data, np.eye(4)), seg_sc)

                elif step == "sct_label_vertebrae":
                    labeled = os.path.join(out_dir, "sub-01_T2w_labeled.nii.gz")
                    lab_data = np.zeros((10, 10, 10), dtype=np.int32)
                    lab_data[0:3, :, :] = 3   # C3
                    lab_data[3:6, :, :] = 4   # C4
                    lab_data[6:9, :, :] = 5   # C5
                    nib.save(nib.Nifti1Image(lab_data, np.eye(4)), labeled)

                elif step == "sct_process_segmentation":
                    csa_csv = os.path.join(out_dir, "csa_perlevel.csv")
                    with open(csa_csv, "w", newline="") as f:
                        writer = csv.writer(f)
                        writer.writerow(["VertLevel", "MEAN(area)"])
                        writer.writerow(["3", "64.5"])
                        writer.writerow(["4", "62.1"])
                        writer.writerow(["5", "60.8"])

                elif step == "sct_compute_compression":
                    comp_csv = os.path.join(out_dir, "compression.csv")
                    with open(comp_csv, "w", newline="") as f:
                        writer = csv.writer(f)
                        writer.writerow(["aMCC", "aSCOR", "MSCC", "ratio_AP"])
                        writer.writerow(["0.15", "0.72", "0.31", "0.85"])

                return mock_result

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert result.tool == "sct"
            assert "deepseg_sc" in result.metrics["steps_completed"]
            assert "label_vertebrae" in result.metrics["steps_completed"]
            assert "process_segmentation" in result.metrics["steps_completed"]
            assert "compute_compression" in result.metrics["steps_completed"]
            assert result.metrics["csa_per_level"]["3"] == 64.5
            assert result.metrics["mean_csa_mm2"] == 62.47
            assert result.metrics["compression"]["aMCC"] == 0.15
            assert "C3" in result.metrics["vertebral_levels_detected"]
            assert call_count[0] == 4  # 4 SCT commands

            # Summary JSON should exist
            summary_path = os.path.join(out_dir, "summary.json")
            assert os.path.isfile(summary_path)
            with open(summary_path) as f:
                summary = json.load(f)
            assert summary["atlas"] == "sct"

    def test_partial_pipeline_label_fails(self):
        """If labeling fails, CSA still runs (without per-level) and compression is skipped."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            data = np.ones((10, 10, 10), dtype=np.float32) * 100
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            def fake_run(cmd, **kwargs):
                step = cmd[0]
                mock_result = MagicMock()
                mock_result.stderr = ""

                if step == "sct_deepseg_sc":
                    mock_result.returncode = 0
                    seg_sc = os.path.join(out_dir, "seg_sc.nii.gz")
                    seg_data = np.ones((10, 10, 10), dtype=np.int32)
                    nib.save(nib.Nifti1Image(seg_data, np.eye(4)), seg_sc)

                elif step == "sct_label_vertebrae":
                    # Labeling fails
                    mock_result.returncode = 1
                    mock_result.stderr = "labeling failed"

                elif step == "sct_process_segmentation":
                    mock_result.returncode = 0
                    csa_csv = os.path.join(out_dir, "csa_perlevel.csv")
                    with open(csa_csv, "w", newline="") as f:
                        writer = csv.writer(f)
                        writer.writerow(["VertLevel", "MEAN(area)"])
                        writer.writerow(["1", "65.0"])

                else:
                    mock_result.returncode = 0

                return mock_result

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert "deepseg_sc" in result.metrics["steps_completed"]
            assert "label_vertebrae" not in result.metrics["steps_completed"]
            assert "process_segmentation" in result.metrics["steps_completed"]
            # compression not attempted without labels
            assert "compute_compression" not in result.metrics["steps_completed"]

    def test_pipeline_with_dwi(self):
        """Mock pipeline including DTI step when DWI data is present."""
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            # Create spine NIfTI
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            data = np.ones((10, 10, 10), dtype=np.float32)
            nib.save(nib.Nifti1Image(data, np.eye(4)), nifti)

            # Create DWI data
            dwi_dir = os.path.join(bids_dir, "sub-01", "dwi")
            os.makedirs(dwi_dir)
            dwi = os.path.join(dwi_dir, "sub-01_dwi.nii.gz")
            nib.save(nib.Nifti1Image(data, np.eye(4)), dwi)
            with open(dwi.replace(".nii.gz", ".bval"), "w") as f:
                f.write("0 1000\n")
            with open(dwi.replace(".nii.gz", ".bvec"), "w") as f:
                f.write("1 0\n0 1\n0 0\n")

            steps_run = []

            def fake_run(cmd, **kwargs):
                step = cmd[0]
                steps_run.append(step)
                mock_result = MagicMock()
                mock_result.returncode = 0
                mock_result.stderr = ""

                if step == "sct_deepseg_sc":
                    seg_sc = os.path.join(out_dir, "seg_sc.nii.gz")
                    nib.save(nib.Nifti1Image(data, np.eye(4)), seg_sc)

                elif step == "sct_label_vertebrae":
                    labeled = os.path.join(out_dir, "sub-01_T2w_labeled.nii.gz")
                    lab_data = np.zeros((10, 10, 10), dtype=np.int32)
                    lab_data[0:5, :, :] = 3
                    nib.save(nib.Nifti1Image(lab_data, np.eye(4)), labeled)

                elif step == "sct_process_segmentation":
                    csa = os.path.join(out_dir, "csa_perlevel.csv")
                    with open(csa, "w", newline="") as f:
                        csv.writer(f).writerow(["VertLevel", "MEAN(area)"])

                elif step == "sct_compute_compression":
                    comp = os.path.join(out_dir, "compression.csv")
                    with open(comp, "w", newline="") as f:
                        csv.writer(f).writerow(["aMCC", "aSCOR"])

                elif step == "sct_dmri_compute_dti":
                    dti_dir = os.path.join(out_dir, "dti")
                    os.makedirs(dti_dir, exist_ok=True)
                    fa = os.path.join(dti_dir, "dti_FA.nii.gz")
                    nib.save(nib.Nifti1Image(data, np.eye(4)), fa)

                return mock_result

            with patch("subprocess.run", side_effect=fake_run):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")

            assert result.success is True
            assert "sct_dmri_compute_dti" in steps_run
            assert "dmri_compute_dti" in result.metrics["steps_completed"]
            assert "dti_outputs" in result.metrics

    def test_timeout_on_deepseg(self):
        with tempfile.TemporaryDirectory() as bids_dir, tempfile.TemporaryDirectory() as out_dir:
            anat = os.path.join(bids_dir, "sub-01", "anat")
            os.makedirs(anat)
            nifti = os.path.join(anat, "sub-01_T2w.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 100)

            with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("cmd", 3600)):
                result = self.backend.analyze(bids_dir, out_dir, "1.2.3")
                assert result.success is False
                assert "sct_deepseg_sc failed" in result.error
