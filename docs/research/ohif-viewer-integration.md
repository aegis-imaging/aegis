# OHIF Viewer Integration Research

**Status:** Research only — no implementation yet
**Researched:** 2026-03-04
**Purpose:** Evaluate adding OHIF as an optional second viewer alongside DWV in AEGIS

---

## 1. OHIF Overview

**Full Name:** Open Health Imaging Foundation Viewer
**License:** MIT
**Website:** https://ohif.org
**GitHub:** https://github.com/OHIF/Viewers
**Docs:** https://docs.ohif.org
**Live Demo:** https://viewer.ohif.org
**Stable Version:** 3.11 / Current Beta: 3.13.x
**Governance:** Program of Massachusetts General Hospital (non-profit); development led by Radical Imaging; NIH NCI ITCR funded since 2015; Chan Zuckerberg Initiative grant 2020–2023.

**What it is:**
A web-based, extensible, open-source medical imaging viewer. Runs entirely in the browser — zero install at reading sites. Supports radiology-grade viewing with GPU-accelerated rendering via Cornerstone3D.

**Core Capabilities:**
- Multi-modality: MRI, CT, PET/CT fusion, ultrasound, 4D volumes, slide microscopy, video DICOM
- Annotations and measurement tracking (longitudinal)
- 3D volume rendering and MPR (multiplanar reconstruction)
- Segmentation overlay display
- Structured reports (DICOM SR)
- DICOM PDF and video
- RT Dose and RT Structure Set visualization
- Hanging protocols for automated layout
- Internationalization (i18n)
- Keyboard shortcuts

**Real-world users:** The Cancer Imaging Archive (TCIA), NCI Imaging Data Commons, XNAT, ProstateCancer.ai, University Children's Hospital Zurich, MIDRC, Pixilib (nuclear medicine), Gradient Health (mammography AI).

---

## 2. OHIF vs DWV Comparison

| Feature | DWV (current) | OHIF |
|---------|---------------|------|
| License | Apache 2.0 | MIT |
| Rendering engine | Cornerstone (2D) | Cornerstone3D (2D + 3D GPU) |
| Volume/3D MPR | No | Yes |
| PET/CT fusion | No | Yes |
| Segmentation overlay | No | Yes |
| Measurement tracking | Limited | Full (longitudinal) |
| DICOM SR | No | Yes |
| RT Structures | No | Yes |
| Hanging protocols | No | Yes |
| Plugin/extension system | Minimal | Comprehensive |
| Docker image size | ~50 MB (nginx + static) | ~200–500 MB (more JS) |
| Complexity | Low | High |
| Configuration | Simple env var | JavaScript config file |
| Yoked scrolling | Custom postMessage impl | Native (linked viewports) |
| Study list UI | No (viewer only) | Yes (built-in worklist) |
| Authentication integration | Via CSP/nginx | OAuth2 Proxy / Keycloak, or bypass |

**Summary:** DWV is simple, fast, and sufficient for basic DICOM viewing. OHIF is significantly more powerful and feature-rich, especially for research workflows, advanced analytics review (PET/CT, segmentations, MPR), and institutions that need a full radiological workflow UI.

---

## 3. OHIF Architecture Deep Dive

### 3.1 Tech Stack

- **Framework:** React 18 + TypeScript
- **Styling:** Tailwind CSS v3.3.2
- **Rendering:** Cornerstone3D (GPU-accelerated via WebGL)
- **Build:** Yarn workspaces monorepo, Webpack
- **Package structure:** `platform/app` (main app), `platform/core` (services), `platform/ui` (component library), `extensions/` (feature modules), `modes/` (workflow configs)

### 3.2 Extension System

OHIF v3 redesigned around an extension + mode architecture:

**Extensions** are reusable packages of functionality:
- `@ohif/extension-default` — layout, study/series browser, DICOMweb datasource
- `@ohif/extension-cornerstone` — 2D/3D rendering viewports
- `@ohif/extension-cornerstone-dicom-sr` — Structured Report visualization
- `@ohif/extension-measurement-tracking` — longitudinal measurement panels
- `@ohif/extension-dicom-pdf` — PDF rendering
- `@ohif/extension-dicom-video` — DICOM video support

Each extension provides typed **modules**: layout, dataMapping, viewports, panels, commands, toolbar, contexts, hangingProtocols, utilities.

