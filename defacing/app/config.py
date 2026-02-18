import os


class Config:
    # Which defacing tool to use. Options: "auto", "mri_reface", "mri_deface", "nibabel"
    # "auto" tries each in priority order and uses the first available.
    deface_tool: str = os.environ.get("DEFACE_TOOL", "auto")

    # Path to the mri_deface binary (FreeSurfer standalone)
    mri_deface_bin: str = os.environ.get("MRI_DEFACE_BIN", "mri_deface")
    # Path to the brain atlas used by mri_deface
    mri_deface_brain: str = os.environ.get(
        "MRI_DEFACE_BRAIN", "/opt/mri_deface/talairach_mixed_with_skull.gca"
    )
    mri_deface_face: str = os.environ.get(
        "MRI_DEFACE_FACE", "/opt/mri_deface/face.gca"
    )

    # Path to the mri_reface binary (MATLAB Runtime required)
    mri_reface_bin: str = os.environ.get("MRI_REFACE_BIN", "mri_reface")

    # Path to dcm2niix binary (used by mri_deface and mri_reface backends)
    dcm2niix_bin: str = os.environ.get("DCM2NIIX_BIN", "dcm2niix")


cfg = Config()
