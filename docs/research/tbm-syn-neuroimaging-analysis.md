# TBM-SyN (Tensor-Based Morphometry with Symmetric Normalization) — Technical Research

**Date**: 2026-02-27
**Purpose**: Technical reference for implementing a TBM-SyN analysis pipeline in a medical imaging analytics service

---

## 1. Core Methodology

### What TBM-SyN Is

TBM-SyN (Tensor-Based Morphometry with Symmetric Normalization) is a neuroimaging technique that measures volumetric brain changes between pairs of longitudinal T1-weighted MRI scans. It produces voxel-wise maps of local tissue expansion or contraction by computing the Jacobian determinant of a symmetric diffeomorphic deformation field that aligns two time-point images.

The method was developed at the Mayo Clinic (Jack Lab) by Prashanthi Vemuri, Matthew Senjem, Jeffrey Gunter, Christopher Schwarz, and Clifford Jack Jr., primarily for tracking neurodegeneration in Alzheimer's disease through the ADNI (Alzheimer's Disease Neuroimaging Initiative) consortium.

### The SyN Algorithm

SyN (Symmetric Normalization) is a symmetric diffeomorphic image registration algorithm from the ANTs (Advanced Normalization Tools) software suite, developed at the University of Pennsylvania by Brian Avants, Nicholas Tustison, and James Gee.

**Key properties:**

- **Symmetric**: Both images deform along a geodesic diffeomorphism to a fixed point midway between them. Results are identical regardless of input image order — this eliminates the directional bias that plagues asymmetric registration methods.
- **Diffeomorphic**: Transformations are guaranteed invertible (one-to-one, smooth, with smooth inverse). The inverse consistency error is typically less than a few thousandths of a millimeter within brain tissue.
- **Cross-Correlation metric**: The default similarity metric for intra-subject longitudinal registration is neighborhood cross-correlation (CC), which is robust to intensity inhomogeneities.

**Mathematical formulation** (from Avants et al. 2008):

The core optimization minimizes the symmetric energy:

```
E_sym(I,J) = inf ∫[0→0.5] {||v1(x,t)||^2_L + ||v2(x,t)||^2_L} dt
             + ∫_Omega |I(phi1(0.5)) - J(phi2(0.5))|^2 dOmega
```

Both images deform along the shape manifold toward a shared midpoint, ensuring results are identical regardless of input image order. Velocity fields are integrated via ODE: `dphi(x,t)/dt = v(phi(x,t),t)`. Regularization uses a Gaussian smoothing kernel approximating the Green's function for `L = nabla^2 + Id`.

### Jacobian Determinant Maps

The Jacobian determinant of the deformation field at each voxel quantifies local volume change:
- `det(J) > 1` = local expansion (e.g., ventricle enlargement, CSF space growth)
- `det(J) < 1` = local contraction (e.g., grey matter atrophy)
- `det(J) = 1` = no volume change

**Log transformation**: Jacobian values are log-transformed (`log(det(J))`) so that:
- Expansion and contraction are symmetric around zero
- Values are normally distributed (suitable for parametric statistics)
- `log(det(J)) > 0` = expansion, `log(det(J)) < 0` = contraction

**Annualization**: Log-Jacobian values are divided by the inter-scan interval in years to produce annualized atrophy rates, enabling comparison across subjects scanned at different intervals.

### Why Symmetric Registration Matters

Asymmetric registration (warping image B to match image A, but not vice versa) introduces systematic bias in longitudinal measurements. Vemuri et al. (2015) specifically note that asymmetric warping can cause "biologically implausible deceleration of atrophy" and introduces directional bias. Hua et al. (2011) demonstrated that enforcing inverse consistency reduced the offset in atrophy rate estimates from 1.4% to 0.28%.

---

## 2. ADNI Implementation — The Mayo Clinic TBM-SyN Pipeline

### Processing Pipeline (as described in Vemuri et al. 2015)

**Step 1 — Preprocessing (N3m images)**
- N3 intensity homogeneity correction (nonparametric nonuniform intensity normalization)
- Gradient field non-linearity correction
- SPM5 bias correction
- Output: "N3m" preprocessed images

