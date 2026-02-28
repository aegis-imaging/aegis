FROM python:3.12-slim

# System libraries for DICOM decompression
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgdcm-tools \
    libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*

# ── Optional neuroimaging tool installation ──────────────────────────────
# Each tool is gated behind a build arg. Set to "true" to install.
# In production, prefer a custom base image with tools pre-installed.
#
# Tool             | MCR? | License          | Size
# -----------------|------|------------------|-------
# FreeSurfer       | No   | Free registration| ~500 MB (minimal)
# FSL              | No   | Oxford/academic  | ~2 GB
# ANTs             | No   | Apache 2.0       | ~300 MB
# SPM              | Yes  | GPL v2           | ~3 GB (includes MCR)
# SynthSeg         | No   | Apache 2.0       | ~200 MB (+ TensorFlow)
# nnU-Net          | No   | Apache 2.0       | ~1 GB (+ PyTorch)
# TotalSegmentator | No   | Apache 2.0       | ~1 GB (+ PyTorch)

ARG INCLUDE_FREESURFER=false
ARG INCLUDE_FSL=false
ARG INCLUDE_ANTS=false
ARG INCLUDE_SPM=false
ARG INCLUDE_SYNTHSEG=false
ARG INCLUDE_NNUNET=false
ARG INCLUDE_TOTALSEGMENTATOR=false

# FreeSurfer (minimal distribution — recon-all + stats tools)
# Requires FS_LICENSE env var pointing to a license file at runtime.
# Get a free license at https://surfer.nmr.mgh.harvard.edu/registration.html
RUN if [ "$INCLUDE_FREESURFER" = "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        wget ca-certificates bc libgomp1 libglu1-mesa libxmu6 && \
    rm -rf /var/lib/apt/lists/*; \
    fi

# FSL (FMRIB Software Library — BET, FAST, FLIRT, DTIFIT)
# Free for academic/research use. https://fsl.fmrib.ox.ac.uk/fsl/
RUN if [ "$INCLUDE_FSL" = "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        wget ca-certificates libopenblas-base dc && \
    rm -rf /var/lib/apt/lists/*; \
    fi

# ANTs (Advanced Normalization Tools — cortical thickness, registration, N4)
# Apache 2.0 — fully open, no license needed.
RUN if [ "$INCLUDE_ANTS" = "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        wget ca-certificates && \
    rm -rf /var/lib/apt/lists/*; \
    fi

# SPM12 standalone + MATLAB Compiler Runtime (MCR)
# MCR is FREE to redistribute — no MATLAB license required.
# SPM is GPL v2. MCR ~3 GB download.
# https://www.fil.ion.ucl.ac.uk/spm/software/spm12/
RUN if [ "$INCLUDE_SPM" = "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        wget ca-certificates unzip libxext6 libxt6 libxmu6 && \
    rm -rf /var/lib/apt/lists/* && \
    echo "SPM+MCR: Install MCR v9.13 (R2022b) and SPM12 standalone here" && \
    echo "MCR download: https://ssd.mathworks.com/supportfiles/downloads/R2022b/Release/10/deployment_files/installer/complete/glnxa64/MATLAB_Runtime_R2022b_Update_10_glnxa64.zip" && \
    echo "SPM standalone: https://www.fil.ion.ucl.ac.uk/spm/download/restricted/utopia/spm12/spm12_r7771_Linux_R2022b.zip" && \
    echo "Set MCR_DIR and SPM_BIN env vars after installing"; \
    fi

# SynthSeg standalone (Apache 2.0, ~200 MB + TensorFlow)
# Contrast-agnostic brain MRI segmentation — works on any MRI contrast/resolution.
# Alternative: use FreeSurfer 7.3.2+ (mri_synthseg) when INCLUDE_FREESURFER=true.
RUN if [ "$INCLUDE_SYNTHSEG" = "true" ]; then \
    pip install --no-cache-dir tensorflow && \
    pip install --no-cache-dir SynthSeg || \
    echo "SynthSeg standalone not available via pip — use FreeSurfer mri_synthseg instead"; \
    fi

# nnU-Net v2 (Apache 2.0, ~1 GB + PyTorch)
# Self-configuring DL segmentation. Pre-trained models must be volume-mounted
# at NNUNET_MODEL_DIR (/opt/nnunet/models by default).
RUN if [ "$INCLUDE_NNUNET" = "true" ]; then \
    pip install --no-cache-dir nnunetv2; \
    fi

# TotalSegmentator (Apache 2.0, ~1 GB + PyTorch + nnU-Net)
# 117-structure whole-body CT/MRI segmentation. Models auto-downloaded on first
# run (~1.6 GB). Replaces 3D Slicer's SlicerTotalSegmentator extension.
# Note: shares PyTorch with nnU-Net if both are enabled.
RUN if [ "$INCLUDE_TOTALSEGMENTATOR" = "true" ]; then \
    pip install --no-cache-dir TotalSegmentator; \
    fi

# ── Atlas data ────────────────────────────────────────────────────────────
# Atlases are stored in /opt/atlases/{name}/ with a NIfTI volume and
# a {name}_labels.csv file. Custom atlases can be volume-mounted at runtime.
#
# AAL3: Tzourio-Mazoyer et al. NeuroImage 2002;15(1):273-289
#   166 regions. Labels bundled in app/atlases/aal3_labels.csv.
#   NIfTI must be provided at runtime or via INCLUDE_AAL3 build arg.
#
# MCALT/ADIR122: https://www.nitrc.org/projects/mcalt/ (CC-BY-NC-SA 4.0)
#   Mayo Clinic template + atlas. Must be downloaded separately.

ARG INCLUDE_AAL3=false

RUN if [ "$INCLUDE_AAL3" = "true" ]; then \
    mkdir -p /opt/atlases/aal3 && \
    echo "AAL3: Place aal3v1_1mm.nii.gz in /opt/atlases/aal3/ at runtime" && \
    echo "Download from: https://www.gin.cnrs.fr/en/tools/aal/"; \
    fi

# ── Python application ───────────────────────────────────────────────────

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ app/

# Non-root user for security
RUN useradd -r -s /bin/false appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
