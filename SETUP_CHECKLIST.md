# AEGIS — Setup Checklist

Personal environment setup tasks for building the MVP/POC. Complete these in order — each section unblocks the next.

---

## 1. Local Development Environment

- [ ] Install Go 1.23+ (`brew install go`)
- [ ] Install Terraform 1.5+ (`brew install terraform`)
- [ ] Verify Node.js 20+ and npm are installed (`node --version`)
- [ ] Install Docker Desktop (for building/testing containers locally)
- [ ] Clone the AEGIS repo and verify the monorepo structure

## 2. GCP Project Setup

- [ ] Create a new GCP project (e.g., `aegis-dev`) in your personal GCP account
- [ ] Link a billing account to the project (free tier covers most dev usage)
- [ ] Install the gcloud CLI (`brew install google-cloud-sdk`)
- [ ] Authenticate: `gcloud auth login` and `gcloud auth application-default login`
- [ ] Set default project: `gcloud config set project aegis-dev`

## 3. Terraform — Project Bootstrap

- [ ] Copy `terraform/project/terraform.tfvars.example` to `terraform/project/terraform.tfvars`
- [ ] Fill in your `project_id`, `region`, and `billing_account`
- [ ] Run `terraform init` in `terraform/project/`
- [ ] Run `terraform plan` and review the output
- [ ] Run `terraform apply` to enable all required GCP APIs
- [ ] Verify APIs are enabled: `gcloud services list --enabled`

## 4. Terraform — Infrastructure

- [ ] Copy `terraform/infra/terraform.tfvars.example` to `terraform/infra/terraform.tfvars`
- [ ] Fill in your `project_id` and `region`
- [ ] Run `terraform init` in `terraform/infra/`
- [ ] Run `terraform plan` and review
- [ ] Run `terraform apply` to create Healthcare API DICOM stores, GCS bucket, Pub/Sub
- [ ] Verify DICOM store exists: `gcloud healthcare dicom-stores list --dataset=aegis --location=us-central1`
- [ ] Verify staging bucket exists: `gsutil ls`

## 5. Sample DICOM Data for Local Testing

- [ ] Download sample brain MRI DICOM files for testing (options below):
  - TCIA (The Cancer Imaging Archive): https://www.cancerimagingarchive.net/
  - dicom.offis.de sample files: https://dicom.offis.de/dcmtk/
  - NEMA WG-6 sample DICOM: ftp://medical.nema.org/medical/dicom/DataSets/
- [ ] Place sample files in a local test directory (not committed to git)
- [ ] Verify files open in a DICOM viewer (e.g., Horos, 3D Slicer) to confirm validity

## 6. Frontend — Upload Portal

- [ ] `cd frontend/upload-portal && npm install`
- [ ] `npm run dev` — verify it runs on http://localhost:3000
- [ ] Install DICOM libraries: `npm install dcmjs dicom-parser`
- [ ] Test parsing a sample DICOM file in the browser console

## 7. Frontend — Admin Dashboard

- [ ] `cd frontend/admin-dashboard && npm install`
- [ ] `npm run dev` — verify it runs on http://localhost:3001
- [ ] Click **View** on any study row → OHIF Viewer iframe appears inline
- [ ] Click **Open in new tab ↗** → viewer opens in a new browser tab

## 8. Go API

- [ ] `cd api && go run .` — verify health endpoint at http://localhost:8080/healthz
- [ ] Add GCP SDK dependencies: `go get cloud.google.com/go/storage cloud.google.com/go/healthcare`
- [ ] Implement signed URL generation endpoint
- [ ] Test upload flow: browser → signed URL → GCS staging bucket

## 8a. Local Services (Docker Compose)

`docker-compose.yml` in the repo root starts all local dev dependencies in one command:

- [ ] `docker compose up -d` — starts postgres (5432), mailpit (1025/8025), OHIF (3002)
- [ ] Verify OHIF loads at http://localhost:3002 (shows the AEGIS data source)
- [ ] Verify Mailpit web UI at http://localhost:8025

Or start services individually:
```bash
docker compose up -d postgres
docker compose up -d mailpit
docker compose up -d ohif
```

## 8b. Email (Local Dev with Mailpit)

Email is disabled by default — all calls are silent no-ops when `SMTP_HOST` is unset.
Mailpit is included in `docker-compose.yml` (step 8a). To run standalone:

Email is disabled by default — all calls are silent no-ops when `SMTP_HOST` is unset.

- [ ] Run Mailpit (or use `docker compose up -d mailpit`):
  ```bash
  docker run -p 1025:1025 -p 8025:8025 axllent/mailpit
  ```
- [ ] Start the API with SMTP env vars:
  ```bash
  cd api && SMTP_HOST=localhost SMTP_PORT=1025 go run .
  ```
- [ ] Open Mailpit web UI at http://localhost:8025
- [ ] Trigger each notification to verify:
  - Upload a study via the portal — enter your email in the "Your email (optional)" field on the preview screen
  - Approve the study in the admin dashboard → uploader gets approval email
  - Reject a study → uploader gets rejection email
  - Create a share for an approved study → recipient gets share email with download link
- [ ] Verify no crash when `SMTP_HOST` is unset (email silently skipped, no error returned)
- [ ] Verify upload with no email entered still completes successfully (no notification sent)

## 9. GitHub Repository

- [ ] Verify remote is set: `git remote -v`
- [ ] Push monorepo scaffold to `develop` branch
- [ ] Set up branch protection on `main` (require PR reviews)
- [ ] (Optional) Set up GitHub Actions for CI (lint, typecheck, Go test)

## 10. Future — Before Proposing to Work

- [ ] Have a working end-to-end demo: upload → anonymize → view in OHIF
- [ ] Prepare a 5-minute screen recording of the demo flow
- [ ] Draft a one-page proposal covering: problem, solution, differentiation, cost estimate
- [ ] Identify potential pilot users / departments at your institution
- [ ] Research your company's internal startup / innovation program requirements

---

*Generated 2026-02-17. See ARCHITECTURE.md for the full system design.*
