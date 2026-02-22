# DICOM Conformance Statement

**Application**: Anonymization & Exchange Gateway for Imaging Studies (AEGIS)
**Version**: 1.0
**Date**: 2026-02-22
**Standard**: DICOM PS3 (2024c)

---

## 1. Introduction

This conformance statement describes the DICOM capabilities of the AEGIS platform. AEGIS is a HIPAA-compliant medical imaging data exchange and anonymization system supporting brain MRI, PET, CT, and all other standard DICOM modalities.

AEGIS provides two DICOM network interfaces:

| Interface | Role | Protocol |
|-----------|------|---------|
| **DIMSE Receiver** | Storage SCP | DICOM C-STORE (DIMSE, port 11112) |
| **DICOMweb Proxy** | Query/Retrieve SCU (read-only) | QIDO-RS + WADO-RS (HTTP/HTTPS) |

---

## 2. Implementation Model

### 2.1 Application Entity Characteristics

| AE Title | Role | Network Port | Transport |
|----------|------|-------------|-----------|
| `AEGIS` (configurable via `DIMSE_AE_TITLE`) | Storage SCP | 11112 (configurable via `DIMSE_PORT`) | TCP |
| *(calling AE — OHIF Viewer or admin client)* | DICOMweb SCU | 8080 (Go API HTTP) | HTTP/HTTPS |

### 2.2 Association Policies

**Storage SCP (DIMSE Receiver):**
- Maximum simultaneous associations: 10 (configurable via `DIMSE_MAX_ASSOCIATIONS`)
- Association initiator: no (SCP only)
- Association acceptor: yes
- Asynchronous operations: not supported
- Extended negotiation: not supported

---

## 3. DIMSE Storage SCP

### 3.1 Supported Transfer Syntaxes

The DIMSE Receiver accepts the following transfer syntaxes for all SOP classes:

| Transfer Syntax UID | Name |
|--------------------|------|
| `1.2.840.10008.1.2` | Implicit VR Little Endian |
| `1.2.840.10008.1.2.1` | Explicit VR Little Endian (preferred) |
| `1.2.840.10008.1.2.2` | Explicit VR Big Endian (deprecated, accepted for compatibility) |
| `1.2.840.10008.1.2.4.50` | JPEG Baseline (Process 1) |
| `1.2.840.10008.1.2.4.51` | JPEG Extended (Process 2 & 4) |
| `1.2.840.10008.1.2.4.57` | JPEG Lossless (Process 14) |
| `1.2.840.10008.1.2.4.70` | JPEG Lossless SV1 (Process 14, Selection Value 1) |
| `1.2.840.10008.1.2.4.90` | JPEG 2000 Lossless |
| `1.2.840.10008.1.2.4.91` | JPEG 2000 |
| `1.2.840.10008.1.2.5` | RLE Lossless |

### 3.2 Supported SOP Classes

The DIMSE Receiver accepts all standard DICOM Storage SOP classes, including but not limited to:

