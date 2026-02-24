# AEGIS — Architecture Plan
*Anonymization & Exchange Gateway for Imaging Studies*

**Matthew L. Senjem, M.S.** | February 23, 2026

### Why "AEGIS"?
The name **AEGIS** serves double duty. As an acronym, it describes exactly what the system does: an **A**nonymization & **E**xchange **G**ateway for **I**maging **S**tudies. The word itself comes from Greek mythology — the aegis was the shield of Zeus and Athena, a symbol of protection. This captures the platform's core mission: shielding patient identity while enabling the free flow of medical imaging data for research and clinical care.

## Context

Medical imaging studies across **all DICOM modalities** need to be shared between hospitals, universities, and research institutions. Before transmission, DICOM images must be de-identified of all PHI per HIPAA Safe Harbor rules (18 identifier categories). Head imaging additionally requires **defacing** (removing facial features from 3D volumes to prevent re-identification via facial reconstruction). Sending hospitals have locked-down IT environments where installing software is difficult or impossible.

AEGIS is a multi-cloud platform (GCP, AWS, or Azure) with **two ingress paths**:
1. **External-site ingress** — external institutions upload DICOM data of any modality after browser-based tag anonymization.
2. **Internal-enterprise ingress** — studies originating within the enterprise network are ingested directly via API or batch import CLI.

Both paths converge on a common automated processing pipeline (classification, PHI detection, protocol compliance, defacing, QC, and BIDS conversion), after which approved data is routed/shared to authorized downstream recipients.

### Supported Modalities
AEGIS supports **all DICOM-compliant imaging modalities**, including but not limited to:
- **MRI** — brain, spine, cardiac, musculoskeletal, abdominal, breast
- **CT** — head, chest, abdomen/pelvis, cardiac, angiography
- **PET** — FDG, amyloid, tau, PSMA, whole-body
- **PET/CT and PET/MR** — fused multimodal studies
- **Ultrasound** — echocardiography, abdominal, vascular
- **X-Ray / CR / DR** — chest, extremity, mammography (FFDM/DBT)
- **Nuclear Medicine** — SPECT, planar scintigraphy
- **Fluoroscopy** — GI, interventional
- **Secondary Capture / Structured Reports / RT** — dose reports, annotations, treatment plans

The de-identification engine operates at the DICOM tag level per PS3.15 Annex E and is **modality-agnostic** — it works identically for any DICOM data.

### MVP Focus
- **Brain MRI** (T1, T2, FLAIR, DWI, fMRI, etc.)
- **Brain PET** (FDG, Amyloid, Tau tracers)
- **Brain CT**

Brain imaging is the MVP focus because it exercises the most complex pipeline (tag de-identification + automated defacing). All other modalities work through the same upload and de-identification flow but skip the defacing step.

### Future Scope
- Non-DICOM formats (pathology whole-slide imaging, electron microscopy)
- Extended DIMSE service classes (C-FIND/C-MOVE) for advanced legacy PACS workflows

### Design Principles
- **Open source** tools and libraries wherever possible
- **Minimal vulnerability surface** — prefer compiled languages (Go) over interpreted (Python) for backend services to reduce dependency sprawl and CVE exposure
- **Client-side tag anonymization** — PHI stripped in the browser before upload
- **Server-side defacing** — facial feature removal from head imaging after upload (too compute-intensive for browser); other modalities pass through without defacing
- **Zero-install at sending sites** — pure web app, no browser extensions, no desktop software required
- **Multi-cloud** — application layer is cloud-agnostic (GCP, AWS, or Azure); storage, auth, and container orchestration abstracted behind pluggable interfaces; Terraform modules provided for GCP and AWS

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  SENDING SITE (Hospital Browser)                                │
│                                                                 │
│  React Upload Portal                                            │
│  ├── DICOM parsing (dcmjs / dicomParser)                        │
│  ├── Tag-level de-identification (DICOM PS3.15 Annex E)         │
│  ├── Per-project anonymization profiles (retained tags)         │
│  ├── Validation & anonymization preview (before/after diff)     │
│  └── Encrypted upload (TLS 1.2+ to signed URL or direct PUT)   │
│                                                                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTPS (TLS 1.2+)
                           │ Only tag-de-identified data
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  CLOUD / ON-PREM (GCP, AWS, Azure, or Docker Compose)          │
│                                                                 │
│  ┌──────────────────────────────────────────────────────┐       │
│  │  Go API Backend (Cloud Run / ECS Fargate / Docker)    │       │
│  │  ├── Upload orchestration (signed URLs, validation)   │       │
│  │  ├── Study/project/institution management             │       │
│  │  ├── Multi-provider auth (IAP, Azure AD, ALB+Cognito)│       │
│  │  ├── Routing rules engine (priority-ordered)          │       │
│  │  ├── Automated pipeline orchestrator                  │       │
│  │  ├── DICOMweb proxy (QIDO-RS + WADO-RS)             │       │
│  │  ├── Export (zip download + STOW-RS forwarding)       │       │
│  │  └── Batch import CLI                                 │       │
│  └──────┬───────────────────────────────────────────────┘       │
│         │                                                        │
│  ┌──────▼────────────────────────────────────────────┐          │
│  │  6 Python Processing Services (FastAPI, Cloud Run) │          │
│  │  ├── Defacing (DeepDefacer / mri_deface)          │          │
│  │  ├── PHI Detection (Tesseract / Vision / Textract)│          │
│  │  ├── QC Automation (pydicom + numpy)              │          │
│  │  ├── NIfTI/BIDS Conversion (dcm2niix)             │          │
│  │  ├── Metadata Classification (heuristic + cloud)  │          │
│  │  └── Protocol Compliance (parameter validation)    │          │
│  └───────────────────────────────────────────────────┘          │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  DIMSE Receiver (Compute Engine VM — aegis-prod-dimse)  │    │
│  │  ├── pynetdicom C-STORE SCP on TCP port 11112           │    │
│  │  ├── Static IP 35.232.172.221 (us-central1-a)           │    │
│  │  ├── GCS staging bucket mounted via gcsfuse             │    │
│  │  └── Calls POST /api/ingest + durable retry queue       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌──────────────┐  ┌────────────────────────┐                   │
│  │ File Storage  │  │ PostgreSQL 15           │                   │
│  │ (local/S3/GCS)│  │ ├── Users, RBAC         │                   │
│  │ ├── raw/      │  │ ├── Projects, routing   │                   │
│  │ └── clean/    │  │ ├── Upload sessions     │                   │
│  └──────────────┘  │ └── Audit trail          │                   │
│                     └────────────────────────┘                   │
│                                                                  │
│  ┌──────────────────────────────────────────────────┐           │
│  │  React Frontends (4 apps)                         │           │
│  │  ├── Admin Dashboard (OHIF Viewer, study mgmt)    │           │
│  │  ├── Upload Portal (public-facing, anonymization) │           │
│  │  ├── Export Portal (token-authenticated download)  │           │
│  │  └── Landing Page (aegisimaging.ai)               │           │
│  └──────────────────────────────────────────────────┘           │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐                             │
│  │  OHIF Viewer  │  │  Email (SMTP) │                             │
│  │  (DICOMweb)   │  │  Dev: Mailpit │                             │
│  └──────────────┘  └──────────────┘                             │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## Repository Structure (Monorepo)

