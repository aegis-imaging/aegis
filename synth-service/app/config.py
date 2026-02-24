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


cfg = Config()