### 3.3 Modes (Workflow Configurations)

Modes are task-specific viewer configurations that consume extensions:

- **Basic Viewer** (`/basic`) — standard 2D viewer, no tracking
- **Longitudinal / Measurement Tracking** (`/`) — adds measurement panels, persistence
- **TMTV** (`/tmtv`) — Total Metabolic Tumor Volume, for PET/CT
- **Segmentation** — for displaying/editing DICOM SEG objects
- Custom modes can be built for AEGIS-specific workflows (e.g. a defacing review mode)

Modes are tied to URL routes. A URL like `viewer?StudyInstanceUIDs=X` can include `&mode=basic` to select the mode.

### 3.4 Configuration System

OHIF is configured via `app-config.js` (JavaScript, not JSON — allows functions):

```javascript
window.config = {
  routerBasename: '/ohif',
  extensions: [],
  modes: [],
  showStudyList: true,          // show worklist page
  showPatientInfo: 'visible',   // 'visible' | 'visibleCollapsed' | 'disabled'
  defaultDataSourceName: 'dicomweb',
  dataSources: [{
    namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
    sourceName: 'dicomweb',
    configuration: {
      friendlyName: 'AEGIS DICOMweb',
      name: 'aegis',
      wadoUriRoot:  'https://api.aegisimaging.ai/dicomweb',
      qidoRoot:     'https://api.aegisimaging.ai/dicomweb',
      wadoRoot:     'https://api.aegisimaging.ai/dicomweb',
      imageRendering: 'wadors',
      thumbnailRendering: 'wadors',
      enableStudyLazyLoad: true,
      supportsFuzzyMatching: false,
      supportsWildcard: true,
      dicomUploadEnabled: false,
    }
  }]
}
```

**Environment variables** (build-time):

| Variable | Purpose | Default |
|----------|---------|---------|
| `APP_CONFIG` | Path to config JS file | `config/default.js` |
| `PUBLIC_URL` | App serving base path | `/` |
| `OHIF_PORT` | Dev server port | `3000` |

**Runtime config override:** The `APP_CONFIG` file can be volume-mounted into the Docker container at `/usr/share/nginx/html/app-config.js` without rebuilding. This is the preferred deployment pattern.

### 3.5 URL Parameters

OHIF supports deep-linking studies directly via URL:

```
/viewer?StudyInstanceUIDs=1.2.3.4.5
/viewer?StudyInstanceUIDs=1.2.3&SeriesInstanceUIDs=1.2.3.4
/viewer?StudyInstanceUIDs=1.2.3&initialSeriesInstanceUID=1.2.3.5
/viewer?StudyInstanceUIDs=1.2.3&hangingProtocolId=@ohif/mnGrid
/viewer?StudyInstanceUIDs=1.2.3&mode=basic
```

Multiple studies: `?StudyInstanceUIDs=1.2.3,4.5.6` or `?StudyInstanceUIDs=1.2.3&StudyInstanceUIDs=4.5.6`

**Note:** OHIF uses `StudyInstanceUIDs` (plural). DWV uses `?studyUID=` (singular). These are different and require separate URL construction in the admin dashboard.

### 3.6 Deployment Options

**Option A: Pre-built Docker image (simplest)**
```bash
docker run -d -p 3000:80 \
  -v /path/to/app-config.js:/usr/share/nginx/html/app-config.js \
  ohif/app:v3.11
```
Image available on Docker Hub: `ohif/app`. Config is injected via volume mount.

**Option B: Custom build**
Clone OHIF repo, modify config, `yarn install && yarn build` → serve static `dist/` via nginx.

**Option C: iframe embedding**
OHIF supports embedding via `<iframe src="https://ohif.aegisimaging.ai/viewer?StudyInstanceUIDs=X" />`. `PUBLIC_URL` and `routerBasename` must be set correctly at build time.

**Option D: Static CDN hosting**
The build output is static HTML/CSS/JS; can be hosted on GCS, S3, or Azure Blob with CDN.

### 3.7 Authentication Integration

OHIF itself has no built-in auth — it's a pure frontend app. Auth is handled at the infrastructure layer:

**Recommended pattern (from OHIF docs):**
- Nginx reverse proxy → OAuth2 Proxy → Keycloak (OIDC)
- Role-based: Viewer (OHIF only), PacsAdmin (OHIF + PACS admin), Admin (Keycloak management)

