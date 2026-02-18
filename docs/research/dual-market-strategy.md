# AEGIS Dual-Market Strategy: Research + Enterprise Radiology

Last updated: 2026-02-18

## Strategic Rationale

AEGIS was originally positioned as a research data sharing platform for multi-site clinical trials. The competitive analysis of Enlitic's ENCOG (see `docs/research/encog-competitive-analysis.md`) revealed that the enterprise radiology de-identification market is adjacent and served by the same core capabilities.

### Shared Core

The following capabilities serve both audiences equally:

- **De-identification engine** — DICOM PS3.15 Annex E Basic Profile, tag-level + pixel-level
- **Burned-in PHI detection** — OCR on image pixels for overlaid text
- **Routing rules engine** — configurable per project, modality, body part, source
- **Audit trail** — per-study, per-action, with actor and timestamp
- **OHIF Viewer** — browser-based DICOM review
- **Admin dashboard** — study management, approve/reject, filtering
- **Institution management** — multi-site, role-based
- **Zero-install upload** — browser-based, no IT approval at sending sites

### Research-Specific Extensions

Features that primarily serve multi-site research teams:

| Feature | Research Need |
|---------|--------------|
| Automated defacing | Face reconstruction risk in neuroimaging (Schwarz 2019) |
| Protocol compliance | Multi-site acquisition consistency (ADNI, HCP, ABCD) |
| Automated QC | Reject non-conforming data before it enters the archive |
| BIDS conversion | Standard format for neuroimaging analysis pipelines |
| Anonymization profiles | Per-project tag retention for research metadata |

### Enterprise-Specific Extensions (Phase 5)

Features that primarily serve hospital radiology departments:

| Feature | Enterprise Need |
|---------|----------------|
| DIMSE receive | Accept studies pushed from PACS/VNA (standard radiology workflow) |
| Tag standardization | Normalize study/series descriptions across scanners |
| HL7 FHIR notifications | Notify EMR/RIS when studies are processed |
| Reversible de-identification | Re-identification for internal use cases (AI training, follow-up) |
| PACS query-retrieve | Pull studies on demand from hospital archives |
| Multi-tenant SaaS | Per-organization isolation for radiology groups |

### Why Both Markets from One Platform

1. **The core is the same** — de-identification, routing, audit, and review are needed by both audiences. Building two separate products would duplicate 80% of the work.

2. **Cross-selling** — a research team at a hospital that already uses AEGIS for enterprise de-identification can add the research pipeline for a multi-site trial with zero additional deployment. A hospital that adopted AEGIS for a research study can expand it to cover enterprise de-identification.

3. **Competitive positioning** — no existing platform serves both:
   - XNAT, Flywheel → research only
   - ENCOG, MIRC CTP → enterprise only
   - AEGIS → both, from a shared core

4. **Zero-install differentiator applies to both** — enterprise radiology departments also prefer not to install new PACS integrations. Browser-based de-identification is as valuable to an enterprise radiologist as it is to a research coordinator.

### Risks

- **Diluted messaging** — must be careful not to confuse the pitch. The "Who It's For" section addresses this by making both audiences explicit with clear use cases.
- **Feature creep** — enterprise features (DIMSE, HL7, tag standardization) are Phase 5, after the research pipeline is solid.
- **Enterprise sales cycles** — selling to hospital IT is slower and more political than selling to research PIs. The browser-based architecture partly mitigates this (less IT involvement).

## Competitive Context

See also:
- `docs/research/encog-competitive-analysis.md` — detailed analysis of Enlitic's ENCOG
- `docs/research/medical-imaging-deidentification.md` — foundational citations for de-identification gaps
- `docs/research/mri-protocol-compliance.md` — protocol compliance research and consortia protocols
