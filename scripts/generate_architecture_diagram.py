#!/usr/bin/env python3
"""
Generate AEGIS architecture diagram.

Outputs:
  AEGIS_Architecture_Diagram.html  — always written (open in any browser)
  AEGIS_Architecture_Diagram.pdf   — via Chrome headless (or weasyprint fallback)
  AEGIS_Architecture_Diagram.png   — via Chrome headless screenshot

Run: python3 scripts/generate_architecture_diagram.py
"""

import base64
import os
import subprocess
import shutil

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)

# ── Color palette ──────────────────────────────────────────────────────────────
C = {
    "go":     "#00ADD8",
    "py":     "#3776AB",
    "react":  "#0EA5E9",
    "gcp":    "#4285F4",
    "orange": "#EA580C",
    "purple": "#7C3AED",
    "amber":  "#D97706",
    "cyan":   "#0891B2",
    "red":    "#EA580C",   # orange-600 — colorblind-safe (was #DC2626 red)
    "green":  "#0F766E",   # teal-700 — colorblind-safe (was #15803D green)
    "navy":   "#1D4ED8",
    "azure":  "#0078D4",
    "slate":  "#374151",
    "emerald":"#0D9488",   # teal-600 — colorblind-safe (was #059669 emerald)
}


# ── HTML helpers ───────────────────────────────────────────────────────────────

def card(title, items, color):
    """Service card: filled coloured header + bullet list body."""
    lis = "".join(f"<li>{item}</li>" for item in items)
    return (
        f'<div class="card" style="--c:{C[color]}">'
        f'<div class="card-hdr">{title}</div>'
        f"<ul>{lis}</ul>"
        f"</div>"
    )


def info_box(title, sections, color, bg):
    """Multi-section panel with coloured title (no filled bar)."""
    body = ""
    for heading, items in sections:
        lis = "".join(f"<li>{item}</li>" for item in items)
        body += f'<div class="ib-sec">{heading}</div><ul>{lis}</ul>'
    return (
        f'<div class="info-box" style="--c:{C[color]};background:{bg}">'
        f'<div class="ib-title">{title}</div>'
        f"{body}"
        f"</div>"
    )


def kv_table(rows):
    """Key-value rows for tech stack / multi-cloud sections."""
    html = ""
    for key, val, color in rows:
        html += (
            f'<div class="kv">'
            f'<span class="kv-k" style="color:{C[color]}">{key}</span>'
            f'<span class="kv-v">{val}</span>'
            f"</div>"
        )
    return html


# ── CSS ────────────────────────────────────────────────────────────────────────

