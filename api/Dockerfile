FROM golang:1.27 AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -p 2 -o /aegis-api .
# Pre-create writable data directories for Docker Compose local dev.
RUN mkdir -p /app/data-init/dicom/raw /app/data-init/dicom/clean \
             /app/data-init/synth /app/data-init/bids

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /aegis-api /aegis-api
# Pre-create the local storage directory structure owned by the nonroot user
# (uid 65532) so Docker volume initialisation copies these directories into a
# new named volume with correct ownership — allowing the nonroot API process to
# create sub-directories and write DICOM files without root privileges.
# Production Cloud Run uses GCS storage (STORAGE_MODE=gcs) so this path is
# only exercised in Docker Compose local dev.
COPY --from=builder --chown=65532:65532 /app/data-init /app/data
EXPOSE 8080
ENTRYPOINT ["/aegis-api"]
