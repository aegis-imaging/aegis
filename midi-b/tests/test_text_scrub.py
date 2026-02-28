"""Tests for text scrubbing."""

from midi_b.deid.text_scrub import ScrubContext, scrub_free_text


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
