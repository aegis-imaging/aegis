#!/usr/bin/env bash
# AEGIS on-premises installer.
#
# What this does:
#   1. Checks Docker + Docker Compose v2 are present
#   2. Sets up /srv/aegis with a sane directory layout
#   3. Generates random secrets the operator would otherwise have to invent
#   4. Drops in a .env file (only if one doesn't already exist)
#   5. Pulls images and starts the stack
#
# What this does NOT do:
#   - Configure TLS (use Caddy / nginx — see docs/deployment/on-premises.md)
#   - Configure SSO (see the Auth section in the same doc)
#   - Open firewall ports for DICOM
#
# Re-running is safe: existing .env / data / pgdata are preserved.

set -euo pipefail

AEGIS_HOME=${AEGIS_HOME:-/srv/aegis}
REPO_URL=${AEGIS_REPO_URL:-https://github.com/aegis-imaging/aegis.git}
REPO_BRANCH=${AEGIS_REPO_BRANCH:-develop}

log()   { printf '\033[1;36m▶\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m⚠\033[0m %s\n' "$*"; }
abort() { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || abort "missing required command: $1"
}

# ── 1. Preflight ────────────────────────────────────────────────────────────
require_cmd docker
docker compose version >/dev/null 2>&1 || abort "Docker Compose v2 is required (got: $(docker compose version 2>&1 || true))"
require_cmd openssl
require_cmd git

DOCKER_VERSION=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
log "Detected Docker $DOCKER_VERSION"

if [[ $EUID -ne 0 ]] && [[ ! -w "$(dirname "$AEGIS_HOME")" ]]; then
    abort "Run as root or with write access to $(dirname "$AEGIS_HOME")"
fi

# ── 2. Directory layout ─────────────────────────────────────────────────────
log "Setting up $AEGIS_HOME"
mkdir -p "$AEGIS_HOME"/{data,pgdata,backups}

if [[ ! -d "$AEGIS_HOME/src/.git" ]]; then
    log "Cloning $REPO_URL ($REPO_BRANCH) into $AEGIS_HOME/src"
    git clone --branch "$REPO_BRANCH" --depth 1 "$REPO_URL" "$AEGIS_HOME/src"
else
    log "Repo already present — pulling latest from $REPO_BRANCH"
    git -C "$AEGIS_HOME/src" fetch --depth 1 origin "$REPO_BRANCH"
    git -C "$AEGIS_HOME/src" reset --hard "origin/$REPO_BRANCH"
fi

# Surface the compose file in the AEGIS root so `docker compose` finds it.
cp "$AEGIS_HOME/src/docker-compose.yml" "$AEGIS_HOME/docker-compose.yml"

# ── 3. Secrets + .env ───────────────────────────────────────────────────────
ENV_FILE="$AEGIS_HOME/.env"
if [[ -f "$ENV_FILE" ]]; then
    log ".env already exists — keeping it"
else
    log "Generating .env at $ENV_FILE"
    DB_PASSWORD=$(openssl rand -hex 24)
    DIMSE_KEY=$(openssl rand -hex 24)
    cp "$AEGIS_HOME/src/docs/deployment/onprem.env.example" "$ENV_FILE"
    sed -i \
        -e "s|__GENERATE_DB_PASSWORD__|$DB_PASSWORD|" \
        -e "s|__GENERATE_DIMSE_KEY__|$DIMSE_KEY|" \
        "$ENV_FILE"
    chmod 600 "$ENV_FILE"
    log "Generated secrets — .env locked to 0600"
fi

# ── 4. Start stack ──────────────────────────────────────────────────────────
log "Pulling images (this can take a while on a fresh host)"
(cd "$AEGIS_HOME" && docker compose --env-file .env pull)

log "Starting the AEGIS stack"
(cd "$AEGIS_HOME" && docker compose --env-file .env up -d)

# ── 5. Health check ─────────────────────────────────────────────────────────
log "Waiting for /healthz (up to 90s)…"
HEALTH_OK=false
for _ in $(seq 1 30); do
    if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then
        HEALTH_OK=true
        break
    fi
    sleep 3
done

if $HEALTH_OK; then
    log "AEGIS is up."
else
    warn "/healthz did not respond — check 'docker compose logs api' for errors."
fi

cat <<EOF

AEGIS on-prem install complete.

  Admin dashboard:   http://localhost:3001
  Upload portal:     http://localhost:3000
  API health:        http://localhost:8080/healthz
  DICOM C-STORE:     localhost:11112 (AE Title: AEGIS)

Next:
  1. Open .env at $ENV_FILE and review the auth settings.
  2. Put a TLS reverse proxy in front of port 8080 / 3000 / 3001
     (Caddy one-liner in docs/deployment/on-premises.md).
  3. Open TCP 11112 to your on-site PACS.
  4. Run scripts/cloud_smoke_test.py against http://localhost:8080
     to validate the full pipeline.

Logs:     docker compose -f $AEGIS_HOME/docker-compose.yml --env-file $ENV_FILE logs -f
Backup:   /etc/cron.daily/aegis-backup (see on-premises.md §"Backup & restore")
EOF
