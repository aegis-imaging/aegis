"""
AEGIS DICOM Tools — consolidated Python sidecar.

Replaces six previously-separate Cloud Run services (phi-detection,
qc-service, classification-service, protocol-service, synth-service,
bids-service) with one FastAPI host. Each former sidecar lives as a
sub-module under `app/`, and its FastAPI app is mounted at a prefix
so the routes look like:

    POST /phi/detect       /phi/detect-tags     /phi/redact
    POST /qc/check
    POST /classify         (mounted at /classify, route at / inside)
    POST /protocol/check
    POST /synth/generate
    POST /bids/convert

The Go API used to configure six environment variables
(PHI_DETECTION_SERVICE_URL, QC_SERVICE_URL, …) — all collapse to one
DICOM_TOOLS_URL. See api/config/config.go.

Why consolidate: each old sidecar was ~50 MB of mostly-identical pydicom
plumbing. Idle cost was already zero (Cloud Run scale-to-zero) but every
CI run rebuilt six Docker images and every deploy created six Cloud Run
revisions. One image, one revision, one mental model.

What stays separate: defacing (FreeSurfer/DeepDefacer system deps),
analytics-service (FreeSurfer/FSL/ANTs), sct-service (Spinal Cord
Toolbox). Those carry multi-GB native toolchains and shouldn't share an
image.
"""

import logging
import os

from fastapi import FastAPI

from .phi import main as phi_app_module
from .qc import main as qc_app_module
from .classify import main as classify_app_module
from .protocol import main as protocol_app_module
from .synth import main as synth_app_module
from .bids import main as bids_app_module

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger("dicom-tools")

app = FastAPI(
    title="AEGIS DICOM Tools",
    description=(
        "Consolidated Python sidecar — hosts phi-detection, qc-service, "
        "classification-service, protocol-service, synth-service, and "
        "bids-service under a single Cloud Run service."
    ),
    version="1.0.0",
)


@app.get("/healthz")
def healthz():
    return {"ok": True, "service": "dicom-tools"}


# Cloud Run's external liveness probes hit /health; keep that working too.
@app.get("/health")
def health():
    return {"ok": True, "service": "dicom-tools"}


# Mount each former sidecar at a prefix. The sub-apps keep their existing
# /detect, /check, /classify, /generate, /convert internal routes —
# mounting prefixes them in the URL space without code changes inside.
app.mount("/phi",      phi_app_module.app)
app.mount("/qc",       qc_app_module.app)
app.mount("/classify", classify_app_module.app)
app.mount("/protocol", protocol_app_module.app)
app.mount("/synth",    synth_app_module.app)
app.mount("/bids",     bids_app_module.app)


log.info("dicom-tools starting; mounted /phi /qc /classify /protocol /synth /bids")
