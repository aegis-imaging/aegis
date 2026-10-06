FROM python:3.14-slim

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
# ITK-SNAP/c3d     | No   | GPL v3           | ~200 MB
# BrainSuite       | No   | GPL v2           | ~1 GB
# MONAI Label      | No   | Apache 2.0       | ~500 MB (+ PyTorch)
# QSM (scipy)      | No   | BSD              | ~50 MB
# PETSurfer        | No   | (FreeSurfer)     | (included w/ FreeSurfer)
# BASIL            | No   | (FSL)            | (included w/ FSL)
# volBrain         | No   | Online API       | 0 (no local binary)
# TotalSpineSeg    | No   | LGPL-3.0         | ~1 GB (+ PyTorch/nnU-Net)
# SPINEPS          | No   | Apache 2.0       | ~500 MB (+ PyTorch)
# MedSAM2          | No   | Apache 2.0       | ~500 MB (+ PyTorch)

ARG INCLUDE_FREESURFER=false
ARG INCLUDE_FSL=false
ARG INCLUDE_ANTS=false
ARG INCLUDE_SPM=false
ARG INCLUDE_SYNTHSEG=false
ARG INCLUDE_NNUNET=false
ARG INCLUDE_TOTALSEGMENTATOR=false
ARG INCLUDE_ITKSNAP=false
ARG INCLUDE_BRAINSUITE=false
ARG INCLUDE_MONAI=false
ARG INCLUDE_QSM=false
ARG INCLUDE_TOTALSPINESEG=false
ARG INCLUDE_SPINEPS=false
ARG INCLUDE_MEDSAM2=false

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

# ITK-SNAP / Convert3D (GPL v3, ~200 MB)
# c3d is a lightweight command-line tool for NIfTI manipulation and segmentation.
# http://www.itksnap.org/pmwiki/pmwiki.php?n=Convert3D.Convert3D
RUN if [ "$INCLUDE_ITKSNAP" = "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        wget ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    echo "c3d: Download from http://www.itksnap.org/pmwiki/pmwiki.php?n=Downloads.C3D" && \
    echo "Set C3D_BIN env var after installing"; \
    fi

# BrainSuite (GPL v2, ~1 GB)
# Cortical surface extraction and volumetric analysis.
# http://brainsuite.org/
RUN if [ "$INCLUDE_BRAINSUITE" = "true" ]; then \
    apt-get update && apt-get install -y --no-install-recommends \
        wget ca-certificates libgomp1 && \
    rm -rf /var/lib/apt/lists/* && \
    echo "BrainSuite: Download from http://brainsuite.org/download/" && \
    echo "Set BRAINSUITE_DIR env var after installing"; \
    fi

# MONAI Label (Apache 2.0, ~500 MB + PyTorch)
# AI-based interactive and automatic segmentation framework.
# https://monai.io/label.html
RUN if [ "$INCLUDE_MONAI" = "true" ]; then \
    pip install --no-cache-dir monailabel; \
    fi

# QSM tools — scipy for Python-based TKD/Laplacian QSM pipeline (~50 MB)
# For external tgv_qsm binary, install separately.
RUN if [ "$INCLUDE_QSM" = "true" ]; then \
    pip install --no-cache-dir scipy; \
    fi

# PETSurfer: included with INCLUDE_FREESURFER (mri_gtmpvc is part of FreeSurfer)
# BASIL/oxford_asl: included with INCLUDE_FSL (oxford_asl is part of FSL)
# volBrain: online API only, no local binary needed

# TotalSpineSeg (LGPL-3.0, ~1 GB + PyTorch/nnU-Net)
# Automatic spine MRI segmentation: vertebrae, IVDs, spinal cord, spinal canal.
# Cascaded nnU-Net models. GPU recommended (8 GB VRAM). ~2-5 min on GPU.
# https://github.com/neuropoly/totalspineseg
RUN if [ "$INCLUDE_TOTALSPINESEG" = "true" ]; then \
    pip install --no-cache-dir "totalspineseg[nnunetv2]"; \
    fi

# SPINEPS (Apache 2.0, ~500 MB + PyTorch)
# Automatic whole-spine MRI segmentation with 14 semantic classes + instance
# vertebrae. Unique substructure segmentation (body, arch, processes, facets).
# https://github.com/Hendrik-code/spineps
RUN if [ "$INCLUDE_SPINEPS" = "true" ]; then \
    pip install --no-cache-dir spineps; \
    fi

# MedSAM2 (Apache 2.0, ~500 MB + PyTorch)
# General-purpose prompted medical image segmentation (any modality).
# Fine-tuned SAM 2 on 455K+ 3D medical image-mask pairs.
# Very low GPU needs (~1 GB VRAM). Produces binary masks only.
# https://github.com/bowang-lab/MedSAM2
RUN if [ "$INCLUDE_MEDSAM2" = "true" ]; then \
    pip install --no-cache-dir "git+https://github.com/bowang-lab/MedSAM.git"; \
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
