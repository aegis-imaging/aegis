FROM python:3.12-slim

# DICOM decompression libraries (JPEG2000, JPEG-LS, RLE).
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

ENV PROTOCOL_TOOL=auto
ENV PROTOCOL_DEFAULT_TOLERANCE=5.0

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
