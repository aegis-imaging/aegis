FROM python:3.12-slim

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

COPY app/ ./app/

ENV SYNTH_DATA_DIR=/app/data
ENV SYNTH_USE_GPU=false
ENV SYNTH_MAX_SLICES=200
ENV SYNTH_MAX_SIZE=512

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
