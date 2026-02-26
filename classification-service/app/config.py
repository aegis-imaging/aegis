"""Configuration from environment variables."""

import os


class Config:
    # Options: "auto", "gemini", "heuristic", "google_vision", "aws_rekognition"
    # "auto" tries each in priority order: gemini > google_vision > aws_rekognition > heuristic
    tool: str = os.environ.get("CLASSIFY_TOOL", "auto")
    confidence_threshold: float = float(os.environ.get("CLASSIFY_CONFIDENCE_THRESHOLD", "0.5"))


cfg = Config()