CSS = """
*  { box-sizing: border-box; margin: 0; padding: 0; }

@page {
  size: 500mm 420mm;   /* wide landscape — height sized to hold all content on one page */
  margin: 5mm;
}

body {
  font-family: Helvetica Neue, Helvetica, Arial, sans-serif;
  font-size: 10px;
  line-height: 1.35;
  color: #0F172A;
  background: #F8FAFC;
}

.page {
  width: 1875px;          /* matches 500mm @ 96 dpi exactly */
  padding: 6px 10px 10px;
}

/* ── Header ─────────────────────────────────────────────────────────────── */
.hdr {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 7px;
  border-bottom: 1.5px solid #CBD5E1;
  margin-bottom: 7px;
}
.hdr img   { height: 58px; }
.hdr-title { font-size: 24px; font-weight: 700; color: #0F172A; }
.hdr-sub   { font-size: 13px; color: #64748B; margin-top: 2px; }

/* ── Zone wrapper ────────────────────────────────────────────────────────── */
.zone {
  border-radius: 8px;
  border: 1.5px solid #CBD5E1;
  padding: 7px 9px;
  margin-bottom: 6px;
}
.zone-hdr {
  font-size: 12px;
  font-weight: 700;
  margin-bottom: 6px;
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}
.zone-sub { font-size: 9.5px; color: #64748B; font-weight: normal; }

.z-src  { background: #EFF6FF; border-color: #1D4ED8; }
.z-src  .zone-hdr { color: #1D4ED8; }
.z-gcp  { background: #F0FDFA; border-color: #0F766E; }
.z-gcp  .zone-hdr { color: #0F766E; }
.z-aws  { background: #FFF7ED; border-color: #EA580C; }
.z-aws  .zone-hdr { color: #EA580C; }
.z-azure { background: #EFF6FF; border-color: #0078D4; }
.z-azure .zone-hdr { color: #0078D4; }
.z-pipe { background: #EFF6FF; border-color: #1D4ED8; }
.z-pipe .zone-hdr { color: #1D4ED8; }
.z-ph   { background: #FAF5FF; border-color: #7C3AED; }
.z-ph   .zone-hdr { color: #7C3AED; }
.z-tech { background: #F8FAFC; border-color: #CBD5E1; }

/* ── Service cards ───────────────────────────────────────────────────────── */
.card {
  border: 1.5px solid var(--c);
  border-radius: 7px;
  background: #fff;
  overflow: hidden;          /* header bar stays clipped */
  display: flex;
  flex-direction: column;
}
.card-hdr {
  background: var(--c);
  color: #fff;
  font-weight: 700;
  font-size: 10.5px;
  padding: 5px 10px;
  flex-shrink: 0;
}
.card ul {
  list-style: none;
  padding: 6px 10px 7px;
  flex: 1;
}
.card li {
  font-size: 9px;
  color: #334155;
  padding: 1.5px 0;
}
.card li::before { content: "• "; color: var(--c); font-weight: 700; }

/* ── Info boxes (no filled header) ──────────────────────────────────────── */
.info-box {
  border: 1.5px solid var(--c);
  border-radius: 7px;
  padding: 6px 9px 7px;
  overflow: hidden;
}
.ib-title { font-size: 10.5px; font-weight: 700; color: var(--c); margin-bottom: 5px; }
.ib-sec   { font-size: 9.5px;  font-weight: 700; color: var(--c); margin: 5px 0 2px; }
.info-box ul  { list-style: none; }
.info-box li  { font-size: 9px; color: #334155; padding: 1px 0; }
.info-box li::before { content: "• "; color: var(--c); font-weight: 700; }

/* ── LB bar ──────────────────────────────────────────────────────────────── */
.lb-bar {
  display: flex;
  align-items: center;
  gap: 0;
  flex-wrap: wrap;
  background: #DBEAFE;
  border: 1.5px solid #4285F4;
  border-radius: 6px;
  padding: 5px 12px;
  font-size: 9.5px;
  color: #334155;
  margin-bottom: 6px;
}
.lb-bar strong { color: #4285F4; font-size: 10px; font-weight: 700; margin-right: 6px; }
.lb-bar span   { margin-right: 18px; }

/* ── Grid helpers ────────────────────────────────────────────────────────── */
.g2 { display: grid; grid-template-columns: 1fr 1fr;           gap: 7px; }
.g3 { display: grid; grid-template-columns: repeat(3,1fr);     gap: 7px; }
.g4 { display: grid; grid-template-columns: repeat(4,1fr);     gap: 7px; }
.g5 { display: grid; grid-template-columns: repeat(5,1fr);     gap: 7px; }

/* core: API(29%) | Dashboard(32%) | Infra(39%) */
.core-grid {
  display: grid;
  grid-template-columns: 29fr 32fr 39fr;
  gap: 7px;
  margin-bottom: 6px;
}
.infra-stack { display: flex; flex-direction: column; gap: 7px; }

/* sidecars */
.sidecar-lbl {
  font-size: 10px;
  font-weight: 700;
  color: #3776AB;
  margin: 6px 0 4px;
}
/* row2: 3 cards same width as 1 of 4 in row1 */
.row2 {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 7px;
  width: 75%;          /* ≈ 3/4 of parent so each card ≈ 1/4 */
  margin-top: 6px;
}

/* ── Privacy callout ─────────────────────────────────────────────────────── */
.privacy {
  background: #FFF7ED;
  border: 1.5px solid #EA580C;
  border-radius: 7px;
  padding: 7px 10px;
}
.privacy-title { font-size: 10.5px; font-weight: 700; color: #EA580C; margin-bottom: 4px; }
.privacy-main  { font-size: 10px;   font-weight: 700; color: #0F172A; margin-bottom: 5px; }
.privacy ul    { list-style: none; }
.privacy li    { font-size: 9px; color: #334155; padding: 1.5px 0; }
.privacy li::before { content: "• "; color: #EA580C; font-weight: 700; }

/* ── Pipeline stages ─────────────────────────────────────────────────────── */
.pipe-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
  margin-top: 5px;
}
.stage {
  border-radius: 6px;
  padding: 6px 6px 5px;
  text-align: center;
  background: var(--c);
}
.stage-name { font-size: 10px;  font-weight: 700; color: #fff; }
.stage-desc { font-size: 8px;   color: rgba(255,255,255,.85); margin-top: 3px; }

/* ── Phase cards ─────────────────────────────────────────────────────────── */
.phase {
  border: 1.5px solid var(--c);
  border-radius: 7px;
  background: #fff;
  padding: 7px 9px;
}
.phase-title { font-size: 9.5px; font-weight: 700; color: var(--c); margin-bottom: 4px; }
.phase-desc  { font-size: 8.5px; color: #334155; line-height: 1.4; }

/* ── Tech / multi-cloud ──────────────────────────────────────────────────── */
.kv-box { background: #fff; border: 1.5px solid #CBD5E1; border-radius: 7px; padding: 7px 10px; }
.kv-box-title { font-size: 11px; font-weight: 700; color: #0F172A; margin-bottom: 6px; }
.kv   { display: flex; gap: 8px; margin: 2.5px 0; font-size: 9px; }
.kv-k { font-weight: 700; min-width: 80px; flex-shrink: 0; }
.kv-v { color: #334155; }
"""


