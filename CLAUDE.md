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

Operational docs:
- `docs/dicom-conformance.md` — formal DICOM conformance statement: supported SOP classes, transfer syntaxes, DICOMweb (QIDO-RS/STOW-RS/WADO-RS), DIMSE (C-STORE SCP, C-ECHO SCP, C-STORE SCU), de-identification profile, limitations
- `docs/runbooks/alert-response.md` — per-alert incident response for all 9 Cloud Monitoring alert policies: triage steps, remediation commands, escalation paths
- `docs/runbooks/incident-response.md` — general incident management workflow (severity levels, communication, post-mortem)
- `docs/runbooks/secret-rotation.md` — secret rotation procedures for DB password, API keys, and DIMSE operator key
- `docs/runbooks/cross-cloud-routing.md` — GCP→AWS cross-cloud DICOM routing setup (STOW-RS + DIMSE), step-by-step config, troubleshooting

## Repository Structure (Monorepo)

```
aegis/
├── terraform/project/    # GCP project bootstrap (IAM, KMS, VPC-SC)
├── terraform/infra/      # GCP infrastructure (Cloud Run, Healthcare API, Cloud Armor)
├── terraform/aws/        # AWS infrastructure (ECS Fargate, S3, RDS, ALB)
├── api/                  # Go backend — upload orchestration, DICOMweb proxy
├── frontend/
│   ├── upload-portal/    # React — public-facing upload + anonymization UI
│   ├── admin-dashboard/  # React — internal QC, Weasis viewer, study management
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
- **Viewer**: Weasis DWV (embedded in admin dashboard)
- **AI/ML**: Pluggable — local backends (Tesseract OCR, pydicom heuristics) or cloud AI (Google Cloud Vision, AWS Textract/Rekognition)
- **Email**: Standard SMTP (works with any provider). Dev: Mailpit.
- **Infrastructure**: Terraform (GCP and AWS modules), Docker Compose for local dev
- **Auth**: Multi-provider — GCP IAP, Azure AD Easy Auth, AWS ALB + Cognito; dev mode auto-auth

### Multi-Cloud Support

AEGIS is cloud-agnostic at the application layer. The same Go API, Python sidecars, and React frontends run on any cloud or on-premises.

| Component | GCP | AWS | Azure | Local Dev |
|-----------|-----|-----|-------|-----------|
| **File storage** | GCS (`STORAGE_MODE=gcs`) | S3 (`STORAGE_MODE=s3`) | Azure Blob Storage (`STORAGE_MODE=azure`) | Filesystem (`STORAGE_MODE=local`) |
| **Database** | Cloud SQL | RDS | Azure Database for PostgreSQL - Flexible Server | Docker postgres |
| **Containers** | Cloud Run | ECS Fargate | Azure Container Apps | Docker Compose |
| **Auth** | IAP (`AUTH_PROVIDER=iap`) | ALB + Cognito (`AUTH_PROVIDER=aws`) | Easy Auth (`AUTH_PROVIDER=azure`) | Auto-auth (`AUTH_ENABLED=false`) |
| **Terraform** | `terraform/project/` + `terraform/infra/` | `terraform/aws/` | `terraform/azure/` | N/A |

S3-compatible stores (MinIO, LocalStack) are supported via the `S3_ENDPOINT` env var.

Azure env vars (only used when `STORAGE_MODE=azure`):

| Var | Default | Notes |
|-----|---------|-------|
| `AZURE_STORAGE_ACCOUNT` | *(required)* | Storage account name |
| `AZURE_STORAGE_CONTAINER` | `dicom` | Blob container name |

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

Core runtime env vars:

| Var | Default | Notes |
|-----|---------|-------|
| `PORT` | `8080` | API listen port |
| `DATABASE_URL` | *(empty)* | Optional explicit Postgres DSN; if empty, API builds DSN from `DB_*` vars |
| `DB_HOST` | `localhost` | Used when `DATABASE_URL` is empty |
| `DB_PORT` | `5432` | Used when `DATABASE_URL` is empty |
| `DB_NAME` | `aegis` | Used when `DATABASE_URL` is empty |
| `DB_USER` | `aegis` | Used when `DATABASE_URL` is empty |
| `DB_PASSWORD` | `aegis` | Used when `DATABASE_URL` is empty; in GCP Cloud Run this is injected from Secret Manager |
| `APP_TIMEZONE` | `UTC` | Applies DB session timezone (`SET TimeZone`) and uses UTC log timestamps |

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
| `SMTP_FROM` | `noreply@aegisimaging.ai` | Envelope sender address |
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

Static marketing site for aegisimaging.ai. Deployed on GCP Cloud Run (`aegis-prod-landing`).

**Invite Code Gate** — server-side access control for the private beta. When `VITE_INVITE_GATE_ENABLED=true` is baked in at build time, all visitors see a code entry form before the site content. Codes are validated via `POST /api/invite/validate` — no secret is stored in the client bundle.

| Component | Description |
|-----------|-------------|
| `frontend/landing/src/hooks/useInviteCode.ts` | State hook — calls API for validation, persists admission in localStorage |
| `frontend/landing/src/components/InviteGate.tsx` | Full-screen gate UI shown to unadmitted visitors |
| `api/handler/invite_code.go` | Go handler for validate (public) + CRUD (admin-only) |
| `api/model/invite_code.go` | Model — random code generation (`XXXX-XXXX-XXXX` format), validate+record usage |
| Migration 036 | `invite_codes` table (`id`, `code`, `label`, `enabled`, `created_at`, `used_at`, `used_by_ip`) |

**Invite code API endpoints:**
- `POST /api/invite/validate` — public, rate-limited; `{"code":"..."}` → `{"valid":true/false}`
- `GET /api/invite-codes` — admin; list all codes with usage stats
- `POST /api/invite-codes` — admin; `{"label":"Dr. Smith"}` → generates new `XXXX-XXXX-XXXX` code
- `POST /api/invite-codes/{id}/revoke` — admin; disables a code (keeps record)
- `DELETE /api/invite-codes/{id}` — admin; permanently removes a code

**Invite request system** — prospective users can submit an access request from the landing page (`POST /api/invite/request`, public + rate-limited). Requests are stored in the `invite_requests` table (migration 040) and surface in the admin dashboard "Access Requests" sub-tab under Invite Codes.

| Component | Description |
|-----------|-------------|
| `api/handler/invite_request.go` | `RequestInvite` (public submit), `ListInviteRequestsAdmin`, `ApproveInviteRequestAdmin`, `DenyInviteRequestAdmin` |
| `api/model/invite_request.go` | Model: `CreateInviteRequest`, `GetInviteRequest`, `ListInviteRequests`, `ApproveInviteRequest`, `DenyInviteRequest` |
| Migration 040 | `invite_requests` table (`id`, `name`, `email`, `org`, `message`, `status`, `ip`, `created_at`, `reviewed_at`, `reviewed_by`, `invite_code_id`) |

**Invite request API endpoints:**
- `POST /api/invite/request` — public, rate-limited; submits a request and sends an email notification to admins
- `GET /api/invite/requests` — admin; list requests (filterable by `status=pending|approved|denied|all`), returns `{requests, total}`
- `POST /api/invite/requests/{id}/approve` — admin-only; creates a new invite code, emails the requester with their code, marks the request approved; returns `{status, invite_code}`
- `POST /api/invite/requests/{id}/deny` — admin-only; marks request denied; returns `{status}`

**Admin dashboard:** "Invite Codes" tab has two sub-tabs — "Invite Codes" (existing code management) and "Access Requests" (pending/approved/denied request list with Approve/Deny action buttons). Status badges use colorblind-friendly teal (approved) and orange (denied) palette.

**Dockerfile build arg:**

| Arg | Default | Notes |
|-----|---------|-------|
| `VITE_INVITE_GATE_ENABLED` | `false` | Set to `true` to enable the gate; `false` = open access (dev default) |
| `VITE_API_BASE_URL` | `https://api.aegisimaging.ai` | API base for invite validation calls |

**Deploy the gated landing page:**
```bash
docker build --platform linux/amd64 \
  --build-arg VITE_INVITE_GATE_ENABLED=true \
  --build-arg VITE_API_BASE_URL=https://api.aegisimaging.ai \
  -f frontend/landing/Dockerfile \
  -t us-central1-docker.pkg.dev/aegis-prod-488120/aegis-services/landing:gated .
docker push us-central1-docker.pkg.dev/aegis-prod-488120/aegis-services/landing:gated
gcloud run services update aegis-prod-landing --image .../landing:gated --region us-central1
```

**Generate invite codes (after deployment):**
```bash
# Create a code for a specific person
curl -X POST https://api.aegisimaging.ai/api/invite-codes \
  -H "Authorization: Bearer <api_key>" \
  -H "Content-Type: application/json" \
  -d '{"label": "Dr. Smith – Stanford"}'
# Returns: {"id":"...","code":"ABCD-EFGH-IJKL","label":"...","enabled":true,...}

# Send the invite URL: https://aegisimaging.ai/?invite=ABCD-EFGH-IJKL
```

**CORS:** `aegisimaging.ai` and `www.aegisimaging.ai` are included in the default `ALLOWED_ORIGINS` so the invite validation call works cross-origin.

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

`docker compose up` starts the entire platform: PostgreSQL, Mailpit, Weasis, Go API, and all 7 Python sidecar services. All services share a named `aegis-data` volume for DICOM file exchange.

```bash
docker compose up -d          # start everything (background)
docker compose up weasis      # start Weasis only
docker compose down           # stop all (data persists)
docker compose down -v        # stop all + destroy volumes
```

| Service | Port | Notes |
|---------|------|-------|
| postgres | 5432 | PostgreSQL 15, data in `pgdata` volume |
| mailpit | 1025 (SMTP) / 8025 (UI) | Email capture for dev |
| weasis | 3005 | Weasis DWV viewer |
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

### Weasis DWV Viewer
Weasis (DWV — DICOM Web Viewer) runs as a Docker container on `:3005`, configured to load DICOM images via the Go API's DICOMweb proxy. It is the sole viewer used in the admin dashboard.

**URL params**: `?studyUID=<UID>` (required) and `?store=raw|clean` (optional, default `clean`). When `store=raw`, all DICOMweb fetches use `/dicomweb-raw/` instead of `/dicomweb/`.

**Build arg** (`VITE_WEASIS_BASE_URL`): baked into admin-dashboard at build time via `--build-arg VITE_WEASIS_BASE_URL=<url>`.

