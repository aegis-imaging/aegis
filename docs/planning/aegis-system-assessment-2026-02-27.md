# AEGIS System Assessment — 2026-02-27

## Executive Summary

Comprehensive assessment of the AEGIS platform (Anonymization & Exchange Gateway for Imaging Studies) across all components: Go API, 4 React frontends, 9 Python sidecar services, infrastructure, and test coverage.

**Overall**: The system is remarkably complete for its phase. The Go API has 295 routes and 70 migrations, Python sidecars cover the full processing pipeline, and multi-cloud infrastructure (GCP/AWS/Azure) is provisioned. The gaps below represent the delta to production-grade completeness.

### Critical Gaps

| Priority | Gap | Impact |
|----------|-----|--------|
| P0 | No DICOM pixel decompression backends installed | JPEG2000 lossless DICOM fails across all Python services |
| P0 | Zero frontend React test coverage | No regression safety for 4 UI applications |
| P1 | Enhanced DICOM (multi-frame) not handled in pixel processing | Multi-frame DICOM crashes PHI detection, QC, pixel redaction |
| P1 | Mosaic DICOM not detected or handled | Siemens mosaic DICOM produces incorrect QC results |
| P1 | 11 Dockerfiles missing HEALTHCHECK | Container orchestrators cannot detect unhealthy services |
| P1 | Python services run as root in containers | Security compliance gap |
| P2 | No neuroimaging analytics pipelines | No FreeSurfer/FSL/ANTs/SPM integration for brain imaging analysis |
| P2 | No OpenAPI/Swagger documentation | API consumers lack machine-readable specs |
| P2 | Missing structured logging / OpenTelemetry | Limited observability in production |

---

## 1. Go API Assessment

**Scale**: 295 registered routes, 70 database migrations, ~50 handler files, ~30 model files.

### What's Complete
- Full study lifecycle: upload → classification → PHI scan → protocol check → defacing → QC → BIDS → export
- 3-phase automated pipeline orchestrator with atomic claim dispatch
- Routing engine with 10 rule actions, simulation, bulk re-evaluation, import/export
- Multi-cloud storage backends (local, GCS, S3, Azure Blob)
- Authentication: GCP IAP, Azure AD, AWS ALB+Cognito, API key fallback
- RBAC: admin/viewer roles enforced on all 33 write routes
- DICOMweb proxy (QIDO-RS + WADO-RS) for DWV viewer
- Export workflow: shares with HMAC-SHA256 signed tokens, forwarding to DICOMweb + DIMSE destinations
- Batch import CLI, bulk operations, CSV export, audit trail
- Webhook subscriptions with HMAC-signed payloads and retry delivery
- Federation peers stub, destination health probing, retention policies
- MCP server with 70+ read tools and 50+ write tools
- Comprehensive stats: pipeline funnel, processing times, routing health, cohort reports

### Gaps Identified
1. **Auto-share rule execution** — `routing/engine.go` has `case "auto_share"` with a TODO comment but no implementation. The `auto_share` action is defined in the routing rules schema but the execution path is empty.
2. **Background workers for retention** — `retention_days` is stored on projects but no background worker actually expires studies. The `RetentionWorker` is referenced in documentation but not implemented.

### Assessment: Near-complete. The API covers essentially all documented features.

---

## 2. Frontend Assessment

**Scale**: 4 React/TypeScript applications + 1 shared client library.

| App | Purpose | Component Count |
|-----|---------|----------------|
| upload-portal | Public upload + anonymization | ~15 components |
| admin-dashboard | Internal QC, management | ~40+ components |
| export-portal | Share download | ~5 components |
| landing | Marketing site + invite gate | ~10 components |
| client (lib) | DICOM anonymization library | ~10 modules |

### What's Complete
- Upload portal: drag-and-drop, multi-study detection, anonymization preview, auto-retry
- Admin dashboard: full study management, pipeline visualization, routing rules, institutions, projects, protocol templates, webhook management, audit log, stats panels
- Export portal: token-based download with server-anchored expiry countdown
- Landing page: invite gate, contact form, access request flow
- Client library: PS3.15 Basic Profile de-identification, date shifting, pseudonymization, text scrubbing, mapping export (74 vitest tests)

