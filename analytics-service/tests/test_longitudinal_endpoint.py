"""Tests for the /analyze-longitudinal endpoint."""

import os
import tempfile
from unittest.mock import patch, MagicMock

import pytest
from fastapi.testclient import TestClient

from app.backends.base import AnalyticsResult


@pytest.fixture
def client():
    from app.main import app
    return TestClient(app)


@pytest.fixture
def paired_bids():
    """Create minimal paired BIDS directories with T1w files."""
    with tempfile.TemporaryDirectory() as bl, tempfile.TemporaryDirectory() as fu:
        for d in (bl, fu):
            anat = os.path.join(d, "sub-01", "anat")
            os.makedirs(anat)
            t1w = os.path.join(anat, "sub-01_T1w.nii.gz")
            with open(t1w, "wb") as f:
                f.write(b"\x1f\x8b" + b"\x00" * 100)
        yield bl, fu


class TestAnalyzeLongitudinalEndpoint:
    def test_missing_baseline_dir(self, client):
        resp = client.post("/analyze-longitudinal", json={
            "baseline_study_uid": "1.2.3",
            "followup_study_uid": "1.2.4",
            "baseline_bids_dir": "",
            "followup_bids_dir": "/some/path",
            "output_dir": "/tmp/out",
            "scan_interval_days": 365,
        })
        assert resp.status_code == 400

    def test_nonexistent_baseline_dir(self, client):
        with tempfile.TemporaryDirectory() as fu:
            resp = client.post("/analyze-longitudinal", json={
                "baseline_study_uid": "1.2.3",
                "followup_study_uid": "1.2.4",
                "baseline_bids_dir": "/nonexistent/path",
                "followup_bids_dir": fu,
                "output_dir": "/tmp/out",
                "scan_interval_days": 365,
            })
            assert resp.status_code == 400

    def test_nonexistent_followup_dir(self, client):
        with tempfile.TemporaryDirectory() as bl:
            resp = client.post("/analyze-longitudinal", json={
                "baseline_study_uid": "1.2.3",
                "followup_study_uid": "1.2.4",
                "baseline_bids_dir": bl,
                "followup_bids_dir": "/nonexistent",
                "output_dir": "/tmp/out",
                "scan_interval_days": 365,
            })
            assert resp.status_code == 400

    def test_negative_scan_interval(self, client, paired_bids):
        bl, fu = paired_bids
        resp = client.post("/analyze-longitudinal", json={
            "baseline_study_uid": "1.2.3",
            "followup_study_uid": "1.2.4",
            "baseline_bids_dir": bl,
            "followup_bids_dir": fu,
            "output_dir": "/tmp/out",
            "scan_interval_days": -10,
        })
        assert resp.status_code == 400

    def test_zero_scan_interval(self, client, paired_bids):
        bl, fu = paired_bids
        resp = client.post("/analyze-longitudinal", json={
            "baseline_study_uid": "1.2.3",
            "followup_study_uid": "1.2.4",
            "baseline_bids_dir": bl,
            "followup_bids_dir": fu,
            "output_dir": "/tmp/out",
            "scan_interval_days": 0,
        })
        assert resp.status_code == 400

    def test_no_backends_available(self, client, paired_bids):
        bl, fu = paired_bids
        with patch("app.main._select_longitudinal_backends", return_value=[]):
            resp = client.post("/analyze-longitudinal", json={
                "baseline_study_uid": "1.2.3",
                "followup_study_uid": "1.2.4",
                "baseline_bids_dir": bl,
                "followup_bids_dir": fu,
                "output_dir": "/tmp/analytics-test-long",
                "scan_interval_days": 365,
            })
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "failed"
        assert "No longitudinal backends" in data["error"]

    def test_successful_analysis(self, client, paired_bids):
        bl, fu = paired_bids
        mock_backend = MagicMock()
        mock_backend.name = "tbm_syn"
        mock_backend.available.return_value = True
        mock_backend.analyze_longitudinal.return_value = AnalyticsResult(
            tool="tbm_syn",
            success=True,
            duration_seconds=120.0,
            outputs=["/out/log_jacobian.nii.gz"],
            metrics={"scan_interval_days": 365, "ad_composite_mean": -0.025},
        )

        with patch("app.main.LONGITUDINAL_BACKENDS", [mock_backend]):
            with patch("app.main._select_longitudinal_backends", return_value=[mock_backend]):
                resp = client.post("/analyze-longitudinal", json={
                    "baseline_study_uid": "1.2.3",
                    "followup_study_uid": "1.2.4",
                    "baseline_bids_dir": bl,
                    "followup_bids_dir": fu,
                    "output_dir": "/tmp/analytics-test-long",
                    "scan_interval_days": 365,
                })

        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "complete"
        assert data["study_uid"] == "1.2.4"  # follow-up UID
        assert len(data["results"]) == 1
        assert data["results"][0]["tool"] == "tbm_syn"
        assert data["results"][0]["success"] is True

    def test_backend_crash_handled(self, client, paired_bids):
        bl, fu = paired_bids
        mock_backend = MagicMock()
        mock_backend.name = "tbm_syn"
        mock_backend.available.return_value = True
        mock_backend.analyze_longitudinal.side_effect = RuntimeError("crash")

        with patch("app.main.LONGITUDINAL_BACKENDS", [mock_backend]):
            with patch("app.main._select_longitudinal_backends", return_value=[mock_backend]):
                resp = client.post("/analyze-longitudinal", json={
                    "baseline_study_uid": "1.2.3",
                    "followup_study_uid": "1.2.4",
                    "baseline_bids_dir": bl,
                    "followup_bids_dir": fu,
                    "output_dir": "/tmp/analytics-test-long",
                    "scan_interval_days": 365,
                })

        data = resp.json()
        assert data["status"] == "failed"
        assert data["results"][0]["error"] == "crash"

    def test_atlas_param_passed_through(self, client, paired_bids):
        bl, fu = paired_bids
        mock_backend = MagicMock()
        mock_backend.name = "tbm_syn"
        mock_backend.available.return_value = True
        mock_backend.analyze_longitudinal.return_value = AnalyticsResult(
            tool="tbm_syn", success=True, duration_seconds=1.0,
        )

        with patch("app.main.LONGITUDINAL_BACKENDS", [mock_backend]):
            with patch("app.main._select_longitudinal_backends", return_value=[mock_backend]):
                resp = client.post("/analyze-longitudinal", json={
                    "baseline_study_uid": "1.2.3",
                    "followup_study_uid": "1.2.4",
                    "baseline_bids_dir": bl,
                    "followup_bids_dir": fu,
                    "output_dir": "/tmp/analytics-test-long",
                    "scan_interval_days": 365,
                    "atlas": "mcalt",
                })

        # Verify atlas kwarg was passed through
        call_kwargs = mock_backend.analyze_longitudinal.call_args
        assert call_kwargs.kwargs.get("atlas") == "mcalt"


class TestHealthzLongitudinal:
    def test_healthz_shows_longitudinal_backends(self, client):
        resp = client.get("/healthz")
        data = resp.json()
        assert "longitudinal_backends" in data
        assert "tbm_syn" in data["longitudinal_backends"]
