#!/bin/sh
set -e

# Default API backend for local Docker Compose (internal network hostname).
# Override with API_URL env var when deploying (e.g. https://api.aegisimaging.ai).
API_URL="${API_URL:-http://api:8080}"

# Substitute ${API_URL} in the nginx template.
# Specifying the variable name prevents envsubst from mangling nginx's own $variables.
envsubst '${API_URL}' < /nginx.conf.template > /etc/nginx/conf.d/default.conf

exec "$@"
