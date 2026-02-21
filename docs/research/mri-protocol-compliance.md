# MRI Protocol Compliance Checking — Research References

Compiled for use in building protocol validation rules for the AEGIS QC service.
All sources verified as of February 2026. Notes on source type and citeability included.

---

## 1. Key DICOM Tags for MRI Protocol Compliance

The DICOM MR Image Module (Part 3, Section C.8.3) defines the acquisition-related attributes for MR images. The following tags are critical for protocol compliance validation:

### Primary Acquisition Parameters (Group 0018)

| Tag | Keyword | DICOM Type | Description |
|-----|---------|------------|-------------|
| (0018,0020) | ScanningSequence | 1 (Required) | e.g. SE, IR, GR, EP |
| (0018,0021) | SequenceVariant | 1 (Required) | e.g. SK, MTC, SS, TRSS, SP, MP, OSP |
| (0018,0022) | ScanOptions | 2 | e.g. PER, RG, CG, PPG, FC, PFF |
| (0018,0023) | MRAcquisitionType | 2 | 2D or 3D |
| (0018,0024) | SequenceName | 3 (Optional) | Vendor-specific (e.g. *tfl3d1_16ns for Siemens MPRAGE) |
| (0018,0050) | SliceThickness | 2 | Nominal slice thickness in mm |
| (0018,0080) | RepetitionTime | 2C | TR in ms |
| (0018,0081) | EchoTime | 2 | TE in ms |
| (0018,0082) | InversionTime | 2C | TI in ms (required if IR sequence) |
| (0018,0083) | NumberOfAverages | 3 | NEX / NSA |
| (0018,0084) | ImagingFrequency | 3 | MHz (related to field strength) |
| (0018,0085) | ImagedNucleus | 3 | e.g. 1H |
| (0018,0087) | MagneticFieldStrength | 3 | Tesla |
| (0018,0088) | SpacingBetweenSlices | 3 | Center-to-center distance in mm |
| (0018,0089) | NumberOfPhaseEncodingSteps | 3 | Phase encoding matrix |
| (0018,0091) | EchoTrainLength | 2 | Number of echoes per TR (ETL) |
| (0018,0093) | PercentSampling | 3 | Phase encoding fraction (%) |
| (0018,0094) | PercentPhaseFieldOfView | 3 | Phase FOV as % of frequency FOV |
| (0018,0095) | PixelBandwidth | 3 | Hz/pixel |
| (0018,1310) | AcquisitionMatrix | 3 | [freq rows, freq cols, phase rows, phase cols] |
| (0018,1312) | InPlanePhaseEncodingDirection | 3 | ROW or COL |
| (0018,1314) | FlipAngle | 3 | Degrees |
| (0018,1316) | SAR | 3 | W/kg |
| (0018,9087) | DiffusionBValue | 3 | b-value in s/mm^2 (Enhanced MR) |
| (0018,9089) | DiffusionGradientDirectionSequence | 3 | Gradient direction vectors |

### Image Geometry (Group 0028)

| Tag | Keyword | Type | Description |
|-----|---------|------|-------------|
| (0028,0010) | Rows | 1 | Pixel rows |
| (0028,0011) | Columns | 1 | Pixel columns |
| (0028,0030) | PixelSpacing | 1 | [row spacing, col spacing] in mm |

### Additional Relevant Tags

| Tag | Keyword | Description |
|-----|---------|-------------|
| (0008,0060) | Modality | Should be "MR" |
| (0008,103E) | SeriesDescription | Free text (used for sequence identification) |
| (0018,0015) | BodyPartExamined | e.g. HEAD, BRAIN |
| (0018,1020) | SoftwareVersions | Scanner software version |
| (0018,1030) | ProtocolName | Operator-assigned protocol name |
| (0020,0032) | ImagePositionPatient | Slice position (for gap/coverage checks) |
| (0020,0037) | ImageOrientationPatient | Slice orientation |
| (0020,1041) | SliceLocation | Relative slice position |
| (0051,100F) | Private: CoilString | Siemens-specific: receive coil name |

**Source:** NEMA. *DICOM PS3.3: Information Object Definitions.* Section C.8.3, "MR Modules." Living standard.
URL: https://dicom.nema.org/medical/dicom/current/output/chtml/part03/sect_C.8.3.html
Innolitics DICOM Browser: https://dicom.innolitics.com/ciods/mr-image/mr-image

---

## 2. Common MRI Pulse Sequences for Brain Imaging Research

### 2.1 T1-Weighted (MPRAGE / SPGR / IR-FSPGR)

**Purpose:** High-resolution structural imaging for volumetric analysis, cortical/subcortical segmentation, cortical thickness measurement.

**Defining DICOM characteristics:**
- ScanningSequence: GR (gradient echo) + IR (inversion recovery)
- SequenceVariant: SP (spoiled), MP (MAG prepared)
- MRAcquisitionType: 3D
- Short TE (1.5-4 ms), moderate TR (1900-2500 ms), TI 800-1100 ms
- Flip angle 7-12 degrees
- Isotropic or near-isotropic voxels (0.7-1.2 mm)