**For AEGIS:**
- Embed OHIF as an iframe inside the IAP-authenticated admin dashboard — inherits the IAP session, no separate auth needed for the iframe
- Or deploy OHIF Cloud Run service itself behind IAP (same IAP members as admin dashboard)

**DICOMweb backend auth:** OHIF's nginx sidecar proxies `/dicomweb/` calls to the AEGIS API backend (same pattern as DWV), so auth is handled at the reverse proxy layer — not in the browser. The AEGIS DICOMweb routes are public (no auth required per `api/main.go`), consistent with the current DWV setup.

---

## 4. AEGIS DICOMweb Compatibility Analysis

AEGIS already exposes a DICOMweb proxy at `/dicomweb/` and `/dicomweb-raw/`. The question is whether OHIF's DICOMweb client is compatible with AEGIS's minimal implementation.

### What AEGIS Provides Today

| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /dicomweb/studies` | ✅ | Returns QIDO-RS JSON |
| `GET /dicomweb/studies/{uid}/series` | ✅ | Returns single fake series |
| `GET /dicomweb/studies/{uid}/series/{s}/instances` | ✅ | Returns instance list |
| `GET /dicomweb/studies/{uid}/series/{s}/instances/{sop}` | ✅ | Streams DICOM bytes (multipart/related) |
| `GET /dicomweb/studies/{uid}/series/{s}/instances/{sop}/metadata` | ✅ | Returns DICOMweb JSON metadata |
| `POST /api/stow/studies` (STOW-RS) | ✅ | Not used by OHIF viewer |
| `GET /dicomweb/studies/{uid}/series/{s}/instances/{sop}/rendered` | ❌ | Not implemented |
| `GET /dicomweb/studies/{uid}/thumbnail` | ❌ | Not implemented |
| `GET /dicomweb/studies/{uid}/series/{s}/thumbnail` | ❌ | Not implemented |

### Gap Analysis

**Critical gaps for OHIF:**

1. **Thumbnail rendering** (`/rendered` endpoint): OHIF uses thumbnail images in the series browser. Without this, OHIF can't show series thumbnails. **Workaround:** set `thumbnailRendering: 'wadors'` — OHIF fetches the first full WADO-RS frame instead. Works but is slower.

2. **Single fake series per study**: AEGIS returns ONE fake series per study regardless of actual DICOM series. OHIF will show all files in one flat series panel. This is a limitation shared by DWV currently. Needs addressing for multi-series studies.

3. **SOP Class UIDs**: AEGIS synthesizes SOP Class UIDs from modality. OHIF's mode selector uses SOP Class UIDs to determine applicable modes. Synthetic mapping works for basic viewing but may fail for advanced modes (e.g. TMTV requires PET SOP class).

4. **QIDO-RS metadata completeness**: AEGIS's `studyQIDO()` omits PatientName/PatientID/StudyDate (de-identification). OHIF's study list will show empty patient info — acceptable for AEGIS's context.

5. **BulkDataURI**: Not implemented. OHIF falls back to WADO-RS for pixel data, which AEGIS supports. No issue.

6. **WADO-URI** (legacy): Not implemented. OHIF uses WADO-RS by default when `imageRendering: 'wadors'`. No issue.

### What Works Out of the Box

- OHIF can open studies via `?StudyInstanceUIDs=X` — AEGIS's QIDO-RS returns enough metadata to bootstrap
- WADO-RS instance retrieval works — AEGIS streams `application/dicom` multipart responses
- WADO-RS metadata works — AEGIS returns DICOMweb JSON for all tags
- The Go API's synthetic SOP Instance UID scheme (`{studyUID}.1.{index}`) should be parseable by OHIF

---

## 5. Integration Architecture Options

Three viable architectures for adding OHIF to AEGIS:

### Option A: OHIF as a New Cloud Run Sidecar ✅ Recommended

Mirror the DWV pattern exactly:

```
admin-dashboard (Cloud Run)
    ├── <iframe src="https://dwv-xxx.run.app/viewer?studyUID=X&store=clean" />  ← DWV (existing)
    └── <iframe src="https://ohif-xxx.run.app/viewer?StudyInstanceUIDs=X" />   ← OHIF (new)
