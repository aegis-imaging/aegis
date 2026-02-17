# AEGIS GCP Upload Notes (Human Runbook)

This file documents what you need to configure in GCP for both:

- external-site signed URL uploads (`STORAGE_MODE=gcs`), and
- internal-enterprise ingest flowing through the same processing and sharing controls.

## 1) Required GCP resources

- A GCP project with billing enabled
- A Cloud Storage bucket for staging and raw DICOM objects
- A service account with permission to sign URLs and access bucket objects
- Healthcare API dataset + DICOM stores (`raw`, `clean`)
- A controlled export/download mechanism for external partners (signed URL generation or authorized DICOMweb access)

## 2) Service account IAM

Create (or reuse) a service account for the API runtime.

Minimum roles to start:

- `roles/storage.objectAdmin` on the upload bucket
- `roles/iam.serviceAccountTokenCreator` on the signing service account (if different account is used for signing)

If you are using a downloaded service-account key directly for signing, keep it short-lived and rotate frequently.

## 3) Bucket CORS (required for browser PUT to signed URL)

Create `cors.json`:

```json
[
  {
    "origin": ["http://localhost:3000"],
    "method": ["PUT", "GET", "HEAD", "OPTIONS"],
    "responseHeader": ["Content-Type", "x-goog-resumable"],
    "maxAgeSeconds": 3600
  }
]
```

Apply it:

```bash
gcloud storage buckets update gs://YOUR_BUCKET --cors-file=cors.json
```

## 4) API environment variables

Set these in your API environment (`api/.env`, shell export, or Cloud Run env vars):

```bash
STORAGE_MODE=gcs
GCS_BUCKET=YOUR_BUCKET

# Used by signed URL generation in api/storage/gcs.go
GCS_SIGNING_EMAIL=signer-sa@YOUR_PROJECT.iam.gserviceaccount.com
GCS_SIGNING_PRIVATE_KEY='-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n'
```

Notes:

- `GCS_SIGNING_PRIVATE_KEY` supports PEM with escaped `\n` or base64-encoded key text.
- Keep secrets out of git. Use Secret Manager for Cloud Run where possible.

## 5) Local verification flow

1. Start API with DB available and env vars set.
2. Start upload portal (`frontend/upload-portal`) on `http://localhost:3000`.
3. Select DICOM files and click **Confirm anonymization & upload**.
4. Confirm:
   - `/api/upload/init` returns signed URLs
   - Browser `PUT` uploads succeed to GCS
   - `/api/upload/complete` creates a study record

## 6) Internal-enterprise ingest lane (planning checklist)

For internal-origin studies (already inside enterprise network), plan an intake path that still enters the same audit/QC pipeline:

- Define intake mechanism: API endpoint, DICOMweb STOW-RS bridge, or scheduled batch ingest
- Tag internal-ingest studies with source metadata (`source = internal`) for audit and policy decisions
- Apply same downstream checks: server-side de-id validation, optional defacing, QC approval
- Enforce approval gate before any external sharing/export
- Log every outbound download/export event to audit trail

## 7) Common failure points

- **403 on signed URL PUT**: signing key/email mismatch, expired URL, or bucket IAM issue
- **CORS error in browser**: bucket CORS missing `http://localhost:3000` and `PUT`
- **Init fails**: missing `GCS_SIGNING_EMAIL` / `GCS_SIGNING_PRIVATE_KEY`
- **Complete fails with no files found**: uploads never reached bucket (usually CORS or signed URL permissions)
- **Uncontrolled external sharing risk**: missing approval and audit checks on export/download endpoints
