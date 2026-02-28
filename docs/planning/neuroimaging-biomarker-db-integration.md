# Neuroimaging Biomarker Database Integration Plan

**Date:** 2026-02-28
**Status:** Draft — awaiting review
**Origin:** Matt Senjem's Neuroimaging Biomarker Database design (HINF 5510, Dec 2021)

## Executive Summary

Integrate a structured neuroimaging biomarker database into AEGIS to close the "last mile" gap between running analytics (FreeSurfer, FSL, ANTs, SPM, Atlas ROI, TBM-SyN) and storing/querying the results in a research-ready format. Adds three major capabilities:

1. **Structured ROI biomarker storage** — queryable per-region metrics (volume, mean, median, std) across subjects and timepoints
2. **Human analyst QC ratings** — structured Likert-scale quality + motion ratings alongside existing automated QC
3. **Web-based NIfTI & surface viewer** — Niivue-powered viewer for NIfTI volumes, FreeSurfer surfaces, atlas overlays, and longitudinal analytics results (replacing desktop tools like FSLeyes, MRIcroGL, FreeView)

Plus a de-identified subject demographics table for research metadata (sex, education, diagnosis code, MMSE).

---

## Part 1: Structured ROI Biomarker Storage

### Background

AEGIS currently runs 7 analytics backends (FreeSurfer, FSL, ANTs, SPM, Atlas ROI, TBM-SyN, FreeSurfer Long) but stores results only as:
- Output files on disk (`analytics/{studyUID}/`)
- Summary metadata in audit trail entries (`analytics.complete`)

There is no structured database storage, so ROI biomarkers are not queryable, exportable, or comparable across subjects/timepoints.

### Database Schema

#### Migration: `roi_results` table

```sql
CREATE TABLE roi_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    tool TEXT NOT NULL,                    -- 'freesurfer', 'atlas_roi', 'fsl', 'ants', 'spm', 'tbm_syn', 'freesurfer_long'
    atlas_name TEXT NOT NULL DEFAULT '',   -- 'aal3', 'aparc', 'aseg', 'mcalt', etc.
    template_name TEXT NOT NULL DEFAULT '', -- 'OASIS-30', 'MNI152', 'MCALT', etc.
    roi_number INT NOT NULL DEFAULT 0,     -- integer label in atlas (for colormap lookup)
    roi_name TEXT NOT NULL,                -- 'Left-Hippocampus', 'Hippocampus_L', etc.
    metric_type TEXT NOT NULL,             -- 'volume_mm3', 'mean_intensity', 'median_intensity', 'std', 'thickness_mm', 'suvr', 'atrophy_rate', 'jacobian_mean'
    metric_value DOUBLE PRECISION NOT NULL,
    hemisphere TEXT NOT NULL DEFAULT '',    -- 'L', 'R', 'bilateral', ''
    scan_type TEXT NOT NULL DEFAULT '',     -- 'T1', 'T2', 'PET-PIB', 'PET-FDG', 'PET-AV1451', etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Query patterns: per-study, per-subject across studies, per-ROI across subjects
CREATE INDEX idx_roi_results_study ON roi_results(study_id);
CREATE INDEX idx_roi_results_roi_name ON roi_results(roi_name);
CREATE INDEX idx_roi_results_tool_atlas ON roi_results(tool, atlas_name);
CREATE INDEX idx_roi_results_metric ON roi_results(metric_type);
```

#### Migration: `longitudinal_roi_results` table

For TBM-SyN and FreeSurfer Long paired results:

```sql
CREATE TABLE longitudinal_roi_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    followup_study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    tool TEXT NOT NULL,                     -- 'tbm_syn', 'freesurfer_long'
    atlas_name TEXT NOT NULL DEFAULT '',
    roi_name TEXT NOT NULL,
    baseline_value DOUBLE PRECISION,        -- e.g. baseline volume
    followup_value DOUBLE PRECISION,        -- e.g. follow-up volume
    change_value DOUBLE PRECISION,          -- absolute change
    change_percent DOUBLE PRECISION,        -- percentage change
    annualized_change DOUBLE PRECISION,     -- annualized rate (per year)
    metric_type TEXT NOT NULL,              -- 'volume_mm3', 'atrophy_rate', 'jacobian_mean', 'thickness_mm'
    scan_interval_days INT NOT NULL DEFAULT 0,
    hemisphere TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_long_roi_baseline ON longitudinal_roi_results(baseline_study_id);
CREATE INDEX idx_long_roi_followup ON longitudinal_roi_results(followup_study_id);
CREATE INDEX idx_long_roi_name ON longitudinal_roi_results(roi_name);
```

