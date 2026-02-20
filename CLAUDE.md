# AEGIS — Project Guide

Anonymization & Exchange Gateway for Imaging Studies. Cloud-hosted (GCP, AWS, or Azure) platform for HIPAA-compliant sharing of medical imaging data across all DICOM modalities. MVP focus: brain MRI, PET, and CT.

## PDF Generation Rules

Whenever any of these markdown files are edited, regenerate the corresponding PDF and commit both files in the same PR:

```bash
npx md-to-pdf AEGIS_Executive_Summary.md   # generates AEGIS_Executive_Summary.pdf
npx md-to-pdf SETUP_CHECKLIST.md           # generates SETUP_CHECKLIST.pdf
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
- `docs/research/mri-protocol-compliance.md` — MRI acquisition parameter ranges, consortia protocols (ADNI4, HCP, ABCD, UK Biobank, ENIGMA), tolerance recommendations, mrQA tool, Enhanced vs Classic DICOM
- `docs/research/mri-defacing-tools-comparison.md` — tool comparison (afni_refacer, DeepDefacer, PyDeface, mri_deface, Quickshear), success rates, speed benchmarks, Docker size, licensing

## Repository Structure (Monorepo)

```
aegis/
├── terraform/project/    # GCP project bootstrap (IAM, KMS, VPC-SC)
├── terraform/infra/      # GCP infrastructure (Cloud Run, Healthcare API, Cloud Armor)
├── terraform/aws/        # AWS infrastructure (ECS Fargate, S3, RDS, ALB)
├── api/                  # Go backend — upload orchestration, DICOMweb proxy
├── frontend/
│   ├── upload-portal/    # React — public-facing upload + anonymization UI
│   ├── admin-dashboard/  # React — internal QC, OHIF viewer, study management
│   ├── export-portal/    # React — public-facing export share download UI
│   └── landing/          # React — public landing page (aegisimaging.ai)
├── client/               # TypeScript DICOM anonymization library (npm package)
├── defacing/             # Python defacing service (DeepDefacer, mri_deface, dcm2niix)
├── phi-detection/        # Python burned-in PHI detection service (Tesseract / Cloud Vision / Textract)
├── qc-service/           # Python QC automation service (pydicom + numpy)
├── bids-service/            # Python NIfTI/BIDS conversion service (dcm2niix)
├── classification-service/  # Python metadata classification service (heuristic / Cloud Vision / Rekognition)
├── protocol-service/        # Python MRI protocol compliance service (pydicom)
├── dimse-receiver/          # Python DIMSE adapter (pynetdicom C-STORE SCP + ingest trigger)
└── docs/                    # Shared research, references, and analysis (see docs/README.md)
```

Planned to split into 5 separate repos once interfaces stabilize:
`aegis-terraform-prj`, `aegis-terraform-infra`, `aegis-api`, `aegis-frontend`, `aegis-client`

## Tech Stack

- **Backend**: Go 1.24 on Cloud Run / ECS Fargate (distroless containers)
- **Database**: PostgreSQL 15 (Cloud SQL on GCP, RDS on AWS) — users, projects, routing, audit
- **Frontend**: React 19 + TypeScript + Vite
- **DICOM Storage**: Cloud-neutral file storage (local, GCS, or S3) with DICOMweb proxy
- **DICOM Networking**: DIMSE receiver sidecar (pynetdicom C-STORE SCP on port 11112)
- **Defacing**: Python — mri_deface, dcm2niix, pydicom
- **Viewer**: OHIF Viewer (embedded in admin dashboard)
- **AI/ML**: Pluggable — local backends (Tesseract OCR, pydicom heuristics) or cloud AI (Google Cloud Vision, AWS Textract/Rekognition)
- **Email**: Standard SMTP (works with any provider). Dev: Mailpit.
- **Infrastructure**: Terraform (GCP and AWS modules), Docker Compose for local dev
- **Auth**: Multi-provider — GCP IAP, Azure AD Easy Auth, AWS ALB + Cognito; dev mode auto-auth

### Multi-Cloud Support

AEGIS is cloud-agnostic at the application layer. The same Go API, Python sidecars, and React frontends run on any cloud or on-premises.

| Component | GCP | AWS | Azure | Local Dev |
|-----------|-----|-----|-------|-----------|
| **File storage** | GCS (`STORAGE_MODE=gcs`) | S3 (`STORAGE_MODE=s3`) | — | Filesystem (`STORAGE_MODE=local`) |
| **Database** | Cloud SQL | RDS | Azure Database | Docker postgres |
| **Containers** | Cloud Run | ECS Fargate | Container Apps | Docker Compose |
| **Auth** | IAP (`AUTH_PROVIDER=iap`) | ALB + Cognito (`AUTH_PROVIDER=aws`) | Easy Auth (`AUTH_PROVIDER=azure`) | Auto-auth (`AUTH_ENABLED=false`) |
| **Terraform** | `terraform/project/` + `terraform/infra/` | `terraform/aws/` | — (planned) | N/A |

S3-compatible stores (MinIO, LocalStack) are supported via the `S3_ENDPOINT` env var.

## Development

### Testing (Go)

The Go API has a comprehensive test suite (~120 tests) using `testify` for assertions and `testcontainers-go` for integration tests against real PostgreSQL.

```bash
# All tests (requires Docker for testcontainers)
cd api && go test -v -count=1 ./...

# Unit tests only (no Docker needed, fast)
cd api && go test -short -v ./...

# With race detector
cd api && go test -race ./...

