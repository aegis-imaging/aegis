---
pdf_options:
  format: Letter
  margin: 20mm
  printBackground: true
stylesheet: https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.2.0/github-markdown.min.css
body_class: markdown-body
---

<div style="text-align: center; padding: 60px 0 30px;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width: 220px; margin-bottom: 20px;" />
  <h1 style="font-size: 42px; margin: 0; color: #1a1a2e;">AEGIS</h1>
  <p style="font-size: 22px; color: #4a4a6a; margin: 8px 0 0;">Anonymization &amp; Exchange Gateway for Imaging Studies</p>
  <hr style="width: 60%; margin: 30px auto; border: 1px solid #e0e0e0;" />
  <p style="font-size: 16px; color: #6b7280;">A GCP-native platform for secure, HIPAA-compliant sharing of medical imaging data across all DICOM modalities</p>
</div>

---

## Naming Quick Check

- First mention: **Anonymization & Exchange Gateway for Imaging Studies**
- After first mention: **AEGIS**

---

## The Problem

Hospitals and research institutions need to share medical imaging data — MRI, CT, PET, ultrasound, X-ray, mammography, nuclear medicine, and more — for multi-site clinical trials, research collaborations, and second opinions. Today this requires:

- **Installing site software** — platforms like XNAT require a Java-based desktop client or server daemon installed and IT-approved at every participating hospital. MIRC CTP requires a standalone Java application. Even simpler tools need command-line setup by hospital technical staff.
- **Manual de-identification** prone to human error, leaving PHI exposed. A 2024 NCI benchmark challenge (MIDI-B) confirmed that inconsistencies in DICOM metadata and burned-in PHI remain unsolved problems even for specialized tools.
- **No automated defacing** — facial features in 3D head scans (MRI, CT) enable re-identification via facial reconstruction software, yet most platforms offer no defacing pipeline.
- **Self-hosted infrastructure burden** — XNAT and Orthanc require dedicated servers, database maintenance, and ongoing IT support at the receiving institution.
- **Fragmented compliance** — each site manages its own HIPAA controls, audit trails, and BAAs, with no centralized visibility.
- **Modality limitations** — most platforms were built for a single specialty (XNAT for neuroimaging, commercial tools for cardiology) and don't generalize cleanly to other DICOM data.

**Result**: Data sharing is slow, risky, and expensive. Multi-site studies are delayed by months waiting for IT approvals and software installations at each new site.

---

## Market Opportunity

Medical imaging data sharing sits at the intersection of two fast-growing markets:

| Market | 2024 Size | Projected | CAGR |
|--------|-----------|-----------|------|
| Medical Image Exchange Systems | $4.3B | $8.7B (2034) | 8.1% |
| Enterprise Imaging IT | — | — | 12.2% (2025–2030) |
| Medical Digital Imaging Systems (total) | $36.1B | $38.3B (2025) | — |

**Why now:**
- NIH, NSF, and FDA are all mandating data sharing for publicly funded research — but tooling lags far behind policy.
- A 2024 NCI/MICCAI challenge (MIDI-B) formally benchmarked de-identification tools against HIPAA Safe Harbor + DICOM Confidentiality Profiles — confirming that no existing open-source tool handles all cases reliably.
- Cloud-native healthcare platforms (GCP Healthcare API, AWS HealthImaging) have matured to the point where managed DICOM storage is now commodity infrastructure — reducing build risk.
- Growing multi-site clinical trial volume (particularly in oncology and neurology) is driving demand for scalable, zero-friction data collection from distributed hospital networks.

---

## The Solution

**AEGIS** is a cloud-hosted platform that makes secure medical image sharing as easy as uploading a file — for **any DICOM modality**.

| Capability | How It Works |
|-----------|-------------|
| **Zero-install anonymization** | Browser-based DICOM tag de-identification. No software to install at hospital sites. Works with any DICOM data. |
| **Modality-agnostic** | Supports all DICOM modalities: MRI, CT, PET, PET/CT, US, X-ray, mammography, NM, SPECT, RT, and more. |
| **Automated defacing** | Server-side facial feature removal from head scans. Non-head studies bypass defacing automatically. |
| **HIPAA by design** | Client-side PHI stripping before data leaves the hospital network. VPC-SC, CMEK, audit trails. |
| **Managed DICOM storage** | GCP Healthcare API — DICOMweb endpoints, no servers to maintain. |
| **AI-powered QC** | Vertex AI detects burned-in PHI, assesses image quality, flags issues automatically. |
| **Admin review** | OHIF web viewer for QC, defacing review, and routing — all in the browser. |

---

## Architecture

<div style="text-align: center;">
  <img src="architecture.png" alt="AEGIS Architecture Diagram" style="max-width: 100%; border: 1px solid #e5e7eb; border-radius: 8px;" />
</div>

---

## How It Works

```
1.  Uploader opens AEGIS in their browser (no install)
2.  Drags DICOM files into the upload portal (any modality)
3.  Browser parses and strips all PHI tags (PS3.15 Annex E)
4.  Uploader reviews before/after anonymization preview
5.  Confirms → encrypted upload to GCP (de-identified only)
6.  Server validates de-identification completeness
7.  Head scans → auto-defacing; others → clean store
8.  Admin reviews in OHIF viewer, approves or rejects
9.  Approved study routed to destination institution
```