| SOP Class UID | Name |
|--------------|------|
| `1.2.840.10008.5.1.4.1.1.4` | MR Image Storage |
| `1.2.840.10008.5.1.4.1.1.4.1` | Enhanced MR Image Storage |
| `1.2.840.10008.5.1.4.1.1.4.4` | Legacy Converted Enhanced MR Image Storage |
| `1.2.840.10008.5.1.4.1.1.2` | CT Image Storage |
| `1.2.840.10008.5.1.4.1.1.2.1` | Enhanced CT Image Storage |
| `1.2.840.10008.5.1.4.1.1.128` | Positron Emission Tomography Image Storage |
| `1.2.840.10008.5.1.4.1.1.128.1` | Enhanced PET Image Storage |
| `1.2.840.10008.5.1.4.1.1.7` | Secondary Capture Image Storage |
| `1.2.840.10008.5.1.4.1.1.7.1` | Multi-frame Single Bit SC Image Storage |
| `1.2.840.10008.5.1.4.1.1.7.2` | Multi-frame Grayscale Byte SC Image Storage |
| `1.2.840.10008.5.1.4.1.1.7.3` | Multi-frame Grayscale Word SC Storage |
| `1.2.840.10008.5.1.4.1.1.7.4` | Multi-frame True Color SC Image Storage |
| `1.2.840.10008.5.1.4.1.1.481.1` | RT Image Storage |
| `1.2.840.10008.5.1.4.1.1.481.2` | RT Dose Storage |
| `1.2.840.10008.5.1.4.1.1.481.3` | RT Structure Set Storage |
| `1.2.840.10008.5.1.4.1.1.481.5` | RT Plan Storage |
| `1.2.840.10008.5.1.4.1.1.20` | NM Image Storage |
| `1.2.840.10008.5.1.4.1.1.6.1` | Ultrasound Image Storage |
| `1.2.840.10008.5.1.4.1.1.6.2` | Enhanced US Volume Storage |
| `1.2.840.10008.5.1.4.1.1.1` | CR Image Storage |
| `1.2.840.10008.5.1.4.1.1.1.1` | Digital X-Ray Image Storage (For Presentation) |
| `1.2.840.10008.5.1.4.1.1.1.2` | Digital Mammography X-Ray Image Storage (For Presentation) |
| `1.2.840.10008.5.1.4.1.1.1.3` | Digital Intra-Oral X-Ray Image Storage (For Presentation) |
| `1.2.840.10008.5.1.4.1.1.3.1` | Ultrasound Multi-frame Image Storage |
| `1.2.840.10008.5.1.4.1.1.12.1` | X-Ray Angiographic Image Storage |
| `1.2.840.10008.5.1.4.1.1.12.2` | X-Ray Radiofluoroscopic Image Storage |
| `1.2.840.10008.5.1.4.1.1.77.1.1` | VL Endoscopic Image Storage |
| `1.2.840.10008.5.1.4.1.1.77.1.4` | VL Photographic Image Storage |
| `1.2.840.10008.5.1.4.1.1.11.1` | Grayscale Softcopy Presentation State Storage |
| `1.2.840.10008.5.1.4.1.1.11.2` | Color Softcopy Presentation State Storage |
| `1.2.840.10008.5.1.4.1.1.88.11` | Basic Text SR Storage |
| `1.2.840.10008.5.1.4.1.1.88.22` | Enhanced SR Storage |
| `1.2.840.10008.5.1.4.1.1.88.33` | Comprehensive SR Storage |
| `1.2.840.10008.5.1.4.1.1.104.1` | Encapsulated PDF Storage |

> The receiver uses pynetdicom's `AllStoragePresentationContexts` list, which accepts all
> DICOM-registered Storage SOP classes. The table above lists the most commonly encountered
> classes in research and clinical imaging.

### 3.3 DIMSE Services

| Service | Role | Notes |
|---------|------|-------|
| C-STORE | SCP | Accepts all Storage SOP classes |
| C-ECHO | SCP | Responds to PACS connectivity verification |

### 3.4 Ingest Behaviour

On each C-STORE association close (`EVT_RELEASED`):
1. All received files are written to `dicom/raw/{StudyInstanceUID}/{index}.dcm` in shared storage.
2. A `POST /api/ingest` call is made to the Go API, triggering study creation and routing rule evaluation.
3. The calling AE title (`institution_ae_title`) is forwarded for institution auto-attribution.
4. If the ingest call fails, the study is queued for automatic retry (exponential backoff, configurable max attempts).

---

## 4. DICOMweb Proxy (QIDO-RS + WADO-RS)

The Go API exposes a minimal DICOMweb proxy at `/dicomweb` (and `/dicomweb-raw` for pre-defacing access). This interface is read-only and is consumed by the OHIF Viewer embedded in the admin dashboard.

### 4.1 QIDO-RS (Query)

| Endpoint | DICOM Standard Reference |
|----------|--------------------------|
| `GET /dicomweb/studies` | PS3.18 §10.6.1 — Search for Studies |
| `GET /dicomweb/studies/{studyUID}/series` | PS3.18 §10.6.2 — Search for Series |
| `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances` | PS3.18 §10.6.3 — Search for Instances |

**Supported QIDO-RS query parameters:**

| Parameter | Studies | Series | Instances |
|-----------|---------|--------|-----------|
| `StudyInstanceUID` | ✓ | — | — |
| `limit` | ✓ | ✓ | ✓ |
| `offset` | ✓ | ✓ | ✓ |

