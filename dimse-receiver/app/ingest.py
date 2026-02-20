"""HTTP client for calling the Go API ingest endpoint."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field

import httpx

from app import config

log = logging.getLogger(__name__)


@dataclass
class StudyAccumulator:
    """Tracks metadata for a single study during a DICOM association."""

    study_instance_uid: str = ""
    modality: str = ""
    body_part: str = ""
    study_description: str = ""
    series_uids: set = field(default_factory=set)
    file_count: int = 0
    calling_ae_title: str = ""


def trigger_ingest(acc: StudyAccumulator) -> bool:
    """POST study metadata to the Go API ingest endpoint.

    Returns True on success (HTTP 2xx), False otherwise.
    """
    url = f"{config.API_URL}/api/ingest"
    payload = {
        "project_slug": config.DIMSE_PROJECT_SLUG,
        "study_metadata": {
            "study_instance_uid": acc.study_instance_uid,
            "modality": acc.modality,
            "body_part": acc.body_part,
            "study_description": acc.study_description,
            "series_count": len(acc.series_uids),
            "instance_count": acc.file_count,
        },
    }

    try:
        with httpx.Client(timeout=config.DIMSE_INGEST_TIMEOUT) as client:
            resp = client.post(url, json=payload)
        if resp.status_code < 300:
            log.info(
                "Ingest OK for %s (%d files, %d series)",
                acc.study_instance_uid,
                acc.file_count,
                len(acc.series_uids),
            )
            return True
        log.error(
            "Ingest failed for %s: HTTP %d — %s",
            acc.study_instance_uid,
            resp.status_code,
            resp.text[:200],
        )
        return False
    except Exception as e:
        log.error("Ingest request failed for %s: %s", acc.study_instance_uid, e)
        return False
