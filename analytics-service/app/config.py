"""Analytics service configuration from environment variables."""

import os

ANALYTICS_TOOL = os.getenv("ANALYTICS_TOOL", "auto")
ANALYTICS_TIMEOUT = int(os.getenv("ANALYTICS_TIMEOUT", "86400"))  # 24h default
DATA_DIR = os.getenv("ANALYTICS_DATA_DIR", "/app/data")

# FreeSurfer
FS_LICENSE = os.getenv("FS_LICENSE", "/app/secrets/freesurfer.license")

# FSL
FSL_DIR = os.getenv("FSL_DIR", "/usr/local/fsl")
FSLDIR = os.getenv("FSLDIR", FSL_DIR)

# ANTs
ANTSPATH = os.getenv("ANTSPATH", "/usr/local/ANTs/bin")

# SPM
SPM_BIN = os.getenv("SPM_BIN", "spm12")
MCR_DIR = os.getenv("MCR_DIR", "/usr/local/MATLAB/MATLAB_Runtime")

# Atlas ROI + TBM-SyN
ATLAS_DIR = os.getenv("ATLAS_DIR", "/opt/atlases")
DEFAULT_ATLAS = os.getenv("DEFAULT_ATLAS", "aal3")
ANTS_TEMPLATE_DIR = os.getenv(
    "ANTS_TEMPLATE_DIR", "/opt/ants/templates/OASIS-30_Atropos_template"
)

# SynthSeg
SYNTHSEG_PARC = os.getenv("SYNTHSEG_PARC", "true")  # --parc for 97 ROIs (vs 32)
SYNTHSEG_ROBUST = os.getenv("SYNTHSEG_ROBUST", "false")  # --robust for clinical scans
SYNTHSEG_QC = os.getenv("SYNTHSEG_QC", "true")  # Output QC scores

# nnU-Net
NNUNET_MODEL_DIR = os.getenv("NNUNET_MODEL_DIR", "/opt/nnunet/models")
NNUNET_DATASET_ID = os.getenv("NNUNET_DATASET_ID", "Dataset001_BrainTumour")
NNUNET_FOLDS = os.getenv("NNUNET_FOLDS", "all")
NNUNET_USE_GPU = os.getenv("NNUNET_USE_GPU", "true").lower() == "true"

# TotalSegmentator
TOTALSEG_TASK = os.getenv("TOTALSEG_TASK", "total")
TOTALSEG_FAST = os.getenv("TOTALSEG_FAST", "false").lower() == "true"
TOTALSEG_USE_GPU = os.getenv("TOTALSEG_USE_GPU", "true").lower() == "true"

# ITK-SNAP / Convert3D
C3D_BIN = os.getenv("C3D_BIN", "c3d")
ITKSNAP_WT_BIN = os.getenv("ITKSNAP_WT_BIN", "itksnap-wt")

# BrainSuite
BRAINSUITE_DIR = os.getenv("BRAINSUITE_DIR", "/opt/BrainSuite")
BSE_BIN = os.getenv("BSE_BIN", "bse")

# volBrain (online service)
VOLBRAIN_API_URL = os.getenv("VOLBRAIN_API_URL", "")
VOLBRAIN_API_KEY = os.getenv("VOLBRAIN_API_KEY", "")
VOLBRAIN_POLL_INTERVAL = int(os.getenv("VOLBRAIN_POLL_INTERVAL", "30"))
VOLBRAIN_TIMEOUT = int(os.getenv("VOLBRAIN_TIMEOUT", "3600"))

# MONAI Label
MONAI_MODEL_DIR = os.getenv("MONAI_MODEL_DIR", "/opt/monai/models")
MONAI_MODEL_NAME = os.getenv("MONAI_MODEL_NAME", "segmentation")
MONAI_USE_GPU = os.getenv("MONAI_USE_GPU", "true").lower() == "true"

# PETSurfer
PETSURFER_PSF_FWHM = os.getenv("PETSURFER_PSF_FWHM", "6")
PETSURFER_KM_REF = os.getenv("PETSURFER_KM_REF", "8 47")

# QSM
QSM_TOOL = os.getenv("QSM_TOOL", "auto")
QSM_UNWRAP_METHOD = os.getenv("QSM_UNWRAP_METHOD", "laplacian")
QSM_BFR_METHOD = os.getenv("QSM_BFR_METHOD", "vsharp")
QSM_DIPOLE_METHOD = os.getenv("QSM_DIPOLE_METHOD", "ilsqr")

# BASIL / oxford_asl
BASIL_BOLUS_DURATION = os.getenv("BASIL_BOLUS_DURATION", "1.8")
BASIL_TIS = os.getenv("BASIL_TIS", "3.6")
BASIL_CALIB_METHOD = os.getenv("BASIL_CALIB_METHOD", "voxel")

# TotalSpineSeg (LGPL-3.0)
TOTALSPINESEG_USE_GPU = os.getenv("TOTALSPINESEG_USE_GPU", "true").lower() == "true"
TOTALSPINESEG_ISO = os.getenv("TOTALSPINESEG_ISO", "false").lower() == "true"
TOTALSPINESEG_STEP1_ONLY = os.getenv("TOTALSPINESEG_STEP1_ONLY", "false").lower() == "true"

# SPINEPS (Apache 2.0)
SPINEPS_MODEL = os.getenv("SPINEPS_MODEL", "t2w")  # "t2w" or "t1w"
SPINEPS_USE_GPU = os.getenv("SPINEPS_USE_GPU", "true").lower() == "true"

# MedSAM2 (Apache 2.0)
MEDSAM2_CHECKPOINT = os.getenv("MEDSAM2_CHECKPOINT", "/opt/medsam2/medsam2_checkpoint.pth")
MEDSAM2_USE_GPU = os.getenv("MEDSAM2_USE_GPU", "true").lower() == "true"
