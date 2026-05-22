# AEGIS Router

On-prem DICOM router for spoke sites. Acts as a mini-AEGIS deployment that
receives DICOM studies from the local PACS (or via the same browser uploader
that already targets the cloud), de-identifies them in place, then forwards
the cleaned studies to the cloud receiver over mTLS.

## What it does

1. **Listens** for incoming studies on two paths:
   - **DICOM C-STORE SCP** on port `11112` (auto-route from PACS)
   - **HTTPS web upload** on port `8080` (same `/api/upload/*` contract as the
     cloud upload-portal, so the existing browser client just points at the
     router instead of the cloud)
2. **De-identifies** locally:
   - Always: tag de-id via the MIDI-B profile (`midi-b/`)
   - Optional: pixel PHI scrub via `phi-detection` sidecar (Tesseract OCR)
   - Optional: face de-id via `defacing` sidecar (mri_deface, deepdefacer,
     mri_reface)
3. **Forwards** the de-identified study to the cloud over **mTLS** using the
   per-spoke client cert issued at onboarding.
4. **Quarantines** locally on any failure. Nothing PHI-bearing ever leaves the
   building until cleared by an operator.

## Why a spoke router

The web upload portal lets a single user upload a study at a time. That's fine
for occasional research contributions but doesn't scale to "every study this
hospital reads needs to go to the registry." The router lets a site:

- Auto-route the PACS so every study flows out without human intervention
- Do all the de-identification on-prem (compliance teams like this)
- Keep operating during cloud outages (studies queue locally, ship when the
  link recovers)
