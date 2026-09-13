"""Tests for the de-id orchestration layer.

These exercise the layer's branching logic (which stages run, what fails)
using monkeypatched stubs for midi_b, defacing HTTP, and phi-detection HTTP.
They don't validate the actual de-id correctness — that's the job of the
upstream service test suites.
"""

from __future__ import annotations

import json
import sys
import types

import pytest

from app import deid, storage
from app.config import RouterConfig
from tests.conftest import make_dicom


def _seed_raw_study(data_dir: str, study_uid: str, n: int = 2):
    paths = storage.layout(data_dir, study_uid)
    for i in range(n):
        storage.write_raw_bytes(paths, i, make_dicom(study_uid=study_uid))
    return paths


def test_tag_deid_fallback_copies_to_deid(base_env, monkeypatch, tmp_data_dir):
    """When midi_b isn't importable, we copy raw → deid as a dev fallback."""
    # Force ImportError on midi_b
    monkeypatch.setitem(sys.modules, "midi_b", None)
    cfg = RouterConfig.load()
    paths = _seed_raw_study(tmp_data_dir, "1.2.3", n=3)

    result = deid.run_pipeline(cfg, paths)
    assert result.ok
    assert result.stages_run == ["tag_deid"]
    deid_files = list(paths.deid.glob("*.dcm"))
    assert len(deid_files) == 3


def test_phi_scrub_called_when_url_configured(base_env, monkeypatch, tmp_data_dir):
    monkeypatch.setitem(sys.modules, "midi_b", None)
    monkeypatch.setenv("PHI_DETECTION_SERVICE_URL", "http://phi:8080")
    cfg = RouterConfig.load()
    paths = _seed_raw_study(tmp_data_dir, "1.2.3.4", n=2)

    captured = {}

    def fake_post(self, url, json=None, **_):
        captured["url"] = url
        captured["payload"] = json

        class _Resp:
            status_code = 200

            def raise_for_status(self) -> None:
                return None

            def json(self_inner):
                return {"status": "complete"}

        return _Resp()

    monkeypatch.setattr("httpx.Client.post", fake_post)
    result = deid.run_pipeline(cfg, paths)
    assert result.ok
    assert "phi_scrub" in result.stages_run
    assert captured["url"].endswith("/redact")
    assert captured["payload"]["study_uid"] == "1.2.3.4"


def test_defacing_failure_surfaces_as_pipeline_failure(base_env, monkeypatch, tmp_data_dir):
    monkeypatch.setitem(sys.modules, "midi_b", None)
    monkeypatch.setenv("DEFACING_SERVICE_URL", "http://deface:8080")
    cfg = RouterConfig.load()
    paths = _seed_raw_study(tmp_data_dir, "1.2.3.5", n=1)

    def fake_post(self, url, json=None, **_):
        class _Resp:
            status_code = 200

            def raise_for_status(self) -> None:
                return None

            def json(self_inner):
                return {"status": "failed", "error": "skull extraction failed"}

        return _Resp()

    monkeypatch.setattr("httpx.Client.post", fake_post)
    result = deid.run_pipeline(cfg, paths)
    assert not result.ok
    assert result.failed_stage == "defacing"
    assert "skull extraction failed" in result.error


def test_summarize_study_reads_metadata(base_env, tmp_data_dir, monkeypatch):
    monkeypatch.setitem(sys.modules, "midi_b", None)
    cfg = RouterConfig.load()
    paths = _seed_raw_study(tmp_data_dir, "1.2.3.6", n=3)
    deid.run_pipeline(cfg, paths)
    meta = deid.summarize_study(paths)
    assert meta["study_instance_uid"] == "1.2.3.6"
    assert meta["modality"] == "MR"
    assert meta["body_part"] == "HEAD"
    assert meta["instance_count"] == 3
    assert meta["series_count"] == 3  # each fake dicom gets a fresh series uid
    assert meta["study_size_bytes"] > 0