Currently a single monorepo. Planned to split into 5 repos (`aegis-terraform-prj`, `aegis-terraform-infra`, `aegis-api`, `aegis-frontend`, `aegis-client`) once interfaces stabilize.

```
aegis/
├── terraform/
│   ├── project/                # GCP project bootstrap (IAM, KMS, VPC-SC)
│   ├── infra/                  # GCP infrastructure (Cloud Run, Healthcare API)
│   └── aws/                    # AWS infrastructure (ECS Fargate, S3, RDS, ALB)
├── api/                        # Go backend (~8k LOC)
│   ├── handler/                # HTTP handlers (studies, projects, users, routing, export, DICOMweb)
│   ├── model/                  # PostgreSQL CRUD (studies, projects, admin_users, routing, audit)
│   ├── middleware/             # Auth (IAP, Azure AD, ALB+Cognito), CORS
│   ├── routing/                # Rules engine (priority-ordered, all-matching)
│   ├── storage/                # File storage interface (local, S3, GCS)
│   ├── config/                 # Environment-based config
│   ├── email/                  # SMTP client + templates
│   ├── migrate/                # Goose-based schema migrations
│   ├── importer/               # Batch DICOM import library
│   ├── cmd/import/             # Batch import CLI binary
│   ├── digest/                 # Email digest scheduler
│   └── testutil/               # Test helpers (testcontainers, fixtures)
├── frontend/
│   ├── upload-portal/          # React — public upload + anonymization UI (:3000)
│   ├── admin-dashboard/        # React — study mgmt, OHIF viewer, 10-tab admin (:3001)
│   ├── export-portal/          # React — token-authenticated export download (:3004)
│   └── landing/                # React — marketing site for aegisimaging.ai (:3003)
├── client/                     # @aegis/client TypeScript npm package
├── defacing/                   # Python FastAPI — DeepDefacer, mri_deface, nibabel
├── phi-detection/              # Python FastAPI — Tesseract OCR burned-in text detection
├── qc-service/                 # Python FastAPI — 5 automated quality checks
├── bids-service/               # Python FastAPI — dcm2niix DICOM→NIfTI/BIDS conversion
├── classification-service/     # Python FastAPI — DICOM header heuristic classification
├── protocol-service/           # Python FastAPI — MRI parameter compliance checking
├── dimse-receiver/             # Python FastAPI — DIMSE C-STORE SCP ingest adapter (deployed on GCE VM)
└── docs/                       # Shared research and documentation
```

### Go API — Key Endpoints

- `POST /api/upload/init` — generate signed URL for upload
- `POST /api/upload/complete` — trigger ingest + automated pipeline
- `GET /api/studies` — paginated list with filters (status, modality, source, search)
- `POST /api/studies/{id}/approve|reject` — study status management
- `POST /api/studies/{uid}/trigger-deface|phi-scan|qc-check|bids-convert|classify|protocol-check|trigger-export` — manual processing triggers
- `GET /api/studies/{uid}/dicom-download|bids-download` — file downloads (zip)
- `GET/POST /api/projects|destinations|routing-rules|institutions|admin-users|anon-profiles|digest-subscriptions|protocol-templates` — admin CRUD
- `GET /api/export/{token}` — share token validation + download
- `POST /api/import/batch` — batch DICOM import from directory
- `GET /api/dicomweb/*` — QIDO-RS + WADO-RS proxy (from current DICOM store)
- `GET /api/dicomweb-raw/*` — WADO-RS from raw store (defacing review)
- `GET /api/auth/me` — current user identity
- `GET /healthz` — health check with database, storage, and sidecar status

