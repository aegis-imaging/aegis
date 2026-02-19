# MRI Defacing Tools Comparison

Research reference for AEGIS defacing service backend selection. Compares available tools by accuracy, speed, Docker footprint, and licensing.

## Tool Comparison

| Tool | Method | Success Rate | Speed | Docker Size | License | AEGIS Status |
|------|--------|-------------|-------|-------------|---------|--------------|
| **afni_refacer** | Template registration | 89% | 13-30 min | ~3-5 GB | Public domain (NIMH) | Not planned (too slow, too large) |
| **mri_reface** | Registration + re-facing | N/A (best identity protection) | >1 min | ~2-4 GB | Non-commercial only | Implemented (PR #3) |
| **PyDeface** | FSL FLIRT registration | 83% | 2-10 min | ~3-6 GB (FSL required) | MIT | Not planned (FSL too heavy) |
| **DeepDefacer** | 3D U-Net segmentation | ~75% | ~1-2 min | ~500 MB | Research-friendly | Implemented (PR #54) |
| **mri_deface** | Atlas registration | ~70% | 2-10 min | ~500 MB | Free (FreeSurfer) | Implemented (PR #3) |
| **MiDeFace** | FreeSurfer deep | Very good | ~30 min | ~4-8 GB | FreeSurfer license | Not planned (too slow) |
| **Quickshear** | Brain mask trimming | Low | <1 min | ~100 MB | MIT | Not recommended |
| **nibabel-fallback** | Zeros anterior 30% | N/A | <1 min | 0 | MIT | Implemented (dev only, PR #3) |

## Key Findings

### Multisite Defacing Comparison (Theyers et al. 2021)

The largest systematic comparison of defacing tools across 300 scans from multiple sites, ages 3-85.

- **afni_refacer** achieved the highest success rate at 89%, with the best downstream brain-age prediction accuracy (MAE 7.96)
- **PyDeface** achieved 83% success but struggled with elderly cohorts (ages 44-85)
- **mri_deface** had lower success rates than both, with facial detection confidence of 0.42 vs afni_refacer's 0.95
- **Quickshear** had the lowest success rate and left identifiable facial structures (cheeks, jawline)
- Human facial recognition of defaced scans ranged from 4.3-13.6% vs 64.9% for originals

### DeepDefacer (Khazane et al. 2022)

First application of deep learning to MRI defacing. Uses a modified 3D U-Net with 92% fewer parameters than the original U-Net architecture.

- **Training data**: IXI (Image-X) and ICBM (International Consortium for Brain Mapping) datasets
- **Speed**: ~90% faster than PyDeface (couple of minutes per scan)
- **Architecture**: TensorFlow/Keras, works on both CPU and GPU
- **Installation**: `pip install deepdefacer` (self-contained, no external binaries needed)
- **Format**: NIfTI input/output (DICOM round-trip via dcm2niix + pydicom)

### Brain Segmentation Impact (2023 Reproducibility Study)

- Quickshear caused **catastrophic brain segmentation failures** — worst performer
- afni_refacer and PyDeface had minimal impact on downstream analysis
- DeepDefacer showed acceptable impact on brain measurements

### AEGIS Selection Rationale

**Auto-selection priority**: mri_reface > deepdefacer > mri_deface > nibabel

1. **mri_reface** stays first (best modality coverage: MRI + PET + CT) but is rarely available (needs MATLAB Runtime, non-commercial license)
2. **DeepDefacer** is the effective default: pip-installable, fastest, good quality, no external binary dependencies beyond dcm2niix
3. **mri_deface** remains as a proven fallback for environments where TensorFlow is undesirable
4. **nibabel** is the dev-only fallback (crude 30% anterior zeroing)

**Why not afni_refacer?** Best quality (89%) but requires the full AFNI suite (~3-5 GB), takes 13-30 minutes per volume, and is impractical for production throughput.

**Why not PyDeface?** Requires FSL FLIRT (~2-6 GB), uses the same registration approach as mri_deface, and doesn't differentiate enough to justify the image size.

## Citations

1. **Theyers AE**, Zamyadi M, O'Reilly M, et al. "Multisite Comparison of MRI Defacing Software Across Multiple Cohorts." *Frontiers in Psychiatry*. 2021;12:617997. DOI: [10.3389/fpsyt.2021.617997](https://doi.org/10.3389/fpsyt.2021.617997). PMID: 33716819. PMC: PMC7943842. **Source type**: Peer-reviewed journal article.

2. **Khazane A**, Hoachuck J, Gorgolewski KJ, Poldrack RA. "DeepDefacer: Automatic Removal of Facial Features via U-Net Image Segmentation." arXiv:2205.15536. 2022. DOI: [10.48550/arXiv.2205.15536](https://doi.org/10.48550/arXiv.2205.15536). **Source type**: Preprint (arXiv), not peer-reviewed.

3. **Schwarz CG**, Kremers WK, Therneau TM, et al. "Identification of Anonymous MRI Research Participants with Face-Recognition Software." *New England Journal of Medicine*. 2019;381(17):1684-1686. DOI: [10.1056/NEJMc1908881](https://doi.org/10.1056/NEJMc1908881). **Source type**: Peer-reviewed correspondence, NEJM.

4. **Rubbert C**, Wolf S, Gaser C, Mikolajczyk R. "Impact of Defacing on Automated Brain Atrophy Estimation." *Insights into Imaging*. 2023;14:54. DOI: [10.1186/s13244-023-01396-0](https://doi.org/10.1186/s13244-023-01396-0). PMC: PMC10246049. **Source type**: Peer-reviewed journal article.

5. **Bhalerao GV**, Parekh P, Sathe S, et al. "Quality Assessment of Brain MRI Defacing." *NeuroImage*. 2024;299:120828. DOI: [10.1016/j.neuroimage.2024.120828](https://doi.org/10.1016/j.neuroimage.2024.120828). PMID: 39176821. **Source type**: Peer-reviewed journal article.

## Tool Links

- [DeepDefacer on PyPI](https://pypi.org/project/deepdefacer/)
- [DeepDefacer GitHub](https://github.com/AKhazane/DeepDeface)
- [afni_refacer documentation](https://afni.nimh.nih.gov/pub/dist/doc/htmldoc/tutorials/refacer/refacer_run.html)
- [mri_deface (FreeSurfer)](https://surfer.nmr.mgh.harvard.edu/fswiki/mri_deface)
- [mri_reface (NITRC)](https://www.nitrc.org/projects/mri_reface)
- [PyDeface (GitHub)](https://github.com/poldracklab/pydeface)
