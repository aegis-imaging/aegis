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

<div style="text-align: center; padding: 60px 0 30px;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width: 220px; margin-bottom: 20px;" />
  <h1 style="font-size: 42px; margin: 0; color: #1a1a2e;">AEGIS</h1>
  <p style="font-size: 22px; color: #4a4a6a; margin: 8px 0 0;">Anonymization &amp; Exchange Gateway for Imaging Studies</p>
  <hr style="width: 60%; margin: 30px auto; border: 1px solid #e0e0e0;" />
  <p style="font-size: 16px; color: #6b7280;">A cloud-hosted platform for secure, HIPAA-compliant sharing of medical imaging data across all DICOM modalities</p>
  <p style="font-size: 13px; color: #9ca3af; margin-top: 20px; font-style: italic;">In Greek mythology, the <em>aegis</em> was the divine shield of Zeus and Athena — a symbol of protection. The name captures our mission: shielding patient identity while enabling the free flow of imaging data for research.</p>
</div>

---

## The Problem

Hospitals and research institutions need to share medical imaging data — MRI, CT, PET, ultrasound, X-ray — for multi-site clinical trials, research collaborations, and second opinions. Doing this safely is harder than it should be.

**De-identification tools frequently fail.** A 2015 study in *European Radiology* tested 10 free DICOM de-identification tools against 50 required PHI data elements. With default settings, only 1 of 10 tools successfully removed all required elements; 4 tools achieved success rates of 26% or less.<sup>[1]</sup>

**Metadata removal alone is not enough for brain imaging.** A 2019 *New England Journal of Medicine* correspondence from Mayo Clinic showed that automated face-recognition software correctly matched de-identified brain MRI participants to their photographs in **83% of cases** — using only the scan geometry, no metadata at all.<sup>[2]</sup> This means head imaging requires an additional step: automated defacing (removal of facial surface features from the 3D volume).

**Burned-in PHI is invisible to tag-level tools.** Patient names, dates of birth, and accession numbers are routinely overlaid directly on image pixels — particularly in ultrasound, CT topograms, and fluoroscopy. Standard DICOM de-identification tools operate on metadata only and miss this entirely.<sup>[3]</sup>

**Installing software at sending sites is a barrier.** Platforms like XNAT require a Java-based desktop client or server daemon installed and IT-approved at every participating hospital. MIRC CTP requires a standalone Java application. Each new site adds months of IT negotiation.

**Data sharing is now federally mandated — but tooling lags.** The NIH Data Management and Sharing Policy (NOT-OD-21-013, effective January 25, 2023) requires all NIH-funded investigators to share scientific data as a condition of their grant award, no later than the time of first publication.<sup>[4]</sup> Institutions are now obligated to share data they have no reliable, low-friction way to de-identify and transmit.

**Result:** Multi-site studies are delayed by months. In a 2021 study of global randomized trials, contract execution alone averaged 7.9 months per US site.<sup>[5]</sup> Software installation and IT approval add to this.

---

## Market Opportunity

The medical image exchange market is growing, driven by federal data sharing mandates and expanding multi-site clinical trial volume in oncology and neurology.

| Market | 2024 Estimate | 2034 Projection | CAGR |
|--------|--------------|-----------------|------|
| Medical Image Exchange Systems<sup>[6]</sup> | $3.9B | $9.9B | 9.7% |

**Why now:**
- The NIH DMS Policy (2023) creates a compliance obligation for tens of thousands of active grants that did not previously require a data sharing plan
- A 2024 NCI-sponsored benchmark (MIDI-B, presented at MICCAI 2024) formally tested de-identification tools and found that burned-in pixel PHI and free-text fields remain the hardest unsolved problems — no existing tool handles all cases reliably<sup>[7]</sup>
- Growing multi-site trial volume in oncology (PSMA PET, amyloid PET, whole-body MRI) and neurology (tau PET, fMRI) is driving demand for scalable data collection from distributed hospital networks
- Cloud-hosted DICOM storage (GCP Healthcare API, AWS HealthImaging) is now mature commodity infrastructure, reducing the barrier to building managed platforms

---

## The Solution

**AEGIS** is a cloud-hosted platform that makes secure medical image sharing as easy as uploading a file — for any DICOM modality.