### React Frontends (4 apps)

- **Upload Portal** — DICOM file picker, client-side PS3.15 de-identification, per-project anonymization profiles, before/after preview, per-file progress with auto-retry
- **Admin Dashboard** — 10 tabs: Studies, Audit Log, Routing, Institutions, Profiles, Notifications, Projects, Users, Protocol Templates; OHIF Viewer integration; RBAC (admin/viewer roles)
- **Export Portal** — Token-authenticated study download page for share recipients
- **Landing Page** — Marketing site for aegisimaging.ai (deployed to Cloud Run `aegis-prod-landing`)

### @aegis/client (TypeScript npm package)

Reusable library for browser-based DICOM anonymization and upload. Core modules: DICOM parser, PS3.15 de-identification engine, upload client with `onFileStart` callback and exponential-backoff retry.

---

## Technology Choices

### Backend: Go

**Rationale for Go over Python for the main API:**
- **Smaller vulnerability surface**: Single static binary, no runtime, no pip dependency tree. `distroless` or `scratch` Docker image ~10-20 MB vs ~200-500 MB for Python
- **Fewer CVEs**: Go stdlib covers HTTP, JSON, crypto, TLS. No equivalent of the constant Python/Alpine package churn
- **Cloud Run fit**: ~100ms cold starts vs ~2-5s for Python, lower memory
- **Concurrency**: Goroutines for concurrent upload handling without GIL
- **DICOM library**: [suyashkumar/dicom](https://github.com/suyashkumar/dicom) for validation/metadata and batch-import header parsing
- **GCP SDK**: First-class Go SDK (`cloud.google.com/go/healthcare`, `cloud.google.com/go/storage`)

**Python services (6 processing + 1 DIMSE ingress adapter):**
- All ML/imaging tools (DeepDefacer, Tesseract, dcm2niix, pydicom) are Python or C with Python bindings
- Isolated containers with their own dependency trees and update cycles
- Keeps Python dependency surface completely separate from the Go API
- Processing services (Cloud Run): defacing, PHI detection, QC automation, BIDS conversion, classification, protocol compliance
- Ingress adapter: DIMSE receiver (pynetdicom C-STORE SCP) — runs on **Compute Engine VM** (`aegis-prod-dimse-receiver`, static IP `35.232.172.221`) because Cloud Run cannot expose raw TCP port 11112

### Frontend: React + TypeScript
- **Key libraries**:
  - `dcmjs` (MIT) — DICOM read/write in browser
  - `dicomParser` (MIT) — robust DICOM Part 10 parsing
  - OHIF Viewer components (MIT) for admin dashboard
- Web Workers for background DICOM processing (keeps UI responsive)

### Infrastructure: Terraform (Multi-Cloud)
- **GCP**: `terraform/project/` (bootstrap) + `terraform/infra/` (Cloud Run, Healthcare API, Cloud SQL, GCS)
- **AWS**: `terraform/aws/` (VPC, ECS Fargate, RDS, S3, ALB, ECR, KMS, SNS/SQS)
- **Azure**: Planned
- **Local dev**: `docker-compose.yml` (PostgreSQL, Mailpit, OHIF, Go API, 7 Python services)
- CI: GitHub Actions (Go build+vet+test, Python syntax, TypeScript type check, Docker build); CD: Cloud Build triggers auto-deploy all services on push to `develop`

### DICOM Storage: Cloud-Neutral File Storage

The Go API uses a `storage.Storage` interface with three implementations:

| Mode | Backend | Config |
|------|---------|--------|
| `local` | Local filesystem | `LOCAL_STORAGE_DIR` (default: `./dicom`) |
| `s3` | Amazon S3 (or S3-compatible: MinIO, LocalStack) | `S3_BUCKET`, `S3_REGION`, `S3_ENDPOINT` |
| `gcs` | Google Cloud Storage | `GCS_BUCKET` |

Files organized as `dicom/{store}/{studyUID}/{index}.dcm` with two stores:
- **`raw`** — tag-de-identified but not defaced (original upload)
- **`clean`** — fully processed (after defacing, if applicable)

A built-in DICOMweb proxy (`QIDO-RS` + `WADO-RS`) serves study/series/instance metadata from PostgreSQL and streams DICOM bytes from the storage backend. No need for dcm4chee, Orthanc, or GCP Healthcare API for local development.

### Application Database: PostgreSQL 15

PostgreSQL stores all application state:
- **User accounts and RBAC** — roles (admin, viewer), institution membership
- **Projects and institutions** — multi-tenant organization of studies
- **Upload sessions** — tracking upload state, completion
- **Studies** — 28-field model with processing status fields for each sidecar
- **Routing rules and destinations** — configurable forwarding rules per project/modality
- **Protocol templates** — per-project MRI parameter compliance rules
- **Anonymization profiles** — per-project retained tag overrides
- **Export shares** — token-authenticated download links with expiry
- **App-level audit trail** — who did what, when

Hosted on Cloud SQL (GCP), RDS (AWS), or Docker postgres (local dev). CMEK encryption in production.

### AI/ML: Pluggable Backends

Each processing service supports local backends for development and optional cloud AI backends where they provide meaningful accuracy gains:

| Service | Local Backend | Cloud Backend |
|---------|--------------|---------------|
| PHI Detection | Tesseract OCR | Google Cloud Vision `text_detection`, AWS Textract `detect_document_text` |
| Classification | DICOM header heuristics | Google Cloud Vision `label_detection`, AWS Rekognition `detect_labels` |
| QC Automation | pydicom + numpy | — (local backend currently sufficient) |
| Protocol Compliance | pydicom parameter extraction | — (local backend currently sufficient) |
| Defacing | DeepDefacer / mri_deface | — (local backend currently sufficient) |
| BIDS Conversion | dcm2niix | — (local backend currently sufficient) |

### Email Notifications

- Single SMTP client using Go stdlib `net/smtp` — zero external dependencies
- Disabled by default; enabled by setting `SMTP_HOST` env var (silent no-op when unset)
- Triggers: share created → recipient; upload confirmed → uploader; study approved/rejected → uploader; weekly/monthly digest summaries
- No PHI in email bodies — only anonymized study counts, file counts, and export URLs
- Dev: Mailpit (included in Docker Compose)
- Production: any SMTP provider (institutional relay, SendGrid, SES, etc.)

### Viewing: OHIF Viewer

**Local dev**: OHIF v3 runs as a Docker container on `:3002` via `docker compose up ohif`. Configured via `ohif-config.js` (repo root) to use the Go API's DICOMweb proxy at `http://localhost:8080/dicomweb`.

**DICOMweb proxy** (`api/handler/dicomweb.go`): minimal QIDO-RS + WADO-RS implemented in Go without a DICOM library. Serves study/series/instance metadata from PostgreSQL and streams raw DICOM bytes from local storage. Uses fake deterministic UIDs (`{studyUID}.1.{fileIndex}`) that map directly to file paths (`dicom/{store}/{studyUID}/{index}.dcm`).

Admin dashboard **View button**: each study row shows an inline iframe panel (OHIF embedded in the dashboard) and an "Open in new tab ↗" link. Both modes open `http://localhost:3002/viewer?StudyInstanceUIDs={uid}`.

**Production**: OHIF served from a container behind the cloud's auth layer (IAP, ALB+Cognito, or Azure AD). The Go DICOMweb proxy continues to serve DICOM data from cloud storage (S3/GCS) — no external DICOM server required.

- Web-based, React, MIT license
- Supports all standard DICOM modalities (MRI, CT, PET, US, X-Ray, NM, etc.)

---

## Anonymization Architecture

### Two-Phase De-identification

**Phase 1 — Client-side tag anonymization (browser, before upload):**
- Tag-level de-id per DICOM PS3.15 Annex E Basic Profile
- Action codes per tag: D (dummy), Z (zero), X (remove), U (replace UID), C (clean)
- All 18 HIPAA Safe Harbor identifier categories addressed
- Key tags handled:
  - PatientName, PatientID, PatientBirthDate → zeroed/replaced
  - InstitutionName, ReferringPhysician, OperatorsName → removed
  - StudyInstanceUID, SeriesInstanceUID, SOPInstanceUID → deterministic hash (preserves longitudinal linkage)
  - StudyDate, SeriesDate → date-shifted with preserved temporal relationships
  - All private tags → removed by default (retain list for known-safe manufacturer tags needed for research, e.g., diffusion gradient tables)
- Configurable profiles: "Research (full)" vs "Institutional (partial)"
- **Anonymization preview**: user sees before/after tag diff before confirming
- Mapping table (original → anonymized IDs) stays local, never uploaded

**Phase 2 — Server-side processing (after upload):**
- **PHI detection service** (Tesseract OCR) scans pixel data for burned-in text (patient names, dates, accession numbers)
- **Defacing pipeline** (see below) removes facial features from head scans
- **QC automation** validates file integrity, slice consistency, SNR, coverage, missing slices
- **Protocol compliance** checks acquisition parameters against per-project templates
- **Classification service** fills in missing modality/body_part from DICOM headers
- **BIDS conversion** produces NIfTI files with BIDS-compliant directory structure
- Results written to `clean` DICOM store; `raw` store used as staging

### Defacing Pipeline (Server-Side — Head Imaging Only)

Defacing applies **only to head/brain imaging** (MR, PT, CT with BodyPartExamined = HEAD or BRAIN). All other modalities and anatomies bypass defacing and proceed directly to the `clean` store after tag de-identification validation.

**Pipeline:**
```
1. Routing rule with action require_defacing sets defacing_required=true
2. Pipeline orchestrator dispatches defacing service (Phase 1, parallel with PHI scan)
3. Defacing service (head imaging only):
   a. Reads DICOM files from cloud-neutral storage (local/S3/GCS)
   b. Converts to NIfTI via dcm2niix
   c. Runs defacing tool (DeepDefacer, mri_deface, or mri_reface)
   d. Injects defaced pixel data back into original DICOM (pydicom)
   e. Stores defaced DICOM to "clean" store
4. Study status: received → defacing → defaced; dicom_store switches from raw to clean
5. Admin reviews defacing quality in dashboard (OHIF side-by-side before/after)
6. Admin approves → study available for export/download
```

**Defacing tool options (ranked by deployment practicality):**

| Tool | Docker Size | Time/Volume | Quality | License | Best For |
|------|------------|-------------|---------|---------|----------|
| **mri_deface** (FreeSurfer) | ~0.5 GB | 2-10 min | Good | Free (no license needed) | MVP — smallest footprint |
| **DeepDefacer** | ~1-2 GB | ~2 min | Good | MIT | Speed-optimized alternative |
| **pydeface** | ~3-6 GB | 2-10 min | Good (83%) | MIT | If FSL already available |
| **mri_reface** | ~2-4 GB | >1 min | Best identity protection | Non-commercial only | Research-only deployments |
| **afni_refacer** | ~3-5 GB | ~30 min | Best overall (89%) | Public domain | Quality-critical cases |
| **MiDeFace** (FreeSurfer) | ~4-8 GB | ~30 min | Very good | FreeSurfer license | When accuracy is paramount |

**Current default**: **DeepDefacer** (pip-installable, ~500 MB with TensorFlow, ~1-2 min/volume, no external binary dependencies beyond dcm2niix). Falls back to **mri_deface** if TensorFlow is unavailable. See `docs/research/mri-defacing-tools-comparison.md` for detailed analysis.

**Modality-specific defacing needs:**
- **Brain MRI**: All defacing tools work well. Primary use case.
- **Brain PET**: mri_reface explicitly supports Amyloid PET, Tau PET, FDG PET. Others may need co-registered MRI for alignment.
- **Brain CT**: mri_reface supports CT. Others may require adaptation. Less common for research sharing.

**Key dependencies for defacing service container:**
- `dcm2niix` (static binary, ~2 MB) — DICOM to NIfTI conversion
- `deepdefacer` (pip, ~500 MB with TensorFlow) — 3D U-Net defacing (default)
- `mri_deface` + atlas files (~0.5 GB) — FreeSurfer atlas-based defacing (fallback)
- `pydicom` + `nibabel` — pixel data injection back into DICOM
- `FastAPI` / `uvicorn` — lightweight HTTP server
- Base image: `python:3.12-slim` or Alpine-based for minimal CVE surface

---

## Data Flow

```
1. Data enters through one of two ingress paths:
   - External-site path: browser performs client-side tag anonymization,
     then uploads de-identified files via signed URL or direct PUT.
   - Internal-enterprise path: batch import CLI or API ingest directly
     into the same processing pipeline.
2. Go API records intake session and metadata in PostgreSQL.
3. Files stored in cloud-neutral storage (local/S3/GCS) under dicom/raw/{studyUID}/.
4. Routing rules evaluate → set processing flags on the study.
5. Automated pipeline orchestrates services in dependency order:
   Phase 0: Classification (fills modality/body_part, re-evaluates routing)
   Phase 1: PHI scan + Protocol check + Defacing (parallel, raw files)
   Phase 2: QC check + BIDS conversion (after defacing, final files)
6. Admin reviews in dashboard (OHIF Viewer)
   - Processing status badges for each service
   - Defacing quality review (head imaging only — before/after OHIF panels)
7. Admin approves → study becomes shareable/exportable.
8. Export: token-authenticated zip download, DICOMweb STOW-RS forwarding
   to external destinations, or admin DICOM download.
```

---

## Security Architecture

| Layer | Measure |
|-------|---------|
| **Network** | Cloud: VPC, WAF/DDoS protection (Cloud Armor, AWS WAF), HTTPS LB |
| **Transport** | TLS 1.2+ enforced on all endpoints |
| **Auth (admin)** | Multi-provider: GCP IAP, Azure AD Easy Auth, AWS ALB + Cognito; dev mode auto-auth |
| **Authorization** | `admin_users` table with RBAC (admin/viewer roles); `RequireAuth`/`RequireRole` middleware |
| **Encryption at rest** | Cloud KMS (GCP), KMS (AWS), or filesystem encryption; CMEK for managed databases |
| **Audit** | App-level audit trail in PostgreSQL (per-study, per-action, with actor email); cloud audit logs |
| **PHI protection** | Client-side tag de-id; server-side PHI scan (Tesseract OCR) + defacing; no PHI in logs or email |
| **Container security** | Go: `distroless` base (~10 MB). Python sidecars: `slim` base, pinned deps |
| **Database** | PostgreSQL 15 (Cloud SQL / RDS / Docker) with private networking, encrypted, automated backups |
| **Email** | Standard SMTP (any provider); no PHI in email bodies |
| **Secrets** | Secret Manager (GCP), Secrets Manager (AWS), or env vars (dev) |
| **CI/CD** | GitHub Actions CI (Go build+vet+test, Python syntax, TypeScript type check, Docker build); Cloud Build CD (auto-deploy Cloud Run services + GCE DIMSE VM on push to `develop`) |

---

## Comparison to Existing Platforms

| Aspect | AEGIS | XNAT | Flywheel | LONI IDA | MIRC CTP |
|--------|-------------|------|----------|----------|----------|
| **Hosting** | Multi-cloud (GCP, AWS, Azure) | Self-hosted | Commercial SaaS | On-prem | On-prem gateway |
| **Client de-id** | Browser (zero install) | Electron desktop app | CLI / Edge connector | Java desktop app | On-site Java app |
| **Server de-id** | Defacing + PHI detection services | DicomEdit scripts | Built-in | Post-upload | Pipeline stages |
| **Defacing** | Automated server-side | Manual or plugin | Built-in | Separate | Not included |
| **DICOM store** | Cloud-neutral storage (local/S3/GCS) + built-in DICOMweb proxy | PostgreSQL + filesystem | MongoDB + S3 | MySQL + Isilon | Filesystem |
| **Viewer** | OHIF (embedded) | Built-in viewer | Built-in viewer | Web viewer | None |
| **Open source** | Yes | Yes | No | Partial | Yes |
| **Install at site** | None | Desktop client (optional) | CLI (optional) | Java app (required) | Java app (required) |
| **Modalities** | All DICOM | Neuroimaging | All imaging | Neuroimaging | All imaging |

**Key differentiators**:
1. Zero-install browser-based anonymization — no Java, no Electron, no CLI
2. Modality-agnostic: works with any DICOM data, not limited to a single specialty
3. Multi-cloud — runs on GCP, AWS, or Azure with Terraform modules for each; no cloud lock-in
4. Automated server-side defacing pipeline for head imaging
5. Pluggable AI backends — local (Tesseract, pydicom) for dev, optional cloud AI (Google Cloud Vision, AWS Textract/Rekognition) for production edge cases
6. Minimal vulnerability surface (Go backend, distroless containers)

---

## Phased Implementation

### Phase 1: Foundation (MVP)
1. **Terraform (prj)**: GCP project, APIs, IAM, KMS, VPC-SC
2. **Terraform (infra)**: VPC, Cloud Run, GCS, Healthcare API, Cloud SQL (PostgreSQL), BigQuery dataset, Cloud Armor, Pub/Sub
3. **Cloud SQL schema**: Users, projects, institutions, upload sessions, audit trail
4. **Go API**: Signed URL generation, upload orchestration, STOW-RS ingest, basic auth, PostgreSQL integration
5. **Upload Portal**: DICOM file picker, tag-level de-id engine, preview, chunked upload
6. **Admin Dashboard**: Study list, OHIF viewer, basic QC accept/reject
7. **BigQuery**: Healthcare API metadata export, basic audit queries

### Phase 2: Defacing Pipeline
8. **Defacing service**: Python Cloud Run, mri_deface + dcm2niix, DICOM pixel injection
9. **Admin defacing review**: Before/after comparison in dashboard
10. **Automated trigger**: Pub/Sub → detect head imaging → trigger defacing
11. **Email notifications**: PSC to on-prem SMTP, upload confirmations, defacing alerts

### Phase 3: Operations & Routing
12. ✅ **Routing rules engine**: Destinations table + RoutingRules table; priority-ordered evaluation on every ingest; actions: `require_defacing`, `auto_approve`, `require_qa`, `reject`, `route_to` (async DICOMweb forward); routing log per study; admin UI in Routing tab (PR #12)
13. ✅ **Institution management**: Institutions table + institution_projects join table; institution_id on studies/sessions; CRUD API + admin UI in Institutions tab; project-scoped roles (sender/receiver/admin) (PR #13)
14. ✅ **Per-project anonymization profiles**: Multiple named profiles per project; `retained_tags` JSONB list overrides Basic Profile strip/zero actions client-side; `default_anon_profile_id` on projects; upload portal fetches active profile at upload time; admin Profiles tab (PR #15)
15. ✅ **Audit log viewer**: Full trail from PostgreSQL + filterable UI in admin dashboard (PR #7)
16. ✅ **Email digests**: `digest_subscriptions` table; per-project weekly/monthly subscriptions; hourly scheduler goroutine sends plain-text summaries (study counts, share activity, no PHI); admin Notifications tab (PR #17)
17. ✅ **Admin user management**: `admin_users` table (migration 011); CRUD API; admin Users tab with role (admin|viewer), enable/disable, audit logging (PR #20)
18. ✅ **Project settings UI**: `GET/PUT /api/projects/{id}`; slug auto-generation on create; admin Projects tab with create/edit; `project.created`/`project.updated` audit entries (PR #21)
19. ✅ **Upload portal QoL**: `onFileStart` callback in `@aegis/client`; per-file filename display below progress bar; per-file PUT auto-retry (3× with exponential backoff) (PR #22)

### Phase 4: Advanced Processing (Pluggable Cloud AI)
20. ✅ **Burned-in PHI detection**: Python OCR service (`phi-detection/`) with Tesseract backend (local) and optional Google Cloud Vision / AWS Textract cloud backends; `require_phi_scan` routing rule action; `phi_scan_required`/`phi_scan_status` study fields; async dispatch from Go API; admin dashboard PHI scan badge + scan button (PR #29, PR #83)
21. ✅ **QC automation**: Python QC service (`qc-service/`) with pydicom+numpy backend; `require_qc_check` routing rule action; `qc_required`/`qc_status` study fields; 5 automated checks (file integrity, slice consistency, SNR, coverage, missing slices); async dispatch from Go API; admin dashboard QC badge + Run QC button (PR #32)
22. ✅ **Smart routing**: Python classification service (`classification-service/`) with heuristic DICOM tag analysis backend and optional Google Cloud Vision / AWS Rekognition cloud augmentation; `require_classification` routing rule action; `classification_required`/`classification_status` study fields; classifies modality + body_part from DICOM headers (SOP Class UID, SeriesDescription, BodyPartExamined patterns); updates study metadata and **re-evaluates routing rules** so downstream rules fire correctly; admin dashboard Classification badge + Classify button (PR #34, PR #83)
23. ✅ **NIfTI/BIDS conversion**: Python BIDS service (`bids-service/`) with dcm2niix backend; `require_bids_conversion` routing rule action; `bids_required`/`bids_status` study fields; DICOM→NIfTI conversion with BIDS-compliant directory structure + JSON sidecars; series-to-datatype classification; zip download endpoint; admin dashboard BIDS badge + Convert/Download buttons (PR #33)
24. ✅ **Batch import tools + contract hardening**: `api/cmd/import/` CLI + `POST /api/import/batch` API; recursive DICOM directory scan with `suyashkumar/dicom` header parsing; groups files by StudyInstanceUID; creates upload sessions + study records; evaluates routing rules; `--dry-run` mode. Hardening shipped: canonical institution selectors, strict JSON contract (`DisallowUnknownFields`), required absolute `dir` path, strict `source` validation (PR #31, PR #116–#125)
25. ✅ **MRI protocol compliance**: Python protocol service (`protocol-service/`) with pydicom-based parameter extraction for both Classic and Enhanced DICOM; `protocol_templates` table for per-project, per-manufacturer/model/software-version/sequence-type parameter rules with configurable tolerances and severity levels (critical/warning/info); `require_protocol_check` routing rule action; `protocol_required`/`protocol_status` study fields; async dispatch from Go API; admin dashboard Protocol badge + Check Protocol button + Protocol Templates CRUD tab (PR #41)
26. ✅ **Defacing tool upgrades**: DeepDefacer added as pluggable backend (`defacing/app/backends/deepdefacer_backend.py`); 3D U-Net deep learning defacing ~90% faster than registration-based tools; pip-installable with no external binaries beyond dcm2niix; `INCLUDE_DEEPDEFACER` Dockerfile build arg; auto-selection priority updated: mri_reface > deepdefacer > mri_deface > nibabel; `DEEPDEFACER_GPU` env var for CUDA support; `docs/research/mri-defacing-tools-comparison.md` with 5 peer-reviewed citations (PR #54)
26. ✅ **Authentication middleware**: Per-route auth middleware (`api/middleware/auth.go`) supporting GCP IAP and Azure AD Easy Auth; `RequireAuth` wrapper for admin routes; `RequireRole` for future viewer enforcement; `GET /api/auth/me` identity endpoint; `admin_users` lookup with case-insensitive email; dev mode auto-auth via `DEV_USER_EMAIL`; all audit entries now record real user email; admin dashboard shows current user and handles 401/403 errors (PR #36)
27. ✅ **Multi-cloud AWS support**: S3 storage backend (`api/storage/s3.go`) implementing the Storage interface with presigned URLs, copy-based move, and S3-compatible endpoint support (MinIO/LocalStack); AWS ALB + Cognito auth provider in middleware (JWT email extraction from `X-Amzn-Oidc-Data` header); `S3_BUCKET`/`S3_REGION`/`S3_ENDPOINT` config; `terraform/aws/main.tf` with VPC, RDS PostgreSQL 15, S3, ECS Fargate cluster, ALB, ECR, KMS, SNS/SQS, CloudWatch (PR #50)
28. ✅ **Study export workflow**: DICOM zip download for admins (`GET /api/studies/{studyUID}/dicom-download`) and share recipients (`GET /api/export/{token}/download`) using cloud-agnostic storage interface; export portal (`frontend/export-portal/`) React app for share recipients with study info display + ZIP download; DICOMweb STOW-RS forwarding (`multipart/related; type="application/dicom"`) with `io.Pipe()` streaming and 10-minute timeout; `require_export` routing rule action; `export_required`/`export_status` study fields; auto-dispatch on study approval; admin dashboard Export column + badge + trigger button (PR #56)

### Post-Phase 4 Shipped Enhancements (2026)
29. ✅ **DIMSE receiver service**: `dimse-receiver/` Python service with pynetdicom C-STORE SCP ingress on port `11112`; supports C-ECHO, writes to shared `dicom/raw/{studyUID}/`, triggers `POST /api/ingest` on association release, and reports health through API `/healthz` sidecar map. **Deployed on Compute Engine VM** (`aegis-prod-dimse-receiver`, static IP `35.232.172.221`, `us-central1-a`) — required because Cloud Run cannot expose raw TCP ports. VM uses gcsfuse to mount shared GCS staging bucket and Docker to run the receiver container.
30. ✅ **Timezone hardening + UI controls**: UTC-stable backend date logic and digest windows, epoch-based share countdowns, explicit timezone labels, and user-selectable display timezone controls synchronized across admin, upload, and export portals.
31. ✅ **Batch importer contract hardening**: strict input normalization/validation (`source`, `project_slug`, absolute `dir`), canonical institution selectors for external provenance, strict JSON decoding on `/api/import/batch`, and removal of deprecated importer AE-title selector inputs.
32. ✅ **Cloud Build CI/CD pipeline**: Two second-gen Cloud Build triggers (`deploy-on-develop`, `terraform-apply-on-develop`) in `us-central1`, connected to GitHub via Cloud Build GitHub App. Deploy trigger builds+pushes all 9 service images, deploys 8 Cloud Run services, and hot-swaps the DIMSE Receiver GCE VM via metadata update + instance reset. Terraform trigger applies `terraform/infra/` changes when `terraform/infra/**` files change on `develop`.
33. ✅ **Cloud Build IAM hardening**: Cloud Build service account `aegis-cloud-build@aegis-prod-488120.iam.gserviceaccount.com` granted all roles required for `terraform apply` (`roles/run.admin`, `roles/compute.admin`, `roles/iap.admin`, `roles/resourcemanager.projectIamAdmin`, `roles/artifactregistry.admin`, `roles/editor`, `roles/secretmanager.secretAccessor`, `roles/storage.admin`, `roles/iam.serviceAccountUser`). IAM roles tracked in `terraform/project/main.tf` and `scripts/gcp_setup_cloudbuild.sh`.

### Future Ideas (not planned)
- Non-DICOM formats: pathology whole-slide imaging, electron microscopy
- Client-side defacing preview via NiiVue (WebGL volume rendering)
- Modality-specific processing plugins (e.g., mammography CAD, cardiac segmentation)
- Federated learning integration

---

## Key Open-Source Dependencies

| Component | Library | License | Purpose |
|-----------|---------|---------|---------|
| Go DICOM | [suyashkumar/dicom](https://github.com/suyashkumar/dicom) | MIT | Server-side DICOM header parsing (batch import) |
| Browser DICOM read/write | [dcmjs](https://github.com/dcmjs-org/dcmjs) | MIT | Client-side tag modification |
| Browser DICOM parsing | [dicomParser](https://github.com/cornerstonejs/dicomParser) | MIT | Robust Part 10 parsing |
| Viewer | [OHIF Viewer](https://github.com/OHIF/Viewers) | MIT | DICOMweb viewer in admin |
| DICOM→NIfTI | [dcm2niix](https://github.com/rordenlab/dcm2niix) | BSD | Format conversion for defacing + BIDS |
| Defacing (default) | [DeepDefacer](https://pypi.org/project/deepdefacer/) | Research-friendly | 3D U-Net facial feature removal |
| Defacing (fallback) | [mri_deface](https://surfer.nmr.mgh.harvard.edu/fswiki/mri_deface) | Free | Atlas-based facial feature removal |
| Python DICOM | [pydicom](https://github.com/pydicom/pydicom) | MIT | Pixel data injection, parameter extraction |
| NIfTI I/O | [nibabel](https://github.com/nipy/nibabel) | MIT | NIfTI reading in defacing/BIDS |
| OCR | [Tesseract](https://github.com/tesseract-ocr/tesseract) | Apache-2.0 | Burned-in PHI detection |
| IaC (GCP) | [Terraform Google Provider](https://registry.terraform.io/providers/hashicorp/google/) | MPL-2.0 | GCP infrastructure |
| IaC (AWS) | [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/) | MPL-2.0 | AWS infrastructure |
| Go PostgreSQL | [pgx](https://github.com/jackc/pgx) | MIT | PostgreSQL driver |
| Go Migrations | [goose](https://github.com/pressly/goose) | MIT | Database schema migrations |
| Go S3 | [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | Apache-2.0 | S3 storage backend |
| Go GCS | [cloud.google.com/go/storage](https://pkg.go.dev/cloud.google.com/go/storage) | Apache-2.0 | GCS storage backend |
| Go Testing | [testify](https://github.com/stretchr/testify) + [testcontainers-go](https://github.com/testcontainers/testcontainers-go) | MIT | Assertions + PostgreSQL test containers |

---

## Verification Plan

After each phase, verify:

1. **Automated tests**: `make test-unit` (55 unit tests, no Docker) + `make test` (120 tests including PostgreSQL integration via testcontainers)
2. **Go API**: `make api` → `curl http://localhost:8080/healthz` shows healthy status for database, storage, and all configured sidecars
3. **Upload Portal**: Load a sample brain MRI DICOM directory, verify all PHI tags are stripped in the preview, upload succeeds
4. **Admin Dashboard**: View uploaded study in OHIF, verify de-identified tags, verify pipeline auto-dispatches processing
5. **Defacing**: Upload a head MRI with `require_defacing` routing rule → pipeline triggers defacing → compare original vs defaced in OHIF side-by-side
6. **End-to-end**: External browser → upload → de-id → ingest → classify → deface → QC → review → approve → export
7. **Docker Compose**: `docker compose up -d` → all 11 services healthy → full pipeline works