- Optionally run any of the cloud's analysis sidecars locally too — see
  [Modular capabilities](#modular-capabilities).

## Modular capabilities (compose profiles)

The router itself is small (~50 MB). Every additional capability is an
opt-in docker-compose profile that pulls in the corresponding sidecar
container:

| Profile | Sidecar | What it adds | CPU-only OK? |
|---|---|---|---|
| _(none)_ | — | Tag de-id only, forward to cloud | ✔ |
| `--profile defacing` | `defacing` | Face de-id on head MR/CT/PET | ✔ (slow) |
| `--profile phi` | `phi-detection` | Pixel PHI scrub (Tesseract OCR) | ✔ |
| `--profile qc` | `qc-service` | Automated QC checks | ✔ |
| `--profile classification` | `classification-service` | Modality / body-part classification | ✔ |
| `--profile protocol` | `protocol-service` | MRI protocol compliance | ✔ |
| `--profile bids` | `bids-service` | NIfTI/BIDS conversion | ✔ |
| `--profile analytics` | `analytics-service` | FreeSurfer, FSL, ANTs, etc. | partial |
| `--profile sct` | `sct-service` | Spinal cord toolbox | ✔ |
| `--profile full` | all of the above | Cloud parity | mixed |

Profiles can stack: `docker compose --profile defacing --profile phi up -d`.

## Install scenarios

The router runs anywhere Docker runs. Three common patterns:

### 1. Single laptop / dev workstation (no cloud)

For self-testing on a Mac or Linux laptop. Runs the router locally; no AEGIS
cloud account or certificate needed. Studies de-identify and stay on disk
under `/var/lib/aegis-router/studies/` for inspection.

```bash
cd router
cp .env.example .env
# In .env, leave CLOUD_RECEIVER_URL empty so the router stays local-only.

docker compose up -d                            # router only
# or with face de-id + pixel scrub:
docker compose --profile defacing --profile phi up -d

# Send a synthetic study at it:
pip install pynetdicom pydicom httpx
./bin/aegis-router-send-test --slices 20

# Inspect what the router did:
curl http://localhost:8080/audit | jq
./bin/aegis-router-deid-diff \
    --raw  /var/lib/aegis-router/studies/<study_uid>/raw \
    --deid /var/lib/aegis-router/studies/<study_uid>/deid
```

**Apple Silicon note:** all images in this stack have native ARM64 builds
(`python:3.12-slim`, our wheels are pure Python). No Rosetta needed.

### 2. Linux server + remote laptop client

Run the router on a small server (a NUC, a workstation, an old desktop) and
send DICOM to it from your laptop on the same LAN. Useful for testing PACS
auto-routing setups before deploying to a clinical site.

```bash
# On the Linux server:
cd router
cp .env.example .env
docker compose up -d
sudo ufw allow 11112/tcp   # if you run ufw

# Find the server's LAN IP:
ip -4 addr show | grep inet

# On your laptop (or any DICOM-capable client):
pip install pynetdicom pydicom httpx
./bin/aegis-router-send-test --host 192.168.1.42 --port 11112
```

Any DICOM client works (`storescu`, `dcm4che`, your PACS's auto-route config) —
the router speaks standard DIMSE on `11112`. The router's web UI / operator
API is on `8080` from the server's IP.

### 3. University / research lab (thesis-grade)

For long-running research deployments — measure throughput, validate
de-identification accuracy, run modality-specific experiments. The "no-cloud"
mode is fine here too if you're staying purely local; otherwise enroll an
mTLS cert and forward to a cloud project you control.

Recommended extras for a research install:
- `--profile defacing --profile phi` to exercise the full de-id stack
- A bind-mount (`/srv/aegis-router:/var/lib/aegis-router`) onto research
  storage so the SQLite quarantine + audit log + study archive survives
  container rebuilds
- Scrape `:8080/metrics` from Prometheus for throughput tracking
  (`aegis_router_studies_received_total`, `aegis_router_pipeline_duration_seconds`)
- Use `bin/aegis-router-deid-diff` to produce tag-level evaluation tables for
  the paper

See [`docs/research-deployment.md`](docs/research-deployment.md) for the
detailed walkthrough (cohorts, accuracy harness, throughput benchmarking,
data retention).

## Quickstart (production spoke)

Pre-reqs at the spoke: a Linux host with Docker + Compose v2 and outbound
HTTPS to the AEGIS cloud receiver. ~4 GB RAM minimum for router-only;
8–16 GB recommended once defacing/analytics are enabled.

```bash
# 1. Get a per-site enrollment token from your AEGIS administrator.
#    (The cloud admin-dashboard will have an "Add spoke" form once #17 lands;
#    for now, see docs/spoke-deployment.md for the manual cert-issuance path.)

# 2. Clone the repo (or download the prebuilt compose bundle).
git clone https://github.com/aegis-imaging/aegis.git
cd aegis/router

# 3. Bootstrap (provisions certs, writes .env).
./bin/aegis-router-init --site-token=$AEGIS_ENROLLMENT_TOKEN

# 4. Bring it up. Minimal:
docker compose up -d
# Or with all capabilities enabled:
docker compose --profile full up -d

# 5. Verify.
curl http://localhost:8080/healthz
```

## Architecture

```
                              spoke site                                     │   cloud
                                                                             │
  ┌────────────┐     C-STORE      ┌──────────────────────────────┐           │
  │   PACS     │ ──────────────► │                              │           │
  └────────────┘   port 11112     │      AEGIS Router            │           │
                                   │                              │           │
  ┌────────────┐  /api/upload/*   │   ┌──────────────────────┐   │   mTLS    │   ┌─────────────────┐
  │  browser   │ ──────────────► │   │  per-study pipeline  │   │ ────────► │   │  AEGIS cloud    │
  │  (upload   │   port 8080      │   │  ────────────────    │   │           │   │   receiver      │
  │   portal)  │                  │   │  1. midi-b tag de-id │   │           │   │  (existing      │
  └────────────┘                  │   │  2. phi-detection*   │   │           │   │   /api/upload/* │
                                   │   │  3. defacing*        │   │           │   │   endpoint)     │
                                   │   │  4. ship to cloud    │   │           │   └─────────────────┘
                                   │   └──────────────────────┘   │           │
                                   │                              │           │
                                   │   ┌──────────────────────┐   │           │
                                   │   │  quarantine (SQLite) │   │           │
                                   │   └──────────────────────┘   │           │
                                   └──────────────────────────────┘           │
                                                                             │
                                   * = optional sidecar enabled via          │
                                       docker-compose --profile X            │
```

## Operator endpoints

When `OPERATOR_API_KEY` is set, all `/quarantine` and `/audit` endpoints
require `Authorization: Bearer <key>` or `X-Aegis-Operator-Key: <key>`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/healthz` | Liveness + sidecar status |
| GET | `/info` | Site identity + enabled capabilities |
| GET | `/quarantine` | List quarantined studies |
| GET | `/quarantine/{study_uid}` | Inspect one |
| POST | `/quarantine/{study_uid}/retry` | Retry the pipeline |
| DELETE | `/quarantine/{study_uid}` | Discard |
| GET | `/audit?event=pipeline.shipped` | Recent events |

## Testing

```bash
cd router
pip install -r requirements.txt -r requirements-test.txt
pip install -e ../midi-b
pytest -v
```

## Deployment notes

See [`docs/spoke-deployment.md`](docs/spoke-deployment.md) for the full
runbook: cert provisioning, firewall rules, PACS configuration, monitoring,
and the first-week operator checklist.
