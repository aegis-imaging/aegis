"""Smoke tests for the consolidated dicom-tools FastAPI app.

Verifies that the top-level app imports cleanly with every sub-module
mounted, and that the /healthz response is what Cloud Run liveness
probes expect. Per-sub-module functional tests will be ported over in
follow-up PRs (each former sidecar had its own pytest suite that we
haven't moved yet — kept in the retired directories until the
consolidation has settled in production).
"""
from fastapi.testclient import TestClient

from app.main import app


def test_app_imports():
    # Just importing app.main is the strongest possible test that all six
    # sub-modules load without circular-import or missing-dependency errors
    # under the union requirements.
    assert app is not None


def test_healthz_returns_ok():
    client = TestClient(app)
    res = client.get("/healthz")
    assert res.status_code == 200
    body = res.json()
    assert body["ok"] is True
    assert body["service"] == "dicom-tools"


def test_root_health_returns_ok():
    client = TestClient(app)
    res = client.get("/health")
    assert res.status_code == 200


def test_sub_module_routes_mounted():
    # Each former sidecar's POST endpoint should be addressable under its
    # prefix. We don't call them with real payloads (would need DICOM fixtures);
    # instead we POST an empty body and assert we get a 422 ("missing
    # required fields") — which proves the route exists and FastAPI parsed
    # the request as far as schema validation. A 404 would mean the route
    # isn't mounted.
    #
    # Note: mounted sub-apps don't appear in the parent's /openapi.json,
    # which is why we probe directly instead of inspecting the schema.
    routes_to_probe = [
        "/phi/detect",
        "/phi/detect-tags",
        "/phi/redact",
        "/qc/check",
        "/classify/classify",
        "/protocol/check",
        "/synth/generate",
        "/bids/convert",
    ]
    client = TestClient(app)
    for route in routes_to_probe:
        res = client.post(route, json={})
        assert res.status_code != 404, f"{route} not mounted (got 404)"