#### Migration: `analytics_composite_scores` table

For derived summary scores (AD composite, total intracranial volume, etc.):

```sql
CREATE TABLE analytics_composite_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    baseline_study_id UUID REFERENCES studies(id),  -- NULL for single-study scores
    tool TEXT NOT NULL,
    score_name TEXT NOT NULL,               -- 'ad_composite', 'total_icv', 'brain_volume', 'mean_cortical_thickness'
    score_value DOUBLE PRECISION NOT NULL,
    metadata JSONB DEFAULT '{}',            -- additional context (roi_count, contributing regions, etc.)
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_composite_study ON analytics_composite_scores(study_id);
CREATE INDEX idx_composite_name ON analytics_composite_scores(score_name);
```

### Go API Changes

#### Model: `api/model/roi_result.go`

```
ROIResult struct:
  ID, StudyID, Tool, AtlasName, TemplateName, ROINumber, ROIName,
  MetricType, MetricValue, Hemisphere, ScanType, CreatedAt

Functions:
  CreateROIResults(ctx, db, results []ROIResult) error         -- bulk INSERT
  GetROIResultsByStudy(ctx, db, studyID) ([]ROIResult, error)
  GetROIResultsBySubject(ctx, db, subjectID, projectID) ([]ROIResult, error)  -- joins studies on subject_id
  DeleteROIResultsByStudy(ctx, db, studyID) error              -- for re-processing
```

