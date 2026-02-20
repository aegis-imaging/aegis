"""Tests for app.ingest."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

from app.ingest import StudyAccumulator, trigger_ingest


def _acc() -> StudyAccumulator:
    return StudyAccumulator(
        study_instance_uid="1.2.3.4",
        modality="MR",
        body_part="HEAD",
        study_description="BRAIN MRI",
        series_uids={"1", "2"},
        file_count=5,
        calling_ae_title="PACS_AE",
    )


def test_trigger_ingest_success(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")
    monkeypatch.setattr("app.config.DIMSE_PROJECT_SLUG", "default")
    monkeypatch.setattr("app.config.DIMSE_INGEST_TIMEOUT", 30)

    mock_resp = MagicMock(status_code=200, text="ok")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is True
    mock_client.post.assert_called_once()
    called_url = mock_client.post.call_args[0][0]
    called_json = mock_client.post.call_args.kwargs["json"]
    assert called_url == "http://api:8080/api/ingest"
    assert called_json["project_slug"] == "default"
    assert called_json["study_metadata"]["study_instance_uid"] == "1.2.3.4"
    assert called_json["study_metadata"]["series_count"] == 2
    assert called_json["study_metadata"]["instance_count"] == 5


def test_trigger_ingest_http_error(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")

    mock_resp = MagicMock(status_code=500, text="boom")
    mock_client = MagicMock()
    mock_client.post.return_value = mock_resp

    with patch("app.ingest.httpx.Client") as mock_client_cls:
        mock_client_cls.return_value.__enter__.return_value = mock_client
        ok = trigger_ingest(_acc())

    assert ok is False


def test_trigger_ingest_exception(monkeypatch):
    monkeypatch.setattr("app.config.API_URL", "http://api:8080")

    with patch("app.ingest.httpx.Client", side_effect=RuntimeError("network down")):
        ok = trigger_ingest(_acc())

    assert ok is False
