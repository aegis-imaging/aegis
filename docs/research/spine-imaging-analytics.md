# Spine Imaging Analytics — Tool Comparison & Integration Research

**Date:** 2026-02-28
**Purpose:** Evaluate open-source spine imaging analysis tools for integration into the AEGIS analytics pipeline.

## Tool Comparison

### TotalSpineSeg (NeuroPoly) — INTEGRATED

| Aspect | Detail |
|--------|--------|
| **Repository** | [github.com/neuropoly/totalspineseg](https://github.com/neuropoly/totalspineseg) |
| **License** | LGPL-3.0 |
| **Installation** | `pip install totalspineseg[nnunetv2]` |
| **Interface** | CLI: `totalspineseg INPUT OUTPUT [--step1] [--iso]` |
| **Input** | NIfTI (.nii.gz), MRI only (robust to T1w, T2w, FLAIR) |
| **Output** | NIfTI segmentation masks — vertebrae (C1-sacrum), IVDs (C2/C3-L5/S1), spinal cord, spinal canal |
| **Architecture** | Two cascaded nnU-Net models (Dataset101 + Dataset102) |
| **Performance** | Dice 0.88 vertebrae, 0.80 IVDs, 99% labeling accuracy |
| **Hardware** | GPU recommended (8 GB VRAM), 32 GB RAM; ~2-5 min on GPU |
| **Training data** | 1,404 MRIs from four public datasets |

**Reference:** Houde JC et al. TotalSpineSeg: automatic segmentation of the spine in MRI using cascaded nnU-Net models. ISMRM 2024. DOI: 10.5281/zenodo.13894354

### SPINEPS — INTEGRATED

| Aspect | Detail |
|--------|--------|
| **Repository** | [github.com/Hendrik-code/spineps](https://github.com/Hendrik-code/spineps) |
| **License** | Apache 2.0 |
| **Installation** | `pip install spineps` |
| **Interface** | Python API (`from spineps.seg_run import process_img_nii`) + CLI |
| **Input** | NIfTI (.nii.gz), T2w sagittal primary (also T1w via model flag) |
| **Output** | `seg-spine` (14 semantic labels), `seg-vert` (instance vertebrae), JSON centroids |
| **Unique capability** | Only tool that segments vertebral substructures (body, arch, processes, facet joints, endplates) |
| **Hardware** | GPU recommended; ~1-3 min on GPU |

**14 semantic classes:** vertebral corpus, arch, spinous process, transverse processes (L/R), costovertebral joints (L/R), articular facets (sup/inf), endplates (sup/inf), intervertebral disc, spinal cord, spinal canal.

**Reference:** Greve H et al. "SPINEPS — Automatic Whole Spine Segmentation of T2-weighted MR images using a Two-Phase Approach to Multi-class Semantic and Instance Segmentation." European Radiology, 2025. DOI: 10.1007/s00330-024-11155-y. arXiv:2402.16368

### MedSAM2 (bowang-lab) — INTEGRATED

| Aspect | Detail |
|--------|--------|
| **Repository** | [github.com/bowang-lab/MedSAM2](https://github.com/bowang-lab/MedSAM2) |
| **License** | Apache 2.0 |
| **Installation** | `pip install git+https://github.com/bowang-lab/MedSAM.git` |
| **Base model** | Meta SAM 2.1 Tiny (38.9M params) |
| **Training** | Fine-tuned on 455K+ 3D image-mask pairs (CT, MRI, PET, US, endoscopy) |
| **Performance** | Dice 88.84% CT organs, 87.06% MRI organs, 87.22% PET lesions |
| **Hardware** | Very low GPU needs (~1 GB VRAM); <1s per slice |
| **Limitation** | Produces binary masks only (no anatomical labels); needs prompts |

**Key insight:** MedSAM2 treats 3D volumes as video frames, propagating segmentation from a prompted middle slice. Best suited as complementary/interactive tool alongside specialized label-based backends.

**Reference:** Ma J et al. "MedSAM2: Segment Anything in 3D Medical Images and Videos." arXiv:2504.03600, 2025.

### TotalSegmentator Spine Tasks — ENHANCED

Already integrated. Added `vertebrae_mr` task support with fallback label map.

- `total` task: 26 vertebrae among 117 total structures (CT)
- `vertebrae_mr` task: C1-C7, T1-T12, L1-L5, sacrum (MRI)
- Does NOT segment IVDs, spinal cord, or canal in free version

**Reference:** Wasserthal J et al. "TotalSegmentator: Robust Segmentation of 104 Anatomic Structures in CT Images." Radiology: AI, 2023. DOI: 10.1148/ryai.230024

### SCT (Spinal Cord Toolbox) — DEFERRED (Phase 3)

| Aspect | Detail |
|--------|--------|
| **Repository** | [github.com/spinalcordtoolbox/spinalcordtoolbox](https://github.com/spinalcordtoolbox/spinalcordtoolbox) |
| **License** | LGPL-3.0 |
| **Installation** | Custom installer (NOT pip-installable); Docker available |
| **Tools** | 58 CLI tools covering segmentation, registration, metrics |
| **Key capabilities** | CSA per vertebral level, compression metrics (aMCC, aSCOR), DTI per tract (FA, MD, AD, RD), MTR, MTsat, lesion analysis, PAM50 template registration |
| **Integration plan** | Separate Docker sidecar service (like defacing, phi-detection) |

**Deferred reason:** Bundles its own Python/Miniforge environment. Requires a new sidecar service directory, Dockerfile, Go handler, Docker Compose entry, and Terraform provisioning.

**Reference:** De Leener B et al. "SCT: Spinal Cord Toolbox, an open-source software for processing spinal cord MRI data." NeuroImage, 2017. DOI: 10.1016/j.neuroimage.2016.10.009

### SpineNet — EXCLUDED

| Aspect | Detail |
|--------|--------|
| **Repository** | [github.com/rwindsor1/SpineNet](https://github.com/rwindsor1/SpineNet) |
| **License** | Non-commercial (prohibits production use) |
| **Capability** | Clinical radiological grading (Pfirrmann, stenosis, spondylolisthesis, Modic changes) |

**Excluded reason:** Non-commercial license prevents production deployment.

**Reference:** Windsor R et al. "Automated detection, labelling and radiological grading of clinical spinal MRIs." Scientific Reports, 2024. DOI: 10.1038/s41598-024-64580-w

## Meta SAM 2 / SAM 3 Assessment

**SAM 2** ([github.com/facebookresearch/sam2](https://github.com/facebookresearch/sam2), Apache 2.0) is a general-purpose prompted segmentation model. Key findings:

- **Cannot produce labeled anatomical segmentations** without external prompts and label assignment
- "Everything mode" has low recall for medical structures
- No native 3D volume support (uses slice-as-video-frame approach)
- Very low GPU requirements (~1 GB VRAM for SAM 2.1 Tiny)

**SAM 3** (Meta, November 2025) adds text-prompted concept segmentation but has no medical fine-tuning yet.

**Medical adaptations:**
- **MedSAM2** (bowang-lab): Best medical variant. Fine-tuned on 455K images. Integrated.
- **SAM-Med3D**: Native 3D architecture. ECCV 2024 Workshops.
- **Brain-SAM**: 3D Hiera encoder for brain tumors/stroke. medRxiv 2026.
- **SpinalSAM-R1**: LLM-guided spine CT segmentation. arXiv:2511.00095

**Conclusion:** SAM 2 is best as a complementary/interactive tool. Specialized backends (SynthSeg, TotalSpineSeg, SPINEPS) remain superior for automated pipeline use.

## Standard Spine Metrics

### Vertebral metrics
- Volume (mm³), height (anterior/middle/posterior mm), width (AP/lateral mm)
- Compression ratio, wedge angle

### Disc metrics
- Height (mm), volume (mm³), Pfirrmann grade (1-5), signal intensity (T2w)

### Canal metrics
- AP diameter (mm), CSA (mm²), canal-to-body ratio (Torg-Pavlov)
- aMCC (adapted Maximal Canal Compromise), aSCOR (Spinal Cord Occupation Ratio)

### Spinal cord metrics
- CSA per vertebral level, AP/lateral diameters, eccentricity, solidity
- DTI per tract (FA, MD, AD, RD), MTR, MTsat

### Alignment
- Cobb angle, sagittal vertical axis, lumbar lordosis, thoracic kyphosis

## Public Datasets

| Dataset | Modality | Structures | Cases |
|---------|----------|-----------|-------|
| VerSe 2020 | CT | Vertebrae | 374 |
| SPIDER | MRI (T1/T2) | Vertebrae, IVDs, canal | 447 (218 patients) |
| CSpineSeg (Duke) | MRI (cervical) | Cervical spine | 1,255 |
| CTSpine1K | CT | Vertebrae | 1,005 |