#### Handler: `api/handler/roi_results.go`

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/studies/{id}/roi-results` | read | Per-study ROI results, filterable by tool/atlas/metric_type |
| `GET /api/studies/{id}/roi-results/summary` | read | Aggregated summary (total ROIs, tools used, atlas names) |
| `GET /api/subjects/{subjectID}/roi-results` | read | Cross-study ROI comparison for a subject (longitudinal) |
| `GET /api/projects/{id}/roi-export` | read | CSV export of all ROI results for a project |
| `DELETE /api/studies/{id}/roi-results` | admin | Clear results (for re-processing) |

#### Analytics callback enhancement

When `runAnalytics()` receives the response from the Python service, parse `metrics` and bulk-insert into `roi_results`:

- **FreeSurfer**: `subcortical_volumes` → metric_type=`volume_mm3`, atlas=`aseg`; cortical parcellation → atlas=`aparc`
- **Atlas ROI**: `roi_volumes` → metric_type=`volume_mm3`, atlas from response
- **TBM-SyN**: per-ROI atrophy rates → `longitudinal_roi_results`; AD composite → `analytics_composite_scores`
- **FreeSurfer Long**: baseline/followup volumes + change_percent → `longitudinal_roi_results`

### MCP Tools

| Tool | Type | Description |
|------|------|-------------|
| `get_roi_results` | read | Per-study ROI results with filters |
| `get_subject_roi_comparison` | read | Cross-timepoint ROI comparison for a subject |
| `get_composite_scores` | read | Analytics composite scores (AD composite, ICV, etc.) |
| `export_project_roi_data` | read | Bulk CSV export for a project |

### Admin Dashboard

- **Biomarkers tab** on study detail panel: sortable table of ROI results grouped by atlas, with bar charts for key regions
- **Subject longitudinal view**: side-by-side ROI comparison across timepoints (table + sparkline trends)
- **Project ROI export**: "Export ROI Data (CSV)" button in Projects tab

---

## Part 2: Human Analyst QC Ratings

### Background

AEGIS has automated QC (SNR, slice consistency, missing slices) but no structured human review workflow. The original Neuroimaging Biomarker Database design includes a `ScanQuality` table with Likert-scale ratings — this fills that gap.

### Database Schema

#### Migration: `analyst_qc_ratings` table

```sql
CREATE TABLE analyst_qc_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    analyst_id UUID NOT NULL REFERENCES admin_users(id),
    rating_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    rating_quality SMALLINT NOT NULL CHECK (rating_quality BETWEEN 0 AND 4),
        -- 0: unusable, 1: poor, 2: fair, 3: good, 4: excellent
    rating_motion SMALLINT NOT NULL CHECK (rating_motion BETWEEN 0 AND 4),
        -- 0: no motion, 1: slight, 2: moderate, 3: severe, 4: extreme/unusable
    rating_snr SMALLINT CHECK (rating_snr BETWEEN 0 AND 4),
        -- optional: 0: unusable, 1: poor, 2: fair, 3: good, 4: excellent
    rating_coverage SMALLINT CHECK (rating_coverage BETWEEN 0 AND 4),
        -- optional: 0: incomplete, 1: poor, 2: fair, 3: good, 4: full
    rating_artifacts SMALLINT CHECK (rating_artifacts BETWEEN 0 AND 4),
        -- optional: 0: none, 1: minimal, 2: moderate, 3: significant, 4: severe
    rating_overall SMALLINT NOT NULL CHECK (rating_overall BETWEEN 0 AND 4),
        -- composite: 0: reject, 1: marginal, 2: acceptable, 3: good, 4: excellent
    comments TEXT NOT NULL DEFAULT '',
    review_type TEXT NOT NULL DEFAULT 'standard'
        CHECK (review_type IN ('standard', 'defacing', 'analytics', 'protocol')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_qc_ratings_study ON analyst_qc_ratings(study_id);
CREATE INDEX idx_qc_ratings_analyst ON analyst_qc_ratings(analyst_id);
CREATE UNIQUE INDEX idx_qc_ratings_unique ON analyst_qc_ratings(study_id, analyst_id, review_type);
```

### Go API

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/studies/{id}/qc-ratings` | read | All ratings for a study |
| `POST /api/studies/{id}/qc-ratings` | admin | Submit a QC rating (one per analyst per review_type) |
| `PUT /api/qc-ratings/{id}` | admin | Update own rating |
| `DELETE /api/qc-ratings/{id}` | admin | Delete own rating |
| `GET /api/qc-ratings/summary` | read | Aggregate QC metrics: mean quality, inter-rater agreement |

### Admin Dashboard

- **QC Rating form** on study detail panel: Likert radio buttons for quality (0–4) and motion (0–4) with text labels, optional SNR/coverage/artifacts ratings, free-text comments
- **Rating display**: show all analyst ratings for the study with colored badges (red/orange/yellow/green/teal for 0–4)
- **Inter-rater summary**: when multiple analysts rate the same study, show agreement metrics

---

## Part 3: De-identified Subject Demographics

### Background

The original design stores PHI (patient names, birthdates). AEGIS replaces this with de-identified research metadata keyed by `subject_id`.

### Database Schema

#### Migration: `subject_demographics` table

```sql
CREATE TABLE subject_demographics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id TEXT NOT NULL,
    project_id UUID NOT NULL REFERENCES projects(id),
    sex TEXT NOT NULL DEFAULT '' CHECK (sex IN ('', 'M', 'F', 'NB', 'O')),
    age_at_scan INT,                        -- age in years at time of scan (not birthdate)
    diagnosis TEXT NOT NULL DEFAULT '',      -- e.g. 'CN', 'MCI', 'AD', 'FTD', 'DLB'
    education_years SMALLINT,               -- 0–25 typical
    mmse_score SMALLINT CHECK (mmse_score BETWEEN 0 AND 30),
    moca_score SMALLINT CHECK (moca_score BETWEEN 0 AND 30),
    cdr_global NUMERIC(3,1) CHECK (cdr_global IN (0, 0.5, 1, 2, 3)),
    apoe_genotype TEXT DEFAULT '',           -- e.g. 'e3/e4', 'e4/e4'
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(subject_id, project_id)
);

CREATE INDEX idx_subj_demo_subject ON subject_demographics(subject_id);
CREATE INDEX idx_subj_demo_project ON subject_demographics(project_id);
```

**Key design decisions:**
- `age_at_scan` instead of birthdate (Safe Harbor compliant — no dates, no ages over 89)
- Keyed by `subject_id` + `project_id` (not patient name)
- Standard cognitive scores: MMSE (0–30), MoCA (0–30), CDR (0, 0.5, 1, 2, 3)
- APOE genotype for AD research (non-identifying)
- No names, no dates, no MRN — fully de-identified

### Go API

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/subjects/{subjectID}/demographics` | read | Get demographics for a subject in a project |
| `PUT /api/subjects/{subjectID}/demographics` | admin | Create or update demographics (upsert) |
| `GET /api/projects/{id}/demographics` | read | All subject demographics for a project |
| `GET /api/projects/{id}/demographics.csv` | read | CSV export |

### Admin Dashboard

- **Demographics panel** on subject detail view: editable form for sex, age, diagnosis, cognitive scores
- **Project demographics table**: sortable list of all subjects with demographic columns

---

## Part 4: Web-Based NIfTI & Surface Viewer (Niivue)

### Background

AEGIS currently uses Weasis DWV for viewing raw DICOM files. But analytics pipeline outputs are NIfTI volumes, FreeSurfer surfaces, and overlay maps — formats Weasis cannot display. Desktop tools (FSLeyes, MRIcroGL, FreeView) require local installation. A web-based viewer is needed.

### Tool Selection: Niivue

**[Niivue](https://github.com/niivue/niivue)** is the clear choice for web-based neuroimaging visualization:

| Criteria | Niivue | FSLeyes (web) | BrainBrowser | MRIcroGL |
|----------|--------|--------------|-------------|----------|
| **Runtime** | Browser (WebGL2) | Limited web ver | Browser | Desktop only |
| **NIfTI support** | Yes (.nii, .nii.gz) | Yes | Limited | Yes |
| **FreeSurfer surfaces** | Yes (pial, inflated, white) | No | Yes | No |
| **FreeSurfer overlays** | Yes (curv, annot, thickness) | No | Yes | No |
| **MGH/MGZ volumes** | Yes | Yes | No | Yes |
| **Atlas overlays** | Yes (multi-layer) | Yes | No | Yes |
| **React integration** | `@niivue/niivue` npm + `niivue-react` | N/A | Custom | N/A |
| **License** | BSD-2-Clause | Apache 2.0 | MIT | BSD |
| **Actively maintained** | Yes (v0.67, 2026) | Desktop only | Dormant | Desktop only |
| **npm package** | `@niivue/niivue` | N/A | N/A | N/A |

**Notable:** FreeSurfer's own team built **[FreeBrowse](https://github.com/freesurfer/freebrowse)** on top of Niivue — validating it as the standard for web-based FreeSurfer visualization.

### Architecture

```
Admin Dashboard (React)
  └─ NiivueViewer component
       ├─ Volume mode: NIfTI slices (axial/coronal/sagittal) + 3D render
       ├─ Surface mode: FreeSurfer pial/inflated surfaces + overlays
       ├─ Overlay mode: atlas ROI labels, thickness maps, Jacobian maps
       └─ Comparison mode: side-by-side baseline vs follow-up
            │
            ▼
     Go API (file serving endpoints)
       ├─ GET /api/studies/{id}/analytics-files          → list available files
       ├─ GET /api/studies/{id}/analytics-files/{path}   → stream NIfTI/surface file
       └─ GET /api/studies/{id}/analytics-files/{path}/metadata → NIfTI header info
```

### Implementation

#### 1. Go API: Analytics file serving

New handler `api/handler/analytics_files.go`:

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/studies/{id}/analytics-files` | read | List all analytics output files for a study (tree structure) |
| `GET /api/studies/{id}/analytics-files/{path...}` | read | Stream a specific file (NIfTI, surface, CSV, JSON) |

The file listing endpoint returns a structured tree:

```json
{
  "study_id": "...",
  "files": [
    {
      "path": "freesurfer/sub-abc123/mri/brain.mgz",
      "tool": "freesurfer",
      "type": "volume",
      "size_bytes": 12345678,
      "description": "Brain extraction (skull-stripped)"
    },
    {
      "path": "freesurfer/sub-abc123/surf/lh.pial",
      "tool": "freesurfer",
      "type": "surface",
      "description": "Left hemisphere pial surface"
    },
    {
      "path": "atlas_roi/warped_atlas.nii.gz",
      "tool": "atlas_roi",
      "type": "overlay",
      "description": "AAL3 atlas warped to subject space"
    },
    {
      "path": "atlas_roi/roi_stats.csv",
      "tool": "atlas_roi",
      "type": "data",
      "description": "Per-ROI volumetric statistics"
    }
  ]
}
```

File type classification based on path patterns:
- `*.nii`, `*.nii.gz`, `*.mgz`, `*.mgh` → `volume`
- `*/surf/*` (no extension or `.pial`, `.white`, `.inflated`) → `surface`
- `*.curv`, `*.annot`, `*.thickness` → `surface_overlay`
- `*atlas*`, `*labels*`, `*seg*`, `*aseg*` → `overlay`
- `*.csv`, `*.json`, `*.stats` → `data`

#### 2. React: Niivue viewer component

New file: `frontend/admin-dashboard/src/components/NiivueViewer.tsx`

```
Dependencies:
  npm install @niivue/niivue

Component props:
  studyId: string
  mode: 'volume' | 'surface' | 'overlay' | 'comparison'
  files?: string[]         -- pre-selected file paths
  baselineStudyId?: string -- for comparison mode
```

**View modes:**

| Mode | Use Case | Niivue Config |
|------|----------|---------------|
| **Volume** | View brain.mgz, T1w.nii.gz, segmentations | Multiplanar (axial/coronal/sagittal) + 3D render |
| **Surface** | View pial/inflated surfaces with thickness/curvature overlays | 3D mesh rendering with colormap |
| **Overlay** | View T1w with atlas ROI labels overlaid (like Fig. 1 in the original paper) | Volume + overlay layers with configurable opacity |
| **Comparison** | Side-by-side baseline vs follow-up (TBM-SyN Jacobian map) | Dual-canvas synced navigation |

**Viewer toolbar controls:**
- Slice navigation (axial/coronal/sagittal scrollbars)
- Colormap selector (for overlays: viridis, hot, cool, spectral, FreeSurfer LUT)
- Opacity slider (for atlas/overlay layers)
- Crosshair toggle
- 3D render toggle
- Screenshot button
- File picker dropdown (populated from analytics-files listing)

#### 3. Integration points in admin dashboard

| Location | Trigger | Viewer Mode |
|----------|---------|-------------|
| Study detail panel → "View Analytics" button | Click | Volume mode with brain + atlas overlay |
| Study detail panel → "View Surfaces" button | Click (FreeSurfer studies only) | Surface mode with pial + thickness |
| Study detail panel → "View Longitudinal" button | Click (paired studies) | Comparison mode: baseline T1w vs Jacobian map |
| ROI results table → ROI name click | Click on specific ROI | Volume mode with atlas, highlight clicked ROI |
| Defacing review (existing) | Enhance existing | Add NIfTI view alongside existing DICOM Weasis view |

#### 4. Niivue service container (optional, for SSR/thumbnails)

For generating static thumbnail images of analytics results (used in study list cards, email digests, PDF reports):

```
niivue-renderer/
  ├── Dockerfile        # Node.js + headless Chrome + Niivue
  ├── app.js            # Express server
  └── POST /render      # {nifti_url, overlay_url?, view, size} → PNG thumbnail
```

This is **optional/future** — the primary viewer is client-side in the admin dashboard.

### Viewer Presets

Pre-configured view recipes for common analytics outputs:

| Preset | Tool | Files Loaded | Description |
|--------|------|-------------|-------------|
| FreeSurfer Recon | freesurfer | brain.mgz + aseg.mgz overlay | Subcortical segmentation review |
| FreeSurfer Surfaces | freesurfer | lh.pial + rh.pial + lh.thickness overlay | Cortical thickness on pial surface |
| Atlas ROI | atlas_roi | T1w.nii.gz + warped_atlas.nii.gz | Atlas labels overlaid on structural MRI |
| FSL Segmentation | fsl | brain.nii.gz + brain_seg.nii.gz | Tissue segmentation (GM/WM/CSF) |
| TBM-SyN Atrophy | tbm_syn | baseline T1w + log_jacobian_annualized.nii.gz | Annualized atrophy rate map |
| ANTs Thickness | ants | CorticalThickness.nii.gz | Cortical thickness volume map |

---

## Part 5: Implementation Phases

### Phase A: Database + ROI Storage (foundation)

**Estimated scope: 4 migrations, 3 model files, 3 handlers, tests**

1. Migration: `roi_results` table
2. Migration: `longitudinal_roi_results` table
3. Migration: `analytics_composite_scores` table
4. Migration: `analyst_qc_ratings` table
5. Migration: `subject_demographics` table
6. Model + handler: ROI results CRUD + bulk insert
7. Model + handler: Analyst QC ratings CRUD
8. Model + handler: Subject demographics CRUD
9. Enhance `runAnalytics()` to parse metrics → bulk insert ROI results
10. Enhance `runLongitudinalAnalytics()` to parse metrics → insert longitudinal results + composite scores
11. Tests: model integration tests, handler HTTP tests

### Phase B: API Endpoints + MCP Tools

**Estimated scope: 4 handlers, 8 MCP tools**

1. Handler: `GET /api/studies/{id}/roi-results` + summary + filters
2. Handler: `GET /api/subjects/{subjectID}/roi-results` (cross-study longitudinal)
3. Handler: `GET /api/projects/{id}/roi-export` (CSV)
4. Handler: `GET /api/studies/{id}/analytics-files` + file streaming
5. Handler: `GET /api/projects/{id}/demographics.csv`
6. MCP tools: `get_roi_results`, `get_subject_roi_comparison`, `get_composite_scores`, `export_project_roi_data`
7. MCP tools: `get_qc_ratings`, `submit_qc_rating`
8. Tests for all new endpoints

### Phase C: Niivue Viewer Integration

**Estimated scope: 1 npm dependency, 4 React components, viewer presets**

1. `npm install @niivue/niivue` in admin-dashboard
2. `NiivueViewer.tsx` — core viewer component wrapping Niivue canvas
3. `AnalyticsFileExplorer.tsx` — file tree browser with type icons
4. `ViewerToolbar.tsx` — slice navigation, colormap, opacity, crosshair controls
5. `ComparisonViewer.tsx` — dual-canvas side-by-side for longitudinal
6. Integration: "View Analytics" button on study detail panel
7. Integration: "View Surfaces" button (FreeSurfer studies)
8. Integration: "View Longitudinal" button (paired studies)
9. Viewer presets (FreeSurfer Recon, Atlas ROI, TBM-SyN Atrophy, etc.)

### Phase D: Admin Dashboard UI

**Estimated scope: 5 React components/panels**

1. **Biomarkers tab** on study detail: ROI results table + bar chart
2. **QC Rating form** on study detail: Likert radio buttons + comments
3. **Subject demographics panel**: editable form
4. **Subject longitudinal view**: cross-timepoint ROI comparison table
5. **Project ROI export button**: CSV download
6. **Cohort report enhancement**: integrate demographics + mean ROI values

### Phase E: Polish + Documentation

1. CLAUDE.md updates for all new endpoints, tables, components
2. Python test updates (analytics service response parsing)
3. Go test coverage for new models/handlers
4. Admin dashboard: viewer presets and keyboard shortcuts
5. MCP tool documentation

---

## File Impact Summary

### New Files

| File | Description |
|------|-------------|
| `api/migrate/migrations/074_roi_results.sql` | ROI results + longitudinal + composite tables |
| `api/migrate/migrations/075_analyst_qc_ratings.sql` | Human QC ratings table |
| `api/migrate/migrations/076_subject_demographics.sql` | De-identified demographics table |
| `api/model/roi_result.go` | ROI result model + CRUD |
| `api/model/longitudinal_roi_result.go` | Longitudinal ROI result model |
| `api/model/composite_score.go` | Analytics composite score model |
| `api/model/analyst_qc_rating.go` | Human QC rating model |
| `api/model/subject_demographics.go` | Subject demographics model |
| `api/handler/roi_results.go` | ROI results API endpoints |
| `api/handler/analyst_qc_rating.go` | QC rating API endpoints |
| `api/handler/subject_demographics.go` | Demographics API endpoints |
| `api/handler/analytics_files.go` | Analytics file listing + streaming |
| `frontend/admin-dashboard/src/components/NiivueViewer.tsx` | Core Niivue viewer component |
| `frontend/admin-dashboard/src/components/AnalyticsFileExplorer.tsx` | File tree browser |
| `frontend/admin-dashboard/src/components/ViewerToolbar.tsx` | Viewer controls |
| `frontend/admin-dashboard/src/components/ComparisonViewer.tsx` | Side-by-side longitudinal viewer |
| `frontend/admin-dashboard/src/components/BiomarkersPanel.tsx` | ROI results display |
| `frontend/admin-dashboard/src/components/QCRatingForm.tsx` | Human QC rating form |
| `frontend/admin-dashboard/src/components/SubjectDemographicsPanel.tsx` | Demographics editor |

### Modified Files

| File | Change |
|------|--------|
| `api/handler/analytics.go` | Parse metrics → bulk insert ROI results after analytics complete |
| `api/handler/analytics_longitudinal.go` | Parse metrics → insert longitudinal results + composite scores |
| `api/main.go` | Register new routes |
| `frontend/admin-dashboard/src/App.tsx` | Add viewer buttons, biomarkers tab, QC form |
| `frontend/admin-dashboard/package.json` | Add `@niivue/niivue` dependency |
| `mcp-server/src/index.ts` | Add new MCP tools |
| `CLAUDE.md` | Document new tables, endpoints, components |

---

## Mapping: Original Design → AEGIS Implementation

| Original Table | AEGIS Equivalent | Adaptation |
|----------------|-----------------|------------|
| `Patients` | `subject_demographics` + studies.`subject_id` | De-identified: no names/birthdates; age_at_scan instead |
| `PatientScans` | `studies` + `study_series` | Already exists; voxel dimensions available from DICOM tags |
| `ScanRegions` | `roi_results` | Enhanced: multiple metric types, hemisphere, scan_type |
| `Analysts` | `admin_users` | Already exists; same role-based access |
| `ScanQuality` | `analyst_qc_ratings` | Enhanced: additional rating dimensions (SNR, coverage, artifacts) |
| `roi_atlas_name` | `roi_results.atlas_name` | Direct mapping; supports AAL3, aparc, aseg, MCALT, custom |
| `roi_template_name` | `roi_results.template_name` | Direct mapping; OASIS-30, MNI152, MCALT |
| `scan_filename` (NIfTI) | BIDS output paths | Already managed by bids-service |
| fslview (desktop viewer) | Niivue (web viewer) | Modern WebGL2 replacement; same functionality in browser |

---

## References

- Senjem, M.L. et al. (2008). P1-288: Automated ROI analysis of 11C Pittsburgh compound B images using structural magnetic resonance imaging atlases. Alzheimer's & Dementia, 4: T302. https://doi.org/10.1016/j.jalz.2008.05.878
- Rolls, E.T. et al. (2020). Automated anatomical labelling atlas 3. NeuroImage, 206, 116189. https://doi.org/10.1016/j.neuroimage.2019.116189
- Niivue — WebGL2 medical image viewer. https://github.com/niivue/niivue
- FreeBrowse — Browser-based FreeView built on Niivue. https://github.com/freesurfer/freebrowse
- Niivue React component. https://github.com/niivue/niivue-react
- BrainBrowser — Web-based neurological data visualization. https://github.com/aces/brainbrowser
- NIfTI-1 format specification. https://nifti.nimh.nih.gov/nifti-1/
- Mayo Clinic Adult Lifespan Template (MCALT). https://www.nitrc.org/projects/mcalt/
