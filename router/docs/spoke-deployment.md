# Spoke Router Deployment Runbook

Step-by-step for an IT administrator standing up an AEGIS Router at a new
spoke site. Aimed at someone comfortable with Docker but new to AEGIS.

## Prerequisites

- Linux host (Ubuntu 22.04+ / RHEL 9 / Debian 12 tested), 4 vCPU / 8 GB RAM
  minimum (router-only); 8 vCPU / 16 GB / 100 GB scratch disk recommended once
  defacing or analytics are enabled.
- Docker Engine 24+ and Compose v2.
- Outbound HTTPS (TCP/443) to your AEGIS cloud URL.
- Inbound TCP/11112 reachable from your PACS (DICOM C-STORE).
- Inbound TCP/8080 from staff workstations (web upload UI). Either keep this
  on a closed LAN or front it with a TLS-terminating reverse proxy.

## Step 1 — get an enrollment token

The AEGIS cloud admin issues a short-lived enrollment token per spoke from
the admin-dashboard ("Spokes → Add new"). The token encodes the cloud URL,
project slug, and a one-time secret that the bootstrap CLI exchanges for a
long-lived mTLS client certificate.

> **Manual fallback** (until the admin-dashboard spoke management page lands):
> the cloud admin can run `gh workflow run mint-spoke-cert ...` (see
> [`docs/runbooks/mint-spoke-cert.md`](../../docs/runbooks/) — to be added)
> and hand the resulting `client.crt` + `client.key` + `ca.crt` to the site IT
> team out-of-band.

## Step 2 — bootstrap

```bash
git clone https://github.com/aegis-imaging/aegis.git
cd aegis/router

./bin/aegis-router-init --site-token="$AEGIS_ENROLLMENT_TOKEN" \
                        --site-name="Example General Hospital"
```

The init script:
1. Exchanges the enrollment token with the cloud for a per-spoke client cert.
2. Writes `certs/client.crt`, `certs/client.key`, `certs/ca.crt`.
3. Writes a starter `.env` with the site name, cloud URL, and a fresh
   MIDI-B salt baked in.
4. Prints next-step guidance.

## Step 3 — decide on capabilities

Edit `.env` to enable the sidecars you want. Pair each with the matching
compose profile when bringing the stack up.

Common patterns:

| Site profile | What to enable |
|---|---|
| Forward-only (smallest) | nothing extra — tag de-id only |
| Forward + face de-id | `DEFACING_SERVICE_URL=http://defacing:8080` + `--profile defacing` |
| Forward + face de-id + pixel scrub | both `DEFACING_SERVICE_URL` and `PHI_DETECTION_SERVICE_URL` + `--profile defacing --profile phi` |
| Local analytics + cloud forwarding | add `--profile analytics` (heavy — needs more RAM/disk) |

## Step 4 — start

```bash
docker compose --profile defacing --profile phi up -d
docker compose logs -f router
```

Verify:

```bash
curl http://localhost:8080/healthz | jq
curl http://localhost:8080/info    | jq
```

## Step 5 — point your PACS at it

On the PACS:
- AE Title: whatever you set in `DIMSE_AE_TITLE` (default `AEGIS_ROUTER`)
- Host: the router host's IP
- Port: `11112`

Send a single test study and confirm:

```bash
# Should show one entry, no failures.
docker compose logs router | grep "pipeline.shipped"
```

## Step 6 — daily operations

Check the quarantine queue:
```bash
curl -H "Authorization: Bearer $OPERATOR_API_KEY" \
     http://localhost:8080/quarantine | jq
```

Retry a quarantined study after fixing the root cause:
```bash
curl -X POST -H "Authorization: Bearer $OPERATOR_API_KEY" \
     http://localhost:8080/quarantine/<study_uid>/retry
```

Recent pipeline events:
```bash
curl -H "Authorization: Bearer $OPERATOR_API_KEY" \
     "http://localhost:8080/audit?limit=50" | jq
```

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `/healthz` returns `degraded` with `scp: not_running` | port 11112 already in use | Stop the conflicting service or change `DIMSE_PORT` |
| Studies hit quarantine with `failed_stage=ship` and "certificate verify failed" | corporate proxy doing TLS inspection | Add the proxy CA bundle to `CLOUD_CA_CERT` or have the proxy exempt the cloud URL |
| `failed_stage=phi_scrub`, "tesseract not available" | running `--profile phi` without the sidecar built | `docker compose --profile phi build phi-detection` |
| Defacing takes >10 min per study | running `mri_deface` (CPU-only) on a large study | Switch backend to `deepdefacer` (still CPU but ~3× faster) or add a GPU |
| Cloud forwarding silently disabled | one of `CLOUD_RECEIVER_URL` / `CLOUD_CLIENT_CERT` / `CLOUD_CLIENT_KEY` missing | All three must be set together |

## Upgrading

```bash
cd aegis
git pull
cd router
docker compose --profile <your-profiles> up -d --build
```

Local SQLite quarantine + study data persists in the `router-data` named
volume across upgrades.

## Decommissioning

```bash
docker compose down
docker volume rm router_router-data   # WARNING: deletes quarantined studies
rm -rf certs/                          # revoke at the cloud side too
```

The cloud admin should revoke the spoke's cert in the admin-dashboard once
decommissioning is complete.
