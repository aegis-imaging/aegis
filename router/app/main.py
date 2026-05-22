"""AEGIS Router — FastAPI app + DICOM SCP thread + pipeline orchestration.

Endpoints:
  GET  /healthz                  health + sidecar status
  GET  /info                     site identity + enabled capabilities

  POST /api/upload/init          web upload — start a session
  PUT  /api/upload/file/{sid}/{i} web upload — one DICOM
  POST /api/upload/complete      web upload — finalize → trigger pipeline

  GET  /quarantine               operator: list quarantined studies
  GET  /quarantine/{study_uid}   operator: inspect
  POST /quarantine/{study_uid}/retry    operator: retry pipeline
  DELETE /quarantine/{study_uid}        operator: discard

  GET  /audit                    recent pipeline+operator events
"""

from __future__ import annotations

import hmac
import logging
import threading
import uuid
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any

from fastapi import Depends, FastAPI, HTTPException, Path as PathParam, Query, Request, UploadFile
from pydantic import BaseModel, Field

from app import metrics as metrics_mod
from app import quarantine as quarantine_mod
from app import scp as scp_mod
from app import storage
from app.audit import AuditLog
from app.config import RouterConfig
from app.pipeline import Orchestrator

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
log = logging.getLogger(__name__)


# ── Module-level state set up in lifespan ──────────────────────────────────────

_cfg: RouterConfig | None = None
_store: quarantine_mod.QuarantineStore | None = None
_audit: AuditLog | None = None
_orchestrator: Orchestrator | None = None
_ae = None  # pynetdicom AE
_sessions: dict[str, "UploadSession"] = {}
_sessions_lock = threading.Lock()


class UploadSession:
    """In-memory web upload session. Files are staged under
    {data_dir}/uploads/<sid>/ and moved into the study layout on /complete.
    """

    def __init__(self, cfg: RouterConfig, sid: str, file_count: int, metadata: dict[str, Any]) -> None:
        self.sid = sid
        self.file_count = file_count
        self.metadata = metadata
        self.received: dict[int, Path] = {}
        self.staging = Path(cfg.data_dir) / "uploads" / sid
        self.staging.mkdir(parents=True, exist_ok=True)


@asynccontextmanager
async def lifespan(app: FastAPI):
    global _cfg, _store, _audit, _orchestrator, _ae
    _cfg = RouterConfig.load()
    Path(_cfg.data_dir).mkdir(parents=True, exist_ok=True)
    Path(_cfg.quarantine_dir).mkdir(parents=True, exist_ok=True)

    metrics_mod.init()
    _store = quarantine_mod.QuarantineStore(quarantine_mod.default_db_path(_cfg.quarantine_dir))
    _audit = AuditLog(str(Path(_cfg.data_dir) / "audit.log"))
    _orchestrator = Orchestrator(_cfg, _store, _audit)

    # DICOM SCP in a daemon thread.
    def _on_study_complete(acc: scp_mod.StudyAccumulator) -> None:
        # Defensive nil-check for typecheckers; _orchestrator is set above.
        assert _orchestrator is not None
        _orchestrator.submit(
            acc.study_instance_uid,
            file_count=acc.file_count,
            source=f"dimse({acc.calling_ae_title})",
        )

    _ae = scp_mod.create_ae(_cfg, _on_study_complete)
    t = threading.Thread(target=scp_mod.start_listening, args=(_ae, _cfg), daemon=True, name="dimse-scp")
    t.start()
    log.info(
        "router up: site=%s ae=%s dimse_port=%d http_port=%d sidecars=%s cloud_forwarding=%s",
        _cfg.site_name,
        _cfg.dimse_ae_title,
        _cfg.dimse_port,
        _cfg.http_port,
        list(_cfg.enabled_sidecars().keys()),
        _cfg.cloud_forwarding_configured(),
    )
    _audit.record("router.start", site=_cfg.site_name)

    yield

    log.info("router shutting down")
    if _ae is not None:
        try:
            _ae.shutdown()
        except Exception:
            log.exception("AE shutdown failed")


app = FastAPI(title="AEGIS Router", lifespan=lifespan)


