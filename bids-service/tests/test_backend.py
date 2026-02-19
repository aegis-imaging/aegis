"""Unit tests for the dcm2niix BIDS conversion backend.

Tests the pure utility functions (_subject_label, _classify_series) directly.
These are module-level functions in app.backends.dcm2niix and require no
external tools (dcm2niix is NOT needed for these tests).
"""

import pytest

from app.backends.dcm2niix import _subject_label, _classify_series


# ---------------------------------------------------------------------------
# _subject_label tests
# ---------------------------------------------------------------------------

class TestSubjectLabel:
    """Tests for _subject_label (SHA-256 hash, first 8 chars)."""

    def test_subject_label_deterministic(self):
        """Same UID always produces the same label."""
        uid = "1.2.840.113619.2.55.3.12345"
        label1 = _subject_label(uid)
        label2 = _subject_label(uid)
        assert label1 == label2
        assert len(label1) == 8
        # Should be lowercase hex
        assert all(c in "0123456789abcdef" for c in label1)

    def test_subject_label_different_uids(self):
        """Different UIDs produce different labels."""
        label_a = _subject_label("1.2.3.4.5")
        label_b = _subject_label("1.2.3.4.6")
        assert label_a != label_b


# ---------------------------------------------------------------------------
# _classify_series tests
# ---------------------------------------------------------------------------

class TestClassifySeries:
    """Tests for _classify_series (pattern matching against BIDS suffix map)."""

    def test_classify_series_t1w(self):
        """'T1_SAG_MPRAGE' matches the T1w pattern."""
        datatype, suffix = _classify_series("T1_SAG_MPRAGE", "", "MR")
        assert datatype == "anat"
        assert suffix == "T1w"

    def test_classify_series_flair(self):
        """'FLAIR' matches the FLAIR pattern."""
        datatype, suffix = _classify_series("FLAIR", "", "MR")
        assert datatype == "anat"
        assert suffix == "FLAIR"

    def test_classify_series_bold(self):
        """'resting_state_bold' matches the bold/func pattern."""
        datatype, suffix = _classify_series("resting_state_bold", "", "MR")
        assert datatype == "func"
        assert suffix == "bold"

    def test_classify_series_dwi(self):
        """'DTI_b1000' matches the DWI pattern."""
        datatype, suffix = _classify_series("DTI_b1000", "", "MR")
        assert datatype == "dwi"
        assert suffix == "dwi"

    def test_classify_series_ct_fallback(self):
        """No pattern match + CT modality falls back to ('ct', 'CT')."""
        datatype, suffix = _classify_series("", "axial_scan", "CT")
        assert datatype == "ct"
        assert suffix == "CT"

    def test_classify_series_unknown(self):
        """No pattern match + unknown modality falls back to ('anat', 'unknown')."""
        datatype, suffix = _classify_series("", "some_random_sequence", "XX")
        assert datatype == "anat"
        assert suffix == "unknown"

    def test_classify_series_asl(self):
        """'pcasl' matches the perfusion/ASL pattern."""
        datatype, suffix = _classify_series("pcasl", "", "MR")
        assert datatype == "perf"
        assert suffix == "asl"

    def test_classify_series_t2w_from_description(self):
        """SeriesDescription 'T2_TSE_AX' also matches T2w."""
        datatype, suffix = _classify_series("", "T2_TSE_AX", "MR")
        assert datatype == "anat"
        assert suffix == "T2w"

    def test_classify_series_pet_from_protocol(self):
        """Protocol containing 'FDG' matches PET."""
        datatype, suffix = _classify_series("FDG_BRAIN", "", "PT")
        assert datatype == "pet"
        assert suffix == "pet"