| Capability | What It Does |
|-----------|-------------|
| **No software to install** | Upload portal runs entirely in the web browser. Hospitals use their existing computers and internet connection — no desktop app, no IT approval for new software. |
| **Tag-level anonymization in the browser** | Patient data is stripped from DICOM tags before it leaves the hospital network, following the DICOM PS3.15 Annex E Basic Confidentiality Profile — the established international standard covering all 18 HIPAA Safe Harbor identifier categories.<sup>[8]</sup> |
| **Automated defacing for head scans** | MRI, CT, and PET studies of the head and brain undergo server-side defacing (removal of facial surface geometry) before entering the research archive. Non-head studies bypass this step automatically. |
| **Burned-in PHI detection** | Automated OCR scans image pixels for overlaid text (patient names, dates, accession numbers) that tag-level tools miss. Studies with detected pixel PHI are flagged for review. |
| **Supports all DICOM modalities** | MRI, CT, PET, PET/CT, ultrasound, X-ray, mammography, nuclear medicine, and more — using the same upload and anonymization workflow. |
| **Centralized audit trail** | Every upload, approval, rejection, and data export is logged with a timestamp and actor. Institutions can demonstrate HIPAA compliance from a single dashboard. |
| **Clinician review before release** | Administrators review anonymized images in a web-based DICOM viewer (OHIF) before approving studies for sharing. Nothing is shared automatically without human sign-off. |

---

## How De-identification Works

AEGIS follows a two-phase approach that addresses the limitations documented in the literature:

```
Phase 1 — In the browser (before upload):
  ├── DICOM tags stripped per PS3.15 Annex E Basic Profile [8]
  ├── 18 HIPAA Safe Harbor identifiers addressed [9]
  ├── Reviewer sees before/after tag comparison before confirming
  └── Only de-identified data is transmitted — PHI never leaves the hospital

Phase 2 — On the server (after upload):
  ├── De-identification completeness check
  ├── OCR scan of image pixels for burned-in text [3]
  ├── Head imaging → automated defacing [2]
  └── Administrator review and approval before data is shared
```

This two-phase design directly addresses the gaps identified in the Aryanto (2015) and Schwarz (2019) studies.

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
| **Backend** | Go on Google Cloud Run | Compiled binary, minimal dependencies, fast cold starts |
| **Frontend** | React + TypeScript | Runs in any modern browser, no installation required |
| **Database** | PostgreSQL 15 (Cloud SQL) | Audit trail, project/user management, routing rules |
| **DICOM Storage** | GCP Healthcare API | Managed DICOMweb storage, no servers to maintain |
| **Analytics** | BigQuery | DICOM metadata export, compliance reporting |
| **OCR / PHI Detection** | Google Cloud Document AI | Detects burned-in text in image pixels |
| **Defacing** | mri_deface (FreeSurfer) | Established tool, used in neuroimaging research |
| **DICOM Viewer** | OHIF Viewer (v3) | Open-source, browser-based, supports all modalities |
| **Infrastructure** | Terraform | Reproducible, version-controlled infrastructure |

**On the use of automated tools:** AEGIS uses automated tools to assist with — not replace — human review. Automated de-identification flags potential issues; a trained administrator reviews and approves every study before it is shared. Automated defacing quality is reviewed side-by-side against the original in the admin interface.

---

## Compliance Framework

AEGIS is designed around established regulatory and technical standards:

| Standard | Reference | What It Covers |
|----------|-----------|----------------|
| HIPAA Privacy Rule | 45 CFR § 164.514(b) | De-identification methods: Safe Harbor (18 identifiers) and Expert Determination |
| DICOM Confidentiality Profile | NEMA PS3.15 Annex E | Tag-level de-identification actions for all DICOM attributes |
| NIH Data Management Policy | NOT-OD-21-013 (eff. Jan 25, 2023) | Data sharing obligations for NIH-funded investigators |
| GCP Security | BAA, VPC, CMEK | Business Associate Agreement, encrypted storage, network controls |

AEGIS does not create a new de-identification standard — it implements the ones that already exist.

---

## Competitive Landscape

| | AEGIS | XNAT | Flywheel | MIRC CTP |
|--|-------|------|----------|----------|
| **Hosting** | Cloud-hosted (GCP) | Self-hosted | Commercial SaaS | On-premises |
| **Install at sending site** | None — browser only | Desktop Java client | CLI tool | Java application |
| **Modalities** | All DICOM | Neuroimaging focus | All imaging | All imaging |
| **Defacing** | Automated, reviewable | Manual or plugin | Built-in | None |
| **Open source** | Yes | Yes | No | Yes |
| **Audit trail** | Centralized, per-study | Per-instance | Built-in | Limited |

**Key differences from existing platforms:**

1. **No software installation at sending sites** — the full anonymization workflow runs in the browser
2. **Two-phase de-identification** — both tag-level and pixel-level (burned-in PHI) are addressed
3. **Defacing is automated and reviewable** — not optional or manual
4. **Not specialty-specific** — designed for all DICOM modalities from the start

