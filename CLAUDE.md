# AEGIS — Project Guide

Anonymization & Exchange Gateway for Imaging Studies. GCP-hosted platform for HIPAA-compliant sharing of medical imaging data across all DICOM modalities. MVP focus: brain MRI, PET, and CT.

## Pitch Deck PDF

Whenever `PITCH_DECK.md` is edited, regenerate the PDF and commit both files in the same PR:

```bash
npx md-to-pdf PITCH_DECK.md   # generates PITCH_DECK.pdf in repo root
```

Never update the markdown without updating the PDF.

## Shared Knowledge: `docs/` Folder

All research, analysis, and reference material lives in `docs/` and is shared among agents, developers, and collaborators.

**Before doing a web search**, check `docs/research/` — the answer may already be there.
**After completing research**, save findings to `docs/research/<topic>.md` with full citations.

Rules:
- Always include full citations (author, journal, year, DOI/URL) and note source type (peer-reviewed, preprint, market report, government primary source)
- **Never save PHI (Protected Health Information) or CBI (Confidential Business Information)** to `docs/` or anywhere in the repository
- See `docs/README.md` for the full convention

Current research files:
- `docs/research/medical-imaging-deidentification.md` — citations for de-identification failures, face reconstruction from MRI, burned-in PHI, NIH DMS policy, HIPAA Safe Harbor, DICOM PS3.15, MIDI-B challenge, market sizing

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
├── defacing/             # Python defacing service (mri_deface, dcm2niix)
├── phi-detection/        # Python burned-in PHI detection service (Tesseract OCR)
└── docs/                 # Shared research, references, and analysis (see docs/README.md)
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
- **Studies tab** — filter by status, modality, source, project; search by UID or description; paginated 50 per page.
- **Routing tab** — manage Destinations and Routing Rules (see below).
- **Institutions tab** — manage institutions and their project memberships.
- **Profiles tab** — manage per-project anonymization profiles (see below).
- **Notifications tab** — manage email digest subscriptions (see below).
- **Projects tab** — create and edit projects (name, slug, description); shows default anon profile badge.
- **Users tab** — manage authorised admin users and their roles (admin|viewer).

### Studies List (`GET /api/studies`)

Returns a paginated envelope `{ studies, total, limit, offset }`.

| Query param | Notes |
|-------------|-------|
| `limit` | Page size (default 50, max 200) |
| `offset` | Row offset for pagination |
| `project_id` | Filter by project UUID |
| `status` | `received\|defacing\|clean\|defaced\|approved\|rejected` |
| `modality` | Case-insensitive exact match (e.g. `MRI`, `CT`) |
| `source` | `external\|internal` |
| `search` | Substring match on `study_instance_uid` or `study_description` |

### Routing Rules Engine (`api/routing/`, `api/handler/routing.go`)

Routing rules are evaluated on every study ingest (upload complete + internal ingest). Rules are ordered by `priority` (lower = first); all matching rules fire.

**Destinations** (`/api/destinations`) — external DICOMweb endpoints:

| Field | Notes |
|-------|-------|
| `type` | `dicomweb` or `dimse` (DIMSE is placeholder) |
| `dicomweb_url` | Base URL for STOW-RS; auth via `dicomweb_auth_header` |

**Routing Rules** (`/api/routing-rules`):

| Condition field | Meaning |
|----------------|---------|
| `project_id` | Match a specific project (null = any) |
| `modality` | e.g. `MRI`, `CT`, `PET` (null = any) |
| `body_part` | e.g. `HEAD`, `CHEST` (null = any) |
| `source` | `external` or `internal` (null = any) |

| Action | Effect |
|--------|--------|
| `require_defacing` | Forces `defacing_required=true` |
| `require_phi_scan` | Forces `phi_scan_required=true`, sets `phi_scan_status=pending` |
| `auto_approve` | Skips manual QC, sets `status=approved` |
| `require_qa` | No-op — holds for manual review (default) |
| `reject` | Auto-rejects the study |
| `route_to` | Async forward to a Destination via DICOMweb |

Other endpoints:
- `POST /api/routing-rules/evaluate/{studyID}` — re-evaluate rules for an existing study
- `GET /api/studies/{studyID}/routing-log` — per-study rule execution log

### Institution Management (`api/handler/institution.go`)

Institutions represent organisations that send or receive studies.

**REST API** (`/api/institutions`): CRUD + project linking.

| Field | Notes |
|-------|-------|
| `institution_type` | `sender`, `receiver`, or `both` |
| `ip_ranges` | Comma-separated CIDR blocks (future: auto-attribute uploads) |
| `ae_title` | DICOM AE title (future: DIMSE sender identification) |

**Institution-Project links** (`/api/institutions/{id}/projects`):
- `POST` — link with role (`sender`, `receiver`, `admin`)
- `DELETE /api/institutions/{id}/projects/{projectID}` — unlink

Studies carry an `institution_id` FK (nullable) for full traceability.

### Anonymization Profiles (`api/handler/anon_profile.go`, `api/model/anon_profile.go`)

Named per-project overrides for the client-side DICOM PS3.15 Basic Profile de-identification.
Tags listed in `retained_tags` are kept as-is instead of being stripped/zeroed.

