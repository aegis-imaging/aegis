"""Tests for GeminiClassificationBackend -- mocks google-genai Client."""

from unittest.mock import MagicMock, patch

from app.backends.gemini import GeminiClassificationBackend
from app.backends.base import ClassificationResult


class TestGeminiParseResponse:
    def test_parses_valid_json_object(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult()
        text = '{"modality": "MR", "body_part": "HEAD"}'
        modality, body_part = backend._parse_response(text, heuristic)
        assert modality == "MR"
        assert body_part == "HEAD"

    def test_strips_markdown_code_fences(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult()
        text = '```json\n{"modality": "CT", "body_part": "CHEST"}\n```'
        modality, body_part = backend._parse_response(text, heuristic)
        assert modality == "CT"
        assert body_part == "CHEST"

    def test_empty_string_returns_empty(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult()
        modality, body_part = backend._parse_response("", heuristic)
        assert modality == ""
        assert body_part == ""

    def test_non_json_returns_empty(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult()
        modality, body_part = backend._parse_response("This is a brain MRI.", heuristic)
        assert modality == ""
        assert body_part == ""

    def test_invalid_modality_falls_back_to_heuristic(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult(modality="MR", body_part="", confidence=0.3)
        text = '{"modality": "UNKNOWN", "body_part": "HEAD"}'
        modality, body_part = backend._parse_response(text, heuristic)
        assert modality == "MR"  # fell back to heuristic
        assert body_part == "HEAD"

    def test_invalid_body_part_falls_back_to_heuristic(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult(modality="", body_part="CHEST", confidence=0.3)
        text = '{"modality": "CT", "body_part": "FOOT"}'
        modality, body_part = backend._parse_response(text, heuristic)
        assert modality == "CT"
        assert body_part == "CHEST"  # fell back to heuristic

    def test_empty_fields_fall_back(self):
        backend = GeminiClassificationBackend()
        heuristic = ClassificationResult(modality="PT", body_part="HEAD", confidence=0.4)
        text = '{"modality": "", "body_part": ""}'
        modality, body_part = backend._parse_response(text, heuristic)
        assert modality == "PT"
        assert body_part == "HEAD"


def _make_genai_modules():
    """Build fake sys.modules entries for google.genai."""
    mock_types = MagicMock()
    mock_part_cls = MagicMock()
    mock_part_cls.from_bytes.return_value = MagicMock()
    mock_types.Part = mock_part_cls

    mock_genai = MagicMock()

    return {
        "google": MagicMock(),
        "google.genai": mock_genai,
        "google.genai.types": mock_types,
    }, mock_types


class TestGeminiClassificationBackend:
    def test_high_confidence_heuristic_skips_api(self, dicom_study):
        """When heuristic returns confidence >= threshold, Gemini API is never called."""
        backend = GeminiClassificationBackend(confidence_threshold=0.5)

        mock_client = MagicMock()
        backend._client = mock_client

        # dicom_study fixture creates MR HEAD files with direct tags -> confidence=0.95
        result = backend.classify(dicom_study)

        assert result.modality == "MR"
        assert result.body_part == "HEAD"
        assert result.confidence == 0.95
        mock_client.models.generate_content.assert_not_called()

    def test_low_confidence_triggers_gemini(self, tmp_path, make_dicom_file):
        """When heuristic confidence < threshold, Gemini API is called."""
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="1.2.840.10008.5.1.4.1.1.88.11",  # unknown SOP
            series_description="",
            study_description="",
            protocol_name="",
        )

        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.text = '{"modality": "MR", "body_part": "HEAD"}'
        mock_client.models.generate_content.return_value = mock_response

        backend = GeminiClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        mods, _ = _make_genai_modules()
        with patch.dict("sys.modules", mods):
            result = backend.classify([path])

        mock_client.models.generate_content.assert_called_once()
        assert result.modality == "MR"
        assert result.body_part == "HEAD"
        assert result.method == "gemini_multimodal"
        assert result.confidence == 0.75

    def test_api_error_falls_back_to_heuristic(self, tmp_path, make_dicom_file):
        """When Gemini API raises an exception, the heuristic result is returned."""
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="1.2.840.10008.5.1.4.1.1.4",  # MR
            series_description="",
            study_description="",
            protocol_name="",
        )

        mock_client = MagicMock()
        mock_client.models.generate_content.side_effect = RuntimeError("API error")

        backend = GeminiClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        mods, _ = _make_genai_modules()
        with patch.dict("sys.modules", mods):
            result = backend.classify([path])

        # Heuristic should find modality=MR from SOP UID -> confidence=0.60
        assert result.modality == "MR"
        assert result.method != "gemini_multimodal"

    def test_available_returns_false_without_sdk(self):
        backend = GeminiClassificationBackend()
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False

    def test_gemini_returns_empty_falls_back(self, tmp_path, make_dicom_file):
        """When Gemini returns empty modality and body_part, heuristic is used."""
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="1.2.840.10008.5.1.4.1.1.4",  # MR SOP
            series_description="",
            study_description="",
            protocol_name="",
        )

        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.text = '{"modality": "", "body_part": ""}'
        mock_client.models.generate_content.return_value = mock_response

        backend = GeminiClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        mods, _ = _make_genai_modules()
        with patch.dict("sys.modules", mods):
            result = backend.classify([path])

        # Heuristic found MR from SOP UID with confidence 0.60
        assert result.modality == "MR"
        assert result.method != "gemini_multimodal"