```

**Architecture:**
- New Cloud Run service `aegis-ohif` serving OHIF static app via nginx
- `app-config.js` volume-injected (or baked at build time) pointing to AEGIS DICOMweb proxy
- nginx in OHIF container proxies `/dicomweb/` → `https://api.aegisimaging.ai/dicomweb/` (same pattern as DWV)
- Admin dashboard gets new `VITE_OHIF_BASE_URL` build arg
- Viewer toggle in admin dashboard: "DWV | OHIF" button (stored in `localStorage`)

**Pros:** Consistent with existing DWV pattern; isolated service; independent deployments
**Cons:** Additional Cloud Run service (cost ~$0/month at low traffic due to scale-to-zero); another image to build in CI

### Option B: OHIF Embedded in Admin Dashboard

Build OHIF's `dist/` output into the admin dashboard Docker image, served under `/ohif/` by admin dashboard's nginx.

**Pros:** No extra Cloud Run service
**Cons:** Admin dashboard build time increases significantly (OHIF build adds ~5–10 min + ~500 MB); mixed concerns; harder to update OHIF independently

### Option C: Shared Nginx Gateway

Add a new nginx gateway service routing `/dwv/*` and `/ohif/*` to their respective containers.

**Pros:** Centralized auth and CORS handling
**Cons:** New gateway service to maintain; not needed for AEGIS's current architecture

---

## 6. CORS and Authentication Considerations

### CORS

AEGIS's CORS middleware (`api/middleware/cors.go`) maintains an `ALLOWED_ORIGINS` list. The OHIF Cloud Run URL needs to be added:

```
# Current
http://localhost:3000,...,https://aegisimaging.ai,https://www.aegisimaging.ai

# Additions needed
https://ohif-<CLOUD_RUN_HASH>-uc.a.run.app
https://aws.ohif.aegisimaging.ai   (future AWS)
```

**Better approach (same as DWV):** OHIF's nginx sidecar proxies DICOMweb calls to the API backend server-side. The browser only communicates with the OHIF nginx origin — no cross-origin DICOMweb requests. This eliminates the CORS issue entirely for DICOMweb fetches.

### Content-Security-Policy

DWV's `nginx.conf.template` restricts iframe embedding via CSP `frame-ancestors`. OHIF's container needs the equivalent so it can be iframed from `admin.aegisimaging.ai`:

```nginx
add_header Content-Security-Policy "frame-ancestors https://admin.aegisimaging.ai https://aws.admin.aegisimaging.ai https://*-<CLOUD_RUN_HASH>-uc.a.run.app http://localhost:3001 'self'" always;
```

DWV's CSP also needs updating to include the OHIF origin (in case OHIF ever iframes DWV — unlikely but defensive).

### Raw vs Clean Store

DWV maps `?store=raw` to `/dicomweb-raw/` in nginx. OHIF handles this differently using named data sources:

```javascript
// OHIF app-config.js — two data sources
dataSources: [
  { sourceName: 'aegis-clean', configuration: { wadoRoot: '/dicomweb', ... } },
  { sourceName: 'aegis-raw',   configuration: { wadoRoot: '/dicomweb-raw', ... } },
]
```

To open raw store: `?StudyInstanceUIDs=X&dataSources=aegis-raw`

OHIF's nginx must proxy both `/dicomweb/` and `/dicomweb-raw/` to the API backend (same as DWV).

---

## 7. OHIF Docker Implementation Design

Following the DWV pattern at `dwv/`:

### `ohif/Dockerfile`

**Approach 1 (Recommended): Use official `ohif/app` Docker image**

The `ohif/app` image (Docker Hub) is a pre-built nginx-served static app. Config is injected at runtime via volume or env var — no Node.js build step needed.

```dockerfile
FROM ohif/app:v3.11

# Override nginx to add our proxy routes and CSP
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY docker-entrypoint.sh /docker-entrypoint.sh

# app-config.js is injected at runtime:
# docker run -v /path/to/app-config.js:/usr/share/nginx/html/app-config.js ohif/app

ENTRYPOINT ["/docker-entrypoint.sh"]
```

**Approach 2: Build from OHIF source (more control, custom extensions)**

```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
RUN git clone --depth 1 --branch release/3.11 https://github.com/OHIF/Viewers.git .
COPY app-config.js platform/app/public/config/default.js
RUN yarn config set workspaces-experimental true && \
    yarn install --frozen-lockfile && \
    yarn run build

FROM nginx:1.27-alpine
COPY --from=builder /app/platform/app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY docker-entrypoint.sh /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]
```

