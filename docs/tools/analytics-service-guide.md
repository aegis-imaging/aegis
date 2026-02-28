# Analytics Service — User Guide

**AEGIS Phase 3 Pipeline: Neuroimaging Analytics**

The analytics service runs neuroimaging analysis tools on BIDS-converted studies. It is Phase 3 of the AEGIS processing pipeline and operates on NIfTI outputs produced by the BIDS conversion service (Phase 2).

---

## Overview

After a study completes BIDS conversion, the analytics service can run one or more neuroimaging tools to produce derived measurements — cortical thickness maps, tissue segmentation, brain extraction, DTI fitting, and more. Results are stored alongside the BIDS output and recorded in the audit trail.

**Pipeline position:**
```
Phase 0: Classification
Phase 1: PHI scan + Pixel Redaction + Protocol check + Defacing
Phase 2: QC check + BIDS conversion
Phase 3: Analytics  ← this service
```

---

## Supported Tools

### FreeSurfer (`recon-all`)

Full cortical reconstruction and volumetric segmentation pipeline.

| Item | Detail |
|------|--------|
| Binary | `recon-all` |
| License | Requires `FS_LICENSE` file (free registration at surfer.nmr.mgh.harvard.edu) |
| Runtime | 6–12 hours per subject (CPU) |
| Input | T1-weighted NIfTI (`.nii.gz`) |
| Output | `surf/`, `mri/`, `stats/` directories with cortical parcellation, volumetric segmentation, thickness maps |

**Key outputs:**
- `stats/aseg.stats` — subcortical volume statistics
- `stats/lh.aparc.stats` / `rh.aparc.stats` — cortical parcellation statistics
- `surf/lh.thickness` / `rh.thickness` — vertex-wise cortical thickness

**Env vars:**
- `FREESURFER_HOME` — path to FreeSurfer installation
- `FS_LICENSE` — path to license file (or mount at `/opt/freesurfer/license.txt`)

### FSL (BET, FAST, FLIRT, DTIFIT)

FMRIB Software Library — modular tools for brain extraction, tissue segmentation, registration, and diffusion tensor fitting.

| Tool | Binary | What it does | Runtime |
|------|--------|-------------|---------|
| BET | `bet` | Brain extraction (skull stripping) | ~1 min |
| FAST | `fast` | Tissue-type segmentation (GM, WM, CSF) | ~5 min |
| FLIRT | `flirt` | Linear registration to standard space | ~2 min |
| DTIFIT | `dtifit` | Diffusion tensor model fitting (FA, MD maps) | ~1 min |

**Input:** T1w NIfTI for BET/FAST/FLIRT; DWI NIfTI + bvec/bval for DTIFIT.

**Env vars:**
- `FSLDIR` — path to FSL installation
- `FSLOUTPUTTYPE` — output format (default: `NIFTI_GZ`)

### ANTs (`antsCorticalThickness`)

Advanced Normalization Tools — cortical thickness analysis with diffeomorphic registration.

| Item | Detail |
|------|--------|
| Binary | `antsCorticalThickness.sh` |
| Runtime | 2–4 hours per subject |
| Input | T1-weighted NIfTI |
| Output | Cortical thickness map, brain extraction mask, tissue priors, registration transforms |

**Env vars:**
- `ANTSPATH` — path to ANTs binaries

### SPM (Segmentation, DARTEL)

Statistical Parametric Mapping — voxel-based morphometry and spatial normalization.

| Item | Detail |
|------|--------|
| Runtime | MATLAB or GNU Octave |
| Input | T1-weighted NIfTI |
| Output | Tissue probability maps (c1–c6), DARTEL flow fields, spatially normalized images |

**Env vars:**
- `SPM_HOME` — path to SPM installation
- `MATLAB_CMD` or `OCTAVE_CMD` — path to MATLAB or Octave binary

---

## Running Locally

