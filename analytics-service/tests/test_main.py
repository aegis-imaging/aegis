"""Tests for the analytics service FastAPI app."""

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
def bids_dir():
    with tempfile.TemporaryDirectory() as d:
        # Create a minimal BIDS structure with a NIfTI file
        anat_dir = os.path.join(d, "sub-test01", "anat")
        os.makedirs(anat_dir)
        nifti = os.path.join(anat_dir, "sub-test01_T1w.nii.gz")
        with open(nifti, "wb") as f:
            f.write(b"\x1f\x8b" + b"\x00" * 100)  # minimal gzip header
        yield d


class TestHealthz:
    def test_healthz_returns_ok(self, client):
        resp = client.get("/healthz")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] in ("ok", "degraded")
        assert "backends" in data

    def test_health_alias(self, client):
        resp = client.get("/health")
        assert resp.status_code == 200

    def test_healthz_shows_all_backends(self, client):
        resp = client.get("/healthz")
        data = resp.json()
        for name in ("freesurfer", "fsl", "ants", "spm"):
            assert name in data["backends"]


class TestAnalyzeEndpoint:
    def test_missing_bids_dir(self, client):
        resp = client.post("/analyze", json={
            "study_uid": "1.2.3",
            "bids_dir": "",
            "output_dir": "/tmp/out",
        })
        assert resp.status_code == 400

    def test_nonexistent_bids_dir(self, client):
        resp = client.post("/analyze", json={
            "study_uid": "1.2.3",
            "bids_dir": "/nonexistent/path",
            "output_dir": "/tmp/out",
        })
        assert resp.status_code == 400

    def test_no_backends_available(self, client, bids_dir):
        with patch("app.main._select_backends", return_value=[]):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
            })
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "failed"
        assert "No analytics backends" in data["error"]

    def test_single_backend_success(self, client, bids_dir):
        mock_backend = MagicMock()
        mock_backend.name = "fsl"
        mock_backend.available.return_value = True
        mock_backend.analyze.return_value = AnalyticsResult(
            tool="fsl",
            success=True,
            duration_seconds=5.0,
            outputs=["/tmp/brain.nii.gz"],
        )

        with patch("app.main._select_backends", return_value=[mock_backend]):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
            })

        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "complete"
        assert len(data["results"]) == 1
        assert data["results"][0]["tool"] == "fsl"
        assert data["results"][0]["success"] is True

    def test_partial_success(self, client, bids_dir):
        backend_ok = MagicMock()
        backend_ok.name = "fsl"
        backend_ok.available.return_value = True
        backend_ok.analyze.return_value = AnalyticsResult(
            tool="fsl", success=True, duration_seconds=3.0,
        )

        backend_fail = MagicMock()
        backend_fail.name = "freesurfer"
        backend_fail.available.return_value = True
        backend_fail.analyze.return_value = AnalyticsResult(
            tool="freesurfer", success=False, error="recon-all not found",
        )

        with patch("app.main._select_backends", return_value=[backend_ok, backend_fail]):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
            })

        data = resp.json()
        assert data["status"] == "partial"
        assert len(data["results"]) == 2

    def test_all_fail(self, client, bids_dir):
        backend = MagicMock()
        backend.name = "ants"
        backend.available.return_value = True
        backend.analyze.return_value = AnalyticsResult(
            tool="ants", success=False, error="template not found",
        )

        with patch("app.main._select_backends", return_value=[backend]):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
            })

        data = resp.json()
        assert data["status"] == "failed"

    def test_backend_crash_handled(self, client, bids_dir):
        backend = MagicMock()
        backend.name = "spm"
        backend.available.return_value = True
        backend.analyze.side_effect = RuntimeError("segfault")

        with patch("app.main._select_backends", return_value=[backend]):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
            })

        data = resp.json()
        assert data["status"] == "failed"
        assert data["results"][0]["error"] == "segfault"

    def test_explicit_tools_selection(self, client, bids_dir):
        with patch("app.main.ALL_BACKENDS") as mock_all:
            mock_fsl = MagicMock()
            mock_fsl.name = "fsl"
            mock_fsl.available.return_value = True
            mock_fsl.analyze.return_value = AnalyticsResult(
                tool="fsl", success=True, duration_seconds=2.0,
            )
            mock_all.__iter__ = lambda self: iter([mock_fsl])

            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
                "tools": ["fsl"],
            })

        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "complete"

    def test_response_includes_metrics(self, client, bids_dir):
        backend = MagicMock()
        backend.name = "freesurfer"
        backend.available.return_value = True
        backend.analyze.return_value = AnalyticsResult(
            tool="freesurfer",
            success=True,
            duration_seconds=100.0,
            outputs=["/tmp/stats/aseg.stats"],
            metrics={"subcortical_volumes": {"Hippocampus": 3200.5}},
        )

        with patch("app.main._select_backends", return_value=[backend]):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/analytics-test-out",
            })

        data = resp.json()
        assert data["results"][0]["metrics"]["subcortical_volumes"]["Hippocampus"] == 3200.5


class TestSelectBackends:
    def test_auto_returns_available(self):
        from app.main import _select_backends
        with patch("app.main.ALL_BACKENDS") as mock_all:
            b1 = MagicMock()
            b1.available.return_value = False
            b2 = MagicMock()
            b2.available.return_value = True
            mock_all.__iter__ = lambda self: iter([b1, b2])

            result = _select_backends("auto")
            assert result == [b2]

    def test_auto_empty_when_none_available(self):
        from app.main import _select_backends
        with patch("app.main.ALL_BACKENDS") as mock_all:
            b1 = MagicMock()
            b1.available.return_value = False
            mock_all.__iter__ = lambda self: iter([b1])

            result = _select_backends("auto")
            assert result == []

    def test_specific_tool(self):
        from app.main import _select_backends
        with patch("app.main.ALL_BACKENDS") as mock_all:
            b1 = MagicMock()
            b1.name = "fsl"
            b1.available.return_value = True
            mock_all.__iter__ = lambda self: iter([b1])

            result = _select_backends("fsl")
            assert result == [b1]

    def test_unknown_tool_raises(self):
        from app.main import _select_backends
        with patch("app.main.ALL_BACKENDS", []):
            with pytest.raises(ValueError, match="Unknown analytics tool"):
                _select_backends("nonexistent")