### Gaps Identified
1. **Zero React test coverage** — No test files exist in any of the 4 React applications. The client library has 74 vitest tests, but the UIs have none.
2. **Pixel redaction UI missing** — The backend `phi-detection/app/backends/pixel_redact.py` exists, but no admin dashboard UI exposes it. Users cannot review or confirm pixel redactions.
3. **Missing error boundaries** — No React error boundary components in any application. Unhandled errors crash the entire app.
4. **Filter presets** — `SavedFilter` localStorage feature exists but is basic. No server-side persistence or sharing between users.
5. **Routing rule reorder UI** — Priority ▲/▼ buttons exist but no drag-and-drop reorder.

### Assessment: Functionally complete for MVP; needs test coverage and polish.

---

## 3. Python Sidecar Assessment

**Scale**: 9 services, 468 total tests, FastAPI + pydicom + numpy stack.

| Service | Tests | Key Backends |
|---------|-------|-------------|
| classification-service | 70 | heuristic, gemini, google_vision, azure_vision, aws_rekognition |
| dimse-receiver | 68 | pynetdicom C-STORE SCP, retry queue, dead-letter |
| phi-detection | 47 | tesseract, gemini, google_vision, azure_vision, aws_textract |
| protocol-service | 29 | basic (Classic + Enhanced DICOM) |
| qc-service | 28 | basic (5 QC checks) |
| defacing | 26 | deepdefacer, mri_deface, nibabel |
| bids-service | 17 | dcm2niix |
| synth-service | ~10 | nibabel phantom generator |
| analytics-service | — | **Does not exist yet** |

### Critical Gap: No Pixel Decompression Backends

**None of the 9 Python service Dockerfiles install DICOM pixel decompression libraries.** This means:
- JPEG2000 lossless DICOM (`1.2.840.10008.1.2.4.90`) fails with `RuntimeError: No available image handler could decode this transfer syntax`
- JPEG Baseline/Extended also fails for pixel access
- RLE Lossless fails

The fix requires installing in every Python Dockerfile:
```
python-gdcm>=3.0
pylibjpeg>=2.0
pylibjpeg-openjpeg>=2.0
pylibjpeg-libjpeg>=2.0
```
Plus system packages: `libgdcm-tools`, `libopenjp2-7`

### Gap: Enhanced DICOM (Multi-Frame) Not Handled

Services that access `ds.pixel_array` assume a 2D array `(rows, cols)`. Enhanced DICOM returns `(frames, rows, cols)`:
- **PHI detection**: `pixel_utils.py` — `dicom_to_pil()` will fail on 3D array
- **QC service**: SNR calculation assumes single frame
- **Pixel redaction**: Assumes 2D array for masking

### Gap: Mosaic DICOM Not Detected

Siemens mosaic format packs multiple 2D slices into a single image grid. No service detects `ImageType` containing `'MOSAIC'`:
- **QC**: Slice count and consistency checks produce incorrect results
- **Defacing**: Slice threshold logic is wrong for mosaic images
- **BIDS**: dcm2niix handles mosaic natively (no issue here)

### Gap: Missing Defacing Backend Implementations

- `mri_reface` backend referenced in docs but implementation is incomplete (requires MATLAB Runtime)
- The `nibabel` backend is dev-only (zeros anterior 30% — not suitable for production)

### Gap: Cloud Backend Timeouts

Some cloud AI backends (Gemini, Cloud Vision, Textract) lack explicit HTTP timeouts in their API calls.

### Assessment: Good foundation; DICOM format support is the critical blocker.

---

## 4. Infrastructure & CI/CD Assessment

### Dockerfiles (11 services)

| Issue | Affected | Severity |
|-------|----------|----------|
| Missing `HEALTHCHECK` instruction | All 11 Dockerfiles | P1 |
| Running as root (`USER` not set) | 9 Python service Dockerfiles | P1 |
| No security scanning (Trivy/Snyk) | All images | P2 |
| No `.dockerignore` optimization | Several services | P3 |

### Terraform

| Cloud | Status | Gaps |
|-------|--------|------|
| GCP (`terraform/infra/`) | Production-deployed | Cloud Build auto-deploy working |
| AWS (`terraform/aws/`) | Provisioned | Manual deploy; ALB HTTPS + Cognito configured |
| Azure (`terraform/azure/`) | Provisioned | GitHub Actions auto-deploy; Container Apps |

### CI/CD (GitHub Actions)

