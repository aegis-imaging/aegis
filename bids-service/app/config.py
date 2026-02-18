"""Configuration from environment variables."""

import os


class Config:
    tool: str = os.environ.get("BIDS_TOOL", "auto")
    dcm2niix_bin: str = os.environ.get("DCM2NIIX_BIN", "dcm2niix")


cfg = Config()
