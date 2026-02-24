# ── Stage 1: Build OHIF from source ─────────────────────────────────────────
FROM node:20-alpine AS builder

RUN apk add --no-cache git python3 make g++

WORKDIR /ohif

# Pin to a specific stable OHIF release tag.
# To upgrade: check https://github.com/OHIF/Viewers/releases for the latest stable tag.
ARG OHIF_VERSION=v3.9.0
RUN git clone --depth 1 --branch ${OHIF_VERSION} \
    https://github.com/OHIF/Viewers.git .

# Enable yarn workspaces (required by OHIF monorepo)
RUN yarn config set workspaces-experimental true

# Install dependencies — this layer is cached unless the lockfile changes
RUN yarn install --frozen-lockfile

# Bake our custom config in before the build so it ends up as dist/app-config.js.
# Copying it here (into public/) means OHIF's Vite pipeline picks it up directly
# rather than us racing a post-build COPY against whatever OHIF already put there.
COPY app-config.js platform/app/public/app-config.js

# Build production output → platform/app/dist/
RUN yarn run build

# ── Stage 2: Serve with nginx ─────────────────────────────────────────────────
FROM nginx:1.27-alpine

# gettext provides envsubst for runtime API_URL substitution in nginx config
RUN apk add --no-cache gettext

# Serve OHIF static files from the build output
COPY --from=builder /ohif/platform/app/dist /usr/share/nginx/html

# nginx config template — ${API_URL} is substituted at container startup by the entrypoint
COPY nginx.conf.template /nginx.conf.template

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
