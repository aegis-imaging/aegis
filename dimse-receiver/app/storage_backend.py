"""Storage backend for DICOM files: local filesystem or S3.

On GCP the data directory is gcsfuse-mounted at DIMSE_DATA_DIR so local
writes go straight to GCS — no code change needed there.

On AWS (STORAGE_MODE=s3) we write directly to S3 via boto3 so the same
S3 bucket that the Go API reads from is always the authoritative store.
"""

from __future__ import annotations

import logging
import re
from pathlib import Path

from app import config

log = logging.getLogger(__name__)

# DICOM UIDs are dot-separated numeric components, max 64 characters
# (PS3.5 §9.1) — safe to use verbatim as filenames.
_UID_RE = re.compile(r"^[0-9.]{1,64}$")


def validate_uid(uid: str) -> str:
    """Return *uid* if it is a valid DICOM UID, else raise ValueError.

    Guards against empty/oversized values and anything that is not
    digits+dots so UIDs can be used safely as path components.
    """
    if not _UID_RE.match(uid):
        raise ValueError(f"invalid DICOM UID for storage key: {uid!r}")
    return uid


def write_dicom(study_uid: str, sop_instance_uid: str, dicom_bytes: bytes) -> str:
    """Write a DICOM file to the configured backend.

    Files are named by SOP Instance UID so concurrent associations
    receiving the same study can never collide, and re-sent instances
    overwrite themselves idempotently (correct DICOM semantics).

    Returns the storage key (e.g. ``dicom/raw/{studyUID}/{sopInstanceUID}.dcm``).
    """
    key = f"dicom/raw/{study_uid}/{validate_uid(sop_instance_uid)}.dcm"

    if config.STORAGE_MODE == "s3":
        import boto3

        s3 = boto3.client("s3", region_name=config.S3_REGION)
        s3.put_object(Bucket=config.S3_BUCKET, Key=key, Body=dicom_bytes)
        log.debug("Wrote instance %s for study %s to s3://%s/%s", sop_instance_uid, study_uid, config.S3_BUCKET, key)
    else:
        # Local filesystem — works for dev (local path) and GCP (gcsfuse mount).
        path = Path(config.DIMSE_DATA_DIR) / key
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(dicom_bytes)
        log.debug("Wrote instance %s for study %s to %s", sop_instance_uid, study_uid, path)

    return key
