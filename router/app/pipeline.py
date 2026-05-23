"""Per-study orchestrator: de-identify → ship → quarantine on failure.

Called both by the DIMSE SCP (after EVT_RELEASED) and by the HTTPS upload
endpoint (after the user clicks "Complete"). Idempotent on study UID — if the
same study is processed twice (e.g. PACS re-sends), we re-run de-id and
re-attempt shipment.
"""

from __future__ import annotations

import logging
import threading
import time
from dataclasses import dataclass
from typing import Any

from app import deid, metrics, quarantine, shipper, storage
from app.audit import AuditLog
from app.config import RouterConfig

log = logging.getLogger(__name__)


@dataclass
class PipelineResult:
    study_uid: str
    ok: bool
    stages_run: list[str]
    cloud_study_id: str = ""
    error: str = ""
    quarantined: bool = False


class Orchestrator:
    """Wraps the per-study state and dependencies. One instance per process."""

    def __init__(
        self,
        cfg: RouterConfig,
        store: quarantine.QuarantineStore,
        audit: AuditLog,
    ) -> None:
        self.cfg = cfg
        self.store = store
        self.audit = audit
        self._inflight: set[str] = set()
        self._inflight_lock = threading.Lock()

    def submit(self, study_uid: str, file_count: int = 0, source: str = "dimse") -> None:
        """Kick off a background pipeline run. Returns immediately."""
        with self._inflight_lock:
            if study_uid in self._inflight:
                log.info("study %s already in-flight; skipping duplicate submit", study_uid)
                return
            self._inflight.add(study_uid)

        t = threading.Thread(
            target=self._run,
            args=(study_uid, file_count, source),
            name=f"pipeline-{study_uid[:12]}",
            daemon=True,
        )
        t.start()

    def run_sync(self, study_uid: str, file_count: int = 0, source: str = "dimse") -> PipelineResult:
        """Run the pipeline inline (used by tests and the dev /process endpoint)."""
        with self._inflight_lock:
            self._inflight.add(study_uid)
        try:
            return self._run(study_uid, file_count, source)
        finally:
            with self._inflight_lock:
                self._inflight.discard(study_uid)

    def _run(self, study_uid: str, file_count: int, source: str) -> PipelineResult:
        try:
            return self._do_run(study_uid, file_count, source)
        finally:
            with self._inflight_lock:
                self._inflight.discard(study_uid)

    def _do_run(self, study_uid: str, file_count: int, source: str) -> PipelineResult:
        paths = storage.layout(self.cfg.data_dir, study_uid)
        self.audit.record("pipeline.start", study_uid=study_uid, source=source, file_count=file_count)
        m = metrics.get()
        m.inc("aegis_router_studies_received_total", {"source": _source_label(source)})
        started = time.time()

        # 1. De-identify
        deid_result = deid.run_pipeline(self.cfg, paths)
        m.observe("aegis_router_deid_duration_seconds", deid_result.duration_seconds)
        if not deid_result.ok:
            m.inc(
                "aegis_router_studies_quarantined_total",
                {"stage": deid_result.failed_stage or "deid"},
            )
            self._quarantine(
                paths,
                study_uid,
                reason=f"de-id failed at {deid_result.failed_stage}",
                stage=deid_result.failed_stage,
                error=deid_result.error,
                file_count=file_count,
                detail={"stages_run": deid_result.stages_run},
            )
            return PipelineResult(
                study_uid=study_uid,
                ok=False,
                stages_run=deid_result.stages_run,
                error=deid_result.error,
                quarantined=True,
            )

        # 2. Ship if cloud forwarding configured
        if not self.cfg.cloud_forwarding_configured():
            duration = time.time() - started
            m.observe("aegis_router_pipeline_duration_seconds", duration, {"outcome": "local_only"})
            m.inc("aegis_router_studies_local_only_total")
            self.audit.record(
                "pipeline.complete_local_only",
                study_uid=study_uid,
                stages=deid_result.stages_run,
                duration_sec=duration,
            )
            return PipelineResult(
                study_uid=study_uid,
                ok=True,
                stages_run=deid_result.stages_run,
                cloud_study_id="",
            )

        metadata = deid.summarize_study(paths)
        try:
            ship_result = shipper.ship_study(self.cfg, paths, metadata)
        except shipper.ShipperError as e:
            m.inc("aegis_router_studies_quarantined_total", {"stage": "ship"})
            self._quarantine(
                paths,
                study_uid,
                reason="cloud ship failed",
                stage="ship",
                error=str(e),
                file_count=file_count,
                detail={"stages_run": deid_result.stages_run},
            )
            return PipelineResult(
                study_uid=study_uid,
                ok=False,
                stages_run=deid_result.stages_run,
                error=str(e),
                quarantined=True,
            )

        duration = time.time() - started
        m.observe("aegis_router_pipeline_duration_seconds", duration, {"outcome": "shipped"})
        m.inc("aegis_router_studies_shipped_total")
        self.audit.record(
            "pipeline.shipped",
            study_uid=study_uid,
            stages=deid_result.stages_run,
            cloud_study_id=ship_result.cloud_study_id,
            files=ship_result.files_uploaded,
            duration_sec=duration,
        )
        return PipelineResult(
            study_uid=study_uid,
            ok=True,
            stages_run=deid_result.stages_run + ["ship"],
            cloud_study_id=ship_result.cloud_study_id,
        )

    def _quarantine(
        self,
        paths: storage.StudyPaths,
        study_uid: str,
        *,
        reason: str,
        stage: str,
        error: str,
        file_count: int,
        detail: dict[str, Any],
    ) -> None:
        self.store.quarantine(
            study_uid=study_uid,
            reason=reason,
            failed_stage=stage,
            error=error,
            detail=detail,
            file_count=file_count,
        )
        self.audit.record(
            "pipeline.quarantined",
            study_uid=study_uid,
            stage=stage,
            reason=reason,
            error=error,
        )
        entry = self.store.get(study_uid)
        if entry is not None:
            quarantine.notify_cloud(self.cfg.quarantine_alert_url, entry, self.cfg.site_id)


def _source_label(s: str) -> str:
    # Reduce free-form sources (e.g. "dimse(SOMEPACS)") to a small label set so
    # Prometheus cardinality stays bounded.
    if s.startswith("dimse"):
        return "dimse"
    if s.startswith("web") or s == "web_upload":
        return "web_upload"
    if s == "operator_retry":
        return "operator_retry"
    return "other"