**DICOMweb proxy** (`api/handler/dicomweb.go`) — minimal QIDO-RS + WADO-RS, no DICOM library:
- `GET /dicomweb/studies` — list studies from DB
- `GET /dicomweb/studies/{studyUID}/series` — single fake series per study
- `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances` — enumerate instances by file count
- `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}` — stream DICOM bytes from `dicom_store`
- `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}/metadata` — WADO-RS instance metadata (DICOMweb JSON, no pixel data)

**Raw DICOMweb proxy** — same routes under `/dicomweb-raw/*`, but WADO-RS always reads from `dicom/raw/` regardless of `dicom_store`. Used by the Weasis viewer with `?store=raw` for defacing review. Includes `/metadata` route.

SOPInstanceUID format: `{studyUID}.1.{fileIndex}` (index maps to `dicom/{store}/{studyUID}/{index}.dcm`).

In the admin dashboard:
- Each study row has a **View** button (inline iframe) and an **Open in new tab ↗** link.
- Head studies with `defacing_required=true` and `status=defaced|approved` show a **Review defacing** button that opens a side-by-side Weasis panel: left iframe uses `?store=raw` (before defacing), right iframe uses `?store=clean` (after defacing).
- **Studies tab** — filter by status, modality, source, project; search by UID or description; paginated 50 per page.
- **Routing tab** — manage Destinations and Routing Rules (see below).
- **Institutions tab** — manage institutions and their project memberships.
- **Profiles tab** — manage per-project anonymization profiles (see below).
- **Notifications tab** — manage email digest subscriptions (see below).
- **Projects tab** — create and edit projects (name, slug, description); shows default anon profile badge. Each project row has a **Health** button (opens `ProjectHealthPanel` modal — key metrics + pipeline funnel from `GET /api/stats/project-health`), a **Compliance** button (opens `ComplianceReportPanel` modal), and a **↓ BIDS** link that directly downloads all BIDS-complete approved studies for the project as a merged ZIP archive (calls `GET /api/projects/{id}/bids-export`). `frontend/admin-dashboard/src/components/ProjectHealthPanel.tsx`.
- **Users tab** — manage authorised admin users and their roles (admin|viewer).

**Global project selector:** A dropdown in the admin dashboard header scopes all tabs — studies list, stats banner, breakdown table, storage stats, timeline, and audit log — to a single project. Selecting "All Projects" restores the unfiltered view. The selection is persisted in `localStorage`. The stats banner, breakdown, and timeline panels auto-reload when the project changes.

### Studies List (`GET /api/studies`)

Returns a paginated envelope `{ studies, total, limit, offset }`.

| Query param | Notes |
|-------------|-------|
| `limit` | Page size (default 50, max 200) |
| `offset` | Row offset for pagination |
| `project_id` | Filter by project UUID |
| `status` | `received\|defacing\|clean\|defaced\|approved\|rejected` |
| `modality` | Case-insensitive exact match (e.g. `MRI`, `CT`) |
| `body_part` | Case-insensitive exact match (e.g. `HEAD`, `CHEST`) |
| `source` | `external\|internal` |
| `search` | Substring match on `study_instance_uid` or `study_description` |
| `label` | Case-insensitive substring match on any `study_labels.label` for the study |
| `subject_id` | Exact match on `subject_id` field |
| `flagged` | `true` to return only priority-flagged studies |

### Study Detail (`GET /api/studies/{id}`)

Returns a single study by UUID. Used by the admin dashboard's study detail panel.

### Study Lookup by DICOM UID (`GET /api/study-uid/{studyInstanceUID}`)

Returns a single study by DICOM StudyInstanceUID. Useful for integrations (PACS, DIMSE receivers, external tools) that only have the DICOM UID and not the database UUID. Returns the same payload as `GET /api/studies/{id}`. Returns 404 if no study with that UID exists.

### Study Series (`GET /api/studies/{id}/series`)

Returns per-series DICOM metadata for a study: `{study_id, series: [{id, series_instance_uid, series_description, modality, body_part, instance_count, created_at}], total}`.

Populated at ingest time by the batch importer and the internal ingest endpoint (when the caller provides `series` in `study_metadata`). Upload-portal studies start with no series rows; the classification service may backfill them. Shown in the admin dashboard study detail panel under a "Series" tab (tab only appears when series rows exist).

### Study Audit (`GET /api/studies/{id}/audit`)

Returns all audit trail entries for a specific study (by `resource_id`). Used by the study detail panel's audit tab.

### Study Diagnostics (`GET /api/studies/{id}/diagnostics`)

Returns a "why stuck?" diagnostics payload for one study:
- `study` — current full study record
- `summary` — `terminal`, `stuck`, `blockers[]`, `recommended_actions[]`, and last audit signal
- `recent_audit` — bounded latest audit entries (most recent first)
- `routing_log` — per-study routing rule log entries
- `dimse_retry` — optional DIMSE retry/dead-letter counters for the same StudyInstanceUID when DIMSE receiver is configured

Used for operator triage and MCP-assisted incident diagnosis.

### Study Detail Panel (Admin Dashboard)

Clicking a study UID in the studies table navigates to a dedicated detail view with:
- **Header** — full study UID, status/source badges, description
- **Meta row** — modality, body part, file count, series count, DICOM store, timestamps
- **Timestamp rendering** — admin dashboard, export portal, and upload portal support user-selectable viewing time zones (`UTC`, browser local, or custom IANA zone like `America/Chicago`) for display-only conversion. Preference is synced across all three UIs via shared `localStorage` keys. Upload portal converts DICOM study date/time only when an offset is present (falls back to explicit floating-time text when offset is missing).
- **Pipeline visualization** — 7-stage horizontal pipeline (Classification → PHI Scan → Protocol → Defacing → QC → BIDS → Export) with color-coded status dots
- **Action buttons** — all processing triggers, approve/reject, share, view in Weasis, review defacing, download DICOM/BIDS
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
1. Explicit selector from `POST /api/ingest` (`institution_id`, `institution_slug`, or `institution_ae_title`)
2. Fallback auto-match by request source IP against institution `ip_ranges` (most-specific CIDR wins)
3. If source IP matches multiple institutions at the same most-specific prefix length, attribution is treated as ambiguous and no institution is assigned automatically.

Network identity normalization and validation:
- `ae_title` is trimmed and normalized to uppercase on create/update.
- `ip_ranges` entries are trimmed, deduplicated, and validated (`CIDR` or single IP).
- Invalid `ip_ranges` values are rejected with `400 Bad Request`.
- `institution_ae_title` attribution must resolve to exactly one enabled institution; ambiguous AE title matches are rejected and require explicit `institution_id` or `institution_slug`.

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

**Scheduler**: goroutine started from `main.go` on startup; `time.Ticker` fires every hour; loads enabled subscriptions and evaluates due status in Go using one UTC reference timestamp per cycle (weekly: 7 days, monthly: 1 calendar month since `last_sent_at`); sends email; updates `last_sent_at`. Silent no-op when `SMTP_HOST` is unset.

**Digest content**: project name, period label, received/approved/rejected/pending study counts, export shares created. No study UIDs or identifiers.

Digest period labels are UTC-explicit (for example `2026-02-13 22:45 UTC – 2026-02-20 22:45 UTC`) so summaries are timezone-stable across regions.

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
| `DEV_USER_EMAIL` | `ai@aegisimaging.ai` | Auto-authenticated email when `AUTH_ENABLED=false` |

