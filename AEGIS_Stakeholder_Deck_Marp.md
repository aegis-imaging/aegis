---
marp: true
theme: default
paginate: false
backgroundColor: #ffffff
style: |
  section {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    font-size: 18px;
    color: #1a1a2e;
    padding: 40px 52px;
  }
  h1 { font-size: 2em; color: #1a1a2e; margin-bottom: 0.2em; }
  h2 { font-size: 1.5em; color: #1a1a2e; margin-bottom: 0.5em; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.3em; }
  h3 { font-size: 1.1em; color: #374151; }
  table { width: 100%; border-collapse: collapse; font-size: 14px; margin: 0.6em 0; }
  th { background: #f1f5f9; font-weight: 600; padding: 8px 12px; border: 1px solid #cbd5e1; color: #1e293b; text-align: left; }
  td { padding: 7px 12px; border: 1px solid #e2e8f0; vertical-align: top; }
  ul { padding-left: 1.4em; margin: 0.4em 0; }
  li { margin: 0.3em 0; }
  strong { color: #1e293b; }
  code { background: #f1f5f9; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
  pre { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; padding: 14px 18px; font-size: 13px; }
  .label { display: inline-block; background: #ccfbf1; color: #0f766e; font-size: 12px; font-weight: 700; padding: 3px 10px; border-radius: 4px; margin-bottom: 10px; text-transform: uppercase; letter-spacing: 0.05em; }
  .footer { position: absolute; bottom: 20px; left: 0; right: 0; text-align: center; font-size: 11px; color: #9ca3af; }
  section.title { text-align: center; justify-content: center; }
  section.title h1 { font-size: 2.6em; }
---

<!-- _class: title -->

# AEGIS

**Anonymization & Exchange Gateway for Imaging Studies**

A multi-cloud platform for **HIPAA-compliant de-identification and sharing** of medical imaging data — for research teams and radiology departments alike.

`HIPAA Compliant` · `3 Clouds Live` · `BAA Available` · `Zero Install at Sending Sites`

---

Matthew L. Senjem, M.S. · AEGIS Imaging LLC · March 4, 2026 · aegisimaging.ai

---

<div class="label">The Problem</div>

## Sharing medical images safely is harder than it should be

**De-identification tools frequently fail.**
A 2015 study tested 10 free DICOM tools. Only 1 of 10 removed all required PHI with default settings. Four tools achieved success rates of 26% or less. *(Aryanto et al., European Radiology, 2015)*

**Metadata removal alone isn't enough for brain imaging.**
Face-recognition software matched de-identified MRI participants to photos in **83% of cases** — using only scan geometry, zero metadata. *(Schwarz et al., NEJM, 2019)*

**Burned-in PHI is invisible to tag-level tools.**
Patient names, dates, and accession numbers are routinely overlaid on image pixels. Tag tools miss this entirely. *(Vcelak et al., Int'l J Medical Informatics, 2019)*

**Data sharing is now federally mandated — but tooling lags.**
NIH DMS Policy (Jan 2023) requires all NIH-funded investigators to share scientific data. Institutions must share data they have no reliable way to transmit. *(NIH NOT-OD-21-013)*

---

<div class="label" style="background:#dbeafe;color:#1e40af">Market Opportunity</div>

## The infrastructure gap in medical imaging

| Metric | Value |
|--------|-------|
| Medical image exchange market (2024) | **$3.9B** |
| Projected market by 2034 (9.7% CAGR) | **$9.9B** |
| Imaging studies lacking compliant sharing pipelines | **~62%** |

**Why now:**

- **NIH DMS Policy (2023)** creates a compliance obligation for tens of thousands of active grants
- **MIDI-B benchmark (NCI, MICCAI 2024)** confirmed burned-in pixel PHI remains unsolved
- **Growing multi-site trial volume** in oncology (PSMA PET, amyloid PET) and neurology (tau PET, fMRI)
- **Cloud DICOM infrastructure** (GCS, S3, Azure Blob) is now mature commodity

Multi-site studies delayed by months — contract execution alone averaged 7.9 months per US site. Software installation adds further delay. AEGIS eliminates both.

---

<div class="label">The Solution</div>

## Two-phase de-identification — addressing every known gap

**Phase 1 — In the browser (before upload):**
- DICOM tags stripped per PS3.15 Annex E Basic Profile
- 18 HIPAA Safe Harbor identifiers addressed
- Before-and-after tag diff preview — reviewer confirms before sending
- Only de-identified data is transmitted — PHI never leaves the hospital

**Phase 2 — On the server (after upload):**
- OCR scan of image pixels for burned-in text
- Head imaging → automated defacing (facial surface removal)
- Protocol compliance, QC, BIDS conversion, metadata classification
- Administrator review and approval — nothing shared without human sign-off

**Key principles:** No software to install at sending sites · All modalities (MRI, CT, PET, ultrasound, X-ray, RT) · Every action timestamped and actor-attributed

---

<div class="label">Processing Pipeline</div>

## 7 automated processing services

| Service | What It Does |
|---------|-------------|
| **Classification** | Reads DICOM headers to fill in modality and body part; re-evaluates routing rules |
| **PHI Detection** | OCR scans image pixels for burned-in text (names, dates, accession numbers) |
| **Protocol Compliance** | Verifies acquisition parameters (TR, TE, flip angle) against per-scanner templates |
| **Defacing** | Removes facial surface geometry from head MRI/CT/PET; reviewable before/after |
| **QC** | 5 automated checks: file integrity, slice consistency, SNR, coverage, missing slices |
| **BIDS Conversion** | Converts DICOM → NIfTI with BIDS-compliant structure and JSON sidecar (dcm2niix) |
| **Analytics** | 18 neuroimaging tools: FreeSurfer, SynthSeg, TotalSegmentator, nnU-Net, and more |

**AI-native:** MCP server with 191 typed tools — AI agents can query studies, trigger pipeline steps, and triage issues autonomously. **Auto-pipeline:** services dispatch in dependency order automatically.

---

<div class="label">Audiences</div>

## One platform, two markets

| | **Research Teams** | **Radiology / Enterprise** |
|--|-------------------|-----------------------------|
| **Entry point** | Browser upload link sent to external sites | DIMSE C-STORE receive / batch import from PACS |
| **Key need** | NIH DMS compliance, multi-site consistency | HIPAA de-identification, complete audit trail |
| **Core differentiator** | Zero install at sending sites | No PACS/VNA replacement needed |
| **Pipeline features** | Defacing, protocol compliance, QC, BIDS | Burned-in PHI detection, routing rules, export shares |
| **Review workflow** | PI/coordinator approves before archive | Admin approves before release |

**Research teams** send a browser link to any participating site — no IT negotiation per hospital.

**Radiology / Enterprise** get a centralized de-identification gateway with complete audit trail, burned-in PHI detection, and configurable routing — without replacing their PACS.

---

<div class="label">Enterprise</div>

## Enterprise adoption path — no rip-and-replace required

AEGIS sits alongside existing PACS infrastructure. Studies arrive via DIMSE C-STORE push or batch import from network storage.

| Concern | AEGIS Answer |
|---------|-------------|
| **PACS integration** | No replacement — receives via DIMSE C-STORE (port 11112) or DICOMweb STOW-RS |
| **Software at sending sites** | None — external sites use a browser link; no IT approval required |
| **Cloud preference** | GCP, AWS, or Azure in your existing tenancy; or Docker Compose for evaluation |
| **Compliance readiness** | BAA available; HIPAA audit trail from day one; SOC 2 Type I in progress |
| **Time to first study** | Docker Compose evaluation in minutes; cloud pilot in days |

**Immediate internal use cases:** Multi-site clinical trial data collection · AI/ML training data preparation · Real-world evidence databases · Cross-department imaging data sharing with full audit trail

---

<div class="label">Traction</div>

## Live today — three clouds, production-grade

| Metric | Value |
|--------|-------|
| Days from first commit to GCP production | **7** |
| Clouds live simultaneously | **3 (GCP · AWS · Azure)** |
| Automated tests | **2,100+** |
| MCP AI agent tools | **191** |
| API routes | **322+** |
| Git commits | **1,300+** |

| Cloud | Infrastructure | URL |
|-------|---------------|-----|
| **GCP** | 14 Cloud Run services + DIMSE VM | api.aegisimaging.ai |
| **AWS** | 14 ECS Fargate services + EC2 DIMSE | aws.api.aegisimaging.ai |
| **Azure** | 14 Container Apps + DIMSE VM | azure.api.aegisimaging.ai |

Single merge to `develop` auto-deploys all 15 services on all 3 clouds simultaneously. Built by a single engineer using AI-assisted development tooling.

---

<div class="label" style="background:#f1f5f9;color:#475569">Competitive Landscape</div>

## Key differentiators

| | **AEGIS** | **ENCOG** | **XNAT** | **Flywheel** | **MIRC CTP** |
|--|:---:|:---:|:---:|:---:|:---:|
| Install at sending site | **None** | PACS/VNA | Desktop Java | CLI tool | Java app |
| Burned-in PHI detection | ✓ OCR | ✓ AI CV | ✗ | ✓ | ✗ |
| Automated defacing | ✓ Reviewable | ✗ | Manual | ✓ | ✗ |
| Research pipeline (QC/BIDS) | ✓ | ✗ | ✓ | ✓ | ✗ |
| Multi-cloud | ✓ GCP/AWS/Azure | On-prem | Self-hosted | SaaS | On-prem |
| AI-native MCP tools | ✓ 191 | ✗ | ✗ | ✗ | ✗ |

**Five key differences:** No software at sending sites · Two-phase de-identification (tag + pixel) · Automated reviewable defacing · Dual-market platform · Multi-cloud without re-architecture

---

<div class="label">Roadmap</div>

## From first commit to enterprise

| Milestone | Date | Status |
|-----------|------|--------|
| **Foundation + GCP Production** — Go API, 9 Python sidecars, DIMSE SCP, MCP server, Terraform | Feb 17–24 | ✓ Live |
| **AWS Deployment** — ECS Fargate, RDS, S3; cross-cloud routing verified | Feb 25 | ✓ Live |
| **Azure Deployment** — Container Apps, PostgreSQL Flex, Azure Blob | Feb 26 | ✓ Live |
| **Advanced Processing** — 18 neuroimaging tools, pixel redaction, MIDI-B compliance | Feb 27–28 | ✓ Complete |
| **Private Beta** — Invite-gated; BAA; SOC 2 Type I audit; DIMSE C-MOVE | Q1 2026 | ● Active |
| **Beta Hardening + Compliance** — Email, BAA countersigning, AWS Marketplace | Q2 2026 | → Planned |
| **SOC 2 Type II + Enterprise** — HL7 FHIR, DICOM conformance statement v2 | Q3 2026 | → Planned |
| **Enterprise GA** — SOC 2 Type II report, on-premises Helm chart, multi-tenant SaaS | Q4 2026 | → Planned |

---

<div class="label">Discussion</div>

## Next steps — what we're exploring

**Three ways to engage:**

| Path | Description |
|------|-------------|
| **Internal pilot** | Deploy a project-isolated AEGIS instance in your cloud tenancy — evaluate with your own imaging data under your BAA. No data leaves your environment. |
| **Research collaboration** | Use AEGIS to collect and de-identify imaging data for an active multi-site study — NIH DMS compliant, protocol QC and BIDS export included. |
| **Backing / partnership** | Support development toward SOC 2 Type II, BAA finalization, DIMSE C-MOVE, and AWS Marketplace — in exchange for early-adopter positioning and co-development input. |

**What's next (Q2 2026):** Production email · BAA countersigning workflow · SOC 2 Type I audit initiation · DIMSE C-MOVE / C-FIND · AWS Marketplace listing

---

**Matthew L. Senjem** · ops@aegisimaging.ai · aegisimaging.ai
