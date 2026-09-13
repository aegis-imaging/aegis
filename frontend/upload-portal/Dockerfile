# Build context: repo root
# docker build -f frontend/upload-portal/Dockerfile -t <image> .
#
# The upload-portal imports @aegis/client (a local workspace package), so the
# build context must be the monorepo root to give npm access to the root
# package-lock.json and the client/ workspace package.

FROM node:20-alpine AS build

WORKDIR /app

# Copy root workspace manifest + lockfile so npm ci can resolve workspace deps
COPY package.json package-lock.json ./

# Copy only the workspace packages required for this build
COPY client/ client/
COPY frontend/upload-portal/ frontend/upload-portal/

# Install dependencies for the upload-portal workspace only.
# npm ci resolves @aegis/client via the workspace symlink (client/ above).
RUN npm ci --workspace=@aegis/upload-portal --ignore-scripts

WORKDIR /app/frontend/upload-portal
RUN npm run build

FROM nginx:1.31-alpine AS runtime
# gettext provides envsubst, used to hydrate API_URL into nginx config at startup.
RUN apk add --no-cache gettext

# Default: GCP API URL. Override with API_URL env var for other clouds (e.g. AWS ECS).
ENV API_URL=https://api.aegisimaging.ai

COPY frontend/upload-portal/nginx.conf /etc/nginx/conf.d/nginx.conf.template
COPY --from=build /app/frontend/upload-portal/dist /usr/share/nginx/html

EXPOSE 8080
# Substitute ${API_URL} in the nginx config template, then start nginx.
CMD ["/bin/sh", "-c", "envsubst '${API_URL} ${NGINX_RESOLVER_DIRECTIVE}' < /etc/nginx/conf.d/nginx.conf.template > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"]