**Vendor sequence names:**
- Siemens: MPRAGE (tfl3d1)
- GE: BRAVO / IR-FSPGR
- Philips: 3D TFE

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 1900 | 2200-2400 | 2600 | Longer at higher fields |
| TE (ms) | 1.5 | 2.0-3.0 | 4.5 | Minimum full echo |
| TI (ms) | 800 | 900-1060 | 1100 | Optimized for GM/WM contrast |
| Flip Angle (deg) | 7 | 8-9 | 12 | Ernst angle dependent |
| Slice Thickness (mm) | 0.7 | 1.0 | 1.2 | Isotropic preferred |
| In-plane Resolution (mm) | 0.7 | 1.0 | 1.2 | |
| Bandwidth (Hz/px) | 150 | 210-240 | 300 | |

### 2.2 T2-Weighted (TSE / FSE / SPACE)

**Purpose:** White/gray matter contrast, lesion detection, CSF visualization, myelin mapping (HCP).

**Defining DICOM characteristics:**
- ScanningSequence: SE (spin echo)
- SequenceVariant: SK (segmented k-space)
- Long TR (2500-3200 ms for 3D), long TE (75-560 ms)
- High echo train length
- 3D acquisitions common for research (e.g., Siemens SPACE, GE CUBE)

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 2500 | 3200 | 6000 | Longer for 2D multi-slice |
| TE (ms) | 75 | 100-400 | 560 | Varies greatly by application |
| Slice Thickness (mm) | 0.7 | 1.0 | 3.0 | 3D: 0.7-1.0 mm |
| Echo Train Length | 80 | 120-167 | 200+ | High for 3D SPACE/CUBE |

### 2.3 FLAIR (T2-FLAIR)

**Purpose:** White matter hyperintensity detection, inflammation, tissue damage — CSF signal suppressed.

**Defining DICOM characteristics:**
- ScanningSequence: SE + IR (or GR + IR for 3D)
- Inversion pulse to null CSF (TI ~ 1500-1800 ms at 3T, ~2200-2500 ms at 1.5T)
- Long TR, long TE

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 4800 | 5000-6000 | 9000 | 3D: 5000; 2D: 8000-9000 |
| TE (ms) | 80 | 100-120 | 140 | Effective TE for 3D |
| TI (ms) | 1400 | 1500-1800 | 1800 | 3T CSF null; 1.5T: 2200-2500 |
| Slice Thickness (mm) | 1.0 | 1.0-3.0 | 5.0 | 3D: 1.0 mm preferred |
| Flip Angle (deg) | — | 120 (refocus) | — | Variable for 3D SPACE |

### 2.4 DWI / DTI (Diffusion-Weighted Imaging)

**Purpose:** White matter tract mapping, diffusion parameters (FA, MD, AD, RD), stroke detection.

**Defining DICOM characteristics:**
- ScanningSequence: EP (echo planar) + SE (spin echo)
- DiffusionBValue present (b=0, 1000, 2000, 3000 s/mm^2)
- Multiple gradient directions (6 minimum for tensor, 30+ for HARDI)
- Single-shot EPI readout

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 3000 | 5000-9000 | 12000 | Depends on coverage |
| TE (ms) | 57 | 70-92 | 100 | Minimum achievable |
| Flip Angle (deg) | 90 | 90 | 90 | SE-EPI always 90 |
| b-value (s/mm^2) | 0 | 1000, 2000, 3000 | 5000 | Multi-shell preferred |
| Directions | 6 | 30-100 | 270+ | 6=DTI, 30+=HARDI |
| Resolution (mm) | 1.25 | 2.0-2.5 | 3.0 | Isotropic preferred |
| Multiband Factor | 1 | 2-3 | 4 | For acceleration |

### 2.5 fMRI (BOLD EPI)

**Purpose:** Functional brain mapping via blood oxygen level dependent contrast.

**Defining DICOM characteristics:**
- ScanningSequence: EP + GR (gradient echo EPI) for standard BOLD
- TE optimized for T2* (TE ~ T2* of tissue at field strength)
- Short TR for temporal resolution
- Multiband/SMS acceleration common

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 300 | 700-1500 | 4000 | <1s with multiband |
| TE (ms) | 25 | 30-35 | 40 | Optimized for T2* at 3T |
| Flip Angle (deg) | 45 | 52-75 | 90 | Ernst angle at given TR |
| Resolution (mm) | 1.5 | 2.0-3.0 | 4.0 | 2mm for HCP-style |
| Slice Thickness (mm) | 1.5 | 2.0-3.4 | 4.0 | Isotropic preferred |
| Multiband Factor | 1 | 2-8 | 8 | 2-4 recommended |
| Bandwidth (Hz/px) | 1500 | 2000-2500 | 3000 | Higher = less distortion |

### 2.6 SWI (Susceptibility-Weighted Imaging)

**Purpose:** Microbleed detection, iron content mapping, venous visualization.

**Defining DICOM characteristics:**
- ScanningSequence: GR (gradient echo)
- 3D acquisition, long TE, small flip angle
- Flow compensation in all 3 directions
- Phase images used alongside magnitude

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 27 | 29-45 | 50 | Short TR for 3D GRE |
| TE (ms) | 15 | 20-28 | 30 | Long TE for susceptibility |
| Flip Angle (deg) | 10 | 15 | 20 | Small flip angle |
| Resolution (mm) | 0.5 | 0.8-1.0 | 1.5 | High in-plane resolution |
| Slice Thickness (mm) | 1.0 | 1.8-3.0 | 4.0 | Thin slices preferred |
| Bandwidth (Hz/px) | 80 | 100 | 150 | Low bandwidth typical |

### 2.7 MRA (Time-of-Flight)