# Single package
cd api && go test -v ./routing/
```

Makefile shortcuts: `make test`, `make test-unit`, `make test-race`

**Test architecture:**

| Tier | What | DB? | Location |
|------|------|-----|----------|
| Unit | Pure functions (routing Matches, JWT parsing, CORS, config, email templates, storage, slugify) | No | `*_test.go` in each package |
| Model integration | CRUD against real PostgreSQL | Yes | `api/model/*_test.go` |
| Handler HTTP | Full request/response via httptest | Yes | `api/handler/*_test.go` |
| Auth integration | Middleware with DB lookups | Yes | `api/middleware/auth_integration_test.go` |

**Test helpers** (`api/testutil/`):
- `TestDB(t)` — spins up PostgreSQL 15 container, runs migrations, returns `*sql.DB`
- `TestServer(t, db)` — creates `handler.Server` with temp local storage and `PipelineAuto: false`
- `SeedProject(t, db)` — returns the default project from migration 001
- `CreateTestStudy(t, db, projectID)` — creates a study with sensible defaults
- `CreateTestAdminUser(t, db, email, role)` — creates an admin user
- `CreateTestDestination(t, db, name)` — creates a DICOMweb destination
- `CreateTestRoutingRule(t, db, name, action)` — creates a routing rule

Integration tests are skipped with `-short` flag for fast local feedback.

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

### Upload Portal (React)
```bash
cd frontend/upload-portal && npm install && npm run dev   # runs on :3000, proxies /api to :8080
```

**Features:**
- **Project selector** — dropdown populated from `GET /api/projects`; auto-selects if only one project exists; hidden when single project
- **Drag-and-drop** — folder and multi-file support via `webkitdirectory` and DataTransfer API; shows file count + total size after selection
- **Multi-study detection** — groups files by StudyInstanceUID; each study gets its own summary card and uploads as a separate session
- **Anonymization preview** — before/after tag diff table (PS3.15 Annex E Basic Profile); per-project retained tags from anonymization profiles
- **Email validation** — optional uploader email with format validation; helper text explains notification triggers (approved/rejected)
- **Cancel support** — cancel button during parsing and upload stages; aborts in-flight requests
- **File progress** — per-file upload progress bar with current filename display and auto-retry (3× exponential backoff)

### Admin Dashboard (React)
```bash
cd frontend/admin-dashboard && npm install && npm run dev  # runs on :3001, proxies /api to :8080
```

### Landing Page (React)
```bash
cd frontend/landing && npm install && npm run dev    # runs on :3003
```

Static marketing site for aegisimaging.ai. Deployed to Vercel, separate from the GCP/AWS backend.

**Contact form** has two delivery paths (both use the same `/api/contact` endpoint):

| Path | How it works | When to use |
|------|-------------|-------------|
| **Vercel serverless function** | `frontend/landing/api/contact.ts` — nodemailer + Brevo SMTP | Landing page on Vercel (standalone, no Go API needed) |
| **Go API endpoint** | `POST /api/contact` — uses existing SMTP config | Landing page proxied to Go backend (local dev or production) |

Vercel env vars (set in Vercel dashboard):

| Var | Default | Notes |
|-----|---------|-------|
| `BREVO_SMTP_HOST` | *(empty — disabled)* | `smtp-relay.brevo.com`; empty = logs to console |
| `BREVO_SMTP_PORT` | `587` | |
| `BREVO_SMTP_USER` | *(empty)* | Brevo SMTP credentials |
| `BREVO_SMTP_PASS` | *(empty)* | |
| `BREVO_SMTP_FROM` | `AEGIS <noreply@aegisimaging.ai>` | Envelope sender |

Go API env var (optional):

| Var | Default | Notes |
|-----|---------|-------|
| `CONTACT_EMAIL` | `contact@aegisimaging.ai` | Recipient for contact form submissions |

### Export Portal (React)
```bash
cd frontend/export-portal && npm install && npm run dev  # runs on :3004, proxies /api to :8080
```

Public-facing download page for export share recipients. Reads a share token from `?token=...` query param, calls `GET /api/export/{token}` to validate, and shows study info with a "Download All (ZIP)" button pointing at `GET /api/export/{token}/download`.

### Full-Stack Docker Compose

`docker compose up` starts the entire platform: PostgreSQL, Mailpit, OHIF, Go API, and all 7 Python sidecar services. All services share a named `aegis-data` volume for DICOM file exchange.

```bash
docker compose up -d          # start everything (background)
docker compose up ohif        # start OHIF only
docker compose down           # stop all (data persists)
docker compose down -v        # stop all + destroy volumes
```

| Service | Port | Notes |
|---------|------|-------|
| postgres | 5432 | PostgreSQL 15, data in `pgdata` volume |
| mailpit | 1025 (SMTP) / 8025 (UI) | Email capture for dev |
| ohif | 3002 | OHIF Viewer, waits for API health |
| api | 8080 | Go API, runs migrations on startup |
| defacing | (internal) | Python defacing service |
| phi-detection | (internal) | Burned-in PHI detection (Tesseract) |
| qc-service | (internal) | Automated QC checks |
| bids-service | (internal) | NIfTI/BIDS conversion (dcm2niix) |
| classification-service | (internal) | Metadata classification |
| protocol-service | (internal) | MRI protocol compliance |
| dimse-receiver | 11112 (DICOM), 8080 (internal health) | Receives DICOM via C-STORE and calls API ingest |

Most sidecar services have no host port mapping — the Go API reaches them via Docker internal DNS (e.g., `http://defacing:8080`). `dimse-receiver` exposes port `11112` so external PACS systems can send C-STORE directly. The API's `LOCAL_STORAGE_DIR=/app/data` and all sidecars mount the same volume at `/app/data`.

**Health check** (`GET /healthz`) returns JSON with database, storage, and per-sidecar status:
```json
{"status":"ok","database":"healthy","storage":"healthy","services":{"defacing":"healthy",...}}
```

### OHIF Viewer
OHIF Viewer runs as a Docker container on `:3002`, configured to load DICOM images via the Go API's DICOMweb proxy.

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

### Study Detail (`GET /api/studies/{id}`)

Returns a single study by UUID. Used by the admin dashboard's study detail panel.

### Study Audit (`GET /api/studies/{id}/audit`)

Returns all audit trail entries for a specific study (by `resource_id`). Used by the study detail panel's audit tab.

### Study Detail Panel (Admin Dashboard)

Clicking a study UID in the studies table navigates to a dedicated detail view with:
- **Header** — full study UID, status/source badges, description
- **Meta row** — modality, body part, file count, series count, DICOM store, timestamps
- **Pipeline visualization** — 7-stage horizontal pipeline (Classification → PHI Scan → Protocol → Defacing → QC → BIDS → Export) with color-coded status dots
- **Action buttons** — all processing triggers, approve/reject, share, view in OHIF, review defacing, download DICOM/BIDS
- **Share form** — inline share creation for approved studies (email, note, expiry)
- **Detail tabs** — Audit Trail, Routing Log, Export Shares with per-study data

### Routing Rules Engine (`api/routing/`, `api/handler/routing.go`)

Routing rules are evaluated on every study ingest (upload complete + internal ingest). Rules are ordered by `priority` (lower = first); all matching rules fire.

**Destinations** (`/api/destinations`) — external forwarding endpoints:

| Field | Notes |
|-------|-------|
| `type` | `dicomweb` or `dimse` |
| `dicomweb_url` | Required when `type=dicomweb` (STOW-RS base URL) |
| `dicomweb_auth_header` | Optional `Authorization` value for DICOMweb destinations |
| `ae_title` / `host` / `port` | Required when `type=dimse` (remote DIMSE C-STORE destination) |

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
| `require_qc_check` | Forces `qc_required=true`, sets `qc_status=pending` |
| `require_bids_conversion` | Forces `bids_required=true`, sets `bids_status=pending` |
| `require_classification` | Forces `classification_required=true`, sets `classification_status=pending` |
| `require_protocol_check` | Forces `protocol_required=true`, sets `protocol_status=pending` |
| `require_export` | Forces `export_required=true`, sets `export_status=pending` |
| `auto_approve` | Skips manual QC, sets `status=approved` |
| `require_qa` | No-op — holds for manual review (default) |
| `reject` | Auto-rejects the study |
| `route_to` | Async forward DICOM files to a Destination (`dicomweb` via STOW-RS or `dimse` via C-STORE adapter) |

Other endpoints:
- `POST /api/routing-rules/evaluate/{studyID}` — re-evaluate rules for an existing study
- `GET /api/studies/{studyID}/routing-log` — per-study rule execution log

### Institution Management (`api/handler/institution.go`)

Institutions represent organisations that send or receive studies.

**REST API** (`/api/institutions`): CRUD + project linking.

| Field | Notes |
|-------|-------|
| `institution_type` | `sender`, `receiver`, or `both` |
| `ip_ranges` | Comma-separated CIDR blocks used for internal ingest IP auto-attribution |
| `ae_title` | DICOM AE title used for DIMSE/internal ingest attribution (`institution_ae_title`) |

**Institution-Project links** (`/api/institutions/{id}/projects`):
- `POST` — link with role (`sender`, `receiver`, `admin`)
- `DELETE /api/institutions/{id}/projects/{projectID}` — unlink

Studies carry an `institution_id` FK (nullable) for full traceability.
Internal ingest attribution order:
1. Explicit `institution_id`/`institution_ae_title` from `POST /api/ingest`
2. Fallback auto-match by request source IP against institution `ip_ranges` (most-specific CIDR wins)

Network identity normalization and validation:
- `ae_title` is trimmed and normalized to uppercase on create/update.
- `ip_ranges` entries are trimmed, deduplicated, and validated (`CIDR` or single IP).
- Invalid `ip_ranges` values are rejected with `400 Bad Request`.

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

### Authentication Middleware (`api/middleware/auth.go`)

Per-route authentication middleware that protects all admin endpoints. Supports GCP Identity-Aware Proxy (IAP) and Microsoft Azure AD (Easy Auth) as identity providers.

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `AUTH_ENABLED` | `false` | Enable authentication middleware; `false` = dev mode (auto-auth) |
| `AUTH_PROVIDER` | `auto` | Identity provider: `auto` (try all), `iap` (GCP), `azure` (Azure AD), or `aws` (ALB + Cognito) |
| `DEV_USER_EMAIL` | `dev@aegis.local` | Auto-authenticated email when `AUTH_ENABLED=false` |

**How it works:**
- `AUTH_ENABLED=false` (default, local dev): every request is auto-authenticated as `DEV_USER_EMAIL`. If that email exists in `admin_users`, uses that record; otherwise uses a synthetic admin user. Zero config needed to start developing.
- `AUTH_ENABLED=true` (production): reads identity headers from the reverse proxy:
  - **GCP IAP**: `X-Goog-Authenticated-User-Email` (format: `accounts.google.com:user@example.com`)
  - **Azure AD Easy Auth**: `X-MS-CLIENT-PRINCIPAL-NAME` (user's email)
  - **AWS ALB + Cognito**: `X-Amzn-Oidc-Data` (JWT — email extracted from payload, no signature verification needed since ALB guarantees integrity)
- Looks up the email in `admin_users` table; rejects unknown or disabled users.
- Injects `AuthUser` into request context; all audit entries now record the real user email.

**Route categories:**

| Category | Auth | Examples |
|----------|------|---------|
| Public | None | `/healthz`, `GET /api/projects`, upload portal routes, export token, DICOMweb proxy |
| Admin (read) | `RequireAuth` | `GET /api/studies`, `GET /api/institutions`, `GET /api/audit`, all read-only admin endpoints |
| Admin (write) | `RequireRole("admin")` | All POST/PUT/DELETE endpoints — create, update, delete, processing triggers, ingest, import |

**Endpoints:**
- `GET /api/auth/me` — returns the current authenticated user's `{id, email, name, role}`

**Error responses:**
- `401 {"error":"missing authentication header"}` — no identity header in production mode
- `403 {"error":"user not registered: user@example.com"}` — email not in `admin_users`
- `403 {"error":"account is disabled"}` — user exists but `enabled=false`
- `403 {"error":"insufficient permissions: requires admin role"}` — viewer attempting a write operation

### RBAC Enforcement (`api/main.go`)

The viewer role is read-only. All write endpoints (POST, PUT, DELETE) use `RequireRole("admin")`, which rejects viewer-role users with HTTP 403. GET endpoints use `RequireAuth` and are accessible to both admin and viewer roles.

**Backend** (`api/main.go`):
- `auth` = `RequireAuth(db, cfg)` — used on 19 read-only admin routes
- `adminOnly` = `RequireRole("admin", db, cfg)` — used on 33 write routes

**Frontend** (`frontend/admin-dashboard/src/App.tsx`):
- `isAdmin` derived from `currentUser?.role === 'admin'` and passed as prop to all panel components
- Viewers see all data across all tabs (studies, audit, routing, institutions, profiles, protocol templates, notifications, projects) but write-action buttons are hidden
- The **Users tab** is completely hidden for viewers (privilege escalation prevention)
- View, Download BIDS, and Review Defacing buttons remain visible for viewers (read-only actions)

**Testing RBAC locally:**
1. Create a viewer user: `curl -X POST http://localhost:8080/api/admin-users -H 'Content-Type: application/json' -d '{"email":"viewer@aegis.local","name":"Test Viewer","role":"viewer","enabled":true}'`
2. Set `DEV_USER_EMAIL=viewer@aegis.local` when running the Go API
3. Verify: GET endpoints return 200; POST/PUT/DELETE return 403
4. Open admin dashboard: write buttons hidden, Users tab hidden

### Project Settings (`api/handler/project.go`)

Projects now support full CRUD via the API and a dedicated admin dashboard tab.

**New endpoints:**
- `GET /api/projects/{id}` — fetch a single project by ID
- `PUT /api/projects/{id}` — update name, slug, description; emits `project.updated` audit entry

`POST /api/projects` now auto-generates slug from name if `slug` is omitted.

### Upload Portal QoL (`client/src/upload/client.ts`)

- **`onFileStart` callback** — `UploadOptions.onFileStart?(filename, index, total)` fires before each file's upload begins; upload portal uses it to display the current filename below the progress bar.
- **Auto-retry** — each file PUT is retried up to 3× with 1 s / 2 s / 4 s exponential backoff before failing. Transparent to callers.

### Defacing Service (`defacing/`, `api/handler/deface.go`)

Server-side facial feature removal from head/brain DICOM imaging. Multiple pluggable backends with automatic fallback. Runs as a separate Python FastAPI service, called asynchronously from the Go API.

**Running locally:**
```bash
cd defacing
pip install -r requirements.txt
pip install deepdefacer          # optional: enables DeepDefacer backend
uvicorn app.main:app --port 8081
# Then set DEFACING_SERVICE_URL=http://localhost:8081 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `DEFACING_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = studies stay queued |
| `DEFACE_TOOL` | `auto` | `auto\|mri_reface\|deepdefacer\|mri_deface\|nibabel` |
| `DEEPDEFACER_GPU` | `false` | Enable GPU for DeepDefacer (requires `deepdefacer[gpu]` + CUDA) |
| `MRI_DEFACE_BIN` | `mri_deface` | Path to mri_deface binary |
| `MRI_DEFACE_BRAIN` | `/opt/mri_deface/talairach_mixed_with_skull.gca` | Brain atlas |
| `MRI_DEFACE_FACE` | `/opt/mri_deface/face.gca` | Face atlas |
| `MRI_REFACE_BIN` | `mri_reface` | Path to mri_reface binary |
| `DCM2NIIX_BIN` | `dcm2niix` | Path to dcm2niix binary |

**Auto-selection priority:** mri_reface > deepdefacer > mri_deface > nibabel (first available wins)

**Backends:**

| Backend | Docker Size | Speed | Quality | License | Notes |
|---------|------------|-------|---------|---------|-------|
| `mri_reface` | ~2-4 GB | >1 min | Best | Non-commercial | MRI + PET + CT; requires MATLAB Runtime |
| `deepdefacer` | ~500 MB | ~1-2 min | Good | Research-friendly | 3D U-Net, fastest DL approach; pip-installable |
| `mri_deface` | ~500 MB | 2-10 min | Good | Free | FreeSurfer standalone, MRI only |
| `nibabel` | 0 | <1 min | Crude | MIT | Dev/testing only (zeros anterior 30%) |

**Dockerfile build args:**
- `INCLUDE_DEEPDEFACER=true` — installs DeepDefacer + TensorFlow (~500 MB)
- `INCLUDE_MRI_DEFACE=true` — downloads FreeSurfer mri_deface + atlas (~500 MB)
- `INCLUDE_MRI_REFACE=true` — commented out; requires MATLAB Runtime (~3 GB)

**Study fields:**
- `defacing_required` — boolean flag, set by `require_defacing` routing rule action
- Study `status` transitions: `received` > `defacing` > `defaced` (or back to `received` on failure)
- `dicom_store` — switches from `raw` to `clean` after successful defacing

**API:**
- `POST /api/studies/{studyUID}/trigger-deface` — trigger defacing (returns 202 Accepted, runs async)

See `docs/research/mri-defacing-tools-comparison.md` for detailed tool comparison with citations.

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
| `PHI_TOOL` | `auto` | Backend selection: `auto`, `google_vision`, `aws_textract`, or `tesseract` |
| `PHI_CONFIDENCE_THRESHOLD` | `0.4` | Minimum OCR confidence (0.0–1.0) |
| `PHI_MIN_TEXT_LENGTH` | `3` | Minimum text length to report |

**Dockerfile build args** (cloud SDKs installed conditionally):

| Arg | Default | Installs |
|-----|---------|----------|
| `INCLUDE_GOOGLE_VISION` | `false` | `google-cloud-vision Pillow numpy` (~50 MB) |
| `INCLUDE_AWS_TEXTRACT` | `false` | `boto3 Pillow numpy` (~50 MB) |

**Study fields:**
- `phi_scan_required` — boolean flag, set by `require_phi_scan` routing rule action
- `phi_scan_status` — `''` (not required), `pending`, `scanning`, `clean`, `flagged`, `failed`

**API:**
- `POST /api/studies/{studyUID}/phi-scan` — trigger PHI scan (returns 202 Accepted, runs async)

**Pluggable backends (auto-selection priority: google_vision > aws_textract > tesseract):**

| Backend | SDK | Accuracy | Notes |
|---------|-----|----------|-------|
| `google_vision` | `google-cloud-vision` | Best | Cloud Vision `text_detection`; Application Default Credentials |
| `aws_textract` | `boto3` | Good | Textract `detect_document_text`; IAM roles or `AWS_ACCESS_KEY_ID` |
| `tesseract` | `pytesseract` | Baseline | Local OCR; requires `tesseract-ocr` binary installed |

All backends share `pixel_utils.py` (DICOM pixel extraction → PIL Image). Cloud SDKs are optional — installed via Dockerfile build args.

**Pipeline:**
1. Routing rule with action `require_phi_scan` sets `phi_scan_required=true` and `phi_scan_status=pending`
2. Admin clicks "Scan for PHI" → Go handler sets status to `scanning` and dispatches to Python service
3. Python service reads each DICOM file, extracts pixel data, runs OCR (selected backend)
4. Results returned: `phi_scan_status` set to `clean` (no text found) or `flagged` (text detected)
5. Findings stored as JSONB in audit trail (`phi_scan.complete` entries)
6. Admin can still approve flagged studies (the flag is informational)

**Admin dashboard:**
- PHI Scan column with status badge (pending/scanning/clean/flagged/failed)
- "Scan for PHI" button for pending studies
- `require_phi_scan` option in routing rules action dropdown

### QC Automation Service (`qc-service/`, `api/handler/qc_check.go`)

Automated image quality checks on DICOM studies — detects inconsistent slice dimensions, low SNR, missing slices, and incomplete coverage. Runs as a separate Python FastAPI service, called asynchronously from the Go API — same pattern as the defacing and PHI detection services.

**Running locally:**
```bash
cd qc-service
pip install -r requirements.txt
uvicorn app.main:app --port 8083
# Then set QC_SERVICE_URL=http://localhost:8083 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `QC_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = studies stay in "pending" |
| `QC_TOOL` | `auto` | Backend selection: `auto` or `basic` |
| `QC_SNR_THRESHOLD` | `10.0` | Minimum SNR before warning (signal mean / noise stddev) |
| `QC_GAP_RATIO` | `2.0` | Gap-to-median-spacing ratio that triggers missing slice warning |

**Study fields:**
- `qc_required` — boolean flag, set by `require_qc_check` routing rule action
- `qc_status` — `''` (not required), `pending`, `checking`, `pass`, `warn`, `fail`, `failed`

**API:**
- `POST /api/studies/{studyUID}/qc-check` — trigger QC check (returns 202 Accepted, runs async)

**5 QC checks** (basic backend, pydicom + numpy):
1. **File integrity** — all files parse, required DICOM tags present
2. **Slice consistency** — uniform Rows/Columns/PixelSpacing across all slices
3. **SNR estimation** — signal mean / corner noise stddev, warn if below threshold
4. **Coverage completeness** — slice count vs expected minimum for body part
5. **Missing slices** — gaps in slice position (>2× median spacing)

**Pipeline:**
1. Routing rule with action `require_qc_check` sets `qc_required=true` and `qc_status=pending`
2. Admin clicks "Run QC" → Go handler sets status to `checking` and dispatches to Python service
3. Python service reads each DICOM file, runs all 5 checks
4. Results returned: `qc_status` set to `pass`, `warn` (non-critical issues), or `fail` (critical issues)
5. Findings stored as JSONB in audit trail (`qc_check.complete` entries with `quality_issues` array)
6. Admin can still approve warn/fail studies (the status is informational)

**Admin dashboard:**
- QC column with status badge (pending/checking/pass/warn/fail/failed)
- "Run QC" button for pending studies
- `require_qc_check` option in routing rules action dropdown

### NIfTI/BIDS Conversion Service (`bids-service/`, `api/handler/bids_convert.go`)

Converts DICOM studies to NIfTI format with BIDS-compliant directory structure and JSON sidecar metadata. Uses dcm2niix for the conversion. Runs as a separate Python FastAPI service, called asynchronously from the Go API — same pattern as the defacing, PHI detection, and QC services.

**Running locally:**
```bash
cd bids-service
pip install -r requirements.txt
# Requires dcm2niix installed: brew install dcm2niix (macOS) or apt-get install dcm2niix (Linux)
uvicorn app.main:app --port 8084
# Then set BIDS_SERVICE_URL=http://localhost:8084 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `BIDS_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = studies stay in "pending" |
| `BIDS_TOOL` | `auto` | Backend selection: `auto` or `dcm2niix` |
| `DCM2NIIX_BIN` | `dcm2niix` | Path to dcm2niix binary |

**Study fields:**
- `bids_required` — boolean flag, set by `require_bids_conversion` routing rule action
- `bids_status` — `''` (not required), `pending`, `converting`, `complete`, `failed`

**API:**
- `POST /api/studies/{studyUID}/bids-convert` — trigger BIDS conversion (returns 202 Accepted, runs async)
- `GET /api/studies/{studyUID}/bids-download` — download BIDS output as zip archive

**BIDS output structure:**
```
bids/{studyUID}/
├── dataset_description.json
├── participants.tsv
└── sub-<hash8>/
    └── anat/  (or func/, dwi/, perf/, ct/, pet/)
        ├── sub-<hash8>_T1w.nii.gz
        └── sub-<hash8>_T1w.json
```

Subject label = first 8 chars of SHA-256 hash of StudyInstanceUID (privacy-preserving). Series are classified to BIDS datatypes/suffixes by matching ProtocolName and SeriesDescription against known patterns.

**Pipeline:**
1. Routing rule with action `require_bids_conversion` sets `bids_required=true` and `bids_status=pending`
2. Admin clicks "Convert to BIDS" → Go handler sets status to `converting` and dispatches to Python service
3. Python service groups DICOM files by series, runs dcm2niix per series with BIDS flags
4. Output organized into BIDS directory structure with sidecar JSON metadata
5. Results returned: `bids_status` set to `complete` or `failed`
6. Admin clicks "Download BIDS" → browser downloads zip archive of the BIDS output

**Admin dashboard:**
- BIDS column with status badge (pending/converting/complete/failed)
- "Convert to BIDS" button for pending studies
- "Download BIDS" link for completed studies
- `require_bids_conversion` option in routing rules action dropdown

### Metadata Classification Service (`classification-service/`, `api/handler/classification.go`)

Classifies study modality and body part by reading DICOM headers. Studies can arrive with empty `modality` and `body_part` when DICOM tags are missing — this service fills them in using heuristic tag analysis (free, instant) or cloud image-based label inference (when tags are missing), then **re-evaluates routing rules** so modality/body_part-dependent rules fire correctly.

**Running locally:**
```bash
cd classification-service
pip install -r requirements.txt
uvicorn app.main:app --port 8085
# Then set CLASSIFICATION_SERVICE_URL=http://localhost:8085 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `CLASSIFICATION_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = studies stay in "pending" |
| `CLASSIFY_TOOL` | `auto` | Backend selection: `auto`, `google_vision`, `aws_rekognition`, or `heuristic` |
| `CLASSIFY_CONFIDENCE_THRESHOLD` | `0.5` | Minimum confidence (0.0–1.0) to update metadata |

**Dockerfile build args** (cloud SDKs installed conditionally):

| Arg | Default | Installs |
|-----|---------|----------|
| `INCLUDE_GOOGLE_VISION` | `false` | `google-cloud-vision Pillow numpy` (~50 MB) |
| `INCLUDE_AWS_REKOGNITION` | `false` | `boto3 Pillow numpy` (~50 MB) |

**Study fields:**
- `classification_required` — boolean flag, set by `require_classification` routing rule action
- `classification_status` — `''` (not required), `pending`, `classifying`, `classified`, `failed`

**API:**
- `POST /api/studies/{studyUID}/classify` — trigger classification (returns 202 Accepted, runs async)

**Classification strategy** (priority order):
1. **Direct DICOM tags** — `Modality` (0008,0060) + `BodyPartExamined` (0018,0015) → confidence 0.95
2. **SOP Class UID** (0008,0016) → modality mapping (CT, MR, PT, US, CR, etc.) → confidence 0.90
3. **SeriesDescription / ProtocolName** → body part regex (HEAD, CHEST, ABDOMEN, SPINE, EXTREMITY, NECK) → confidence 0.75
4. **StudyDescription** → same patterns → confidence 0.65
5. **Cloud image inference** (if heuristic confidence < threshold) — render DICOM pixels, run label detection → confidence 0.70
6. **Fallback** → empty (inconclusive)

Cloud backends inherit from HeuristicBackend and only call the API when heuristic strategies 1-4 produce low confidence. This avoids API cost when DICOM tags are present.

**Pluggable backends (auto-selection priority: google_vision > aws_rekognition > heuristic):**

| Backend | SDK | Notes |
|---------|-----|-------|
| `google_vision` | `google-cloud-vision` | Cloud Vision `label_detection` (20 labels) → body_part/modality mapping; Application Default Credentials |
| `aws_rekognition` | `boto3` | Rekognition `detect_labels` → same mapping; IAM roles or `AWS_ACCESS_KEY_ID` |
| `heuristic` | *(none)* | Local DICOM tag analysis only (strategies 1-4, no cloud dependencies) |

All cloud backends share `pixel_utils.py` (DICOM pixel extraction → PIL Image). Cloud SDKs are optional — installed via Dockerfile build args.

**Pipeline:**
1. Routing rule with action `require_classification` sets `classification_required=true` and `classification_status=pending`
2. Admin clicks "Classify" → Go handler sets status to `classifying` and dispatches to Python service
3. Python service reads DICOM files, applies heuristic classification (then cloud inference if needed)
4. Results returned: if confidence >= 0.5, study `modality` and `body_part` are updated
5. `classification_status` set to `classified` or `failed`
6. **Routing rules re-evaluated** — downstream rules (e.g. `require_defacing` for HEAD studies) now fire
7. Audit log entries: `classification.triggered`, `classification.complete`/`classification.failed`

**Admin dashboard:**
- Classification column with status badge (pending/classifying/classified/failed)
- "Classify" button for pending studies
- `require_classification` option in routing rules action dropdown

### MRI Protocol Compliance Service (`protocol-service/`, `api/handler/protocol_check.go`)

Verifies that DICOM acquisition parameters (TR, TE, flip angle, slice thickness, resolution, etc.) match expected values defined in per-project protocol templates. Templates are keyed by scanner manufacturer, model, software version, and sequence type. Supports both Classic (single-frame, flat tags) and Enhanced DICOM (multi-frame, nested functional group sequences). Runs as a separate Python FastAPI service, called asynchronously from the Go API.

**Running locally:**
```bash
cd protocol-service
pip install -r requirements.txt
uvicorn app.main:app --port 8086
# Then set PROTOCOL_SERVICE_URL=http://localhost:8086 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `PROTOCOL_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = studies stay in "pending" |
| `PROTOCOL_TOOL` | `auto` | Backend selection: `auto` or `basic` |
| `PROTOCOL_DEFAULT_TOLERANCE` | `5.0` | Default percentage tolerance for numeric parameter comparison |

**Study fields:**
- `protocol_required` — boolean flag, set by `require_protocol_check` routing rule action
- `protocol_status` — `''` (not required), `pending`, `checking`, `compliant`, `minor_deviations`, `non_compliant`, `failed`

**API:**
- `POST /api/studies/{studyUID}/protocol-check` — trigger protocol check (returns 202 Accepted, runs async)

**Protocol Templates** (`/api/projects/{projectID}/protocol-templates`):

| Field | Notes |
|-------|-------|
| `manufacturer` | Scanner manufacturer, e.g. `SIEMENS` (empty = any) |
| `model` | Scanner model, e.g. `MAGNETOM Prisma` (empty = any) |
| `software_version` | Software version, e.g. `VE11C` (empty = any) |
| `sequence_type` | Pulse sequence identifier, e.g. `T1w_MPRAGE`, `FLAIR`, `DWI` |
| `rules` | JSONB array of parameter rules (tag keyword, target, tolerance, match type, severity) |

**Parameter rule match types:**
- `numeric` — percentage tolerance (default 5%), optional absolute tolerance override
- `exact` — string equality
- `contains_all` — all expected values present in actual list
- `range` — value within [min, max]

**Severity levels:** `critical` (→ non_compliant), `warning` (→ minor_deviations), `info` (→ compliant)

**DICOM format support:**
- **Classic DICOM** — parameters read from top-level tags (RepetitionTime, EchoTime, FlipAngle, etc.)
- **Enhanced DICOM** — detected via SOPClassUID `1.2.840.10008.5.1.4.1.1.4.1`; parameters extracted from `SharedFunctionalGroupsSequence` and `PerFrameFunctionalGroupsSequence` nested sequences

**Pipeline:**
1. Routing rule with action `require_protocol_check` sets `protocol_required=true` and `protocol_status=pending`
2. Admin clicks "Check Protocol" → Go handler loads matching templates for study's project, sets status to `checking`, dispatches to Python service with aggregated rules
3. Python service reads DICOM files, extracts parameters (Classic or Enhanced), compares against rules
4. Results returned: `protocol_status` set to `compliant`, `minor_deviations`, `non_compliant`, or `failed`
5. Findings stored as JSONB in audit trail (`protocol_check.complete` entries with per-parameter findings)
6. Admin can still approve non-compliant studies (the status is informational)

**Admin dashboard:**
- Protocol column with status badge (pending/checking/compliant/minor_deviations/non_compliant/failed)
- "Check Protocol" button for pending studies
- `require_protocol_check` option in routing rules action dropdown
- **Protocol Templates tab** — full CRUD for per-project templates with rules editor

### DIMSE Receiver Service (`dimse-receiver/`)

Receives studies from PACS systems over DICOM network protocol (DIMSE C-STORE SCP). On each C-STORE it writes files to `dicom/raw/{studyUID}/{index}.dcm` in shared storage. When the DICOM association closes (`EVT_RELEASED`), it calls `POST /api/ingest` so the normal AEGIS routing + pipeline flow starts. The ingest payload includes `institution_ae_title` (calling AE title) for institution auto-attribution; `institution_id` can also be set explicitly.

**Running locally:**
```bash
cd dimse-receiver
pip install -r requirements.txt
uvicorn app.main:app --port 8087
# DICOM SCP listens on DIMSE_PORT (default 11112) in a background thread.
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `DIMSE_AE_TITLE` | `AEGIS` | SCP AE title for PACS associations |
| `DIMSE_PORT` | `11112` | DICOM C-STORE/C-ECHO listening port |
| `DIMSE_DATA_DIR` | `/app/data` | Shared storage mount (same as Go API) |
| `API_URL` | `http://api:8080` | Go API base URL for ingest calls |
| `DIMSE_PROJECT_SLUG` | `default` | Project slug sent to `/api/ingest` |
| `DIMSE_INSTITUTION_ID` | *(empty)* | Optional fixed institution UUID sent as `institution_id` |
| `DIMSE_INGEST_TIMEOUT` | `30` | HTTP timeout (seconds) for ingest call |
| `DIMSE_MAX_ASSOCIATIONS` | `10` | Max simultaneous DICOM associations |

### Batch Import CLI (`api/cmd/import/`)

CLI tool for importing DICOM files from a local directory into AEGIS. Used for bulk historical data migration. Files are assumed already de-identified — the import tool does NOT apply de-identification.

**Running:**
```bash
cd api && go run ./cmd/import --dir /path/to/dicom --project default
```

**Building:**
```bash
cd api && go build -o aegis-import ./cmd/import
./aegis-import --dir /path/to/dicom --project default
```

**CLI Flags:**

| Flag | Default | Notes |
|------|---------|-------|
| `--dir` | *(required)* | Directory containing DICOM files to import |
| `--project` | `default` | Project slug |
| `--institution` | *(empty)* | Institution UUID (optional) |
| `--source` | `internal` | `internal` or `external` |
| `--dry-run` | `false` | Scan and report without importing |

Uses same env vars as the API (`DATABASE_URL`, `STORAGE_MODE`, `LOCAL_STORAGE_DIR`).

**How it works:**
1. Recursively scans `--dir` for `.dcm` files
2. Parses DICOM headers (StudyInstanceUID, Modality, BodyPart, StudyDescription, SeriesInstanceUID) using `suyashkumar/dicom` with `SkipPixelData()` for performance
3. Groups files by StudyInstanceUID
4. For each study: creates upload session + study record, copies files to `dicom/raw/{studyUID}/`, evaluates routing rules
5. Duplicate StudyInstanceUIDs are rejected (unique constraint) — safe to re-run

**API endpoint:** `POST /api/import/batch` — accepts `{"dir","project_slug","institution_id","source","dry_run"}`, returns `{files_scanned, files_skipped, studies_created, studies_failed, errors, study_ids}`.

### Automated Processing Pipeline (`api/handler/pipeline.go`)

After routing rules evaluate (upload complete, internal ingest, batch import), the pipeline orchestrator automatically dispatches all required processing services in the correct order. No manual button clicks needed — studies flow through the pipeline hands-free.

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `PIPELINE_AUTO` | `true` | Set to `"false"` to disable auto-dispatch (manual-only mode) |

**Dependency graph (3 phases):**

```
Phase 0: Classification (blocks — fills modality/body_part, re-evaluates routing)
    ↓
Phase 1: PHI scan + Protocol check + Defacing (parallel, raw files)
    ↓
Phase 2: QC check + BIDS conversion (after defacing, final files)
```

- **Phase 0**: Classification must complete first — it fills `modality`/`body_part` from DICOM headers, then re-evaluates routing rules which may add new requirements (e.g. `require_defacing` for HEAD studies)
- **Phase 1**: PHI scan, protocol check, and defacing run in parallel on raw files. Each service is dispatched only if its URL is configured.
- **Phase 2**: QC and BIDS run after defacing completes (if defacing is required). They operate on the final `dicom_store` (clean after defacing, raw otherwise).

**How it works:**
- `AdvancePipeline(ctx, studyID)` is called after routing rules evaluate and after each service completes
- Uses atomic SQL "claim" queries (`UPDATE ... WHERE status='pending'`) to prevent duplicate dispatches
- Services that fail stop their branch; admin can re-trigger manually via dashboard buttons
- Manual trigger buttons continue to work regardless of `PIPELINE_AUTO` setting
- Services with no URL configured are silently skipped (study stays in "pending")

**Audit trail:** Each auto-dispatch creates a `pipeline.dispatch` audit entry recording which service was dispatched and for which study.

### Study Export & DICOM Download (`api/handler/export.go`, `api/handler/dicom_download.go`, `api/handler/export_forward.go`)

Full export workflow for approved studies: admin DICOM download, token-authenticated recipient download, and automated forwarding to external destinations.

**DICOM Download (admin):**
- `GET /api/studies/{studyUID}/dicom-download` — streams all DICOM files as a zip archive
- Requires auth (read-only, like BIDS download); study must be `approved`
- Uses `storage.Storage` interface (cloud-agnostic — works with local, S3, or GCS)

**DICOM Download (export share):**
- `GET /api/export/{token}/download` — token-authenticated zip download (no login required)
- Same token validation as `GET /api/export/{token}` (SHA-256 hash, expiry, revocation)
- Logs to `export_downloads` table + audit trail

**Export share redemption** (`GET /api/export/{token}`) — enhanced response includes:
- `body_part`, `study_description`, `instance_count`, `note`, `created_by`, `download_url`
- Used by the export portal to display study info and download link

**Export forwarding** (`route_to` destinations):
- `POST /api/studies/{studyUID}/trigger-export` — manual trigger (admin only, study must be approved + export_required)
- Background goroutine finds matching `route_to` rules and forwards DICOM files to each destination
- `dicomweb` destinations: STOW-RS `multipart/related; type="application/dicom"` (streamed via `io.Pipe()`)
- `dimse` destinations: calls `dimse-receiver` adapter, which sends C-STORE to remote AE Title/host/port
- Auto-dispatches on study approval when `export_required=true` and `export_status=pending`

**Study fields:**
- `export_required` — boolean flag, set by `require_export` routing rule action
- `export_status` — `''` (not required), `pending`, `exporting`, `exported`, `failed`

**Admin dashboard:**
- Export column with status badge (pending/exporting/exported/failed)
- "Download DICOM" button for approved studies (read-only — visible to viewers)
- "Export" trigger button for admin (approved + export_required + pending/failed)
- `require_export` option in routing rules action dropdown

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

## CI (GitHub Actions)

`.github/workflows/ci.yml` runs on PRs to `develop` and `main`:

| Job | What it checks |
|-----|---------------|
| `go` | `go build ./...` + `go vet ./...` |
| `go-test` | `go test -race -v -count=1 ./...` (~120 tests) |
| `python` (7× matrix) | `py_compile` on all `.py` files per service |
| `python-test` (7× matrix) | `pytest -v --tb=short` per service (~206 tests total) |
| `frontend` (5× matrix) | `npx tsc --noEmit` (client, upload-portal, admin-dashboard, export-portal, landing) |
| `docker` (8× matrix) | `docker build` for all service images |

### Python Sidecar Testing

Each sidecar has `requirements-test.txt` (pytest + httpx) and a `tests/` directory:

```bash
cd {service} && pip install -r requirements.txt -r requirements-test.txt && pytest -v
```

| Service | Tests | Coverage |
|---------|-------|----------|
| classification-service | 49 | Heuristic classification (5 strategies), SOP UID mapping, body part regex, Cloud Vision/Rekognition label mapping, cloud backend inheritance, pixel_utils, endpoint tests |
| dimse-receiver | 22 | C-STORE file write/indexing, EVT_RELEASED ingest trigger, C-ECHO, DIMSE forward endpoint mapping, sender status/path helpers, ingest payload/error handling |
| protocol-service | 29 | Classic + Enhanced DICOM extraction, 4 match types (numeric/exact/contains_all/range), severity aggregation |
| qc-service | 28 | 5 QC checks (file integrity, slice consistency, SNR, coverage, missing slices), controlled pixel arrays |
| defacing | 26 | Pipeline (group_by_series, should_deface_series, run_pipeline), nibabel backend, AP axis detection |
| phi-detection | 35 | Windowing, uint8 normalization, mock Tesseract OCR, Cloud Vision/Textract OCR, pixel_utils, multi-file detection |
| bids-service | 17 | Series classification (T1w/FLAIR/bold/DWI/ASL/PET/CT), subject label hashing, mock dcm2niix |

All tests use **synthetic DICOM files** generated via pydicom — no test data on disk. External tools (tesseract, dcm2niix, mri_deface) are mocked.

## Makefile

Common dev commands available via `make`:

| Target | Description |
|--------|-------------|
| `make up` | `docker compose up -d` (start all services) |
| `make down` | `docker compose down` |
| `make clean` | `docker compose down -v` (destroy volumes) |
| `make build` | `docker compose build` |
| `make api` | Run Go API locally (`go run .`) |
| `make lint` | Lint all languages (Go vet, Python py_compile, TypeScript tsc) |
| `make check` | `curl /healthz` with pretty JSON output |
| `make logs` | `docker compose logs -f` |

## Conventions

- Product naming: always use **Anonymization & Exchange Gateway for Imaging Studies** for first mention, then **AEGIS** thereafter.
- Go: standard library preferred, minimal dependencies
- Frontend: functional React components, TypeScript strict mode
- Terraform: one module per logical resource group
- Docker: distroless for Go, slim for Python
- All services expose `/healthz` for health checks

## Current Phase

Phase 1 (MVP): Foundation — Terraform, Go API, Upload Portal, Admin Dashboard
See AEGIS_Architecture.md for full phased plan.
