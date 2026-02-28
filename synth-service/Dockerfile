FROM python:3.12-slim

# DICOM decompression libraries (JPEG2000, JPEG-LS, RLE).
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# ── Base dependencies ─────────────────────────────────────────────────────────
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# ── Optional: MONAI BraTS LDM GPU backend ────────────────────────────────────
# Build with: docker build --build-arg INCLUDE_MONAI=true ...
# Adds ~2 GB (torch + monai-generative + huggingface-hub + scikit-image).
# Requires NVIDIA L4 or better GPU at runtime (SYNTH_USE_GPU=true).
ARG INCLUDE_MONAI=false
RUN if [ "$INCLUDE_MONAI" = "true" ]; then \
      pip install --no-cache-dir \
        torch torchvision --index-url https://download.pytorch.org/whl/cu121 \
        monai-generative \
        huggingface-hub \
        "scikit-image>=0.21"; \
    fi

# ── Optional: Vertex AI Imagen backend ────────────────────────────────────────
# Build with: docker build --build-arg INCLUDE_GEMINI=true ...
# Uses the unified google-genai SDK.
# Supports Vertex AI (ADC / Workload Identity) and AI Studio (GEMINI_API_KEY).
ARG INCLUDE_GEMINI=false
RUN if [ "$INCLUDE_GEMINI" = "true" ]; then \
      pip install --no-cache-dir \
        "google-genai>=1.0.0" \
        Pillow; \
    fi

COPY app/ ./app/

ENV SYNTH_DATA_DIR=/app/data
ENV SYNTH_USE_GPU=false
ENV SYNTH_USE_GEMINI=false
ENV SYNTH_MAX_SLICES=200
ENV SYNTH_MAX_SIZE=512

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
