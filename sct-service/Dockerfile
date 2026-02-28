FROM neuropoly/spinalcordtoolbox:latest

# SCT image is based on Ubuntu with its own Miniforge Python environment.
# We install FastAPI + deps into the existing Python environment.

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ app/

EXPOSE 8080

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
