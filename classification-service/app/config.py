"""Configuration from environment variables."""

import os


class Config:
    tool: str = os.environ.get("CLASSIFY_TOOL", "auto")
    confidence_threshold: float = float(os.environ.get("CLASSIFY_CONFIDENCE_THRESHOLD", "0.5"))


cfg = Config()
