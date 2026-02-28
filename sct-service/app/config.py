"""SCT service configuration — loaded from environment variables."""

import os


class Config:
    tool: str = os.environ.get("SCT_TOOL", "auto")
    contrast: str = os.environ.get("SCT_CONTRAST", "t2")
    data_dir: str = os.environ.get("SCT_DATA_DIR", "/app/data")
    timeout: int = int(os.environ.get("SCT_TIMEOUT", "3600"))


cfg = Config()
