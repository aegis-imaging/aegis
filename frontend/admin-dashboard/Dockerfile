# Build context: repo root
# docker build -f frontend/admin-dashboard/Dockerfile -t <image> .
#
# The admin-dashboard imports @aegis/client (a local workspace package), so the
# build context must be the monorepo root to give npm access to the root
# package-lock.json and the client/ workspace package.

FROM node:20-alpine AS build

WORKDIR /app

# Copy root workspace manifest + lockfile so npm ci can resolve workspace deps
COPY package.json package-lock.json ./

# Copy only the workspace packages required for this build
COPY client/ client/
COPY frontend/admin-dashboard/ frontend/admin-dashboard/

# Install dependencies for the admin-dashboard workspace only.
# npm ci resolves @aegis/client via the workspace symlink (client/ above).
RUN npm ci --workspace=@aegis/admin-dashboard --ignore-scripts

# VITE_DWV_BASE_URL is baked into the static build at compile time.
# Set this to the DWV Cloud Run service URL for production builds.
ARG VITE_DWV_BASE_URL
ENV VITE_DWV_BASE_URL=$VITE_DWV_BASE_URL

# VITE_OHIF_BASE_URL is baked into the static build at compile time.
# Set this to the OHIF Cloud Run service URL for production builds.
ARG VITE_OHIF_BASE_URL
ENV VITE_OHIF_BASE_URL=$VITE_OHIF_BASE_URL

WORKDIR /app/frontend/admin-dashboard
RUN npm run build

FROM nginx:1.27-alpine AS runtime
# gettext provides envsubst, used to hydrate API_URL into nginx config at startup.
RUN apk add --no-cache gettext

# Default: GCP API URL. Override with API_URL env var for other clouds (e.g. AWS ECS).
ENV API_URL=https://api.aegisimaging.ai
# Default: GCP MCP server URL. Override with MCP_SERVER_URL env var for other clouds (e.g. AWS ECS).
ENV MCP_SERVER_URL=https://aegis-mcp-server-uk5cvzf5nq-uc.a.run.app

COPY nginx.conf /etc/nginx/conf.d/nginx.conf.template
COPY --from=build /app/frontend/admin-dashboard/dist /usr/share/nginx/html

EXPOSE 8080
# Substitute ${API_URL} in the nginx config template, then start nginx.
CMD ["/bin/sh", "-c", "envsubst '${API_URL} ${MCP_SERVER_URL} ${NGINX_RESOLVER_DIRECTIVE}' < /etc/nginx/conf.d/nginx.conf.template > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"]
