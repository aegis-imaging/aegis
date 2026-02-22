"""Unit tests for the phantom generation module.

All tests use synthetic data — no DICOM files on disk required.
The write_dicom_series tests use tmp_path to avoid filesystem side effects.
"""

import numpy as np
import pytest

from app.phantom import (
    _add_cortical_noise,
    _add_face_region,
    _shepp_logan_phantom,
    _t1w_remap,
    generate_phantom_slices,
    write_dicom_series,
)


# ── _shepp_logan_phantom ──────────────────────────────────────────────────────


class TestSheppLogan:
    def test_output_shape(self):
        p = _shepp_logan_phantom(64)
        assert p.shape == (64, 64)

    def test_range_normalised(self):
        p = _shepp_logan_phantom(64)
        assert p.min() >= 0.0
        assert p.max() <= 1.0

    def test_has_nonzero(self):
        p = _shepp_logan_phantom(64)
        assert p.max() > 0.0

    def test_small_size(self):
        p = _shepp_logan_phantom(16)
        assert p.shape == (16, 16)


# ── _t1w_remap ────────────────────────────────────────────────────────────────


class TestT1wRemap:
    def test_output_shape_preserved(self):
        p = _shepp_logan_phantom(32)
        out = _t1w_remap(p)
        assert out.shape == p.shape

    def test_range_non_negative(self):
        p = _shepp_logan_phantom(32)
        out = _t1w_remap(p)
        assert out.min() >= 0.0
        assert out.max() <= 1.0

    def test_background_stays_zero(self):
        # Pure zero input should map to zero.
        p = np.zeros((16, 16))
        out = _t1w_remap(p)
        assert out.max() == 0.0

    def test_high_phantom_is_bright(self):
        # High phantom values (white matter) should be bright (>0.7 in T1w).
        p = np.ones((4, 4)) * 0.95
        out = _t1w_remap(p)
        assert out.mean() > 0.7


# ── _add_cortical_noise ───────────────────────────────────────────────────────


class TestCorticalNoise:
    def test_output_shape_preserved(self):
        img = np.full((32, 32), 0.5)
        rng = np.random.default_rng(0)
        out = _add_cortical_noise(img, rng)
        assert out.shape == img.shape

    def test_output_clipped(self):
        img = np.full((32, 32), 0.5)
        rng = np.random.default_rng(0)
        out = _add_cortical_noise(img, rng)
        assert out.min() >= 0.0
        assert out.max() <= 1.0

    def test_differs_from_input(self):
        img = np.full((32, 32), 0.5)
        rng = np.random.default_rng(99)
        out = _add_cortical_noise(img, rng)
        # Rician noise should change the image.
        assert not np.allclose(img, out)


# ── _add_face_region ──────────────────────────────────────────────────────────


class TestFaceRegion:
    def test_superior_slices_unchanged(self):
        """z_norm >= 0.35 should return the canvas unchanged."""
        canvas = np.full((64, 64), 0.5)
        out = _add_face_region(canvas, z_norm=0.5, size=64)
        assert np.allclose(canvas, out)

    def test_inferior_slices_modified(self):
        """z_norm = 0 (most inferior) should change the canvas."""
        canvas = np.full((64, 64), 0.5)
        out = _add_face_region(canvas, z_norm=0.0, size=64)
        assert not np.allclose(canvas, out)

    def test_output_clipped(self):
        canvas = np.full((64, 64), 0.5)
        out = _add_face_region(canvas, z_norm=0.0, size=64)
        assert out.min() >= 0.0
        assert out.max() <= 1.0

    def test_orbital_region_dark(self):
        """Orbital air cells should be hypointense (dark)."""
        canvas = np.full((64, 64), 0.5)
        out = _add_face_region(canvas, z_norm=0.0, size=64)
        # Near-centre lateral points should have been darkened.
        ctr = 32
        ox = int(ctr + 64 * 0.13)
        oy = int(ctr - 64 * 0.12)
        assert out[oy, ox] < 0.2


# ── generate_phantom_slices ───────────────────────────────────────────────────


