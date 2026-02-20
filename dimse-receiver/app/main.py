"""DIMSE Receiver — DICOM C-STORE SCP for AEGIS.

Dual-protocol service:
- Port 8080: FastAPI /healthz (health check for Docker + API monitoring)
- Port 11112: DICOM SCP (pynetdicom, started as daemon thread)
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from app.scp import create_scp, start_scp
from app.sender import forward_study

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


class DimseDestination(BaseModel):
    ae_title: str = Field(min_length=1)
    host: str = Field(min_length=1)
    port: int = Field(gt=0)


class ForwardRequest(BaseModel):
    study_instance_uid: str = Field(min_length=1)
    dicom_store: str = Field(default="raw", min_length=1)
    destination: DimseDestination


@app.post("/forward")
def forward(request: ForwardRequest):
    """Forward a stored study to a remote DIMSE destination via C-STORE."""
    try:
        result = forward_study(
            study_uid=request.study_instance_uid,
            dicom_store=request.dicom_store,
            host=request.destination.host,
            port=request.destination.port,
            ae_title=request.destination.ae_title,
        )
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=502, detail=str(e))

    return {"status": "complete", **result}
