FROM python:3.12-slim

# Install tesseract-ocr.
RUN apt-get update && \
    apt-get install -y --no-install-recommends tesseract-ocr && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

ENV PHI_TOOL=auto
ENV PHI_CONFIDENCE_THRESHOLD=0.4
ENV PHI_MIN_TEXT_LENGTH=3

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