# ── Build HTML ─────────────────────────────────────────────────────────────────

def build_html():
    # Logo: embed as data URI so HTML is fully self-contained
    logo_html = ""
    logo_path = os.path.join(PROJECT_DIR, "AEGIS_Logo.png")
    if os.path.exists(logo_path):
        with open(logo_path, "rb") as f:
            b64 = base64.b64encode(f.read()).decode()
        logo_html = f'<img src="data:image/png;base64,{b64}">'

    # ── Sending sources ──────────────────────────────────────────────────────
    upload = card("React Upload Portal  (PWA)", [
        "Drag-and-drop: files, folders, DICOM directories",
        "Client-side tag de-identification (DICOM PS3.15)",
        "Anonymization preview — before / after tag diff",
        "Multi-study detection with per-study upload cards",
        "Per-project retained-tag anonymization profiles",
        "Auto-retry: 3× exponential backoff per file",
    ], "react")

    dimse = card("DIMSE Receiver  (pynetdicom · GCE VM)", [
        "Compute Engine VM — aegis-prod-dimse-receiver",
        "Static IP 35.232.172.221 · TCP port 11112",
        "C-STORE SCP — receives from PACS systems / scanners",
        "Institution attribution (AE title or IP CIDR)",
        "Calls POST /api/ingest on DICOM association close",
        "Durable retry queue + dead-letter (disk-persistent)",
    ], "py")

    privacy = """
    <div class="privacy">
      <div class="privacy-title">Key Privacy Principle</div>
      <div class="privacy-main">PHI is stripped in the browser<br>BEFORE data leaves the hospital.</div>
      <ul>
        <li>Only tag-de-identified DICOM is transmitted</li>
        <li>HIPAA Safe Harbor — 18 identifier types removed</li>
        <li>Deterministic UID hashing + date shifting</li>
        <li>Zero software install required at sending site</li>
        <li>DIMSE path ingests raw files (internal-only path)</li>
      </ul>
    </div>"""

    # ── Core GCP services ────────────────────────────────────────────────────
    api = card("API Backend  (Go / Cloud Run)", [
        "Upload orchestration (sessions, chunked PUT)",
        "Study / project / institution management",
        "Routing rules engine → auto-pipeline dispatch",
        "DICOMweb proxy  (QIDO-RS + WADO-RS)",
        "Export shares: token-auth, ZIP download",
        "Email digest scheduler + SMTP relay",
        "DIMSE retry control proxy",
        "MCP tool backend + batch import CLI",
        "distroless image — minimal CVE surface",
    ], "go")

    dashboard = card("Admin Dashboard + Weasis DWV  (React / Cloud Run)", [
        "Behind Identity-Aware Proxy (IAP)",
        "Study browser: filter, search, paginate, bulk ops",
        "Weasis DWV — yoked before/after defacing review",
        "7-stage pipeline visualization per study",
        "RBAC: admin (write) + viewer (read-only)",
        "Routing rules, institutions, anon profiles",
        "Protocol templates, API keys, webhooks",
        "Audit log + CSV export, share management",
        "Federation peers, project lifecycle",
    ], "cyan")

    sql = card("Cloud SQL  (PostgreSQL 15)", [
        "Studies, projects, institutions, admin users",
        "Routing rules, audit trail, export shares",
        "Private IP · Secret Manager credentials",
        "Point-in-time recovery (7-day retention)",
    ], "amber")

    gcs = card("Cloud Storage  (GCS)", [
        "dicom/raw/{uid}/   — tag-de-identified",
        "dicom/clean/{uid}/ — defaced + processed",
        "bids/{uid}/        — NIfTI / BIDS output",
        "Shared volume (Go API + all 8 sidecars)",
    ], "gcp")

    # ── Processing sidecars row 1 ────────────────────────────────────────────
    sc_deface = card("Defacing  (Python)", [
        "Head MRI / PET / CT — facial feature removal",
        "mri_reface / DeepDefacer / mri_deface backends",
        "nibabel fallback for dev / testing only",
        "Writes defaced DICOM to clean/ store",
        "SSIM-based QA score per study",
    ], "py")

    sc_phi = card("PHI Detection  (Python)", [
        "Burned-in text OCR on DICOM pixel data",
        "Tesseract (local) / Cloud Vision / Textract",
        "Per-file findings with confidence scores",
        "Per-project confidence threshold config",
        "Informational — non-blocking to pipeline",
    ], "py")

    sc_qc = card("QC Automation  (Python)", [
        "File integrity + required DICOM tag check",
        "Slice consistency (rows / cols / pixel spacing)",
        "SNR estimation (signal mean / corner noise)",
        "Coverage completeness by body part",
        "Missing slice gap detection",
    ], "py")

    sc_cls = card("Classification  (Python)", [
        "Fills modality + body_part from DICOM headers",
        "Heuristic: tags → SOP UID → series description",
        "Cloud Vision / Rekognition image fallback",
        "Re-evaluates routing rules after classify",
        "Confidence threshold gating (≥ 0.5)",
    ], "py")

    # ── Processing sidecars row 2 ────────────────────────────────────────────
    sc_bids = card("BIDS Conversion  (Python)", [
        "DICOM → NIfTI via dcm2niix",
        "BIDS-compliant directory structure",
        "sub-{hash8}/anat | func | dwi | perf | pet/",
        "JSON sidecar metadata per series",
        "Privacy: UID-hashed subject labels",
        "ZIP download via Go API",
    ], "py")

    sc_proto = card("Protocol Check  (Python)", [
        "Verifies TR / TE / flip / thickness vs template",
        "Classic + Enhanced DICOM (multi-frame)",
        "Per-project protocol templates with CRUD",
        "numeric / exact / range / contains_all match",
        "critical / warning / info severity levels",
    ], "py")

    sc_synth = card("Synth MRI  (Python)  ★", [
        "Synthetic brain MRI generation",
        "CPU: Shepp-Logan phantom (numpy + pydicom)",
        "GPU: MONAI BraTS LDM (INCLUDE_MONAI=true)",
        "T1w contrast, Rician noise, face anatomy",
        "Seeded + reproducible, 20 – 200 slices",
        "Imports via normal pipeline (auto-dispatch)",
    ], "py")

    # ── Secondary info boxes ─────────────────────────────────────────────────
    security = info_box("Security Layers", [
        ("", [
            "VPC private subnets + Cloud NAT",
            "Cloud Armor DDoS / WAF",
            "TLS 1.2+ on all endpoints",
            "CMEK (Cloud KMS)",
            "IAM least privilege",
            "Secret Manager (DB creds, API keys)",
            "Cloud Audit Logs",
            "Artifact Registry vuln scanning",
            "distroless containers (Go API)",
            "IAP on all admin routes",
            "No PHI in email / audit entries",
        ]),
    ], "red", "#FFF1F2")

    export = info_box("Export &amp; Sharing", [
        ("Export Portal  (React / public SPA)", [
            "Token-authenticated share links",
            "Study info, expiry countdown (server-anchored)",
            "ZIP download of approved DICOM files",
        ]),
        ("DICOM Forwarding  (route_to)", [
            "DICOMweb STOW-RS destinations",
            "DIMSE C-STORE to remote AE Title",
            "Auto-forwards on study approval",
            "Cross-cloud: GCP↔AWS STOW-RS live",
        ]),
        ("Share Lifecycle", [
            "Extend / revoke anytime",
            "Immutable download log per share",
            "Export analytics dashboard",
        ]),
    ], "green", "#F0FDF4")

    mcp = info_box("MCP Server + AI Agent Tools", [
        ("Model Context Protocol  (Claude / Cursor)", [
            "52+ read tools: studies, audit, stats, routing, health",
            "29+ write tools (confirm:true + reason required)",
            "Zod-validated schemas — read-safe by default",
            "DIMSE retry proxy: process, replay, dead-letter",
        ]),
        ("Agent Orchestrator", [
            "DICOM tag provenance analysis",
            "9 diagnostic tools for study triage",
            "Cohort report, pipeline funnel, project health",
        ]),
        ("Batch Import + Webhooks + API Keys", [
            "aegis-import CLI — bulk historical migration",
            "5 webhook events, HMAC-SHA256 signed payloads",
            "Machine-to-machine bearer tokens (hashed)",
        ]),
    ], "purple", "#F5F3FF")

    landing = card("Landing Page  (nginx / Cloud Run)", [
        "aegisimaging.ai — public marketing site",
        "React + Vite built, served by nginx",
        "Interactive synthetic DICOM demo widget",
        "Schedule Demo / Contact form",
        "Market opportunity: $45B imaging market",
        "LB default backend — no IAP required",
        "www + apex domains on SSL cert v3",
    ], "react")

    # ── Pipeline stages ──────────────────────────────────────────────────────
    stages = [
        ("1. Classify",   "#6D28D9", "fills modality<br>body_part"),
        ("2. PHI Scan",   "#DC2626", "OCR pixels<br>flag burned-in text"),
        ("3. Protocol",   "#C2410C", "TR / TE / flip<br>vs template"),
        ("4. Deface",     "#1D4ED8", "remove face<br>head imaging only"),
        ("5. QC Check",   "#166534", "SNR · coverage<br>slice consistency"),
        ("6. BIDS",       "#065F46", "NIfTI + sidecar<br>BIDS structure"),
        ("7. Export Fwd", "#374151", "STOW-RS or<br>DIMSE forward"),
    ]
    pipe_html = "\n".join(
        f'<div class="stage" style="--c:{c}">'
        f'<div class="stage-name">{n}</div>'
        f'<div class="stage-desc">{d}</div>'
        f"</div>"
        for n, c, d in stages
    )

    # ── Phases ───────────────────────────────────────────────────────────────
    phases = [
        ("Phase 1: Foundation  ✓",
         "Terraform (GCP + AWS) · Go API · Upload Portal · Admin Dashboard · PostgreSQL · CI/CD",
         "#15803D"),
        ("Phase 2: Processing  ✓",
         "Defacing · PHI Detection · QC · BIDS · Classification · Protocol · DIMSE Receiver",
         "#1D4ED8"),
        ("Phase 3: Operations  ✓",
         "Routing rules · Institutions · Audit · Export Portal · Shares · Email · MCP Server",
         "#D97706"),
        ("Phase 4: Production  ✓",
         "GCP live (aegis-prod) · DIMSE on GCE VM · Cloud Build CI/CD · IAM hardening · Landing Page · RBAC",
         "#7C3AED"),
        ("Phase 5: AWS Live  ✓",
         "AWS production live (Feb 25) · GCP→AWS cross-cloud routing verified · Single codebase deploys both clouds in parallel",
         "#DC2626"),
        ("Phase 6: Azure Deploying  →",
         "Azure Container Apps + PostgreSQL Flexible Server + Azure Blob Storage — deploying Feb 26 (day 9) · GitHub Actions OIDC CI/CD · SOC 2 + HL7 FHIR (Q3) · Enterprise GA (Q4)",
         "#0078D4"),
    ]
    phases_html = "\n".join(
        f'<div class="phase" style="--c:{c}">'
        f'<div class="phase-title">{t}</div>'
        f'<div class="phase-desc">{d}</div>'
        f"</div>"
        for t, d, c in phases
    )

    # ── AWS zone ─────────────────────────────────────────────────────────────
    aws_alb = card("ALB + Cognito  (Auth Layer)", [
        "HTTPS listener on ACM certificate",
        "Cognito hosted UI — admin-create-only user pool",
        "authenticate-cognito default action",
        "Public bypass rules: /healthz, upload, export",
    ], "orange")

    aws_api = card("API + Admin  (ECS Fargate)", [
        "Go API — same image as GCP (1 vCPU / 2 GB)",
        "Admin Dashboard — React / nginx (0.5 vCPU / 1 GB)",
        "Service discovery: api.aegis.local",
        "Force-new-deployment via GitHub Actions",
    ], "go")

    aws_sidecars = card("7 Python Sidecars  (ECS Fargate)", [
        "defacing · phi-detection · qc-service",
        "bids-service · classification-service",
        "protocol-service · synth-service",
        "Cloud Map private DNS: svc.aegis.local:8080",
    ], "py")

    aws_data = card("RDS + S3  (Data Layer)", [
        "RDS PostgreSQL 15 — private subnet",
        "Secrets Manager — master credentials",
        "S3 DICOM bucket — versioned, KMS-encrypted",
        "Same STORAGE_MODE=s3 as GCP (same Go code)",
    ], "amber")

    aws_dimse = card("DIMSE EC2  (t3.small)", [
        "Elastic IP — stable for PACS AE title registration",
        "Amazon Linux 2023 — Docker + SSM agent",
        "SSM Parameter Store → image URI on every boot",
        "GitHub Actions: write SSM param + reboot instance",
    ], "py")

    aws_cicd = card("GitHub Actions CI/CD", [
        "Triggers on push to develop — parallel with GCP Cloud Build",
        "Matrix build: 11 services, --platform linux/amd64",
        "Push SHA tag + latest tag to 13 ECR repositories",
        "Force-new-deployment for 10 ECS services",
        "Update SSM param + reboot DIMSE EC2",
    ], "slate")

    # ── Azure zone ───────────────────────────────────────────────────────────
    az_auth = card("Azure Container Registry + Easy Auth", [
        "ACR — private registry, GitHub Actions OIDC push",
        "Easy Auth on Container Apps — injects X-MS-CLIENT-PRINCIPAL-NAME",
        "AUTH_PROVIDER=azure, AUTH_ENABLED=true",
        "Federated OIDC — no long-lived credentials stored",
    ], "azure")

    az_api = card("API + Admin  (Azure Container Apps)", [
        "Go API — same image as GCP/AWS (0.5–1 vCPU / 2 GB)",
        "Admin Dashboard — React / nginx Container App",
        "Azure Communication Services SMTP relay (smtp.azurecomm.net)",
        "Force-new-revision via GitHub Actions on every push",
    ], "go")

    az_sidecars = card("7 Python Sidecars  (Container Apps)", [
        "defacing · phi-detection · qc-service",
        "bids-service · classification-service",
        "protocol-service · synth-service",
        "Internal ingress only — same Docker images as GCP/AWS",
    ], "py")

    az_data = card("PostgreSQL Flex + Azure Blob  (Data Layer)", [
        "Azure Database for PostgreSQL — Flexible Server",
        "Azure Blob Storage — STORAGE_MODE=azure",
        "DefaultAzureCredential — Workload Identity / Managed Identity",
        "SAS tokens via user-delegation key for signed URLs",
    ], "amber")

    az_dimse = card("DIMSE Azure VM  (Standard_B2s)", [
        "Debian 12 — Docker + Azure VM Extensions",
        "Static public IP — TCP port 11112 for PACS registration",
        "GitHub Actions: az vm run-command + docker pull/restart",
        "Same pynetdicom C-STORE SCP as GCP/AWS",
    ], "py")

    az_cicd = card("GitHub Actions CI/CD  (OIDC)", [
        "Triggers on push to develop — parallel with GCP + AWS",
        "Federated OIDC — AZURE_CLIENT_ID / TENANT_ID / SUBSCRIPTION_ID",
        "Build 13 images → push to ACR → az containerapp update",
        "deploy-dimse job: az vm run-command on DIMSE VM",
    ], "slate")

    # ── Tech stack ───────────────────────────────────────────────────────────
    deps = [
        ("Go:",       "suyashkumar/dicom · pgx · testcontainers-go · testify",          "go"),
        ("Browser:",  "dcmjs · dicomParser · Weasis DWV (MIT) · React 19 · Vite",       "react"),
        ("Defacing:", "mri_reface · DeepDefacer · mri_deface · dcm2niix · nibabel",     "py"),
        ("PHI/OCR:",  "pytesseract · Google Cloud Vision · AWS Textract · Pillow",      "py"),
        ("QC/BIDS:",  "pydicom · numpy · dcm2niix · pynetdicom (C-STORE SCP)",          "py"),
        ("Synth:",    "numpy · pydicom · MONAI Generative · torch · huggingface-hub",   "py"),
        ("Infra:",    "Terraform Google + AWS providers · Docker distroless/slim",       "slate"),
    ]

    mcols = [
        ("GCP:",     "Cloud Run · Cloud SQL · GCS · IAP · Cloud Armor · KMS · Pub/Sub",   "gcp"),
        ("AWS:",     "ECS Fargate · RDS · S3 · ALB + Cognito · ACM · Secrets Manager",    "orange"),
        ("Local:",   "Docker Compose · PostgreSQL 15 · Mailpit · local filesystem",        "slate"),
        ("Auth:",    "GCP IAP · Azure AD Easy Auth · AWS ALB+Cognito · dev auto-auth",     "orange"),
        ("Storage:", "STORAGE_MODE=gcs | s3 | local  —  same Go API, no code changes",    "go"),
        ("Azure:",    "Container Apps · PostgreSQL Flex · Azure Blob · ACR · Easy Auth · Azure VM (DIMSE)",    "azure"),
        ("CI/CD:",   "GitHub Actions: Go (160+) · Python (252+) · TS · Docker (9) · Cloud Build auto-deploy", "green"),
        ("Domains:", "aegisimaging.ai · www · api · admin  —  SSL cert v3",               "py"),
    ]

    # ── Assemble ─────────────────────────────────────────────────────────────
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>AEGIS Architecture</title>
<style>{CSS}</style>
</head>
<body>
<div class="page">

  <!-- HEADER -->
  <div class="hdr">
    {logo_html}
    <div>
      <div class="hdr-title">AEGIS Architecture</div>
      <div class="hdr-sub">Anonymization &amp; Exchange Gateway for Imaging Studies</div>
    </div>
  </div>

  <!-- SENDING SOURCES -->
  <div class="zone z-src">
    <div class="zone-hdr">
      SENDING SOURCES
      <span class="zone-sub">External Browser Upload &nbsp;·&nbsp; DICOM Network (C-STORE SCP, port 11112) &nbsp;·&nbsp; Programmatic Batch Ingest</span>
    </div>
    <div class="g3">
      {upload}
      {dimse}
      {privacy}
    </div>
  </div>

  <!-- GCP PROJECT -->
  <div class="zone z-gcp">
    <div class="zone-hdr">
      GCP PROJECT
      <span class="zone-sub">aegis-prod-488120 &nbsp;·&nbsp; us-central1 &nbsp;·&nbsp; Live, February 2026</span>
    </div>

    <!-- Cloud Armor / LB -->
    <div class="lb-bar">
      <strong>Cloud Armor (DDoS / WAF)</strong>
      <span>+ Global HTTPS Load Balancer</span>
      <span>+ Identity-Aware Proxy (admin routes)</span>
      <span>+ Managed SSL cert v3 &nbsp;(api · admin · aegisimaging.ai · www)</span>
      <span>+ Cloud Run &nbsp;(API · Dashboard · Landing · 8 sidecars)</span>
    </div>

    <!-- Core services -->
    <div class="core-grid">
      {api}
      {dashboard}
      <div class="infra-stack">
        {sql}
        {gcs}
      </div>
    </div>

    <!-- Processing sidecars -->
    <div class="sidecar-lbl">Processing Sidecars &nbsp;(Python / Cloud Run) &nbsp;— dispatched async by Go API, runs hands-free via routing rules</div>
    <div class="g4">{sc_deface}{sc_phi}{sc_qc}{sc_cls}</div>
    <div class="row2">{sc_bids}{sc_proto}{sc_synth}</div>

    <!-- Security / Export / MCP / Landing -->
    <div class="g4" style="margin-top:6px">
      {security}
      {export}
      {mcp}
      {landing}
    </div>
  </div>

  <!-- AWS ACCOUNT -->
  <div class="zone" style="background:#FFF7ED;border-color:#EA580C;margin-bottom:6px">
    <div class="zone-hdr" style="color:#EA580C">
      AWS ACCOUNT
      <span class="zone-sub" style="color:#9A3412">301691475234 &nbsp;·&nbsp; us-east-1 &nbsp;·&nbsp; ✓ Live — aws.api.aegisimaging.ai · Cross-cloud routing verified · Feb 25, 2026</span>
    </div>
    <div class="g3" style="margin-bottom:6px">
      {aws_alb}
      {aws_api}
      {aws_sidecars}
    </div>
    <div class="g3">
      {aws_data}
      {aws_dimse}
      {aws_cicd}
    </div>
  </div>

  <!-- AZURE SUBSCRIPTION -->
  <div class="zone z-azure" style="margin-bottom:6px">
    <div class="zone-hdr">
      AZURE SUBSCRIPTION
      <span class="zone-sub" style="color:#005A9E">Container Apps · PostgreSQL Flexible Server · Azure Blob Storage &nbsp;·&nbsp; ● Deploying — Feb 26, 2026 (Day 9) · GitHub Actions OIDC CI/CD</span>
    </div>
    <div class="g3" style="margin-bottom:6px">
      {az_auth}
      {az_api}
      {az_sidecars}
    </div>
    <div class="g3">
      {az_data}
      {az_dimse}
      {az_cicd}
    </div>
  </div>

  <!-- PIPELINE -->
  <div class="zone z-pipe">
    <div class="zone-hdr">
      Automated Processing Pipeline
      <span class="zone-sub">routing rules define which steps are required → pipeline dispatches each step in dependency order, hands-free</span>
    </div>
    <div class="pipe-row">{pipe_html}</div>
  </div>

  <!-- PHASES -->
  <div class="zone z-ph">
    <div class="zone-hdr">Implementation Phases</div>
    <div class="g5" style="grid-template-columns:repeat(6,1fr)">{phases_html}</div>
  </div>

  <!-- TECH STACK + MULTI-CLOUD -->
  <div class="zone z-tech">
    <div class="g2">
      <div class="kv-box">
        <div class="kv-box-title">Key Open-Source Dependencies</div>
        {kv_table([(k, v, c) for k, v, c in deps])}
      </div>
      <div class="kv-box">
        <div class="kv-box-title">Multi-Cloud + Test Coverage</div>
        {kv_table([(k, v, c) for k, v, c in mcols])}
      </div>
    </div>
  </div>

