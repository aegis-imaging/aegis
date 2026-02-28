# Analytics New Backends — Phase 2 Research

**Date:** 2026-02-28
**Purpose:** Reference citations and integration notes for 7 new neuroimaging analytics backends being added to the AEGIS analytics service.

---

## 1. ITK-SNAP / Convert3D

**Website:** http://www.itksnap.org/

**What it does:** Semi-automated 3D segmentation using active contour (snake) models. ITK-SNAP provides geodesic active contour and region competition segmentation driven by user-placed seeds. Convert3D (`c3d`) is its companion CLI tool for image format conversion, resampling, arithmetic, and label manipulation — usable entirely headless.

**Role in AEGIS pipeline:** Server-side label-map post-processing and format conversion via `c3d`. Useful for resampling segmentation masks to a common space, merging/splitting label maps from other backends (SynthSeg, TotalSegmentator), and computing per-ROI volumes from integer label images. Operates on NIfTI inputs produced by the BIDS service.

**Metrics produced:** Per-ROI volumes (mm^3), label statistics (mean/std/min/max intensity per ROI), voxel counts.

**Scan types:** Any 3D medical image (MRI, CT, PET). Modality-agnostic — works on label maps regardless of source.

**License:** GPL-3.0 (ITK-SNAP); Convert3D is distributed under the same license.