**How it works:**
- `AUTH_ENABLED=false` (default, local dev): every request is auto-authenticated as `DEV_USER_EMAIL`. If that email exists in `admin_users`, uses that record; otherwise uses a synthetic admin user. Zero config needed to start developing.
- `AUTH_ENABLED=true` (production): reads identity headers from the reverse proxy:
  - **GCP IAP**: `X-Goog-Authenticated-User-Email` (format: `accounts.google.com:user@example.com`)
  - **Azure AD Easy Auth**: `X-MS-CLIENT-PRINCIPAL-NAME` (user's email)
  - **AWS ALB + Cognito**: `X-Amzn-Oidc-Data` (JWT — ES256 signature verified against ALB regional public key endpoint; email extracted from payload claims)
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
1. Create a viewer user: `curl -X POST http://localhost:8080/api/admin-users -H 'Content-Type: application/json' -d '{"email":"viewer@aegisimaging.ai","name":"Test Viewer","role":"viewer","enabled":true}'`
2. Set `DEV_USER_EMAIL=viewer@aegisimaging.ai` when running the Go API
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
- `GET /api/studies/{studyUID}/bids-download` — download BIDS output as zip archive (single study)
- `GET /api/projects/{id}/bids-info` — metadata about BIDS-complete studies in a project (`{project_id, bids_complete_count, study_uids[], download_url, truncated}`); optional `?status=approved` filter (default `approved`); capped at 500 studies
- `GET /api/projects/{id}/bids-export` — stream all BIDS-complete approved studies for a project as a merged ZIP archive; sets `X-BIDS-Study-Count` and `X-BIDS-Truncated` headers; optional `?status=...` filter; emits `project.bids_export` audit entry

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

### Synthetic MRI Service (`synth-service/`, `api/handler/synth_generate.go`)

Generates synthetic DICOM brain MRI phantoms for pipeline testing, defacing demos, and
protocol development. Runs as a separate Python FastAPI service (nibabel + NumPy).

**Running locally:**
```bash
cd synth-service
pip install -r requirements.txt
uvicorn app.main:app --port 8088
# Then set SYNTH_SERVICE_URL=http://localhost:8088 when running the Go API
```

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `SYNTH_SERVICE_URL` | *(empty — disabled)* | Set to enable; empty = endpoint returns 503 |

**API endpoint:**
- `POST /api/studies/generate-synthetic` — generates a synthetic study and imports it into AEGIS
- Body: `{"project_slug", "slices", "size", "seed", "with_face", "use_gpu"}`
- Returns `{study_uid, study_ids, file_count, tool_used, duration_seconds, message}`
- Auto-dispatches the processing pipeline on the generated study

**Admin dashboard:** Synthetic MRI Generator panel in the Studies tab with controls for slices (10–200),
size (64/128/256/512), seed (randomisable), with-face toggle, and project selector.
Result shows study UID, file count, tool, and duration; triggers a study list refresh automatically.

### DIMSE Receiver Service (`dimse-receiver/`)

Receives studies from PACS systems over DICOM network protocol (DIMSE C-STORE SCP). On each C-STORE it writes files to `dicom/raw/{studyUID}/{index}.dcm` in shared storage. When the DICOM association closes (`EVT_RELEASED`), it calls `POST /api/ingest` so the normal AEGIS routing + pipeline flow starts. The ingest payload includes `institution_ae_title` (calling AE title) for institution auto-attribution; `institution_id` or `institution_slug` can also be set explicitly.

If ingest calls fail (API temporary outage, network blip), failed studies are added to a retry queue. A background worker retries at `DIMSE_INGEST_RETRY_INTERVAL` until `DIMSE_INGEST_MAX_ATTEMPTS`; exhausted items are moved to dead-letter and surfaced in `/healthz` and `/ingest/retry`. Retry/dead-letter state is persisted to disk (enabled by default) and restored on service startup.

**Running locally:**
```bash
cd dimse-receiver
pip install -r requirements.txt
uvicorn app.main:app --port 8087
# DICOM SCP listens on DIMSE_PORT (default 11112) in a background thread.
```

**PACS E2E harness (automated):**
```bash
python3 scripts/dimse_pacs_e2e_harness.py
```
- Runs four scenarios end-to-end: successful C-STORE ingest, transient failure to retry queue, targeted process recovery, dead-letter replay/process recovery.
- Detailed runbook: `docs/planning/dimse-pacs-e2e-validation-runbook.md`.

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `DIMSE_AE_TITLE` | `AEGIS` | SCP AE title for PACS associations |
| `DIMSE_PORT` | `11112` | DICOM C-STORE/C-ECHO listening port |
| `DIMSE_DATA_DIR` | `/app/data` | Shared storage mount (same as Go API) |
| `API_URL` | `http://api:8080` | Go API base URL for ingest calls |
| `DIMSE_PROJECT_SLUG` | `default` | Project slug sent to `/api/ingest` |
| `DIMSE_INSTITUTION_ID` | *(empty)* | Optional fixed institution UUID sent as `institution_id` |
| `DIMSE_INSTITUTION_SLUG` | *(empty)* | Optional fixed institution slug sent as `institution_slug` (used when ID is empty) |
| `DIMSE_INGEST_TIMEOUT` | `30` | HTTP timeout (seconds) for ingest call |
| `DIMSE_INGEST_RETRY_INTERVAL` | `15` | Retry worker interval (seconds) for queued ingest failures |
| `DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER` | `2.0` | Exponential backoff multiplier applied after each failed retry |
| `DIMSE_INGEST_RETRY_MAX_INTERVAL` | `300` | Max retry delay seconds cap for exponential backoff |
| `DIMSE_INGEST_MAX_ATTEMPTS` | `5` | Maximum attempts before moving an ingest item to dead-letter |
| `DIMSE_INGEST_QUEUE_MAX` | `1000` | Maximum in-memory queued ingest items before queue-full dead-letter |
| `DIMSE_INGEST_DURABLE_STORE_ENABLED` | `true` | Enable on-disk persistence for retry/dead-letter state across restarts |
| `DIMSE_INGEST_DURABLE_STORE_PATH` | `/app/data/dimse-ingest-retry-state.json` | JSON file path for durable retry/dead-letter state |
| `DIMSE_RETRY_ALERTS_ENABLED` | `false` | Enable threshold-based retry alert event emission |
| `DIMSE_RETRY_ALERT_MAX_EVENTS` | `200` | Max retained retry alert events in memory |
| `DIMSE_RETRY_ALERT_COOLDOWN_SECONDS` | `300` | Per-condition cooldown between repeated alerts |
| `DIMSE_RETRY_ALERT_PENDING_AGE_SECONDS` | `0` | Pending age alert threshold (fallback: `DIMSE_INGEST_PENDING_AGE_WARN_SECONDS`) |
| `DIMSE_RETRY_ALERT_DEAD_LETTER_AGE_SECONDS` | `0` | Dead-letter age alert threshold (fallback: `DIMSE_DEAD_LETTER_AGE_WARN_SECONDS`) |
| `DIMSE_RETRY_ALERT_DEAD_LETTER_NONZERO` | `true` | Emit alert when dead-letter queue is non-zero |
| `DIMSE_RETRY_ALERT_WEBHOOK_URL` | *(empty)* | Optional webhook URL for JSON alert delivery |
| `DIMSE_INGEST_PENDING_AGE_WARN_SECONDS` | `0` (disabled) | `/healthz` degrades when oldest pending retry age meets/exceeds this threshold |
| `DIMSE_DEAD_LETTER_AGE_WARN_SECONDS` | `0` (disabled) | `/healthz` includes age-threshold degradation reason when oldest dead-letter age meets/exceeds this threshold |
| `DIMSE_OPERATOR_AUDIT_MAX` | `500` | Max retained operator action records for retry control endpoints |
| `DIMSE_OPERATOR_API_KEY` | *(empty)* | Optional API key for `/ingest/retry*` endpoints via `X-AEGIS-Operator-Key` or `Authorization: Bearer` |
| `DIMSE_MAX_ASSOCIATIONS` | `10` | Max simultaneous DICOM associations |

**Operational endpoints:**
- If `DIMSE_OPERATOR_API_KEY` is set, all `/ingest/retry*` endpoints require that key.
- Admin dashboard/API integration: `/api/dimse/retry*` (admin-only) proxies to the DIMSE sidecar `/ingest/retry*` endpoints and forwards `DIMSE_OPERATOR_API_KEY` when configured.
- `GET /healthz` — includes `ingest_retry` counters (`pending`, `dead_letter`, totals including `deduped_total` and `dead_letter_deduped_total`) plus oldest-age metrics (`pending_oldest_age_seconds`, `dead_letter_oldest_age_seconds`) and next-due pending timing (`pending_next_attempt_at`, `pending_next_attempt_in_seconds`); returns `degraded` when SCP is down, dead-letter is non-zero, or pending age exceeds `DIMSE_INGEST_PENDING_AGE_WARN_SECONDS`; includes `degraded_reasons`, `pending_age_warn_seconds`, and `dead_letter_age_warn_seconds` (with `dead_letter_age_threshold_exceeded` when configured).
- Retry scheduling uses bounded exponential backoff (base interval, multiplier, max interval cap).
- Durable retry state writes are atomic (`*.tmp` swap) and restored at startup when `DIMSE_INGEST_DURABLE_STORE_ENABLED=true`.
- Retry alerts evaluate on each retry worker pass; active threshold conditions emit bounded in-memory alert events (`DIMSE_RETRY_ALERT_*`) with per-condition cooldown and optional webhook POST.
- `GET /ingest/retry` — returns retry/dead-letter counters plus oldest-age and next-due pending timing metrics for troubleshooting.
- `GET /ingest/retry/summary` — returns retry summary signals (`pending_due_now`, `queue_max`, `queue_utilization_percent`, `dead_letter_present`) plus full snapshot counters.
- `GET /ingest/retry/actions?limit=N&action=...` — returns recent operator actions on retry controls (bounded in-memory audit log), optionally filtered to a specific action name.
- `GET /ingest/retry/alerts?limit=N&condition=...` — returns recent retry threshold alerts (e.g., dead-letter non-zero, age threshold exceeded), optionally filtered by condition.
- `GET /ingest/retry/details?limit=N&study_instance_uid=...&sort=next_attempt|age_desc` — returns per-item pending/dead-letter details (`study_instance_uid`, attempts, `queued_at`, next retry timing, age counters, dead-letter timing, last_error), plus list counters (`pending_total`, `dead_letter_total`, `pending_returned`, `dead_letter_returned`) and truncation flags; optional `study_instance_uid` filter scopes results to one study; `sort=age_desc` orders oldest items first.
- `POST /ingest/retry/process` — runs one immediate retry processing pass and returns processed count + counters.
- `POST /ingest/retry/process-all?limit=N` — processes pending retry entries immediately (ignores schedule), up to `N`.
- `POST /ingest/retry/process/{study_instance_uid}` — immediate retry attempt for one pending study.
- `POST /ingest/retry/replay?limit=N` — re-queues up to `N` dead-letter items for retry; returns before/after pending/dead-letter counts and `blocked_by_queue_full` when capacity prevents replay.
- `POST /ingest/retry/replay/{study_instance_uid}` — targeted re-queue for a specific dead-letter study with before/after queue counters.
- `POST /ingest/retry/clear-pending?limit=N` — clears pending retry queue entries and returns before/after pending counts.
- `POST /ingest/retry/clear-pending/{study_instance_uid}` — targeted pending-queue clear for a specific study.
- `POST /ingest/retry/clear-dead-letter?limit=N` — clears acknowledged dead-letter items and returns before/after dead-letter counts.
- `POST /ingest/retry/clear-dead-letter/{study_instance_uid}` — targeted dead-letter clear for a specific study.

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
| `--institution-slug` | *(empty)* | Institution slug (optional, case-insensitive) |
| `--source` | `internal` | `internal` or `external` |
| `--dry-run` | `false` | Scan and report without importing |

Uses same env vars as the API (`DATABASE_URL`, `STORAGE_MODE`, `LOCAL_STORAGE_DIR`, `APP_TIMEZONE`).

**How it works:**
1. Recursively scans `--dir` for `.dcm` files
2. Parses DICOM headers (StudyInstanceUID, Modality, BodyPart, StudyDescription, SeriesInstanceUID) using `suyashkumar/dicom` with `SkipPixelData()` for performance
3. Groups files by StudyInstanceUID
4. For each study: creates upload session + study record, copies files to `dicom/raw/{studyUID}/`, evaluates routing rules
5. Duplicate StudyInstanceUIDs are rejected (unique constraint) — safe to re-run

**API endpoint:** `POST /api/import/batch` — accepts `{"dir","project_slug","institution_id","institution_slug","source","dry_run"}`, returns `{files_scanned, files_skipped, studies_created, studies_failed, errors, study_ids}`.

Validation behavior:
- `/api/import/batch` uses strict JSON decoding (`DisallowUnknownFields`); unknown/deprecated fields are rejected with HTTP `400`.
- `dir` is required, trimmed, cleaned, and must be an absolute path.
- `project_slug` is trimmed/lowercased; empty values default to `default`.
- `source` is normalized and validated; only `internal` or `external` are accepted.
- `source=external` requires canonical institution selector (`institution_id` or `institution_slug`) for provenance.
- `institution_id` and `institution_slug` are mutually exclusive (provide only one).
- Institution selector (ID or slug) must reference an enabled institution with type `sender`/`both`, linked to the target project with role `sender`/`admin`.
- Invalid directory/project/institution input now returns HTTP `400` from `/api/import/batch` (not `500`).

### Automated Processing Pipeline (`api/handler/pipeline.go`)

After routing rules evaluate (upload complete, internal ingest, batch import), the pipeline orchestrator automatically dispatches all required processing services in the correct order. No manual button clicks needed — studies flow through the pipeline hands-free.

**Env vars:**

| Var | Default | Notes |
|-----|---------|-------|
| `PIPELINE_AUTO` | `true` | Set to `"false"` to disable auto-dispatch (manual-only mode) |
| `PIPELINE_ALERT_EMAIL` | *(empty — disabled)* | Email address that receives an alert when any pipeline step fails; requires `SMTP_HOST` to be set |

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

**Pipeline failure alerts:** When `PIPELINE_ALERT_EMAIL` is set and `SMTP_HOST` is configured, a plain-text email is sent whenever a sidecar service actively reports failure (defacing, PHI scan, classification, protocol check, QC, BIDS conversion). The email body contains service name, study UID, error message, and timestamp — no PHI. Subject: `[AEGIS Alert] Pipeline step failed: <service> — <studyUID>`.

### Study Export & DICOM Download (`api/handler/export.go`, `api/handler/dicom_download.go`, `api/handler/export_forward.go`)

Full export workflow for approved studies: admin DICOM download, token-authenticated recipient download, and automated forwarding to external destinations.

**DICOM Download (admin):**
- `GET /api/studies/{studyUID}/dicom-download` — streams all DICOM files as a zip archive
- Requires auth (read-only, like BIDS download); study must be `approved`
- Uses `storage.Storage` interface (cloud-agnostic — works with local, S3, or GCS)

**DICOM Download (export share):**
- `GET /api/export/{token}/download` — token-authenticated zip download (no login required)
- Same token validation as `GET /api/export/{token}` (SHA-256 hash, expiry, revocation)
- Share expiry/revocation checks are centralized and evaluated against UTC to keep both token endpoints consistent
- Logs to `export_downloads` table + audit trail

**Create share request** (`POST /api/studies/{id}/share`):
- Uses `expiry_hours` (integer) to compute `expires_at`
- Server computes expiry in UTC (`time.Now().UTC().Add(...)`)
- Optionally accepts explicit `expires_at` in RFC3339 (timezone-aware) for backward compatibility
- Response includes server-derived `status` and `expires_in_seconds` for immediate UI state consistency

**Export share redemption** (`GET /api/export/{token}`) — enhanced response includes:
- `body_part`, `study_description`, `instance_count`, `note`, `created_by`, `download_url`
- `status` and `expires_in_seconds` (server-derived via UTC guard logic) for client clock-independent expiry UX
- Used by the export portal to display study info and download link
  - Export portal anchors countdown epoch from server `expires_in_seconds`, then ticks locally from that epoch to avoid both tab-throttle drift and client clock skew

**Share listing** (`GET /api/studies/{id}/shares`):
- Each share now includes server-derived `status` (`active` | `expired` | `revoked`) computed with UTC guard logic
- Each share now includes `expires_in_seconds` for server-clock anchored remaining-time display
- Admin UI uses this status directly instead of client-side expiry math
  - Admin share tables anchor per-row countdown epoch from server `expires_in_seconds` and auto-transition rows to `expired` without refresh

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

### Study Operations — Bulk, CSV, Notes, SLA, Re-processing, Expiry

**Bulk approve/reject/delete** (`POST /api/studies/bulk`, admin-only):
- Body: `{"action": "approve"|"reject"|"delete", "study_ids": ["<uuid>", ...]}` (max 200 IDs per call)
- Returns `{processed, errors[]}` — partial success supported; already-terminal studies are skipped with an error entry
- Triggers export forwarding and uploader notification emails on bulk approve (same as single approve)
- `action: "delete"` permanently removes the study record and all DICOM files from storage

**Studies CSV export** (`GET /api/studies.csv`, admin-read):
- Accepts same filter params as `GET /api/studies` (`project_id`, `status`, `modality`, `body_part`, `source`, `search`, plus `date_from`/`date_to` in RFC3339)
- Streams response with `Content-Disposition: attachment; filename="studies.csv"`
- Capped at 10 000 rows to prevent runaway exports

**Admin study notes** (`POST /api/studies/{id}/notes`, admin-only):
- Body: `{"note": "free text"}` (max 2 000 chars)
- Stored as `study.note` audit entry (no separate DB table) — visible in study audit trail
- Returns `{"status": "ok"}`

**SLA / stuck studies** (`GET /api/studies/stuck`, admin-read):
- Returns studies that have not advanced beyond a non-terminal state within a configurable idle window
- Query params: `minutes` (default 60, or project's `stuck_threshold_minutes` if set), `project_id` (optional)
- Response: `{stuck: [], total, minutes}`
- Alerts are stored in `study_sla_alerts` table (migration 020); admin dashboard highlights stuck studies
- Per-project SLA threshold: `PUT /api/projects/{id}/sla-threshold` — body `{"stuck_threshold_minutes": 120}` or `null` to reset to global default; emits `project.sla_threshold_updated` audit entry
- `stuck_threshold_minutes` is stored on the `projects` table (migration 039); when `project_id` is scoped and that project has a non-null value, it is used as the default threshold (still overridable by explicit `?minutes=` param)
- Admin dashboard: "SLA" button per project in Projects tab opens an inline editor (same pattern as Retention); shows current threshold as badge (e.g. `120m`) or `60m` for global default
- MCP `list_projects`: now returns `stuck_threshold_minutes` field

**Pipeline step reset / re-processing** (`POST /api/studies/{id}/reset-pipeline-step`, admin-only):
- Body: `{"step": "deface"|"phi_scan"|"qc"|"bids"|"classify"|"protocol"|"export"}`
- Resets the chosen step's status back to `pending`; auto-pipeline re-dispatches if `PIPELINE_AUTO=true`
- Returns `409 Conflict` if the step is currently in-flight
- Returns `400` if the step is not required for the study (enable it via a routing rule first)
- Emits `study.pipeline_reset` audit entry

**Study expiry warnings** (`GET /api/studies/expiring`, admin-read):
- Returns approved studies that will be soft-expired by the retention worker within the next N days
- Query params: `days` (1–365, default 7), `project_id` (optional), `limit` (1–500, default 200)
- Only studies from projects with a non-null `retention_days` are returned
- Response: `{studies: [{...study, retention_days, expires_at, days_until_expiry}], total, days, truncated}`
- `expires_at` is RFC3339; `days_until_expiry` is a non-negative integer (0 = expires today)
- Results are ordered by `expires_at` ascending (soonest-expiring first)
- Admin dashboard: orange warning banner above the studies list when studies expire within 7 days; "View" button expands an inline table with per-study expiry details; clicking a row opens the study detail panel; `days_until_expiry` is color-coded (orange = today, amber = ≤3 days, default = ≤7 days)
- MCP `get_expiring_studies` read tool

**Study expiry + reactivation** (`POST /api/studies/{id}/reactivate`, admin-only):
- Studies in `expired` status cannot be approved or rejected
- `POST /api/studies/{id}/reactivate` — sets status back to `approved` for expired studies; emits `study.reactivated` audit entry
- Admin dashboard shows "Reactivate" button for expired studies in place of Approve/Reject

### DICOM Tag Inspection (`api/handler/dicom_tags.go`)

Reads all non-pixel DICOM tags from the first file of a study and returns them as a structured list. Useful for debugging de-identification issues and verifying that protocol parameters are present.

**API:**
- `GET /api/studies/{studyUID}/dicom-tags` — returns `{tags: [{tag, keyword, vr, value}], file, store}`
- Reads from the study's current `dicom_store` (`raw` or `clean`) — runs against de-identified files post-defacing
- Uses `suyashkumar/dicom` with `SkipPixelData()` for performance

**Admin dashboard:**
- "Inspect DICOM Tags" button on the study detail panel opens a searchable tag table

### Study Labels / Annotations (`api/handler/label.go`)

Free-text labels (up to 80 characters each) that admins can attach to studies for triage, cohort tagging, or workflow notes. Stored in the `study_labels` table (migration 022).

**REST API:**
- `GET /api/studies/{id}/labels` — list all labels for a study
- `POST /api/studies/{id}/labels` — add a label (`{"label": "text"}`); emits `study.label_added` audit entry
- `DELETE /api/studies/{id}/labels/{labelID}` — remove a label; emits `study.label_removed` audit entry
- `POST /api/studies/bulk-label` — apply or remove a label across multiple studies in one request (admin-only)

**Bulk label API** (`POST /api/studies/bulk-label`):
- Body: `{"study_ids": ["<uuid>", ...], "label": "cohort-A", "action": "add"|"remove"}`
- `study_ids` must be non-empty; max 200 studies per call
- `action: "add"` — inserts the label for each study; duplicates silently ignored (`ON CONFLICT DO NOTHING`)
- `action: "remove"` — deletes the label (case-insensitive) from each study
- Returns `{applied: N, total: M}` (add) or `{removed: N, total: M}` (remove)
- Emits `study.bulk_label_added` / `study.bulk_label_removed` audit entry with count

**Admin dashboard:** Labels shown as chips on the study detail panel. Bulk action bar (shown when studies are selected) includes a label text input with "+ Label" and "− Label" buttons. Viewers can see labels; only admins can add or remove.

### Subject/Session Linking (`api/handler/subject.go`)

Associates studies with a research subject identifier (`subject_id`). Enables grouping of longitudinal imaging sessions from the same participant across multiple studies. The `subject_id` field is stored on the study record (migration 026).

**REST API:**
- `GET /api/subjects?project_id=<uuid>` — list unique subject IDs with study counts for a project
- `PUT /api/studies/{id}/subject` — set or clear `subject_id` (`{"subject_id": "SUB-001"}` or `{"subject_id": ""}` to clear)
- Emits `study.subject_set` audit entry

**Admin dashboard:** Subject input field in the study detail panel; Subjects tab lists all subjects for the current project with a count of associated studies.

### Webhook Subscriptions (`api/handler/webhook.go`, `api/webhook/deliver.go`)

Push notifications to external HTTP endpoints when study events occur. Payloads are signed with HMAC-SHA256 using the subscriber's `secret` (`X-AEGIS-Signature` header) so receivers can verify origin.

**REST API** (`/api/webhook-subscriptions`): CRUD — list, create, get, update, delete.

| Field | Notes |
|-------|-------|
| `project_id` | Scope to one project (null = all projects) |
| `url` | HTTPS endpoint that receives `POST` notifications |
| `events` | Array of event names to subscribe to |
| `secret` | HMAC-SHA256 signing key (stored, included in signature header) |
| `enabled` | Enable/disable without deleting |

**Supported events:**

| Event | Trigger |
|-------|---------|
| `study.approved` | Study moves to `approved` status |
| `study.rejected` | Study moves to `rejected` status |
| `study.phi_flagged` | PHI scan returns `flagged` |
| `study.export_complete` | Export forwarding completes successfully |
| `study.stuck` | Study is detected as stuck by SLA monitor |

**Delivery:** `deliver.go` retries up to 3 times (immediate, +5 s, +30 s). Each attempt — success or failure — is recorded in the `webhook_deliveries` table (migration 030).

**Test delivery** (`POST /api/webhook-subscriptions/{id}/test`, admin-only):
- Sends a synthetic `study.approved` payload to the subscription's URL immediately
- Returns `{success, status_code, url, error?}` so operators can verify connectivity before real events fire
- Records the attempt in `webhook_deliveries` like any real delivery

**Delivery log API:**
- `GET /api/webhook-subscriptions/{id}/deliveries` — returns immutable log of all HTTP delivery attempts for a subscription (`{id, subscription_id, event, url, attempt, status_code, success, error_message, delivered_at}`)

**Admin dashboard:** Webhook list in Notifications tab; "Test" button per row sends a synthetic test payload and shows the result; "Log" button per row opens inline delivery history table.

### API Keys (`api/handler/api_key.go`)

Machine-to-machine bearer tokens for programmatic access to the admin API. Stored hashed (SHA-256) in the `api_keys` table (migration 023) — the raw key is shown exactly once at creation and never retrievable again.

**REST API** (`/api/api-keys`):
- `GET /api/api-keys` — list all keys (shows prefix, not raw key)
- `POST /api/api-keys` — create key (`{"name": "...", "expires_at": "RFC3339 or omit"}`) — response includes `key` field with raw value
- `PATCH /api/api-keys/{id}/enable` / `PATCH /api/api-keys/{id}/disable` — toggle enabled state
- `DELETE /api/api-keys/{id}` — revoke and delete

Key format: `aegis_<base64url(32 random bytes)>`. Use as `Authorization: Bearer <key>` header. The auth middleware validates API keys alongside IAP/Azure/AWS identity headers when `AUTH_ENABLED=true`.

### Per-Project PHI Detection Config (`api/handler/phi_config.go`)

Overrides the global `PHI_CONFIDENCE_THRESHOLD` and `PHI_MIN_TEXT_LENGTH` env vars on a per-project basis. Stored in the `project_phi_config` table (migration 024) — upserted on first write, inherits global defaults if no record exists.

**REST API:**
- `GET /api/projects/{id}/phi-config` — returns `{project_id, confidence_threshold, min_text_length}`
- `PUT /api/projects/{id}/phi-config` — partial update; omitted fields keep their current values; emits `phi_config.updated` audit entry

**Admin dashboard:** PHI Config panel in the project detail view.

### Defacing QA Score (`api/handler/deface.go`, migration 025)

After defacing completes, the `deface_qa_score` field (NUMERIC(5,4), range 0.0–1.0) on the study record holds an SSIM-based visual similarity score comparing the original and defaced volumes. Higher is more similar (face successfully removed without corrupting brain tissue).

- Score is computed by the defacing service and returned in the callback payload
- Stored on the study record; visible in the study detail panel alongside the defacing status
- Score of `null` means no QA was performed (non-head study, nibabel backend, or older study)

### Study Retention Policy + Project Lifecycle (`api/handler/project.go`)

**Retention policy** (`PUT /api/projects/{id}/retention`, admin-only):
- Body: `{"retention_days": 90}` or `{"retention_days": null}` to clear the policy
- Positive integer only; `null` means keep studies indefinitely (default)
- A background worker (planned) will soft-expire approved studies older than `retention_days`
- Emits `project.retention_updated` audit entry; returns updated project

**Retention preview** (`GET /api/projects/{id}/retention-preview`, admin-read):
- Dry-run preview — does NOT modify data
- Query param: `days` (1–3650, default 90) — the simulated retention period to evaluate
- Response: `{project_id, preview_days, would_expire_count, total_approved, age_distribution}`
- `age_distribution`: array of 6 age buckets (0–7d, 8–30d, 31–90d, 91–180d, 181–365d, 365d+), each with `{label, min_days, max_days, count, would_expire}`
- `would_expire`: true when `min_days >= days` (the whole bucket would be cleared by this policy)
- Admin dashboard: "Preview impact" button in the Retention editor shows an inline table of age buckets and impact count before saving; rows are highlighted orange when they would be affected
- MCP `get_retention_preview` read tool: `{project_id, days?}`

**Project archive/restore:**
- `POST /api/projects/{id}/archive` — marks project `archived=true`; emits `project.archived` audit entry
- `POST /api/projects/{id}/restore` — clears `archived` flag; emits `project.restored` audit entry
- Archived projects are visually flagged in the Projects tab; studies remain accessible

**Project clone** (`POST /api/projects/{id}/clone`, adminOnly):
- Body: `{"name": "Copy of X", "slug": "copy-of-x"}` (both optional — name defaults to "Copy of <source>", slug auto-derived from name)
- Clones: routing rules (project-scoped only), anon profiles (all + default profile pointer re-mapped), protocol templates, PHI config, retention_days, stuck_threshold_minutes
- Does NOT clone: studies, audit entries, invite codes
- Returns new project record with HTTP 201; emits `project.cloned` audit entry with source_project_id
- Returns 409 Conflict if slug already exists
- Admin dashboard: "Clone" button per project row → prompts for new name → calls API
- MCP `clone_project` write tool: `{project_id, name?, slug?, confirm, reason}`

### Federation Peers (`api/handler/federation_peer.go`, migration 028)

Stub registry for future cross-tenant federation. Defines trusted remote AEGIS instances that will eventually be able to pull approved studies. **No data flows yet** — this is a placeholder with full CRUD, ready to be activated in a future release.

**REST API** (`/api/federation-peers`): CRUD — list, create, get, update, delete.

| Field | Notes |
|-------|-------|
| `name` | Human-readable peer name |
| `slug` | URL-safe identifier (unique) |
| `api_url` | Base URL of the remote AEGIS instance |
| `api_key_hash` | SHA-256 of the bearer token sent to the peer (stored hashed) |
| `enabled` | Enable/disable without deleting |
| `notes` | Free-text notes |

**Admin dashboard:** Federation Peers tab lists all peers with CRUD controls.

### Batch DICOM Export (`api/handler/export_batch.go`)

Triggers export forwarding for all eligible approved studies in a project in one call, instead of clicking study-by-study.

**API:**
- `POST /api/projects/{id}/export-batch` — dispatches `route_to` forwarding for all studies matching `status=approved AND export_required=true AND export_status IN ('pending','failed')`
- Returns `{dispatched, study_ids[], message}`
- Each study's `export_status` is reset to `pending` if previously `failed`, then claimed for forwarding

### Audit Trail CSV Export (`api/handler/audit_csv.go`)

Streams the audit trail as a CSV download with the same filters as `GET /api/audit`.

**API:**
- `GET /api/audit.csv` — returns `Content-Disposition: attachment; filename="audit.csv"`
- Query params: `actor`, `action`, `resource_type`, `resource_id`, `search`, `date_from`, `date_to` (RFC3339), `limit` (default 1000, max 10 000)
- `search` performs case-insensitive substring match across `action`, `actor`, `resource_type`, and `metadata`
- Columns: `id`, `action`, `actor`, `resource_type`, `resource_id`, `ip`, `created_at`, `metadata` (JSON)

**Admin dashboard:** "Export CSV" button in the Audit Log tab; export uses same active filters (search text, date range) as the currently displayed log.

**Audit log filter params** (both `GET /api/audit` JSON and `GET /api/audit.csv` CSV):

| Param | Notes |
|-------|-------|
| `search` | Case-insensitive substring match across action, actor, resource_type, metadata (ILIKE) |
| `date_from` | RFC3339 start timestamp (inclusive) |
| `date_to` | RFC3339 end timestamp (inclusive) |
| `actor` | Exact actor email filter |
| `action` | Exact action name filter (e.g. `study.approved`) |
| `resource_type` | Exact resource type filter (e.g. `study`) |

**Admin dashboard search bar:** Text input + date range pickers in the Audit Log tab toolbar; "Apply" sets filters, "Clear" resets to unfiltered view.

### Export Download Analytics (`api/handler/export.go`)

Aggregated statistics on how export shares are being redeemed.

**REST API:**
- `GET /api/export-analytics` — returns `{total_shares, total_downloads, unique_recipients, avg_downloads_per_share, shares_by_status: {active, expired, revoked}}` across all projects
- `GET /api/shares/{shareID}/downloads` — returns immutable per-share download log (`{share_id, downloads[], total}`) with `downloaded_at`, `ip`, and `user_agent` for each download event

**Admin dashboard:** Analytics row at the top of the Shares tab showing total shares, total downloads, and unique recipients.

### Institution Stats (`api/handler/institution.go`)

Per-institution aggregate statistics derived from the studies table.

**API:**
- `GET /api/institutions/{id}/stats` — returns `{institution_id, total_studies, studies_by_status: {received, approved, rejected, ...}, studies_by_modality: {...}, last_study_at}`

**Admin dashboard:** Stats panel shown when an institution is selected in the Institutions tab.

### Stats: Breakdown, Storage, Activity Summary, Timeline, and Processing Times

**Modality/body part breakdown** (`GET /api/stats/breakdown`, admin-read):
- Returns `{rows: [{modality, body_part, count}]}` ordered by count descending
- Uses `GROUP BY modality, body_part` across all studies
- Query param: `project_id` (UUID, optional) — scopes breakdown to one project
- **Admin dashboard:** collapsible breakdown table in the Studies tab header; respects global project selector

**DICOM storage stats** (`GET /api/storage/stats`, admin-read):
- Returns `{raw_file_count, clean_file_count, total_file_count, total_studies}` derived from `studies.instance_count` using SQL `FILTER (WHERE dicom_store = ...)` aggregation — cloud-agnostic, no filesystem walk
- Query param: `project_id` (UUID, optional) — scopes counts to one project
- **Admin dashboard:** raw/clean file counts displayed in the stats banner; respects global project selector

**Pipeline stats** (`GET /api/stats`, admin-read):
- Returns status counts (`received`, `defacing`, `clean`, `defaced`, `approved`, `rejected`) and aggregate counts
- Query param: `project_id` (UUID, optional) — scopes counts to one project

**Admin activity summary** (`GET /api/audit/actors`, admin-read):
- Returns `{actors: [{actor, action_count, last_seen_at, last_action}]}` for the last 30 days, ordered by `action_count DESC`
- Query param: `limit` (default 20)
- **Admin dashboard:** collapsible "Recent admin activity" table in the Audit Log tab

**Daily ops summary** (`GET /api/stats/daily-summary`, admin-read):
- System-wide briefing for the last N hours (default 24, max 168 = 7 days)
- Returns: `ingestion` (received/approved/rejected/stuck in period), `pipeline` (current pending review/in-processing/failed snapshot), `routing` (attempts/success rate), `top_projects` (top 5 by received), `recent_events` (last 10 significant audit events)
- Query params: `hours` (1–168, default 24)
- Designed for AI agent "morning briefings" — single call for cross-system platform state

**Daily ingestion timeline** (`GET /api/stats/timeline`, admin-read):
- Returns `{days: [{day: "YYYY-MM-DD", received, approved}]}` for the last N days (default 30)
- Query params: `days` (1–365), `project_id` (UUID, optional)
- **Admin dashboard:** collapsible "Daily ingestion (last 30 days)" table showing received and approved counts per day; respects global project selector; resets when project changes

**Pipeline stage processing times** (`GET /api/stats/processing-times`, admin-read):
- Returns `{generated_at, period_days, stages: [{stage, count, avg_seconds, p95_seconds, min_seconds, max_seconds}]}` derived from paired `.triggered` / `.complete` audit trail events
- Stages: `deface`, `phi_scan`, `qc_check`, `bids_conversion`, `classification`, `protocol_check`, `export`
- Uses a lateral join to find the nearest completion event after each trigger within a 2-hour window, correctly handling re-processing
- Query params: `days` (1–365, default 30), `project_id` (UUID, optional)
- Results ordered slowest-to-fastest by average duration
- **Admin dashboard:** collapsible "Stage processing times (last 30 days)" table showing avg/P95/min/max per stage with human-readable duration formatting
- **MCP:** `get_processing_stats` read tool (supports `days` and `project_id` params)

### Destination Routing Health Stats (`api/handler/destination_stats.go`)

Per-destination and aggregate routing success/failure analytics derived from the `routing_log` table.

**API:**
- `GET /api/stats/routing?days=30` — aggregate routing stats across all destinations; returns `{period_days, generated_at, totals: {attempts, successful, failed, success_rate}, by_destination: [{destination_id, destination_name, destination_type, attempts, successful, failed, success_rate, last_attempt_at}]}` ordered by attempt count descending
- `GET /api/destinations/{id}/stats?days=30` — per-destination stats; returns totals + `recent_errors` (last 10 failure messages) + `daily_breakdown` (per-day counts)
- Both endpoints: `days` param accepts 1–365, default 30

**Admin dashboard:** collapsible "Routing Health" section at the bottom of the Routing tab showing aggregate totals + per-destination breakdown table with color-coded success rates (teal ≥ 95%, amber 80–94%, orange < 80%).

**MCP:**
- `get_routing_stats` read tool — aggregate routing health overview (`days` param)
- `get_destination_stats` read tool — per-destination stats (`destination_id` required, `days` optional)

### Routing Rule Hit Analytics (`api/handler/routing_rule_stats.go`)

Per-rule hit counts from the `routing_log` table — identifies which rules are actively firing and which are stale.

**API:**
- `GET /api/stats/routing-rules?days=30` — returns `{period_days, generated_at, total_hits, active_rules, by_rule: [{rule_id, rule_name, rule_action, rule_enabled, destination_id, destination_name, hit_count, last_matched_at, first_matched_at}], unused_rules: [{rule_id, rule_name, rule_action, rule_enabled, destination_id}]}`
- `by_rule` ordered by `hit_count DESC`; `unused_rules` lists rules with zero log entries in the period (ordered by priority then name)
- `days` param accepts 1–365, default 30

**Admin dashboard:** collapsible "Rule Analytics" section in the Routing tab showing hit-count summary cards, a ranked rules table, and a separate "Unused rules" table (amber heading) for rules that haven't fired in the period.

**MCP:** `get_routing_rule_stats` read tool — returns hit analytics + unused rules list (`days` optional)

### Pipeline Conversion Funnel (`api/handler/pipeline_funnel.go`)

Shows how many studies pass through each pipeline stage from receipt to export, making it easy to spot where studies are dropping out.

**API:**
- `GET /api/stats/pipeline-funnel?days=30&project_id=<uuid>` — returns `{period_days, generated_at, project_id, funnel: [{stage, count, pct_of_total, pct_of_prev}]}`
- Stages: `received` → `classified` → `phi_scanned` → `defaced` → `qc_passed` → `bids_converted` → `approved` → `exported`
- `pct_of_total` = percentage of received studies that reached this stage
- `pct_of_prev` = conversion rate from the immediately preceding stage
- `days` param 1–365 (default 30); `project_id` is optional

**Admin dashboard:** collapsible "Pipeline funnel (last 30 days)" toggle in the Studies tab showing stage counts, percentage of received, conversion rate, and a mini progress bar (teal ≥80%, amber 50–79%, orange <50%).

**MCP:** `get_pipeline_funnel` read tool — returns full funnel breakdown (`days` and `project_id` optional)

### Project Health Summary (`api/handler/project_health.go`)

Aggregates study counts, storage stats, routing totals, stuck count, and the pipeline funnel into a single endpoint — the fastest way to get a complete project status snapshot.

**API:**
- `GET /api/stats/project-health?days=30&project_id=<uuid>&stuck_minutes=60`
- Returns: `{period_days, generated_at, project_id, studies: {received, defacing, clean, defaced, approved, rejected}, storage: {raw_file_count, clean_file_count, total_file_count, total_studies}, stuck_count, routing: {attempts, successful, failed, success_rate}, funnel: [{stage, count, pct_of_total, pct_of_prev}]}`
- `days` scopes routing and funnel look-back (default 30); `project_id` optional; `stuck_minutes` threshold for stuck count (default 60)

**MCP:** `get_project_health` read tool — consolidated health snapshot, ideal for AI-assisted project triage

### Protocol Template Export / Import (`api/handler/protocol_template.go`)

Export and import protocol templates for a project as a JSON file.

**Export API:**
- `GET /api/projects/{projectID}/protocol-templates/export` — returns `Content-Disposition: attachment; filename="protocol-templates.json"` with `{"project_id", "templates": [...], "count"}` indented JSON
- Emits `protocol_template.exported` audit entry with count

**Import API** (admin-only):
- `POST /api/projects/{projectID}/protocol-templates/import` — body: either a raw JSON array `[{name, manufacturer, ...}]` or the export format `{"templates":[...]}`
- Skips templates whose name already exists in the project (no overwrite)
- Returns `{imported: N, skipped: M, errors: []}` — partial success supported
- Emits `protocol_template.imported` audit entry with imported/skipped counts

**Admin dashboard:** "Export JSON" and "Import JSON" buttons in the Protocol Templates section header. Import shows a success/error toast after file selection.

### Export Share Extension (`api/handler/export.go`)

Extends the expiry of an active or already-expired export share without revoking and re-creating it.

**API:**
- `PATCH /api/shares/{shareID}/extend` — body: `{"extend_hours": 48}`
- Extension base is `max(expires_at, now())` — works even after expiry
- Returns `409 Conflict` if the share has been revoked
- Emits `share.extended` audit entry

**Admin dashboard:** "Extend" button in the Shares tab for active and expired (but not revoked) shares.

### Share Revocation Reason (`api/handler/export.go`, migration 038)

Stores an optional free-text reason when revoking an export share. Visible in the audit trail.

**API:**
- `DELETE /api/shares/{shareID}` — body: `{"reason": "optional text (max 500 chars)"}` (body is optional; omitting it revokes with no reason)
- Reason stored in `export_shares.revocation_reason` column (migration 038)
- Included in `share.revoked` audit entry metadata when provided

**Admin dashboard:** Revoke button opens an inline modal with an optional reason textarea (max 500 chars) before confirming. Present in both the global Shares tab and the per-study share panel.

**MCP `revoke_share` tool:** Accepts optional `revocation_reason` field (stored in DB) in addition to the required `reason` field (audit justification only).

### Destination Connectivity Test (`api/handler/routing.go`, `dimse-receiver/app/`)

Probes an external DICOM destination to verify network reachability before adding routing rules that depend on it.

**API:**
- `POST /api/destinations/{id}/test` (adminOnly) — tests connectivity to the destination
  - **DICOMweb**: sends `GET {dicomweb_url}/studies?limit=1` with auth header; checks for non-4xx response
  - **DIMSE**: sends C-ECHO to `ae_title@host:port` via the dimse-receiver `/echo` endpoint (requires `DIMSE_RECEIVER_URL`)
  - Returns `{destination_id, type, success, latency_ms, status_code?, error?}`
  - Emits `destination.tested` audit entry

**dimse-receiver `/echo` endpoint** (`POST /echo`): accepts `{ae_title, host, port}`, sends C-ECHO via pynetdicom, returns `{success, latency_ms}` or HTTP 502 on failure.

**Admin dashboard:** "Test" button per destination row in the Routing tab; shows inline result (teal for success, orange for failure + latency).

**MCP `test_destination` read tool:** `{destination_id}` → calls `POST /api/destinations/{id}/test`, returns connectivity result.

### Destination Health History (`api/handler/destination_health.go`)

Aggregated health statistics for DICOM destinations based on historical connectivity test results stored in the audit trail (`destination.tested` entries).

**API:**
- `GET /api/destinations/{id}/health?limit=20` — single destination health summary + recent test log
- `GET /api/destinations/health` — health summary for all destinations (no test log)

**Response fields:**
- `summary.status` — `healthy` (≥90% success), `degraded` (50–89%), `failing` (<50%), `unknown` (never tested)
- `summary.test_count`, `success_count`, `failure_count`, `success_rate` (0.0–1.0; -1 = never tested)
- `summary.last_tested_at`, `last_success`, `last_failure` — timestamps of last events
- `summary.last_error` — error message from most recent failed test
- `recent_tests` — ordered list of individual test events (tested_at, actor, success, error, latency_ms)

**Admin dashboard:** Health badge per destination row in the Routing tab — colored status dot (`● healthy`, `◐ degraded`, `● failing`, `○ unknown`) with success rate %; loaded automatically when Routing tab opens.

**MCP `get_destination_health` read tool:** `{destination_id?}` → single dest health (with test log) when ID provided, all-destinations summary otherwise.

### Per-IP Rate Limiting (`api/middleware/`)

Token-bucket rate limiting on public upload endpoints prevents abuse without affecting authenticated admin traffic.

- Applied to: `POST /api/upload/session`, `POST /api/upload/{id}/complete`, `POST /api/ingest`, `POST /api/contact`
- Each source IP gets its own bucket (configurable capacity and refill rate via env vars or defaults)
- Exceeding the limit returns `429 Too Many Requests`
- Admin and sidecar endpoints are not rate-limited

### MCP Server (`mcp-server/`)

A Model Context Protocol server that exposes AEGIS admin operations as typed tools for AI agents (Claude, Cursor, etc.). Tools are split into read-only and write (mutating) categories. All tools accept an optional `request_id` for correlation.

**Running locally:**
```bash
cd mcp-server && npm install && npm run build
# Use with Claude Desktop: point config to mcp-server/dist/index.js
```

**Read tools** (safe to call without confirmation):

| Tool | Description |
|------|-------------|
| `list_studies` | Paginated study list with filters (status, modality, body_part, source, project_id, search, date range) |
| `get_study_detail` | Full study record by UUID |
| `get_study_by_uid` | Full study record by DICOM StudyInstanceUID |
| `get_study_diagnostics` | "Why is this stuck?" triage payload |
| `get_study_audit` | Per-study audit trail |
| `get_study_routing_log` | Per-study routing rule execution log |
| `list_export_shares` | Shares for a study |
| `list_all_shares` | All shares across all studies (filterable by status) |
| `get_share_downloads` | Per-share download analytics |
| `get_system_health` | API + sidecar health (`/healthz`) |
| `get_dimse_retry_status` | DIMSE retry/dead-letter counters |
| `get_audit_log` | Global audit log with filters (search, date_from, date_to, action, resource_type, actor) |
| `get_pipeline_stats` | Study status counts (optionally scoped to a project) |
| `get_stuck_studies` | Studies idle beyond a threshold (minutes, optional project_id) |
| `get_expiring_studies` | Approved studies expiring within N days per retention policy (days, project_id, limit) |
| `get_retention_preview` | Dry-run preview of how many studies would expire at a given retention period (project_id, days) |
| `get_breakdown_stats` | Modality/body part breakdown (optional project_id) |
| `get_storage_stats` | Raw/clean file counts (optional project_id) |
| `get_processing_stats` | Per-stage processing-time statistics (avg/p95/min/max) derived from audit trail (optional project_id, days) |
| `get_audit_actors` | Top admin actors in the last 30 days |
| `list_projects` | All projects with id/name/slug/archived/retention_days |
| `list_institutions` | All institutions with type/ae_title/ip_ranges |
| `list_routing_rules` | All routing rules ordered by priority |
| `list_destinations` | All DICOM forwarding destinations |
| `test_destination` | Test connectivity to a DICOM destination (DICOMweb GET probe or DIMSE C-ECHO) |
| `get_destination_health` | Historical health summary from past tests — status (healthy/degraded/failing/unknown), success rate, last errors; optional `destination_id` for single-dest detail |
| `get_routing_stats` | Aggregate routing health overview across all destinations (optional days) |
| `get_destination_stats` | Per-destination routing stats: success rate, recent errors, daily breakdown (destination_id required, optional days) |
| `get_routing_rule_stats` | Per-rule hit analytics: hit counts, last matched, unused rules in period (optional days) |
| `get_pipeline_funnel` | Pipeline conversion funnel: per-stage counts and conversion rates (optional days, project_id) |
| `get_project_health` | Consolidated project health: study counts + storage + routing totals + stuck count + funnel in one call (optional days, project_id, stuck_minutes) |
| `get_compliance_report` | Project compliance report: PHI detection rates, defacing completion, protocol compliance, export activity (project_id required, optional days) |
| `get_storage_usage` | Project storage usage in bytes with quota information (project_id required) |
| `get_anonymization_diff` | Tag-level diff between raw and de-identified DICOM for a study (study_uid = DICOM UID, not DB UUID) |
| `get_system_health_summary` | Cached (30s) system-wide health: API/sidecar status, pipeline activity (24h), DIMSE queue depths |
| `list_study_relationships` | List all typed relationships for a study (baseline, follow_up, comparison, replicate) with related study details |
| `list_digest_subscriptions` | List email digest subscriptions (weekly/monthly plain-text study counts); optional project_id filter |
| `get_daily_summary` | System-wide ops briefing for last N hours (default 24, max 168): ingestion counts, pipeline state, routing stats, top 5 projects by received, last 10 significant audit events |
| `list_admin_users` | List all registered admin dashboard users with roles, enabled state, and last-seen info |
| `list_invite_codes` | List all beta invite codes with label, enabled state, and usage info (used_at, used_by_ip) |
| `list_invite_requests` | List access requests from the landing page; optional `status=pending\|approved\|denied\|all` filter |
| `get_tcia_series` | Browse available TCIA (The Cancer Imaging Archive) series for a collection (optional min_slices filter) |
| `export_protocol_templates` | Export all protocol compliance templates for a project as a JSON structure |
| `get_webhook_stats` | Per-subscription delivery statistics: total, success/failure counts, success rate, last delivery, breakdown by event |
| `list_all_webhook_deliveries` | All webhook delivery attempts across all subscriptions; filterable by subscription_id and success status; paginated |
| `get_user_preferences` | Get notification preferences for an admin user: digest frequency and subscribed notify events |
| `get_project_bids_info` | BIDS availability for a project: count of BIDS-complete studies, UIDs list, and download URL for the bulk ZIP |

**Write tools** (require `confirm: true` and a `reason` string):

| Tool | Description |
|------|-------------|
| `approve_study` | Approve a study |
| `reject_study` | Reject a study |
| `trigger_classification` | Trigger metadata classification |
| `trigger_phi_scan` | Trigger PHI scan |
| `trigger_protocol_check` | Trigger protocol compliance check |
| `trigger_qc_check` | Trigger QC automation |
| `trigger_bids_convert` | Trigger NIfTI/BIDS conversion |
| `trigger_export` | Trigger DICOM export forwarding |
| `trigger_deface` | Trigger defacing |
| `retry_dimse_study` | Retry a DIMSE ingest failure |
| `revoke_share` | Revoke an export share |
| `create_share` | Create a new export share |
| `re_evaluate_routing` | Re-evaluate routing rules for a study |
| `toggle_study_flag` | Set or clear the priority flag (★) on a study |
| `clone_project` | Duplicate a project with all settings (routing rules, profiles, templates, PHI config) |
| `link_studies` | Create a typed relationship between two studies (baseline, follow_up, comparison, replicate) |
| `unlink_studies` | Remove a study relationship by its relationship UUID (use list_study_relationships to find the ID) |
| `create_digest_subscription` | Create a weekly or monthly email digest subscription for a project |
| `delete_digest_subscription` | Delete a digest subscription by ID |
| `create_webhook_subscription` | Create a webhook subscription to receive push notifications on study events (HMAC-SHA256 signed payloads) |
| `update_webhook_subscription` | Update a webhook subscription — change URL, events, secret, or enabled state |
| `delete_webhook_subscription` | Permanently delete a webhook subscription and all its delivery history |
| `retry_webhook_delivery` | Retry a specific failed webhook delivery by delivery UUID |
| `create_project` | Create a new project (name required, slug auto-derived from name if omitted) |
| `update_project` | Update project name, slug, or description |
| `archive_project` | Archive a project (studies remain accessible; visually flagged in dashboard) |
| `restore_project` | Restore (unarchive) a previously archived project |
| `set_project_retention` | Set or clear the study retention policy in days (`null` = indefinite) |
| `set_project_sla_threshold` | Set or clear per-project stuck-study SLA threshold in minutes (`null` = global default 60 min) |
| `create_admin_user` | Register a new admin dashboard user with email and role (admin\|viewer) |
| `update_admin_user` | Update an admin user's email, role, enabled state, or notes |
| `delete_admin_user` | Permanently remove an admin user record |
| `create_invite_code` | Generate a new XXXX-XXXX-XXXX invite code with a human-readable label |
| `revoke_invite_code` | Revoke (disable) an invite code so it can no longer be used |
| `delete_invite_code` | Permanently delete an invite code record |
| `send_invite_code` | Email an invite code directly to a recipient (requires SMTP) |
| `approve_invite_request` | Approve a pending access request — creates invite code and emails it to the requester |
| `deny_invite_request` | Deny a pending access request |
| `create_protocol_template` | Create a new MRI protocol compliance template with acquisition parameter rules (TR, TE, flip angle, etc.) for a scanner/sequence combination |
| `update_protocol_template` | Update an existing protocol compliance template (name, rules, enabled state) |
| `delete_protocol_template` | Permanently delete a protocol compliance template |
| `create_anon_profile` | Create a new anonymization profile defining which DICOM tags to retain during PS3.15 de-identification |
| `update_anon_profile` | Update an anonymization profile (name, description, retained_tags, enabled) |
| `delete_anon_profile` | Permanently delete an anonymization profile |
| `set_default_anon_profile` | Set or clear the default anonymization profile for a project (empty profile_id clears the default) |
| `create_institution` | Register a new institution (hospital, research site) with type, IP ranges, and AE title |
| `update_institution` | Update an institution's metadata, network identity, or enabled state |
| `delete_institution` | Permanently delete an institution record |
| `link_institution_project` | Link an institution to a project with a role (sender/receiver/admin) |
| `unlink_institution_project` | Remove an institution-project link |
| `create_federation_peer` | Register a trusted remote AEGIS instance as a federation peer (stub — no data flows yet) |
| `update_federation_peer` | Update a federation peer's name, URL, notes, or enabled state |
| `delete_federation_peer` | Permanently delete a federation peer registration |
| `set_storage_quota` | Set or clear the per-project DICOM storage quota in bytes (null = unlimited) |
| `update_phi_config` | Override global PHI detection thresholds (confidence_threshold, min_text_length) for a project |
| `delete_study` | Permanently delete a study record and all DICOM files from storage |
| `bulk_pipeline_trigger` | Trigger a pipeline step (classify/phi_scan/protocol/deface/qc/bids/export) for multiple studies at once |
| `import_tcia_series` | Download a TCIA series by SeriesInstanceUID and import it into AEGIS (triggers full pipeline) |
| `import_protocol_templates` | Bulk-import protocol compliance templates into a project from a JSON array (skips duplicates by name) |
| `batch_import_studies` | Import DICOM files from a server-local directory path into AEGIS; supports dry_run preview mode |
| `set_user_preferences` | Update an admin user's personal notification preferences: digest frequency and subscribed notify events |

All schemas validated with Zod at the MCP layer. Write operations use `RequireRole("admin")` on the underlying API endpoints.

### Terraform
```bash
cd terraform/project && terraform init && terraform plan
cd terraform/infra && terraform init && terraform plan
```

`terraform/infra` now provisions production-baseline GCP infra:
- custom VPC + subnet + private-service networking + Cloud NAT (SMTP egress via static NAT IP),
- Artifact Registry, Cloud SQL private IP, Healthcare API dataset/stores, GCS buckets, Pub/Sub, BigQuery,
- Cloud Run services (API + admin dashboard + processing sidecars),
- HTTPS load balancer with Cloud Armor on API backend and IAP on admin backend,
- baseline monitoring notification channel + alert policies.

Required infra tfvars include:
- domains: `api_domain`, `admin_domain`
- IAP OAuth credentials: `iap_oauth_client_id`, `iap_oauth_client_secret`, `iap_access_members`
- runtime images: API/admin/sidecar image URIs
- database credential bootstrap: `db_password` (used to create Cloud SQL user + Secret Manager version)
- optional secret naming: `db_password_secret_id` (default: `aegis-<env>-db-password`)

Secrets posture:
- GCP API runtime now reads DB credentials via Secret Manager reference (`DB_PASSWORD` from secret, not inline DSN).
- AWS RDS now uses `manage_master_user_password = true`, with master credentials stored in AWS Secrets Manager.

`terraform/aws` now includes edge/auth completion:
- ACM-backed ALB HTTPS listener (`aws_lb_listener.https`)
- Cognito user pool/client/domain resources
- ALB `authenticate-cognito` default action with explicit public-path bypass rules (`/healthz`, upload/export/contact/project selector, DICOMweb)

## Key Architecture Decisions

- Client-side DICOM tag anonymization in browser before upload (zero-install at sending sites)
- Dual-ingress model: external-site browser upload and internal-enterprise ingestion both enter the same enterprise GCP tenancy and processing pipeline
- Server-side defacing in separate Python Cloud Run service
- Two DICOM stores: `raw` (tag-de-identified) and `clean` (fully processed including defacing)
- Go for main API (minimal CVE surface, fast cold starts), Python for isolated processing sidecars and DIMSE ingress
- DICOM PS3.15 Annex E Basic Profile for de-identification
- Cloud SQL (PostgreSQL) for application state; cloud-neutral object storage (local/GCS/S3) for DICOM bytes
- Optional cloud AI backends for PHI/classification: Google Cloud Vision + AWS Textract/Rekognition
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

## CI/CD — Auto-Deploy on Push to `develop`

**Never manually deploy after merging a PR to `develop` — Cloud Build handles everything automatically.**

### Cloud Build triggers (GCP)

Two triggers are active in Cloud Build (configured by `scripts/gcp_setup_cloudbuild.sh`, run once per environment):

#### Trigger 1: Application deploy — `deploy-on-develop`
- **Config**: `cloudbuild.yaml`
- **Fires on**: any push to `develop`
- **Phase 1** (all parallel): build + push Docker images to Artifact Registry (`{service}:{SHORT_SHA}` + `{service}:latest`)
- **Phase 2** (each parallel, waits for its own build): `gcloud run deploy` each service with the new image

| Cloud Run service | Source directory |
|---|---|
| `aegis-api` | `api/` |
| `aegis-admin-dashboard` | `frontend/admin-dashboard/` (build arg: `VITE_WEASIS_BASE_URL`) |
| `aegis-prod-landing` | `frontend/landing/` (build arg: `VITE_API_BASE_URL`) |
| `weasis` | `weasis/` |
| `defacing` | `defacing/` |
| `phi-detection` | `phi-detection/` |
| `qc-service` | `qc-service/` |
| `bids-service` | `bids-service/` |
| `classification-service` | `classification-service/` |
| `protocol-service` | `protocol-service/` |
| `synth-service` | `synth-service/` |
| `aegis-mcp-server` | `mcp-server/` |
| `aegis-prod-dimse-receiver` (GCE VM) | `dimse-receiver/` — see note below |

Only the container image is updated on each deploy; all env vars, secrets, CPU/memory, and service accounts are preserved from the running config.

> **dimse-receiver deploy note**: Cloud Run cannot expose raw TCP port 11112 required by DICOM C-STORE SCP. The dimse-receiver runs on a **Compute Engine VM** (Debian 12) provisioned by `terraform/infra/dimse.tf`. Cloud Build deploys it by updating the `dimse-image` metadata key on the VM and issuing `gcloud compute instances reset`. The startup script reads this key on every boot and pulls + starts the new image. The VM is only created when `dimse_receiver_image` is set in `terraform.tfvars` (empty = skip all DIMSE resources). `lifecycle { ignore_changes = [metadata["dimse-image"]] }` prevents `terraform apply` from reverting Cloud Build's metadata updates.

#### Trigger 2: Terraform apply — `terraform-apply-on-develop`
- **Config**: `cloudbuild.terraform.yaml`
- **Fires on**: push to `develop` **where `terraform/infra/**` files changed**
- **Steps**: fetch `terraform.tfvars` from Secret Manager (`aegis-prod-terraform-tfvars`) → `terraform init` (GCS backend: `aegis-prod-488120-tfstate`) → `terraform validate` → `terraform apply -auto-approve`
- **Scope**: only `terraform/infra/`

**`terraform/project/` is intentionally manual** — it manages project-bootstrap resources that are dangerous to auto-apply:
- KMS key ring + crypto key (`prevent_destroy = true`) — accidental re-creation locks encrypted data
- Service accounts — deletion breaks all workload identity bindings
- Project-level IAM and API enables — VPC Service Controls perimeter, audit log config
- Run manually: `cd terraform/project && terraform apply` after careful `terraform plan` review

**`terraform/aws/` is intentionally manual** — AWS is live production (`us-east-1`). Deploy manually: `cd terraform/aws && terraform plan && terraform apply`.

**`terraform/azure/` is intentionally manual** — Azure is the third cloud target. Deploy manually: `cd terraform/azure && terraform init && terraform plan && terraform apply`. Auto-deploy is handled by `.github/workflows/deploy-azure.yml` (GitHub Actions, fires on `develop` push). Terraform infra changes remain manual to avoid accidental resource destruction.

Monitor builds: `gcloud builds list --project=aegis-prod-488120 --limit=5`

## CI (GitHub Actions)

`.github/workflows/ci.yml` runs on PRs to `develop` and `main` — **validation only, no deploy**:

| Job | What it checks |
|-----|---------------|
| `go` | `go build ./...` + `go vet ./...` |
| `go-test` | `go test -race -v -count=1 ./...` (~120 tests) |
| `infra-guard` | `scripts/check-infra-placeholders.sh` blocks known credential placeholders in Terraform |
| `python` (7× matrix) | `py_compile` on all `.py` files per service |
| `python-scripts` | `py_compile` on smoke/DIMSE harness scripts in `scripts/` |
| `python-test` (7× matrix) | `pytest -v --tb=short` per service (~206 tests total) |
| `frontend` (5× matrix) | `npx tsc --noEmit` (client, upload-portal, admin-dashboard, export-portal, landing) |
| `docker` (8× matrix) | `docker build` for all service images |

Manual workflow:
- `.github/workflows/cloud-smoke.yml` (`workflow_dispatch`) runs `scripts/cloud_smoke_test.py` against a deployed environment.
- Optional repo secret `CLOUD_SMOKE_ADMIN_HEADER` provides the admin auth header for protected endpoints.
- `.github/workflows/deploy-azure.yml` — **Azure auto-deploy** (fires on push to `develop`): builds all 13 images, pushes to ACR, deploys all Container Apps. Requires GitHub secrets: `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`, `AZURE_ACR_REGISTRY`, `AZURE_RESOURCE_GROUP`. Variables: `AZURE_WEASIS_URL`, `AZURE_API_URL`, `AZURE_DIMSE_VM_NAME` (optional).

### Python Sidecar Testing

Each sidecar has `requirements-test.txt` (pytest + httpx) and a `tests/` directory:

```bash
cd {service} && pip install -r requirements.txt -r requirements-test.txt && pytest -v
```

| Service | Tests | Coverage |
|---------|-------|----------|
| classification-service | 49 | Heuristic classification (5 strategies), SOP UID mapping, body part regex, Cloud Vision/Rekognition label mapping, cloud backend inheritance, pixel_utils, endpoint tests |
| dimse-receiver | 68 | C-STORE file write/indexing, EVT_RELEASED ingest trigger, C-ECHO, DIMSE forward endpoint mapping, sender status/path helpers, ingest payload/error handling, retry queue/dead-letter behavior, retry deduplication (queue + dead-letter), bounded exponential backoff, operator action audit logging, optional API-key protection, retry status/actions/details/process/process-all/targeted-process/replay/targeted-replay/clear-pending/targeted-pending-clear/clear-dead-letter/targeted-dead-letter-clear endpoints |
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
| `make lint` | Lint all languages with CI-parity scope (Go vet, Python py_compile including protocol/dimse + harness scripts, TypeScript tsc for all apps, MCP typecheck) |
| `make check` | `curl /healthz` with pretty JSON output |
| `make smoke` | Run cloud smoke harness (`BASE_URL=...`, optional `ADMIN_HEADER=...`) |
| `make dimse-e2e` | Run DIMSE PACS E2E harness (mock ingest + retry/dead-letter scenarios) |
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
