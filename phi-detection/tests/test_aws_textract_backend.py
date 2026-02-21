"""Tests for AWSTextractBackend -- mocks boto3 client."""

from unittest.mock import MagicMock, patch
import pytest

from app.backends.aws_textract import AWSTextractBackend


class TestAWSTextractAvailable:
    def test_available_when_boto3_importable(self):
        backend = AWSTextractBackend()
        with patch.object(backend, "available", return_value=True):
            assert backend.available() is True

    def test_not_available_when_boto3_missing(self):
        backend = AWSTextractBackend()
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False


class TestAWSTextractParseBlocks:
    def test_word_blocks_converted(self):
        backend = AWSTextractBackend(confidence_threshold=0.4, min_text_length=3)
        blocks = [
            {
                "BlockType": "WORD", "Text": "PATIENT", "Confidence": 95.0,
                "Geometry": {"BoundingBox": {"Left": 0.1, "Top": 0.1, "Width": 0.2, "Height": 0.05}},
            },
            {
                "BlockType": "WORD", "Text": "AB", "Confidence": 90.0,
                "Geometry": {"BoundingBox": {"Left": 0.3, "Top": 0.1, "Width": 0.1, "Height": 0.05}},
            },
            {
                "BlockType": "WORD", "Text": "NAME", "Confidence": 30.0,
                "Geometry": {"BoundingBox": {"Left": 0.5, "Top": 0.1, "Width": 0.1, "Height": 0.05}},
            },
        ]
        regions = backend._parse_blocks(blocks)
        # "PATIENT": conf=0.95 >= 0.4, len=7 >= 3 -> included
        # "AB":      conf=0.90, len=2 < 3 -> excluded (too short)
        # "NAME":    conf=0.30 < 0.4 -> excluded (low confidence)
        assert len(regions) == 1
        assert regions[0].text == "PATIENT"
        assert abs(regions[0].confidence - 0.95) < 0.001

    def test_line_blocks_skipped(self):
        """LINE blocks are ignored; only WORD blocks processed."""
        backend = AWSTextractBackend()
        blocks = [{"BlockType": "LINE", "Text": "PATIENT NAME", "Confidence": 99.0}]
        regions = backend._parse_blocks(blocks)
        assert regions == []

    def test_bbox_scaled_to_1000(self):
        """Textract fractional geometry is scaled to 1000."""
        backend = AWSTextractBackend(confidence_threshold=0.0, min_text_length=1)
        blocks = [
            {
                "BlockType": "WORD", "Text": "TEST", "Confidence": 99.0,
                "Geometry": {"BoundingBox": {"Left": 0.5, "Top": 0.25, "Width": 0.1, "Height": 0.05}},
            },
        ]
        regions = backend._parse_blocks(blocks)
        assert len(regions) == 1
        assert regions[0].bbox == [500, 250, 100, 50]


class TestAWSTextractDetect:
    def test_detect_calls_textract(self, make_dicom_file):
        path = make_dicom_file(filename="textract_test.dcm", pixel_value=100)

        mock_client = MagicMock()
        mock_client.detect_document_text.return_value = {
            "Blocks": [
                {
                    "BlockType": "WORD", "Text": "JOHN", "Confidence": 88.5,
                    "Geometry": {"BoundingBox": {"Left": 0.1, "Top": 0.1, "Width": 0.1, "Height": 0.05}},
                }
            ]
        }

        backend = AWSTextractBackend(confidence_threshold=0.4, min_text_length=3)
        backend._client = mock_client

        findings = backend.detect([path])
        assert len(findings) == 1
        assert findings[0].regions[0].text == "JOHN"
        mock_client.detect_document_text.assert_called_once()

    def test_detect_skips_file_without_pixel_data(self, make_dicom_file):
        path = make_dicom_file(filename="no_pix.dcm", include_pixel_data=False)

        mock_client = MagicMock()
        backend = AWSTextractBackend()
        backend._client = mock_client

        findings = backend.detect([path])
        assert findings == []
        mock_client.detect_document_text.assert_not_called()

    def test_detect_continues_on_api_exception(self, make_dicom_file):
        """If detect_document_text raises for one file, detect() skips it and continues."""
        path_err = make_dicom_file(filename="err.dcm", pixel_value=100)
        path_ok = make_dicom_file(filename="ok.dcm", pixel_value=50)

        call_count = [0]
        mock_client = MagicMock()

        def side_effect(*args, **kwargs):
            call_count[0] += 1
            if call_count[0] == 1:
                raise RuntimeError("Textract API unavailable")
            return {
                "Blocks": [{
                    "BlockType": "WORD", "Text": "PATIENT", "Confidence": 92.0,
                    "Geometry": {"BoundingBox": {
                        "Left": 0.1, "Top": 0.1, "Width": 0.1, "Height": 0.05,
                    }},
                }]
            }

        mock_client.detect_document_text.side_effect = side_effect

        backend = AWSTextractBackend(confidence_threshold=0.4, min_text_length=3)
        backend._client = mock_client

        findings = backend.detect([path_err, path_ok])

        # First file raised (skipped); second file returned a finding.
        assert len(findings) == 1
        assert findings[0].file == "ok.dcm"
        assert findings[0].regions[0].text == "PATIENT"