| Job | Status |
|-----|--------|
| Go build + vet | Working |
| Go test (120 tests) | Working |
| Python compile (7 services) | Working |
| Python test (7 services) | Working |
| TypeScript typecheck (5 apps) | Working |
| Docker build (8 images) | Working |
| Infrastructure guard | Working |
| Cloud smoke test | Manual dispatch |
| **Frontend unit tests** | **Missing** |
| **Container security scan** | **Missing** |

### Missing Infrastructure

1. **No OpenAPI/Swagger documentation** — 295 routes with no machine-readable API spec
2. **No structured logging** — Services use plain `log.Printf` / Python `logging` without JSON structure
3. **No OpenTelemetry** — No distributed tracing across the 10-service architecture
4. **No container image scanning** — No Trivy, Snyk, or equivalent in CI pipeline

### Assessment: Solid multi-cloud foundation; needs hardening for production security compliance.

---

## 5. Test Coverage Assessment

### Current Coverage

| Component | Tests | Coverage Notes |
|-----------|-------|---------------|
| Go API (unit) | ~200 | Routing, JWT, CORS, config, email templates, storage, slugify |
| Go API (integration) | ~400 | Model CRUD, handler HTTP, auth middleware (testcontainers) |
| Go API (handler) | ~450 | Full request/response via httptest against real PostgreSQL |
| Python sidecars | 468 | All 7 services with synthetic DICOM fixtures |
| Client library | 74 | vitest: de-identification, date shifting, pseudonymization, text scrubbing |
| **React apps** | **0** | **No test files in any of the 4 React applications** |

### Untested Areas

1. **Frontend**: 0% coverage across upload-portal, admin-dashboard, export-portal, landing
2. **Storage backends**: Only `local` storage tested in Go; GCS, S3, Azure Blob untested
3. **Middleware**: Rate limiting middleware untested (33% middleware coverage)
4. **Cloud AI backends**: Mocked in tests; no integration tests against real APIs
5. **Multi-frame DICOM**: No test fixtures for Enhanced or Mosaic DICOM in any service
6. **JPEG2000 DICOM**: No test fixtures for compressed DICOM in any service

### Assessment: Good backend test foundation (1,051 Go + 468 Python + 74 client); frontend testing is a complete gap.

---

## 6. DICOM Format Support Plan

### Current State

| Format | Status | Root Cause |
|--------|--------|------------|
| Classic DICOM (uncompressed) | Fully working | N/A |
| JPEG Baseline/Extended | Broken (pixel access) | No decompression backend |
| JPEG2000 / JPEG2000 Lossless | Broken (pixel access) | No decompression backend |
| JPEG-LS Lossless | Broken (pixel access) | No decompression backend |
| RLE Lossless | Broken (pixel access) | No decompression backend |
| Enhanced DICOM (multi-frame) | Broken (crashes) | 3D pixel array not handled |
| Mosaic DICOM (Siemens) | Incorrect results | No mosaic detection |

### Fix Plan

#### 6.1 Install Decompression Backends (All 9 Python Dockerfiles)

System packages:
```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgdcm-tools libopenjp2-7 \
    && rm -rf /var/lib/apt/lists/*
```

Python packages (add to each `requirements.txt`):
```
python-gdcm>=3.0
pylibjpeg>=2.0
pylibjpeg-openjpeg>=2.0
pylibjpeg-libjpeg>=2.0
```

This gives pydicom transparent decompression of all transfer syntaxes.

#### 6.2 Enhanced DICOM Multi-Frame Handling

Update pixel access code in PHI detection, QC, and pixel redaction to handle `(frames, rows, cols)`:

```python
arr = ds.pixel_array
if arr.ndim == 3:
    # Multi-frame: process middle frame (PHI detection)
    # or iterate all frames (pixel redaction)
    mid = arr.shape[0] // 2
    arr = arr[mid]
```

#### 6.3 Mosaic DICOM Detection

```python
def is_mosaic(ds) -> bool:
    desc = getattr(ds, 'ImageType', [])
    if isinstance(desc, (list, pydicom.multival.MultiValue)):
        return 'MOSAIC' in [str(v).upper() for v in desc]
    return False
```

Update QC (skip slice consistency for mosaic) and defacing (handle tile count) services.

#### 6.4 DIMSE Receiver Transfer Syntax Support

