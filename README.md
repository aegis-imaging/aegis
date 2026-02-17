# AEGIS

**Anonymized Exchange Gateway for Imaging Studies**

A GCP-hosted platform for secure, HIPAA-compliant sharing of brain MRI, PET, and CT imaging data between hospitals, universities, and research institutions.

## What It Does

- **Browser-based anonymization** — DICOM tag-level de-identification happens in the browser before data leaves the hospital network. No software installation required.
- **Server-side defacing** — Automated facial feature removal from 3D head scans after upload.
- **Secure cloud routing** — Encrypted transmission to a GCP Healthcare API DICOM store with DICOMweb endpoints.
- **Admin review** — OHIF-powered viewer for QC, defacing review, and study management.

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full system design.

![Architecture Diagram](architecture.png)

## Repositories

| Repo | Description |
|------|-------------|
| `aegis-terraform-prj` | GCP project bootstrap (IAM, KMS, VPC-SC) |
| `aegis-terraform-infra` | Infrastructure (Cloud Run, Healthcare API, Cloud Armor) |
| `aegis-api` | Go backend — upload orchestration, routing, DICOMweb proxy |
| `aegis-frontend` | React — Upload Portal + Admin Dashboard |
| `aegis-client` | TypeScript uploader library (npm package) |

## Tech Stack

- **Backend**: Go on Cloud Run (`distroless` containers)
- **Frontend**: React + TypeScript
- **DICOM**: GCP Healthcare API (DICOMweb), dcmjs, dicomParser
- **Defacing**: mri_deface, dcm2niix, pydicom (Python Cloud Run sidecar)
- **Viewer**: OHIF Viewer
- **Infrastructure**: Terraform, Cloud Build

## License

*TBD*