```bash
# Install dependencies
cd analytics-service
pip install -r requirements.txt

# Start the service
uvicorn app.main:app --port 8089

# Tell the Go API where to find it
export ANALYTICS_SERVICE_URL=http://localhost:8089
```

The service auto-detects which tools are available on the system. Use `GET /healthz` to see which backends are detected.

### Backend Selection

| Env Var | Default | Options |
|---------|---------|---------|
| `ANALYTICS_TOOL` | `auto` | `auto`, `freesurfer`, `fsl`, `ants`, `spm` |

**Auto-selection priority:** FreeSurfer > FSL > ANTs > SPM (first available wins).

---

## Docker Configuration

The analytics service Docker image includes lightweight Python dependencies only. Neuroimaging tools must be provided via volume mounts or installed in a custom image.

```yaml
# docker-compose.yml snippet
analytics-service:
  build: ./analytics-service
  volumes:
    - aegis-data:/app/data
    - /opt/freesurfer:/opt/freesurfer:ro  # mount FreeSurfer
    - /opt/fsl:/opt/fsl:ro                # mount FSL
  environment:
    - FREESURFER_HOME=/opt/freesurfer
    - FSLDIR=/opt/fsl
    - FS_LICENSE=/opt/freesurfer/license.txt
```

---

## Triggering from the Dashboard

1. Navigate to a study in the **Admin Dashboard**
2. Ensure the study has completed **BIDS conversion** (Phase 2)
3. Click **Run Analytics** in the action buttons
4. The analytics status badge changes: `pending` → `running` → `complete` (or `failed`)
5. Results are recorded in the **Audit Trail** tab

If `PIPELINE_AUTO=true` (default), analytics runs automatically after BIDS conversion completes — no manual trigger needed.

---

## Triggering via MCP

AI agents can trigger analytics via the MCP server:

```
Tool: trigger_analytics
Input: { "study_id": "<uuid>", "confirm": true, "reason": "Run FreeSurfer on completed BIDS study" }
```

---

## API Reference

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/healthz` | GET | None | Service health + detected backends |
| `/analyze` | POST | Internal | Run analytics on a BIDS directory (called by Go API) |

The Go API endpoint:
- `POST /api/studies/{studyUID}/run-analytics` — triggers analytics (returns 202 Accepted, runs async)

---

## Interpreting Results

Analytics results are stored in the BIDS output directory alongside the converted NIfTI files. The specific outputs depend on the tool used:

| Tool | Output location | Key files |
|------|----------------|-----------|
| FreeSurfer | `bids/{studyUID}/derivatives/freesurfer/` | `stats/aseg.stats`, `stats/*.aparc.stats` |
| FSL | `bids/{studyUID}/derivatives/fsl/` | `*_brain.nii.gz` (BET), `*_seg.nii.gz` (FAST) |
| ANTs | `bids/{studyUID}/derivatives/ants/` | `CorticalThickness.nii.gz`, `BrainExtractionMask.nii.gz` |
| SPM | `bids/{studyUID}/derivatives/spm/` | `c1*.nii` (GM), `c2*.nii` (WM), `c3*.nii` (CSF) |

Results are also summarized in the audit trail entry (`analytics.complete`) with key metrics extracted from the tool's output.

---

## Troubleshooting

**Analytics stays in "pending":**
- Check that `ANALYTICS_SERVICE_URL` is set on the Go API
- Check that at least one analytics tool is installed and detected (`GET /healthz`)

**Analytics fails:**
- Check the analytics service logs for tool-specific error messages
- Verify the BIDS directory exists and contains valid NIfTI files
- For FreeSurfer: verify `FS_LICENSE` file is present and readable
- For FSL: verify `FSLDIR` is set and `$FSLDIR/bin/bet` exists

**Re-running analytics:**
- Use the **Reset pipeline step** option on the Study Detail Panel (select "analytics")
- Or via API: `POST /api/studies/{id}/reset-pipeline-step` with `{"step": "analytics"}`