def _require_cfg() -> RouterConfig:
    if _cfg is None:
        raise HTTPException(status_code=503, detail="router not yet initialised")
    return _cfg


def _require_orchestrator() -> Orchestrator:
    if _orchestrator is None:
        raise HTTPException(status_code=503, detail="router not yet initialised")
    return _orchestrator


def _require_store() -> quarantine_mod.QuarantineStore:
    if _store is None:
        raise HTTPException(status_code=503, detail="router not yet initialised")
    return _store


def _require_audit() -> AuditLog:
    if _audit is None:
        raise HTTPException(status_code=503, detail="router not yet initialised")
    return _audit


def _require_operator(request: Request, cfg: RouterConfig = Depends(_require_cfg)) -> None:
    if not cfg.operator_api_key:
        return
    presented = request.headers.get("x-aegis-operator-key", "")
    auth = request.headers.get("authorization", "")
    bearer = auth[7:] if auth.lower().startswith("bearer ") else ""
    if hmac.compare_digest(presented, cfg.operator_api_key) or hmac.compare_digest(bearer, cfg.operator_api_key):
        return
    raise HTTPException(status_code=401, detail="operator auth required")


# ── Health / info ────────────────────────────────────────────────────────────


@app.get("/healthz")
@app.get("/health")
def healthz(cfg: RouterConfig = Depends(_require_cfg)) -> dict[str, Any]:
    scp_running = _ae is not None
    return {
        "status": "ok" if scp_running else "degraded",
        "scp": "running" if scp_running else "not_running",
        "sidecars": list(cfg.enabled_sidecars().keys()),
        "cloud_forwarding": cfg.cloud_forwarding_configured(),
    }


@app.get("/metrics")
def metrics_endpoint() -> Any:
    """Prometheus-text-format metrics. No auth — typical pattern for an
    internal scrape endpoint. Front with a reverse proxy or firewall if
    you're exposing the router to the open internet."""
    from fastapi.responses import PlainTextResponse

    return PlainTextResponse(metrics_mod.get().render(), media_type="text/plain; version=0.0.4")


@app.get("/info")
def info(cfg: RouterConfig = Depends(_require_cfg)) -> dict[str, Any]:
    return {
        "site_id": cfg.site_id,
        "site_name": cfg.site_name,
        "dimse_ae_title": cfg.dimse_ae_title,
        "dimse_port": cfg.dimse_port,
        "enabled_sidecars": list(cfg.enabled_sidecars().keys()),
        "cloud_forwarding": cfg.cloud_forwarding_configured(),
        "cloud_url": cfg.cloud_url if cfg.cloud_forwarding_configured() else "",
    }


# ── Web upload (mirrors the cloud upload-portal contract) ────────────────────


class UploadInitRequest(BaseModel):
    file_count: int = Field(gt=0, le=20000)
    study_metadata: dict[str, Any] = Field(default_factory=dict)


@app.post("/api/upload/init")
def upload_init(req: UploadInitRequest, cfg: RouterConfig = Depends(_require_cfg)) -> dict[str, Any]:
    sid = uuid.uuid4().hex
    session = UploadSession(cfg, sid, req.file_count, req.study_metadata)
    with _sessions_lock:
        _sessions[sid] = session
    # Each PUT URL is relative to the router; the cloud upload-portal client
    # already understands this shape from the cloud's identical endpoint.
    upload_urls = [f"/api/upload/file/{sid}/{i}" for i in range(req.file_count)]
    return {
        "session_id": sid,
        "upload_urls": upload_urls,
        "expires_in_seconds": 3600,
    }


@app.put("/api/upload/file/{session_id}/{index}")
async def upload_file(session_id: str, index: int, request: Request) -> dict[str, Any]:
    with _sessions_lock:
        session = _sessions.get(session_id)
    if session is None:
        raise HTTPException(status_code=404, detail="upload session not found or expired")
    if index < 0 or index >= session.file_count:
        raise HTTPException(status_code=400, detail="file index out of range")

    body = await request.body()
    if not body:
        raise HTTPException(status_code=400, detail="empty file body")
    target = session.staging / f"{index}.dcm"
    target.write_bytes(body)
    session.received[index] = target
    return {"status": "ok", "index": index, "bytes": len(body)}