Approach 2 enables custom OHIF extensions (e.g. AEGIS audit logging, custom yoked scrolling). Approach 1 is faster to ship.

### `ohif/nginx.conf.template`

```nginx
server {
    listen ${PORT:-80};

    # Allow embedding from admin dashboards
    add_header Content-Security-Policy "frame-ancestors https://admin.aegisimaging.ai https://aws.admin.aegisimaging.ai https://aegisimaging.ai https://*-<CLOUD_RUN_HASH>-uc.a.run.app http://localhost:3001 http://localhost:3000 'self'" always;
    add_header X-Frame-Options "ALLOWALL" always;

    # Proxy DICOMweb calls to AEGIS API (avoids browser-level CORS)
    location /dicomweb/ {
        set $api_upstream ${API_URL};
        proxy_pass $api_upstream;
        proxy_http_version 1.1;
        proxy_set_header Host $proxy_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_ssl_server_name on;
        proxy_read_timeout 120s;
    }

    location /dicomweb-raw/ {
        set $api_upstream ${API_URL};
        proxy_pass $api_upstream;
        proxy_http_version 1.1;
        proxy_set_header Host $proxy_host;
        proxy_ssl_server_name on;
        proxy_read_timeout 120s;
    }

    location = /health {
        return 200 '{"status":"healthy"}';
        add_header Content-Type application/json;
    }

    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }
}
```

### `ohif/app-config.js`

```javascript
window.config = {
  routerBasename: '/',
  extensions: [],
  modes: [],
  showStudyList: false,           // AEGIS controls study navigation
  showLoadingIndicator: true,
  showPatientInfo: 'disabled',    // All studies are de-identified
  defaultDataSourceName: 'aegis-clean',

  dataSources: [
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'aegis-clean',
      configuration: {
        friendlyName: 'AEGIS (De-identified)',
        name: 'aegis-clean',
        wadoUriRoot:  '/dicomweb',
        qidoRoot:     '/dicomweb',
        wadoRoot:     '/dicomweb',
        imageRendering: 'wadors',
        thumbnailRendering: 'wadors',   // No /rendered endpoint in AEGIS
        enableStudyLazyLoad: true,
        supportsFuzzyMatching: false,
        supportsWildcard: true,
        dicomUploadEnabled: false,
        singlepart: 'bulkdata,video,pdf',
      }
    },
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'aegis-raw',
      configuration: {
        friendlyName: 'AEGIS (Raw / Pre-defacing)',
        name: 'aegis-raw',
        wadoUriRoot:  '/dicomweb-raw',
        qidoRoot:     '/dicomweb',       // QIDO always uses clean index
        wadoRoot:     '/dicomweb-raw',
        imageRendering: 'wadors',
        thumbnailRendering: 'wadors',
        enableStudyLazyLoad: true,
        supportsFuzzyMatching: false,
        supportsWildcard: true,
        dicomUploadEnabled: false,
      }
    }
  ]
}
```

### `ohif/docker-entrypoint.sh`

Same pattern as `dwv/docker-entrypoint.sh`:

```bash
#!/bin/sh
# Substitute API_URL into nginx config (avoid mangling nginx's own $variables)
envsubst '${API_URL} ${PORT}' < /etc/nginx/templates/default.conf.template \
  > /etc/nginx/conf.d/default.conf
exec nginx -g 'daemon off;'
```

---

## 8. Admin Dashboard Integration

### New Build Arg

`frontend/admin-dashboard/Dockerfile` — add alongside `VITE_DWV_BASE_URL`:

```dockerfile
ARG VITE_OHIF_BASE_URL=http://localhost:3006
ENV VITE_OHIF_BASE_URL=${VITE_OHIF_BASE_URL}
```

### Viewer Toggle in `ViewerPanel.tsx`

Current file: `frontend/admin-dashboard/src/components/ViewerPanel.tsx`

