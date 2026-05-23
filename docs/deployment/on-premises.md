# AEGIS On-Premises Deployment Guide

For institutions with data-residency or air-gap requirements that prevent
deploying to public cloud (GCP / AWS / Azure). The same AEGIS container images
run on-premises with no code changes — every cloud-specific integration
(IAP, Cognito, GCS, S3, Blob) has a local-mode equivalent already in the
codebase.

This guide targets:
- A single Linux host (Debian 12 / Ubuntu 22.04 / RHEL 9) running Docker
  Compose
- 16 GB RAM, 8 vCPU, 500 GB disk minimum for production pilot
- No outbound internet access required after the initial image pull
- DICOM C-STORE accessible to on-site PACS over TCP 11112

Skip to:
- [Prerequisites](#prerequisites)
- [Install](#install)
- [Configuration](#configuration)
- [Smoke test](#smoke-test)
- [Backup & restore](#backup--restore)
- [Upgrade](#upgrade)
- [Troubleshooting](#troubleshooting)
- [Air-gapped install](#air-gapped-install)

## Prerequisites

| Component | Minimum |
|---|---|
| OS | Debian 12, Ubuntu 22.04, or RHEL 9 |
| CPU | 8 vCPU |
| RAM | 16 GB |
| Disk | 500 GB (DICOM-store volume); SSD strongly recommended |
| Docker | 24.x or later |
| Docker Compose v2 | 2.20 or later |
| Network | TCP 11112 reachable from PACS; TCP 443 reachable from clinical workstations |

The whole stack runs as Docker containers — no host Postgres, no host Python.

### Disk layout

```
/srv/aegis/
├── docker-compose.yml             # copied from the repo
├── .env                           # local config (NOT in git)
├── data/                          # DICOM bytes (raw/, clean/)
└── pgdata/                        # PostgreSQL data dir
```

The `data/` directory holds all DICOM files (raw + de-identified). It will
grow as studies arrive. Size it for at least 12 months of expected
ingestion volume.

## Install

### Option A — guided installer (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/aegis-imaging/aegis/develop/scripts/onprem_install.sh -o onprem_install.sh
sudo bash onprem_install.sh
```

The installer:
1. Checks Docker / Compose versions
2. Creates `/srv/aegis/` with the right ownership
3. Drops a `docker-compose.onprem.yml` and a `.env.example` you can customise
4. Generates random secrets (`DB_PASSWORD`, `DIMSE_OPERATOR_API_KEY`) so you
   don't have to invent them yourself
5. Pulls images and runs migrations
6. Prints next-step URLs

### Option B — manual

```bash
sudo mkdir -p /srv/aegis && sudo chown $(whoami) /srv/aegis && cd /srv/aegis
git clone https://github.com/aegis-imaging/aegis.git src
cp src/docker-compose.yml .
cp src/docs/deployment/onprem.env.example .env
$EDITOR .env                                    # set the secrets
docker compose --env-file .env up -d
```

The default `docker-compose.yml` already runs everything on-host with no
cloud dependencies — `STORAGE_MODE=local`, `AUTH_ENABLED=false`,
`SMTP_HOST=` (silent no-op). The only edits needed are the security
hardening fields in `.env`.

## Configuration

The `.env` file is the only thing you maintain. Everything else is the
upstream `docker-compose.yml`.

### Mandatory settings

```bash
# PostgreSQL — change the default before going live.
DB_PASSWORD=<generate with: openssl rand -hex 24>

# DIMSE operator API key — guards the /ingest/retry* endpoints.
DIMSE_OPERATOR_API_KEY=<generate with: openssl rand -hex 24>

# Bind the admin UI to a real hostname so cookies and CORS work.
PUBLIC_BASE_URL=https://aegis.your-institution.local

# Reverse-proxy auth header that identifies the logged-in user. See §Auth.
AUTH_ENABLED=true
AUTH_PROVIDER=header
DEV_USER_EMAIL=                                  # leave empty when AUTH_ENABLED=true
```

### Storage

The defaults map both DICOM bytes and Postgres data to host paths so
backups are easy:

```bash
STORAGE_MODE=local
LOCAL_STORAGE_DIR=/app/data                      # bind-mounted to ./data
DIMSE_DATA_DIR=/app/data
```

If you have an existing institutional NFS/SMB share for medical imaging,
mount it at `/srv/aegis/data` (or wherever the compose file binds the
`aegis-data` volume) and AEGIS will write all DICOMs there.

### Auth

On-prem auth has three supported modes:

| Mode | When to use | How |
|---|---|---|
| Dev (auto-auth) | Initial setup, lab/research only | `AUTH_ENABLED=false`; every request authenticates as `DEV_USER_EMAIL`. Do **not** ship this to production. |
| Reverse-proxy header | LDAP / SSO via nginx-auth, oauth2-proxy, Authentik, Authelia | `AUTH_PROVIDER=header`; the proxy injects `X-Forwarded-Email` (or a configurable header — see below). AEGIS trusts the header and looks up the user in `admin_users`. |
| API key only | Machine-to-machine / appliance mode | All admin endpoints accept `Authorization: Bearer aegis_...`. Create keys via `POST /api/api-keys`. |

For the reverse-proxy header path, point your existing SSO at AEGIS:

```nginx
# nginx in front of AEGIS — terminate TLS here, hand off identity downstream.
location / {
    auth_request /oauth2/auth;
    auth_request_set $email $upstream_http_x_auth_request_email;
    proxy_set_header X-Goog-Authenticated-User-Email accounts.google.com:$email;
    proxy_pass http://localhost:8080;
}
```

AEGIS reads the same `X-Goog-Authenticated-User-Email` header it reads
in GCP/IAP — the format is `<issuer>:<email>` and the issuer prefix is
stripped — so most SSO proxies work without code changes.

### Email (optional)

Email is silent no-op by default. To enable:

```bash
SMTP_HOST=smtp.your-institution.local
SMTP_PORT=587
SMTP_USERNAME=aegis@your-institution.local
SMTP_PASSWORD=<set>
SMTP_FROM="AEGIS <noreply@your-institution.local>"
```

For local-only testing, drop in Mailpit (`docker compose up mailpit` —
already in the compose file) and point `SMTP_HOST=mailpit`,
`SMTP_PORT=1025`.

### TLS

The compose file binds plain HTTP. Put a reverse proxy in front of it
for TLS termination — typical stack:

```
PACS         ───→  TCP 11112 → AEGIS dimse-receiver  (no TLS — DICOM TLS optional)
Clinicians   ───→  HTTPS 443 → nginx/Caddy → AEGIS API/UI on 8080/3000/3001
```

A Caddy one-liner that auto-issues certs from your institutional CA or
Let's Encrypt:

```caddy
aegis.your-institution.local {
    reverse_proxy /api/* localhost:8080
    reverse_proxy /admin/* localhost:3001
    reverse_proxy /* localhost:3000
}
```

## Smoke test

After `docker compose up -d`, validate each layer:

```bash
# 1. API health — should return all services healthy
curl -s http://localhost:8080/healthz | jq

# 2. DIMSE C-ECHO from on-host (validates SCP is listening)
docker run --rm --network=host pydicom/dcmtk \
    echoscu -aec AEGIS localhost 11112

# 3. Upload a synthetic study end-to-end
curl -X POST http://localhost:8080/api/studies/generate-synthetic \
    -H 'Content-Type: application/json' \
    -d '{"project_slug":"default","slices":20,"size":128,"seed":1}' | jq
# Should return a study_uid and trigger the full pipeline.

# 4. Watch the pipeline progress (study should reach `approved` if rules permit)
curl -s 'http://localhost:8080/api/studies?limit=1' | jq '.studies[0] | {status, modality, body_part}'
```

If all four pass, the on-prem install is wired up correctly. The cloud
smoke test (`scripts/cloud_smoke_test.py`) also works against an
on-prem instance — run it with `BASE_URL=http://localhost:8080`.

## Backup & restore

Two things hold state:

1. **Postgres** in the `pgdata` volume — application records, users,
   audit trail, routing rules, etc.
2. **DICOM bytes** in the `aegis-data` volume — raw + clean studies.

### Daily backup (recommended cron)

```bash
#!/bin/bash
# /etc/cron.daily/aegis-backup
set -euo pipefail

DATE=$(date +%Y%m%d-%H%M)
BACKUP_DIR=/srv/backups/aegis/$DATE
mkdir -p "$BACKUP_DIR"

# Postgres — atomic SQL dump
docker exec aegis-postgres-1 pg_dump -U aegis -F c aegis \
    > "$BACKUP_DIR/aegis.dump"

# DICOM bytes — rsync with --delete to a sibling tree (use --link-dest
# for hardlink-based incremental snapshots)
rsync -a --delete /srv/aegis/data/ "$BACKUP_DIR/data/"

# Compress + offload (replace with your institutional backup target)
tar czf "/srv/backups/aegis-$DATE.tar.gz" -C "/srv/backups/aegis" "$DATE"
rm -rf "$BACKUP_DIR"
```

### Restore

```bash
# 1. Stop the stack
docker compose down

# 2. Restore the Postgres dump and the data tree
tar xzf /srv/backups/aegis-YYYYMMDD-HHMM.tar.gz -C /tmp
docker compose up -d postgres
docker exec -i aegis-postgres-1 \
    pg_restore -U aegis -d aegis --clean < /tmp/YYYYMMDD-HHMM/aegis.dump
rsync -a --delete /tmp/YYYYMMDD-HHMM/data/ /srv/aegis/data/

# 3. Start everything else
docker compose up -d
```

DICOMweb and the admin dashboard should resume against the restored
state with no further action.

## Upgrade

AEGIS uses semantic-version tags on its container images.

```bash
cd /srv/aegis
# Update the version pin in your .env (e.g., AEGIS_VERSION=1.4.0)
docker compose pull
docker compose up -d

# Postgres migrations run automatically on api startup.
docker compose logs -f api | grep migration
```

Always take a backup *before* upgrading. Migrations are forward-only —
to roll back a release you need to restore from backup.

## Troubleshooting

| Symptom | Most likely cause | Fix |
|---|---|---|
| `/healthz` returns `degraded` | A sidecar isn't healthy | `docker compose logs <service>`; the response payload names which dependency failed |
| C-STORE from PACS times out | TCP 11112 blocked by host firewall | `sudo ufw allow 11112/tcp` (Ubuntu) or open in `firewalld` |
| Studies stuck in `received` | No routing rules configured | Open the admin dashboard, add a routing rule (e.g., `auto_approve` for testing) |
| DICOM file write fails | `LOCAL_STORAGE_DIR` not writable | `ls -la /srv/aegis/data` — should be writable by container UID `1000` |
| API can't reach Postgres | `DB_PASSWORD` mismatch between API and Postgres containers | Ensure both come from the same `.env` |
| `auth: insufficient permissions` for the first admin | Default `admin_users` row uses `ai@aegisimaging.ai` | `INSERT INTO admin_users (email,name,role,enabled) VALUES ('you@inst.local','You','admin',true);` via `docker exec aegis-postgres-1 psql -U aegis` |
| OOM on inference | Sidecar container memory limit too low | Bump `defacing` / `analytics-service` memory in the compose file |

### Useful diagnostic commands

```bash
# Live logs across all services
docker compose logs -f --tail=50

# Single service
docker compose logs -f api

# Postgres console
docker exec -it aegis-postgres-1 psql -U aegis

# DIMSE retry queue snapshot
curl -s http://localhost:8080/api/dimse/retry \
     -H "X-AEGIS-Operator-Key: $DIMSE_OPERATOR_API_KEY" | jq

# Disk usage by store
du -sh /srv/aegis/data/dicom/raw /srv/aegis/data/dicom/clean
```

## Air-gapped install

For deployments with no internet access at install time:

1. On a connected staging host, pull and save every image referenced by
   the compose file:
   ```bash
   docker compose -f docker-compose.yml config --images | \
       xargs -I{} docker save -o /tmp/aegis-images.tar {}
   tar cf /tmp/aegis-bundle.tar -C /tmp aegis-images.tar
   # Add the repo tree:
   tar rf /tmp/aegis-bundle.tar -C /tmp/aegis-repo .
   gzip /tmp/aegis-bundle.tar
   ```
2. Transfer `aegis-bundle.tar.gz` to the air-gapped target (USB, signed
   transfer drive, etc.).
3. On the target:
   ```bash
   mkdir -p /srv/aegis && tar xzf aegis-bundle.tar.gz -C /srv/aegis
   docker load -i /srv/aegis/aegis-images.tar
   cd /srv/aegis && cp docs/deployment/onprem.env.example .env
   $EDITOR .env
   docker compose --env-file .env up -d
   ```

After the initial bundle, image upgrades follow the same `docker save`
+ `docker load` pattern — there is no `docker compose pull` on the
target.

## What's *not* supported on-prem

These cloud-only features no-op or are disabled on an on-prem install:

- IAP / Cognito / Easy Auth — replace with the reverse-proxy header
  pattern described in [Auth](#auth)
- GCP Cloud Build / AWS CodeBuild / Azure DevOps auto-deploys — on-prem
  upgrades are manual (`docker compose pull && up -d`)
- Cloud Monitoring alert policies — substitute Prometheus +
  Alertmanager or your institutional NMS
- Cloud Secret Manager — secrets live in `/srv/aegis/.env` (chmod 600,
  owned by root)
- HIPAA BAA with AWS/GCP/Azure — N/A; your institution owns the BAA
  relationship

Everything else — the upload portal, admin dashboard, DICOMweb proxy,
DIMSE receiver, all 11 processing sidecars, MCP server, agent
orchestrator, audit trail, webhook subscriptions, federation peer
registry, study labels, cohort reports — works identically to a cloud
deployment.