**Purpose:** Non-contrast angiography, vascular anatomy visualization.

**Defining DICOM characteristics:**
- ScanningSequence: GR
- AngioFlag: Y
- MRAcquisitionType: 3D (typically)
- Short TR, short TE, moderate-to-large flip angle

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 15 | 20-28 | 35 | Short for background suppression |
| TE (ms) | 2.5 | 3.5-5.0 | 7.0 | Short to minimize phase dispersion |
| Flip Angle (deg) | 15 | 20-25 | 60 | Moderate-to-large |
| Resolution (mm) | 0.3 | 0.5-0.8 | 1.0 | High in-plane resolution |
| Slice Thickness (mm) | 0.5 | 0.7-1.0 | 2.0 | Thin for 3D |

### 2.8 ASL (Arterial Spin Labeling)

**Purpose:** Non-contrast cerebral perfusion measurement (cerebral blood flow).

**Defining DICOM characteristics:**
- ScanningSequence: EP (EPI readout)
- ScanOptions may include "ASL"
- Specific ASL tags: (0018,9250) ASL Context, labeling duration, post-label delay

**Typical parameter ranges at 3T:**

| Parameter | Min | Typical | Max | Notes |
|-----------|-----|---------|-----|-------|
| TR (ms) | 3500 | 4000-5200 | 6000 | Long for full relaxation |
| TE (ms) | 9 | 12-32 | 40 | Short preferred |
| Labeling Duration (ms) | 1200 | 1450-1800 | 2000 | pCASL standard |
| Post-Label Delay (ms) | 1200 | 1500-2000 | 2500 | Age-dependent |
| Resolution (mm) | 1.9 | 3.0-4.0 | 5.0 | Coarser for SNR |
| Slice Thickness (mm) | 4.0 | 4.0-8.0 | 10.0 | Thick for SNR |

**Source for typical parameter ranges:** Compiled from mriquestions.com (AD Elster), Radiopaedia, and published consortium protocols. See individual consortium sections below for study-specific values.

---

## 3. Standard Protocols from Major Consortia

### 3.1 ADNI (Alzheimer's Disease Neuroimaging Initiative)

The ADNI MRI Core (Mayo Clinic) has defined standardized protocols across four phases (ADNI-1/GO/2/3/4). All ADNI-3 and ADNI-4 imaging is performed on 3T scanners from Siemens, GE, and Philips. Over 60 scanners are certified for ADNI.

**ADNI4 Core Protocol Parameters (from Arani et al. 2024):**

**T1w MP-RAGE:**

| Parameter | Value |
|-----------|-------|
| Resolution | 1 x 1 x 1 mm isotropic |
| FOV | 240 x 256 x 208 mm |
| TR | 2300 ms |
| TE | Minimum full (vendor-dependent, ~2-3 ms) |
| TI | 900 ms |
| Flip Angle | ~8-12 degrees |
| Acceleration | 2x (GRAPPA/ARC/SENSE) |
| Duration | ~5:12 |

**3D T2-FLAIR:**

| Parameter | Value |
|-----------|-------|
| Resolution | 1 x 1 x 1 mm isotropic |
| FOV | 240 x 256 x 208 mm |
| TR | 5000 ms |
| Effective TE | 104 ms |
| TI | 1526-1700 ms (vendor-specific) |
| Duration | ~6:20 |

**Diffusion MRI:**

| Parameter | Value |
|-----------|-------|
| Resolution | 2 x 2 x 2 mm isotropic |
| FOV | 232 x 232 x 162 mm |
| TR | 3306-3400 ms |
| TE | 70-82 ms (vendor-dependent) |

**Task-Free fMRI (BOLD):**

| Parameter | Basic | Advanced |
|-----------|-------|----------|
| Resolution | 3.4 mm iso | 2.5 mm iso |
| TR | 1500 ms | 600 ms |
| TE | 30 ms | 30 ms |
| Flip Angle | 90 deg | 50 deg |

**T2* GRE (Susceptibility):**

| Parameter | Basic | Advanced |
|-----------|-------|----------|
| Resolution | 0.86 x 0.86 x 4 mm | 0.5 x 0.5 x 1.8 mm |
| TR | 650 ms | 37 ms |
| TE | 20 ms | 6.71-22.35 ms (5 echoes) |

**Multi-PLD ASL:**

| Parameter | Value |
|-----------|-------|
| Resolution | 1.9 x 1.9 x 4.0-4.5 mm |
| FOV | 240 x 240 x 144-160 mm |
| Duration | 7:31-8:20 |

**Quality control:** Two-level QC: (1) adherence to protocol parameters, (2) series-specific quality (motion, coverage). Scans graded 1-3 = acceptable, 4 = failure.

