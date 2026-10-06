# AEGIS DICOM Tools — consolidated Python sidecar.
#
# Hosts six former sidecars (phi-detection, qc-service, classification-service,
# protocol-service, synth-service, bids-service) under a single Cloud Run
# service. See app/main.py for the mount layout.
#
# Optional cloud OCR backends for phi-detection are still available via the
# same INCLUDE_* build args as the old phi-detection image.

FROM python:3.14-slim

# Install tesseract-ocr (phi OCR fallback) + DICOM decompression libraries.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    tesseract-ocr \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Optional phi-detection cloud OCR backends (mirrors the previous
# phi-detection Dockerfile so existing build configurations work).
ARG INCLUDE_GEMINI=false
RUN if [ "$INCLUDE_GEMINI" = "true" ]; then \
    pip install --no-cache-dir "google-genai>=1.0.0"; \
fi

ARG INCLUDE_GOOGLE_VISION=false
RUN if [ "$INCLUDE_GOOGLE_VISION" = "true" ]; then \
    pip install --no-cache-dir google-cloud-vision; \
fi

ARG INCLUDE_AZURE_VISION=false
RUN if [ "$INCLUDE_AZURE_VISION" = "true" ]; then \
    pip install --no-cache-dir azure-ai-vision-imageanalysis; \
fi

COPY app/ ./app/

# phi-detection defaults preserved from the previous standalone image so
# operator configuration carries over unchanged.
ENV PHI_TOOL=auto
ENV PHI_CONFIDENCE_THRESHOLD=0.4
ENV PHI_MIN_TEXT_LENGTH=3

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
