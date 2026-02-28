"""Tests for shared segmentation utilities (seg_utils)."""

import os
import tempfile

import nibabel as nib
import numpy as np
import pytest

from app.backends.seg_utils import (
    compute_label_stats,
    compute_label_volumes,
    find_any_nifti,
    find_asl_nifti,
    find_pet_nifti,
    find_qsm_niftis,
    find_t1w_nifti,
)


def _make_nifti(path: str, data: np.ndarray, voxel_dims=(1.0, 1.0, 1.0)):
    """Create a minimal NIfTI file with the given data and voxel dimensions."""
    affine = np.diag(list(voxel_dims) + [1.0])
    img = nib.Nifti1Image(data, affine)
    nib.save(img, path)
    return path


class TestFindAnyNifti:
    def test_returns_none_empty_dir(self):
        with tempfile.TemporaryDirectory() as d:
            assert find_any_nifti(d) is None

    def test_prefers_t1w(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            flair = os.path.join(anat, "sub-01_FLAIR.nii.gz")
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            for f in [flair, t1w]:
                with open(f, "wb") as fp:
                    fp.write(b"\x00" * 10)

            result = find_any_nifti(d)
            assert "T1w" in result

    def test_falls_back_to_first_nifti(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            other = os.path.join(anat, "sub-01_angio.nii.gz")
            with open(other, "wb") as f:
                f.write(b"\x00" * 10)

            result = find_any_nifti(d)
            assert result == other

    def test_prefers_t2w_over_flair(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            flair = os.path.join(anat, "sub-01_FLAIR.nii.gz")
            t2w = os.path.join(anat, "sub-01_T2w.nii.gz")
            for f in [flair, t2w]:
                with open(f, "wb") as fp:
                    fp.write(b"\x00" * 10)

            result = find_any_nifti(d)
            assert "T2w" in result


class TestFindT1wNifti:
    def test_returns_none_empty_dir(self):
        with tempfile.TemporaryDirectory() as d:
            assert find_t1w_nifti(d) is None

    def test_finds_t1w(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 10)

            result = find_t1w_nifti(d)
            assert result == t1w

    def test_ignores_non_t1w(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            flair = os.path.join(anat, "sub-01_FLAIR.nii.gz")
            with open(flair, "wb") as f:
                f.write(b"\x00" * 10)

            assert find_t1w_nifti(d) is None


class TestComputeLabelVolumes:
    def test_basic_volumes(self):
        with tempfile.TemporaryDirectory() as d:
            # 3x3x3 volume with two labels: 1 has 5 voxels, 2 has 3 voxels
            data = np.zeros((3, 3, 3), dtype=np.int32)
            data[0, 0, 0] = 1
            data[0, 0, 1] = 1
            data[0, 0, 2] = 1
            data[0, 1, 0] = 1
            data[0, 1, 1] = 1
            data[1, 0, 0] = 2
            data[1, 0, 1] = 2
            data[1, 0, 2] = 2

            seg_path = os.path.join(d, "seg.nii.gz")
            _make_nifti(seg_path, data, voxel_dims=(2.0, 2.0, 2.0))

            label_map = {1: "region_A", 2: "region_B"}
            volumes = compute_label_volumes(seg_path, label_map)

            assert "region_A" in volumes
            assert "region_B" in volumes
            # 5 voxels × 8 mm³ = 40.0
            assert volumes["region_A"] == 40.0
            # 3 voxels × 8 mm³ = 24.0
            assert volumes["region_B"] == 24.0

    def test_skips_background(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 2, 2), dtype=np.int32)
            data[0, 0, 0] = 1

            seg_path = os.path.join(d, "seg.nii.gz")
            _make_nifti(seg_path, data)

            label_map = {0: "background", 1: "brain"}
            volumes = compute_label_volumes(seg_path, label_map)

            assert "brain" in volumes
            assert "background" not in volumes

    def test_skips_unmapped_labels(self):
        with tempfile.TemporaryDirectory() as d:
            data = np.zeros((2, 2, 2), dtype=np.int32)
            data[0, 0, 0] = 1
            data[0, 0, 1] = 99  # Not in label_map

            seg_path = os.path.join(d, "seg.nii.gz")
            _make_nifti(seg_path, data)

            label_map = {1: "brain"}
            volumes = compute_label_volumes(seg_path, label_map)

            assert len(volumes) == 1
            assert "brain" in volumes


class TestComputeLabelStats:
    def test_basic_stats(self):
        with tempfile.TemporaryDirectory() as d:
            # Segmentation: one label (1) covering 4 voxels
            seg_data = np.zeros((2, 2, 2), dtype=np.int32)
            seg_data[0, 0, 0] = 1
            seg_data[0, 0, 1] = 1
            seg_data[0, 1, 0] = 1
            seg_data[0, 1, 1] = 1

            # Intensity image: values 10, 20, 30, 40 in those voxels
            int_data = np.zeros((2, 2, 2), dtype=np.float64)
            int_data[0, 0, 0] = 10.0
            int_data[0, 0, 1] = 20.0
            int_data[0, 1, 0] = 30.0
            int_data[0, 1, 1] = 40.0

            seg_path = os.path.join(d, "seg.nii.gz")
            int_path = os.path.join(d, "t1w.nii.gz")
            _make_nifti(seg_path, seg_data)
            _make_nifti(int_path, int_data)

            label_map = {1: "test_region"}
            stats = compute_label_stats(seg_path, int_path, label_map)

            assert len(stats) == 1
            s = stats[0]
            assert s["roi_name"] == "test_region"
            assert s["voxel_count"] == 4
            assert s["mean"] == 25.0
            assert s["median"] == 25.0
            assert s["std"] > 0

    def test_sorted_by_roi_name(self):
        with tempfile.TemporaryDirectory() as d:
            seg_data = np.zeros((3, 1, 1), dtype=np.int32)
            seg_data[0] = 2
            seg_data[1] = 1
            seg_data[2] = 3

            int_data = np.ones((3, 1, 1), dtype=np.float64) * 100

            seg_path = os.path.join(d, "seg.nii.gz")
            int_path = os.path.join(d, "t1w.nii.gz")
            _make_nifti(seg_path, seg_data)
            _make_nifti(int_path, int_data)

            label_map = {1: "B_region", 2: "A_region", 3: "C_region"}
            stats = compute_label_stats(seg_path, int_path, label_map)

            names = [s["roi_name"] for s in stats]
            assert names == ["A_region", "B_region", "C_region"]


class TestFindPetNifti:
    def test_returns_none_empty_dir(self):
        with tempfile.TemporaryDirectory() as d:
            assert find_pet_nifti(d) is None

    def test_finds_pet_by_name(self):
        with tempfile.TemporaryDirectory() as d:
            pet_dir = os.path.join(d, "sub-01", "pet")
            os.makedirs(pet_dir)
            pet = os.path.join(pet_dir, "sub-01_pet.nii.gz")
            with open(pet, "wb") as f:
                f.write(b"\x00" * 10)
            assert find_pet_nifti(d) == pet

    def test_finds_pet_in_pet_subdir(self):
        with tempfile.TemporaryDirectory() as d:
            pet_dir = os.path.join(d, "sub-01", "pet")
            os.makedirs(pet_dir)
            nifti = os.path.join(pet_dir, "sub-01_trc-PIB_run-1.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 10)
            # Should find it in pet/ subdir even without "pet" in filename
            # Actually it won't match filename patterns, but will match subdir
            # This tests the fallback to pet/ directory
            result = find_pet_nifti(d)
            assert result == nifti

    def test_ignores_non_pet(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 10)
            assert find_pet_nifti(d) is None


class TestFindAslNifti:
    def test_returns_none_empty_dir(self):
        with tempfile.TemporaryDirectory() as d:
            assert find_asl_nifti(d) is None

    def test_finds_asl_by_name(self):
        with tempfile.TemporaryDirectory() as d:
            perf = os.path.join(d, "sub-01", "perf")
            os.makedirs(perf)
            asl = os.path.join(perf, "sub-01_asl.nii.gz")
            with open(asl, "wb") as f:
                f.write(b"\x00" * 10)
            assert find_asl_nifti(d) == asl

    def test_finds_perf_by_name(self):
        with tempfile.TemporaryDirectory() as d:
            func = os.path.join(d, "sub-01", "func")
            os.makedirs(func)
            perf = os.path.join(func, "sub-01_perfusion.nii.gz")
            with open(perf, "wb") as f:
                f.write(b"\x00" * 10)
            assert find_asl_nifti(d) == perf

    def test_finds_in_perf_subdir(self):
        with tempfile.TemporaryDirectory() as d:
            perf_dir = os.path.join(d, "sub-01", "perf")
            os.makedirs(perf_dir)
            nifti = os.path.join(perf_dir, "sub-01_cbf.nii.gz")
            with open(nifti, "wb") as f:
                f.write(b"\x00" * 10)
            result = find_asl_nifti(d)
            assert result == nifti

    def test_ignores_non_asl(self):
        with tempfile.TemporaryDirectory() as d:
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x00" * 10)
            assert find_asl_nifti(d) is None


class TestFindQSMNiftis:
    def test_returns_none_empty_dir(self):
        with tempfile.TemporaryDirectory() as d:
            mag, phase = find_qsm_niftis(d)
            assert mag is None
            assert phase is None

    def test_finds_magnitude_and_phase(self):
        with tempfile.TemporaryDirectory() as d:
            fmap = os.path.join(d, "sub-01", "fmap")
            os.makedirs(fmap)
            mag = os.path.join(fmap, "sub-01_magnitude1.nii.gz")
            phase = os.path.join(fmap, "sub-01_phasediff.nii.gz")
            for f in [mag, phase]:
                with open(f, "wb") as fp:
                    fp.write(b"\x00" * 10)

            found_mag, found_phase = find_qsm_niftis(d)
            assert found_mag == mag
            assert found_phase == phase

    def test_finds_phase_only(self):
        with tempfile.TemporaryDirectory() as d:
            fmap = os.path.join(d, "sub-01", "fmap")
            os.makedirs(fmap)
            phase = os.path.join(fmap, "sub-01_phase.nii.gz")
            with open(phase, "wb") as f:
                f.write(b"\x00" * 10)

            found_mag, found_phase = find_qsm_niftis(d)
            assert found_mag is None
            assert found_phase == phase

    def test_echo_fallback(self):
        with tempfile.TemporaryDirectory() as d:
            fmap = os.path.join(d, "sub-01", "fmap")
            os.makedirs(fmap)
            echo1 = os.path.join(fmap, "sub-01_echo-1.nii.gz")
            echo2 = os.path.join(fmap, "sub-01_echo-2.nii.gz")
            for f in [echo1, echo2]:
                with open(f, "wb") as fp:
                    fp.write(b"\x00" * 10)

            found_mag, found_phase = find_qsm_niftis(d)
            assert found_mag == echo1
            assert found_phase == echo2