Add all compressed transfer syntaxes to pynetdicom presentation contexts so PACS systems can send JPEG2000/JPEG-LS/RLE compressed DICOM directly.

#### 6.5 gdcmconv Pre-Processing Utility

`gdcmconv --raw input.dcm output.dcm` decompresses any compressed DICOM to raw format. Available via `apt-get install libgdcm-tools` (~5 MB). Useful as a fallback when pydicom's in-memory decompression is insufficient.

### Related Tools Evaluated

| Tool | License | Notes |
|------|---------|-------|
| **python-gdcm** | BSD | pydicom handler for all compressed formats; recommended |
| **pylibjpeg + pylibjpeg-openjpeg** | MIT | Pure Python JPEG2000 support; lighter than gdcm |
| **DCMTK (OFFIS)** | BSD-like | C++ toolkit; `dcmdjp2k` JPEG2000 module is commercial |
| **dcm4chee** | MPL 2.0 | Full PACS/VNA server (Java); useful for integration testing, not as a dependency |
| **dicom3tools (Clunie)** | MIT-like | `dciodvfy` DICOM IOD validator; useful for conformance testing |

**Recommendation**: Use python-gdcm + pylibjpeg (Python-native, no commercial modules). Use `gdcmconv` CLI as a fallback for pre-processing. Consider `dciodvfy` for CI conformance validation.

---

## 7. Neuroimaging Analytics Pipeline Plan

### Goal

Add a new `analytics-service/` Python sidecar that runs neuroimaging analysis tools on BIDS-converted studies. Integrates as Phase 3 in the pipeline (post-BIDS conversion).

```
Phase 0: Classification
Phase 1: PHI scan + Pixel redaction + Protocol check + Defacing (parallel)
Phase 2: QC + BIDS conversion (post-defacing)
Phase 3: Analytics (post-BIDS) ← NEW
```

### Tools

#### FreeSurfer (v8.x)
- **Docker**: `freesurfer/freesurfer` (official), `antsx/freesurfer` (minimal ~360 MB)
- **License**: Free registration required; license file mounted at runtime (`FS_LICENSE` env var)
- **Key command**: `recon-all -s <subject> -i <T1w.nii.gz> -all`
- **Duration**: ~2.5 hours (v8.0) to ~8 hours (v7.x); `recon-all-clinical` faster (~1 hour)
- **Inputs**: T1w NIfTI from BIDS `anat/sub-*_T1w.nii.gz`
- **Outputs**:
  - `stats/aseg.stats` — subcortical volumetrics (hippocampus, amygdala, thalamus, caudate, putamen)
  - `stats/lh.aparc.stats`, `rh.aparc.stats` — cortical parcellation (Desikan-Killiany atlas, 34 regions/hemisphere)
  - `stats/lh.aparc.a2009s.stats`, `rh.aparc.a2009s.stats` — Destrieux atlas (74 regions/hemisphere)
  - `surf/lh.thickness`, `rh.thickness` — cortical thickness maps
  - `mri/brain.mgz` — skull-stripped brain
  - `mri/aseg.mgz` — subcortical segmentation volume

#### FSL (v6.x)
- **Docker**: `brainlife/fsl` or custom build
- **License**: Free for academic use (Oxford FMRIB)
- **Key tools**:
  - `bet` — brain extraction (~1 min); outputs: `*_brain.nii.gz`, `*_brain_mask.nii.gz`
  - `fast` — tissue segmentation GM/WM/CSF (~5 min); outputs: `*_seg.nii.gz`, `*_pve_0/1/2.nii.gz`
  - `flirt` — linear registration to MNI152 (~2 min); outputs: `*_to_mni.nii.gz`, `*.mat`
  - `dtifit` — diffusion tensor fitting; outputs: FA, MD, RD, AD maps
  - `eddy` — diffusion distortion correction
- **Duration**: Full pipeline ~30 min
- **Inputs**: NIfTI from BIDS (`anat/`, `dwi/`)

#### ANTs (Advanced Normalization Tools)
- **Docker**: `antsx/ants` (Docker Hub)
- **License**: Apache 2.0 (fully open)
- **Key tools**:
  - `antsBrainExtraction.sh` — brain extraction with atlas priors
  - `antsRegistrationSyN.sh` / `antsRegistrationSyNQuick.sh` — diffeomorphic registration
  - `antsCorticalThickness.sh` — full cortical thickness pipeline (extraction + segmentation + registration + thickness)
  - `N4BiasFieldCorrection` — inhomogeneity correction
