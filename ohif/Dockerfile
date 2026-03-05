# ── OHIF Viewer — Docker image ────────────────────────────────────────────────
#
# Uses the official ohif/app pre-built image as the base so we avoid a lengthy
# Node.js build step in CI.  The AEGIS-specific config (app-config.js) and the
# nginx reverse-proxy template are overlaid at image build time; the API_URL
# backend address is substituted at container startup via docker-entrypoint.sh.
#
# Build:
#   docker build -t ohif .
#
# Run locally (Docker Compose injects API_URL=http://api:8080):
#   docker run -e API_URL=http://api:8080 -p 3006:8080 ohif
# ──────────────────────────────────────────────────────────────────────────────

FROM ohif/app:v3.11.0

# gettext provides envsubst for runtime API_URL substitution in nginx config
RUN apk add --no-cache gettext || apt-get install -y gettext-base

# Overlay AEGIS-specific OHIF configuration
# app-config.js defines the two AEGIS data sources (aegis-clean + aegis-raw)
# and disables the OHIF worklist (AEGIS controls study navigation).
COPY app-config.js /usr/share/nginx/html/app-config.js

# nginx config template — ${API_URL} is substituted at container startup.
# Adds DICOMweb proxy routes (/dicomweb/ and /dicomweb-raw/) and the
# frame-ancestors CSP header required for iframe embedding in admin dashboard.
COPY nginx.conf.template /nginx.conf.template

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
