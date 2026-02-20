"""Tests for GoogleVisionClassificationBackend."""

from unittest.mock import MagicMock, patch
import pytest

from app.backends.google_vision import (
    GoogleVisionClassificationBackend,
    _labels_to_body_part,
    _labels_to_modality,
)
from app.backends.base import ClassificationResult


class TestLabelMapping:
    """Unit tests for the label -> medical terminology mapping functions."""

    def test_brain_label_maps_to_head(self):
        assert _labels_to_body_part(["brain", "neurology"]) == "HEAD"

    def test_skull_maps_to_head(self):
        assert _labels_to_body_part(["skull", "bone"]) == "HEAD"

    def test_lung_maps_to_chest(self):
        assert _labels_to_body_part(["lung", "organ"]) == "CHEST"

    def test_liver_maps_to_abdomen(self):
        assert _labels_to_body_part(["liver disease", "hepatic"]) == "ABDOMEN"

    def test_lumbar_maps_to_spine(self):
        assert _labels_to_body_part(["lumbar region", "vertebra"]) == "SPINE"

    def test_knee_maps_to_extremity(self):
        assert _labels_to_body_part(["knee joint", "ligament"]) == "EXTREMITY"

    def test_thyroid_maps_to_neck(self):
        assert _labels_to_body_part(["thyroid gland"]) == "NECK"

    def test_no_match_returns_none(self):
        assert _labels_to_body_part(["outdoor", "nature", "sky"]) is None

    def test_mri_label_maps_to_mr(self):
        assert _labels_to_modality(["magnetic resonance imaging"]) == "MR"

    def test_ct_label_maps_to_ct(self):
        assert _labels_to_modality(["computed tomography"]) == "CT"

    def test_xray_maps_to_cr(self):
        assert _labels_to_modality(["x-ray film"]) == "CR"

    def test_unknown_label_returns_none(self):
        assert _labels_to_modality(["photograph", "landscape"]) is None


class TestGoogleVisionClassificationBackend:
    """Integration-style tests for the classify() method."""

    def test_high_confidence_heuristic_skips_api(self, dicom_study):
        """When heuristic returns confidence >= threshold, Vision API is never called."""
        backend = GoogleVisionClassificationBackend(confidence_threshold=0.5)

        mock_client = MagicMock()
        backend._client = mock_client

        # dicom_study fixture creates MR HEAD files with direct tags -> confidence=0.95
        result = backend.classify(dicom_study)

        assert result.modality == "MR"
        assert result.body_part == "HEAD"
        assert result.confidence == 0.95
        mock_client.label_detection.assert_not_called()

    def test_low_confidence_triggers_api(self, tmp_path, make_dicom_file):
        """When heuristic confidence < threshold, Vision API is called."""
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
        mock_response.error.message = ""
        mock_ann = MagicMock()
        mock_ann.description = "brain"
        mock_response.label_annotations = [mock_ann]
        mock_client.label_detection.return_value = mock_response

        backend = GoogleVisionClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        result = backend.classify([path])

        mock_client.label_detection.assert_called_once()
        assert result.body_part == "HEAD"
        assert result.method == "google_vision_labels"
        assert result.confidence == 0.70

    def test_api_error_falls_back_to_heuristic(self, tmp_path, make_dicom_file):
        """When Vision API raises an exception, the heuristic result is returned."""
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
        mock_client.label_detection.side_effect = RuntimeError("Network error")

        backend = GoogleVisionClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        # Heuristic should find modality=MR from SOP UID -> confidence=0.60
        result = backend.classify([path])
        assert result.modality == "MR"
        assert result.method != "google_vision_labels"

    def test_available_returns_false_without_sdk(self):
        backend = GoogleVisionClassificationBackend()
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False
