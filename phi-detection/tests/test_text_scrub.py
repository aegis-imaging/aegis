"""Tests for the text PHI scrubbing backends."""

import pytest

from app.backends.text_scrub import (
    RegexTextScrubBackend,
    select_text_scrub_backend,
    _build_user_prompt,
    _parse_llm_response,
)


class TestRegexTextScrubBackend:
    """Tests for the regex fallback scrubber."""

    def setup_method(self):
        self.backend = RegexTextScrubBackend()

    def test_name(self):
        assert self.backend.name == "regex"

    def test_always_available(self):
        assert self.backend.available() is True

    def test_empty_text(self):
        result = self.backend.scrub("")
        assert result == {"scrubbed": "", "phi_found": False}

    def test_no_phi(self):
        result = self.backend.scrub("CT CHEST WITH CONTRAST")
        assert result["phi_found"] is False
        assert result["scrubbed"] == "CT CHEST WITH CONTRAST"

    def test_ssn_pattern(self):
        result = self.backend.scrub("SSN 123-45-6789 noted")
        assert result["phi_found"] is True
        assert "123-45-6789" not in result["scrubbed"]
        assert "[REMOVED]" in result["scrubbed"]

    def test_email_pattern(self):
        result = self.backend.scrub("Contact john@hospital.com for info")
        assert result["phi_found"] is True
        assert "john@hospital.com" not in result["scrubbed"]

    def test_mrn_pattern(self):
        result = self.backend.scrub("MRN: 12345 study")
        assert result["phi_found"] is True
        assert "12345" not in result["scrubbed"]
        assert "study" in result["scrubbed"]

    def test_date_pattern(self):
        result = self.backend.scrub("Exam on 01/15/2024 normal")
        assert result["phi_found"] is True
        assert "01/15/2024" not in result["scrubbed"]

    def test_title_name_pattern(self):
        result = self.backend.scrub("Brain MRI, Dr. Smith ordering")
        assert result["phi_found"] is True
        assert "[REMOVED]" in result["scrubbed"]
        assert "Brain MRI," in result["scrubbed"]

    def test_ip_address(self):
        result = self.backend.scrub("From 192.168.1.100")
        assert result["phi_found"] is True
        assert "192.168.1.100" not in result["scrubbed"]

    def test_url_pattern(self):
        result = self.backend.scrub("See https://example.com/patient/123")
        assert result["phi_found"] is True
        assert "https://example.com" not in result["scrubbed"]

    def test_context_patient_name(self):
        context = {"patient_name": "DOE^JOHN"}
        result = self.backend.scrub("CT for Doe reviewed", context)
        assert result["phi_found"] is True
        assert "doe" not in result["scrubbed"].lower()

    def test_context_patient_id(self):
        context = {"patient_id": "MRN-12345"}
        result = self.backend.scrub("Patient MRN-12345 CT", context)
        assert result["phi_found"] is True
        assert "MRN-12345" not in result["scrubbed"]

    def test_context_institution(self):
        context = {"institution_name": "General Hospital"}
        result = self.backend.scrub("At General Hospital radiology", context)
        assert result["phi_found"] is True
        assert "General Hospital" not in result["scrubbed"]

    def test_consecutive_removed_collapsed(self):
        context = {"patient_name": "DOE^JOHN"}
        result = self.backend.scrub("John Doe scan", context)
        assert result["phi_found"] is True
        assert "[REMOVED] [REMOVED]" not in result["scrubbed"]

    def test_preserves_medical_terms(self):
        result = self.backend.scrub("T1 MPRAGE SAGITTAL BRAIN")
        assert result["phi_found"] is False
        assert result["scrubbed"] == "T1 MPRAGE SAGITTAL BRAIN"


class TestSelectTextScrubBackend:
    """Tests for backend selection."""

    def test_auto_selects_regex_when_no_llm(self):
        """Without LLM SDKs, auto selects regex fallback."""
        backend = select_text_scrub_backend("auto")
        # Regex is always the last fallback and always available
        assert backend.available()

    def test_regex_explicit_selection(self):
        backend = select_text_scrub_backend("regex")
        assert backend.name == "regex"
        assert backend.available()


class TestBuildUserPrompt:
    """Tests for prompt construction."""

    def test_basic_prompt(self):
        prompt = _build_user_prompt("CT scan report")
        assert "CT scan report" in prompt

    def test_prompt_with_context(self):
        context = {
            "patient_name": "DOE^JOHN",
            "patient_id": "MRN-123",
        }
        prompt = _build_user_prompt("scan report", context)
        assert "DOE^JOHN" in prompt
        assert "MRN-123" in prompt
        assert "Known identifiers" in prompt

    def test_prompt_with_empty_context(self):
        prompt = _build_user_prompt("scan report", {})
        assert "Known identifiers" not in prompt


class TestParseLlmResponse:
    """Tests for LLM response parsing."""

    def test_valid_json(self):
        response = '{"scrubbed": "CT [REMOVED]", "phi_found": true}'
        result = _parse_llm_response(response, "CT John")
        assert result["scrubbed"] == "CT [REMOVED]"
        assert result["phi_found"] is True

    def test_json_in_code_fence(self):
        response = '```json\n{"scrubbed": "test", "phi_found": false}\n```'
        result = _parse_llm_response(response, "test")
        assert result["scrubbed"] == "test"
        assert result["phi_found"] is False

    def test_empty_response_returns_original(self):
        result = _parse_llm_response("", "original text")
        assert result["scrubbed"] == "original text"
        assert result["phi_found"] is False

    def test_invalid_json_returns_original(self):
        result = _parse_llm_response("not json at all", "original text")
        assert result["scrubbed"] == "original text"
        assert result["phi_found"] is False

    def test_missing_fields_uses_defaults(self):
        result = _parse_llm_response("{}", "original")
        assert result["scrubbed"] == "original"
        assert result["phi_found"] is False
