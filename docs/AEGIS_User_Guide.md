# AEGIS User Guide

**Anonymization & Exchange Gateway for Imaging Studies**

This guide covers the full workflow for three roles: **senders** (sites uploading studies), **administrators** (reviewing and approving studies), and **recipients** (downloading shared studies).

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Role: Sender — Uploading a Study](#2-role-sender--uploading-a-study)
3. [Role: Administrator — Reviewing Studies](#3-role-administrator--reviewing-studies)
4. [Role: Recipient — Downloading a Shared Study](#4-role-recipient--downloading-a-shared-study)
5. [Automated Pipeline — What Happens After Upload](#5-automated-pipeline--what-happens-after-upload)
6. [Frequently Asked Questions](#6-frequently-asked-questions)

---

## 1. System Overview

AEGIS has three separate web interfaces:

| Interface | URL pattern | Who uses it |
|-----------|-------------|-------------|
| **Upload Portal** | `https://<your-instance>/` | Senders at clinical or research sites |
| **Admin Dashboard** | `https://admin.<your-instance>/` | Administrators, data managers, investigators |
| **Export Portal** | `https://<your-instance>/export?token=...` | Recipients of shared studies (link in email) |

No software installation is required. All interfaces run in a standard web browser.

---

## 2. Role: Sender — Uploading a Study

The upload portal handles de-identification entirely in your browser before any data is transmitted. **PHI-bearing DICOM tags are stripped before the first byte leaves your machine.**

### Step-by-step

**Step 1 — Open the upload portal**

Navigate to the upload portal URL provided by your administrator (e.g. `https://upload.aegisimaging.ai`).

**Step 2 — Select a project**

If your institution participates in more than one research project, a project dropdown appears at the top. Select the project this study belongs to. If there is only one project, this step is skipped automatically.

**Step 3 — Add your DICOM files**

You have two options:

- **Drag and drop**: Drag a DICOM folder or a set of `.dcm` files directly onto the upload area.
- **Click to browse**: Click the upload area to open a file picker. To select an entire folder, hold Shift and select all files, or use the folder picker if your browser supports it.

The portal reads the StudyInstanceUID from each file and groups them by study. If you drag a folder with multiple studies, each study is handled as a separate submission.

A summary card appears for each study showing: modality, file count, total size, and any multi-study detection warnings.

**Step 4 — Review the anonymization preview**

Before uploading, AEGIS shows a before-and-after table of every DICOM tag that will be modified. This is your confirmation that de-identification is working correctly.

Key columns:
- **Tag** — the DICOM tag identifier (e.g. `(0010,0010)`)
- **Keyword** — human-readable name (e.g. `PatientName`)
- **Original value** — what is in the file right now
- **Action** — what AEGIS will do: Remove, Zero, Replace, New UID, Keep, or Clean

Retained tags (shown in green) are tags your project administrator has explicitly approved to keep — typically non-identifying acquisition parameters like `Modality`, `MagneticFieldStrength`, or `StudyDate`.

> **Nothing is uploaded at this step.** The preview is computed locally in your browser.

**Step 5 — Enter your email (optional)**

If you want to be notified when the study is approved or rejected, enter your email address. AEGIS will send a brief notification — no PHI is included in the email.

**Step 6 — Click Upload**

Click the **Upload** button. A progress bar shows per-file upload progress. Upload time depends on file count and connection speed.

If a file fails, AEGIS retries it automatically up to three times. You can cancel the upload at any point using the **Cancel** button.

**Step 7 — Done**

Once the upload completes, you will see a confirmation message with the Study UID. The study is now in the AEGIS queue for administrator review and automated processing.

You do not need to do anything else. If you provided an email, you will receive a notification when the study is approved or rejected.

---

## 3. Role: Administrator — Reviewing Studies

Administrators access the **Admin Dashboard** to review incoming studies, monitor the automated processing pipeline, and share approved studies with collaborators.

### Accessing the Dashboard

Navigate to the admin dashboard URL. If your institution uses GCP, this URL is protected by Google Identity-Aware Proxy (IAP) — you will be prompted to log in with your Google account. Your account must be registered by a super-admin before access is granted.

### The Studies Tab

The Studies tab is the main workspace. It shows all studies with their current status.

**Study statuses:**

| Status | Meaning |
|--------|---------|
| `received` | Study uploaded; waiting for pipeline or manual review |
| `defacing` | Automated facial feature removal in progress |
| `defaced` | Defacing complete; pending Phase 2 services |
| `clean` | All automated processing complete; ready for review |
| `approved` | Approved for sharing; export shares can be created |
| `rejected` | Rejected; uploader notified if email was provided |
| `expired` | Previously approved study past its retention window |

**Filtering and searching:**

Use the filter bar at the top of the Studies tab to narrow by status, modality, body part, source (external upload vs internal DIMSE ingest), or project. The search box matches on Study UID and study description.

### Reviewing a Study

Click a Study UID to open the **Study Detail Panel**. This panel shows:

1. **Header** — full Study UID, status and source badges, description
2. **Meta row** — modality, body part, file count, series count, DICOM store, timestamps
3. **Pipeline visualization** — 7-stage horizontal pipeline showing the status of each automated service (Classification → PHI Scan → Protocol → Defacing → QC → BIDS → Export)
4. **Action buttons** — all processing triggers, approve/reject, share creation, DICOM viewer, BIDS download
5. **Tabs** — Audit Trail, Routing Log, Export Shares

### Approving or Rejecting a Study

After reviewing the study (and optionally viewing it in the OHIF viewer), click:

- **Approve** — marks the study `approved`; uploader receives a notification email (if they provided one); auto-export triggers if a routing rule requires it
- **Reject** — marks the study `rejected`; uploader is notified

Both actions are logged in the audit trail with your email address.

### Triggering Processing Services Manually

If `PIPELINE_AUTO=true` (default), services run automatically after upload. You can also trigger them manually via the action buttons:

- **Classify** — detect modality and body part from DICOM headers
- **Scan for PHI** — run OCR to detect burned-in text in pixel data
- **Check Protocol** — compare acquisition parameters against project templates
- **Deface** — remove facial features from head/brain imaging
- **Run QC** — automated image quality checks
- **Convert to BIDS** — convert DICOM to NIfTI/BIDS format

### Creating an Export Share

Once a study is `approved`:

1. Click **Share** in the action buttons
2. Enter the recipient's email address
3. Optionally add a note (e.g. "Brain MRI for project ABCD-42")
4. Set an expiry (e.g. 48 hours, 7 days)
5. Click **Create Share**

The recipient receives an email with a secure download link. The link is time-limited and can be revoked at any time from the Export Shares tab.

### Adding Study Labels

Labels let you tag studies for cohort tracking or workflow notes. On the Study Detail Panel, type a label in the label input and click **+ Label**. Labels appear as chips and are searchable.

For bulk labeling, select multiple studies in the table using the checkboxes, then use the bulk action bar at the bottom of the page.

### Inspecting DICOM Tags

Click **Inspect DICOM Tags** on the Study Detail Panel to see a searchable table of all non-pixel DICOM tags from the study's current store (raw or clean). Useful for verifying that de-identification worked correctly.

---

## 4. Role: Recipient — Downloading a Shared Study

Recipients receive a share link by email. No account or login is required.

**Step 1 — Click the link in the email**

The email contains a URL of the form `https://<instance>/export?token=...`. Click it to open the Export Portal.

**Step 2 — Review the study summary**

The Export Portal shows:
- Study UID and description (no PHI)
- Modality and number of files
- Expiry time (a countdown shows how much time remains)
- An optional note from the sender

**Step 3 — Download**

Click **Download All (ZIP)**. The browser downloads a zip archive containing all DICOM files for the study. Files are already de-identified — no further processing is needed.

If the link has expired or been revoked, you will see an error message. Contact the sender to request a new share.

---

## 5. Automated Pipeline — What Happens After Upload

After a study is uploaded and routing rules evaluate, AEGIS automatically dispatches the following services (if configured and required by routing rules):

```
Phase 0: Classification
    ↓  (fills modality/body_part, re-evaluates routing rules)
Phase 1: PHI scan + Protocol check + Defacing  (parallel)
    ↓  (defacing must complete for head studies before Phase 2)
Phase 2: QC check + BIDS conversion
    ↓
Admin review → Approve/Reject
    ↓  (on approve)
Export forwarding  (if routing rule requires route_to destination)
```

**What each service does:**

| Service | What it checks |
|---------|---------------|
| **Classification** | Reads DICOM headers to determine modality (MR/CT/PET/…) and body part (HEAD/CHEST/…). Triggers routing re-evaluation — e.g. a HEAD study may automatically require defacing. |
| **PHI scan** | Runs OCR on every DICOM slice's pixel data to detect burned-in text (patient names, dates, accession numbers) that tag-level de-identification cannot remove. Flags studies for human review. |
| **Protocol check** | Compares acquisition parameters (TR, TE, flip angle, slice thickness, resolution) against per-project templates. Flags deviations as compliant / minor deviations / non-compliant. |
| **Defacing** | Removes facial features from head/brain DICOM studies using automated segmentation tools (mri_reface, DeepDefacer, mri_deface, or a fast fallback). Prevents face reconstruction from 3D volumetric data. |
| **QC check** | Runs 5 automated quality checks: file integrity, slice consistency, SNR estimation, coverage completeness, and missing slice detection. Flags studies as pass / warn / fail. |
| **BIDS conversion** | Converts DICOM to NIfTI format with a BIDS-compliant directory structure and JSON sidecar metadata. Output is downloadable as a zip archive. |
| **Export forwarding** | Forwards de-identified DICOM files to external DICOMweb or DIMSE destinations (e.g. a partner PACS or a cloud archive). Triggered automatically on approval for studies matched by a `route_to` routing rule. |

All service results are recorded in the **Audit Trail** — accessible from the Study Detail Panel. No PHI appears in any audit entry.

---

## 6. Frequently Asked Questions

**Q: Does AEGIS transmit my original DICOM files?**

No. The upload portal applies de-identification in your browser before transmitting any data. Only the de-identified (anonymized) files are sent to the server. The original files remain on your machine.

**Q: What if I don't have a DICOM viewer — can I still verify the study?**

Yes. The admin dashboard includes an embedded OHIF Viewer. Click **View** on any study to open it inline, or **Open in new tab** to view it full-screen.

**Q: What happens if defacing or PHI scan fails?**

The study reverts to `received` status and an alert email is sent to the `PIPELINE_ALERT_EMAIL` address (if configured). An administrator can re-trigger the step manually from the Study Detail Panel using the **Reset pipeline step** option.

**Q: Can I upload studies that are already de-identified?**

Yes. You can use the batch import CLI (`aegis-import`) for bulk historical data. This path skips client-side de-identification (files are assumed already de-identified) and copies files directly to the raw store, then triggers routing rules and the pipeline normally.

**Q: What file formats are supported?**

DICOM (`.dcm`) files from any modality. The upload portal accepts individual files or entire folder trees. DICOM Part 10 files are required (standard output from any PACS or scanner).

**Q: Can multiple studies be in one upload?**

Yes. If you drag a folder containing multiple StudyInstanceUIDs, the portal automatically groups them and creates one submission per study. Each study gets its own progress bar and summary card.

**Q: How long does a share link last?**

The expiry is set by the administrator when creating the share (typically 24–168 hours). The Export Portal shows a live countdown. Expired or revoked links display an error — contact the sender for a new link.

**Q: Who can see the audit trail?**

All administrators (admin and viewer roles) can see the audit trail. It records every action — upload, processing events, approvals, share creation, downloads — with actor email, timestamp, and IP address. No PHI appears in any audit entry.

**Q: How do I get access to the admin dashboard?**

Your institution's AEGIS administrator must register your email in the admin user list. Contact them and provide your Google account email (for GCP/IAP deployments) or your Azure AD email (for Azure deployments).

---

*For technical setup, deployment, and configuration, see [SETUP_CHECKLIST.md](../SETUP_CHECKLIST.md) and [CLAUDE.md](../CLAUDE.md).*
