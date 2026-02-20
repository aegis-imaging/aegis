"""DIMSE Receiver — DICOM C-STORE SCP for AEGIS.

Dual-protocol service:
- Port 8080: FastAPI /healthz (health check for Docker + API monitoring)
- Port 11112: DICOM SCP (pynetdicom, started as daemon thread)
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.scp import create_scp, start_scp

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
log = logging.getLogger(__name__)

_ae = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Start the DICOM SCP in a daemon thread alongside FastAPI."""
    global _ae
    import threading

    _ae = create_scp()
    scp_thread = threading.Thread(target=start_scp, args=(_ae,), daemon=True)
    scp_thread.start()
    log.info("DICOM SCP thread started")

    yield

    # Shutdown
    if _ae:
        log.info("Shutting down DICOM SCP")
        _ae.shutdown()
        _ae = None


app = FastAPI(title="AEGIS DIMSE Receiver", lifespan=lifespan)


@app.get("/healthz")
def healthz():
    """Health check endpoint."""
    scp_running = _ae is not None and _ae.active_associations is not None
    if scp_running:
        return {"status": "ok", "scp": "running"}
    return {"status": "degraded", "scp": "not_running"}