**Key citation:**
> Arani A, Swihart DI, Schwarz CG, Cha RH, Olson ML, Sodhi A, Takahashi N, Reid RI, Kantarci K, Vemuri P, Petersen RC, Gunter JL, Jack CR Jr. "Design and validation of the ADNI MR protocol." *Alzheimer's & Dementia*. 2024;20(10):e14162.
> DOI: [10.1002/alz.14162](https://alz-journals.onlinelibrary.wiley.com/doi/full/10.1002/alz.14162)
> PMC: PMC11497751
> **Source type:** Peer-reviewed. *Alzheimer's & Dementia* (Wiley).

**Supporting reference for protocol overview:**
> Jack CR Jr, Wiste HJ, Weigand SD, Therneau TM, Lowe VJ, Knopman DS, Gunter JL, Senjem ML, Jones DT, Kantarci K, Machulda MM, Mielke MM, Roberts RO, Vemuri P, Reyes DA, Petersen RC. "Overview of ADNI MRI." *Alzheimer's & Dementia*. 2024;20(10):e14166.
> DOI: [10.1002/alz.14166](https://alz-journals.onlinelibrary.wiley.com/doi/full/10.1002/alz.14166)
> PMC: PMC11485416
> **Source type:** Peer-reviewed.

### 3.2 HCP (Human Connectome Project)

The WU-Minn HCP uses a customized Siemens 3T Connectome Skyra with 100 mT/m gradients. Parameters are highly specific to this scanner but widely used as reference for high-quality research protocols.

**HCP 3T Structural:**

| Sequence | TR (ms) | TE (ms) | TI (ms) | FA (deg) | Resolution | FOV | Accel | Duration |
|----------|---------|---------|---------|----------|------------|-----|-------|----------|
| T1w MPRAGE | 2400 | 2.14 | 1000 | 8 | 0.7 mm iso | 224x320 | GRAPPA 2 | 7:40 |
| T2w SPACE | 3200 | 565 | — | Variable | 0.7 mm iso | 224x320 | GRAPPA 2 | 8:24 |

**HCP 3T Diffusion:**

| Parameter | Value |
|-----------|-------|
| Resolution | 1.25 mm isotropic |
| FOV PE x Readout | 210 x 180 mm |
| Matrix | 144 x 168 |
| Slices | 111 |
| TR / TE | 5520 / 89.5 ms |
| Multiband Factor | 3 |
| b-values | 1000, 2000, 3000 s/mm^2 |
| Directions | 90 per shell (270 total) |

**HCP 3T fMRI (Resting-State):**

| Parameter | Value |
|-----------|-------|
| Resolution | 2.0 mm isotropic |
| TR | 720 ms |
| TE | 33.1 ms |
| Flip Angle | 52 deg |
| Multiband Factor | 8 |
| FOV | 208 x 180 mm |
| Slices | 72 |

**Key citation:**
> Glasser MF, Sotiropoulos SN, Wilson JA, Coalson TS, Fischl B, Andersson JL, Xu J, Jbabdi S, Webster M, Polimeni JR, Van Essen DC, Jenkinson M; WU-Minn HCP Consortium. "The Minimal Preprocessing Pipelines for the Human Connectome Project." *NeuroImage*. 2013 Oct 15;80:105-124.
> DOI: [10.1016/j.neuroimage.2013.04.127](https://www.sciencedirect.com/science/article/abs/pii/S1053811913005053)
> PMC: PMC3720813
> **Source type:** Peer-reviewed. *NeuroImage* (Elsevier).

**Supporting citations:**
> Sotiropoulos SN, Jbabdi S, Xu J, Andersson JL, Moeller S, Auerbach EJ, Glasser MF, Hernandez M, Sapiro G, Jenkinson M, Feinberg DA, Yacoub E, Lenglet C, Van Essen DC, Ugurbil K, Behrens TE. "Advances in diffusion MRI acquisition and processing in the Human Connectome Project." *NeuroImage*. 2013 Oct 15;80:125-143.
> PMC: PMC3720790

> Van Essen DC, Smith SM, Barch DM, Behrens TE, Yacoub E, Ugurbil K; WU-Minn HCP Consortium. "The Human Connectome Project's Neuroimaging Approach." *Nature Neuroscience*. 2012;15:E2-E4.
> PMC: PMC6172654

### 3.3 ABCD Study (Adolescent Brain Cognitive Development)

The ABCD study harmonizes HCP-style imaging across 21 sites using Siemens Prisma/Prisma Fit, GE MR750, and Philips Achieva dStream/Ingenia 3T scanners. Protocol achieves HCP-style temporal and spatial resolution without non-commercial upgrades.

**ABCD Key Parameters:**

| Sequence | TR (ms) | TE (ms) | FA (deg) | Resolution | MB Factor |
|----------|---------|---------|----------|------------|-----------|
| T1w MPRAGE | ~2500 | ~2-3 | 8 | 1.0 mm iso | — |
| T2w | ~3200 | ~500+ | variable | 1.0 mm iso | — |
| rfMRI (rest) | 800 | 30 | 52 | 2.4 mm iso | 6 |
| tfMRI (task) | 800 | 30 | 52 | 2.4 mm iso | 6 |
| DWI | ~4100 | ~88 | — | 1.7 mm iso | 3 |

**Non-compliance findings (from mrQA analysis):**

| Modality | Vendor | Non-Compliance Rate (5% tolerance) | Primary Issue |
|----------|--------|-------------------------------------|---------------|
| T1w | Philips | 64.43% | TE varies 1.4-3.56 ms |
| T1w | GE | 2.0% | TE, TR, Pixel Bandwidth |
| T1w | Siemens | 39.96% | Minor TE variation (2.88-2.9 ms) |
| T2w | Philips | 62.82% | TE varies 251.49-285.23 ms |
| T2w | GE | 32.19% | TE, TR variations |
| rfMRI | Siemens | 40.34% | iPAT parameter issues |

**Key citation:**
> Casey BJ, Cannonier T, Conley MI, et al. "The Adolescent Brain Cognitive Development (ABCD) study: Imaging acquisition across 21 sites." *Developmental Cognitive Neuroscience*. 2018 Aug;32:43-54.
> DOI: [10.1016/j.dcn.2018.03.001](https://www.sciencedirect.com/science/article/pii/S1878929317301214)
> **Source type:** Peer-reviewed.

**Protocol implementation reference:**
> Kang K, Ramirez de Noriega F, Belyavka V, Chaudhary S, Sabaroedin K, Tassarotti L, Whitaker L, Fornito A. "Implementing ABCD study MRI sequences for multi-site cohort studies: Practical guide to necessary steps, preprocessing methods, and challenges." *MethodsX*. 2024 Jun;12:102762.
> DOI: [10.1016/j.mex.2024.102762](https://www.sciencedirect.com/science/article/pii/S2215016124002425)
> **Source type:** Peer-reviewed.

### 3.4 UK Biobank Brain Imaging

All UK Biobank brain MRI is acquired on a single 3T Siemens Skyra (VD13 software, 32-channel head coil). Six modalities in ~35 minutes.

**UK Biobank Parameters:**

| Sequence | TR (ms) | TE (ms) | TI (ms) | FA (deg) | Resolution | Other |
|----------|---------|---------|---------|----------|------------|-------|
| T1 MPRAGE | 2000 | 2.01 | 880 | 8 | 1x1x1 mm iso | GRAPPA 2; 4:54 |
| T2 FLAIR | 1800 | 395 | 1800 | — | 1.05x1x1 mm | 3D SPACE; 5:52 |
| SWI | — | 9.4 & 20 | — | — | 0.8x0.8x3 mm | Dual echo; 2:37 |
| dMRI | — | 92 | — | — | 2x2x2 mm iso | MB=3; b=1000,2000; 100 dirs; 7:08 |
| rfMRI | 0.735s* | — | — | — | 2.4x2.4x2.4 mm | MB=8; 6:10 |
| tfMRI | 0.735s* | — | — | — | 2.4x2.4x2.4 mm | MB=8; 2:00 |

*TR=735 ms (0.735 s) for fMRI.

**Key citation:**
> Miller KL, Alfaro-Almagro F, Bangerter NK, Thomas DL, Yacoub E, Xu J, Bartsch AJ, Jbabdi S, Sotiropoulos SN, Andersson JL, Griffanti L, Douaud G, Okell TW, Weale P, Dragonu I, Gaez S, Sherif N, Smith SM. "Multimodal population brain imaging in the UK Biobank prospective epidemiological study." *Nature Neuroscience*. 2016 Nov;19(11):1523-1536.
> DOI: [10.1038/nn.4393](https://www.nature.com/articles/nn.4393)
> PMC: PMC5086094
> **Source type:** Peer-reviewed. *Nature Neuroscience* (Springer Nature).

### 3.5 ENIGMA Consortium

ENIGMA takes a different approach: rather than prescribing acquisition parameters, it provides standardized **analysis protocols** (FreeSurfer, VBM, tract-based spatial statistics) and performs **meta-analyses** across sites using whatever parameters were locally available.

**Approach:** Site-wise analysis with standardized pipelines, then meta-analysis of summary statistics. Does not mandate specific acquisition parameters but recommends:
- T1w structural at 1 mm isotropic resolution
- DTI with at least 30 directions and b=1000 s/mm^2
- Minimum 1.5T field strength

**Key citation:**
> Thompson PM, Stein JL, Medland SE, Hibar DP, et al. "The ENIGMA Consortium: large-scale collaborative analyses of neuroimaging and genetic data." *Brain Imaging and Behavior*. 2014 Jun;8(2):153-182.
> DOI: [10.1007/s11682-013-9269-5](https://link.springer.com/article/10.1007/s11682-013-9269-5)
> PMC: PMC4008818
> **Source type:** Peer-reviewed.

**fMRI guidance:**
> Dennis EL, et al. "ENIGMA's simple seven: Recommendations to enhance the reproducibility of resting-state fMRI in traumatic brain injury." *NeuroImage: Clinical*. 2024;41:103562.
> DOI: [10.1016/j.nicl.2024.103562](https://www.sciencedirect.com/science/article/pii/S221315822400024X)
> **Source type:** Peer-reviewed.

### 3.6 ACR (American College of Radiology) MRI Accreditation

The ACR MRI Accreditation Program focuses on **clinical quality** rather than research protocol compliance. Requirements apply to clinical imaging centers.

**Requirements:**
- Phantom testing with ACR MRI phantom: 11 slices of 5 mm thickness using standard T1w and T2w brain protocols
- In-plane resolution: FOV / acquisition matrix (e.g., 18 cm FOV with matrix >= 240)
- Tests: slice thickness accuracy, spatial resolution, low-contrast detectability, geometric accuracy
- SNR measurement requirements
- 4-7 clinical exams per unit required for submission
- Required exams: brain, C-spine, L-spine, knee (for full body program)

**Key source:**
> American College of Radiology. "Complete Accreditation Information: MRI." Revised April 9, 2025.
> URL: https://accreditationsupport.acr.org/support/solutions/articles/11000063276
> **Source type:** Primary standard from professional body.

> American College of Radiology. "Quality Control: MRI/Breast MRI." Revised July 3, 2024.
> URL: https://accreditationsupport.acr.org/support/solutions/articles/11000061043
> **Source type:** Primary standard from professional body.

---

## 4. Acceptable Parameter Ranges by Field Strength

### 4.1 T1-Weighted (MPRAGE)

| Parameter | 1.5T | 3T | 7T |
|-----------|------|-----|-----|
| TR (ms) | 1900-2200 | 2200-2500 | 4000-6000 |
| TE (ms) | 3.0-4.5 | 1.5-3.0 | 1.5-3.0 |
| TI (ms) | 700-900 | 880-1100 | 1050-1200 |
| Flip Angle (deg) | 9-15 | 7-12 | 4-8 |
| Resolution (mm) | 1.0-1.5 | 0.7-1.2 | 0.5-1.0 |

### 4.2 T2-Weighted (FSE/TSE)

| Parameter | 1.5T | 3T | 7T |
|-----------|------|-----|-----|
| TR (ms) | 3000-6000 | 2500-4000 (3D) | 3000-5000 |
| TE (ms) | 80-120 | 75-120 (2D), 300-565 (3D SPACE) | 60-100 |
| Resolution (mm) | 1.0-1.5 | 0.7-1.5 | 0.5-1.0 |

### 4.3 FLAIR

| Parameter | 1.5T | 3T |
|-----------|------|-----|
| TR (ms) | 8000-11000 (2D), 5000-6000 (3D) | 5000-9000 |
| TE (ms) | 100-140 | 80-120 (3D) |
| TI (ms) | 2200-2500 | 1500-1800 |

### 4.4 DWI/DTI

| Parameter | 1.5T | 3T |
|-----------|------|-----|
| TR (ms) | 6000-12000 | 3000-9000 |
| TE (ms) | 80-120 | 57-92 |
| b-value (s/mm^2) | 1000 | 1000, 2000, 3000 |
| Resolution (mm) | 2.0-3.0 | 1.25-2.5 |

### 4.5 fMRI (BOLD)

| Parameter | 1.5T | 3T | 7T |
|-----------|------|-----|-----|
| TR (ms) | 2000-4000 | 300-2000 | 500-1500 |
| TE (ms) | 40-60 | 28-36 | 18-26 |
| Flip Angle (deg) | 70-90 | 45-80 | 40-70 |
| Resolution (mm) | 3.0-4.0 | 1.5-3.0 | 0.8-2.0 |

### 4.6 SWI

| Parameter | 1.5T | 3T | 7T |
|-----------|------|-----|-----|
| TR (ms) | 40-60 | 27-50 | 15-35 |
| TE (ms) | 30-40 | 15-28 | 10-20 |
| Flip Angle (deg) | 15-20 | 10-20 | 10-15 |

### 4.7 Tissue Relaxation Values (for parameter optimization context)

| Tissue | T1 at 1.5T (ms) | T1 at 3T (ms) | T1 at 7T (ms) |
|--------|-----------------|---------------|---------------|
| Gray Matter | 1197 +/- 134 | 1607 +/- 112 | 1939 +/- 149 |
| White Matter | 646 +/- 32 | 838 +/- 50 | 1126 +/- 97 |
| CSF | ~4000 | ~4000 | ~4000 |

**Source for relaxation values:**
> Wansapura JP, Holland SK, Dunn RS, Ball WS Jr. "NMR relaxation times in the human brain at 3.0 Tesla." *Journal of Magnetic Resonance Imaging*. 1999;9(4):531-538.

> Wright PJ, Mougin OE, Totman JJ, Peters AM, Sheridan CJ, et al. "Water proton T1 measurements in brain tissue at 7, 3, and 1.5T using IR-EPI, IR-TSE, and MPRAGE." *Magnetic Resonance Materials in Physics, Biology and Medicine*. 2008;21:121-130.
> DOI: [10.1007/s10334-008-0104-8](https://link.springer.com/article/10.1007/s10334-008-0104-8)
> **Source type:** Peer-reviewed.

---

## 5. Tolerance and Compliance Thresholds

### 5.1 Recommended Tolerance Levels

No universal standard exists for MRI protocol compliance tolerances. The mrQA tool uses configurable thresholds and reports non-compliance at multiple tolerance levels:

| Tolerance Level | Use Case | Notes |
|-----------------|----------|-------|
| 0% (exact match) | Ideal / QC audit | Most parameters will show deviations |
| 1% | Strict compliance | Catches minor vendor/software variations |
| 5% | Standard research | Recommended default for multisite studies |
| 10% | Lenient / screening | May miss meaningful deviations |

### 5.2 Practical Guidance for Tolerance by Parameter

Based on findings from the mrQA paper (Ravi et al. 2024) and ABCD study analysis:

| Parameter | Strict Tolerance | Recommended Tolerance | Notes |
|-----------|-----------------|----------------------|-------|
| TR | +/- 1% | +/- 5% | Vendor variations expected |
| TE | +/- 2% | +/- 10% | Most variable; minimum-TE auto-selection by vendors |
| TI | +/- 1% | +/- 5% | Critical for FLAIR and MPRAGE contrast |
| Flip Angle | +/- 1 degree | +/- 2 degrees | Or +/- 5% of nominal |
| Slice Thickness | +/- 5% | +/- 10% | Scanner calibration dependent |
| In-plane Resolution | +/- 5% | +/- 10% | Reconstruction-dependent |
| Pixel Bandwidth | +/- 5% | +/- 10% | Vendor default variations |
| Phase Encoding Direction | Exact match | Exact match | Must be consistent |
| Number of Directions (DTI) | Exact match | Exact match | Protocol-defined |
| b-value | +/- 1% | +/- 5% | Critical for diffusion metrics |

### 5.3 Non-Compliance Severity Classification

Suggested severity levels for protocol deviations:

**Critical (reject/flag):**
- Wrong sequence type (e.g., T2 when T1 expected)
- Wrong field strength
- Missing required sequence from protocol
- Resolution >2x expected (e.g., 2mm when 1mm specified)
- Wrong phase encoding direction

**Warning (review):**
- Parameter deviation 5-20% from target
- Minor resolution difference (e.g., 1.0 vs 1.2 mm)
- Slight TR/TE variation within vendor tolerance

**Info (log only):**
- Parameter deviation <5% from target
- Known vendor-specific minor variations (e.g., Siemens TE 2.88-2.90)

---

## 6. Existing Tools and Standards

### 6.1 mrQA

**Description:** Open-source Python tool for automatic evaluation of protocol compliance in MRI datasets. Reads DICOM headers (via pydicom) or BIDS JSON sidecars. Compares acquisition parameters against a reference protocol and generates HTML compliance reports with percent scores.

**Parameters checked:** RepetitionTime, EchoTime, FlipAngle, PixelBandwidth, PhaseEncodingDirection, EchoTrainLength, PhaseEncodingSteps, SequenceVariant, MRAcquisitionType, MagneticFieldStrength, parallel imaging (iPAT/GRAPPA from Siemens private headers).

**Input:** DICOM files or BIDS dataset
**Output:** HTML report with per-protocol compliance percentage

**Key finding:** In analysis of >20 open neuroimaging datasets including ABCD, found non-compliance rates of 0.19% to 64.43% depending on vendor, modality, and tolerance level.

**Repository:** https://github.com/Open-Minds-Lab/mrQA
**Documentation:** https://open-minds-lab.github.io/mrQA/

**Key citation:**
> Ravi H, Anand VK, Thompson PM, Prabhakaran V, Raamana PR. "Solving the Pervasive Problem of Protocol Non-Compliance in MRI using an Open-Source tool mrQA." *Neuroinformatics*. 2024;22(3):293-308.
> DOI: [10.1007/s12021-024-09668-4](https://link.springer.com/article/10.1007/s12021-024-09668-4)
> PMC: PMC11329586
> **Source type:** Peer-reviewed. *Neuroinformatics* (Springer Nature).

### 6.2 Protocol Checker (XNAT-integrated)

**Description:** Python-based tool integrated into the XNAT imaging repository. Generates a "master file" during site qualification that defines expected imaging protocol parameters, then compares newly acquired DICOM series against this master file. Highlights missing series and parameter deviations.

**Key finding:** Statistically significant correlation between protocol compliance and overall exam radiological image quality.

**Key citation:**
> [Authors not fully available from search]. "An open-source repository-based tool for quality control of imaging protocol compliance: demonstration in a multicentre MRI study." *British Journal of Radiology*. 2025;98(1172):1236.
> DOI: Available at https://academic.oup.com/bjr/article/98/1172/1236/8160018
> **Source type:** Peer-reviewed. *British Journal of Radiology* (Oxford Academic).

### 6.3 MRIAcqParameterCheck

**Description:** MATLAB script for checking MRI acquisition parameters in DICOM image files against a template. Compares parameters from DICOM headers against user-defined reference values.

**Repository:** https://github.com/jamtheim/MRIAcqParameterCheck

**Source type:** Open-source tool (not peer-reviewed).

### 6.4 BIDS Validator

**Description:** Validates Brain Imaging Data Structure (BIDS) dataset organization and metadata. Checks that required metadata fields are present in JSON sidecars but does NOT validate that acquisition parameter values are within acceptable ranges for a specific protocol.

**What it checks:**
- Required metadata fields present (RepetitionTime, EchoTime, FlipAngle)
- Consistent parameters within a logical group of files
- File naming conventions
- JSON sidecar structure

**What it does NOT check:**
- Whether TR/TE/FA are within expected ranges for a given protocol
- Whether the sequence is correct for the intended modality
- Protocol-level compliance against a reference standard

**Repository:** https://github.com/bids-standard/bids-validator

**Key citation:**
> Gorgolewski KJ, Auer T, Calhoun VD, et al. "The brain imaging data structure, a format for organizing and describing outputs of neuroimaging experiments." *Scientific Data*. 2016;3:160044.
> DOI: 10.1038/sdata.2016.44

### 6.5 dcm2niix

**Description:** DICOM to NIfTI converter that creates JSON sidecar files containing acquisition parameters extracted from DICOM headers. Does NOT perform protocol compliance checking, but the JSON sidecars it produces are the input format for BIDS validator and mrQA.

**Relevant output fields in JSON sidecar:**
RepetitionTime, EchoTime, InversionTime, FlipAngle, SliceThickness, SpacingBetweenSlices, AcquisitionMatrix, PhaseEncodingDirection, PixelBandwidth, MagneticFieldStrength, Manufacturer, ManufacturerModelName, SoftwareVersions, ProtocolName, SeriesDescription

**Repository:** https://github.com/rordenlab/dcm2niix

### 6.6 DVTk (DICOM Validation Toolkit)

**Description:** Open-source framework for testing, validating, and diagnosing DICOM communication. Validates DICOM conformance at the protocol/network level (SOP Classes, Transfer Syntaxes, IOD compliance) rather than acquisition parameter compliance.

**Repository:** https://github.com/dvtk-org/DVTk

### 6.7 dciodvfy (DICOM IOD Verify)

**Description:** Command-line tool by David Clunie for validating DICOM files against the IOD (Information Object Definition) specification. Checks required tags, correct VR (Value Representation), and IOD module requirements. Does not check acquisition parameter ranges.

**URL:** https://dclunie.com/dicom3tools/dciodvfy.html

---

## 7. Recommended Approach for Building Protocol Validation Rules

### 7.1 Architecture

Based on existing tools and consortia approaches, a protocol validation system should:

1. **Define reference protocols** as structured JSON/YAML configurations specifying:
   - Target values for each DICOM tag
   - Acceptable tolerance (percentage or absolute)
   - Severity level (critical/warning/info)
   - Per-vendor overrides where needed

2. **Extract parameters** from DICOM headers using pydicom (Python) or a DICOM parser

3. **Compare** each parameter against reference with configurable tolerance

4. **Report** compliance percentage and specific deviations

### 7.2 Minimum Tag Set for Protocol Compliance

For a brain MRI protocol compliance checker, the minimum set of DICOM tags to validate:

**Required for all MR sequences:**
- (0018,0087) MagneticFieldStrength
- (0018,0020) ScanningSequence
- (0018,0021) SequenceVariant
- (0018,0023) MRAcquisitionType (2D/3D)
- (0018,0080) RepetitionTime
- (0018,0081) EchoTime
- (0018,1314) FlipAngle
- (0018,0050) SliceThickness
- (0028,0010) Rows
- (0028,0011) Columns
- (0028,0030) PixelSpacing
- (0018,0095) PixelBandwidth
- (0018,1312) InPlanePhaseEncodingDirection

**Required for specific sequences:**
- (0018,0082) InversionTime — MPRAGE, FLAIR
- (0018,0091) EchoTrainLength — FSE/TSE, SPACE
- (0018,0083) NumberOfAverages — all (when specified)
- (0018,9087) DiffusionBValue — DWI/DTI
- Number of diffusion directions — DTI/HARDI (derive from series file count)

**Useful for identification:**
- (0008,103E) SeriesDescription
- (0018,1030) ProtocolName
- (0018,0024) SequenceName

### 7.3 Example Reference Protocol (ADNI4-like T1w)

```json
{
  "protocol_name": "ADNI4 T1w MPRAGE",
  "sequence_type": "T1w",
  "rules": {
    "MagneticFieldStrength": {
      "target": 3.0,
      "tolerance_pct": 1,
      "severity": "critical"
    },
    "ScanningSequence": {
      "expected": ["GR", "IR"],
      "match": "contains_all",
      "severity": "critical"
    },
    "MRAcquisitionType": {
      "expected": "3D",
      "match": "exact",
      "severity": "critical"
    },
    "RepetitionTime": {
      "target": 2300,
      "tolerance_pct": 10,
      "severity": "warning"
    },
    "EchoTime": {
      "target": 2.96,
      "tolerance_pct": 20,
      "severity": "info",
      "note": "Vendor auto-selects minimum TE"
    },
    "InversionTime": {
      "target": 900,
      "tolerance_pct": 5,
      "severity": "warning"
    },
    "FlipAngle": {
      "target": 9,
      "tolerance_abs": 3,
      "severity": "warning"
    },
    "SliceThickness": {
      "target": 1.0,
      "tolerance_pct": 20,
      "severity": "warning"
    },
    "PixelSpacing": {
      "target": [1.0, 1.0],
      "tolerance_pct": 20,
      "severity": "warning"
    }
  }
}
```

---

## 8. Key Findings for AEGIS Implementation

1. **No universal compliance standard exists.** Each consortium defines its own protocol, and tolerance levels are study-specific. AEGIS should allow per-project protocol definitions.

2. **TE is the most variable parameter** across vendors, because many scanners auto-select minimum TE. Tight TE tolerances will generate excessive false positives.

3. **Vendor-specific variations are expected.** Siemens, GE, and Philips implement the same conceptual sequence differently. A compliance system must account for this.

4. **5% tolerance is the de facto standard** for multisite research studies, with configurable overrides per parameter.

5. **mrQA is the closest existing tool** to what AEGIS needs. It is Python-based, uses pydicom, and generates HTML reports. Its architecture could inform AEGIS's approach.

6. **BIDS validator does not check parameter ranges.** It only validates presence and structure of metadata, not whether values match a protocol specification.

7. **Non-compliance is pervasive.** The mrQA study found rates from 0.19% to 64.43% across 20+ datasets. Continuous monitoring is essential.

8. **Severity-based reporting is key.** Not all deviations are equally important. A wrong sequence type is critical; a 0.02 ms TE variation is informational.

---

## Notes for Future Agents

- The mrQA paper (Ravi et al. 2024, Neuroinformatics) is the single most relevant reference for building MRI protocol compliance checking into AEGIS
- ADNI is the gold standard for multi-vendor protocol standardization in neuroimaging
- HCP parameters are aspirational (specialized hardware) — use ADNI or ABCD as more realistic targets
- UK Biobank is single-site (Siemens Skyra) so less relevant for multi-vendor compliance but excellent for parameter reference values
- ENIGMA does not prescribe acquisition parameters — it focuses on analysis harmonization
- The ACR focuses on clinical quality (phantom testing, resolution) rather than research protocol compliance
- Consider implementing protocol validation as a new check type in the existing QC service, or as part of the classification service pipeline