**Key**: Patient-identifying data **never leaves the hospital network**. The de-identification engine follows [**DICOM PS3.15 Annex E**](https://dicom.nema.org/medical/dicom/current/output/chtml/part15/chapter_e.html) — the official industry standard that defines exactly which data elements to remove, zero out, or replace to strip all 18 HIPAA Safe Harbor identifier categories from medical images. The process is **modality-agnostic** — it works identically whether the study is a brain MRI, a chest X-ray, or a cardiac ultrasound.

---

## Supported Modalities

| Category | Modalities |
|----------|-----------|
| **Cross-sectional** | MRI, CT, PET, PET/CT, PET/MR |
| **Projection** | X-Ray (CR/DR), Mammography (FFDM/DBT), Fluoroscopy |
| **Nuclear Medicine** | SPECT, Planar Scintigraphy |
| **Ultrasound** | Echocardiography, Abdominal, Vascular, OB/GYN |
| **Radiation Therapy** | RT Structure, RT Plan, RT Dose, RT Image |
| **Other DICOM** | Secondary Capture, Structured Reports, Key Images, Presentation States |

**MVP focus**: Brain MRI, PET, and CT — these exercise the full pipeline including automated defacing. All other modalities use the same upload and de-identification flow.

---

## Technology Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| **Backend** | Go on Cloud Run | ~10 MB container, 100ms cold starts, minimal CVE surface |
| **Frontend** | React + TypeScript | Modern, fast, PWA-capable |
| **Database** | Cloud SQL (PostgreSQL 15) | Relational data: users, projects, audit trail |
| **DICOM Storage** | GCP Healthcare API | Managed DICOMweb, built-in de-id validation |
| **Analytics** | BigQuery | DICOM metadata export, compliance reporting |
| **AI/ML** | Vertex AI | Burned-in PHI detection, image QC automation |
| **Defacing** | Python (mri_deface) | Proven facial feature removal, isolated sidecar |
| **Viewer** | OHIF Viewer | MIT-licensed, DICOMweb-native, all modalities |
| **Infrastructure** | Terraform | Reproducible, auditable IaC |
| **Email** | PSC→on-prem SMTP (internal) + SendGrid (external) | Dual-path, no PHI in notifications |

---

## Competitive Landscape

| | AEGIS | XNAT | Flywheel | MIRC CTP |
|--|-------|------|----------|----------|
| **Hosting** | GCP managed | Self-hosted | Commercial SaaS | On-prem |
| **Client install** | None (browser) | Desktop app | CLI tool | Java app |
| **Modalities** | All DICOM | Neuroimaging | All imaging | All imaging |
| **Defacing** | Automated | Manual/plugin | Built-in | None |
| **Open source** | Yes | Yes | No | Yes |
| **DICOM store** | Healthcare API | PostgreSQL + FS | MongoDB + S3 | Filesystem |

**Differentiators**:
1. Zero-install browser anonymization for any DICOM modality
2. Modality-agnostic — not locked to a single specialty
3. GCP-native with Healthcare API — managed DICOMweb, built-in de-id, BigQuery analytics
4. Automated defacing pipeline for head imaging
5. Vertex AI for burned-in PHI detection and image QC
6. Minimal attack surface (Go + distroless)

---

## Phased Roadmap

### Phase 1 — MVP (Brain MRI/PET/CT)
- GCP project + infrastructure (Terraform)
- Upload Portal with browser-based DICOM anonymization (all modalities)
- Go API with signed upload and Healthcare API ingest
- Admin Dashboard with OHIF viewer for QC
- Cloud SQL for app data, BigQuery for analytics

### Phase 2 — Defacing + Notifications
- Automated mri_deface pipeline on Cloud Run (head imaging)
- Before/after comparison in admin dashboard
- Email notifications: SendGrid (external), SMTP/PSC (internal)

### Phase 3 — Operations + Multi-Modality Expansion
- Multi-project routing engine for all modalities
- Institution onboarding and RBAC
- Configurable anonymization profiles per modality/project
- Audit reporting and compliance dashboards

### Phase 4 — AI/ML
- Vertex AI burned-in PHI detection (all modalities)
- Automated image quality assessment
- Smart routing with modality/anatomy classification
- BIDS format conversion for research output

---

## Cost Estimate (Dev/POC)

| Resource | Monthly Cost |
|----------|-------------|
| Cloud Run (API + dashboard) | ~$5-15 (free tier covers dev) |
| Cloud SQL (db-f1-micro) | ~$10 |
| Healthcare API (DICOM store) | ~$0.50/GB stored |
| Cloud Storage (staging) | ~$0.02/GB |
| BigQuery | Free tier (1 TB queries/month) |
| Vertex AI | Pay per prediction (minimal during dev) |
| SendGrid | Free tier (100 emails/day) |
| **Total (dev)** | **~$20-40/month** |

Production costs scale with data volume. A typical multi-site research study with 1,000 imaging sessions (~500 GB) would cost approximately $50-100/month in storage.

---

<div style="text-align: center; padding: 40px 0;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width: 120px; margin-bottom: 16px;" />
  <h2 style="color: #1a1a2e;">Ready to Build</h2>
  <p style="font-size: 16px; color: #6b7280; max-width: 600px; margin: 0 auto;">
    AEGIS enables secure, zero-install medical image sharing across all DICOM modalities — with automated anonymization, intelligent defacing, and AI-powered quality control — built on GCP with HIPAA compliance from day one.
  </p>
</div>