```typescript
const DWV_BASE  = import.meta.env.VITE_DWV_BASE_URL  || 'http://localhost:3005'
const OHIF_BASE = import.meta.env.VITE_OHIF_BASE_URL || 'http://localhost:3006'

type ViewerEngine = 'dwv' | 'ohif'
// Persisted in localStorage so users remember their preference
const [viewerEngine, setViewerEngine] = useLocalStorage<ViewerEngine>('aegis_viewer_engine', 'dwv')

// URL construction — note the different query param names
const dwvUrl  = `${DWV_BASE}/viewer?studyUID=${studyUID}&store=${store}`
const ohifUrl = `${OHIF_BASE}/viewer?StudyInstanceUIDs=${studyUID}` +
                (store === 'raw' ? `&dataSources=aegis-raw` : '')

const viewerUrl = viewerEngine === 'ohif' ? ohifUrl : dwvUrl
```

A small "DWV | OHIF" toggle button (teal active state per colorblind palette) would sit in the viewer panel header alongside the "Open in new tab ↗" link.

### URL Parameter Mapping

| Purpose | DWV URL | OHIF URL |
|---------|---------|---------|
| View study (clean) | `?studyUID=X` | `?StudyInstanceUIDs=X` |
| View study (raw) | `?studyUID=X&store=raw` | `?StudyInstanceUIDs=X&dataSources=aegis-raw` |
| Open in new tab | Direct URL | Direct URL |
| Specific series | Not supported | `?SeriesInstanceUIDs=X` |
| Specific image | Not supported | `?initialSopInstanceUID=X` |
| 3D volume mode | Not supported | `?mode=basic` (or custom mode) |
| PET/CT fusion | Not supported | `?mode=tmtv` |

### Defacing Review Panel (Side-by-Side)

The defacing review panel (`App.tsx`) uses DWV-specific yoked scrolling via postMessage (`dwv-position` / `dwv-goto` protocol). This is a custom AEGIS feature implemented in `dwv/src/main.js`.

**OHIF consideration:** OHIF has linked viewport scrolling natively within a single OHIF instance (multiple viewports in one tab), but cross-iframe synchronization requires a custom OHIF extension that emits and receives postMessage events from the parent frame. This is not built into OHIF today.

**Recommendation:** Keep DWV as the default for the defacing review panel. OHIF can be offered for standard single-panel viewing. Yoked OHIF defacing review is a future enhancement requiring a custom extension.

---

## 9. DICOMweb Enhancements Needed

To get the best OHIF experience, the AEGIS DICOMweb proxy should be enhanced:

### Enhancement 1: Multi-Series Support (Medium priority)

**Problem:** AEGIS currently returns ONE fake series per study. OHIF shows all files in a single flat series panel.

**Fix:** `GET /dicomweb/studies/{uid}/series` should return actual series from the `study_series` table when rows exist, falling back to the synthetic single series when they don't.

**File:** `api/handler/dicomweb.go` — `dicomwebSeries()` function

**Impact:** Multi-series studies (common in MRI with T1, T2, FLAIR, DWI sequences) would display correctly in both OHIF and DWV.

### Enhancement 2: Thumbnail Rendered Endpoint (Low priority)

**Problem:** OHIF's series browser shows thumbnails. AEGIS has no `/rendered` endpoint.

**Workaround (no code change):** `thumbnailRendering: 'wadors'` in OHIF config — OHIF fetches first full WADO-RS frame for thumbnails instead. Slower but functional.

**Future fix:** Add `GET /dicomweb/studies/{uid}/series/{s}/instances/{sop}/rendered` returning a JPEG thumbnail using `suyashkumar/dicom` to extract the first pixel frame.

### Enhancement 3: Study-Level QIDO Metadata (Low priority)

**Problem:** AEGIS's `studyQIDO()` omits StudyDate, Modality, and other fields for de-identification.

**Note:** With `showStudyList: false`, the OHIF worklist is disabled and this only matters if enabled in the future.

---

## 10. Cloud Deployment Design (GCP)

Following the exact pattern used for the DWV service at `dwv/`:

### Cloud Run Service: `aegis-ohif`

- **Image:** `us-central1-docker.pkg.dev/<GCP_PROJECT_ID>/aegis-services/ohif:latest`
- **Port:** 80
- **Env vars:** `API_URL=https://api.aegisimaging.ai` (set at deploy time)
- **Health check:** `GET /health`
- **Scale-to-zero:** yes (same as all other services)
- **IAP:** same IAP policy as admin dashboard (or unauthenticated if embedded only within IAP-protected iframe)

### CI/CD (`cloudbuild.yaml` additions)

