---
pdf_options:
  format: Letter
  margin: 16mm
  printBackground: true
css: |
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    font-size: 14px;
    line-height: 1.6;
    color: #1a1a2e;
    overflow: visible !important;
  }
  h1 { font-size: 2.6em; margin: 0.4em 0 0.2em; color: #1a1a2e; }
  h2 { font-size: 1.8em; margin: 0.3em 0 0.6em; color: #1a1a2e; border: none; padding: 0; }
  h3 { font-size: 1.15em; margin: 0.8em 0 0.3em; color: #374151; }
  p  { margin: 0.6em 0; }
  ul, ol { padding-left: 1.6em; margin: 0.5em 0; }
  li { margin: 0.35em 0; }
  table { border-collapse: collapse; width: 100%; margin: 0.8em 0; font-size: 13px; }
  th { background: #f1f5f9; font-weight: 600; text-align: left; padding: 8px 12px; border: 1px solid #cbd5e1; color: #1e293b; }
  td { padding: 7px 12px; border: 1px solid #e2e8f0; vertical-align: top; }
  hr { border: none; border-top: 2px solid #e2e8f0; margin: 1.2em 0; }
  .slide { page-break-before: always; min-height: 90vh; display: flex; flex-direction: column; justify-content: center; padding: 12mm 0; }
  .slide--title { text-align: center; justify-content: center; }
  .label { display: inline-block; background: #ccfbf1; color: #0f766e; font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: 4px; margin-bottom: 12px; letter-spacing: 0.04em; text-transform: uppercase; }
  .label--amber { background: #fef3c7; color: #92400e; }
  .label--blue { background: #dbeafe; color: #1e40af; }
  .label--slate { background: #f1f5f9; color: #475569; }
  .metric-row { display: flex; gap: 20px; margin: 14px 0; }
  .metric { flex: 1; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 14px 18px; text-align: center; }
  .metric-value { font-size: 2em; font-weight: 700; color: #0d9488; display: block; }
  .metric-label { font-size: 12px; color: #64748b; display: block; margin-top: 2px; }
  .cite { font-size: 11px; color: #9ca3af; font-style: italic; }
  .done { color: #0d9488; font-weight: 600; }
  .active { color: #2563eb; font-weight: 600; }
  .planned { color: #6b7280; }
  strong { font-weight: 600; color: #1e293b; }
  .footer { font-size: 11px; color: #9ca3af; text-align: center; margin-top: auto; padding-top: 8mm; }
---

<div class="slide slide--title">

<div style="text-align:center; padding: 20px 0 10px;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width:160px; margin-bottom:10px;" />
  <div style="font-size:40px; font-weight:700; color:#1a1a2e; letter-spacing:-0.5px;">AEGIS</div>
  <div style="font-size:18px; color:#4a4a6a; margin:6px 0 20px;">Anonymization &amp; Exchange Gateway for Imaging Studies</div>
  <hr style="width:50%; margin:0 auto 20px;" />
  <div style="font-size:16px; color:#374151; max-width:560px; margin:0 auto 24px; line-height:1.7;">
    A multi-cloud platform for <strong>HIPAA-compliant de-identification and sharing</strong> of medical imaging data — for research teams and radiology departments alike.
  </div>
  <div style="display:flex; justify-content:center; gap:10px; flex-wrap:wrap; margin-bottom:24px;">
    <span class="label">HIPAA Compliant</span>
    <span class="label">3 Clouds Live</span>
    <span class="label">BAA Available</span>
    <span class="label">Zero Install at Sending Sites</span>
  </div>
</div>

<div class="footer">Matthew L. Senjem, M.S. &nbsp;·&nbsp; AEGIS Imaging LLC &nbsp;·&nbsp; March 4, 2026 &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label label--amber">The Problem</div>

## Sharing medical images safely is harder than it should be

**De-identification tools frequently fail.**
A 2015 study tested 10 free DICOM de-identification tools. Only 1 of 10 removed all required PHI with default settings. Four tools achieved success rates of 26% or less. <span class="cite">(Aryanto et al., European Radiology, 2015)</span>

**Metadata removal alone isn't enough for brain imaging.**
Face-recognition software matched de-identified brain MRI participants to photographs in **83% of cases** — using only scan geometry, zero metadata. <span class="cite">(Schwarz et al., NEJM, 2019)</span>

**Burned-in PHI is invisible to tag-level tools.**
Patient names, dates, and accession numbers are routinely overlaid on image pixels. Tag-level tools miss this entirely. <span class="cite">(Vcelak et al., Int'l J Medical Informatics, 2019)</span>

**Data sharing is now federally mandated — but tooling lags.**
The NIH Data Management and Sharing Policy (Jan 25, 2023) requires all NIH-funded investigators to share scientific data as a condition of their grant award. Institutions are obligated to share data they have no reliable, low-friction way to transmit. <span class="cite">(NIH NOT-OD-21-013)</span>

**Installing software at sending sites is a barrier.**
Platforms like XNAT and MIRC CTP require desktop clients or server daemons at every participating hospital — months of IT negotiation per site.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label label--blue">Market Opportunity</div>

## The infrastructure gap in medical imaging

<div class="metric-row">
  <div class="metric">
    <span class="metric-value">$3.9B</span>
    <span class="metric-label">Medical image exchange market (2024)</span>
  </div>
  <div class="metric">
    <span class="metric-value">$9.9B</span>
    <span class="metric-label">Projected market by 2034 (9.7% CAGR)</span>
  </div>
  <div class="metric">
    <span class="metric-value">~62%</span>
    <span class="metric-label">Imaging studies lacking compliant sharing pipelines</span>
  </div>
</div>

**Why now:**

- **NIH DMS Policy (2023)** creates a compliance obligation for tens of thousands of active grants — federally funded research *must* share imaging data
- **MIDI-B benchmark (NCI, MICCAI 2024)** confirmed burned-in pixel PHI and free-text fields remain unsolved; no existing tool handles all cases reliably
- **Growing multi-site trial volume** in oncology (PSMA PET, amyloid PET) and neurology (tau PET, fMRI) demands scalable collection from distributed hospital networks
- **Cloud DICOM infrastructure** (GCS, S3, Azure Blob) is now mature commodity, reducing the barrier to building managed platforms

**The result:** Multi-site studies delayed by months. Contract execution alone averaged 7.9 months per US site in a 2021 global randomized trial analysis. Software installation adds further delay. AEGIS eliminates both.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">The Solution</div>

## Two-phase de-identification — addressing every known gap

```
Phase 1 — In the browser (before upload):
  ├── DICOM tags stripped per PS3.15 Annex E Basic Profile
  ├── 18 HIPAA Safe Harbor identifiers addressed
  ├── Before-and-after tag diff preview — reviewer confirms before sending
  └── Only de-identified data is transmitted — PHI never leaves the hospital
        ↓
Phase 2 — On the server (after upload):
  ├── OCR scan of image pixels for burned-in text
  ├── Head imaging → automated defacing (facial surface removal)
  ├── Protocol compliance, QC, BIDS conversion, metadata classification
  └── Administrator review and approval — nothing shared without human sign-off
```

**Key principles:**

- No software to install at sending sites — any browser, any OS, zero IT approval
- All modalities: MRI, CT, PET, PET/CT, ultrasound, X-ray, mammography, RT, nuclear medicine
- Automated defacing is reviewable — side-by-side before/after in the admin dashboard
- Every action is logged — upload, approval, rejection, export — timestamped and actor-attributed

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">Processing Pipeline</div>

## 7 automated processing services — dispatched in dependency order

| Service | What It Does |
|---------|-------------|
| **Classification** | Reads DICOM headers to fill in modality and body part; re-evaluates routing rules after classification |
| **PHI Detection** | OCR scans image pixels for burned-in text (names, dates, accession numbers) that tag tools miss |
| **Protocol Compliance** | Verifies acquisition parameters (TR, TE, flip angle, resolution) against per-scanner templates |
| **Defacing** | Removes facial surface geometry from head MRI/CT/PET volumes; multiple backends with automatic fallback |
| **QC** | 5 automated checks: file integrity, slice consistency, SNR estimation, coverage completeness, missing slices |
| **BIDS Conversion** | Converts DICOM to NIfTI with BIDS-compliant structure and JSON sidecar metadata (dcm2niix) |
| **Synthetic MRI** | Generates synthetic brain MRI phantoms for pipeline testing and defacing demos |

**AI-native operations:** An MCP server exposes 191 typed tools (96 read + 95 write) to AI agents — Claude, Cursor, or any MCP-compatible agent can query studies, trigger pipeline steps, and triage stuck studies autonomously.

**Auto-pipeline:** After routing rules evaluate, services dispatch in the correct dependency order automatically. No manual clicks required.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">Audiences</div>

## One platform, two markets

| | **Research Teams** | **Radiology Departments** |
|--|-------------------|--------------------------|
| **Entry point** | Browser upload link sent to external sites | DIMSE C-STORE receive / batch import from PACS |
| **Key need** | NIH DMS compliance, multi-site consistency | HIPAA de-identification, complete audit trail |
| **Core differentiator** | Zero install at sending sites | No PACS/VNA integration or replacement |
| **Pipeline features** | Defacing, protocol compliance, QC, BIDS | Burned-in PHI detection, routing rules, export shares |
| **Review workflow** | PI/coordinator approves before archive | Radiologist/admin approves before release |

**Research teams** get a browser link they can send to any participating site — no IT negotiation per hospital. The research pipeline (protocol compliance, automated QC, BIDS conversion, defacing) ensures data arrives analysis-ready for NIH-funded multi-site studies.

**Radiology departments** get a centralized de-identification gateway with a complete audit trail, burned-in PHI detection, and configurable routing rules — without replacing their PACS or embarking on a separate enterprise integration project.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">Enterprise</div>

## Enterprise adoption path — no rip-and-replace required

AEGIS works alongside existing infrastructure. Studies arrive via DIMSE C-STORE push from your PACS, or batch-imported from network storage. The de-identification engine, routing rules, and audit trail sit between your PACS and any downstream consumer.

| Concern | AEGIS Answer |
|---------|-------------|
| **PACS integration** | No PACS/VNA replacement — receives via standard DIMSE C-STORE (port 11112) or DICOMweb STOW-RS |
| **Software at sending sites** | None — external sites use a browser link; no desktop app, no IT approval at participating hospitals |
| **Cloud preference** | GCP, AWS, or Azure in your existing cloud tenancy; or Docker Compose for evaluation |
| **Compliance readiness** | BAA available; HIPAA audit trail from day one; SOC 2 Type I audit in progress |
| **Time to first study** | Docker Compose evaluation in minutes; cloud pilot in days |

**Immediate internal use cases:**
- Multi-site clinical trial imaging data collection — send a browser link to each site
- AI/ML training data preparation — de-identify and QC at scale before model training
- Real-world evidence databases — build a HIPAA-compliant archive from your clinical store
- Cross-department and cross-institution imaging data sharing with full audit trail

**Pilot structure:** A project-isolated instance can be deployed in your cloud tenancy within days. You control who uploads and who accesses the admin dashboard. No data leaves your tenancy.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">Traction</div>

## Live today — three clouds, production-grade

<div class="metric-row">
  <div class="metric">
    <span class="metric-value">7</span>
    <span class="metric-label">days to GCP production</span>
  </div>
  <div class="metric">
    <span class="metric-value">3</span>
    <span class="metric-label">clouds live (GCP · AWS · Azure)</span>
  </div>
  <div class="metric">
    <span class="metric-value">2,100+</span>
    <span class="metric-label">automated tests</span>
  </div>
  <div class="metric">
    <span class="metric-value">191</span>
    <span class="metric-label">MCP AI agent tools</span>
  </div>
</div>

| What's Live | Details |
|-------------|---------|
| **GCP** | 14 Cloud Run services + DIMSE VM · `api.aegisimaging.ai` · `admin.aegisimaging.ai` |
| **AWS** | 14 ECS Fargate services + EC2 DIMSE · `aws.api.aegisimaging.ai` |
| **Azure** | 14 Container Apps + DIMSE VM · `azure.api.aegisimaging.ai` |
| **Cross-cloud routing** | GCP→AWS→Azure STOW-RS verified live; bidirectional API key auth |
| **CI/CD** | Single merge to `develop` auto-deploys all 15 services on all 3 clouds simultaneously |
| **API** | 322+ routes, 17-tab admin dashboard, full RBAC, audit trail, webhook subscriptions |
| **Private beta** | Invite-gated; research institutions and imaging centers welcome |

**Development velocity:** 1,300+ git commits, built entirely with AI-assisted tooling on a single-engineer team.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label label--slate">Competitive Landscape</div>

## Key differentiators from existing platforms

| | **AEGIS** | **ENCOG (Enlitic)** | **XNAT** | **Flywheel** | **MIRC CTP** |
|--|:---:|:---:|:---:|:---:|:---:|
| **Install at sending site** | None — browser | PACS/VNA integration | Desktop Java client | CLI tool | Java application |
| **Hosting** | GCP · AWS · Azure | On-prem / hybrid | Self-hosted | Commercial SaaS | On-premises |
| **Burned-in PHI detection** | ✓ OCR | ✓ AI CV | ✗ | ✓ | ✗ |
| **Automated defacing** | ✓ Reviewable | ✗ | Manual / plugin | ✓ | ✗ |
| **Research pipeline** | ✓ Protocol / QC / BIDS | ✗ | ✓ | ✓ | ✗ |
| **DIMSE C-STORE receive** | ✓ | ✓ | ✓ | ✓ | ✓ |
| **All DICOM modalities** | ✓ | 4 listed | Neuroimaging | ✓ | ✓ |
| **AI-native operations (MCP)** | ✓ 191 tools | ✗ | ✗ | ✗ | ✗ |

**Five key differences:**
1. **No software at sending sites** — full anonymization in the browser; ENCOG, XNAT, and MIRC CTP all require software at the sending institution
2. **Two-phase de-identification** — tag-level (browser) + pixel-level OCR (server) in a unified pipeline
3. **Automated, reviewable defacing** — not optional or absent; side-by-side before/after in the dashboard
4. **Dual-market platform** — research features (protocol compliance, QC, BIDS) + enterprise features (DIMSE, routing, audit trail) extend a shared core
5. **Multi-cloud without re-architecture** — same codebase, same API, same frontends on GCP, AWS, and Azure

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">Roadmap</div>

## From first commit to three-cloud production — and a clear path to enterprise

| Milestone | Date | Status |
|-----------|------|--------|
| **1 · Foundation + GCP Production** — Go API, 7 Python microservices, DIMSE SCP, MCP server, admin dashboard, Terraform IaC, CI/CD | Feb 17–24, 2026 | <span class="done">✓ Live</span> |
| **2 · Private Beta** — Invite-gated early adopters; BAA finalized; SOC 2 Type I audit initiated; DIMSE C-MOVE / C-FIND | Q1 2026 | <span class="active">● Active</span> |
| **3 · AWS Deployment** — ECS Fargate, RDS, S3, ALB + Cognito; cross-cloud GCP→AWS routing verified | Feb 25, 2026 | <span class="done">✓ Live</span> |
| **4 · Azure Deployment** — Container Apps, PostgreSQL Flex, Azure Blob; GitHub Actions OIDC CI/CD | Feb 26, 2026 | <span class="done">✓ Live</span> |
| **5 · Beta Hardening + Compliance** — Production email, BAA countersigning, SOC 2 Type I audit, DIMSE C-MOVE, AWS Marketplace listing | Q2 2026 | <span class="planned">→ Planned</span> |
| **6 · SOC 2 Type II + Enterprise Integrations** — HL7 FHIR notifications, DICOM conformance statement v2, cross-tenant federated sharing | Q3 2026 | <span class="planned">→ Planned</span> |
| **7 · Enterprise GA** — SOC 2 Type II report, on-premises Helm chart, multi-tenant SaaS, PACS/VNA native query-retrieve | Q4 2026 | <span class="planned">→ Planned</span> |

All four completed milestones were delivered by a single engineer using AI-assisted development tooling. Three clouds live in 9 days from first commit.

<div class="footer">AEGIS &nbsp;·&nbsp; aegisimaging.ai</div>

</div>

---

<div class="slide">

<div class="label">Discussion</div>

## Next steps — what we're exploring

**The opportunity we're presenting:**

AEGIS is production-ready across three major clouds and actively accepting pilot institutions. We are exploring internal partnerships and backing to accelerate the path from private beta to enterprise GA.

**Three ways to engage:**

| Path | Description |
|------|-------------|
| **Internal pilot deployment** | Deploy a project-isolated AEGIS instance in your cloud tenancy — evaluate the full pipeline with your own imaging data under your BAA. No data leaves your environment. |
| **Research collaboration** | Use AEGIS to collect and de-identify imaging data for an active multi-site study — NIH DMS compliant, protocol QC and BIDS export included. |
| **Backing / partnership** | Support continued development toward SOC 2 Type II, BAA finalization, DIMSE C-MOVE, and AWS Marketplace listing — in exchange for early-adopter positioning and co-development input on features. |

**What happens next (Q2 2026):**
- Production email configuration (share notifications, pipeline alerts)
- Business Associate Agreement countersigning workflow
- SOC 2 Type I audit initiation — engages auditor, starts 6-month Type II observation window
- DIMSE C-MOVE / C-FIND for active PACS pull
- AWS Marketplace listing for enterprise procurement

**Contact:** Matthew L. Senjem · ops@aegisimaging.ai · [aegisimaging.ai](https://aegisimaging.ai) · [Schedule a demo →](https://aegisimaging.ai)

<div class="footer">AEGIS Imaging LLC &nbsp;·&nbsp; © 2026 &nbsp;·&nbsp; All rights reserved</div>

</div>
