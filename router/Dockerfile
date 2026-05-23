# AEGIS Router — on-prem DICOM receive + de-identify + forward.
#
# Built from the repo root so we can include the midi-b package as a
# library dependency without publishing it to PyPI:
#   docker build -f router/Dockerfile -t aegis-router .
FROM python:3.12-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install runtime dependencies first so the layer caches across code changes.
COPY router/requirements.txt /app/router/requirements.txt
RUN pip install -r /app/router/requirements.txt

# Install midi-b as an editable dep so tag de-id stays in lock-step with the
# rest of the repo. (When this is built from the monorepo, midi-b is right
# next to router/.)
COPY midi-b /app/midi-b
RUN pip install -e /app/midi-b

# Application code
COPY router /app/router

WORKDIR /app/router

EXPOSE 8080 11112

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -fsS http://localhost:8080/healthz || exit 1

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