- **Outputs**:
  - `BrainExtractionMask.nii.gz`, `BrainSegmentation.nii.gz`
  - `BrainSegmentationPosteriors*.nii.gz` — probability maps (CSF, GM, WM, deep GM, brainstem, cerebellum)
  - `CorticalThickness.nii.gz` — voxelwise cortical thickness
  - `*Warp.nii.gz`, `*Affine.txt` — deformation fields
- **Duration**: antsCorticalThickness ~4-8 hours; registration alone ~30 min–2 hours

#### SPM (Statistical Parametric Mapping)
- **Docker**: `spmcentral/spm` (official, MATLAB Compiler Runtime ~3 GB)
- **License**: GPL v2 (free); standalone doesn't require MATLAB license
- **Key functions**:
  - Segmentation — tissue classification (GM/WM/CSF) + bias correction
  - DARTEL normalization — high-dimensional warping to template
  - VBM (Voxel-Based Morphometry) — smoothed, normalized grey matter images
- **Duration**: Segmentation ~5 min, DARTEL ~30 min per subject

### Architecture

#### Service Structure
```
analytics-service/
├── Dockerfile
├── requirements.txt
├── requirements-test.txt
├── app/
│   ├── __init__.py
│   ├── main.py                    # FastAPI: POST /analyze, GET /healthz
│   ├── config.py                  # Env var parsing, tool selection
│   └── backends/
│       ├── __init__.py
│       ├── base.py                # AnalyticsBackend ABC
│       ├── freesurfer.py          # recon-all wrapper
│       ├── fsl.py                 # BET + FAST + FLIRT + DTIFIT
│       ├── ants.py                # antsCorticalThickness + registration
│       └── spm.py                 # Segmentation + DARTEL
└── tests/
    ├── conftest.py
    ├── test_main.py
    ├── test_freesurfer.py
    ├── test_fsl.py
    ├── test_ants.py
    └── test_spm.py
```

#### Database Migration
```sql
ALTER TABLE studies
    ADD COLUMN analytics_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN analytics_status TEXT NOT NULL DEFAULT ''
        CHECK (analytics_status IN ('', 'pending', 'processing', 'complete', 'partial', 'failed'));
```

The `partial` status covers the case where some tools succeed and others fail (e.g., FreeSurfer succeeds but FSL fails).

#### Go API Integration

- **Handler**: `api/handler/analytics.go` — `TriggerAnalytics()` (validate, check BIDS complete, dispatch async, return 202) + `runAnalytics()` (POST to sidecar, update status, audit trail)
- **Pipeline**: Add Phase 3 gate — dispatch analytics only after `bids_status == "complete"`
- **Routing**: Add `require_analytics` action to routing engine
- **Config**: `ANALYTICS_SERVICE_URL` env var (empty = disabled)
- **Route**: `POST /api/studies/{studyUID}/analytics`

#### Dockerfile (Multi-Stage, Conditional Tools)
```dockerfile
FROM python:3.12-slim AS base
ARG INCLUDE_FREESURFER=false
ARG INCLUDE_FSL=false
ARG INCLUDE_ANTS=false
ARG INCLUDE_SPM=false
# Each tool optional via build arg; auto mode detects what's installed
```

#### Environment Variables

| Var | Default | Notes |
|-----|---------|-------|
| `ANALYTICS_SERVICE_URL` | *(empty — disabled)* | Set to enable |
| `ANALYTICS_TOOL` | `auto` | `auto\|freesurfer\|fsl\|ants\|spm` |
| `ANALYTICS_TIMEOUT` | `86400` | Max seconds per study (24h for FreeSurfer) |
| `FS_LICENSE` | `/app/secrets/freesurfer.license` | FreeSurfer license path |
| `FSL_DIR` | `/usr/local/fsl` | FSL installation directory |
| `ANTSPATH` | `/usr/local/ANTs/bin` | ANTs binary directory |

