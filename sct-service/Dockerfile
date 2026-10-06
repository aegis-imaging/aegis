FROM python:3.14-slim

# System libraries needed for nibabel/numpy
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

# ── Optional SCT installation ────────────────────────────────────────────
# SCT is NOT pip-installable — it bundles its own Miniforge Python environment.
# In production, prefer a custom base image with SCT pre-installed, or
# volume-mount SCT binaries. The service gracefully degrades when SCT
# binaries are not on PATH (shutil.which check in backend).
#
# To build with SCT available, use a pre-built SCT base image instead:
#   docker build --build-arg SCT_BASE_IMAGE=neuropoly/sct:6.4 ...
# Or mount /opt/sct at runtime with SCT_DIR set.

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ app/

# Non-root user for security
RUN useradd -r -s /bin/false appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
