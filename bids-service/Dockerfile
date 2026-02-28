FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

# ── dcm2niix (static binary, ~5 MB) ────────────────────────────────────────
ARG DCM2NIIX_VERSION=v1.0.20241211
RUN curl -fsSL \
    "https://github.com/rordenlab/dcm2niix/releases/download/${DCM2NIIX_VERSION}/dcm2niix_lnx.zip" \
    -o /tmp/dcm2niix.zip \
    && apt-get update && apt-get install -y --no-install-recommends unzip \
    && unzip /tmp/dcm2niix.zip -d /usr/local/bin \
    && chmod +x /usr/local/bin/dcm2niix \
    && rm /tmp/dcm2niix.zip \
    && apt-get remove -y unzip && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

ENV BIDS_TOOL=auto
ENV DCM2NIIX_BIN=/usr/local/bin/dcm2niix

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
