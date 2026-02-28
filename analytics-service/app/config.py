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
