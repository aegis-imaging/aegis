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
DIMSE_INSTITUTION_SLUG: str = os.getenv("DIMSE_INSTITUTION_SLUG", "")

# HTTP client settings
DIMSE_INGEST_TIMEOUT: int = int(os.getenv("DIMSE_INGEST_TIMEOUT", "30"))
DIMSE_INGEST_RETRY_INTERVAL: int = int(os.getenv("DIMSE_INGEST_RETRY_INTERVAL", "15"))
DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER: float = float(
    os.getenv("DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER", "2.0")
)
DIMSE_INGEST_RETRY_MAX_INTERVAL: int = int(os.getenv("DIMSE_INGEST_RETRY_MAX_INTERVAL", "300"))
DIMSE_INGEST_MAX_ATTEMPTS: int = int(os.getenv("DIMSE_INGEST_MAX_ATTEMPTS", "5"))
DIMSE_INGEST_QUEUE_MAX: int = int(os.getenv("DIMSE_INGEST_QUEUE_MAX", "1000"))
# Optional /healthz degradation threshold for stale pending retries (0 disables)
DIMSE_INGEST_PENDING_AGE_WARN_SECONDS: int = int(os.getenv("DIMSE_INGEST_PENDING_AGE_WARN_SECONDS", "0"))

# In-memory operator action audit (retry control endpoints)
DIMSE_OPERATOR_AUDIT_MAX: int = int(os.getenv("DIMSE_OPERATOR_AUDIT_MAX", "500"))
# Optional API key for retry-control endpoints (/ingest/retry*)
DIMSE_OPERATOR_API_KEY: str = os.getenv("DIMSE_OPERATOR_API_KEY", "")