> Note: The proxy generates synthetic series/instance metadata derived from AEGIS study records.
> Full tag-level QIDO filtering (e.g., by PatientName, StudyDate) is not implemented — use the
> AEGIS REST API (`GET /api/studies`) for advanced filtering.

### 4.2 WADO-RS (Retrieve)

| Endpoint | DICOM Standard Reference |
|----------|--------------------------|
| `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}` | PS3.18 §10.4 — Retrieve Instance |

- Returns raw DICOM bytes (`application/dicom` content type).
- SOPInstanceUID format: `{StudyInstanceUID}.1.{fileIndex}` where `fileIndex` maps to a file in the study's DICOM store.
- `/dicomweb-raw` endpoint always reads from `dicom/raw/` (pre-defacing store) regardless of study state.
- `/dicomweb` endpoint reads from the study's current `dicom_store` (`raw` or `clean`).

### 4.3 Not Supported

The following DICOMweb services are **not** implemented in this version:

- STOW-RS (ingest via DICOMweb) — use browser upload portal or DIMSE C-STORE
- WADO-URI (legacy URL-based retrieve)
- UPS-RS (Unified Procedure Step)
- Bulk Data retrieve (`/bulkdata`)
- Thumbnail retrieve (`/thumbnail`)

---

## 5. De-identification

AEGIS applies DICOM PS3.15 Annex E — Basic Application Level Confidentiality Profile (de-identification) at the browser before upload. The de-identification is performed in the client's browser via the `@aegis/client` TypeScript library, which applies the Basic Profile (E.1.1) tag actions: keep, remove, empty, replace, or UID-remap.

**Additional de-identification steps (server-side):**

| Step | Scope | Reference |
|------|-------|-----------|
| Facial feature removal (defacing) | Head/brain MRI, PET, CT | NIST SP 800-188; mri_deface / DeepDefacer backends |
| Burned-in PHI detection | All modalities | PS3.15 §E.3 — pixel data de-identification |

**Per-project retained tag overrides**: Administrators may configure named anonymization profiles that preserve specific DICOM tags (e.g., `StudyDate`, `PatientAge`) for research projects that require them under a Data Use Agreement.

---

## 6. Security

- **Authentication**: GCP Identity-Aware Proxy (IAP), Azure AD Easy Auth, or AWS ALB + Cognito in production. Dev mode: auto-authenticate via `DEV_USER_EMAIL`.
- **Transport security**: All HTTP endpoints should be deployed behind an HTTPS load balancer (enforced by Terraform modules). The DIMSE port (11112) is a plain TCP/DICOM port — TLS wrapping via stunnel or a VPN is recommended for inter-site deployments.
- **API keys**: Machine-to-machine integrations use HMAC-SHA256 hashed API keys stored in the database.
- **DIMSE operator key**: `/ingest/retry*` control endpoints can be protected by `DIMSE_OPERATOR_API_KEY`.

---

## 7. Limitations and Known Deviations

| Item | Description |
|------|-------------|
| C-FIND / C-MOVE | Not implemented. Query/retrieve from PACS is not supported in this release. Use STOW-RS from PACS or DIMSE C-STORE push. |
| C-GET | Not implemented. |
| WADO-RS multipart/related bulk retrieve | Not implemented. Instances are retrieved one at a time. |
| QIDO-RS tag-level filtering | Not implemented in the proxy. Use the AEGIS REST API for study-level filtering. |
| Enhanced DICOM (multi-frame) | Accepted and stored as-is. Protocol compliance service reads Enhanced DICOM functional groups; viewer support depends on OHIF version. |
| Study-level merging | Multiple associations for the same StudyInstanceUID are addended to the existing study record. |
| Large series (>10,000 instances) | Not performance-tested. File count is stored but no chunked-retrieval pagination is implemented on the WADO-RS endpoint. |
| TLS on DIMSE port | Not natively supported. Deploy behind a TLS proxy (stunnel, Nginx stream) for encrypted PACS communication. |

---

## 8. References

- DICOM Standard: PS3 (2024c) — https://www.dicomstandard.org/current
- PS3.15 Annex E — Basic Application Level Confidentiality Profile
- PS3.18 — Web Services (DICOMweb)
- pynetdicom — https://pydicom.github.io/pynetdicom/
- OHIF Viewer — https://ohif.org/
