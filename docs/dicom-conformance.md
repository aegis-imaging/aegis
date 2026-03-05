# DICOM Conformance Statement

**Application**: Anonymization & Exchange Gateway for Imaging Studies (AEGIS)
**Version**: 1.3
**Date**: 2026-02-28
**Standard**: DICOM PS3 (2024c)

---

## 1. Introduction

This conformance statement describes the DICOM capabilities of the AEGIS platform. AEGIS is a HIPAA-compliant medical imaging data exchange and anonymization system supporting brain MRI, PET, CT, and all other standard DICOM modalities.

AEGIS provides the following DICOM network interfaces:

| Interface | Role | Protocol |
|-----------|------|---------|
| **DIMSE Receiver** | Storage SCP | DICOM C-STORE (DIMSE, port 11112) |
| **DIMSE Query** | C-FIND SCU | Query remote PACS/AE for matching studies (via sidecar `/api/dimse/query`) |
| **STOW-RS Receiver** | Storage SCU (receive) | DICOMweb STOW-RS (`POST /api/stow/studies`) |
| **DICOMweb Proxy** | Query/Retrieve (read-only) | QIDO-RS + WADO-RS (HTTP/HTTPS) |
| **DIMSE Forwarder** | C-STORE SCU | Forward studies to remote DIMSE destinations (routing rules) |

AEGIS is cloud-agnostic. The same application stack is certified for deployment on Google Cloud Platform (GCP), Amazon Web Services (AWS), and Microsoft Azure. See §9 for cloud-specific deployment notes.

---

## 2. Implementation Model

### 2.1 Application Entity Characteristics

| AE Title | Role | Network Port | Transport |
|----------|------|-------------|-----------|
| `AEGIS` (configurable via `DIMSE_AE_TITLE`) | Storage SCP, C-FIND SCU, C-STORE SCU | 11112 (configurable via `DIMSE_PORT`) | TCP |
| *(calling AE — DWV viewer or admin client)* | DICOMweb SCU | 8080 (Go API HTTP) | HTTP/HTTPS |

### 2.2 Association Policies

**Storage SCP (DIMSE Receiver):**
- Maximum simultaneous associations: 10 (configurable via `DIMSE_MAX_ASSOCIATIONS`)
- Association initiator: no (SCP only for inbound C-STORE)
- Association acceptor: yes
- Asynchronous operations: not supported
- Extended negotiation: not supported

**C-FIND SCU (Outbound Query):**
- Association initiator: yes
- Association acceptor: no
- Query Information Models: Study Root (default); Patient Root (when `query_level=PATIENT`)
- Triggered by: `POST /api/dimse/query` admin API endpoint

**C-STORE SCU (Outbound Forwarding):**
- Association initiator: yes
- Association acceptor: no
- Triggered by: routing rules with `route_to` action pointing to a DIMSE destination

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
| `1.2.840.10008.1.2.4.80` | JPEG-LS Lossless |
| `1.2.840.10008.1.2.4.81` | JPEG-LS Near-Lossless |
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

### 3.3 DIMSE Services — SCP

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

## 4. DIMSE C-FIND SCU (Outbound Query)

AEGIS can query remote PACS/AE systems using C-FIND SCU via the admin API endpoint `POST /api/dimse/query`. The query is proxied through the DIMSE receiver sidecar.

### 4.1 Supported Query Information Models

| Information Model | SOP Class UID | `query_level` values |
|-------------------|--------------|---------------------|
| Study Root Q/R — FIND | `1.2.840.10008.5.1.4.1.2.2.1` | `STUDY`, `SERIES`, `IMAGE` (default: `STUDY`) |
| Patient Root Q/R — FIND | `1.2.840.10008.5.1.4.1.2.1.1` | `PATIENT`, `STUDY`, `SERIES`, `IMAGE` |

### 4.2 API Request

```
POST /api/dimse/query
Authorization: Bearer <api-key>
Content-Type: application/json

{
  "ae_title": "REMOTE_PACS",
  "host": "pacs.example.com",
  "port": 11112,
  "query_level": "STUDY",
  "query_params": {
    "PatientID": "12345",
    "StudyDate": "20240101-20240201",
    "StudyInstanceUID": "",
    "StudyDescription": ""
  }
}
```

**Response:**
```json
{"matches": [{"StudyInstanceUID": "...", "PatientID": "...", ...}], "count": N}
```

Empty-string values in `query_params` are treated as wildcard matches (standard DICOM C-FIND behaviour). Any DICOM keyword accepted by pydicom's `Dataset` may be specified.

---

## 5. DICOMweb Proxy (QIDO-RS, WADO-RS, STOW-RS)

The Go API exposes DICOMweb endpoints at `/dicomweb` (and `/dicomweb-raw` for pre-defacing access). The proxy is consumed by the DWV viewer embedded in the admin dashboard, and by STOW-RS clients forwarding studies cross-cloud.

### 5.1 QIDO-RS (Query)

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

### 5.2 WADO-RS (Retrieve)

