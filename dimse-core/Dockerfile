# dimse-core is a library, not a service — there's no runnable binary here.
# This Dockerfile exists so CI can run the test suite against an installable
# package without dragging the receiver or router along.
FROM python:3.12-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

COPY dimse-core /app/dimse-core
RUN pip install --no-cache-dir -r /app/dimse-core/requirements.txt \
    && pip install --no-cache-dir -r /app/dimse-core/requirements-test.txt \
    && pip install --no-cache-dir -e /app/dimse-core

CMD ["pytest", "-v", "/app/dimse-core/tests"]
