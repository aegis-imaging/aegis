"""Configuration from environment variables."""

from __future__ import annotations

import os


# DICOM SCP settings
DIMSE_AE_TITLE: str = os.getenv("DIMSE_AE_TITLE", "AEGIS")
DIMSE_PORT: int = int(os.getenv("DIMSE_PORT", "11112"))
DIMSE_MAX_ASSOCIATIONS: int = int(os.getenv("DIMSE_MAX_ASSOCIATIONS", "10"))

# File storage — shared volume with Go API
DIMSE_DATA_DIR: str = os.getenv("DIMSE_DATA_DIR", "/app/data")

# Go API endpoint for study ingest
API_URL: str = os.getenv("API_URL", "http://api:8080")

# Default project for ingested studies
DIMSE_PROJECT_SLUG: str = os.getenv("DIMSE_PROJECT_SLUG", "default")
# Optional explicit institution attribution override
DIMSE_INSTITUTION_ID: str = os.getenv("DIMSE_INSTITUTION_ID", "")

# HTTP client settings
DIMSE_INGEST_TIMEOUT: int = int(os.getenv("DIMSE_INGEST_TIMEOUT", "30"))
