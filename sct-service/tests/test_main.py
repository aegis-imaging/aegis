"""Tests for the SCT service FastAPI app."""

import os
import tempfile
from unittest.mock import patch, MagicMock

import pytest
from fastapi.testclient import TestClient

from app.backends.base import SctResult


@pytest.fixture
def client():
    from app.main import app
    return TestClient(app)


@pytest.fixture
def bids_dir():
    with tempfile.TemporaryDirectory() as d:
        anat_dir = os.path.join(d, "sub-test01", "anat")
        os.makedirs(anat_dir)
        nifti = os.path.join(anat_dir, "sub-test01_T2w.nii.gz")
        with open(nifti, "wb") as f:
            f.write(b"\x1f\x8b" + b"\x00" * 100)
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

    def test_healthz_shows_sct_backend(self, client):
        resp = client.get("/healthz")
        data = resp.json()
        assert "sct" in data["backends"]

    def test_healthz_degraded_when_unavailable(self, client):
        with patch("app.main.ALL_BACKENDS") as mock_backends:
            b = MagicMock()
            b.name = "sct"
            b.available.return_value = False
            mock_backends.__iter__ = lambda self: iter([b])

            resp = client.get("/healthz")
            data = resp.json()
            assert data["status"] == "degraded"


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

    def test_no_backend_available(self, client, bids_dir):
        with patch("app.main._select_backend", return_value=None):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/sct-test-out",
            })
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "failed"
        assert "No SCT backends" in data["error"]

    def test_successful_analysis(self, client, bids_dir):
        mock_backend = MagicMock()
        mock_backend.name = "sct"
        mock_backend.available.return_value = True
        mock_backend.analyze.return_value = SctResult(
            tool="sct",
            success=True,
            duration_seconds=120.0,
            outputs=["/tmp/seg_sc.nii.gz", "/tmp/csa_perlevel.csv"],
            metrics={
                "atlas": "sct",
                "steps_completed": ["deepseg_sc", "label_vertebrae", "process_segmentation"],
                "csa_per_level": {"C3": 64.5, "C4": 62.1},
                "mean_csa_mm2": 63.3,
            },
        )

        with patch("app.main._select_backend", return_value=mock_backend):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/sct-test-out",
            })

        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "complete"
        assert len(data["results"]) == 1
        assert data["results"][0]["tool"] == "sct"
        assert data["results"][0]["success"] is True
        assert data["results"][0]["metrics"]["mean_csa_mm2"] == 63.3

    def test_partial_success(self, client, bids_dir):
        mock_backend = MagicMock()
        mock_backend.name = "sct"
        mock_backend.available.return_value = True
        mock_backend.analyze.return_value = SctResult(
            tool="sct",
            success=True,
            duration_seconds=30.0,
            outputs=["/tmp/seg_sc.nii.gz"],
            metrics={
                "atlas": "sct",
                "steps_completed": ["deepseg_sc"],
            },
        )

        with patch("app.main._select_backend", return_value=mock_backend):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/sct-test-out",
            })

        data = resp.json()
        assert data["status"] == "partial"

    def test_all_fail(self, client, bids_dir):
        mock_backend = MagicMock()
        mock_backend.name = "sct"
        mock_backend.available.return_value = True
        mock_backend.analyze.return_value = SctResult(
            tool="sct",
            success=False,
            error="sct_deepseg_sc failed",
        )

        with patch("app.main._select_backend", return_value=mock_backend):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/sct-test-out",
            })

        data = resp.json()
        assert data["status"] == "failed"

    def test_backend_crash_handled(self, client, bids_dir):
        mock_backend = MagicMock()
        mock_backend.name = "sct"
        mock_backend.available.return_value = True
        mock_backend.analyze.side_effect = RuntimeError("segfault")

        with patch("app.main._select_backend", return_value=mock_backend):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/sct-test-out",
            })

        data = resp.json()
        assert data["status"] == "failed"
        assert data["results"][0]["error"] == "segfault"

    def test_contrast_override(self, client, bids_dir):
        mock_backend = MagicMock()
        mock_backend.name = "sct"
        mock_backend.available.return_value = True
        mock_backend.analyze.return_value = SctResult(
            tool="sct",
            success=True,
            duration_seconds=60.0,
            metrics={"steps_completed": ["deepseg_sc", "label_vertebrae", "process_segmentation"]},
        )

        with patch("app.main._select_backend", return_value=mock_backend):
            resp = client.post("/analyze", json={
                "study_uid": "1.2.3",
                "bids_dir": bids_dir,
                "output_dir": "/tmp/sct-test-out",
                "contrast": "t1",
            })

        assert resp.status_code == 200
        mock_backend.analyze.assert_called_once()
        _, kwargs = mock_backend.analyze.call_args
        assert kwargs["contrast"] == "t1"


class TestSelectBackend:
    def test_auto_returns_available(self):
        from app.main import _select_backend
        with patch("app.main.ALL_BACKENDS") as mock_all:
            b = MagicMock()
            b.name = "sct"
            b.available.return_value = True
            mock_all.__iter__ = lambda self: iter([b])

            result = _select_backend("auto")
            assert result == b

    def test_auto_none_when_unavailable(self):
        from app.main import _select_backend
        with patch("app.main.ALL_BACKENDS") as mock_all:
            b = MagicMock()
            b.available.return_value = False
            mock_all.__iter__ = lambda self: iter([b])

            result = _select_backend("auto")
            assert result is None

    def test_specific_tool(self):
        from app.main import _select_backend
        with patch("app.main.ALL_BACKENDS") as mock_all:
            b = MagicMock()
            b.name = "sct"
            b.available.return_value = True
            mock_all.__iter__ = lambda self: iter([b])

            result = _select_backend("sct")
            assert result == b

    def test_unknown_tool_returns_none(self):
        from app.main import _select_backend
        with patch("app.main.ALL_BACKENDS", []):
            result = _select_backend("nonexistent")
            assert result is None
