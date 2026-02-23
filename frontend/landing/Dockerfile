# Build context: repo root
# docker build -f frontend/landing/Dockerfile -t <image> .
#
# The landing page imports @aegis/client (a local workspace package), so the
# build context must be the monorepo root to give npm access to the root
# package-lock.json and the client/ workspace package.

# Stage 1: Build the React/Vite app
FROM node:22-slim AS build

WORKDIR /app

# Copy root workspace manifest + lockfile so npm ci can resolve workspace deps
COPY package.json package-lock.json ./

# Copy only the workspace packages required for this build
COPY client/ client/
COPY frontend/landing/ frontend/landing/

# Install dependencies for the landing workspace only.
# npm ci resolves @aegis/client via the workspace symlink (client/ above).
RUN npm ci --workspace=@aegis/landing --ignore-scripts

# Vite bakes env vars at build time — pass API and OHIF URLs as build args.
# Defaults point to production; override with --build-arg when building for other environments.
ARG VITE_API_BASE_URL=https://api.aegisimaging.ai
ARG VITE_OHIF_BASE_URL=https://ohif.aegisimaging.ai
# VITE_INVITE_GATE_ENABLED: set to 'true' to enable the server-side invite code gate.
# Leave 'false' (default) for a fully public site.
# Codes are managed via POST /api/invite-codes and validated server-side — no secret
# is baked into the bundle.
# Example: --build-arg VITE_INVITE_GATE_ENABLED=true
ARG VITE_INVITE_GATE_ENABLED=false
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_OHIF_BASE_URL=$VITE_OHIF_BASE_URL
ENV VITE_INVITE_GATE_ENABLED=$VITE_INVITE_GATE_ENABLED

# Build the Vite app
WORKDIR /app/frontend/landing
RUN npm run build

# Stage 2: Serve with nginx
FROM nginx:1.27-alpine

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf

COPY frontend/landing/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/frontend/landing/dist /usr/share/nginx/html

# Cloud Run listens on 8080
EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