**Phase 1 additions (parallel builds):**
```yaml
- id: build-push-ohif
  name: 'gcr.io/cloud-builders/docker'
  args:
    - build
    - -t
    - ${_REPO}/ohif:${SHORT_SHA}
    - -t
    - ${_REPO}/ohif:latest
    - --platform
    - linux/amd64
    - ohif
```

**Phase 2 additions (deploy after image push):**
```yaml
- id: deploy-ohif
  waitFor: ['build-push-ohif']
  name: 'gcr.io/google.com/cloudsdktool/cloud-sdk:slim'
  entrypoint: gcloud
  args:
    - run
    - deploy
    - aegis-ohif
    - --image
    - ${_REPO}/ohif:${SHORT_SHA}
    - --region
    - us-central1
    - --project
    - ${PROJECT_ID}
    - --set-env-vars
    - API_URL=https://api.aegisimaging.ai
```

**Admin dashboard build args update:**
```yaml
- --build-arg
- VITE_DWV_BASE_URL=${_DWV_URL}
- --build-arg
- VITE_OHIF_BASE_URL=${_OHIF_URL}
```

**New substitution variable:**
```yaml
substitutions:
  _OHIF_URL: https://ohif-<CLOUD_RUN_HASH>-uc.a.run.app
```

---

## 11. Docker Compose (Local Dev)

Add to `docker-compose.yml`:

```yaml
ohif:
  build:
    context: ./ohif
  ports:
    - "3006:80"
  environment:
    API_URL: "http://api:8080"
  depends_on:
    api:
      condition: service_healthy
  healthcheck:
    test: ["CMD-SHELL", "wget -qO- http://localhost:80/health || exit 1"]
    interval: 30s
    timeout: 10s
    retries: 3
    start_period: 30s
```

Local dev access: `http://localhost:3006/viewer?StudyInstanceUIDs=<uid>`

---

## 12. Implementation Roadmap (When Ready to Build)

Ordered steps:

1. **Create `ohif/` directory** with `Dockerfile` (using `ohif/app` base image), `nginx.conf.template`, `app-config.js`, `docker-entrypoint.sh`
2. **Test locally** via `docker compose up ohif` — verify OHIF loads and can browse AEGIS studies
3. **Diagnose DICOMweb gaps** — check OHIF browser console for errors against AEGIS's DICOMweb proxy
4. **Enhance DICOMweb proxy** in `api/handler/dicomweb.go` — multi-series support from `study_series` table
5. **Update `ViewerPanel.tsx`** — add viewer engine toggle (DWV default, OHIF option)
6. **Update admin dashboard Dockerfile** — add `VITE_OHIF_BASE_URL` build arg
7. **Update `docker-compose.yml`** — add `ohif` service
8. **Update CORS config** in `api/config/config.go` `ALLOWED_ORIGINS` default — add OHIF Cloud Run URL
9. **Update DWV CSP** in `dwv/nginx.conf.template` — add OHIF Cloud Run domain to `frame-ancestors`
10. **Update `cloudbuild.yaml`** — add OHIF build + deploy steps + `_OHIF_URL` substitution
11. **Update Terraform** `terraform/infra/` — add Cloud Run service for OHIF (or manage via `gcloud run deploy` in CI/CD only, same as current DWV)
12. **Update `CLAUDE.md`** and memory — document OHIF service URL, build args, integration

**Future enhancements (not MVP):**
- Custom OHIF extension for AEGIS audit logging (viewer open/close events)
- Custom OHIF extension for cross-iframe yoked scrolling (defacing review parity with DWV)
- Multi-series DICOMweb support in Go API (`study_series` table integration)
- OHIF worklist integration (`showStudyList: true`, scoped to a project via QIDO filter)
- OHIF segmentation display for analytics service DICOM SEG outputs
- PET/CT fusion mode (`/tmtv`) for AEGIS PET studies
- OHIF-based protocol compliance viewer (overlay acquisition parameters on images)

---

