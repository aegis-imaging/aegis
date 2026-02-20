"""Tests for AWSRekognitionClassificationBackend."""

from unittest.mock import MagicMock, patch
import pytest

from app.backends.aws_rekognition import (
    AWSRekognitionClassificationBackend,
    _rekog_labels_to_body_part,
    _rekog_labels_to_modality,
)


class TestRekognitionLabelMapping:
    def test_brain_maps_to_head(self):
        assert _rekog_labels_to_body_part(["Brain", "Neurology"]) == "HEAD"

    def test_lung_maps_to_chest(self):
        assert _rekog_labels_to_body_part(["Lung"]) == "CHEST"

    def test_vertebra_maps_to_spine(self):
        assert _rekog_labels_to_body_part(["Vertebra", "Disc"]) == "SPINE"

    def test_extremity_keyword_maps(self):
        assert _rekog_labels_to_body_part(["Knee", "Joint"]) == "EXTREMITY"

    def test_mri_maps_to_mr(self):
        assert _rekog_labels_to_modality(["MRI", "Scan"]) == "MR"

    def test_no_match_returns_none(self):
        assert _rekog_labels_to_body_part(["Dog", "Cat"]) is None


class TestAWSRekognitionBackend:
    def test_high_confidence_heuristic_skips_api(self, dicom_study):
        backend = AWSRekognitionClassificationBackend(confidence_threshold=0.5)
        mock_client = MagicMock()
        backend._client = mock_client

        result = backend.classify(dicom_study)
        assert result.confidence == 0.95
        mock_client.detect_labels.assert_not_called()

    def test_low_confidence_triggers_rekognition(self, tmp_path, make_dicom_file):
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="1.2.840.10008.5.1.4.1.1.88.11",
            series_description="",
            study_description="",
            protocol_name="",
        )

        mock_client = MagicMock()
        mock_client.detect_labels.return_value = {
            "Labels": [{"Name": "Brain"}, {"Name": "Head"}]
        }

        backend = AWSRekognitionClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        result = backend.classify([path])
        mock_client.detect_labels.assert_called_once()
        assert result.body_part == "HEAD"
        assert result.method == "aws_rekognition_labels"
        assert result.confidence == 0.70

    def test_api_exception_falls_back(self, tmp_path, make_dicom_file):
        path = make_dicom_file(
            tmp_dir=tmp_path,
            modality_value=None,
            body_part_value=None,
            sop_class_uid="1.2.840.10008.5.1.4.1.1.4",
            series_description="",
            study_description="",
            protocol_name="",
        )
        mock_client = MagicMock()
        mock_client.detect_labels.side_effect = RuntimeError("Timeout")

        backend = AWSRekognitionClassificationBackend(confidence_threshold=0.5)
        backend._client = mock_client

        result = backend.classify([path])
        assert result.method != "aws_rekognition_labels"

    def test_available_returns_false_without_sdk(self):
        backend = AWSRekognitionClassificationBackend()
        with patch.object(backend, "available", return_value=False):
            assert backend.available() is False
