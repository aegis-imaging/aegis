"""Tests for app.config — environment variable defaults and custom overrides."""

import importlib
import os
from unittest.mock import patch


class TestConfigDefaults:
    """Config values when no env vars are set."""

    def test_default_phi_tool(self):
        from app.config import cfg
        assert cfg.phi_tool == "auto"

    def test_default_confidence_threshold(self):
        from app.config import cfg
        assert cfg.confidence_threshold == 0.4

    def test_default_min_text_length(self):
        from app.config import cfg
        assert cfg.min_text_length == 3


class TestConfigCustomEnv:
    """Config reads custom env vars on module load (tested via reload)."""

    def test_custom_env_vars(self):
        """Env var overrides are picked up when the module is reloaded."""
        import app.config

        with patch.dict(os.environ, {
            "PHI_TOOL": "tesseract",
            "PHI_CONFIDENCE_THRESHOLD": "0.7",
            "PHI_MIN_TEXT_LENGTH": "5",
        }):
            importlib.reload(app.config)
            assert app.config.cfg.phi_tool == "tesseract"
            assert app.config.cfg.confidence_threshold == 0.7
            assert app.config.cfg.min_text_length == 5

        # Restore defaults so other tests are unaffected.
        importlib.reload(app.config)