</div>
</body>
</html>"""


# ── Conversion ─────────────────────────────────────────────────────────────────

def find_chrome():
    candidates = [
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
        "/Applications/Chromium.app/Contents/MacOS/Chromium",
        shutil.which("google-chrome"),
        shutil.which("chromium-browser"),
        shutil.which("chromium"),
    ]
    return next((c for c in candidates if c and os.path.exists(c)), None)


def get_page_height(chrome, html_path):
    """Measure document.body.scrollHeight via Chrome headless DOM dump."""
    import re
    import tempfile

    with open(html_path, encoding="utf-8") as f:
        html = f.read()

    # Inject a one-liner that writes the height into the page title
    html2 = html.replace(
        "</body>",
        '<script>document.title="H:"+document.body.scrollHeight;</script></body>',
    )
    tmp = tempfile.NamedTemporaryFile(suffix=".html", delete=False, mode="w", encoding="utf-8")
    tmp.write(html2)
    tmp.close()

    try:
        r = subprocess.run(
            [
                chrome,
                "--headless=new",
                "--disable-gpu",
                "--no-sandbox",
                "--virtual-time-budget=3000",
                "--run-all-compositor-stages-before-draw",
                "--dump-dom",
                f"file://{tmp.name}",
            ],
            capture_output=True,
            text=True,
            timeout=30,
        )
        m = re.search(r"<title>H:(\d+)</title>", r.stdout)
        if m:
            return int(m.group(1))
    except Exception:
        pass
    finally:
        os.unlink(tmp.name)

    return None


def convert(html_path, pdf_path, png_path):
    chrome = find_chrome()

    if chrome:
        file_url = f"file://{html_path}"

        # PDF — @page size is set to 420mm tall; all content fits on one page
        r = subprocess.run(
            [
                chrome,
                "--headless=new",
                "--disable-gpu",
                "--no-sandbox",
                "--run-all-compositor-stages-before-draw",
                "--virtual-time-budget=2000",
                f"--print-to-pdf={pdf_path}",
                "--print-to-pdf-no-header",
                "--no-margins",
                file_url,
            ],
            capture_output=True, text=True, timeout=60,
        )
        if os.path.exists(pdf_path):
            print(f"Saved PDF  → {pdf_path}  (Chrome headless)")
        else:
            print(f"Chrome PDF failed:\n{r.stderr[:400]}")

        # PNG — measure actual page height first, then add a 100px buffer
        page_h = get_page_height(chrome, html_path)
        if page_h:
            png_h = page_h + 100
            print(f"  (detected page height: {page_h}px → using {png_h}px for PNG viewport)")
        else:
            png_h = 1600  # safe fallback

        r2 = subprocess.run(
            [
                chrome,
                "--headless=new",
                "--disable-gpu",
                "--no-sandbox",
                "--virtual-time-budget=2000",
                f"--window-size=1875,{png_h}",
                "--force-device-scale-factor=2",
                f"--screenshot={png_path}",
                file_url,
            ],
            capture_output=True, text=True, timeout=60,
        )
        if os.path.exists(png_path):
            print(f"Saved PNG  → {png_path}  (Chrome headless, 2×)")
        else:
            print(f"Chrome PNG failed:\n{r2.stderr[:400]}")

        if os.path.exists(pdf_path) or os.path.exists(png_path):
            return

    # Fallback: weasyprint (PDF only)
    try:
        from weasyprint import HTML as WP
        WP(filename=html_path).write_pdf(pdf_path)
        print(f"Saved PDF  → {pdf_path}  (weasyprint)")
    except ImportError:
        print("No PDF/PNG converter found.")
        print("  Open the HTML in Chrome and use Cmd+P → Save as PDF.")


# ── Entry point ────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    html     = build_html()
    html_path = os.path.join(PROJECT_DIR, "AEGIS_Architecture_Diagram.html")
    pdf_path  = os.path.join(PROJECT_DIR, "AEGIS_Architecture_Diagram.pdf")
    # Canonical PNG location: served directly by the landing page
    png_path  = os.path.join(PROJECT_DIR, "frontend", "landing", "public", "architecture.png")

    with open(html_path, "w", encoding="utf-8") as f:
        f.write(html)
    print(f"Saved HTML → {html_path}")

    convert(html_path, pdf_path, png_path)