---

## Phased Roadmap

### Phase 1 — Foundation (Complete)
- Cloud infrastructure (Terraform, GCP)
- Browser-based upload portal with DICOM tag anonymization and before/after preview
- Go API with DICOM ingest and PostgreSQL audit trail
- Admin dashboard with OHIF viewer for QC, approve/reject workflow

### Phase 2 — Defacing + Notifications (Complete)
- Automated mri_deface pipeline on Cloud Run for head imaging
- Side-by-side before/after defacing review in admin interface
- Email notifications for upload confirmation, approval, and rejection

### Phase 3 — Operations (Complete)
- Multi-project routing engine with configurable rules by modality, source, and anatomy
- Institution management with project-scoped roles
- Configurable anonymization profiles per project (which tags to retain for research)
- Audit log viewer with filtering
- Admin user management and access control

### Phase 4 — Advanced Validation (Next)
- Burned-in PHI detection using OCR on image pixels (addresses gap identified in [3, 7])
- Automated image quality assessment (motion artifact detection, coverage completeness)
- BIDS format conversion for neuroimaging research output
- Batch import tools for historical data migration

---

## Cost Estimate (Development / Proof of Concept)

| Resource | Monthly |
|----------|---------|
| Cloud Run (API + dashboard) | ~$5–15 (covered by free tier in dev) |
| Cloud SQL (PostgreSQL, small instance) | ~$10 |
| Healthcare API (DICOM store) | ~$0.50/GB stored |
| Cloud Storage (staging) | ~$0.02/GB |
| BigQuery | Free tier (1 TB queries/month) |
| Document AI (OCR for burned-in PHI) | Pay per page (minimal during dev) |
| **Total (development)** | **~$20–40/month** |

Production costs scale with data volume. A 1,000-session multi-site study (~500 GB) would cost approximately $50–100/month in storage.

---

<div style="page-break-before: always;">

## References

<ol style="font-size: 13px; line-height: 1.7;">
<li>Aryanto KYE, Oudkerk M, van Ooijen PMA. "Free DICOM de-identification tools in clinical research: functioning and safety of patient privacy." <em>European Radiology.</em> 2015;25(12):3685–3695. DOI: 10.1007/s00330-015-3794-0</li>

<li>Schwarz CG, et al. "Identification of Anonymous MRI Research Participants with Face-Recognition Software." <em>New England Journal of Medicine.</em> 2019;381(17):1684–1686. DOI: 10.1056/NEJMc1908881</li>

<li>Vcelak P, et al. "Identification and classification of DICOM files with burned-in text content." <em>International Journal of Medical Informatics.</em> 2019;126:128–137. DOI: 10.1016/j.ijmedinf.2019.02.011</li>

<li>National Institutes of Health. "Final NIH Policy for Data Management and Sharing." NOT-OD-21-013. Effective January 25, 2023. grants.nih.gov/grants/guide/notice-files/NOT-OD-21-013.html</li>

<li>Lai J, et al. "Drivers of Start-Up Delays in Global Randomized Clinical Trials." <em>Therapeutic Innovation &amp; Regulatory Science.</em> 2021;55(1):212–227. DOI: 10.1007/s43441-020-00207-2</li>

<li>Fact.MR. <em>Medical Image Exchange System Market.</em> February 2024. globenewswire.com/news-release/2024/02/08/2825768</li>

<li>Pei L, Farahani K, et al. "Medical Image De-Identification Benchmark Challenge." arXiv:2507.23608 (preprint, under review). NCI CBIIT, MICCAI 2024.</li>

<li>National Electrical Manufacturers Association. <em>DICOM PS3.15: Security and System Management Profiles, Annex E.</em> dicom.nema.org/medical/dicom/current/output/chtml/part15/chapter_e.html</li>

<li>U.S. Department of Health and Human Services. "Guidance Regarding Methods for De-identification of PHI in Accordance with the HIPAA Privacy Rule." 45 CFR § 164.514(b). hhs.gov/hipaa/for-professionals/special-topics/de-identification/</li>
</ol>

</div>

---

<div style="text-align: center; padding: 40px 0;">
  <img src="logo-small.png" alt="AEGIS Logo" style="width: 120px; margin-bottom: 16px;" />
  <h2 style="color: #1a1a2e;">AEGIS</h2>
  <p style="font-size: 16px; color: #6b7280; max-width: 600px; margin: 0 auto;">
    Secure, browser-based medical image sharing for any DICOM modality — built on established de-identification standards, with automated defacing, pixel-level PHI detection, and administrator review before any data is released.
  </p>
</div>