class UploadCompleteRequest(BaseModel):
    session_id: str


@app.post("/api/upload/complete")
def upload_complete(
    req: UploadCompleteRequest,
    cfg: RouterConfig = Depends(_require_cfg),
    orchestrator: Orchestrator = Depends(_require_orchestrator),
) -> dict[str, Any]:
    with _sessions_lock:
        session = _sessions.pop(req.session_id, None)
    if session is None:
        raise HTTPException(status_code=404, detail="upload session not found or expired")
    if len(session.received) != session.file_count:
        raise HTTPException(
            status_code=400,
            detail=f"incomplete upload: {len(session.received)}/{session.file_count} files received",
        )

    # Identify study UID from the first uploaded file.
    import pydicom

    first = sorted(session.received.items(), key=lambda kv: kv[0])[0][1]
    try:
        ds = pydicom.dcmread(str(first), stop_before_pixels=True)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"first file is not a valid DICOM: {e}")
    study_uid = str(getattr(ds, "StudyInstanceUID", "")).strip()
    if not study_uid:
        raise HTTPException(status_code=400, detail="StudyInstanceUID missing from uploaded DICOM")

    paths = storage.layout(cfg.data_dir, study_uid)
    for idx, src in sorted(session.received.items()):
        next_idx = storage.next_raw_index(paths)
        storage.write_raw_bytes(paths, next_idx, src.read_bytes())

    orchestrator.submit(study_uid, file_count=session.file_count, source="web_upload")
    return {"status": "accepted", "study_instance_uid": study_uid, "files": session.file_count}


# ── Quarantine operator API ──────────────────────────────────────────────────


@app.get("/quarantine")
def list_quarantine(
    limit: int = Query(default=100, ge=1, le=1000),
    _: None = Depends(_require_operator),
    store: quarantine_mod.QuarantineStore = Depends(_require_store),
) -> dict[str, Any]:
    items = [e.to_json() for e in store.list(limit=limit)]
    return {"items": items, "count": len(items)}


@app.get("/quarantine/{study_uid}")
def get_quarantine(
    study_uid: str = PathParam(...),
    _: None = Depends(_require_operator),
    store: quarantine_mod.QuarantineStore = Depends(_require_store),
) -> dict[str, Any]:
    entry = store.get(study_uid)
    if entry is None:
        raise HTTPException(status_code=404, detail="not in quarantine")
    return entry.to_json()


@app.post("/quarantine/{study_uid}/retry")
def retry_quarantine(
    study_uid: str = PathParam(...),
    _: None = Depends(_require_operator),
    store: quarantine_mod.QuarantineStore = Depends(_require_store),
    orchestrator: Orchestrator = Depends(_require_orchestrator),
) -> dict[str, Any]:
    entry = store.get(study_uid)
    if entry is None:
        raise HTTPException(status_code=404, detail="not in quarantine")
    store.mark_attempted(study_uid)
    # Clear before submit so a successful retry isn't double-tracked. If the
    # pipeline fails again it re-quarantines.
    store.clear(study_uid)
    orchestrator.submit(study_uid, file_count=entry.file_count, source="operator_retry")
    return {"status": "retry_submitted", "study_uid": study_uid}


@app.delete("/quarantine/{study_uid}")
def clear_quarantine(
    study_uid: str = PathParam(...),
    _: None = Depends(_require_operator),
    store: quarantine_mod.QuarantineStore = Depends(_require_store),
    audit: AuditLog = Depends(_require_audit),
) -> dict[str, Any]:
    if not store.clear(study_uid):
        raise HTTPException(status_code=404, detail="not in quarantine")
    audit.record("quarantine.cleared", study_uid=study_uid)
    return {"status": "cleared", "study_uid": study_uid}


# ── Audit ────────────────────────────────────────────────────────────────────


@app.get("/audit")
def get_audit(
    limit: int = Query(default=100, ge=1, le=2000),
    event: str = Query(default=""),
    _: None = Depends(_require_operator),
    audit: AuditLog = Depends(_require_audit),
) -> dict[str, Any]:
    return {"entries": audit.recent(limit=limit, event=event.strip())}
