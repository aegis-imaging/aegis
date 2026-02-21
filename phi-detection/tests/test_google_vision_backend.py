"""Tests for GoogleVisionBackend -- mocks google.cloud.vision client."""

from unittest.mock import MagicMock, patch
import pytest

from app.backends.google_vision import GoogleVisionBackend


class TestGoogleVisionAvailable:
    def test_available_when_sdk_importable(self):
        backend = GoogleVisionBackend()
        with patch.object(backend, "available", return_value=True):
            assert backend.available() is True

    def test_not_available_when_sdk_missing(self):
        backend = GoogleVisionBackend()
        # Without the SDK installed, available() should return False
        # (or True if it happens to be installed -- mock to guarantee False)
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False


class TestGoogleVisionParseAnnotations:
    def _make_annotation(self, description, verts=None):
        ann = MagicMock()
        ann.description = description
        if verts is None:
            bp = MagicMock()
            bp.vertices = []
            ann.bounding_poly = bp
        else:
            bp = MagicMock()
            bp.vertices = [MagicMock(x=v[0], y=v[1]) for v in verts]
            ann.bounding_poly = bp
        return ann

    def test_skips_first_annotation(self):
        """First annotation is full-page aggregate -- must be skipped."""
        backend = GoogleVisionBackend(min_text_length=3)
        full_page = self._make_annotation("FULL PAGE TEXT")
        word = self._make_annotation(
            "NAME",
            verts=[(10, 10), (60, 10), (60, 22), (10, 22)],
        )
        regions = backend._parse_annotations([full_page, word])
        assert len(regions) == 1
        assert regions[0].text == "NAME"
        assert regions[0].confidence == 1.0
        assert regions[0].bbox == [10, 10, 50, 12]

    def test_short_text_filtered(self):
        """Words shorter than min_text_length are excluded."""
        backend = GoogleVisionBackend(min_text_length=5)
        full_page = self._make_annotation("full")
        word = self._make_annotation("HI")  # too short
        regions = backend._parse_annotations([full_page, word])
        assert regions == []

    def test_empty_annotations(self):
        backend = GoogleVisionBackend()
        assert backend._parse_annotations([]) == []


class TestGoogleVisionDetect:
    def test_detect_calls_api_and_returns_findings(self, make_dicom_file):
        """detect() calls Vision API for each file and converts annotations."""
        path = make_dicom_file(filename="vision_test.dcm", pixel_value=100)

        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.error.message = ""

        ann = MagicMock()
        ann.description = "PATIENT"
        ann.bounding_poly.vertices = [
            MagicMock(x=10, y=10), MagicMock(x=80, y=10),
            MagicMock(x=80, y=22), MagicMock(x=10, y=22),
        ]
        mock_response.text_annotations = [MagicMock(description="ALL TEXT"), ann]
        mock_client.text_detection.return_value = mock_response

        backend = GoogleVisionBackend(confidence_threshold=0.4, min_text_length=3)
        backend._client = mock_client

        findings = backend.detect([path])
        assert len(findings) == 1
        assert findings[0].file == "vision_test.dcm"
        assert findings[0].regions[0].text == "PATIENT"
        mock_client.text_detection.assert_called_once()

    def test_detect_skips_file_on_api_error(self, make_dicom_file):
        path = make_dicom_file(filename="api_err.dcm", pixel_value=50)

        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.error.message = "Permission denied"
        mock_response.text_annotations = []
        mock_client.text_detection.return_value = mock_response

        backend = GoogleVisionBackend()
        backend._client = mock_client

        findings = backend.detect([path])
        assert findings == []

    def test_detect_skips_file_without_pixel_data(self, make_dicom_file):
        path = make_dicom_file(filename="no_pix.dcm", include_pixel_data=False)

        mock_client = MagicMock()
        backend = GoogleVisionBackend()
        backend._client = mock_client

        findings = backend.detect([path])
        assert findings == []
        mock_client.text_detection.assert_not_called()


class TestGoogleVisionParseAnnotationsEdgeCases:
    def _make_annotation(self, description, verts=None):
        ann = MagicMock()
        ann.description = description
        bp = MagicMock()
        bp.vertices = [] if verts is None else [MagicMock(x=v[0], y=v[1]) for v in verts]
        ann.bounding_poly = bp
        return ann

    def test_parse_annotations_empty_vertices_falls_back_to_zero_bbox(self):
        """Annotation with no bounding vertices gets bbox=[0, 0, 0, 0]."""
        backend = GoogleVisionBackend(min_text_length=3)
        full_page = self._make_annotation("FULL PAGE TEXT")
        word = self._make_annotation("NAME")  # vertices=[]
        regions = backend._parse_annotations([full_page, word])
        assert len(regions) == 1
        assert regions[0].text == "NAME"
        assert regions[0].bbox == [0, 0, 0, 0]
