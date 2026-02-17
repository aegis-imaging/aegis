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

### Upload Portal (React)
```bash
cd frontend/upload-portal && npm install && npm run dev   # runs on :3000, proxies /api to :8080
```

### Admin Dashboard (React)
```bash
cd frontend/admin-dashboard && npm install && npm run dev  # runs on :3001, proxies /api to :8080
```

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
