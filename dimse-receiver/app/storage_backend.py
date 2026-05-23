"""Storage backend for DICOM files: local filesystem or S3.

Local writes go through `dimse_core.storage.StudyLayout` so the receiver and
the spoke router share the same on-disk layout helpers (and pick up
StudyLayout's atomic-write-tmp-swap behavior for free).

On GCP the data directory is gcsfuse-mounted at DIMSE_DATA_DIR so local
writes go straight to GCS — no code change needed there.

On AWS (STORAGE_MODE=s3) we write directly to S3 via boto3 so the same
S3 bucket that the Go API reads from is always the authoritative store.
S3 has no equivalent in dimse-core yet, so that path stays receiver-local.
"""

from __future__ import annotations

import logging

from dimse_core.storage import StudyLayout

from app import config

log = logging.getLogger(__name__)


def _layout(study_uid: str) -> StudyLayout:
    return StudyLayout(data_dir=config.DIMSE_DATA_DIR, study_instance_uid=study_uid)


def write_dicom(study_uid: str, file_index: int, dicom_bytes: bytes) -> str:
    """Write a DICOM file to the configured backend.

    Returns the storage key (e.g. ``dicom/raw/{studyUID}/{index}.dcm``).
    """
    key = f"dicom/raw/{study_uid}/{file_index}.dcm"

    if config.STORAGE_MODE == "s3":
        import boto3

        s3 = boto3.client("s3", region_name=config.S3_REGION)
        s3.put_object(Bucket=config.S3_BUCKET, Key=key, Body=dicom_bytes)
        log.debug("Wrote file %d for study %s to s3://%s/%s", file_index, study_uid, config.S3_BUCKET, key)
    else:
        target = _layout(study_uid).write_raw(file_index, dicom_bytes)
        log.debug("Wrote file %d for study %s to %s", file_index, study_uid, target)

    return key


def next_file_index(study_uid: str) -> int:
    """Return the next 0-based file index for a study (counts existing files)."""
    if config.STORAGE_MODE == "s3":
        import boto3

        s3 = boto3.client("s3", region_name=config.S3_REGION)
        prefix = f"dicom/raw/{study_uid}/"
        resp = s3.list_objects_v2(Bucket=config.S3_BUCKET, Prefix=prefix)
        return len(resp.get("Contents", []))

    return _layout(study_uid).next_raw_index()
