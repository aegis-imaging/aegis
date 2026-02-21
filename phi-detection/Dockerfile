# AEGIS PHI Detection Service
#
# Backend is controlled by PHI_TOOL env var:
#   "auto" (default): google_vision > aws_textract > tesseract
#   "google_vision":  Google Cloud Vision text_detection
#   "aws_textract":   AWS Textract detect_document_text
#   "tesseract":      local Tesseract OCR (no cloud dependencies)
#
# Cloud backends are optional — build with:
#   --build-arg INCLUDE_GOOGLE_VISION=true
#   --build-arg INCLUDE_AWS_TEXTRACT=true

FROM python:3.12-slim

# Install tesseract-ocr (always included — offline fallback).
RUN apt-get update && \
    apt-get install -y --no-install-recommends tesseract-ocr && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Google Cloud Vision (optional, ~50 MB)
ARG INCLUDE_GOOGLE_VISION=false
RUN if [ "$INCLUDE_GOOGLE_VISION" = "true" ]; then \
    pip install --no-cache-dir google-cloud-vision; \
fi

# AWS Textract via boto3 (optional, ~50 MB)
ARG INCLUDE_AWS_TEXTRACT=false
RUN if [ "$INCLUDE_AWS_TEXTRACT" = "true" ]; then \
    pip install --no-cache-dir boto3; \
fi

COPY app/ ./app/

ENV PHI_TOOL=auto
ENV PHI_CONFIDENCE_THRESHOLD=0.4
ENV PHI_MIN_TEXT_LENGTH=3

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
