"""Endpoint tests for classification-service."""

from unittest.mock import patch, MagicMock

from fastapi.testclient import TestClient

from app.backends.base import ClassificationResult


# Patch _backend before importing app to avoid side effects
with patch("app.main._backend") as _mock:
    _mock.name = "heuristic"
    _mock.available.return_value = True
    from app.main import app

client = TestClient(app)


def test_healthz_ok():
    response = client.get("/healthz")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "ok"
    assert data["available"] is True


def test_healthz_degraded():
    with patch("app.main._backend", None):
        response = client.get("/healthz")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "degraded"
    assert data["available"] is False


def test_classify_success(dicom_study):
    response = client.post("/classify", json={
        "study_uid": "1.2.3.4.5",
        "input_paths": dicom_study,
    })
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "complete"
    assert data["tool_used"] == "heuristic"
    assert data["modality"] == "MR"
    assert data["body_part"] == "HEAD"
    assert data["confidence"] == 0.95


def test_classify_no_backend(dicom_study):
    with patch("app.main._backend", None):
        response = client.post("/classify", json={
            "study_uid": "1.2.3.4.5",
            "input_paths": dicom_study,
        })
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "failed"
    assert "No classification backend" in data["error"]


def test_classify_invalid_files(tmp_path):
    txt_file = tmp_path / "not_dicom.txt"
    txt_file.write_text("not a dicom file")
    response = client.post("/classify", json={
        "study_uid": "1.2.3.4.5",
        "input_paths": [str(txt_file)],
    })
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "complete"
    assert data["confidence"] == 0.0


def test_classify_response_fields(dicom_study):
    response = client.post("/classify", json={
        "study_uid": "1.2.3.4.5",
        "input_paths": dicom_study,
    })
    data = response.json()
    for field in ["status", "modality", "body_part", "confidence", "method", "tool_used", "duration_seconds"]:
        assert field in data, f"Missing field: {field}"
