# SynthSeg Integration Research

## Overview

SynthSeg is a contrast-agnostic brain MRI segmentation tool that segments scans regardless of acquisition parameters (T1w, T2w, FLAIR, any resolution). Developed at Harvard/MIT, it ships with FreeSurfer 7.3.2+ and is also available as a standalone Python package.

## Key Features

- **Contrast-agnostic**: Works on T1w, T2w, FLAIR, PD, and any resolution without retraining
- **97 ROIs** with `--parc` flag: 68 cortical (Desikan-Killiany atlas) + 29 subcortical
- **32 ROIs** without `--parc`: subcortical structures only
- **Built-in QC scoring**: `--qc` outputs a quality score (0–1) for each scan
- **Robust mode**: `--robust` for clinical-grade scans with pathology
- **Speed**: ~6 seconds on GPU, ~2 minutes on CPU

## Installation Paths

### Path 1: FreeSurfer (preferred)
- Bundled with FreeSurfer 7.3.2+ as `mri_synthseg` binary
- Requires FreeSurfer license (`FS_LICENSE` env var)
- Most stable, actively maintained by FreeSurfer team

### Path 2: Standalone Python
- `pip install SynthSeg` (Apache 2.0)
- Requires TensorFlow
- Module: `python -m SynthSeg.predict --i <input> --o <output>`
- ~200 MB with TensorFlow dependency

## CLI Usage

```bash
mri_synthseg \
  --i input.nii.gz \
  --o seg.nii.gz \
  --vol volumes.csv \
  --qc qc.csv \
  --parc \
  --robust
```

## Output Format

### Volumes CSV
```csv
subject,Left-Hippocampus,Right-Hippocampus,Left-Amygdala,...
/path/to/input.nii.gz,3456.2,3400.1,1200.5,...
```

### QC CSV
```csv
subject,qc_score
/path/to/input.nii.gz,0.9523
```

## AEGIS Integration

- Backend: `analytics-service/app/backends/synthseg.py`
- Name: `"synthseg"`
- Availability: `mri_synthseg` binary OR `SynthSeg` Python package
- Input: Any NIfTI (contrast-agnostic — uses `find_any_nifti()`)
- Output: Segmentation NIfTI, volumetric CSV, QC CSV, summary JSON
- Metrics structure: `{atlas: "synthseg", roi_count, parcellation, qc_score, roi_volumes}`
- Go storage: `storeSynthSegROIs()` — volumes + QC composite score

## References

- Billot et al. "SynthSeg: Segmentation of brain MRI scans of any contrast and resolution without retraining." Medical Image Analysis, 2023. DOI: 10.1016/j.media.2023.102789
- FreeSurfer SynthSeg documentation: https://surfer.nmr.mgh.harvard.edu/fswiki/SynthSeg
