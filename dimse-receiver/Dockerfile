# AEGIS DIMSE Receiver
#
# Receives DICOM studies over DIMSE C-STORE (port 11112), writes files to the
# shared data volume, then triggers Go API ingest via POST /api/ingest.

FROM python:3.14-slim

# DICOM decompression libraries (JPEG2000, JPEG-LS, RLE).
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ app/

ENV DIMSE_AE_TITLE=AEGIS
ENV DIMSE_PORT=11112
ENV DIMSE_DATA_DIR=/app/data
ENV API_URL=http://api:8080
ENV DIMSE_PROJECT_SLUG=default
ENV DIMSE_INGEST_TIMEOUT=30

EXPOSE 8080
EXPOSE 11112

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