**Step 2 — Brain segmentation and masking**
- SPM5 unified segmentation using custom tissue priors
- Creates brain masks and ventricle masks
- Uses custom template space (STAND400)

**Step 3 — Within-subject co-registration**
- SPM5-based mutual information co-registration
- Registers individual time-point images to an iteratively updated mean (max 10 iterations)

**Step 4 — Intensity balancing**
- White matter peak intensity mapped to constant value (20,000)
- CSF peak intensity mapped to constant value (5,000)
- Normalizes intensity scale across time points

**Step 5 — Secondary co-registration**
- Aladin rigid-body (6DOF) registration to baseline (from NiftyReg)
- Affine (9DOF) registration to mean space at 1mm isotropic resolution

**Step 6 — Differential Bias Correction (DBC)**
- Spline-based intensity remapping
- Log-transformed ratio images smoothed with 20mm Gaussian kernel
- Corrects residual intensity differences between time points

**Step 7 — SyN deformation**
- Symmetric diffeomorphic normalization computed bidirectionally using ANTs
- Late image is warped to early image space
- A "soft mean" of the warped late image and early image produces a synthetic early image
- Log-Jacobian determinant images saved
- Values annualized by dividing by inter-scan interval in years

**Step 8 — ROI mapping and grey matter segmentation**
- SPM5 unified segmentation applied to the synthetic "soft-mean" image
- ROI masks propagated from template space
- Grey matter probability maps used to weight the log-Jacobian values within each ROI

### The 31 AD-Signature ROIs

The TBM-SyN summary score is computed over 31 regions of interest (15 bilateral pairs + ventricles) from an in-house modified atlas of 119 grey matter regions plus one ventricular region. The regions characteristically affected in Alzheimer's disease include:

**Medial temporal lobe:**
- Amygdala (L/R)
- Hippocampus (L/R)
- Parahippocampal gyrus (L/R)
- Entorhinal cortex (L/R)

**Lateral temporal lobe:**
- Fusiform gyrus (L/R)
- Superior temporal gyrus (L/R)
- Middle temporal gyrus (L/R)
- Inferior temporal gyrus (L/R)
- Superior temporal pole (L/R)
- Middle temporal pole (L/R)

**Parietal lobe:**
- Angular gyrus (L/R)
- Precuneus (L/R)

**Occipital lobe:**
- Superior occipital (L/R)
- Middle occipital (L/R)
- Inferior occipital (L/R)

**Ventricular:**
- Ventricles (combined)

### The STAND400 Template

The custom template used by the Mayo Clinic pipeline is called "STAND400" — a study-specific average template built from 400 ADNI subjects. This is not publicly distributed (it is an in-house Mayo Clinic resource).

### The AD-Signature Meta-ROI Concept

Related work by the same group (Schwarz et al. 2016) identified the six most sensitive cortical regions for AD measurement:
- Entorhinal cortex
- Fusiform gyrus
- Parahippocampal gyrus
- Middle temporal gyrus
- Inferior temporal gyrus
- Angular gyrus

These six regions form the "AD-signature meta-ROI" composite measure. The TBM-SyN summary score averages log-Jacobian values (weighted by grey matter probability) across the full 31-region set, but smaller composites of these six regions are also used.

### Software Used in ADNI

- MATLAB R2013a (MathWorks)
- ANTs 1.9.x (Penn Image Computing and Science Lab)
- SPM5 (Wellcome Trust Center for Neuroimaging, UCL)
- Aladin / NiftyReg (for rigid-body co-registration)

### ADNI4 Status

**TBM-SyN has been discontinued in ADNI4.** Per Jack et al. (2024): "This is a precise longitudinal morphometric measure that was included in ADNI 2/GO and ADNI 3. However, it was not highly used by the ADNI user base, which has overwhelmingly preferred the FreeSurfer output for morphometric analyses." As of April 2024, 12,019 TBM/TBM-SyN analyses were available in the ADNI archive.

---

## 3. Key Publications

### Primary TBM-SyN Methodology Paper

