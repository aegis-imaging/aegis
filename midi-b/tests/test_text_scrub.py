"""Tests for text scrubbing."""

import json
from unittest.mock import MagicMock, patch

from midi_b.deid.text_scrub import RemoteTextScrubBackend, ScrubContext, scrub_free_text


def test_no_phi():
    result = scrub_free_text("CT CHEST WITH CONTRAST")
    assert result.phi_found is False
    assert result.text == "CT CHEST WITH CONTRAST"


def test_empty_string():
    result = scrub_free_text("")
    assert result.phi_found is False
    assert result.text == ""


def test_scrub_patient_name_tokens():
    ctx = ScrubContext(patient_name="DOE^JOHN")
    result = scrub_free_text("CT for John Doe with contrast", ctx)
    assert result.phi_found is True
    assert "[REMOVED]" in result.text
    assert "John" not in result.text
    assert "Doe" not in result.text
    assert "CT for" in result.text


def test_scrub_patient_id():
    ctx = ScrubContext(patient_id="MRN-12345")
    result = scrub_free_text("Study for MRN-12345", ctx)
    assert result.phi_found is True
    assert "MRN-12345" not in result.text
    assert "[REMOVED]" in result.text


def test_scrub_referring_physician():
    ctx = ScrubContext(referring_physician="SMITH^ALICE")
    result = scrub_free_text("Ordered by Dr. Smith Alice for imaging", ctx)
    assert result.phi_found is True
    assert "Smith" not in result.text
    assert "Alice" not in result.text


def test_scrub_institution():
    ctx = ScrubContext(institution_name="General Hospital")
    result = scrub_free_text("From General Hospital", ctx)
    assert result.phi_found is True
    assert "General Hospital" not in result.text


def test_ssn_pattern():
    result = scrub_free_text("SSN: 123-45-6789")
    assert result.phi_found is True
    assert "123-45-6789" not in result.text


def test_phone_pattern():
    result = scrub_free_text("Call (555) 123-4567")
    assert result.phi_found is True
    assert "555" not in result.text


def test_email_pattern():
    result = scrub_free_text("Contact patient@email.com")
    assert result.phi_found is True
    assert "patient@email.com" not in result.text


def test_mrn_pattern():
    result = scrub_free_text("MRN: 99999")
    assert result.phi_found is True
    assert "99999" not in result.text


def test_date_us_pattern():
    result = scrub_free_text("DOB: 01/15/1980")
    assert result.phi_found is True
    assert "01/15/1980" not in result.text


def test_medical_terms_not_flagged():
    ctx = ScrubContext(patient_name="HEAD^BRAIN")
    result = scrub_free_text("CT HEAD BRAIN axial sequence", ctx)
    # HEAD and BRAIN are medical terms — should NOT be removed
    assert result.phi_found is False


def test_collapse_consecutive_removed():
    ctx = ScrubContext(patient_name="DOE^JOHN", patient_id="12345")
    result = scrub_free_text("John Doe 12345", ctx)
    assert result.phi_found is True
    # Consecutive [REMOVED] tokens should be collapsed
    assert result.text.count("[REMOVED]") >= 1


def test_preserves_non_phi_text():
    ctx = ScrubContext(patient_name="DOE^JOHN")
    result = scrub_free_text("CT for John Doe with contrast enhancement", ctx)
    assert "CT for" in result.text
    assert "contrast enhancement" in result.text


# ---------------------------------------------------------------------------
# RemoteTextScrubBackend tests
# ---------------------------------------------------------------------------


def test_remote_backend_success():
    """Remote backend returns scrubbed text from the service."""
    backend = RemoteTextScrubBackend("http://localhost:8082")

    response_body = json.dumps({
        "status": "complete",
        "results": [{"original": "John Doe", "scrubbed": "[REMOVED]", "phi_found": True}],
        "tool_used": "gemini",
    }).encode()

    mock_resp = MagicMock()
    mock_resp.read.return_value = response_body
    mock_resp.__enter__ = lambda s: s
    mock_resp.__exit__ = MagicMock(return_value=False)

    with patch("urllib.request.urlopen", return_value=mock_resp):
        result = backend.scrub("John Doe")

    assert result.phi_found is True
    assert result.text == "[REMOVED]"


def test_remote_backend_batch():
    """Remote backend sends batch request."""
    backend = RemoteTextScrubBackend("http://localhost:8082")

    response_body = json.dumps({
        "status": "complete",
        "results": [
            {"original": "John Doe", "scrubbed": "[REMOVED]", "phi_found": True},
            {"original": "Normal finding", "scrubbed": "Normal finding", "phi_found": False},
        ],
        "tool_used": "regex",
    }).encode()

    mock_resp = MagicMock()
    mock_resp.read.return_value = response_body
    mock_resp.__enter__ = lambda s: s
    mock_resp.__exit__ = MagicMock(return_value=False)

    with patch("urllib.request.urlopen", return_value=mock_resp):
        results = backend.scrub_batch(["John Doe", "Normal finding"])

    assert len(results) == 2
    assert results[0].phi_found is True
    assert results[1].phi_found is False


def test_remote_backend_fallback_on_error():
    """Remote backend falls back to local regex on HTTP failure."""
    backend = RemoteTextScrubBackend("http://localhost:9999")

    with patch("urllib.request.urlopen", side_effect=Exception("connection refused")):
        ctx = ScrubContext(patient_name="DOE^JOHN")
        result = backend.scrub("CT for John Doe", ctx)

    # Should fall back to local regex and still find PHI
    assert result.phi_found is True
    assert "John" not in result.text


def test_remote_backend_empty_input():
    """Empty string returns immediately without HTTP call."""
    backend = RemoteTextScrubBackend("http://localhost:8082")
    result = backend.scrub("")
    assert result.phi_found is False
    assert result.text == ""


def test_remote_backend_batch_empty():
    """Empty batch returns immediately."""
    backend = RemoteTextScrubBackend("http://localhost:8082")
    results = backend.scrub_batch([])
    assert results == []
