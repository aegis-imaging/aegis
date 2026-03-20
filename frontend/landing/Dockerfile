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

# Vite bakes env vars at build time — pass API URL as build arg.
# Defaults point to production; override with --build-arg when building for other environments.
ARG VITE_API_BASE_URL=https://api.aegisimaging.ai
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ARG VITE_UPLOAD_PORTAL_URL=https://upload.aegisimaging.ai
ENV VITE_UPLOAD_PORTAL_URL=$VITE_UPLOAD_PORTAL_URL
# Gate is now server-side — VITE_INVITE_GATE_ENABLED is no longer needed.
# The React-side hook stays in the code but is a no-op (GATE_ENABLED env
# var on the server controls access; the client always sees the full app).

# Build the Vite app
WORKDIR /app/frontend/landing
RUN npm run build

# Stage 2: Build the Go gate server
FROM golang:1.24 AS server

WORKDIR /srv
COPY frontend/landing/server/go.mod ./
RUN go mod download
COPY frontend/landing/server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /gate-server .

# Stage 3: Final image (distroless — no shell, minimal attack surface)
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=server /gate-server /gate-server
COPY --from=build /app/frontend/landing/dist /app/dist

# Runtime env vars (set in Cloud Run):
#   GATE_ENABLED=true
#   GATE_SECRET=<random 32+ char secret>
#   API_BASE_URL=https://api.aegisimaging.ai  (default)

EXPOSE 8080
ENTRYPOINT ["/gate-server"]
