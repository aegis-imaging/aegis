"""Tests for DIMSE receiver compressed transfer syntax support.

Verifies that the SCP accepts JPEG2000, JPEG-LS, RLE, and other
compressed transfer syntaxes in addition to uncompressed formats.
"""

import pytest

from app.scp import ACCEPTED_TRANSFER_SYNTAXES, create_scp

pynetdicom = pytest.importorskip("pynetdicom")


class TestAcceptedTransferSyntaxes:
    """Verify the ACCEPTED_TRANSFER_SYNTAXES list is comprehensive."""

    def test_includes_implicit_vr(self):
        assert "1.2.840.10008.1.2" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_explicit_vr_le(self):
        assert "1.2.840.10008.1.2.1" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_explicit_vr_be(self):
        assert "1.2.840.10008.1.2.2" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg_baseline(self):
        assert "1.2.840.10008.1.2.4.50" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg_lossless(self):
        assert "1.2.840.10008.1.2.4.57" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg_lossless_sv1(self):
        assert "1.2.840.10008.1.2.4.70" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg2000_lossless(self):
        assert "1.2.840.10008.1.2.4.90" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg2000(self):
        assert "1.2.840.10008.1.2.4.91" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg_ls_lossless(self):
        assert "1.2.840.10008.1.2.4.80" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_jpeg_ls_near_lossless(self):
        assert "1.2.840.10008.1.2.4.81" in ACCEPTED_TRANSFER_SYNTAXES

    def test_includes_rle_lossless(self):
        assert "1.2.840.10008.1.2.5" in ACCEPTED_TRANSFER_SYNTAXES

    def test_minimum_syntax_count(self):
        """We should support at least 12 transfer syntaxes."""
        assert len(ACCEPTED_TRANSFER_SYNTAXES) >= 12


class TestCreateSCPTransferSyntaxes:
    """Verify that create_scp() builds presentation contexts with all transfer syntaxes."""

    def test_scp_has_compressed_contexts(self):
        """The SCP should accept JPEG2000 and other compressed syntaxes."""
        ae = create_scp()

        # Find MR Image Storage context
        mr_sop = "1.2.840.10008.5.1.4.1.1.4"
        mr_contexts = [c for c in ae.supported_contexts if str(c.abstract_syntax) == mr_sop]

        assert len(mr_contexts) > 0, "MR Image Storage should have a presentation context"

        ctx = mr_contexts[0]
        ts_uids = [str(ts) for ts in ctx.transfer_syntax]

        # Check that JPEG2000 Lossless is accepted
        assert "1.2.840.10008.1.2.4.90" in ts_uids, "JPEG2000 Lossless should be accepted"
        # Check that RLE Lossless is accepted
        assert "1.2.840.10008.1.2.5" in ts_uids, "RLE Lossless should be accepted"

    def test_scp_still_has_verification(self):
        """Verification (C-ECHO) context should still be present."""
        ae = create_scp()
        verification_sop = "1.2.840.10008.1.1"
        verify_contexts = [c for c in ae.supported_contexts if str(c.abstract_syntax) == verification_sop]
        assert len(verify_contexts) > 0, "Verification SOP should be present"
