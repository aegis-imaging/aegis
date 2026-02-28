"""Tests for private tag PHI scanning."""

import os
import tempfile

import numpy as np
import pydicom
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import ExplicitVRLittleEndian, generate_uid
import pytest

from app.backends.tag_phi import scan_private_tags, TagPHIResult


def _make_dicom_with_private_tags(
    tmp_dir: str,
    filename: str = "test.dcm",
    private_tags: dict | None = None,
) -> str:
    """Create a minimal DICOM file with specified private tags."""
    filepath = os.path.join(tmp_dir, filename)

    file_meta = pydicom.dataset.FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.2"
    file_meta.MediaStorageSOPInstanceUID = generate_uid()
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)
    ds.SOPClassUID = file_meta.MediaStorageSOPClassUID
    ds.SOPInstanceUID = file_meta.MediaStorageSOPInstanceUID
    ds.StudyInstanceUID = generate_uid()
    ds.SeriesInstanceUID = generate_uid()
    ds.Modality = "MR"
    ds.Rows = 16
    ds.Columns = 16
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME2"
    ds.PixelData = np.zeros((16, 16), dtype=np.uint16).tobytes()

    # Add private block creator
    if private_tags:
        for (group, element), value in private_tags.items():
            ds.add_new((group, element), "LO", value)

    ds.save_as(filepath)
    return filepath


class TestScanPrivateTags:
    """Tests for the scan_private_tags function."""

    def test_no_private_tags(self, tmp_dir):
        """DICOM with no private tags returns no findings."""
        path = _make_dicom_with_private_tags(tmp_dir)
        result = scan_private_tags([path])
        assert result.files_scanned == 1
        assert result.findings == []

    def test_detects_name_caret_pattern(self, tmp_dir):
        """Detect DICOM PN-format name (Last^First) in private tag."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "Smith^John"},
        )
        result = scan_private_tags([path])
        assert len(result.findings) == 1
        assert result.findings[0].pattern == "name_caret"
        assert "Smith^John" in result.findings[0].value_preview

    def test_detects_ssn_pattern(self, tmp_dir):
        """Detect SSN pattern in private tag."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "123-45-6789"},
        )
        result = scan_private_tags([path])
        assert len(result.findings) == 1
        assert result.findings[0].pattern == "ssn"

    def test_detects_mrn_pattern(self, tmp_dir):
        """Detect MRN pattern in private tag."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "MRN: 12345678"},
        )
        result = scan_private_tags([path])
        assert len(result.findings) == 1
        assert result.findings[0].pattern == "mrn"

    def test_detects_accession_pattern(self, tmp_dir):
        """Detect accession number pattern in private tag."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "ACC#12345678"},
        )
        result = scan_private_tags([path])
        assert len(result.findings) == 1
        assert result.findings[0].pattern == "accession"

    def test_detects_phone_pattern(self, tmp_dir):
        """Detect phone number pattern in private tag."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "555-123-4567"},
        )
        result = scan_private_tags([path])
        assert len(result.findings) == 1
        assert result.findings[0].pattern == "phone"

    def test_ignores_safe_acquisition_params(self, tmp_dir):
        """Vendor acquisition parameters should not trigger findings."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0019, 0x100A): "CSA Header Info"},
        )
        result = scan_private_tags([path])
        assert result.findings == []

    def test_ignores_short_values(self, tmp_dir):
        """Values shorter than 3 chars are skipped."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "AB"},
        )
        result = scan_private_tags([path])
        assert result.findings == []

    def test_multiple_files(self, tmp_dir):
        """Scan multiple files and aggregate findings."""
        path1 = _make_dicom_with_private_tags(
            tmp_dir,
            filename="file1.dcm",
            private_tags={(0x0009, 0x0010): "Smith^John"},
        )
        path2 = _make_dicom_with_private_tags(
            tmp_dir,
            filename="file2.dcm",
            private_tags={(0x0009, 0x0010): "12345"},  # no pattern match
        )
        result = scan_private_tags([path1, path2])
        assert result.files_scanned == 2
        assert len(result.findings) == 1

    def test_missing_file_skipped(self, tmp_dir):
        """Non-existent paths are silently skipped."""
        result = scan_private_tags(["/nonexistent/file.dcm"])
        assert result.files_scanned == 0
        assert result.findings == []

    def test_dob_context_only(self, tmp_dir):
        """Dates are only flagged when DOB context is present."""
        # Plain date — NOT flagged
        path1 = _make_dicom_with_private_tags(
            tmp_dir,
            filename="plain_date.dcm",
            private_tags={(0x0009, 0x0010): "20240115"},
        )
        # Date with DOB context — flagged
        path2 = _make_dicom_with_private_tags(
            tmp_dir,
            filename="dob_date.dcm",
            private_tags={(0x0009, 0x0010): "Patient DOB: 1990-01-15"},
        )
        result = scan_private_tags([path1, path2])
        assert len(result.findings) == 1
        assert result.findings[0].pattern == "date_of_birth"
        assert result.findings[0].file == "dob_date.dcm"

    def test_tag_format_in_findings(self, tmp_dir):
        """Tag string is formatted as (group,element)."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "Smith^Jane"},
        )
        result = scan_private_tags([path])
        assert result.findings[0].tag == "(0009,0010)"

    def test_value_preview_truncated(self, tmp_dir):
        """Long values are truncated to 80 chars in preview."""
        long_val = "Smith^" + "A" * 200
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): long_val},
        )
        result = scan_private_tags([path])
        assert len(result.findings) == 1
        assert len(result.findings[0].value_preview) == 80

    def test_private_tags_scanned_count(self, tmp_dir):
        """Counts total private tags scanned across files."""
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={
                (0x0009, 0x0010): "some data",
                (0x0009, 0x0011): "more data",
            },
        )
        result = scan_private_tags([path])
        assert result.private_tags_scanned >= 2


class TestDetectTagsEndpoint:
    """Tests for the POST /detect-tags FastAPI endpoint."""

    @pytest.fixture()
    def client(self):
        from fastapi.testclient import TestClient
        from app.main import app
        return TestClient(app)

    def test_detect_tags_empty_paths(self, client):
        resp = client.post("/detect-tags", json={
            "study_uid": "1.2.3",
            "input_paths": [],
        })
        assert resp.status_code == 400

    def test_detect_tags_missing_files(self, client):
        resp = client.post("/detect-tags", json={
            "study_uid": "1.2.3",
            "input_paths": ["/no/such/file.dcm"],
        })
        assert resp.status_code == 400

    def test_detect_tags_clean(self, client, tmp_dir):
        path = _make_dicom_with_private_tags(tmp_dir)
        resp = client.post("/detect-tags", json={
            "study_uid": "1.2.3",
            "input_paths": [path],
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "complete"
        assert data["phi_detected"] is False
        assert data["findings"] == []

    def test_detect_tags_with_phi(self, client, tmp_dir):
        path = _make_dicom_with_private_tags(
            tmp_dir,
            private_tags={(0x0009, 0x0010): "Doe^Jane"},
        )
        resp = client.post("/detect-tags", json={
            "study_uid": "1.2.3",
            "input_paths": [path],
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "complete"
        assert data["phi_detected"] is True
        assert len(data["findings"]) == 1
        assert data["findings"][0]["pattern"] == "name_caret"
