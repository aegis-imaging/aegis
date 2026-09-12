# AEGIS — Project Guide

Anonymization & Exchange Gateway for Imaging Studies. Cloud-hosted (GCP/AWS/Azure) HIPAA-compliant medical imaging data sharing platform. MVP: brain MRI, PET, CT.

## Critical Rules

- **PDF generation**: When editing a markdown file that has a committed PDF twin (`SETUP_CHECKLIST.md`, the checklists in `docs/planning/`), regenerate it with `npx md-to-pdf <file>` and commit both
- **Business and personal documents** (executive summary, decks, financials, pricing, competitive analyses, personal notes) live in a private repository, never here
- **No PHI/CBI** in the repo ever
- **Research docs**: Check `docs/research/` before web searching; save findings there with full citations
- **Git workflow**: Never commit directly to `develop` or `main`. Branch from `develop`, PR back, merge via `gh pr merge`
- **CI/CD**: Never deploy by hand — merging to `develop` deploys through GitHub Actions on every cloud (see `docs/runbooks/ci-cd.md`)

## Repository Structure

```
aegis/
├── terraform/{project,infra,aws,azure}/  # IaC per cloud
├── api/                    # Go 1.24 backend (Cloud Run / ECS Fargate)
├── frontend/
│   ├── upload-portal/      # React 19 — public upload + anonymization
│   ├── admin-dashboard/    # React 19 — QC, viewer, study management
│   ├── export-portal/      # React 19 — share download UI
│   └── landing/            # React 19 — aegisimaging.ai (static, Cloudflare Pages)
├── client/                 # TypeScript DICOM anonymization library
├── defacing/               # Python — mri_deface, DeepDefacer, dcm2niix
├── phi-detection/          # Python — burned-in PHI OCR + pixel redaction
├── qc-service/             # Python — automated QC (pydicom + numpy)
├── bids-service/           # Python — NIfTI/BIDS conversion (dcm2niix)
├── classification-service/ # Python — modality/body part classification
├── protocol-service/       # Python — MRI protocol compliance
├── synth-service/          # Python — synthetic DICOM brain MRI
├── analytics-service/      # Python — neuroimaging (FreeSurfer, FSL, ANTs, etc.)
├── sct-service/            # Python — Spinal Cord Toolbox
├── dimse-receiver/         # Python — DIMSE C-STORE SCP + ingest
├── mcp-server/             # TypeScript MCP server for AI agents
└── docs/                   # Research, runbooks, references
```

## Tech Stack

- **Backend**: Go 1.24, PostgreSQL 15, distroless containers
- **Frontend**: React 19 + TypeScript + Vite
- **Storage**: Cloud-neutral (`STORAGE_MODE`: `local`/`gcs`/`s3`/`azure`) with DICOMweb proxy
- **Auth**: GCP IAP / Azure AD / AWS Cognito / API keys; dev: `AUTH_ENABLED=false`
- **Email**: SMTP (dev: Mailpit on port 1025/8025)
- **Infra**: Terraform, Docker Compose for local dev

## Development Quick Start

```bash
# Full stack
docker compose up -d

# Individual services
cd api && go run .                                          # :8080
cd frontend/upload-portal && npm install && npm run dev     # :3000
cd frontend/admin-dashboard && npm install && npm run dev   # :3001
cd frontend/landing && npm install && npm run dev           # :3003
cd frontend/export-portal && npm install && npm run dev     # :3004
```

## Testing

```bash
# Go (~137 tests, needs Docker for testcontainers)
cd api && go test -v -count=1 ./...     # all tests
cd api && go test -short -v ./...       # unit only (no Docker)
cd api && go test -race ./...           # race detector

# Python sidecars (~664 tests total)
cd {service} && pip install -r requirements.txt -r requirements-test.txt && pytest -v

# Makefile shortcuts
make test          # Go all
make test-unit     # Go unit only
make lint          # All languages
```

**Test helpers** in `api/testutil/`: `TestDB(t)` (PostgreSQL container), `TestServer(t, db)`, `SeedProject(t, db)`, `CreateTestStudy(t, db, projectID)`, `CreateTestAdminUser(t, db, email, role)`, `CreateTestDestination(t, db, name)`, `CreateTestRoutingRule(t, db, name, action)`.