**Vemuri P, Senjem ML, Gunter JL, et al.** Accelerated vs. unaccelerated serial MRI based TBM-SyN measurements for clinical trials in Alzheimer's disease. *NeuroImage*. 2015;113:61-69. doi:[10.1016/j.neuroimage.2015.03.026](https://doi.org/10.1016/j.neuroimage.2015.03.026). PMID: [25797830](https://pubmed.ncbi.nlm.nih.gov/25797830/). PMC: [PMC4456670](https://pmc.ncbi.nlm.nih.gov/articles/PMC4456670/).

This is the definitive description of the Mayo Clinic TBM-SyN pipeline applied to ADNI data. It describes the full processing pipeline, validates that accelerated MRI scans produce comparable TBM-SyN measurements to unaccelerated scans, and reports the 31 AD-signature ROI approach. **Source type: peer-reviewed journal article.**

### TBM-SyN Applied to AD Biomarker Studies

**Jack CR Jr, Wiste HJ, Knopman DS, et al.** Rates of beta-amyloid accumulation are independent of hippocampal neurodegeneration. *Neurology*. 2014;82(17):1605-1612. doi:[10.1212/WNL.0000000000000386](https://doi.org/10.1212/WNL.0000000000000386).

Uses TBM-SyN to measure atrophy rates and shows that the AD-signature meta-ROI with TBM-SyN produces more precise rate measures than hippocampal volumes. **Source type: peer-reviewed journal article.**

### Evolution of Neurodegeneration Biomarkers

**Vemuri P, Wiste HJ, Weigand SD, et al.** Evolution of neurodegeneration-imaging biomarkers from clinically normal to dementia in the Alzheimer disease spectrum. *Radiology*. 2016;281(2):527-535. PMC: [PMC5018437](https://pmc.ncbi.nlm.nih.gov/articles/PMC5018437/).

Uses 14 AAL atlas cortical regions with TBM-SyN grey matter volume to track disease progression across the AD continuum. **Source type: peer-reviewed journal article.**

### Large-Scale Comparison of Cortical Thickness and Volume Methods

**Schwarz CG, Gunter JL, Wiste HJ, et al.** A large-scale comparison of cortical thickness and volume methods for measuring Alzheimer's disease severity. *NeuroImage: Clinical*. 2016;11:802-812. doi:[10.1016/j.nicl.2016.05.017](https://doi.org/10.1016/j.nicl.2016.05.017). PMC: [PMC5187496](https://pmc.ncbi.nlm.nih.gov/articles/PMC5187496/).

Identifies the six-region AD-signature meta-ROI (entorhinal, fusiform, parahippocampal, mid-temporal, inferior temporal, angular gyrus). FreeSurfer thickness achieved highest diagnostic separability (AUROC 0.86); SPM+DiReCT volume was most reliable. **Source type: peer-reviewed journal article.**

### ADNI MRI Overview (Discontinuation of TBM-SyN)

**Jack CR Jr, Arani A, Borowski BJ, et al.** Overview of ADNI MRI. *Alzheimer's & Dementia*. 2024;20(10):7350-7360. doi:[10.1002/alz.14166](https://doi.org/10.1002/alz.14166). PMC: [PMC11485416](https://pmc.ncbi.nlm.nih.gov/articles/PMC11485416/).

Comprehensive overview of all ADNI MRI processing pipelines. Documents TBM-SyN discontinuation in ADNI4 in favor of FreeSurfer. **Source type: peer-reviewed journal article.**

### Original SyN Algorithm Paper

**Avants BB, Epstein CL, Grossman M, Gee JC.** Symmetric diffeomorphic image registration with cross-correlation: Evaluating automated labeling of elderly and neurodegenerative brain. *Medical Image Analysis*. 2008;12(1):26-41. doi:[10.1016/j.media.2007.06.004](https://doi.org/10.1016/j.media.2007.06.004). PMC: [PMC2276735](https://pmc.ncbi.nlm.nih.gov/articles/PMC2276735/).

Foundational paper describing the SyN algorithm: symmetric energy formulation, cross-correlation metric, diffeomorphic constraints, Euler-Lagrange equations, multi-resolution scheme (100/100/100/20 iterations across 4 levels). Reports computational cost: ~20 minutes on 2.0 GHz processor per registration. **Source type: peer-reviewed journal article.**

### ANTs Similarity Metric Evaluation

**Avants BB, Tustison NJ, Song G, Cook PA, Klein A, Gee JC.** A reproducible evaluation of ANTs similarity metric performance in brain image registration. *NeuroImage*. 2011;54(3):2033-2044. doi:[10.1016/j.neuroimage.2010.09.025](https://doi.org/10.1016/j.neuroimage.2010.09.025). PMC: [PMC3065962](https://pmc.ncbi.nlm.nih.gov/articles/PMC3065962/).

Validates SyN as a top-performing registration algorithm. Describes gradient step range (0.1-0.5 useful), Gaussian regularization (variance = 3x image spacing), and template convergence (< 10 iterations). Brain extraction Jaccard overlap 0.958. **Source type: peer-reviewed journal article.**

### Unbiased TBM for Clinical Trials

**Hua X, Gutman B, Boyle C, et al.** Accurate measurement of brain changes in longitudinal MRI scans using tensor-based morphometry. *NeuroImage*. 2011;57(1):5-14. doi:[10.1016/j.neuroimage.2011.01.079](https://doi.org/10.1016/j.neuroimage.2011.01.079). PMC: [PMC3394184](https://pmc.ncbi.nlm.nih.gov/articles/PMC3394184/).

Demonstrates that inverse-consistent TBM reduces atrophy rate estimation bias from 1.4% to 0.28%. Reports sample sizes needed for clinical trials: 62 AD / 129 MCI subjects for 80% power to detect 25% slowing of atrophy over 24 months. **Source type: peer-reviewed journal article.**

### AD-Signature Cortical Thickness Biomarker

**Dickerson BC, Bakkour A, Salat DH, et al.** The cortical signature of Alzheimer's disease: regionally specific cortical thinning relates to symptom severity in very mild to mild AD dementia and is detectable in asymptomatic amyloid-positive individuals. *Cerebral Cortex*. 2009;19(3):497-510.

Defines the original "AD signature" concept. See also: **Bakkour A, et al.** The cortical signature of prodromal AD: Regional thinning predicts mild AD dementia. *Neurology*. 2009;72(12):1048-1055.

**Source type: peer-reviewed journal articles.**

---

## 4. Technical Requirements

### Software Dependencies

**Core requirement: ANTs (Advanced Normalization Tools)**

The following ANTs binaries/scripts are needed:

| Binary/Script | Purpose |
|---|---|
| `antsRegistration` | Core registration engine (rigid + affine + SyN stages) |
| `antsRegistrationSyN.sh` | Wrapper script with standard SyN parameters |
| `antsRegistrationSyNQuick.sh` | Faster variant with reduced iterations |
| `CreateJacobianDeterminantImage` | Computes Jacobian determinant (and log-Jacobian) from warp field |
| `N4BiasFieldCorrection` | N4 bias field correction (successor to N3) |
| `antsBrainExtraction.sh` | Atlas-based brain extraction |
| `antsApplyTransforms` | Apply transforms to images or label maps |
| `ImageMath` | Image arithmetic operations |

**Command-line usage for CreateJacobianDeterminantImage:**
```bash
CreateJacobianDeterminantImage 3 warpField.nii.gz output_logjacobian.nii.gz 1 0 0
# Args: dimension, deformationField, output, doLogJacobian, useGeometric, deformationGradient
```

**ANTs SyN registration parameters (standard):**
```bash
antsRegistration \
  --dimensionality 3 \
  --transform Rigid[0.1] \
  --metric MI[$fixed,$moving,1,32,Regular,0.25] \
  --convergence [1000x500x250x100,1e-6,10] \
  --shrink-factors 8x4x2x1 \
  --smoothing-sigmas 3x2x1x0vox \
  --transform Affine[0.1] \
  --metric MI[$fixed,$moving,1,32,Regular,0.25] \
  --convergence [1000x500x250x100,1e-6,10] \
  --shrink-factors 8x4x2x1 \
  --smoothing-sigmas 3x2x1x0vox \
  --transform SyN[0.1,3,0] \
  --metric CC[$fixed,$moving,1,4] \
  --convergence [100x70x50x20,1e-6,10] \
  --shrink-factors 8x4x2x1 \
  --smoothing-sigmas 3x2x1x0vox \
  --output [$outputPrefix,${outputPrefix}Warped.nii.gz,${outputPrefix}InverseWarped.nii.gz]
```

**ANTsPy (Python) equivalent:**
```python
import ants

fixed = ants.image_read('timepoint1.nii.gz')
moving = ants.image_read('timepoint2.nii.gz')

# SyN registration with cross-correlation
reg = ants.registration(
    fixed=fixed,
    moving=moving,
    type_of_transform='SyNCC',  # SyN with cross-correlation
    grad_step=0.1,
    syn_metric='CC',
    reg_iterations=(100, 70, 50, 20),
    write_composite_transform=True
)

# Compute log-Jacobian
log_jacobian = ants.create_jacobian_determinant_image(
    domain_image=fixed,
    tx=reg['fwdtransforms'][0],  # the warp field
    do_log=True,
    geom=False
)
```

**Additional software (for the full ADNI-style pipeline):**

| Software | Purpose | Notes |
|---|---|---|
| dcm2niix | DICOM to NIfTI conversion | Already in AEGIS bids-service |
| SPM12 (or CAT12) | Tissue segmentation (GM/WM/CSF) | MATLAB required; or use ANTs-based Atropos |
| FSL | Optional: brain extraction (BET), template registration (FLIRT/FNIRT) | Alternative to ANTs-only pipeline |
| FreeSurfer | Optional: cortical parcellation for ROI definition | Alternative ROI source |

### Atlas / Template Requirements

**For an implementation pipeline (alternatives to STAND400):**

| Template | Description | Availability |
|---|---|---|
| MNI152 (ICBM 2009c) | Standard 1mm brain template | Public, included with FSL/ANTs |
| OASIS-30 ANTs template | ANTs-optimized brain template | Public, from ANTs distribution |
| STAND400 | Mayo Clinic custom 400-subject ADNI template | Not publicly available |

**ROI Atlases:**

| Atlas | Regions | Use |
|---|---|---|
| AAL (Automated Anatomical Labeling) | 116 cortical + subcortical regions | Used by Vemuri et al. 2016 (14 regions for TBM-SyN) |
| AAL3 | 170 regions (updated) | Current version with cerebellum parcels |
| Desikan-Killiany (FreeSurfer) | 68 cortical + subcortical | Standard FreeSurfer parcellation |
| Harvard-Oxford | 48 cortical + 21 subcortical | Probabilistic atlas from FSL |
| Mayo in-house (119 GM + ventricle) | 120 regions | Not publicly available |

**For an open-source implementation, the AAL3 atlas on MNI152 template is the closest publicly available equivalent to the Mayo approach.**

### Input Requirements

- **Paired longitudinal T1-weighted scans** from the same subject at two (or more) time points
- 3D volumetric acquisitions (MPRAGE, IR-FSPGR, or equivalent)
- Resolution: 1mm isotropic preferred; 1.2mm acceptable
- Minimum inter-scan interval: typically >= 6 months (shorter intervals may not show measurable change above noise)
- Scans should be from the same scanner/protocol when possible (Vemuri et al. 2015 showed systematic differences between accelerated and unaccelerated protocols)

### Computational Requirements

| Resource | Estimate | Notes |
|---|---|---|
| **CPU time per pair** | 20-45 minutes | For full rigid + affine + SyN with CC metric at 1mm |
| **Memory (RAM)** | 4-8 GB per registration | Can occupy ~85% of available memory; float precision |
| **Disk per study** | ~500 MB | Input NIfTIs + warp fields + Jacobian maps |
| **Parallelism** | Multi-threaded (ITK-based) | Set `ITK_GLOBAL_DEFAULT_NUMBER_OF_THREADS` |

The SyN stage with cross-correlation is computationally dominant — full-resolution iterations can take longer than all preceding stages combined.

---

## 5. Output Format

### Files Produced by TBM-SyN Pipeline

| File | Description |
|---|---|
| `*_Warped.nii.gz` | Moving image warped to fixed image space |
| `*_InverseWarped.nii.gz` | Fixed image warped to moving image space |
| `*_0GenericAffine.mat` | Affine transformation matrix |
| `*_1Warp.nii.gz` | Forward deformation field (vector field) |
| `*_1InverseWarp.nii.gz` | Inverse deformation field |
| `*_logjacobian.nii.gz` | Log-Jacobian determinant map (voxel-wise atrophy) |
| `*_logjacobian_annualized.nii.gz` | Log-Jacobian divided by inter-scan interval in years |
| `*_softmean.nii.gz` | Symmetric midpoint image (for tissue segmentation) |
| `*_gm_mask.nii.gz` | Grey matter probability map from segmentation |

### ROI-Level Summary Metrics

For each ROI, the pipeline typically produces:

| Metric | Description | Units |
|---|---|---|
| `mean_log_jacobian` | Mean log-Jacobian within the ROI, weighted by GM probability | dimensionless |
| `annualized_atrophy_rate` | Mean log-Jacobian / inter-scan interval | %/year (after ×100) |
| `volume_change_pct` | Percentage volume change | % |
| `gm_volume_baseline` | Grey matter volume at baseline in the ROI | mm^3 |
| `gm_volume_followup` | Grey matter volume at follow-up in the ROI | mm^3 |

### TBM-SyN Summary Score

The ADNI TBM-SyN summary score is a single scalar value: the grey-matter-weighted mean of annualized log-Jacobian values across the 31 AD-signature ROIs. A negative score indicates net atrophy (tissue loss); a more negative score indicates faster progression.

### CSV Output Format

The ADNI archive distributes TBM-SyN results as CSV files with columns including:
- Subject ID (RID)
- Scan date (baseline and follow-up)
- Inter-scan interval
- Per-ROI annualized log-Jacobian values
- TBM-SyN composite summary score
- Scan metadata (protocol, scanner, acceleration)

---

## 6. Comparison with FreeSurfer Longitudinal

### Methodological Differences

| Aspect | TBM-SyN | FreeSurfer Longitudinal |
|---|---|---|
| **Approach** | Deformation-based (warp fields between time points) | Surface-based (cortical thickness at each time point) |
| **Registration** | Symmetric diffeomorphic (ANTs SyN) | Subject-specific template (SST) created from all time points |
| **Measurement** | Jacobian determinant of deformation field | Cortical thickness (mm) and subcortical volumes (mm^3) |
| **Output** | Voxel-wise atrophy maps + ROI summaries | Per-vertex thickness maps + parcellation volumes |
| **Bias** | Symmetric by construction | Uses unbiased within-subject template |
| **Processing time** | ~30-45 min per scan pair | ~8-20 hours per time point (full recon-all) |
| **Software** | ANTs (open source, C++/Python) | FreeSurfer (free for research, C/Tcl) |
| **Clinical trial power** | Higher precision for detecting atrophy rates | Better for cross-sectional group differences |
| **User adoption** | Low (discontinued in ADNI4) | Dominant (overwhelmingly preferred by ADNI users) |

### When to Use Each

**TBM-SyN is preferred when:**
- Precise longitudinal change measurement is the primary goal
- Clinical trial sample size optimization matters (lower N needed)
- Voxel-wise atrophy maps are desired
- Fast turnaround is needed (~30 min vs ~20 hrs)
- Symmetry/bias-free measurement is critical

**FreeSurfer longitudinal is preferred when:**
- Cross-sectional cortical thickness measurements are also needed
- Surface-based analysis (e.g., vertex-wise statistics) is required
- Integration with large existing FreeSurfer-based studies
- Subcortical structure volumes are needed (hippocampus, amygdala, etc.)
- The community standard is required for reproducibility

### Complementary Use

The two methods are complementary. TBM-SyN excels at measuring rates of change with high precision and low sample-size requirements for clinical trials. FreeSurfer provides richer anatomical detail (cortical thickness, surface area, folding patterns, subcortical volumes) but at higher computational cost. Both can be run on the same data and cross-validated.

Schwarz et al. (2016) found that FreeSurfer thickness achieved the highest diagnostic separability (AUROC 0.86) among single methods, while SPM+DiReCT (ANTs-based) volume was most reliable on repeated scans. Volume- and thickness-based measures generally perform similarly for separating clinically normal from AD populations.

---

## 7. Implementation Considerations for an AEGIS Pipeline Service

### Simplified Open-Source Pipeline

The original Mayo Clinic pipeline uses proprietary tools (STAND400 template, custom atlas, MATLAB/SPM5). A modern open-source equivalent would use:

1. **dcm2niix** -- DICOM to NIfTI conversion (already in AEGIS)
2. **N4BiasFieldCorrection** (ANTs) -- replaces N3 correction
3. **antsBrainExtraction.sh** (ANTs) -- brain extraction with OASIS template
4. **antsRegistrationSyN.sh** (ANTs) -- full SyN registration pipeline
5. **CreateJacobianDeterminantImage** (ANTs) -- log-Jacobian computation
6. **Atropos** (ANTs) -- tissue segmentation (replaces SPM5 unified segmentation)
7. **antsApplyTransforms** + AAL3 atlas -- ROI mapping in subject space

### Docker Container Size

ANTs compiled binaries: ~200 MB. With OASIS template + AAL3 atlas: ~400 MB total. Significantly smaller than FreeSurfer (~14 GB).

### Pipeline as AEGIS Sidecar

This would fit the existing AEGIS sidecar pattern:
- Python FastAPI service (like defacing, qc-service, etc.)
- Calls ANTs binaries via subprocess or ANTsPy
- Receives pairs of NIfTI files (from bids-service output)
- Returns: log-Jacobian maps, ROI CSV summaries, composite score
- Study fields: `tbm_required`, `tbm_status` (pending/processing/complete/failed)

---

## Citations Index

1. Vemuri P, Senjem ML, Gunter JL, et al. (2015). Accelerated vs. unaccelerated serial MRI based TBM-SyN measurements for clinical trials in Alzheimer's disease. *NeuroImage*, 113:61-69. doi:10.1016/j.neuroimage.2015.03.026. PMID:25797830.

2. Avants BB, Epstein CL, Grossman M, Gee JC. (2008). Symmetric diffeomorphic image registration with cross-correlation. *Medical Image Analysis*, 12(1):26-41. doi:10.1016/j.media.2007.06.004.

3. Avants BB, Tustison NJ, Song G, Cook PA, Klein A, Gee JC. (2011). A reproducible evaluation of ANTs similarity metric performance in brain image registration. *NeuroImage*, 54(3):2033-2044. doi:10.1016/j.neuroimage.2010.09.025.

4. Schwarz CG, Gunter JL, Wiste HJ, et al. (2016). A large-scale comparison of cortical thickness and volume methods for measuring Alzheimer's disease severity. *NeuroImage: Clinical*, 11:802-812. doi:10.1016/j.nicl.2016.05.017.

5. Jack CR Jr, Arani A, Borowski BJ, et al. (2024). Overview of ADNI MRI. *Alzheimer's & Dementia*, 20(10):7350-7360. doi:10.1002/alz.14166.

6. Jack CR Jr, Wiste HJ, Knopman DS, et al. (2014). Rates of beta-amyloid accumulation are independent of hippocampal neurodegeneration. *Neurology*, 82(17):1605-1612. doi:10.1212/WNL.0000000000000386.

7. Vemuri P, Wiste HJ, Weigand SD, et al. (2016). Evolution of neurodegeneration-imaging biomarkers from clinically normal to dementia in the Alzheimer disease spectrum. *Radiology*, 281(2):527-535.

8. Hua X, Gutman B, Boyle C, et al. (2011). Accurate measurement of brain changes in longitudinal MRI scans using tensor-based morphometry. *NeuroImage*, 57(1):5-14. doi:10.1016/j.neuroimage.2011.01.079.

9. Tzourio-Mazoyer N, et al. (2002). Automated Anatomical Labeling of activations in SPM using a macroscopic anatomical parcellation of the MNI MRI single-subject brain. *NeuroImage*, 15(1):273-289.

10. Dickerson BC, Bakkour A, Salat DH, et al. (2009). The cortical signature of Alzheimer's disease. *Cerebral Cortex*, 19(3):497-510.
