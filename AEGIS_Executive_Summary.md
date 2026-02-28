---
pdf_options:
  format: Letter
  margin: 20mm
  printBackground: true
css: |
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    font-size: 14px;
    line-height: 1.6;
    color: #24292e;
    overflow: visible !important;
  }
  h1 { font-size: 2em; margin: 0.67em 0; }
  h2 { font-size: 1.5em; margin: 1.2em 0 0.6em; border-bottom: 1px solid #e5e7eb; padding-bottom: 0.4em; }
  h3 { font-size: 1.2em; margin: 1em 0 0.4em; }
  p  { margin: 0.8em 0; }
  ul, ol { padding-left: 2em; margin: 0.8em 0; }
  li { margin: 0.3em 0; }
  table { border-collapse: collapse; width: 100%; margin: 1em 0; font-size: 13px; }
  th { background: #f6f8fa; font-weight: 600; text-align: left; padding: 8px 12px; border: 1px solid #d0d7de; }
  td { padding: 7px 12px; border: 1px solid #d0d7de; vertical-align: top; }
  code { background: #f6f8fa; padding: 2px 5px; border-radius: 4px; font-size: 85%; }
  pre  { background: #f6f8fa; padding: 16px; border-radius: 6px; overflow-x: auto; }
  hr   { border: none; border-top: 1px solid #e5e7eb; margin: 2em 0; }
  blockquote { border-left: 4px solid #d0d7de; margin: 0; padding: 0 1em; color: #57606a; }
  sup  { font-size: 75%; vertical-align: super; }
  em   { font-style: italic; }
  strong { font-weight: 600; }
  a { color: #0969da; }
---

<div style="text-align: center; padding: 40px 0 20px;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width: 200px; margin-bottom: 12px;" />
  <h1 style="font-size: 42px; margin: 0; color: #1a1a2e;">AEGIS</h1>
  <p style="font-size: 22px; color: #4a4a6a; margin: 8px 0 0;">Anonymization &amp; Exchange Gateway for Imaging Studies</p>
  <hr style="width: 60%; margin: 20px auto; border: 1px solid #e0e0e0;" />
  <p style="font-size: 16px; color: #6b7280; margin: 0.5em 0;">A multi-cloud platform for secure, HIPAA-compliant de-identification and sharing of medical imaging data — for research teams and radiology departments alike</p>
  <p style="font-size: 13px; color: #9ca3af; margin-top: 12px; font-style: italic;">In Greek mythology, the <em>aegis</em> was the divine shield of Zeus and Athena — a symbol of protection. The name captures our mission: shielding patient identity while enabling the free flow of imaging data for research and clinical care.</p>
  <p style="font-size: 14px; color: #4a4a6a; margin-top: 20px; margin-bottom: 2px;"><strong>Matthew L. Senjem, M.S.</strong></p>
  <p style="font-size: 13px; color: #6b7280; margin-top: 0; margin-bottom: 2px;">AEGIS Imaging LLC</p>
  <p style="font-size: 13px; color: #9ca3af; margin-top: 0;">February 28, 2026</p>
</div>

---

## The Problem

Hospitals and research institutions need to share medical imaging data — MRI, CT, PET, ultrasound, X-ray — for multi-site clinical trials, research collaborations, and second opinions. Doing this safely is harder than it should be.

**De-identification tools frequently fail.** A 2015 study in *European Radiology* tested 10 free DICOM de-identification tools against 50 required PHI data elements. With default settings, only 1 of 10 tools successfully removed all required elements; 4 tools achieved success rates of 26% or less.<sup><a href="#ref-1">[1]</a></sup>

**Metadata removal alone is not enough for brain imaging.** A 2019 *New England Journal of Medicine* correspondence from Mayo Clinic showed that automated face-recognition software correctly matched de-identified brain MRI participants to their photographs in **83% of cases** — using only the scan geometry, no metadata at all.<sup><a href="#ref-2">[2]</a></sup> This means head imaging requires an additional step: automated defacing (removal of facial surface features from the 3D volume).

**Burned-in PHI is invisible to tag-level tools.** Patient names, dates of birth, and accession numbers are routinely overlaid directly on image pixels — particularly in ultrasound, CT topograms, and fluoroscopy. Standard DICOM de-identification tools operate on metadata only and miss this entirely.<sup><a href="#ref-3">[3]</a></sup>

**Installing software at sending sites is a barrier.** Platforms like XNAT require a Java-based desktop client or server daemon installed and IT-approved at every participating hospital. MIRC CTP requires a standalone Java application. Each new site adds months of IT negotiation.

**Data sharing is now federally mandated — but tooling lags.** The NIH Data Management and Sharing Policy (NOT-OD-21-013, effective January 25, 2023) requires all NIH-funded investigators to share scientific data as a condition of their grant award, no later than the time of first publication.<sup><a href="#ref-4">[4]</a></sup> Institutions are now obligated to share data they have no reliable, low-friction way to de-identify and transmit.

**Result:** Multi-site studies are delayed by months. In a 2021 study of global randomized trials, contract execution alone averaged 7.9 months per US site.<sup><a href="#ref-5">[5]</a></sup> Software installation and IT approval add to this.

---

## Market Opportunity

The medical image exchange market is growing, driven by federal data sharing mandates and expanding multi-site clinical trial volume in oncology and neurology.

| Market | 2024 Estimate | 2034 Projection | CAGR |
|--------|--------------|-----------------|------|
| Medical Image Exchange Systems<sup><a href="#ref-6">[6]</a></sup> | $3.9B | $9.9B | 9.7% |

**Why now:**
- The NIH DMS Policy (2023) creates a compliance obligation for tens of thousands of active grants that did not previously require a data sharing plan
- A 2024 NCI-sponsored benchmark (MIDI-B, presented at MICCAI 2024) formally tested de-identification tools and found that burned-in pixel PHI and free-text fields remain the hardest unsolved problems — no existing tool handles all cases reliably<sup><a href="#ref-7">[7]</a></sup>
- Growing multi-site trial volume in oncology (PSMA PET, amyloid PET, whole-body MRI) and neurology (tau PET, fMRI) is driving demand for scalable data collection from distributed hospital networks
- Cloud-hosted DICOM storage (GCS, S3, Azure Blob) and managed databases (Cloud SQL, RDS, Azure Database) are now mature commodity infrastructure, reducing the barrier to building managed platforms

---

## The Solution

**AEGIS** is a multi-cloud platform that makes secure medical image sharing as easy as uploading a file — for any DICOM modality. It runs on GCP, AWS, or Azure — or locally via Docker Compose. It serves two complementary audiences from a single platform.

| Capability | What It Does |
|-----------|-------------|
| **No software to install** | Upload portal runs entirely in the web browser. Hospitals use their existing computers and internet connection — no desktop app, no IT approval for new software. |
| **Tag-level anonymization in the browser** | Patient data is stripped from DICOM tags before it leaves the hospital network, following the DICOM PS3.15 Annex E Basic Confidentiality Profile — the established international standard covering all 18 HIPAA Safe Harbor identifier categories.<sup><a href="#ref-8">[8]</a></sup> |
| **Automated defacing for head scans** | MRI, CT, and PET studies of the head and brain undergo server-side defacing (removal of facial surface geometry) before entering the research archive. Non-head studies bypass this step automatically. |
| **Burned-in PHI detection** | Automated OCR scans image pixels for overlaid text (patient names, dates, accession numbers) that tag-level tools miss. Studies with detected pixel PHI are flagged for review. |
| **Supports all DICOM modalities** | MRI, CT, PET, PET/CT, ultrasound, X-ray, mammography, nuclear medicine, and more — using the same upload and anonymization workflow. |
| **MRI protocol compliance** | Automated verification that acquisition parameters (TR, TE, flip angle, resolution) match site-specific templates per scanner manufacturer, model, and software version — catching the 0.19–64% non-compliance rates found across multi-site studies.<sup><a href="#ref-10">[10]</a></sup> |
| **Centralized audit trail** | Every upload, approval, rejection, and data export is logged with a timestamp and actor. Institutions can demonstrate HIPAA compliance from a single dashboard. |
| **Clinician review before release** | Administrators review anonymized images in a web-based DICOM viewer (Weasis DWV) before approving studies for sharing. Side-by-side before/after defacing review is built in. Nothing is shared automatically without human sign-off. |
| **Pixel redaction** | Automated detection and masking of burned-in PHI directly in DICOM pixel data — patient names, dates, and accession numbers overlaid on images are located by OCR and redacted at the pixel level. |
| **Neuroimaging analytics** | Post-BIDS analysis pipelines with 18 backends: brain (FreeSurfer, SynthSeg, BrainSuite, volBrain, Atlas ROI), spine (SCT, TotalSpineSeg, SPINEPS), whole-body (TotalSegmentator, nnU-Net, MedSAM2, MONAI Label), and specialized (PETSurfer, QSM, BASIL, FSL, ANTs, SPM, ITK-SNAP). Longitudinal analytics (TBM-SyN, FreeSurfer Long) for tracking brain volume changes over time. |
| **Spinal cord analysis** | Spinal Cord Toolbox (SCT) integration: automated cord segmentation, cross-sectional area (CSA) per vertebral level, compression metrics (aMCC, aSCOR), and DTI mapping for spinal cord studies. |
| **Comprehensive DICOM format support** | JPEG2000 lossless, JPEG-LS, Enhanced (multi-frame) DICOM, and Siemens Mosaic DICOM handled transparently across all processing services via decompression backends. |
| **Vendor private tag preservation** | Per-project option to retain vendor-specific private tags (diffusion gradients, CSA headers) during de-identification, with automated PHI scanning of preserved tag values. |

---

## Who It's For

AEGIS serves two audiences from a single de-identification and routing platform:

**Multi-site research teams** need to collect imaging data from hospitals they don't control, de-identify it to meet NIH and IRB requirements, and convert it to research-ready formats. AEGIS gives them a browser link they can send to any participating site — no software installation, no IT negotiation. The research pipeline (protocol compliance, automated QC, BIDS conversion, defacing) ensures data arrives analysis-ready.

**Hospital radiology departments** need to de-identify imaging data for internal uses — training AI models, building real-world evidence databases, sharing with referring physicians, and fulfilling data requests from payers and registries. AEGIS gives them a centralized de-identification gateway with a complete audit trail, burned-in PHI detection, and configurable routing rules — without replacing their PACS or requiring a separate enterprise integration project.

| | Research Teams | Radiology Departments |
|--|---------------|----------------------|
| **Entry point** | Browser upload link sent to external sites | DIMSE receive / internal batch import |
| **Key need** | NIH DMS compliance, multi-site consistency | HIPAA de-identification, audit trail |
| **Differentiator** | Zero-install at sending sites | No PACS/VNA integration required |
| **Pipeline features** | Defacing, protocol compliance, QC, BIDS | Burned-in PHI, routing rules, export shares |
| **Review workflow** | PI/coordinator approves before archive | Radiologist/admin approves before release |

The underlying platform — de-identification engine, routing rules, audit trail, Weasis DWV viewer, and admin dashboard — is shared. Research-specific features (defacing, BIDS, protocol templates) and enterprise features (DIMSE receive, HL7/FHIR notifications, tag standardization) extend it for each audience.

---

## How De-identification Works

AEGIS follows a two-phase approach that addresses the limitations documented in the literature:

<pre style="background: #f6f8fa; padding: 16px; border-radius: 6px; font-size: 13px; line-height: 1.8;">
Phase 1 — In the browser (before upload):
  ├── DICOM tags stripped per PS3.15 Annex E Basic Profile <a href="#ref-8">[8]</a>
  ├── 18 HIPAA Safe Harbor identifiers addressed <a href="#ref-9">[9]</a>
  ├── Reviewer sees before/after tag comparison before confirming
  └── Only de-identified data is transmitted — PHI never leaves the hospital

Phase 2 — On the server (after upload):
  ├── De-identification completeness check
  ├── OCR scan of image pixels for burned-in text <a href="#ref-3">[3]</a>
  ├── Head imaging → automated defacing <a href="#ref-2">[2]</a>
  └── Administrator review and approval before data is shared
</pre>

This two-phase design directly addresses the gaps identified in the Aryanto (2015) <a href="#ref-1">[1]</a> and Schwarz (2019) <a href="#ref-2">[2]</a> studies.

---

## Supported Modalities

| Category | Modalities |
|----------|-----------|
| **Cross-sectional** | MRI, CT, PET, PET/CT, PET/MR |
| **Projection** | X-Ray (CR/DR), Mammography (FFDM/DBT), Fluoroscopy |
| **Nuclear Medicine** | SPECT, Planar Scintigraphy |
| **Ultrasound** | Echocardiography, Abdominal, Vascular, OB/GYN |
| **Radiation Therapy** | RT Structure, RT Plan, RT Dose, RT Image |
| **Other DICOM** | Secondary Capture, Structured Reports, Key Images |

**Initial focus**: Brain MRI, PET, and CT — these exercise the full pipeline including defacing. All other modalities use the same upload and de-identification workflow.

---

## Technology

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Backend** | Go on Cloud Run (GCP), ECS Fargate (AWS), or Container Apps (Azure) | Compiled binary, minimal dependencies, fast cold starts |
| **Frontend** | React + TypeScript | Runs in any modern browser, no installation required |
| **Database** | PostgreSQL 15 (Cloud SQL / RDS) | Audit trail, project/user management, routing rules |
| **DICOM Storage** | Cloud-neutral (GCS, S3, Azure Blob, or local filesystem) | Abstracted behind a pluggable storage interface |
| **OCR / PHI Detection** | Tesseract OCR (local) / cloud AI (pluggable) | Detects burned-in text in image pixels |
| **Defacing** | DeepDefacer (default), mri_deface, mri_reface | Multiple backends with automatic fallback; see `docs/research/mri-defacing-tools-comparison.md` |
| **DICOM Viewer** | Weasis DWV (browser-based) | Lightweight, open-source; supports QIDO-RS/WADO-RS; built-in side-by-side defacing review |
| **Auth** | GCP IAP / AWS ALB+Cognito / Azure AD | Multi-provider auth middleware, auto-detection |
| **Processing Pipeline** | Microservices architecture — 9 Python microservices (Cloud Run / ECS Fargate / Container Apps) + DIMSE receiver (Compute Engine VM / EC2 / Azure VM) + MCP server (Cloud Run) | Classification, PHI detection + pixel redaction, protocol compliance, QC, defacing, BIDS conversion, synthetic MRI generation, neuroimaging analytics (18 backends incl. FreeSurfer, FSL, ANTs, SPM, SynthSeg, nnU-Net, TotalSegmentator, MONAI Label, PETSurfer, QSM, BASIL), spinal cord analysis (SCT) — auto-dispatched in 4-phase dependency order; DIMSE C-STORE SCP on dedicated VM (static IP, port 11112); MCP server exposes 96 read tools + 95 write tools; agent orchestrator with DICOM tag provenance and diagnostic toolchain |
| **Infrastructure** | Terraform (GCP + AWS modules), Docker Compose | Reproducible, version-controlled, multi-cloud; local dev stack starts everything with one command |

**Live deployments:** GCP — [admin.aegisimaging.ai](https://admin.aegisimaging.ai) (dashboard) · [api.aegisimaging.ai](https://api.aegisimaging.ai) (API) · AWS — [aws.admin.aegisimaging.ai](https://aws.admin.aegisimaging.ai) (dashboard) · [aws.api.aegisimaging.ai](https://aws.api.aegisimaging.ai) (API) · Azure — [azure.admin.aegisimaging.ai](https://azure.admin.aegisimaging.ai) (dashboard) · [azure.api.aegisimaging.ai](https://azure.api.aegisimaging.ai) (API).

**On the use of automated tools:** AEGIS uses automated tools to assist with — not replace — human review. Automated de-identification flags potential issues; a trained administrator reviews and approves every study before it is shared. Automated defacing quality is reviewed side-by-side against the original in the admin interface.

---

## Compliance Framework

AEGIS is designed around established regulatory and technical standards:

| Standard | Reference | What It Covers |
|----------|-----------|----------------|
| HIPAA Privacy Rule | 45 CFR § 164.514(b) | De-identification methods: Safe Harbor (18 identifiers) and Expert Determination |
| DICOM Confidentiality Profile | NEMA PS3.15 Annex E | Tag-level de-identification actions for all DICOM attributes |
| NIH Data Management Policy | NOT-OD-21-013 (eff. Jan 25, 2023) | Data sharing obligations for NIH-funded investigators |
| Cloud Security | BAA, VPC/VPC-SC, CMEK/KMS | Business Associate Agreement, encrypted storage, network controls — available on GCP, AWS, and Azure |

AEGIS does not create a new de-identification standard — it implements the ones that already exist.

---

## Competitive Landscape

| | AEGIS | ENCOG (Enlitic) | XNAT | Flywheel | MIRC CTP |
|--|-------|-----------------|------|----------|----------|
| **Hosting** | Multi-cloud (GCP, AWS, Azure) | On-premises / hybrid | Self-hosted | Commercial SaaS | On-premises |
| **Install at sending site** | None — browser only | PACS/VNA integration | Desktop Java client | CLI tool | Java application |
| **Burned-in PHI** | OCR detection | AI CV detection | None | Built-in | None |
| **Defacing** | Automated, reviewable | None | Manual or plugin | Built-in | None |
| **Modalities** | All DICOM | MR, CT, XR, US | Neuroimaging focus | All imaging | All imaging |
| **Open source** | Private (built on open-source stack) | No (commercial) | Yes | No | Yes |
| **Audit trail** | Centralized, per-study | Chain of custody | Per-instance | Built-in | Limited |

Existing platforms tend to serve either research (XNAT, Flywheel) or enterprise radiology (ENCOG/Enlitic, MIRC CTP) — but not both. AEGIS is designed for both audiences from a shared platform, and is the only option that runs on any major cloud provider without re-architecture.

Enlitic's ENCOG is the closest commercial analogue to AEGIS's de-identification pipeline.<sup><a href="#ref-12">[12]</a></sup> It uses AI-driven computer vision to detect burned-in text overlays and claims to protect over 4,000 DICOM fields. However, ENCOG requires PACS/VNA integration at each sending site, does not offer volumetric defacing for head imaging, and publicly documents only four modalities (MR, CT, XR, ultrasound). ENCOG is part of a broader commercial suite (Ensight) that includes data standardization (ENDEX, FDA 510(k) cleared) and migration tools — but lacks the research pipeline features (protocol compliance, QC, BIDS) that multi-site studies require.

XNAT and Flywheel serve research well but require software installation at sending sites and offer limited enterprise radiology integration. MIRC CTP is a robust open-source pipeline but requires on-premises Java infrastructure and has no burned-in PHI detection or defacing.

**Key differences from existing platforms:**

1. **No software installation at sending sites** — the full anonymization workflow runs in the browser. ENCOG, XNAT, and MIRC CTP all require software deployed at the sending institution.
2. **Two-phase de-identification** — both tag-level (browser) and pixel-level (server-side OCR) are addressed in a unified pipeline
3. **Defacing is automated and reviewable** — not optional, manual, or absent. Neither ENCOG nor MIRC CTP offer volumetric defacing.
4. **Not specialty-specific** — designed for all DICOM modalities from the start, unlike XNAT (neuroimaging focus) or ENCOG (4 listed modalities)
5. **Dual-market platform** — research features (protocol compliance, QC, BIDS) and enterprise features (DIMSE receive, routing rules, tag standardization) extend a shared de-identification core. Competitors serve one audience or the other.

---

## Development Velocity

AEGIS was built from a blank repository to full GCP production deployment in **7 days** (February 17–24, 2026), using AI-assisted development tooling. AWS production followed on day 8. Azure Container Apps deployment completed on day 9. The resulting platform is production-grade across all three major clouds: version-controlled infrastructure, automated CI/CD, 2,100+ tests, and all services deployed and monitored simultaneously.

A single merge to `develop` deploys to all three clouds simultaneously — GCP Cloud Build, AWS GitHub Actions, and Azure GitHub Actions (OIDC federated auth) trigger concurrently, updating all services across GCP, AWS, and Azure within minutes from a single shared codebase. Cross-cloud DICOM routing (STOW-RS and DIMSE C-STORE) is verified live across all cloud pairs.

| Metric | Value |
|--------|-------|
| Days from first commit to GCP production | **7** |
| Days from first commit to AWS production | **8** |
| Days from first commit to Azure production | **9** |
| Git commits | **1,300+** |
| API routes (Go) | **322+** |
| Automated tests (Go + Python + client) | **2,100+** |
| Database migrations | **77** |
| Cloud Run services deployed (GCP) | **15** |
| ECR repositories provisioned (AWS) | **13** |
| ECS Fargate services (AWS) | **10** |
| Azure Container Apps | **15** |
| Python processing services | **9** |
| Pipeline phases | **4** (Classification → PHI/Protocol/Deface → QC/BIDS → Analytics/SCT) |
| MCP AI agent tools (read + write) | **191** |

---

## Phased Roadmap

> **Production status:** The full platform is **live on three clouds.** GCP: Go API, admin dashboard, Weasis DWV viewer, 9 Python processing services (Cloud Run), DIMSE receiver (Compute Engine VM, static IP `35.232.172.221`, port 11112), and MCP server at `api.aegisimaging.ai` and `admin.aegisimaging.ai`. AWS: 10 ECS Fargate services, RDS PostgreSQL, S3, ALB + Cognito auth, DIMSE receiver (EC2 with Elastic IP, port 11112) at `aws.api.aegisimaging.ai` and `aws.admin.aegisimaging.ai`. Azure: 15 Container Apps + PostgreSQL Flexible Server + Azure Blob Storage, DIMSE receiver (Azure Linux VM, static IP `20.97.180.87`, port 11112) at `azure.api.aegisimaging.ai` and `azure.admin.aegisimaging.ai`. All clouds share one codebase; GCP Cloud Build and GitHub Actions deploy in parallel on every merge to `develop`. Cross-cloud DICOM routing (GCP→AWS→Azure) is verified live. **All three DIMSE C-STORE receivers are operational** — accepting inbound studies from PACS systems on TCP port 11112.

### ✓ Milestone 1 — Foundation + GCP Production (February 17–24, 2026)

Everything listed below was built and deployed to GCP production within 7 days of the first commit:

- Browser-based upload portal with DICOM tag anonymization (PS3.15 Basic Profile, 18 HIPAA identifiers), date shifting, pseudonymization, and before/after tag diff preview
- Go API with 322+ routes: DICOM ingest, routing engine, DICOMweb proxy, export shares, audit trail, webhook subscriptions, API keys, clinical trial access control
- Admin dashboard: study browser, RBAC (admin/viewer + capability-based guards), protocol templates, routing rules, institutions, 17+ management tabs
- Weasis DWV viewer with side-by-side before/after defacing review (yoked scroll synchronization)
- 9 Python processing services on Cloud Run: classification, PHI scan + pixel redaction, protocol compliance, QC, defacing, BIDS conversion, synthetic MRI generation, neuroimaging analytics (18 backends), spinal cord analysis (SCT)
- DIMSE C-STORE SCP on dedicated Compute Engine VM (port 11112) — durable retry queue, dead-letter, exponential backoff
- MCP server (96 read tools + 95 write tools) + agent orchestrator with DICOM tag provenance and diagnostic toolchain — AI-native platform operations
- Terraform IaC (GCP + AWS + Azure modules), GitHub Actions CI, Cloud Build CD (auto-deploy on push to `develop`)
- 2,100+ automated tests (1,072 Go tests, 892 Python pytest tests across 9 sidecars, 166 client library tests)
- Comprehensive DICOM format support: JPEG2000, JPEG-LS, Enhanced multi-frame, Siemens Mosaic
- Vendor private tag preservation with automated PHI scanning of retained tag values
- 8 phases of cross-cloud security hardening with CI enforcement

### ✓ Milestone 2 — AWS Deployment (February 25, 2026)

- ECS Fargate (10 services), RDS PostgreSQL, S3, ALB + Cognito auth — fully live at `aws.api.aegisimaging.ai`
- Terraform infrastructure provisioned; 13 ECR repositories; GitHub Actions CI/CD auto-deploys on every push to `develop`
- DIMSE receiver on EC2 with Elastic IP, SSM-driven rolling deploys
- **Cross-cloud DICOM routing verified live** — GCP→AWS STOW-RS tested end-to-end; bidirectional API key authentication; routing loop benign (deduplicated by unique constraint)
- Single shared codebase; one merge deploys to all three clouds simultaneously

### ✓ Milestone 3 — Azure Deployment (February 26, 2026)

- Azure Container Apps deployment — same Go API, Python sidecars, and React frontends as GCP and AWS
- Azure Database for PostgreSQL (Flexible Server), Azure Blob Storage (STORAGE_MODE=azure), Azure Container Registry
- GitHub Actions CI/CD with OIDC federated auth — auto-deploy on every push to `develop`
- DIMSE receiver on Azure Linux VM — same static-IP PACS integration pattern
- Azure Communication Services Email relay for SMTP notifications

### ✓ Milestone 4 — Advanced Processing & Security (February 27, 2026)

- Comprehensive DICOM format support: JPEG2000 lossless, JPEG-LS, Enhanced multi-frame, Siemens Mosaic across all 9 services
- Pixel redaction pipeline step: automated detection and masking of burned-in PHI in DICOM pixel data
- Vendor private tag preservation with automated PHI scanning of retained tag values
- Clinical trial access control: project membership roles, capability-based write guards, scoped study visibility
- 8 phases of cross-cloud security hardening with CI enforcement scripts

### ✓ Milestone 5 — Analytics Expansion & MIDI-B Compliance (February 28, 2026)

- Neuroimaging analytics service with 18 backends: brain (FreeSurfer, SynthSeg, BrainSuite, volBrain, Atlas ROI), spine (SCT, TotalSpineSeg, SPINEPS), whole-body (TotalSegmentator, nnU-Net, MedSAM2, MONAI Label), and specialized (PETSurfer, QSM, BASIL, FSL, ANTs, SPM, ITK-SNAP)
- Spinal Cord Toolbox (SCT) sidecar: cord segmentation, cross-sectional area (CSA), compression metrics (aMCC, aSCOR), DTI mapping
- Longitudinal analytics: TBM-SyN tensor-based morphometry and FreeSurfer longitudinal stream for brain volume change tracking
- Biomarker database: ROI volumetric results, QC ratings, demographics, analytics file management
- MIDI-B de-identification benchmark compliance (Tracks 1 & 2): date shifting, pseudonymization, LLM-based text scrubbing, mapping export
- Niivue NIfTI viewer embedded in admin dashboard for analytics output review
- 2,100+ automated tests (1,072 Go, 892 Python, 166 client), 322+ API routes, 191 MCP tools, 77 database migrations

### → Milestone 6 — Private Beta (Q1 2026)

- First enterprise pilot customers (research institutions + radiology departments)
- Invite-gated access with self-service request workflow
- Gathering feedback on de-identification workflows, PACS integration, and protocol compliance
- DIMSE C-STORE receivers operational on all three clouds (GCP, AWS, Azure) — accepting inbound studies from PACS systems on TCP port 11112
- DIMSE C-MOVE / C-FIND workflows for active PACS pull integration
- SLA monitoring, retention policies, and routing rules hardening for production workloads

### → Milestone 7 — Beta Hardening + Compliance Foundations (Q2 2026)

- Business Associate Agreement (BAA) template + countersigning workflow
- SOC 2 Type I audit initiation — point-in-time controls assessment
- DIMSE C-MOVE / C-FIND at scale — AEGIS actively queries and retrieves studies from PACS and VNA systems
- AWS Marketplace listing for enterprise procurement

### → Milestone 8 — SOC 2 Type II + Enterprise Integrations (Q3 2026)

- SOC 2 Type II certification — formal audit after 6-month observation period
- HL7 FHIR notifications — integrate with hospital EMR/RIS systems
- Cross-tenant federated sharing — peer AEGIS instances can exchange approved studies

### → Milestone 9 — Enterprise GA (Q4 2026)

- On-premises deployment option for institutions with strict data residency requirements
- PACS/VNA native query-retrieve — pull studies on demand rather than waiting for push
- Multi-tenant SaaS with per-organization data isolation for radiology groups

---

## Cost Estimate (Development / Proof of Concept)

| Resource | GCP Monthly | AWS Monthly | Azure Monthly |
|----------|------------|------------|--------------|
| Containers (Cloud Run / ECS Fargate / Container Apps) | ~$5–15 | ~$5–15 | ~$5–15 |
| PostgreSQL (Cloud SQL / RDS / Flexible Server) | ~$10 | ~$15 | ~$15 |
| Object Storage (GCS / S3 / Azure Blob) | ~$0.02/GB | ~$0.02/GB | ~$0.02/GB |
| Load Balancer | Included | ~$16 (ALB) | Included |
| **Total (development)** | **~$20–40/month** | **~$40–60/month** | **~$25–50/month** |

Production costs scale with data volume. A 1,000-session multi-site study (~500 GB) would cost approximately $50–100/month in storage on any cloud.

---

<div style="page-break-before: always;">

## References

<ol style="font-size: 13px; line-height: 1.7;">
<li id="ref-1">Aryanto KYE, Oudkerk M, van Ooijen PMA. "Free DICOM de-identification tools in clinical research: functioning and safety of patient privacy." <em>European Radiology.</em> 2015;25(12):3685–3695. DOI: 10.1007/s00330-015-3794-0</li>

<li id="ref-2">Schwarz CG, et al. "Identification of Anonymous MRI Research Participants with Face-Recognition Software." <em>New England Journal of Medicine.</em> 2019;381(17):1684–1686. DOI: 10.1056/NEJMc1908881</li>

<li id="ref-3">Vcelak P, et al. "Identification and classification of DICOM files with burned-in text content." <em>International Journal of Medical Informatics.</em> 2019;126:128–137. DOI: 10.1016/j.ijmedinf.2019.02.011</li>

<li id="ref-4">National Institutes of Health. "Final NIH Policy for Data Management and Sharing." NOT-OD-21-013. Effective January 25, 2023. grants.nih.gov/grants/guide/notice-files/NOT-OD-21-013.html</li>

<li id="ref-5">Lai J, et al. "Drivers of Start-Up Delays in Global Randomized Clinical Trials." <em>Therapeutic Innovation &amp; Regulatory Science.</em> 2021;55(1):212–227. DOI: 10.1007/s43441-020-00207-2</li>

<li id="ref-6">Fact.MR. <em>Medical Image Exchange System Market.</em> February 2024. globenewswire.com/news-release/2024/02/08/2825768</li>

<li id="ref-7">Pei L, Farahani K, et al. "Medical Image De-Identification Benchmark Challenge." arXiv:2507.23608 (preprint, under review). NCI CBIIT, MICCAI 2024.</li>

<li id="ref-8">National Electrical Manufacturers Association. <em>DICOM PS3.15: Security and System Management Profiles, Annex E.</em> dicom.nema.org/medical/dicom/current/output/chtml/part15/chapter_e.html</li>

<li id="ref-9">U.S. Department of Health and Human Services. "Guidance Regarding Methods for De-identification of PHI in Accordance with the HIPAA Privacy Rule." 45 CFR § 164.514(b). hhs.gov/hipaa/for-professionals/special-topics/de-identification/</li>

<li id="ref-10">Ravi H, et al. "mrQA: MR Quality Assurance framework for automated protocol compliance assessment across 20+ open neuroimaging datasets." <em>Neuroinformatics.</em> 2024;22:637–651. DOI: 10.1007/s12021-024-09679-3</li>

<li id="ref-11">Arani A, et al. "Alzheimer's Disease Neuroimaging Initiative 4 (ADNI4): MRI Protocol Update with Rationale for Changes." <em>Alzheimer's &amp; Dementia.</em> 2024;20(S2):e088739. DOI: 10.1002/alz.088739</li>

<li id="ref-12">Enlitic, Inc. "ENCOG — Healthcare Data Anonymization Tools." enlitic.com/encog/ (accessed February 2026). Product page describing AI-driven de-identification of DICOM metadata, burned-in pixel data, and private tags across MR, CT, XR, and ultrasound modalities.</li>
</ol>

</div>

---

<div style="text-align: center; padding: 40px 0;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width: 120px; margin-bottom: 16px;" />
  <h2 style="color: #1a1a2e;">AEGIS</h2>
  <p style="font-size: 14px; color: #4a4a6a; margin: 4px 0 12px;"><strong>AEGIS Imaging LLC</strong></p>
  <p style="font-size: 16px; color: #6b7280; max-width: 600px; margin: 0 auto;">
    Secure, browser-based medical image de-identification and sharing for any DICOM modality — serving both multi-site research teams and hospital radiology departments from a single multi-cloud platform, with automated defacing, pixel-level PHI detection, and administrator review before any data is released. Runs on GCP, AWS, or Azure.
  </p>
  <p style="font-size: 12px; color: #9ca3af; margin-top: 24px;">&copy; 2026 AEGIS Imaging LLC. All rights reserved.</p>
</div>
