import os


class Config:
    # Path to the shared DICOM data volume.
    data_dir: str = os.environ.get("SYNTH_DATA_DIR", "/app/data")

    # Enable GPU-backed MONAI LDM generation (requires INCLUDE_MONAI=true build arg).
    use_gpu: bool = os.environ.get("SYNTH_USE_GPU", "false").lower() == "true"

    # Enable Vertex AI Imagen generation (requires INCLUDE_GEMINI=true build arg).
    use_gemini: bool = os.environ.get("SYNTH_USE_GEMINI", "false").lower() == "true"

    # Generation limits (prevent runaway requests).
    max_slices: int = int(os.environ.get("SYNTH_MAX_SLICES", "200"))
    max_size: int = int(os.environ.get("SYNTH_MAX_SIZE", "512"))

    # Storage backend: "local" (filesystem / GCS-FUSE), "s3" (AWS S3), or "azure" (Azure Blob).
    # On GCP, data_dir is GCS-FUSE mounted so local writes go to GCS automatically.
    # On AWS ECS, set STORAGE_MODE=s3 so generated files are uploaded to S3 where
    # the Go API can find them via storage.List("synth/{studyUID}").
    # On Azure Container Apps, set STORAGE_MODE=azure for Azure Blob Storage upload.
    storage_mode: str = os.environ.get("STORAGE_MODE", "local")
    s3_bucket: str = os.environ.get("S3_BUCKET", "")
    s3_region: str = os.environ.get("S3_REGION", "us-east-1")
    azure_storage_account: str = os.environ.get("AZURE_STORAGE_ACCOUNT", "")
    azure_storage_container: str = os.environ.get("AZURE_STORAGE_CONTAINER", "dicom")


cfg = Config()
