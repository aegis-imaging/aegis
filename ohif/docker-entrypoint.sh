#!/bin/sh
set -e

# Substitute API_URL into the OHIF app config.
# Default to localhost for local Docker Compose usage.
API_URL="${API_URL:-http://localhost:8080}"

envsubst < /app-config.js.template > /usr/share/nginx/html/app-config.js

exec "$@"