| Endpoint | DICOM Standard Reference |
|----------|--------------------------|
| `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}` | PS3.18 §10.4 — Retrieve Instance |
| `GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}/metadata` | PS3.18 §10.4.1 — Retrieve Instance Metadata |

- Instance retrieval returns raw DICOM bytes (`application/dicom` content type).
- Metadata retrieval returns DICOMweb JSON (pixel data excluded).
- SOPInstanceUID format: `{StudyInstanceUID}.1.{fileIndex}` where `fileIndex` maps to a file in the study's DICOM store.
- `/dicomweb-raw` endpoints always read from `dicom/raw/` (pre-defacing store) regardless of study state.
- `/dicomweb` endpoints read from the study's current `dicom_store` (`raw` or `clean`).

### 5.3 STOW-RS (Store — Cross-Cloud Routing)

AEGIS implements a STOW-RS receiver at `POST /api/stow/studies` for receiving forwarded studies from other cloud deployments (cross-cloud routing). This endpoint is **not** a general-purpose PACS ingestion path — it is used internally for GCP ↔ AWS ↔ Azure study replication via routing rules.

| Endpoint | DICOM Standard Reference |
|----------|--------------------------|
| `POST /api/stow/studies` | PS3.18 §10.5.1 — Store Instances |

**Authentication**: API key (`Authorization: Bearer <key>`) required. The API key is registered in the destination tenancy's `api_keys` table. On each cross-cloud routing rule with `route_to` action pointing to a DICOMweb destination, the source tenancy injects the key via `dicomweb_auth_header`.

**Accepted media types**: `multipart/related; type="application/dicom"` (standard STOW-RS).

### 5.4 Not Supported

The following DICOMweb services are **not** implemented in this version:

- WADO-URI (legacy URL-based retrieve)
- UPS-RS (Unified Procedure Step)
- Bulk Data retrieve (`/bulkdata`)
- Thumbnail retrieve (`/thumbnail`)
- STOW-RS for general external PACS push (use DIMSE C-STORE or the browser upload portal)

---

## 6. De-identification

AEGIS applies DICOM PS3.15 Annex E — Basic Application Level Confidentiality Profile (de-identification) at the browser before upload. The de-identification is performed in the client's browser via the `@aegis/client` TypeScript library, which applies the Basic Profile (E.1.1) tag actions: keep, remove, empty, replace, or UID-remap.

**Additional de-identification steps (server-side):**

| Step | Scope | Reference |
|------|-------|-----------|
| Facial feature removal (defacing) | Head/brain MRI, PET, CT | NIST SP 800-188; mri_deface / DeepDefacer backends |
| Burned-in PHI detection | All modalities | PS3.15 §E.3 — pixel data de-identification |
| Pixel redaction | All modalities | Detects and masks burned-in PHI text in pixel data |
| Vendor private tag PHI scanning | All modalities | Scans private tags for embedded PHI before preservation |

**Per-project retained tag overrides**: Administrators may configure named anonymization profiles that preserve specific DICOM tags (e.g., `StudyDate`, `PatientAge`) for research projects that require them under a Data Use Agreement.

**Vendor private tag preservation**: When `keep_private_tags` is enabled on an anonymization profile, vendor-specific private tags (odd-group tags) are retained instead of being removed by the Basic Profile. A dedicated PHI scanning service inspects private tag values for embedded patient identifiers before preservation.

---

## 7. Security

- **Authentication**: GCP Identity-Aware Proxy (IAP), Azure AD Easy Auth, or AWS ALB + Cognito in production. Dev mode: auto-authenticate via `DEV_USER_EMAIL`.
- **Transport security**: All HTTP endpoints are deployed behind an HTTPS load balancer (enforced by Terraform modules). The DIMSE port (11112) is a plain TCP/DICOM port — TLS wrapping via stunnel or a VPN is recommended for inter-site deployments.
- **API keys**: Machine-to-machine integrations use SHA-256 hashed API keys stored in the database.
- **DIMSE operator key**: `/ingest/retry*` control endpoints can be protected by `DIMSE_OPERATOR_API_KEY`.

---

## 8. Limitations and Known Deviations