## 13. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| OHIF's DICOMweb client rejects AEGIS's synthetic SOP Instance UIDs | Medium | High | Test early in local dev; adjust UID scheme if needed |
| OHIF Docker image size significantly increases CI build time | High | Low | Use `ohif/app` pre-built image — no Node.js build step |
| Single fake series causes poor OHIF series browser UX | High | Medium | Implement multi-series DICOMweb support (step 4) before dashboard rollout |
| OHIF iframe blocked by CSP in admin dashboard | Medium | High | Configure `frame-ancestors` in OHIF nginx; test locally first |
| OHIF has worse performance than DWV for simple 2D viewing | High | Low | DWV remains the default; OHIF is opt-in |
| Yoked scrolling for defacing review not available in OHIF | High | Medium | Keep DWV for defacing review panel; OHIF for standard viewing only |
| Cloud Run cold start latency for OHIF | Medium | Low | OHIF is large (~2 MB JS) — first load slow; subsequent loads cached in browser |
| IAP token forwarding from admin dashboard iframe to OHIF | Medium | Medium | Both services under same IAP; browser handles cookie forwarding for same IAP config |
| OHIF's Cornerstone3D requires WebGL — blocked on some corporate browsers | Low | Medium | DWV fallback always available |
| OHIF license (MIT) — compatible with AEGIS | Confirmed | None | MIT is permissive; no concerns |

---

## 14. Key File Paths Reference

### Existing Files (DWV Pattern to Mirror)

| File | Role |
|------|------|
| `dwv/Dockerfile` | Multi-stage build reference |
| `dwv/nginx.conf.template` | Nginx config with proxy + CSP |
| `dwv/docker-entrypoint.sh` | Runtime env var substitution |
| `dwv/src/main.js` | DWV app initialization |
| `api/handler/dicomweb.go` | DICOMweb proxy handler |
| `api/main.go` | Route registration (`/dicomweb/` + `/dicomweb-raw/`) |
| `api/middleware/cors.go` | CORS allowlist middleware |
| `api/config/config.go` | `ALLOWED_ORIGINS` default value |
| `frontend/admin-dashboard/src/components/ViewerPanel.tsx` | Iframe embedding |
| `frontend/admin-dashboard/Dockerfile` | Build args pattern |
| `docker-compose.yml` | Service definition pattern |
| `cloudbuild.yaml` | Build + deploy pipeline |

### New Files to Create (When Implementing)

| File | Purpose |
|------|---------|
| `ohif/Dockerfile` | OHIF container build |
| `ohif/nginx.conf.template` | Nginx with DICOMweb proxy + CSP |
| `ohif/app-config.js` | OHIF configuration (two data sources: aegis-clean, aegis-raw) |
| `ohif/docker-entrypoint.sh` | Runtime API_URL substitution |

### Files to Modify (When Implementing)

| File | Change |
|------|--------|
| `frontend/admin-dashboard/src/components/ViewerPanel.tsx` | Add OHIF iframe + DWV/OHIF toggle |
| `frontend/admin-dashboard/Dockerfile` | Add `VITE_OHIF_BASE_URL` build arg |
| `docker-compose.yml` | Add `ohif` service on port 3006 |
| `api/config/config.go` | Add OHIF Cloud Run URL to `ALLOWED_ORIGINS` default |
| `dwv/nginx.conf.template` | Add OHIF origin to `frame-ancestors` CSP |
| `cloudbuild.yaml` | Add OHIF build/deploy steps + `_OHIF_URL` substitution |
| `api/handler/dicomweb.go` | Multi-series support from `study_series` table |
| `CLAUDE.md` | Document OHIF service, build args, URL |

---

## 15. Sources

- OHIF website: https://ohif.org
- OHIF documentation: https://docs.ohif.org
- OHIF GitHub: https://github.com/OHIF/Viewers
- OHIF Docker Hub: https://hub.docker.com/r/ohif/app
- OHIF Showcase: https://ohif.org/showcase
- OHIF Configuration docs: https://docs.ohif.org/configuration/configurationFiles
- OHIF DICOMweb data source docs: https://docs.ohif.org/configuration/dataSources/dicom-web
- OHIF Deployment docs: https://docs.ohif.org/deployment/
- OHIF Extensions docs: https://docs.ohif.org/platform/extensions
- OHIF Modes docs: https://docs.ohif.org/platform/modes
- OHIF URL params docs: https://docs.ohif.org/configuration/url
- OHIF User Account Control docs: https://docs.ohif.org/deployment/user-account-control
- AEGIS codebase: `dwv/`, `api/handler/dicomweb.go`, `api/middleware/cors.go`, `frontend/admin-dashboard/src/components/ViewerPanel.tsx`
