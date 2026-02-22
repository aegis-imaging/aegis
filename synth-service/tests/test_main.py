"""Endpoint tests for synth-service."""

from unittest.mock import patch

from fastapi.testclient import TestClient


# ── helpers ───────────────────────────────────────────────────────────────────

# A minimal 3-slice phantom dataset that tests can reuse.
_MOCK_SLICES = None


def _get_mock_slices(size: int = 32):
    global _MOCK_SLICES
    if _MOCK_SLICES is None:
        from app.phantom import generate_phantom_slices
        _MOCK_SLICES = generate_phantom_slices(n_slices=3, size=size)
    return _MOCK_SLICES


# ── /healthz ──────────────────────────────────────────────────────────────────


class TestHealthz:
    def test_healthz_ok(self):
        from app.main import app
        client = TestClient(app)
        resp = client.get("/healthz")
        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "ok"
        assert "tool" in body

    def test_health_alias(self):
        from app.main import app
        client = TestClient(app)
        resp = client.get("/health")
        assert resp.status_code == 200

    def test_healthz_degraded_on_import_error(self):
        import builtins
        real_import = builtins.__import__

        def _fail_numpy(name, *args, **kwargs):
            if name == "numpy":
                raise ImportError("mocked failure")
            return real_import(name, *args, **kwargs)

        from app.main import app
        client = TestClient(app)
        with patch("builtins.__import__", side_effect=_fail_numpy):
            resp = client.get("/healthz")
        # Should still respond (not 500); may return degraded or ok depending on cached state.
        assert resp.status_code == 200


# ── /generate ─────────────────────────────────────────────────────────────────


class TestGenerate:
    def test_generate_defaults(self, tmp_path):
        mock_slices = generate_phantom_slices(n_slices=3, size=32)

        def _mock_generate(n_slices, size, seed, with_face):
            return mock_slices

        with patch("app.main.generate_phantom_slices", side_effect=_mock_generate), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            client = TestClient(app)
            resp = client.post("/generate", json={})

        assert resp.status_code == 200
        body = resp.json()
        assert body["study_uid"] != ""
        assert body["file_count"] == 3
        assert body["tool_used"] == "phantom"
        assert body["error"] is None

    def test_generate_creates_dicom_files(self, tmp_path):
        import os
        from app.phantom import generate_phantom_slices

        mock_slices = generate_phantom_slices(n_slices=2, size=32)

        with patch("app.main.generate_phantom_slices", return_value=mock_slices), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            client = TestClient(app)
            resp = client.post("/generate", json={"slices": 2, "size": 32})

        assert resp.status_code == 200
        body = resp.json()
        out_dir = body["output_dir"]
        assert os.path.isdir(out_dir)
        dcm_files = list(os.scandir(out_dir))
        assert len(dcm_files) == 2

    def test_generate_with_face(self, tmp_path):
        from app.phantom import generate_phantom_slices

        captured = {}

        def _capture_generate(n_slices, size, seed, with_face):
            captured["with_face"] = with_face
            return generate_phantom_slices(n_slices=n_slices, size=size, seed=seed, with_face=with_face)

        with patch("app.main.generate_phantom_slices", side_effect=_capture_generate), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            client = TestClient(app)
            resp = client.post("/generate", json={"slices": 2, "size": 32, "with_face": True})

        assert resp.status_code == 200
        assert captured.get("with_face") is True

    def test_generate_respects_max_slices(self, tmp_path):
        from app.phantom import generate_phantom_slices

        captured = {}

        def _capture(n_slices, size, seed, with_face):
            captured["n_slices"] = n_slices
            return generate_phantom_slices(n_slices=n_slices, size=32)

        with patch("app.main.generate_phantom_slices", side_effect=_capture), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 5  # limit to 5
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            client = TestClient(app)
            # Request 999 slices — should be clamped to 5.
            resp = client.post("/generate", json={"slices": 999, "size": 32})

        assert resp.status_code == 200
        assert captured["n_slices"] == 5

    def test_generate_returns_error_on_exception(self, tmp_path):
        with patch("app.main.generate_phantom_slices", side_effect=RuntimeError("boom")), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            client = TestClient(app)
            resp = client.post("/generate", json={"slices": 2, "size": 32})

        assert resp.status_code == 200  # errors are returned in body, not HTTP status
        body = resp.json()
        assert body["error"] is not None
        assert "boom" in body["error"]
        assert body["file_count"] == 0

    def test_generate_gpu_disabled_in_config_uses_phantom(self, tmp_path):
        """When cfg.use_gpu=false, GPU path is never taken even if use_gpu=true in request."""
        captured = {}

        def _capture(n_slices, size, seed, with_face):
            captured["called"] = True
            from app.phantom import generate_phantom_slices as _gps
            return _gps(n_slices=n_slices, size=size, seed=seed)

        with patch("app.main.generate_phantom_slices", side_effect=_capture), \
             patch("app.main.cfg") as mock_cfg:
            mock_cfg.max_slices = 200
            mock_cfg.max_size = 512
            mock_cfg.use_gpu = False  # GPU disabled at server level
            mock_cfg.data_dir = str(tmp_path)

            from app.main import app
            client = TestClient(app)
            resp = client.post("/generate", json={"slices": 2, "size": 32, "use_gpu": True})

        assert resp.status_code == 200
        body = resp.json()
        assert body["tool_used"] == "phantom"
        assert captured.get("called") is True


# ── helpers ───────────────────────────────────────────────────────────────────

def generate_phantom_slices(n_slices=3, size=32):
    from app.phantom import generate_phantom_slices as _gps
    return _gps(n_slices=n_slices, size=size, seed=0)
