"""Storage backend for synth DICOM files: local filesystem, S3, or Azure Blob Storage.

On GCP, synth-service's data dir is GCS-FUSE mounted — local writes go
straight to GCS automatically.  No upload step needed.

On AWS (STORAGE_MODE=s3), generated files are uploaded to S3 under
synth/{studyUID}/ so the Go API's importSynthStudy can find them via
storage.List("synth/{studyUID}").  Local temp files are removed after upload.

On Azure (STORAGE_MODE=azure), generated files are uploaded to Azure Blob Storage
under synth/{studyUID}/ using the same convention.  Local temp files are removed
after upload.
"""

from __future__ import annotations

import logging
import shutil
from pathlib import Path

from .config import cfg

log = logging.getLogger(__name__)


def upload_synth_study(study_uid: str, local_paths: list[str]) -> None:
    """Upload locally generated synth DICOM files to cloud storage under synth/{studyUID}/.

    - STORAGE_MODE=s3: uploads to S3, cleans up local temp files.
    - STORAGE_MODE=azure: uploads to Azure Blob Storage, cleans up local temp files.
    - Otherwise: no-op (local filesystem / GCS-FUSE handles it automatically).
    """
    if cfg.storage_mode == "s3":
        _upload_s3(study_uid, local_paths)
    elif cfg.storage_mode == "azure":
        _upload_azure(study_uid, local_paths)


def _upload_s3(study_uid: str, local_paths: list[str]) -> None:
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
    _cleanup_local(local_paths)


def _upload_azure(study_uid: str, local_paths: list[str]) -> None:
    from azure.identity import DefaultAzureCredential
    from azure.storage.blob import BlobServiceClient

    account = cfg.azure_storage_account
    container = cfg.azure_storage_container
    url = f"https://{account}.blob.core.windows.net"
    cred = DefaultAzureCredential()
    client = BlobServiceClient(url, credential=cred)
    cc = client.get_container_client(container)

    for local_path in local_paths:
        filename = Path(local_path).name
        key = f"synth/{study_uid}/{filename}"
        with open(local_path, "rb") as f:
            cc.upload_blob(key, f, overwrite=True)
        log.debug("Uploaded %s to azure://%s/%s/%s", filename, account, container, key)

    log.info(
        "Uploaded %d synth files for study %s to azure://%s/%s/synth/%s/",
        len(local_paths),
        study_uid,
        account,
        container,
        study_uid,
    )

    # Clean up local temp files — Azure Blob is now the authoritative store.
    _cleanup_local(local_paths)


def _cleanup_local(local_paths: list[str]) -> None:
    if local_paths:
        study_dir = Path(local_paths[0]).parent
        if study_dir.exists():
            shutil.rmtree(study_dir, ignore_errors=True)
            log.debug("Cleaned up local synth dir %s", study_dir)