**Citation:**
- Yushkevich PA, Piven J, Hazlett HC, Smith RG, Ho S, Gee JC, Gerig G. "User-guided 3D active contour segmentation of anatomical structures: Significantly improved efficiency and reliability." *NeuroImage*. 2006;31(3):1116-28. DOI: [10.1016/j.neuroimage.2006.01.015](https://doi.org/10.1016/j.neuroimage.2006.01.015). **Source type:** Peer-reviewed.

---

## 2. BrainSuite

**Website:** http://brainsuite.org/

**What it does:** Automated cortical surface extraction pipeline with gray/white matter boundary identification, cortical thickness estimation, and surface-based atlas registration (BrainSuite Anatomical Pipeline, BDP). Includes diffusion pipeline for DTI fitting and tractography.

**Role in AEGIS pipeline:** Alternative to FreeSurfer for cortical surface analysis when faster turnaround is needed (BrainSuite completes in ~10 minutes vs FreeSurfer's 6-12 hours). Produces cortical thickness maps, surface area measurements, and gray/white matter volume estimates. Invoked via CLI (`bse`, `bfc`, `cortical_extraction_sequence`).

**Metrics produced:** Cortical thickness (mm), surface area (mm^2), gray matter volume, white matter volume, CSF volume, per-ROI volumes using USCBrain atlas (130+ regions).

**Scan types:** T1-weighted MRI (cortical pipeline), diffusion MRI (BDP pipeline).

**License:** GPL (open-source, free for non-commercial and commercial use).

**Citation:**
- Shattuck DW, Leahy RM. "BrainSuite: An automated cortical surface identification tool." *Medical Image Analysis*. 2002;6(2):129-42. DOI: [10.1016/S1361-8415(02)00054-3](https://doi.org/10.1016/S1361-8415(02)00054-3). **Source type:** Peer-reviewed.

---

## 3. volBrain

**Website:** https://volbrain.upv.es/

**What it does:** Automated online brain volumetry system providing whole-brain segmentation, subcortical structure labeling, and brain tissue classification. Uses a multi-atlas patch-based segmentation approach with non-local means label fusion — no training data or GPU required.

**Role in AEGIS pipeline:** Cloud-based brain volumetry backend for automated hippocampal, amygdala, thalamus, caudate, putamen, pallidum, and accumbens volume quantification. Particularly useful for Alzheimer's disease and epilepsy cohorts where hippocampal volume is a primary biomarker. AEGIS would submit NIfTI via the volBrain API and retrieve volumetric reports.

**Metrics produced:** Intracranial volume (ICV), brain parenchymal fraction, tissue volumes (GM, WM, CSF), subcortical structure volumes (bilateral, ~30 structures), hemispheric asymmetry indices.

**Scan types:** T1-weighted MRI (1.5T and 3T).

**License:** Free for research use (online service; no local installation required).

**Citation:**
- Manjon JV, Coupe P. "volBrain: An Online MRI Brain Volumetry System." *Frontiers in Neuroinformatics*. 2016;10:30. DOI: [10.3389/fninf.2016.00030](https://doi.org/10.3389/fninf.2016.00030). **Source type:** Peer-reviewed.

---

## 4. MONAI Label

**Website:** https://monai.io/label.html

**What it does:** AI-assisted interactive and automatic segmentation framework built on PyTorch. Provides a REST API server that hosts trained segmentation models and supports active learning — models improve as users annotate more data. Ships with pre-trained models for whole-brain segmentation, tumor segmentation, and organ segmentation.

**Role in AEGIS pipeline:** Automatic brain segmentation backend using pre-trained DeepEdit or SegResNet models. MONAI Label runs as a standalone server (no 3D Slicer or Xvfb dependency), exposing `/infer` and `/scoring` REST endpoints that the analytics service calls directly. Supports custom model deployment for institution-specific segmentation tasks.

**Metrics produced:** Segmentation label maps (NIfTI), per-ROI volumes, Dice/IoU scores against reference segmentations (when available), model confidence scores.

**Scan types:** MRI (T1w, T2w, FLAIR), CT. Model-dependent — pre-trained models cover brain MRI and abdominal CT; custom models can target any modality.

**License:** Apache-2.0.

**Citation:**
- Diaz-Pinto A, Alle S, Nath V, Tang Y, Ihsani A, Asber M, Bermudez Noguera F, Guo D, Obinata H, Niu J, Yang J, Zheng Y, Xu D, Hatamizadeh A, Flores M, Zephyr A, He Y, Calvett A, Moric I, Almeida AG, Ravindran S, Tsehay Y, Ferreira da Costa T, Berihu S, Negassi M, Ravishankar H, Ren J, Naga Srinivas C, Roth HR, Dogra P, Myronenko A, Feng A, Klapholz B, Cardoso MJ. "MONAI Label: A framework for AI-assisted Interactive Labeling of 3D Medical Images." *Medical Image Analysis*. 2024;95:103207. DOI: [10.1016/j.media.2024.103207](https://doi.org/10.1016/j.media.2024.103207). **Source type:** Peer-reviewed.

---

## 5. PETSurfer

**Website:** https://surfer.nmr.mgh.harvard.edu/ (part of FreeSurfer)

**What it does:** PET analysis tools integrated into the FreeSurfer framework for PET-MRI multimodal analysis. Performs partial volume correction (PVC) using the geometric transfer matrix (GTM) method, surface-based kinetic modeling, and ROI-based quantification using FreeSurfer's cortical and subcortical parcellations as the anatomical reference.

**Role in AEGIS pipeline:** PET quantification backend for studies with co-registered PET and MRI data. Uses FreeSurfer's `recon-all` output (cortical parcellation + subcortical segmentation) as the anatomical framework, then applies `mri_gtmpvc` for partial volume correction and `mri_glmfit` for kinetic modeling. Produces regional SUVr (standardized uptake value ratio), binding potentials, and distribution volumes.

**Metrics produced:** Regional SUVr (reference: cerebellar gray matter), partial-volume-corrected PET values per FreeSurfer ROI, binding potential (BP_ND), distribution volume ratio (DVR), kinetic rate constants (K1, k2, k3, k4 for compartmental models).

**Scan types:** PET (amyloid, tau, FDG, dopamine tracers) co-registered with T1-weighted MRI. Requires FreeSurfer `recon-all` output as prerequisite.

**License:** FreeSurfer license (free for research; see https://surfer.nmr.mgh.harvard.edu/fswiki/License).

**Citation:**
- Greve DN, Svarer C, Fisher PM, Feng L, Hansen AE, Baare W, Rosen B, Fischl B, Knudsen GM. "Cortical surface-based analysis reduces bias and variance in kinetic modeling of brain PET data." *NeuroImage*. 2014;92:460-70. DOI: [10.1016/j.neuroimage.2013.12.021](https://doi.org/10.1016/j.neuroimage.2013.12.021). **Source type:** Peer-reviewed.

---

## 6. QSM Tools (TGV-QSM, TKD)

**What they do:** Quantitative Susceptibility Mapping (QSM) reconstructs tissue magnetic susceptibility from MRI phase data. TGV-QSM uses total generalized variation regularization for high-quality dipole inversion. TKD (Thresholded K-space Division) is a fast, direct inversion method. Both convert GRE (gradient-recalled echo) phase images into quantitative susceptibility maps.

**Role in AEGIS pipeline:** QSM reconstruction backend for iron/calcium quantification in brain tissue. Clinically relevant for neurodegenerative disease assessment (iron accumulation in substantia nigra for Parkinson's, basal ganglia for Huntington's) and for detecting microbleeds and calcifications. Operates on phase images from multi-echo GRE sequences after BIDS conversion.

**Metrics produced:** Susceptibility maps (ppm), regional mean susceptibility values per atlas ROI, iron concentration estimates (ug/g), R2* maps (from magnitude data).

**Scan types:** Multi-echo gradient-recalled echo (GRE) MRI with phase data. 3T or 7T field strength preferred.

**License:** TGV-QSM — academic/research license (contact authors); TKD — open implementation available in MEDI toolbox (Cornell, academic license) and QSMxT (open-source, MIT-like).

**Citations:**
- Langkammer C, Bredies K, Poser BA, Barth M, Reishofer G, Fan AP, Bilgic B, Fazekas F, Mainero C, Ropele S. "Fast quantitative susceptibility mapping using 3D EPI and total generalized variation." *NeuroImage*. 2015;111:622-30. DOI: [10.1016/j.neuroimage.2015.02.041](https://doi.org/10.1016/j.neuroimage.2015.02.041). **Source type:** Peer-reviewed.
- Shmueli K, de Zwart JA, van Gelderen P, Li TQ, Dodd SJ, Duyn JH. "Magnetic susceptibility mapping of brain tissue in vivo using MRI phase data." *Magnetic Resonance in Medicine*. 2009;62(6):1510-22. DOI: [10.1002/mrm.22135](https://doi.org/10.1002/mrm.22135). **Source type:** Peer-reviewed.

---

## 7. BASIL / oxford_asl

**Website:** https://fsl.fmrib.ox.ac.uk/fsl/fslwiki/BASIL

**What it does:** Arterial Spin Labeling (ASL) perfusion MRI analysis tool, part of the FSL suite. Uses variational Bayesian inference to fit kinetic models to ASL data, producing calibrated cerebral blood flow (CBF) maps. Supports pulsed (PASL), pseudo-continuous (pCASL), and multi-PLD ASL acquisitions. The `oxford_asl` wrapper provides a complete pipeline from raw ASL data to calibrated perfusion maps.

**Role in AEGIS pipeline:** ASL perfusion quantification backend for cerebral blood flow mapping. Processes BIDS-converted ASL NIfTI data through `oxford_asl`, which handles label-control subtraction, motion correction, kinetic model fitting (via BASIL), spatial regularization, partial volume correction, and calibration (using M0 scan or CSF reference). Produces voxel-wise and ROI-based CBF measurements.

**Metrics produced:** Calibrated CBF maps (ml/100g/min), arterial transit time (ATT) maps (seconds), per-ROI mean CBF (using FreeSurfer or atlas parcellations), partial-volume-corrected CBF (gray matter and white matter separately), bolus arrival time maps.

**Scan types:** Arterial Spin Labeling (ASL) MRI — pCASL, PASL, multi-PLD/multi-TI acquisitions. Requires M0 calibration scan or proton-density-weighted reference for absolute quantification.

**License:** FSL license (free for non-commercial use; see https://fsl.fmrib.ox.ac.uk/fsl/fslwiki/Licence).

**Citation:**
- Chappell MA, Groves AR, Whitcher B, Woolrich MW. "Variational Bayesian Inference for a Nonlinear Forward Model." *IEEE Transactions on Signal Processing*. 2009;57(1):223-36. DOI: [10.1109/TSP.2008.2005752](https://doi.org/10.1109/TSP.2008.2005752). **Source type:** Peer-reviewed.

---

## AEGIS Integration Summary

All 7 backends integrate into the existing `analytics-service/app/backends/` structure. Each backend follows the same pattern: availability check, NIfTI input from BIDS output, CLI or Python API invocation, structured metrics returned to the Go API via callback.

| Backend | Binary / Package | Input | Primary Use Case | Docker Size Impact |
|---------|-----------------|-------|-----------------|-------------------|
| Convert3D (`c3d`) | `c3d` CLI | NIfTI label maps | ROI post-processing, format conversion | ~100 MB |
| BrainSuite | `bse`, `bfc`, `cortical_extraction_sequence` | T1w NIfTI | Cortical surface analysis (fast alternative to FreeSurfer) | ~500 MB |
| volBrain | HTTP API (remote) | T1w NIfTI | Automated brain volumetry | None (remote service) |
| MONAI Label | `monailabel` (Python) | NIfTI | AI-based segmentation with active learning | ~2-4 GB (PyTorch + models) |
| PETSurfer | `mri_gtmpvc`, `mri_glmfit` (FreeSurfer) | PET + T1w NIfTI | PET-MRI quantification, SUVr, kinetic modeling | Included with FreeSurfer |
| QSM (TGV-QSM/TKD) | Python / MATLAB | GRE phase NIfTI | Iron/calcium quantification, susceptibility mapping | ~200 MB |
| BASIL / oxford_asl | `oxford_asl` (FSL) | ASL NIfTI + M0 | Cerebral blood flow mapping | Included with FSL |

**Auto-selection note:** These backends supplement the existing auto-selection chain (freesurfer > fsl > ants > spm > atlas_roi > synthseg > nnunet > totalsegmentator). Modality-specific backends (PETSurfer for PET, BASIL for ASL, QSM for GRE phase) are selected based on study modality and sequence type rather than general priority ordering.