class TestGeneratePhantomSlices:
    def test_returns_correct_count(self):
        slices = generate_phantom_slices(n_slices=5, size=32)
        assert len(slices) == 5

    def test_tuple_structure(self):
        slices = generate_phantom_slices(n_slices=3, size=32)
        for arr, study_uid, series_uid, instance_num, slice_z in slices:
            assert arr.shape == (32, 32)
            assert isinstance(study_uid, str) and len(study_uid) > 5
            assert isinstance(series_uid, str) and len(series_uid) > 5
            assert isinstance(instance_num, int)
            assert isinstance(slice_z, float)

    def test_shared_study_uid(self):
        slices = generate_phantom_slices(n_slices=4, size=32)
        study_uids = {s[1] for s in slices}
        assert len(study_uids) == 1, "All slices should share the same study_uid"

    def test_shared_series_uid(self):
        slices = generate_phantom_slices(n_slices=4, size=32)
        series_uids = {s[2] for s in slices}
        assert len(series_uids) == 1

    def test_instance_numbers_sequential(self):
        slices = generate_phantom_slices(n_slices=5, size=32)
        nums = [s[3] for s in slices]
        assert nums == list(range(1, 6))

    def test_pixel_range(self):
        slices = generate_phantom_slices(n_slices=3, size=32)
        for arr, *_ in slices:
            assert arr.min() >= 0.0
            assert arr.max() <= 1.0

    def test_seed_reproducibility(self):
        s1 = generate_phantom_slices(n_slices=2, size=32, seed=7)
        s2 = generate_phantom_slices(n_slices=2, size=32, seed=7)
        for (a1, *_), (a2, *_) in zip(s1, s2):
            assert np.allclose(a1, a2)

    def test_different_seeds_differ(self):
        s1 = generate_phantom_slices(n_slices=2, size=32, seed=1)
        s2 = generate_phantom_slices(n_slices=2, size=32, seed=2)
        assert not np.allclose(s1[0][0], s2[0][0])

    def test_with_face_differs_from_without(self):
        s_no_face = generate_phantom_slices(n_slices=10, size=64, seed=0, with_face=False)
        s_face = generate_phantom_slices(n_slices=10, size=64, seed=0, with_face=True)
        # Inferior slices (lower index) should differ; superior slices may be similar.
        # At least the first slice should differ.
        assert not np.allclose(s_no_face[0][0], s_face[0][0])


# ── write_dicom_series ────────────────────────────────────────────────────────


class TestWriteDicomSeries:
    def test_creates_files(self, tmp_path):
        slices = generate_phantom_slices(n_slices=3, size=32)
        paths = write_dicom_series(slices, str(tmp_path / "out"), size=32)
        assert len(paths) == 3
        for p in paths:
            import os
            assert os.path.exists(p)

    def test_files_are_valid_dicom(self, tmp_path):
        import pydicom
        slices = generate_phantom_slices(n_slices=2, size=32)
        paths = write_dicom_series(slices, str(tmp_path / "out"), size=32)
        for p in paths:
            ds = pydicom.dcmread(p)
            assert ds.Modality == "MR"
            assert ds.Rows == 32
            assert ds.Columns == 32

    def test_synthetic_phi_tags(self, tmp_path):
        import pydicom
        slices = generate_phantom_slices(n_slices=1, size=32)
        paths = write_dicom_series(slices, str(tmp_path / "out"), size=32)
        ds = pydicom.dcmread(paths[0])
        assert str(ds.PatientID) == "PAT-SYNTH-001"
        assert str(ds.PatientName) == "DEMO^PATIENT^MRI"

    def test_study_uid_matches(self, tmp_path):
        import pydicom
        slices = generate_phantom_slices(n_slices=2, size=32)
        expected_uid = slices[0][1]
        paths = write_dicom_series(slices, str(tmp_path / "out"), size=32)
        for p in paths:
            ds = pydicom.dcmread(p)
            assert ds.StudyInstanceUID == expected_uid

    def test_creates_output_dir(self, tmp_path):
        slices = generate_phantom_slices(n_slices=1, size=32)
        nested = tmp_path / "deep" / "nested" / "dir"
        write_dicom_series(slices, str(nested), size=32)
        assert nested.exists()
