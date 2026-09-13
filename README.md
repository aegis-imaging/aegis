# AEGIS

![AEGIS Logo](logo-small.png)

**Anonymization & Exchange Gateway for Imaging Studies**

A multi-cloud platform for secure, HIPAA-compliant sharing of medical imaging data (all DICOM modalities) between hospitals, universities, and research institutions. Runs on GCP, AWS, or Azure — or locally via Docker Compose.

The name carries a double meaning. As an acronym it describes exactly what the system does — an **A**nonymization & **E**xchange **G**ateway for **I**maging **S**tudies. As a word it comes from Greek mythology: the *aegis* was the divine shield of Zeus and Athena, a symbol of protection. That duality captures the platform's core mission — shielding patient identity while enabling the free flow of medical imaging data for research and clinical care.

## What It Does

- **Browser-based anonymization** — DICOM tag-level de-identification happens in the browser before data leaves the hospital network. No software installation required. Works with any DICOM modality.
- **Internal enterprise ingress** — Studies originating inside the enterprise can be ingested via API, batch import CLI, or DIMSE C-STORE receiver and routed through the same processing pipeline.
- **Server-side defacing** — Automated facial feature removal from 3D head scans (DeepDefacer, mri_deface, mri_reface). Non-head modalities bypass defacing automatically.
- **Automated processing pipeline** — Classification, PHI detection, protocol compliance, QC, defacing, and BIDS conversion run automatically in dependency order after upload.
- **Cloud-neutral storage** — DICOM files stored in local filesystem, S3, or GCS with a built-in DICOMweb proxy.
- **Admin review** — dwvewer for QC, defacing review (before/after), and study management.
- **Controlled external sharing** — Token-authenticated export downloads, DICOMweb STOW-RS forwarding to external destinations, and configurable routing rules.
- **Modality-agnostic** — Supports MRI, CT, PET, PET/CT, ultrasound, X-ray, nuclear medicine, mammography, and all other DICOM-compliant imaging. MVP focuses on brain MRI/PET/CT.

## Architecture

See [AEGIS_Architecture.md](AEGIS_Architecture.md) for the full system design.

![Architecture Diagram](architecture.png)

## Repository Structure (Monorepo)

Currently a single monorepo. Planned to split into separate repos once interfaces stabilize.

```
aegis/
├── terraform/project/          # GCP project bootstrap (IAM, KMS, VPC-SC)
├── terraform/infra/            # GCP infrastructure (Cloud Run, Healthcare API)
├── terraform/aws/              # AWS infrastructure (ECS Fargate, S3, RDS, ALB)
├── api/                        # Go backend — upload orchestration, DICOMweb proxy
├── frontend/
│   ├── upload-portal/          # React — public-facing upload + anonymization UI
│   ├── admin-dashboard/        # React — internal QC, dwvewer, study management
│   ├── export-portal/          # React — public-facing export share download UI
│   └── landing/                # React — public landing page (aegisimaging.ai, Cloudflare Pages)
├── client/                     # TypeScript DICOM anonymization library (npm)
├── defacing/                   # Python defacing service (DeepDefacer, mri_deface)
├── phi-detection/              # Python burned-in PHI detection (Tesseract OCR)
├── qc-service/                 # Python QC automation service
├── bids-service/               # Python NIfTI/BIDS conversion service (dcm2niix)
├── classification-service/     # Python metadata classification service
├── protocol-service/           # Python MRI protocol compliance service
├── dimse-receiver/             # Python DIMSE adapter (pynetdicom C-STORE SCP + ingest trigger)
└── docs/                       # Shared research and documentation
```

## Tech Stack

- **Backend**: Go 1.24 on Cloud Run / ECS Fargate (`distroless` containers)
- **Database**: PostgreSQL 15 (Cloud SQL on GCP, RDS on AWS, Docker for local dev)
- **Frontend**: React 19 + TypeScript + Vite (4 apps)
- **DICOM Storage**: Cloud-neutral file storage (local, GCS, or S3) with built-in DICOMweb proxy
- **Processing**: 7 Python services (6 processing sidecars + 1 DIMSE receiver adapter)
- **AI/ML**: Pluggable — local backends (Tesseract, pydicom heuristics) or cloud AI (Google Cloud Vision, AWS Textract/Rekognition)
- **Defacing**: DeepDefacer (default), mri_deface (fallback), mri_reface (research)
- **Viewer**: dwvmbedded in admin dashboard)
- **Auth**: Multi-provider — GCP IAP, Azure AD Easy Auth, AWS ALB + Cognito
- **Email**: Standard SMTP (any provider). Dev: Mailpit
- **Infrastructure**: Terraform (GCP + AWS modules), Docker Compose for local dev
- **CI**: GitHub Actions (Go build+vet+test, Python syntax, TypeScript type check, Docker build)

## License

Copyright 2026 AEGIS Imaging LLC. Licensed under the [Apache License, Version 2.0](LICENSE).
