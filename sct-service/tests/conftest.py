"""Shared test fixtures for SCT service tests."""

import os
import tempfile

import pytest


@pytest.fixture
def bids_dir():
    """Create a minimal BIDS directory with a spine T2w NIfTI."""
    with tempfile.TemporaryDirectory() as d:
        anat_dir = os.path.join(d, "sub-test01", "anat")
        os.makedirs(anat_dir)
        nifti = os.path.join(anat_dir, "sub-test01_T2w.nii.gz")
        with open(nifti, "wb") as f:
            f.write(b"\x1f\x8b" + b"\x00" * 100)  # minimal gzip header
        yield d


@pytest.fixture
def spine_bids_dir():
    """Create a BIDS directory with 'spine' in the filename."""
    with tempfile.TemporaryDirectory() as d:
        anat_dir = os.path.join(d, "sub-test01", "anat")
        os.makedirs(anat_dir)
        nifti = os.path.join(anat_dir, "sub-test01_spine_T2w.nii.gz")
        with open(nifti, "wb") as f:
            f.write(b"\x1f\x8b" + b"\x00" * 100)
        yield d


@pytest.fixture
def dwi_bids_dir():
    """Create a BIDS directory with DWI data (dwi NIfTI + bval + bvec)."""
    with tempfile.TemporaryDirectory() as d:
        dwi_dir = os.path.join(d, "sub-test01", "dwi")
        os.makedirs(dwi_dir)

        nifti = os.path.join(dwi_dir, "sub-test01_dwi.nii.gz")
        with open(nifti, "wb") as f:
            f.write(b"\x1f\x8b" + b"\x00" * 100)

        bval = os.path.join(dwi_dir, "sub-test01_dwi.bval")
        with open(bval, "w") as f:
            f.write("0 1000 1000\n")

        bvec = os.path.join(dwi_dir, "sub-test01_dwi.bvec")
        with open(bvec, "w") as f:
            f.write("1 0 0\n0 1 0\n0 0 1\n")

        # Also include a T2w for the main pipeline
        anat_dir = os.path.join(d, "sub-test01", "anat")
        os.makedirs(anat_dir)
        t2w = os.path.join(anat_dir, "sub-test01_T2w.nii.gz")
        with open(t2w, "wb") as f:
            f.write(b"\x1f\x8b" + b"\x00" * 100)

        yield d
