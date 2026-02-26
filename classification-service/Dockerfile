# AEGIS Classification Service
#
# Backend controlled by CLASSIFY_TOOL env var:
#   "auto" (default): gemini > google_vision > aws_rekognition > heuristic
#   "gemini":         Gemini multimodal classification (google-genai SDK)
#   "google_vision":  Cloud Vision label_detection augments heuristic
#   "aws_rekognition": Rekognition detect_labels augments heuristic
#   "heuristic":      local DICOM tag analysis only (no cloud dependencies)
#
# Cloud backends are optional — build with:
#   --build-arg INCLUDE_GEMINI=true
#   --build-arg INCLUDE_GOOGLE_VISION=true
#   --build-arg INCLUDE_AWS_REKOGNITION=true

FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Gemini multimodal backend via unified google-genai SDK (optional)
# Supports Vertex AI (ADC / Workload Identity) and AI Studio (GEMINI_API_KEY).
ARG INCLUDE_GEMINI=false
RUN if [ "$INCLUDE_GEMINI" = "true" ]; then \
    pip install --no-cache-dir "google-genai>=1.0.0" Pillow numpy; \
fi

# Google Cloud Vision (optional, ~50 MB) — includes Pillow+numpy for pixel rendering
ARG INCLUDE_GOOGLE_VISION=false
RUN if [ "$INCLUDE_GOOGLE_VISION" = "true" ]; then \
    pip install --no-cache-dir google-cloud-vision Pillow numpy; \
fi

# AWS Rekognition via boto3 (optional, ~50 MB) — includes Pillow+numpy for pixel rendering
ARG INCLUDE_AWS_REKOGNITION=false
RUN if [ "$INCLUDE_AWS_REKOGNITION" = "true" ]; then \
    pip install --no-cache-dir boto3 Pillow numpy; \
fi

COPY app/ app/

ENV CLASSIFY_TOOL=auto
ENV CLASSIFY_CONFIDENCE_THRESHOLD=0.5

EXPOSE 8080

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
