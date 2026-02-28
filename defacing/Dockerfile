# AEGIS Defacing Service
#
# Defacing backend is controlled by DEFACE_TOOL env var:
#   - "auto" (default): tries mri_reface → deepdefacer → mri_deface → nibabel-fallback
#   - "mri_reface": uses mri_reface (best modality coverage; non-commercial research only)
#   - "deepdefacer": uses DeepDefacer (3D U-Net, fast; requires TensorFlow)
#   - "mri_deface": uses mri_deface from FreeSurfer (MRI only; no license restrictions)
#   - "nibabel": pure-Python fallback (dev/testing only)
#
# To include DeepDefacer, build with:  --build-arg INCLUDE_DEEPDEFACER=true
# To include mri_deface, build with:   --build-arg INCLUDE_MRI_DEFACE=true
# To include mri_reface, build with:   --build-arg INCLUDE_MRI_REFACE=true
# (mri_reface requires MATLAB Runtime + ANTs + NiftyReg — adds ~3 GB)

FROM python:3.12-slim

ARG INCLUDE_MRI_DEFACE=false
ARG INCLUDE_MRI_REFACE=false
ARG MRI_DEFACE_VERSION=1.22

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

# ── dcm2niix (static binary, ~5 MB) ────────────────────────────────────────
# Used by mri_deface and mri_reface backends to convert DICOM → NIfTI.
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

# ── mri_deface (optional, FreeSurfer standalone, ~0.5 GB with atlas) ───────
# Supports: MRI only. License: free, no restrictions.
# https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/
RUN if [ "$INCLUDE_MRI_DEFACE" = "true" ]; then \
    mkdir -p /opt/mri_deface \
    && curl -fsSL \
        "https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/${MRI_DEFACE_VERSION}/linux/mri_deface" \
        -o /usr/local/bin/mri_deface \
    && chmod +x /usr/local/bin/mri_deface \
    && curl -fsSL \
        "https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/${MRI_DEFACE_VERSION}/talairach_mixed_with_skull.gca" \
        -o /opt/mri_deface/talairach_mixed_with_skull.gca \
    && curl -fsSL \
        "https://surfer.nmr.mgh.harvard.edu/pub/dist/mri_deface/${MRI_DEFACE_VERSION}/face.gca" \
        -o /opt/mri_deface/face.gca; \
fi

# ── mri_reface (optional, non-commercial research use only) ─────────────────
# Supports: T1/T2/FLAIR/T2*/ASL MRI, PET (Amyloid/Tau/FDG), CT.
# Requires: MATLAB Runtime, ANTs, NiftyReg.
# Download binary from: https://www.nitrc.org/projects/mri_reface
# Uncomment and configure the following block after downloading:
#
# COPY --chown=root:root mri_reface /usr/local/bin/mri_reface
# RUN chmod +x /usr/local/bin/mri_reface
#
# Install MATLAB Runtime (MCR) v912 (R2021b):
# RUN curl -fsSL https://ssd.mathworks.com/supportfiles/downloads/R2021b/Release/9/deployment_files/installer/complete/glnxa64/MATLAB_Runtime_R2021b_Update_9_glnxa64.zip \
#     -o /tmp/mcr.zip && unzip /tmp/mcr.zip -d /tmp/mcr_installer \
#     && /tmp/mcr_installer/install -mode silent -agreeToLicense yes \
#     && rm -rf /tmp/mcr*
#
# ENV LD_LIBRARY_PATH=/usr/local/MATLAB/MATLAB_Runtime/v912/runtime/glnxa64:...

# ── Python dependencies ─────────────────────────────────────────────────────
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# ── DeepDefacer (optional, pip-installable, ~500 MB with TensorFlow) ────────
# Supports: MRI only. License: no commercial restrictions documented.
# Uses a 3D U-Net model — significantly faster than registration-based tools.
# https://github.com/AKhazane/DeepDeface
ARG INCLUDE_DEEPDEFACER=false
RUN if [ "$INCLUDE_DEEPDEFACER" = "true" ]; then \
    pip install --no-cache-dir deepdefacer \
    && python -c "import deepdefacer; print('DeepDefacer installed successfully')"; \
fi

# ── Application code ────────────────────────────────────────────────────────
COPY app/ ./app/

# ── Runtime ─────────────────────────────────────────────────────────────────
ENV DEFACE_TOOL=auto
ENV MRI_DEFACE_BIN=/usr/local/bin/mri_deface
ENV MRI_DEFACE_BRAIN=/opt/mri_deface/talairach_mixed_with_skull.gca
ENV MRI_DEFACE_FACE=/opt/mri_deface/face.gca
ENV MRI_REFACE_BIN=/usr/local/bin/mri_reface
ENV DCM2NIIX_BIN=/usr/local/bin/dcm2niix
ENV DEEPDEFACER_GPU=false

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s \
    CMD curl -f http://localhost:8080/healthz || exit 1

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
