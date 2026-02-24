"""Tests for GeminiBackend -- mocks vertexai GenerativeModel."""

from unittest.mock import MagicMock, patch

from app.backends.gemini import GeminiBackend


class TestGeminiAvailable:
    def test_available_when_sdk_importable(self):
        backend = GeminiBackend()
        with patch.object(backend, "available", return_value=True):
            assert backend.available() is True

    def test_not_available_when_sdk_missing(self):
        backend = GeminiBackend()
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False


class TestGeminiParseResponse:
    def test_parses_valid_json_array(self):
        backend = GeminiBackend(confidence_threshold=0.4, min_text_length=3)
        text = '[{"text": "JOHN DOE", "confidence": 0.95}, {"text": "01/01/1990", "confidence": 0.9}]'
        regions = backend._parse_response(text)
        assert len(regions) == 2
        assert regions[0].text == "JOHN DOE"
        assert regions[0].confidence == 0.95

    def test_strips_markdown_code_fences(self):
        backend = GeminiBackend(confidence_threshold=0.4, min_text_length=3)
        text = '```json\n[{"text": "MRN12345", "confidence": 0.88}]\n```'
        regions = backend._parse_response(text)
        assert len(regions) == 1
        assert regions[0].text == "MRN12345"

    def test_empty_array_returns_no_regions(self):
        backend = GeminiBackend()
        assert backend._parse_response("[]") == []

    def test_non_json_response_returns_empty(self):
        backend = GeminiBackend()
        assert backend._parse_response("No PHI detected in this image.") == []

    def test_filters_short_text(self):
        backend = GeminiBackend(confidence_threshold=0.0, min_text_length=5)
        text = '[{"text": "AB", "confidence": 0.9}]'
        regions = backend._parse_response(text)
        assert regions == []

    def test_filters_low_confidence(self):
        backend = GeminiBackend(confidence_threshold=0.7, min_text_length=3)
        text = '[{"text": "PATIENT NAME", "confidence": 0.5}]'
        regions = backend._parse_response(text)
        assert regions == []

    def test_defaults_confidence_to_1_when_missing(self):
        backend = GeminiBackend(confidence_threshold=0.4, min_text_length=3)
        text = '[{"text": "HOSPITAL"}]'
        regions = backend._parse_response(text)
        assert len(regions) == 1
        assert regions[0].confidence == 1.0

    def test_empty_string_returns_empty(self):
        backend = GeminiBackend()
        assert backend._parse_response("") == []


def _make_vertexai_modules():
    """Build fake sys.modules entries for vertexai so detect() can import Part."""
    mock_part_cls = MagicMock()
    mock_part_cls.from_data.return_value = MagicMock()

    mock_gm = MagicMock()
    mock_gm.Part = mock_part_cls

    mock_vertexai = MagicMock()

    return {
        "vertexai": mock_vertexai,
        "vertexai.generative_models": mock_gm,
    }, mock_part_cls


class TestGeminiDetect:
    def test_detect_calls_model_and_returns_findings(self, make_dicom_file):
        path = make_dicom_file(filename="gemini_test.dcm", pixel_value=100)

        mock_model = MagicMock()
        mock_response = MagicMock()
        mock_response.text = '[{"text": "PATIENT NAME", "confidence": 0.95}]'
        mock_model.generate_content.return_value = mock_response

        backend = GeminiBackend(confidence_threshold=0.4, min_text_length=3)
        backend._model = mock_model

        mods, _ = _make_vertexai_modules()
        with patch.dict("sys.modules", mods):
            findings = backend.detect([path])

        assert len(findings) == 1
        assert findings[0].file == "gemini_test.dcm"
        assert findings[0].regions[0].text == "PATIENT NAME"

    def test_detect_returns_empty_when_model_returns_empty_array(self, make_dicom_file):
        path = make_dicom_file(filename="gemini_clean.dcm", pixel_value=100)

        mock_model = MagicMock()
        mock_response = MagicMock()
        mock_response.text = "[]"
        mock_model.generate_content.return_value = mock_response

        backend = GeminiBackend()
        backend._model = mock_model

        mods, _ = _make_vertexai_modules()
        with patch.dict("sys.modules", mods):
            findings = backend.detect([path])

        assert findings == []

    def test_detect_skips_file_on_exception(self, make_dicom_file):
        path = make_dicom_file(filename="gemini_err.dcm", pixel_value=50)

        mock_model = MagicMock()
        mock_model.generate_content.side_effect = Exception("API error")

        backend = GeminiBackend()
        backend._model = mock_model

        mods, _ = _make_vertexai_modules()
        with patch.dict("sys.modules", mods):
            findings = backend.detect([path])

        assert findings == []

    def test_detect_skips_file_without_pixel_data(self, make_dicom_file):
        path = make_dicom_file(filename="no_pixels_gemini.dcm", include_pixel_data=False)

        mock_model = MagicMock()
        backend = GeminiBackend()
        backend._model = mock_model

        mods, _ = _make_vertexai_modules()
        with patch.dict("sys.modules", mods):
            findings = backend.detect([path])

        assert findings == []
        mock_model.generate_content.assert_not_called()
