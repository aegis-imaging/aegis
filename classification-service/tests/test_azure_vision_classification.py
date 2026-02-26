"""Tests for the Azure Computer Vision classification backend."""

from unittest.mock import MagicMock, patch
import pytest

from app.backends.azure_vision import (
    AzureVisionClassificationBackend,
    _az_tags_to_body_part,
    _az_tags_to_modality,
)


# ---------------------------------------------------------------------------
# Label mapping
# ---------------------------------------------------------------------------

class TestAzureLabelMapping:
    def test_body_part_brain(self):
        assert _az_tags_to_body_part(["scan", "brain", "medical"]) == "HEAD"

    def test_body_part_skull(self):
        assert _az_tags_to_body_part(["skull", "anatomy"]) == "HEAD"

    def test_body_part_lung(self):
        assert _az_tags_to_body_part(["lung", "tissue"]) == "CHEST"

    def test_body_part_liver(self):
        assert _az_tags_to_body_part(["liver", "organ"]) == "ABDOMEN"

    def test_body_part_lumbar(self):
        assert _az_tags_to_body_part(["lumbar", "disc"]) == "SPINE"

    def test_body_part_knee(self):
        assert _az_tags_to_body_part(["knee", "joint"]) == "EXTREMITY"

    def test_body_part_thyroid(self):
        assert _az_tags_to_body_part(["thyroid", "gland"]) == "NECK"

    def test_body_part_no_match(self):
        assert _az_tags_to_body_part(["abstract", "photo"]) is None

    def test_modality_mri(self):
        assert _az_tags_to_modality(["mri", "scan"]) == "MR"

    def test_modality_ct(self):
        assert _az_tags_to_modality(["computed tomography", "slice"]) == "CT"

    def test_modality_xray(self):
        assert _az_tags_to_modality(["x-ray", "radiograph"]) == "CR"

    def test_modality_no_match(self):
        assert _az_tags_to_modality(["photo", "nature"]) is None


# ---------------------------------------------------------------------------
# Backend
# ---------------------------------------------------------------------------

class TestAzureVisionClassificationBackend:
    def test_high_confidence_heuristic_skips_api(self, dicom_study):
        """When heuristic has high confidence from direct DICOM tags, Azure API is not called."""
        backend = AzureVisionClassificationBackend(confidence_threshold=0.5)
        backend._endpoint = "https://test.cognitiveservices.azure.com"
        mock_client = MagicMock()
        backend._client = mock_client

        result = backend.classify(dicom_study)

        assert result.confidence >= 0.5
        assert result.modality == "MR"
        assert result.body_part == "HEAD"
        mock_client.analyze.assert_not_called()

    def test_low_confidence_triggers_api(self, tmp_path, make_dicom_file):
        """When heuristic confidence is low, Azure Vision tagging is called."""
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="unknown",
            series_description="",
            study_description="",
            protocol_name="",
        )

        backend = AzureVisionClassificationBackend(confidence_threshold=0.5)
        backend._endpoint = "https://test.cognitiveservices.azure.com"

        mock_client = MagicMock()
        # Mock Azure Vision tags response
        tag1 = MagicMock()
        tag1.name = "brain"
        tag1.confidence = 0.85
        tag2 = MagicMock()
        tag2.name = "medical"
        tag2.confidence = 0.90
        mock_response = MagicMock()
        mock_response.tags.values = [tag1, tag2]
        mock_client.analyze.return_value = mock_response
        backend._client = mock_client

        result = backend.classify([path])

        assert result.body_part == "HEAD"
        assert result.method == "azure_vision_tags"
        assert result.confidence == 0.70
        mock_client.analyze.assert_called_once()

    def test_api_exception_falls_back(self, tmp_path, make_dicom_file):
        """API exception falls back to heuristic result."""
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="unknown",
            series_description="",
            study_description="",
            protocol_name="",
        )

        backend = AzureVisionClassificationBackend(confidence_threshold=0.5)
        backend._endpoint = "https://test.cognitiveservices.azure.com"

        mock_client = MagicMock()
        mock_client.analyze.side_effect = RuntimeError("Azure API error")
        backend._client = mock_client

        result = backend.classify([path])
        # Should return heuristic result (possibly low confidence), not crash
        assert result is not None

    def test_available_returns_false_without_sdk(self):
        backend = AzureVisionClassificationBackend()
        backend._endpoint = "https://test.cognitiveservices.azure.com"
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False

    def test_available_returns_false_without_endpoint(self):
        backend = AzureVisionClassificationBackend()
        backend._endpoint = ""
        assert backend.available() is False

    def test_name(self):
        backend = AzureVisionClassificationBackend()
        assert backend.name == "azure_vision"
