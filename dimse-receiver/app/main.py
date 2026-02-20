"""DIMSE Receiver — DICOM C-STORE SCP for AEGIS.

Dual-protocol service:
- Port 8080: FastAPI /healthz (health check for Docker + API monitoring)
- Port 11112: DICOM SCP (pynetdicom, started as daemon thread)
"""

from __future__ import annotations

import hmac
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Query, Request
from pydantic import BaseModel, Field

from app import config
from app.ingest import (
    clear_pending,
    clear_pending_study,
    clear_dead_letter,
    clear_dead_letter_study,
    process_retry_queue,
    process_retry_all,
    process_retry_study,
    replay_dead_letter,
    replay_dead_letter_study,
    retry_details,
    retry_snapshot,
)
from app.operator_audit import get_actions, record_action
from app.scp import create_scp, start_scp
from app.sender import forward_study

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
log = logging.getLogger(__name__)

_ae = None
_stop_retry = None
_retry_thread = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Start the DICOM SCP in a daemon thread alongside FastAPI."""
    global _ae, _stop_retry, _retry_thread
    import threading

    _ae = create_scp()
    scp_thread = threading.Thread(target=start_scp, args=(_ae,), daemon=True)
    scp_thread.start()
    log.info("DICOM SCP thread started")

    _stop_retry = threading.Event()

    def _retry_loop() -> None:
        while _stop_retry and not _stop_retry.wait(timeout=config.DIMSE_INGEST_RETRY_INTERVAL):
            process_retry_queue()

    _retry_thread = threading.Thread(target=_retry_loop, daemon=True)
    _retry_thread.start()
    log.info("Ingest retry worker started")

    yield

    # Shutdown
    if _stop_retry is not None:
        _stop_retry.set()
        _stop_retry = None
    if _retry_thread is not None:
        _retry_thread.join(timeout=2)
        _retry_thread = None

    if _ae:
        log.info("Shutting down DICOM SCP")
        _ae.shutdown()
        _ae = None


app = FastAPI(title="AEGIS DIMSE Receiver", lifespan=lifespan)


def _require_operator_key(request: Request) -> None:
    """Require API key for retry-control endpoints when configured."""
    required = config.DIMSE_OPERATOR_API_KEY
    if not required:
        return

    presented = request.headers.get("x-aegis-operator-key", "")
    auth = request.headers.get("authorization", "")
    bearer = auth[7:] if auth.lower().startswith("bearer ") else ""
    if hmac.compare_digest(presented, required) or hmac.compare_digest(bearer, required):
        return

    raise HTTPException(status_code=401, detail="operator auth required")


@app.get("/healthz")
def healthz():
    """Health check endpoint."""
    scp_running = _ae is not None and _ae.active_associations is not None
    retry = retry_snapshot()
    status = "ok" if scp_running and retry["dead_letter"] == 0 else "degraded"
    return {
        "status": status,
        "scp": "running" if scp_running else "not_running",
        "ingest_retry": retry,
    }


@app.get("/ingest/retry")
def ingest_retry_status(request: Request):
    """Return ingest retry queue/dead-letter counters."""
    _require_operator_key(request)
    return {"status": "ok", "ingest_retry": retry_snapshot()}


@app.get("/ingest/retry/actions")
def ingest_retry_actions(request: Request, limit: int = Query(default=100, ge=1, le=10000)):
    """Return recent operator retry-control actions."""
    _require_operator_key(request)
    return {"status": "ok", "actions": get_actions(limit=limit)}


@app.get("/ingest/retry/details")
def ingest_retry_details(request: Request, limit: int = Query(default=100, ge=1, le=10000)):
    """Return detailed pending/dead-letter retry items (capped by limit)."""
    _require_operator_key(request)
    return {"status": "ok", "ingest_retry": retry_details(limit=limit)}


@app.post("/ingest/retry/process")
def ingest_retry_process(request: Request):
    """Run one immediate retry processing pass."""
    _require_operator_key(request)
    processed = process_retry_queue()
    snap = retry_snapshot()
    record_action(
        "retry_process",
        processed=processed,
        pending=snap["pending"],
        dead_letter=snap["dead_letter"],
    )
    return {"status": "ok", "processed": processed, "ingest_retry": snap}


@app.post("/ingest/retry/process-all")
def ingest_retry_process_all(request: Request, limit: int = Query(default=10000, ge=1, le=50000)):
    """Immediately process pending retries up to limit, ignoring schedule."""
    _require_operator_key(request)
    result = process_retry_all(limit=limit)
    record_action(
        "retry_process_all",
        limit=limit,
        processed=result.get("processed", 0),
        ok=result.get("ok", 0),
        requeued=result.get("requeued", 0),
        dead_letter=result.get("dead_letter", 0),
    )
    return {"status": "ok", "ingest_retry": result}


@app.post("/ingest/retry/process/{study_instance_uid}")
def ingest_retry_process_study(request: Request, study_instance_uid: str):
    """Run an immediate retry attempt for one pending study."""
    _require_operator_key(request)
    result = process_retry_study(study_instance_uid=study_instance_uid)
    record_action(
        "retry_process_study",
        study_instance_uid=study_instance_uid,
        found=result.get("found", False),
        result=result.get("result", "unknown"),
    )
    return {"status": "ok", "ingest_retry": result}


@app.post("/ingest/retry/replay")
def ingest_retry_replay(request: Request, limit: int = Query(default=100, ge=1, le=10000)):
    """Replay dead-letter items back into the retry queue."""
    _require_operator_key(request)
    snap = replay_dead_letter(limit=limit)
    record_action(
        "retry_replay_bulk",
        limit=limit,
        replayed_now=snap.get("replayed_now", 0),
        pending=snap["pending"],
        dead_letter=snap["dead_letter"],
    )
    return {"status": "ok", "ingest_retry": snap}


@app.post("/ingest/retry/replay/{study_instance_uid}")
def ingest_retry_replay_study(request: Request, study_instance_uid: str):
    """Replay a specific dead-letter study by StudyInstanceUID."""
    _require_operator_key(request)
    result = replay_dead_letter_study(study_instance_uid=study_instance_uid)
    record_action(
        "retry_replay_study",
        study_instance_uid=study_instance_uid,
        found=result.get("found", False),
        moved=result.get("moved", 0),
        blocked_by_queue_full=result.get("blocked_by_queue_full", False),
    )
    return {"status": "ok", "ingest_retry": result}


@app.post("/ingest/retry/clear-dead-letter")
def ingest_retry_clear_dead_letter(request: Request, limit: int = Query(default=10000, ge=1, le=50000)):
    """Clear dead-letter items after operator acknowledgement."""
    _require_operator_key(request)
    snap = clear_dead_letter(limit=limit)
    record_action(
        "retry_clear_dead_letter",
        limit=limit,
        cleared_now=snap.get("cleared_now", 0),
        dead_letter=snap["dead_letter"],
    )
    return {"status": "ok", "ingest_retry": snap}


@app.post("/ingest/retry/clear-pending")
def ingest_retry_clear_pending(request: Request, limit: int = Query(default=10000, ge=1, le=50000)):
    """Clear pending retry queue items after operator acknowledgement."""
    _require_operator_key(request)
    snap = clear_pending(limit=limit)
    record_action(
        "retry_clear_pending",
        limit=limit,
        cleared_now=snap.get("cleared_now", 0),
        pending=snap["pending"],
    )
    return {"status": "ok", "ingest_retry": snap}


@app.post("/ingest/retry/clear-pending/{study_instance_uid}")
def ingest_retry_clear_pending_study(request: Request, study_instance_uid: str):
    """Clear one pending retry study by StudyInstanceUID."""
    _require_operator_key(request)
    result = clear_pending_study(study_instance_uid=study_instance_uid)
    record_action(
        "retry_clear_pending_study",
        study_instance_uid=study_instance_uid,
        found=result.get("found", False),
        cleared=result.get("cleared", 0),
    )
    return {"status": "ok", "ingest_retry": result}


@app.post("/ingest/retry/clear-dead-letter/{study_instance_uid}")
def ingest_retry_clear_dead_letter_study(request: Request, study_instance_uid: str):
    """Clear one dead-letter study by StudyInstanceUID."""
    _require_operator_key(request)
    result = clear_dead_letter_study(study_instance_uid=study_instance_uid)
    record_action(
        "retry_clear_dead_letter_study",
        study_instance_uid=study_instance_uid,
        found=result.get("found", False),
        cleared=result.get("cleared", 0),
    )
    return {"status": "ok", "ingest_retry": result}


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