## Architecture Overview

### Pipeline (4 phases, auto-dispatched when `PIPELINE_AUTO=true`)

```
Phase 0: Classification (fills modality/body_part, re-evaluates routing)
Phase 1: PHI scan + Protocol check + Defacing (parallel, raw files)
Phase 2: QC + BIDS conversion (after defacing)
Phase 3: Analytics + SCT (parallel, post-BIDS NIfTI)
```

### Key Patterns

- **Routing rules**: Priority-ordered, all matching rules fire. Actions: `require_defacing`, `require_phi_scan`, `require_qc_check`, `require_bids_conversion`, `require_classification`, `require_protocol_check`, `require_export`, `require_analytics`, `require_sct`, `auto_approve`, `reject`, `route_to`
- **Two DICOM stores**: `raw` (tag-de-identified) and `clean` (post-defacing)
- **Sidecar pattern**: Each Python service is a FastAPI app called async from Go API. Env var `{SERVICE}_URL` enables each (empty = disabled)
- **Auth middleware**: `RequireAuth` for read endpoints, `RequireRole("admin")` for writes. Viewer role is read-only
- **Audit trail**: All mutations emit audit entries with actor, action, resource, metadata

### Key Env Vars

| Var | Default | Notes |
|-----|---------|-------|
| `DATABASE_URL` | *(empty)* | Postgres DSN; falls back to `DB_HOST`/`DB_PORT`/`DB_NAME`/`DB_USER`/`DB_PASSWORD` |
| `STORAGE_MODE` | `local` | `local`/`gcs`/`s3`/`azure` |
| `AUTH_ENABLED` | `false` | `true` in production |
| `PIPELINE_AUTO` | `true` | Auto-dispatch pipeline steps |
| `DEFACING_SERVICE_URL` | *(empty)* | Enable defacing sidecar |
| `PHI_DETECTION_SERVICE_URL` | *(empty)* | Enable PHI detection |
| `QC_SERVICE_URL` | *(empty)* | Enable QC |
| `BIDS_SERVICE_URL` | *(empty)* | Enable BIDS conversion |
| `CLASSIFICATION_SERVICE_URL` | *(empty)* | Enable classification |
| `PROTOCOL_SERVICE_URL` | *(empty)* | Enable protocol check |
| `ANALYTICS_SERVICE_URL` | *(empty)* | Enable analytics |
| `SCT_SERVICE_URL` | *(empty)* | Enable SCT |
| `SYNTH_SERVICE_URL` | *(empty)* | Enable synthetic MRI |
| `SMTP_HOST` | *(empty)* | Enable email (empty = silent no-op) |

## CI/CD

- **`ci.yml`**: PRs to `develop`/`main` — Go build/vet/test, Python compile/test, TypeScript typecheck, Docker builds, terraform fmt/validate
- **`terraform-{gcp,aws,azure}.yml`**: plan on PRs touching that cloud's `terraform/` dir; on merge to `develop`, apply behind the `gcp-prod` / `aws-prod` / `azure-prod` environment approval. OIDC auth, `-json` output filtered by `scripts/terraform_ci.sh` (logs are public)
- **`deploy-{gcp,aws,azure}.yml`**: every push to `develop` builds all images, pushes to the registry, and rolls the services that exist. AWS/Azure are no-ops until `AWS_CI_ENABLED` / `AZURE_CI_ENABLED` are `true`
- **Landing page**: Cloudflare Pages builds `frontend/landing` from `develop`
- `terraform/project/` is always manual (dangerous bootstrap resources). Full bootstrap and secrets list: `docs/runbooks/ci-cd.md`

## Conventions

- Go: standard library preferred, minimal dependencies
- Frontend: functional React, TypeScript strict mode
- Terraform: one module per logical resource group
- Docker: distroless for Go, slim for Python
- All services expose `/healthz`
- Product name: "Anonymization & Exchange Gateway for Imaging Studies" first mention, then "AEGIS"
