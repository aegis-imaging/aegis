"""mTLS HTTPS forwarder: ship a de-identified study to the cloud receiver.

Wire format mirrors the cloud's existing upload-portal three-step flow:
  POST /api/upload/init          → returns sessionID + per-file PUT URLs
  PUT  /api/upload/file/{sid}/i  → upload each de-identified DICOM
  POST /api/upload/complete      → cloud ingests the staged files, runs cloud-side pipeline

We authenticate every request with a client certificate (mTLS) issued at spoke
onboarding. The cloud side identifies the spoke by the cert's thumbprint and
attaches the resulting study to the spoke's institution.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import httpx

from app import storage
from app.config import RouterConfig

log = logging.getLogger(__name__)


class ShipperError(RuntimeError):
    """Raised when shipping fails. Caller is responsible for retry policy."""


@dataclass
class ShipResult:
    cloud_session_id: str
    cloud_study_id: str
    files_uploaded: int


def ship_study(
    cfg: RouterConfig,
    paths: storage.StudyPaths,
    study_metadata: dict[str, Any],
) -> ShipResult:
    if not cfg.cloud_forwarding_configured():
        raise ShipperError("cloud forwarding not configured (CLOUD_RECEIVER_URL + client cert/key required)")

    files = storage.list_deid_files(paths)
    if not files:
        raise ShipperError(f"no de-identified files to ship for {paths.study_uid}")

    client_kwargs: dict[str, Any] = {
        "timeout": httpx.Timeout(60.0, read=600.0),
        "cert": (cfg.cloud_client_cert, cfg.cloud_client_key),
        "verify": cfg.cloud_ca_cert or True,
        "base_url": cfg.cloud_url.rstrip("/"),
        "headers": {
            "X-Aegis-Spoke-Site": cfg.site_id or cfg.site_name,
        },
    }

    with httpx.Client(**client_kwargs) as client:
        init_payload = {
            "project_slug": cfg.cloud_project_slug,
            "institution_slug": cfg.cloud_institution_slug,
            "file_count": len(files),
            "study_metadata": study_metadata,
            "source": "spoke",
        }
        r = client.post("/api/upload/init", json=init_payload)
        if r.status_code >= 400:
            raise ShipperError(f"upload/init failed: HTTP {r.status_code}: {r.text[:300]}")
        init = r.json()
        session_id = init.get("session_id") or init.get("sessionID")
        upload_urls = init.get("upload_urls") or init.get("uploadURLs") or []
        if not session_id:
            raise ShipperError(f"upload/init missing session_id in response: {init!r}")
        if len(upload_urls) != len(files):
            raise ShipperError(
                f"upload/init returned {len(upload_urls)} URLs but study has {len(files)} files"
            )

        for idx, (file_path, url) in enumerate(zip(files, upload_urls)):
            _put_file(client, url, file_path)

        complete_payload = {"session_id": session_id}
        r = client.post("/api/upload/complete", json=complete_payload)
        if r.status_code >= 400:
            raise ShipperError(f"upload/complete failed: HTTP {r.status_code}: {r.text[:300]}")
        done = r.json()

    return ShipResult(
        cloud_session_id=session_id,
        cloud_study_id=str(done.get("study_id") or done.get("studyID") or ""),
        files_uploaded=len(files),
    )


def _put_file(client: httpx.Client, url: str, file_path: Path) -> None:
    """PUT one file. We allow url to be absolute (signed S3/GCS) or relative
    (when the cloud is in local-storage dev mode)."""
    data = file_path.read_bytes()
    headers = {"Content-Type": "application/dicom"}
    if url.startswith("http://") or url.startswith("https://"):
        # Absolute (signed) URL — issue a one-off request that bypasses base_url.
        with httpx.Client(timeout=600.0) as raw:
            r = raw.put(url, content=data, headers=headers)
    else:
        r = client.put(url, content=data, headers=headers)
    if r.status_code >= 400:
        raise ShipperError(f"file PUT failed for {file_path.name}: HTTP {r.status_code}: {r.text[:300]}")