### Design Notes
- **Long-running**: FreeSurfer can take 2.5-8 hours; generous timeouts and polling needed
- **Resource-heavy**: Limit concurrent analyses (1 per tool per instance)
- **Partial success**: If FreeSurfer succeeds but FSL fails, status = `partial` with per-tool results in audit
- **BIDS dependency**: Analytics only fires after BIDS conversion completes
- **License management**: FreeSurfer license mounted via Docker secret; no license = backend reports unavailable
- **Results storage**: Analytics outputs stored in `analytics/{studyUID}/` under shared volume
- **Download**: `GET /api/studies/{studyUID}/analytics-download` — ZIP of all analytics outputs

---

## 8. Priority Recommendations

### Tier 1 — Critical (Do Now)

1. **Fix DICOM format support** — Install decompression backends in all Python Dockerfiles, handle Enhanced DICOM multi-frame arrays, detect Mosaic DICOM. This is blocking real-world clinical data ingestion.

2. **Frontend test coverage** — Add vitest + React Testing Library to all 4 React apps. Start with the admin dashboard (most complex, highest risk).

### Tier 2 — Important (Next Sprint)

3. **Neuroimaging analytics service** — Build the `analytics-service/` sidecar with FreeSurfer, FSL, ANTs, SPM backends. This is the biggest value-add for research users.

4. **Docker hardening** — Add `HEALTHCHECK` to all 11 Dockerfiles. Add non-root `USER` to Python services. Add container image scanning to CI.

5. **Storage backend testing** — Integration tests for GCS, S3, and Azure Blob storage (currently only local is tested).

### Tier 3 — Nice to Have (Backlog)

6. **OpenAPI/Swagger documentation** — Generate from Go handler annotations or maintain a separate spec file.

7. **Structured logging + OpenTelemetry** — JSON log format, distributed tracing across the 10-service architecture.

8. **Pixel redaction UI** — Admin dashboard interface for reviewing and confirming pixel redactions.

9. **Auto-share rule execution** — Complete the `auto_share` routing rule action implementation.

10. **Retention background worker** — Implement the study expiry worker that enforces `retention_days`.

---

## Source Citations

### DICOM Standards
- DICOM PS3.5 §8: Transfer Syntaxes — defines JPEG2000, JPEG-LS, RLE encoding
- DICOM PS3.3 §C.7.6.6: Multi-frame Functional Groups — Enhanced DICOM specification
- DICOM PS3.15 Annex E: Basic Application Level Confidentiality Profile

### Software Tools
- **GDCM (Grassroots DICOM)**: https://gdcm.sourceforge.net/ — BSD license, `gdcmconv` CLI + python-gdcm bindings
- **pylibjpeg**: https://github.com/pydicom/pylibjpeg — MIT, JPEG2000/JPEG-LS decompression for pydicom
- **DCMTK (OFFIS)**: https://dicom.offis.de/en/dcmtk/ — BSD-like; note `dcmdjp2k` JPEG2000 module requires commercial license
- **dcm4chee**: https://www.dcm4che.org/ — MPL 2.0, Java PACS/VNA server
- **dicom3tools (Clunie)**: https://www.dclunie.com/dicom3tools.html — `dciodvfy` DICOM IOD validator

### Neuroimaging Tools
- **FreeSurfer**: Fischl, B. (2012). FreeSurfer. NeuroImage, 62(2), 774-781. https://surfer.nmr.mgh.harvard.edu/
- **FSL**: Jenkinson, M. et al. (2012). FSL. NeuroImage, 62(2), 782-790. https://fsl.fmrib.ox.ac.uk/fsl/
- **ANTs**: Avants, B.B. et al. (2011). A reproducible evaluation of ANTs similarity metric performance in brain image registration. NeuroImage, 54(3), 2033-2044. https://github.com/ANTsX/ANTs
- **SPM**: Ashburner, J. (2012). SPM: A history. NeuroImage, 62(2), 791-800. https://www.fil.ion.ucl.ac.uk/spm/

---

## Implementation Order

```
1. DICOM format support     — cross-cutting infrastructure, enables everything else
2. Neuroimaging analytics   — largest scope, depends on BIDS + DICOM support
3. Gap analysis follow-ups  — frontend tests, Docker hardening, etc.
```

Suggested branching:
- `feature/dicom-format-support` — JPEG2000 + Enhanced + Mosaic (PR 1)
- `feature/analytics-service` — FreeSurfer/FSL/ANTs/SPM sidecar + Go API (PR 2)
