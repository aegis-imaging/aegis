# Enlitic ENCOG — Competitive Analysis

Last updated: 2026-02-18

## Company Overview

**Enlitic, Inc.** is a Silicon Valley-based healthcare AI company founded in 2014. Publicly traded on the Australian Securities Exchange (ASX: ENL) since December 2023.

- **Headquarters**: San Francisco, CA
- **CEO**: Michael Sistenich
- **Employees**: ~50–100 (estimated)
- **Total funding**: $55M over 6 rounds from 11 investors (Amplify Partners, Marubeni, Capitol Health, Thorney, Regal Funds Management)
- **ASX IPO (Dec 2023)**: Raised $21M
- **Revenue (2024)**: $1.20M (up 115% from $557K in 2023)
- **Net losses (2024)**: -$15.24M
- **Stock price (Jan 2026)**: ~$0.01 AUD; market cap ~$20M AUD (down ~39% YoY)

Previously named one of MIT Technology Review's 50 Smartest Companies (twice).

## Product: ENCOG

**ENCOG** (stylized ENCOG™) is an AI-powered DICOM de-identification tool, part of Enlitic's **Ensight 2.0** platform framework. The broader platform includes:

| Module | Function |
|--------|----------|
| **ENDEX** | DICOM data standardization (study/series description normalization). FDA 510(k) cleared + CE marked. |
| **ENCOG** | DICOM de-identification/anonymization (metadata + pixel data + private tags) |
| **ENABLE** | Search and cohort development (launched 2025) |
| **Migratek** | AI-enabled DICOM data migration (acquired via Laitek, October 2024, $5M) |
| **Curie** | Underlying AI platform hosting all modules |

### De-identification Approach

ENCOG operates on three layers:

1. **DICOM metadata** — removes/redacts PHI from DICOM tag fields. Customers control redaction rules by attribute or value representation, including private tags. Supports hashed identifiers and consistent date shifting for longitudinal data.
2. **Burned-in pixel data** — uses "content-aware" Computer Vision algorithms to detect and remove text overlays. Claims to distinguish clinically significant text (laterality markers, positioning, projection) from PHI text by analyzing "high-contrast strokes and consistent text widths." Claims "100% detection of overlaid text."
3. **Private tags** — handles vendor-specific private DICOM tags that may contain PHI.

PHI can be "blacked out, altered, deleted, or shifted" depending on the use case.

### Supported Modalities

Explicitly listed: **MR, CT, XR (X-Ray), Ultrasound** (4 modalities).

PET, nuclear medicine, mammography, and radiation therapy are **not mentioned** in any public documentation.

### Deployment Model

ENCOG integrates into existing **PACS/VNA workflows** — implying on-premise or hybrid deployment within the hospital network. The product requires infrastructure integration at the sending site. Technical deployment details are gated behind sales process.

### Defacing Capabilities

**None documented.** ENCOG's pixel-level capability focuses exclusively on 2D burned-in text/overlay removal. There is no mention of 3D volumetric defacing (facial feature removal from MRI/CT), integration with defacing tools (mri_deface, mri_reface, pydeface), or NIfTI conversion.

### Audit Trail / Compliance

- Claims HIPAA and GDPR compliance
- "Audits activities and maintains a chain of custody"
- Organization-owned decrypt keys for re-identification control
- No public detail on audit log format, queryability, or event-level granularity

### Re-identification

Supports reversible de-identification via organization-owned decrypt keys. Re-identification is only possible at the originating organization.

### Pricing

No public pricing. Enterprise subscription model delivering "reliable Annual Recurring Revenue" (per investor materials). Target market: large healthcare systems and private radiology reading groups.

## Key Partnerships

| Partner | Details |
|---------|---------|
| **GE HealthCare** | MoU for cloud transitions and Genesis Cloud Product Suite integration (Feb 2025) |
| **Philips Healthcare** | Three-year agreement for ENDEX implementation |
| **Bayer Healthcare** | Three-year distribution agreement; ENDEX integrated into Bayer dose management; initial 50 sites, potential 200 |
| **Select Healthcare Solutions** | Strategic partnership, US cancer center developer/operator |
| **MLP Care** | Multi-faceted partnership with Turkey's largest private healthcare provider |
| **Marubeni Corporation** | Japanese market distribution |
| **INFINITT** | Partnership for radiology department efficiencies |
| **Department of Defense** | Partnership (details not public) |

