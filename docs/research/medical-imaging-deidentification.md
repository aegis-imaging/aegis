# Medical Imaging De-identification — Research References

Compiled for use in the AEGIS Executive Summary, architecture documentation, and compliance materials.
All sources verified. Notes on source type and citeability included.

---

## 1. Face / Identity Reconstruction from Brain MRI

**Claim:** Removing DICOM metadata tags alone is insufficient for anonymization — facial features in structural MRI scans allow re-identification via facial reconstruction software.

**Primary citation:**

> Schwarz CG, Kremers WK, Therneau TM, Sharp RR, Gunter JL, Vemuri P, Arani A, Spychalla AJ, Kantarci K, Knopman DS, Petersen RC, Jack CR Jr. "Identification of Anonymous MRI Research Participants with Face-Recognition Software." *New England Journal of Medicine*. 2019 Oct 24;381(17):1684–1686.
> DOI: [10.1056/NEJMc1908881](https://www.nejm.org/doi/full/10.1056/NEJMc1908881)
> PMID: 31644852. PMC: PMC7091256.
> Institution: Mayo Clinic.

**Key findings:**
- Using 3D face reconstruction from cranial MRI and Microsoft Azure Face API, researchers correctly matched de-identified MRI participants to their photographs in **83% of cases** (70 of 84 volunteers)
- The correct scan appeared in the **top-5 ranked matches for 95%** of participants
- Only the tag-anonymized MRI was used — no metadata, no names

**Citeability:** Excellent. Peer-reviewed, published in NEJM, widely cited.

---

## 2. De-identification Tool Failure Rates (DICOM Metadata)

**Claim:** Free DICOM de-identification tools routinely fail to remove all PHI elements, especially with default settings.

**Primary citation:**

> Aryanto KYE, Oudkerk M, van Ooijen PMA. "Free DICOM de-identification tools in clinical research: functioning and safety of patient privacy." *European Radiology*. 2015 Dec;25(12):3685–3695.
> DOI: [10.1007/s00330-015-3794-0](https://link.springer.com/article/10.1007/s00330-015-3794-0)
> PMID: 26037716. PMC: PMC4636522.

**Key findings:**
- 10 free DICOM de-identification tools tested against 50 required PHI elements
- With **default settings, only 1 of 10 tools** removed all required elements
- **8 of 10 tools** performed below 65% effectiveness with defaults
- **4 tools achieved success rates of 26% or less** with default settings
- Conclusion (quoted): *"free DICOM toolkits should be used with extreme care to prevent the risk of disclosing PHI, especially when using the default configuration"*

**Citeability:** Strong. Peer-reviewed, quantified failure rates, European Radiology.

---

## 3. Burned-In PHI in DICOM Pixel Data

**Claim:** PHI is often embedded directly in image pixels (patient name, DOB, accession number overlaid on the image), not just in metadata tags, and is missed by tag-level de-identification.

**Primary citation:**

> Vcelak P, Kryl M, Kratochvil M, Kleckova J. "Identification and classification of DICOM files with burned-in text content." *International Journal of Medical Informatics*. 2019 Jun;126:128–137.
> DOI: [10.1016/j.ijmedinf.2019.02.011](https://www.sciencedirect.com/science/article/abs/pii/S1386505619302023)
> PMID: 31029254.

**Supporting citation (scale):**

> Wen MH, Kalpathy-Cramer J, et al. "A Method for Efficient De-identification of DICOM Metadata and Burned-in Pixel Text." *Journal of Imaging Informatics in Medicine*. 2024.
> DOI: [10.1007/s10278-024-01098-7](https://link.springer.com/article/10.1007/s10278-024-01098-7)
> PMID: 38587767. PMC: PMC11522224.

**Key findings:**
- Burned-in PHI (name, DOB, accession number, etc.) is a widespread, modality-dependent problem — most common in ultrasound, CT topograms, and fluoroscopy
- Tag-level de-identification tools do not address burned-in pixel PHI
- The 2024 paper validated a removal method on **415,182 images across ten modalities**
- HHS HIPAA guidance specifically acknowledges burned-in PHI as a de-identification risk

**Citeability:** Good. No single published prevalence rate, but problem is well-documented and quantified at scale.

---

## 4. NIH Data Management and Sharing Policy (2023)

**Claim:** NIH mandates data sharing for all funded research, but tooling to do it securely lags behind policy.

**Official sources:**

- NIH Notice NOT-OD-21-013, "Final NIH Policy for Data Management and Sharing"
  URL: https://grants.nih.gov/grants/guide/notice-files/NOT-OD-21-013.html
  Issued: October 29, 2020. **Effective: January 25, 2023.**
- NIH Notice NOT-OD-23-053 (effective date reminder)
  URL: https://grants.nih.gov/grants/guide/notice-files/NOT-OD-23-053.html
- Policy overview: https://grants.nih.gov/policy-and-compliance/policy-topics/sharing-policies/dms/policy-overview

**Key requirements:**
- Applies to all NIH-funded or NIH-conducted research generating scientific data (grants, contracts, intramural projects) with submission dates on or after January 25, 2023
- All investigators must submit a **Data Management and Sharing Plan (DMSP)**
- Data must be shared "as soon as possible, and no later than the time of an associated publication, or the end of performance period, whichever comes first"
- The approved DMSP becomes a **term and condition of the grant award**

**Citeability:** Definitive. Official U.S. government primary source.

---

## 5. MIDI-B Challenge (NCI Medical Image De-identification Benchmark)

**Claim:** A 2024 NCI-sponsored benchmark challenge confirmed that de-identification of DICOM data — particularly burned-in pixel PHI and free-text fields — remains an unsolved problem even for specialized tools.

**Official NCI page:**
https://www.cancer.gov/about-nci/organization/cbiit/news-events/news/2024/participate-nci-medical-image-de-identification-benchmark-challenge-miccai-2024

**TCIA dataset:**
https://www.cancerimagingarchive.net/collection/MIDI-B-Test-MIDI-B-Validation/

**Preprint paper (not yet peer-reviewed):**
> Pei L, Farahani K, et al. "Medical Image De-Identification Benchmark Challenge." arXiv:2507.23608.
> URL: https://arxiv.org/html/2507.23608v1

**Key findings:**
- Held as satellite workshop at MICCAI 2024 (Marrakesh, Morocco), October 24, 2024
- 10 finalist teams; lead organizers: Linmin Pei and Keyvan Farahani (NCI)
- Series-level accuracy: 99.53% ± 0.60% across teams, but **pixel-level burned-in PHI and free-text fields** were the hardest failure modes — only 4 of 10 teams handled pixel PHI well
- Some teams inadvertently produced DICOM-noncompliant files during de-identification

**Citeability:** Good for the challenge itself (cite the NCI official page). The arXiv paper is a preprint — note accordingly.

---

## 6. HIPAA De-identification Methods

**Claim:** HIPAA provides two legally recognized methods for de-identification of protected health information.

**Regulation:** 45 CFR § 164.514 — "Other requirements relating to uses and disclosures of protected health information"

- Official eCFR: https://www.ecfr.gov/current/title-45/subtitle-A/subchapter-C/part-164/subpart-E/section-164.514
- Cornell LII: https://www.law.cornell.edu/cfr/text/45/164.514
- HHS guidance: https://www.hhs.gov/hipaa/for-professionals/special-topics/de-identification/index.html

**Key provisions:**
- **45 CFR § 164.514(b)(1) — Expert Determination:** Statistical/scientific expert certifies that risk of re-identification is "very small"
- **45 CFR § 164.514(b)(2) — Safe Harbor:** Removal of 18 specific categories of identifiers AND no actual knowledge that remaining information can identify an individual
- The 18 Safe Harbor identifiers include names, geographic subdivisions smaller than state, dates (except year) for persons ≥ 90, phone/fax numbers, email addresses, SSNs, MRN, account numbers, certificate/license numbers, device identifiers, URLs, IP addresses, biometric identifiers, full-face photographs, and any other unique identifying number or code

**Citeability:** Definitive. Primary federal regulation.

---

## 7. DICOM Confidentiality Profile Standard

**Claim:** AEGIS implements the DICOM standard for tag-level de-identification.

**Standard:** NEMA. *DICOM PS3.15: Security and System Management Profiles.* Annex E: "Attribute Confidentiality Profiles (Normative)."

- Current version (living standard): https://dicom.nema.org/medical/dicom/current/output/chtml/part15/chapter_e.html
- Standards browser: https://www.dicomstandard.org/standards/view/security-and-system-management-profiles
- Published by: National Electrical Manufacturers Association (NEMA); maintained by DICOM Standards Committee

**Key provisions:**
- Table E.1-1 defines action codes for each DICOM attribute: D (replace with dummy), Z (zero/replace), X (remove), U (replace UID), K (keep), C (clean)
- The Basic Application Level Confidentiality Profile maps these actions to the 18 HIPAA Safe Harbor categories
- The standard is modality-agnostic — applies to all DICOM data regardless of imaging type

**Citeability:** Definitive. Primary standard from the standards body.

---

## 8. Market Size — Medical Image Exchange / Management

**Claim:** The medical image exchange market is growing significantly, driven by mandated data sharing policies and multi-site clinical trial growth.

**Primary report:**

> Fact.MR. "Medical Image Exchange System Market." February 2024.
> Press release: https://www.globenewswire.com/news-release/2024/02/08/2825768/0/en/Medical-Image-Exchange-System-Market-Expected-to-Reach-US-9-89-Billion-by-2034-Fact-MR-Report.html

**Figures:**
- 2024 market size: **US $3.91 billion**
- 2034 projected: **US $9.89 billion**
- CAGR: **9.7%** (2024–2034)

**Corroborating figure:**

> Verified Market Reports. "Medical Image Management Market." 2024.
- 2024 market size: ~USD 4.5 billion
- 2033 projected: ~USD 10.2 billion
- CAGR: 9.8%

**Citeability:** Acceptable for pitch deck use. Commercial market research firm (Fact.MR), not peer-reviewed. Cite the GlobeNewswire press release as a secondary source for the Fact.MR figure. Do not present as peer-reviewed research.

---

## 9. Clinical Trial Site Startup Delays

**Claim:** Multi-site studies are delayed by months waiting for IT approvals and software installations at each new site.

**Peer-reviewed source (general trial startup):**

> Lai J, Forney L, Brinton DL, Simpson KN. "Drivers of Start-Up Delays in Global Randomized Clinical Trials." *Therapeutic Innovation & Regulatory Science*. 2021;55(1):212–227.
> DOI: [10.1007/s43441-020-00207-2](https://link.springer.com/article/10.1007/s43441-020-00207-2)
> PMC: PMC7505220.

**Key findings:**
- Contract execution averaged **7.9 months** for US sites (range 2.5–17.2 months) and **8.7 months** internationally
- Repeat sites had 28% shorter cycle times than newly selected sites

**Imaging-specific source:**

> Gruszauskas NP, Armato SG 3rd. "Critical Challenges to the Management of Clinical Trial Imaging: Recommendations for the Conduct of Imaging at Investigational Sites." *Academic Radiology*. 2020 Feb;27(2):300–306.
> DOI: [10.1016/j.acra.2019.04.003](https://www.sciencedirect.com/science/article/abs/pii/S1076633219301886)
> PMID: 31097377.

**Citeability:** Good for general startup delays; imaging-specific IT/HIPAA onboarding delays are not precisely quantified in peer-reviewed literature. Cite Lai et al. for the 7.9-month figure with the caveat that it covers all trial types.

---

## Notes for Future Agents

- All citations above were verified as of February 2026
- The Schwarz et al. NEJM paper is the strongest single citation for the defacing rationale — use it
- The Aryanto et al. paper provides the best quantified evidence for de-identification tool failures
- The NIH DMS Policy is a primary government source, not a press interpretation — cite NOT-OD-21-013 directly
- Avoid citing the MIDI-B arXiv preprint as "published" — note it as under review
- Market figures from Fact.MR are acceptable for pitch decks with appropriate sourcing; do not use them in academic or regulatory contexts
