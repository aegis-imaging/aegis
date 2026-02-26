"""Tests for the Azure Computer Vision PHI detection backend."""

from unittest.mock import MagicMock, patch
import pytest

from app.backends.azure_vision import AzureVisionBackend


# ---------------------------------------------------------------------------
# Availability
# ---------------------------------------------------------------------------

class TestAzureVisionAvailable:
    def test_available_when_sdk_importable_and_endpoint_set(self):
        backend = AzureVisionBackend()
        backend._endpoint = "https://test.cognitiveservices.azure.com"
        with patch.object(backend, "available", return_value=True):
            assert backend.available() is True

    def test_not_available_when_sdk_missing(self):
        backend = AzureVisionBackend()
        backend._endpoint = "https://test.cognitiveservices.azure.com"
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False

    def test_not_available_when_endpoint_empty(self):
        backend = AzureVisionBackend()
        backend._endpoint = ""
        assert backend.available() is False


# ---------------------------------------------------------------------------
# Parse Read result
# ---------------------------------------------------------------------------

def _make_word(text, confidence=0.95, polygon=None):
    """Create a mock word object matching Azure Vision SDK structure."""
    word = MagicMock()
    word.text = text
    word.confidence = confidence
    if polygon is not None:
        word.bounding_polygon = polygon
    else:
        # Default: 4-point polygon
        pts = []
        for x, y in [(10, 10), (50, 10), (50, 30), (10, 30)]:
            pt = MagicMock()
            pt.x = x
            pt.y = y
            pts.append(pt)
        word.bounding_polygon = pts
    return word


def _make_read_result(words):
    """Wrap words into the Azure Vision Read result structure: result.read.blocks[].lines[].words[]."""
    line = MagicMock()
    line.words = words
    block = MagicMock()
    block.lines = [line]
    result = MagicMock()
    result.read.blocks = [block]
    return result


class TestAzureVisionParseReadResult:
    def test_words_converted_to_regions(self):
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        words = [_make_word("Patient", 0.95), _make_word("Name", 0.90)]
        result = _make_read_result(words)

        regions = backend._parse_read_result(result)
        assert len(regions) == 2
        assert regions[0].text == "Patient"
        assert regions[0].confidence == 0.95
        assert regions[1].text == "Name"

    def test_short_text_filtered(self):
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        words = [_make_word("OK", 0.99), _make_word("Patient", 0.95)]
        result = _make_read_result(words)

        regions = backend._parse_read_result(result)
        assert len(regions) == 1
        assert regions[0].text == "Patient"

    def test_low_confidence_filtered(self):
        backend = AzureVisionBackend(confidence_threshold=0.5, min_text_length=3)
        words = [_make_word("Noise", 0.3), _make_word("Patient", 0.95)]
        result = _make_read_result(words)

        regions = backend._parse_read_result(result)
        assert len(regions) == 1
        assert regions[0].text == "Patient"

    def test_empty_read_result(self):
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        result = MagicMock()
        result.read = None

        regions = backend._parse_read_result(result)
        assert regions == []

    def test_bbox_from_polygon(self):
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        pts = []
        for x, y in [(100, 200), (300, 200), (300, 250), (100, 250)]:
            pt = MagicMock()
            pt.x = x
            pt.y = y
            pts.append(pt)
        words = [_make_word("Patient", 0.95, polygon=pts)]
        result = _make_read_result(words)

        regions = backend._parse_read_result(result)
        assert regions[0].bbox == [100, 200, 200, 50]

    def test_missing_polygon_falls_back_to_zero_bbox(self):
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        words = [_make_word("Patient", 0.95, polygon=[])]
        result = _make_read_result(words)

        regions = backend._parse_read_result(result)
        assert regions[0].bbox == [0, 0, 0, 0]


# ---------------------------------------------------------------------------
# Detect integration
# ---------------------------------------------------------------------------

class TestAzureVisionDetect:
    def test_detect_calls_api_and_returns_findings(self, make_dicom_file):
        path = make_dicom_file()
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        backend._endpoint = "https://test.cognitiveservices.azure.com"

        mock_client = MagicMock()
        words = [_make_word("DOE^JOHN", 0.92)]
        mock_client.analyze.return_value = _make_read_result(words)
        backend._client = mock_client

        findings = backend.detect([path])
        assert len(findings) == 1
        assert findings[0].regions[0].text == "DOE^JOHN"
        mock_client.analyze.assert_called_once()

    def test_detect_skips_file_without_pixel_data(self, make_dicom_file):
        path = make_dicom_file(include_pixel_data=False)
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        backend._endpoint = "https://test.cognitiveservices.azure.com"

        mock_client = MagicMock()
        backend._client = mock_client

        findings = backend.detect([path])
        assert len(findings) == 0
        mock_client.analyze.assert_not_called()

    def test_detect_continues_on_api_exception(self, make_dicom_file):
        path1 = make_dicom_file(filename="a.dcm")
        path2 = make_dicom_file(filename="b.dcm")
        backend = AzureVisionBackend(confidence_threshold=0.4, min_text_length=3)
        backend._endpoint = "https://test.cognitiveservices.azure.com"

        mock_client = MagicMock()
        words = [_make_word("PHI_TEXT", 0.95)]
        mock_client.analyze.side_effect = [
            RuntimeError("API timeout"),
            _make_read_result(words),
        ]
        backend._client = mock_client

        findings = backend.detect([path1, path2])
        assert len(findings) == 1
        assert findings[0].regions[0].text == "PHI_TEXT"

    def test_name(self):
        backend = AzureVisionBackend()
        assert backend.name == "azure_vision"