No named hospital customers for ENCOG specifically are publicly disclosed. Company states "several US customer agreements moving through the proof-of-concept phase."

## Academic Validation

**No peer-reviewed publications found specifically about ENCOG.** The ENDEX/Curie platform has FDA 510(k) clearance and CE marking, but this is for the standardization product, not ENCOG. This contrasts with XNAT and MIRC CTP, which have extensive academic publication histories.

## Competitive Comparison with AEGIS

| Dimension | ENCOG | AEGIS |
|-----------|-------|-------|
| **De-identification** | Server-side AI (PACS integration) | Client-side browser (zero install) + server-side pipeline |
| **Burned-in PHI** | AI CV detection (claims 100%) | Server-side Tesseract OCR |
| **Defacing** | Not supported | Dedicated service (mri_deface/mri_reface) |
| **Deployment** | On-premise/hybrid (PACS/VNA) | Cloud-hosted (GCP) |
| **Install at sending site** | Required (PACS/VNA integration) | None (browser-based) |
| **Modalities** | MR, CT, XR, US (4 listed) | All DICOM modalities |
| **Protocol compliance** | Not offered | Per-scanner template matching |
| **QC automation** | Not offered | 5 automated quality checks |
| **BIDS conversion** | Not offered | dcm2niix-based conversion |
| **Audit trail** | Chain of custody (limited detail) | Per-study event log with queryable dashboard |
| **Open source** | No (proprietary) | Yes (monorepo) |
| **Academic validation** | None published | Research-backed citations |
| **Re-identification** | Org-owned decrypt keys | Not supported (one-way de-identification) |
| **Data monetization** | Explicit value proposition | Not a focus (research sharing) |

### ENCOG Strengths vs. AEGIS
- AI-driven burned-in text detection with clinical text preservation (laterality markers, etc.)
- Bundled with data standardization (ENDEX) and migration (Migratek)
- Major enterprise partnerships (GE, Philips, Bayer)
- Reversible de-identification for internal use cases

### ENCOG Weaknesses vs. AEGIS
- Requires PACS/VNA integration at every sending site (deployment friction)
- No defacing capability (critical gap for neuroimaging research)
- Limited documented modality support (4 vs. all DICOM)
- No academic/peer-reviewed validation of de-identification effectiveness
- No protocol compliance, QC, or BIDS conversion capabilities
- Proprietary/closed source
- Early revenue stage ($1.2M) with significant losses ($15.2M)
- Stock at $0.01 AUD signals financial pressure

## Sources

1. Enlitic. "ENCOG — Healthcare Data Anonymization Tools." enlitic.com/encog/ (accessed 2026-02-18). *Company product page.*
2. Enlitic. "Enlitic at ViVE 2025." enlitic.com/press/enlitic-at-vive-2025/ (accessed 2026-02-18). *Press release.*
3. Enlitic. "Ensight v2.1 DICOM Conformance Statement." enlitic.com/wp-content/uploads/PROD-0443-Ensight-v2.1-DICOM-Conformance-Statement.pdf (accessed 2026-02-18). *Technical document.*
4. Enlitic. "Enlitic and GE HealthCare Partner." enlitic.com/press/enlitic-mou-ge-healthcare/ (accessed 2026-02-18). *Press release.*
5. Enlitic. "Enlitic Commences Trading on the ASX." enlitic.com/press/enlitic-commences-trading-on-the-asx/ (accessed 2026-02-18). *Press release.*
6. Stock Analysis. "ENL.AX — Enlitic Inc." stockanalysis.com/quote/asx/ENL/ (accessed 2026-02-18). *Financial data.*
7. Enlitic. "Deidentifying and Anonymizing Healthcare Data." enlitic.com/blogs/deidentifying-and-anonymizing-healthcare-data/ (accessed 2026-02-18). *Company blog.*
8. Enlitic. "Enlitic Solutions." enlitic.com/solutions/ (accessed 2026-02-18). *Company product page.*
9. Radiology Business. "Radiology data sharing vendor Enlitic completes $5M acquisition of imaging IT pioneer." radiologybusiness.com (accessed 2026-02-18). *Industry news.*
10. Applied Radiology. "Enlitic Curie Platform Nets FDA Clearance, CE Marking." appliedradiology.com (accessed 2026-02-18). *Industry news.*
11. Enlitic. "Standardize, Anonymize, Monetize Imaging Data." enlitic.com/blogs/standardize-anonymize-monetize-imaging-data/ (accessed 2026-02-18). *Company blog.*
