"""Endpoint tests for protocol-service."""

from unittest.mock import patch

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_healthz_ok():
    response = client.get("/healthz")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "ok"
    assert data["backend"] == "basic"


def test_check_success(dicom_study):
    response = client.post("/check", json={
        "study_uid": "1.2.3.4.5",
        "input_paths": dicom_study,
        "rules": [
            {"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
             "severity": "warning", "tolerance_pct": 5.0},
            {"tag_keyword": "FlipAngle", "target": 9.0, "match_type": "numeric",
             "severity": "info", "tolerance_pct": 10.0},
        ],
    })
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "complete"
    assert data["overall_compliance"] in ("compliant", "minor_deviations", "non_compliant")
    assert data["dicom_format"] == "classic"
    assert data["device_info"]["manufacturer"] == "SIEMENS"
    assert len(data["findings"]) == 2


def test_check_empty_paths():
    response = client.post("/check", json={
        "study_uid": "1.2.3",
        "input_paths": [],
        "rules": [{"tag_keyword": "TR", "target": 2000}],
    })
    assert response.status_code == 400


def test_check_empty_rules(dicom_study):
    response = client.post("/check", json={
        "study_uid": "1.2.3",
        "input_paths": dicom_study,
        "rules": [],
    })
    assert response.status_code == 400
