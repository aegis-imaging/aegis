"""Tests for app.main FastAPI health endpoint and lifespan."""

from __future__ import annotations

from fastapi.testclient import TestClient

from app.main import app


class _DummyAE:
    def __init__(self, active_associations):
        self.active_associations = active_associations
        self.shutdown_called = False

    def shutdown(self):
        self.shutdown_called = True


def test_healthz_ok_with_running_scp():
    dummy = _DummyAE(active_associations=[])

    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=dummy), patch("app.main.start_scp", return_value=None):
        with TestClient(app) as client:
            resp = client.get("/healthz")
            assert resp.status_code == 200
            assert resp.json() == {"status": "ok", "scp": "running"}

    assert dummy.shutdown_called is True


def test_healthz_degraded_when_scp_not_running():
    dummy = _DummyAE(active_associations=None)

    from unittest.mock import patch

    with patch("app.main.create_scp", return_value=dummy), patch("app.main.start_scp", return_value=None):
        with TestClient(app) as client:
            resp = client.get("/healthz")
            assert resp.status_code == 200
            assert resp.json() == {"status": "degraded", "scp": "not_running"}
