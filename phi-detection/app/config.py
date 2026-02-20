import os


class Config:
    # Which OCR tool to use.
    # Options: "auto", "tesseract", "google_vision", "aws_textract"
    # "auto" tries each in priority order: google_vision > aws_textract > tesseract
    phi_tool: str = os.environ.get("PHI_TOOL", "auto")

    # Minimum OCR confidence to report a detection (0.0–1.0).
    confidence_threshold: float = float(os.environ.get("PHI_CONFIDENCE_THRESHOLD", "0.4"))

    # Minimum text length to report (filters out OCR noise like single characters).
    min_text_length: int = int(os.environ.get("PHI_MIN_TEXT_LENGTH", "3"))


cfg = Config()
