import os


class Config:
    # Which QC backend to use. Options: "auto", "basic"
    # "auto" tries each in priority order and uses the first available.
    # Future: add "vertex_ai" for production ML-based QC.
    qc_tool: str = os.environ.get("QC_TOOL", "auto")

    # Minimum acceptable signal-to-noise ratio.
    snr_threshold: float = float(os.environ.get("QC_SNR_THRESHOLD", "10.0"))

    # Maximum allowed gap ratio for missing slice detection.
    # If a gap between consecutive slices exceeds this multiple of the median
    # spacing, it is flagged as a missing slice.
    gap_ratio_threshold: float = float(os.environ.get("QC_GAP_RATIO", "2.0"))


cfg = Config()
