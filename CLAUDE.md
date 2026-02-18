# AEGIS — Project Guide

Anonymization & Exchange Gateway for Imaging Studies. GCP-hosted platform for HIPAA-compliant sharing of medical imaging data across all DICOM modalities. MVP focus: brain MRI, PET, and CT.

## Repository Structure (Monorepo)

```
aegis/
├── terraform/project/    # GCP project bootstrap (IAM, KMS, VPC-SC)
├── terraform/infra/      # Infrastructure (Cloud Run, Healthcare API, Cloud Armor)
├── api/                  # Go backend — upload orchestration, DICOMweb proxy
├── frontend/
│   ├── upload-portal/    # React — public-facing upload + anonymization UI
│   └── admin-dashboard/  # React — internal QC, OHIF viewer, study management
├── client/               # TypeScript DICOM anonymization library (npm package)
└── defacing/             # Python defacing service (mri_deface, dcm2niix)
```

Planned to split into 5 separate repos once interfaces stabilize:
`aegis-terraform-prj`, `aegis-terraform-infra`, `aegis-api`, `aegis-frontend`, `aegis-client`

## Tech Stack

- **Backend**: Go 1.23 on Cloud Run (distroless containers)
- **Database**: Cloud SQL (PostgreSQL 15) — users, projects, routing, audit
- **Frontend**: React 19 + TypeScript + Vite
- **DICOM**: GCP Healthcare API (DICOMweb), dcmjs, dicomParser
- **Defacing**: Python — mri_deface, dcm2niix, pydicom
- **Viewer**: OHIF Viewer (embedded in admin dashboard)
- **Analytics**: BigQuery — DICOM metadata export, audit dashboards
- **AI/ML**: Vertex AI — burned-in PHI detection, image QC, smart routing
- **Email**: On-prem SMTP via PSC (internal), SendGrid (external). Dev: standard SMTP.
- **Infrastructure**: Terraform, Cloud Build
- **Auth**: Google Identity / OAuth 2.0 (upload portal), IAP (admin dashboard)

## Development

### API (Go)
```bash
cd api && go run .          # runs on :8080
```

Email is disabled by default (silent no-op). To enable locally, run [Mailpit](https://github.com/axllent/mailpit) and set `SMTP_HOST`:
```bash
docker run -p 1025:1025 -p 8025:8025 axllent/mailpit
SMTP_HOST=localhost SMTP_PORT=1025 go run .
# View captured emails at http://localhost:8025
```

Email env vars (`api/email/client.go`, `api/config/config.go`):

| Var | Default | Notes |
|-----|---------|-------|
| `SMTP_HOST` | *(empty — disabled)* | Set to enable; empty = silent no-op |
| `SMTP_PORT` | `587` | Use `1025` with Mailpit |
| `SMTP_FROM` | `noreply@aegis.local` | Envelope sender address |
| `SMTP_USERNAME` | *(empty)* | Omit for unauthenticated relays |
| `SMTP_PASSWORD` | *(empty)* | |

Triggers: share created → recipient email; upload complete → uploader (if provided at upload time); study approved/rejected → uploader. No PHI in any email body.

### Upload Portal — Email field
The preview step shows an optional "Your email" input. The address is sent with `POST /api/upload/init` as `uploader_email` and stored in `upload_sessions`. If blank, no notification is sent. The `@aegis/client` `UploadOptions.uploaderEmail` field carries it through.

### Upload Portal (React)
```bash
cd frontend/upload-portal && npm install && npm run dev   # runs on :3000, proxies /api to :8080
```

### Admin Dashboard (React)
```bash
cd frontend/admin-dashboard && npm install && npm run dev  # runs on :3001, proxies /api to :8080
```

### OHIF Viewer
OHIF Viewer runs as a Docker container on `:3002`, configured to load DICOM images via the Go API's DICOMweb proxy.

```bash
docker compose up ohif       # start OHIF only
docker compose up            # start postgres + mailpit + ohif
# OHIF available at http://localhost:3002
```

`ohif-config.js` (repo root) configures the OHIF data source pointing at `http://localhost:8080/dicomweb`.

**DICOMweb proxy** (`api/handler/dicomweb.go`) — minimal QIDO-RS + WADO-RS, no DICOM library:
- `GET /dicomweb/studies` — list studies from DB
- `GET /dicomweb/studies/{studyUID}/series` — single fake series per study
- `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances` — enumerate instances by file count
- `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}` — stream DICOM bytes from `dicom_store`

**Raw DICOMweb proxy** — same routes under `/dicomweb-raw/*`, but WADO-RS always reads from `dicom/raw/` regardless of `dicom_store`. Used by OHIF's `dicomweb-raw` data source for defacing review.

SOPInstanceUID format: `{studyUID}.1.{fileIndex}` (index maps to `dicom/{store}/{studyUID}/{index}.dcm`).

In the admin dashboard:
- Each study row has a **View** button (inline iframe) and an **Open in new tab ↗** link.
- Head studies with `defacing_required=true` and `status=defaced|approved` show a **Review defacing** button that opens a side-by-side before/after OHIF panel. OHIF selects the data source via `?dataSource=dicomweb-raw` (before) or `?dataSource=dicomweb` (after).

### Terraform
```bash
cd terraform/project && terraform init && terraform plan
cd terraform/infra && terraform init && terraform plan
```

## Key Architecture Decisions

- Client-side DICOM tag anonymization in browser before upload (zero-install at sending sites)
- Dual-ingress model: external-site browser upload and internal-enterprise ingestion both enter the same enterprise GCP tenancy and processing pipeline
- Server-side defacing in separate Python Cloud Run service
- Two DICOM stores: `raw` (tag-de-identified) and `clean` (fully processed including defacing)
- Go for main API (minimal CVE surface, fast cold starts), Python only for defacing sidecar
- DICOM PS3.15 Annex E Basic Profile for de-identification
- Cloud SQL (PostgreSQL) for application state; Healthcare API for DICOM data
- BigQuery for DICOM metadata analytics and audit reporting
- Vertex AI for burned-in PHI detection and image QC (Phase 4)
- Dual-path email: PSC→on-prem SMTP for internal, SendGrid for external (dev: standard SMTP)
- Modality-agnostic de-identification; defacing only for head imaging

## Git Workflow — Full Feature Lifecycle

Never commit directly to `develop` or `main`.

**Start a feature:**
```bash
git checkout -b feature/your-feature-name origin/develop
```

**Closing process (run after every feature):**
```bash
git add <files>
git commit -m "..."
git push -u origin feature/your-feature-name
gh pr create --base develop --head feature/your-feature-name --title "..." --body "..."
gh pr merge <number> --merge --delete-branch
git checkout develop && git pull
# then branch again for the next feature
```

## Conventions

- Product naming: always use **Anonymization & Exchange Gateway for Imaging Studies** for first mention, then **AEGIS** thereafter.
- Go: standard library preferred, minimal dependencies
- Frontend: functional React components, TypeScript strict mode
- Terraform: one module per logical resource group
- Docker: distroless for Go, slim for Python
- All services expose `/healthz` for health checks

## Current Phase

Phase 1 (MVP): Foundation — Terraform, Go API, Upload Portal, Admin Dashboard
See ARCHITECTURE.md for full phased plan.