| Item | Description |
|------|-------------|
| C-MOVE | Not implemented. C-FIND query (SCU) is supported; C-MOVE retrieve is not. PACS must push via C-STORE or STOW-RS. |
| C-GET | Not implemented. |
| WADO-RS multipart/related bulk retrieve | Not implemented. Instances are retrieved one at a time. |
| QIDO-RS tag-level filtering | Not implemented in the proxy. Use the AEGIS REST API for study-level filtering. |
| Enhanced DICOM (multi-frame) | Fully supported. Protocol compliance reads Enhanced functional groups (`SharedFunctionalGroupsSequence`, `PerFrameFunctionalGroupsSequence`). Pixel processing services (PHI detection, QC, pixel redaction) handle multi-frame arrays `(frames, rows, cols)`. |
| Siemens Mosaic DICOM | Detected via `ImageType` tag containing `MOSAIC`. Handled correctly during BIDS conversion — dcm2niix unwraps mosaic tiles into individual slices. Pixel processing operates on the mosaic tile as-is. |
| Compressed DICOM pixel processing | Pixel processing services (PHI detection, QC, pixel redaction) transparently decompress compressed DICOM via python-gdcm and pylibjpeg backends. Supported compressed formats: JPEG Baseline, JPEG Lossless, JPEG 2000, JPEG-LS, and RLE. No manual decompression step is required. |
| Study-level merging | Multiple associations for the same StudyInstanceUID are addended to the existing study record. |
| Large series (>10,000 instances) | Not performance-tested. File count is stored but no chunked-retrieval pagination is implemented on the WADO-RS endpoint. |
| TLS on DIMSE port | Not natively supported. Deploy behind a TLS proxy (stunnel, Nginx stream) for encrypted PACS communication. |
| C-FIND SCU — Patient Root SERIES/IMAGE levels | Patient Root is supported at PATIENT and STUDY levels. SERIES and IMAGE levels use Study Root regardless of model selection. |

---

## 9. Multi-Cloud Deployment Notes

AEGIS is certified for deployment on three cloud platforms. The DICOM protocol behaviour is identical across all clouds; only the infrastructure hosting the services differs.

### 9.1 Google Cloud Platform (GCP) — Production

| Component | Platform Resource |
|-----------|-----------------|
| Go API | Cloud Run (`aegis-api`) |
| DIMSE Receiver SCP | Compute Engine VM (Debian 12, static IP `35.232.172.221`, port 11112) — operational |
| DICOM Storage | Google Cloud Storage (GCS), `STORAGE_MODE=gcs` |
| Auth | IAP (`AUTH_PROVIDER=iap`), header `X-Goog-Authenticated-User-Email` |
| Cross-cloud routing | STOW-RS to AWS/Azure via routing rules + API key auth |

### 9.2 Amazon Web Services (AWS)

| Component | Platform Resource |
|-----------|-----------------|
| Go API | ECS Fargate |
| DIMSE Receiver SCP | EC2 instance (Amazon Linux 2023, Elastic IP, port 11112) — operational |
| DICOM Storage | S3 (SSE-KMS), `STORAGE_MODE=s3` |
| Auth | ALB + Cognito (`AUTH_PROVIDER=aws`), OIDC JWT via `X-Amzn-Oidc-Data` header |
| Cross-cloud routing | STOW-RS to GCP via routing rules + API key auth |

> **WAF note**: The AWS WAF has a dedicated `allow-stow-rs` rule at priority 15 to bypass the `SizeRestrictions_BODY` managed rule for `POST /api/stow` paths, as DICOM multipart payloads regularly exceed the WAF's default 8 KB body limit.

### 9.3 Microsoft Azure

| Component | Platform Resource |
|-----------|-----------------|
| Go API | Azure Container Apps |
| DIMSE Receiver SCP | Azure Linux VM (Standard_B2s, Debian 12, static IP `20.97.180.87`, port 11112) — operational |
| DICOM Storage | Azure Blob Storage, `STORAGE_MODE=azure` |
| Auth | Easy Auth (`AUTH_PROVIDER=azure`), header `X-MS-CLIENT-PRINCIPAL-NAME` |
| Email (SMTP) | Azure Communication Services Email relay (`smtp.azurecomm.net:587`) |
| Cross-cloud routing | STOW-RS to GCP/AWS via routing rules + API key auth |

### 9.4 Cross-Cloud DICOM Routing

Bidirectional cross-cloud study forwarding is implemented using DICOMweb STOW-RS:

1. A routing rule with `action=route_to` and a DICOMweb destination (`dicomweb_url = https://{cloud}/api/stow`) is configured in the source tenancy.
2. On study approval, the Go API streams DICOM files as `multipart/related` to the destination STOW-RS endpoint.
3. The destination tenancy receives the study via `/api/stow/studies`, creates a study record, and runs its own routing rules.
4. Duplicate StudyInstanceUIDs are rejected by a unique database constraint — the loop terminates after one hop.

**DIMSE alternative:** Cross-cloud forwarding can also use DIMSE C-STORE via `route_to` rules with `type=dimse` destinations. The source cloud's API proxies the C-STORE through its local DIMSE receiver's `/forward` endpoint to the destination cloud's DIMSE receiver IP on port 11112. Each cloud restricts inbound DIMSE traffic via the `dimse_source_ranges` Terraform variable. See `docs/runbooks/cross-cloud-routing.md` for setup instructions.

---

## 10. References

- DICOM Standard: PS3 (2024c) — https://www.dicomstandard.org/current
- PS3.15 Annex E — Basic Application Level Confidentiality Profile
- PS3.18 — Web Services (DICOMweb)
- pynetdicom — https://pydicom.github.io/pynetdicom/
- DWV (DICOM Web Viewer) — https://ivmartel.github.io/dwv/