**REST API:**
- `GET /api/projects/{projectID}/anon-profiles` — list profiles for a project
- `POST /api/projects/{projectID}/anon-profiles` — create profile (`name`, `retained_tags: []string`, `description`)
- `GET /api/anon-profiles/{id}` / `PUT /api/anon-profiles/{id}` / `DELETE /api/anon-profiles/{id}`
- `PUT /api/projects/{projectID}/default-anon-profile` — set default profile (`{"profile_id":"<uuid>"}`, empty string to clear)
- `GET /api/projects/{slug}/active-anon-profile` — returns the default profile for the project (204 if none); used by the upload portal

The upload portal fetches the active profile before each upload and passes `retainedTags` into `@aegis/client`'s `DeidOptions`. Non-fatal if the fetch fails (falls back to full strip).

`retained_tags` is a JSONB array of DICOM keyword strings, e.g. `["PatientAge","StudyDate"]`.

### Email Digest Subscriptions (`api/handler/digest.go`, `api/digest/scheduler.go`)

Weekly or monthly plain-text summary emails per project. No PHI — only study counts and share activity.

**REST API:**
- `GET /api/digest-subscriptions` — all subscriptions
- `GET /api/projects/{projectID}/digest-subscriptions` — filtered by project
- `POST /api/projects/{projectID}/digest-subscriptions` — create (`email`, `frequency: weekly|monthly`)
- `DELETE /api/digest-subscriptions/{id}`

**Scheduler**: goroutine started from `main.go` on startup; `time.Ticker` fires every hour; queries `digest_subscriptions` where digest is due (7 days for weekly, 30 for monthly since `last_sent_at`); sends email; updates `last_sent_at`. Silent no-op when `SMTP_HOST` is unset.

**Digest content**: project name, period label, received/approved/rejected/pending study counts, export shares created. No study UIDs or identifiers.

### Admin Users (`api/handler/admin_user.go`, `api/model/admin_user.go`)

Authorised dashboard users and their roles. Authentication is handled by GCP IAP in production; this table is a registry for access control and auditing.

**REST API** (`/api/admin-users`): CRUD.

| Field | Notes |
|-------|-------|
| `role` | `admin` (full access) or `viewer` (read-only — future enforcement) |
| `enabled` | Soft-disable without deleting |
| `notes` | Free-text notes for the admin record |

All mutations emit audit entries (`admin_user.created`, `admin_user.updated`, `admin_user.deleted`).

### Project Settings (`api/handler/project.go`)

Projects now support full CRUD via the API and a dedicated admin dashboard tab.

**New endpoints:**
- `GET /api/projects/{id}` — fetch a single project by ID
- `PUT /api/projects/{id}` — update name, slug, description; emits `project.updated` audit entry

`POST /api/projects` now auto-generates slug from name if `slug` is omitted.

### Upload Portal QoL (`client/src/upload/client.ts`)

- **`onFileStart` callback** — `UploadOptions.onFileStart?(filename, index, total)` fires before each file's upload begins; upload portal uses it to display the current filename below the progress bar.
- **Auto-retry** — each file PUT is retried up to 3× with 1 s / 2 s / 4 s exponential backoff before failing. Transparent to callers.

### Burned-in PHI Detection (`phi-detection/`, `api/handler/phi_scan.go`)

Server-side OCR on DICOM pixel data to detect burned-in text (patient names, dates, accession numbers) that tag-level de-identification misses. Runs as a separate Python FastAPI service, called asynchronously from the Go API — same pattern as the defacing service.

**Running locally:**
```bash
cd phi-detection
pip install -r requirements.txt
# Requires tesseract-ocr installed: brew install tesseract (macOS) or apt-get install tesseract-ocr (Linux)
uvicorn app.main:app --port 8082
# Then set PHI_DETECTION_SERVICE_URL=http://localhost:8082 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `PHI_DETECTION_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = studies stay in "pending" |
| `PHI_TOOL` | `auto` | Backend selection: `auto` or `tesseract` |
| `PHI_CONFIDENCE_THRESHOLD` | `0.4` | Minimum OCR confidence (0.0–1.0) |
| `PHI_MIN_TEXT_LENGTH` | `3` | Minimum text length to report |

**Study fields:**
- `phi_scan_required` — boolean flag, set by `require_phi_scan` routing rule action
- `phi_scan_status` — `''` (not required), `pending`, `scanning`, `clean`, `flagged`, `failed`

**API:**
- `POST /api/studies/{studyUID}/phi-scan` — trigger PHI scan (returns 202 Accepted, runs async)

**Pipeline:**
1. Routing rule with action `require_phi_scan` sets `phi_scan_required=true` and `phi_scan_status=pending`
2. Admin clicks "Scan for PHI" → Go handler sets status to `scanning` and dispatches to Python service
3. Python service reads each DICOM file, extracts pixel data, runs Tesseract OCR
4. Results returned: `phi_scan_status` set to `clean` (no text found) or `flagged` (text detected)
5. Findings stored as JSONB in audit trail (`phi_scan.complete` entries)
6. Admin can still approve flagged studies (the flag is informational)

**Admin dashboard:**
- PHI Scan column with status badge (pending/scanning/clean/flagged/failed)
- "Scan for PHI" button for pending studies
- `require_phi_scan` option in routing rules action dropdown

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
