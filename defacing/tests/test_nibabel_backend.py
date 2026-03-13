"""
Tests for app.backends.nibabel_fallback — the pure-Python defacing backend.
"""

import os
from pathlib import Path

import numpy as np
import pydicom
from pydicom.uid import generate_uid

import pytest

from app.backends.nibabel_fallback import NibabelFallbackBackend, FACE_FRACTION


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _read_pixel_sum(dcm_path: str) -> int:
    """Read a DICOM file and return the sum of its pixel data."""
    ds = pydicom.dcmread(dcm_path)
    return int(np.sum(ds.pixel_array))


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

class TestNibabelBackendProperties:
    """Basic property tests for the NibabelFallbackBackend."""

    def test_available(self):
        """Backend is available when pydicom + numpy are installed (always in test env)."""
        backend = NibabelFallbackBackend()
        assert backend.available() is True

    def test_name(self):
        """Backend reports its name correctly."""
        backend = NibabelFallbackBackend()
        assert backend.name == "nibabel-fallback"


class TestDeface:
    """Tests for NibabelFallbackBackend.deface()."""

    def test_deface_basic(self, make_dicom_file, tmp_path):
        """
        Run deface on a 25-slice series and verify:
        - Output files exist
        - Same number of output files as input
        - Pixel data has been modified (face region zeroed)
        """
        backend = NibabelFallbackBackend()

        series_uid = generate_uid()
        study_uid = generate_uid()
        input_dir = str(tmp_path / "input")
        output_dir = str(tmp_path / "output")
        os.makedirs(input_dir, exist_ok=True)

        input_paths = []
        for i in range(25):
            p = make_dicom_file(
                input_dir,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )
            input_paths.append(p)

        # Record original pixel sums
        original_sums = [_read_pixel_sum(p) for p in input_paths]

        # Run defacing
        out_paths = backend.deface(input_dir, output_dir)

        assert len(out_paths) == 25
        for p in out_paths:
            assert os.path.exists(p)

        # At least some slices should have different pixel data (zeroed face region).
        # With standard axial orientation, the A/P axis is rows (dim 1),
        # so ~30% of rows are zeroed, affecting most slices.
        defaced_sums = [_read_pixel_sum(p) for p in out_paths]
        changes = sum(1 for orig, new in zip(original_sums, defaced_sums) if orig != new)
        assert changes > 0, "Expected at least some slices to have modified pixel data"

    def test_deface_no_files(self, tmp_path):
        """Deface on an empty directory raises RuntimeError."""
        backend = NibabelFallbackBackend()
        input_dir = str(tmp_path / "empty")
        output_dir = str(tmp_path / "output")
        os.makedirs(input_dir, exist_ok=True)

        with pytest.raises(RuntimeError, match="No DICOM files found"):
            backend.deface(input_dir, output_dir)

    def test_output_file_count(self, make_dicom_file, tmp_path):
        """
        Output file count must equal input file count — every input slice
        produces exactly one output file, whether defaced or passthrough.
        """
        backend = NibabelFallbackBackend()

        series_uid = generate_uid()
        study_uid = generate_uid()
        input_dir = str(tmp_path / "input")
        output_dir = str(tmp_path / "output")
        os.makedirs(input_dir, exist_ok=True)

        n_slices = 30
        for i in range(n_slices):
            make_dicom_file(
                input_dir,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )

        out_paths = backend.deface(input_dir, output_dir)
        assert len(out_paths) == n_slices

    def test_output_is_valid_dicom(self, make_dicom_file, tmp_path):
        """Output files are valid DICOM — pydicom can read them back."""
        backend = NibabelFallbackBackend()

        series_uid = generate_uid()
        study_uid = generate_uid()
        input_dir = str(tmp_path / "input")
        output_dir = str(tmp_path / "output")
        os.makedirs(input_dir, exist_ok=True)

        for i in range(25):
            make_dicom_file(
                input_dir,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )

        out_paths = backend.deface(input_dir, output_dir)
        for p in out_paths:
            ds = pydicom.dcmread(p)
            # Should still have pixel data and be parseable
            assert hasattr(ds, "PixelData")
            arr = ds.pixel_array
            assert arr.shape == (64, 64)


class TestFindApAxis:
    """Tests for NibabelFallbackBackend._find_ap_axis()."""

    def test_standard_axial(self, make_dicom_file, tmp_path):
        """
        Standard axial orientation: ImageOrientationPatient = [1,0,0, 0,1,0].
        The A/P direction is along Y = col_cos = dim 1 (rows in volume).
        """
        backend = NibabelFallbackBackend()
        p = make_dicom_file(
            str(tmp_path),
            filename="axial.dcm",
            instance_number=1,
        )
        ds = pydicom.dcmread(p, stop_before_pixels=True)
        shape = (25, 64, 64)  # (slices, rows, cols)

        ap_dim, face_at_high = backend._find_ap_axis(ds, shape)

        # With [1,0,0, 0,1,0]: col_cos = [0,1,0], dot with [0,-1,0] = -1.0
        # So A/P axis = dim 1 (rows), face at low end (dot < 0, anterior in LPS)
        assert ap_dim == 1
        assert face_at_high == False  # noqa: E712 — np.bool_ vs Python bool

    def test_fallback_without_iop(self, tmp_path):
        """
        When ImageOrientationPatient is missing, the backend falls back
        to standard axial orientation and does not raise.
        """
        backend = NibabelFallbackBackend()

        # Create a minimal dataset without ImageOrientationPatient
        ds = pydicom.Dataset()
        shape = (25, 64, 64)

        # Should not raise; fallback uses [1,0,0, 0,1,0, 0,0,1]
        ap_dim, face_at_high = backend._find_ap_axis(ds, shape)
        assert isinstance(ap_dim, (int, np.integer))
        assert bool(face_at_high) in (True, False)

    def test_coronal_orientation(self, make_dicom_file, tmp_path):
        """
        Coronal orientation: ImageOrientationPatient = [1,0,0, 0,0,-1].
        row_cos = [1,0,0], col_cos = [0,0,-1], normal = cross = [0,1,0].
        A/P is normal (dim 0 = slices direction).
        """
        backend = NibabelFallbackBackend()

        ds = pydicom.Dataset()
        ds.ImageOrientationPatient = [1.0, 0.0, 0.0, 0.0, 0.0, -1.0]
        shape = (25, 64, 64)

        ap_dim, face_at_high = backend._find_ap_axis(ds, shape)
        # normal = cross([1,0,0], [0,0,-1]) = [0,1,0]
        # dot([0,1,0], [0,-1,0]) = -1.0 → dim 0, face_at_high = False (anterior in LPS)
        assert ap_dim == 0
        assert face_at_high == False  # noqa: E712 — np.bool_ vs Python bool
