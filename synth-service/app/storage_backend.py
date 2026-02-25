"""Storage backend for synth DICOM files: local filesystem or S3.

On GCP, synth-service's data dir is GCS-FUSE mounted — local writes go
straight to GCS automatically.  No upload step needed.

On AWS (STORAGE_MODE=s3), generated files are uploaded to S3 under
synth/{studyUID}/ so the Go API's importSynthStudy can find them via
storage.List("synth/{studyUID}").  Local temp files are removed after upload.
"""

from __future__ import annotations

import logging
import shutil
from pathlib import Path

from .config import cfg

log = logging.getLogger(__name__)


def upload_synth_study(study_uid: str, local_paths: list[str]) -> None:
    """Upload locally generated synth DICOM files to S3 under synth/{studyUID}/.

    No-op when cfg.storage_mode != "s3" (local filesystem / GCS-FUSE path).
    Cleans up local temp files after a successful S3 upload.
    """
    if cfg.storage_mode != "s3":
        return

    import boto3

    s3 = boto3.client("s3", region_name=cfg.s3_region)
    for local_path in local_paths:
        filename = Path(local_path).name
        key = f"synth/{study_uid}/{filename}"
        with open(local_path, "rb") as f:
            s3.put_object(Bucket=cfg.s3_bucket, Key=key, Body=f.read())
        log.debug("Uploaded %s to s3://%s/%s", filename, cfg.s3_bucket, key)

    log.info(
        "Uploaded %d synth files for study %s to s3://%s/synth/%s/",
        len(local_paths),
        study_uid,
        cfg.s3_bucket,
        study_uid,
    )

    # Clean up local temp files — S3 is now the authoritative store.
    if local_paths:
        study_dir = Path(local_paths[0]).parent
        if study_dir.exists():
            shutil.rmtree(study_dir, ignore_errors=True)
            log.debug("Cleaned up local synth dir %s", study_dir)
