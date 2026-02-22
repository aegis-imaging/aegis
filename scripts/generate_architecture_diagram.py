#!/usr/bin/env python3
"""Generate AEGIS architecture diagram as PNG."""

from PIL import Image, ImageDraw, ImageFont
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)

# Canvas
W, H = 2500, 2300
img = Image.new("RGB", (W, H), "#FFFFFF")
draw = ImageDraw.Draw(img)


# Fonts
def font(size):
    try:
        return ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", size)
    except Exception:
        return ImageFont.truetype("/System/Library/Fonts/SFNSMono.ttf", size)


TITLE = font(32)
HEADING = font(20)
BODY = font(16)
SMALL = font(13)
BOLD = font(18)

# Colors
BG_HOSPITAL = "#E8F4FD"
BG_GCP = "#F0F7EE"
BG_SERVICE = "#FFFFFF"
BORDER_HOSPITAL = "#1976D2"
BORDER_GCP = "#2E7D32"
BORDER_SERVICE = "#546E7A"
ACCENT_GO = "#00ADD8"
ACCENT_PYTHON = "#3776AB"
ACCENT_REACT = "#61DAFB"
ACCENT_GCP_SVC = "#4285F4"
ACCENT_SECURITY = "#E65100"
ACCENT_VIEWER = "#FF6F00"
ARROW_COLOR = "#37474F"
TEXT_DARK = "#212121"
TEXT_MED = "#424242"
TEXT_LIGHT = "#616161"


def rounded_rect(xy, fill, outline, width=2, radius=12):
    draw.rounded_rectangle(xy, radius=radius, fill=fill, outline=outline, width=width)


def arrow_down(x, y1, y2, label=None):
    draw.line([(x, y1), (x, y2 - 8)], fill=ARROW_COLOR, width=3)
    draw.polygon([(x - 8, y2 - 12), (x + 8, y2 - 12), (x, y2)], fill=ARROW_COLOR)
    if label:
        draw.text((x + 12, (y1 + y2) / 2 - 8), label, fill=ARROW_COLOR, font=SMALL)


def arrow_right(x1, x2, y, label=None):
    draw.line([(x1, y), (x2 - 8, y)], fill=ARROW_COLOR, width=3)
    draw.polygon([(x2 - 12, y - 8), (x2 - 12, y + 8), (x2, y)], fill=ARROW_COLOR)
    if label:
        draw.text((x1 + 8, y - 20), label, fill=ARROW_COLOR, font=SMALL)


def arrow_left(x1, x2, y, label=None):
    draw.line([(x1, y), (x2 + 8, y)], fill=ARROW_COLOR, width=3)
    draw.polygon([(x2 + 12, y - 8), (x2 + 12, y + 8), (x2, y)], fill=ARROW_COLOR)


def service_box(xy, title, items, accent, title_font=BOLD):
    x0, y0, x1, y1 = xy
    rounded_rect(xy, fill=BG_SERVICE, outline=accent, width=2)
    draw.rounded_rectangle((x0, y0, x1, y0 + 36), radius=12, fill=accent, outline=accent)
    draw.rectangle((x0 + 1, y0 + 24, x1 - 1, y0 + 36), fill=accent)
    tw = draw.textlength(title, font=title_font)
    draw.text((x0 + (x1 - x0 - tw) / 2, y0 + 7), title, fill="#FFFFFF", font=title_font)
    for i, item in enumerate(items):
        draw.text((x0 + 16, y0 + 46 + i * 22), f"• {item}", fill=TEXT_MED, font=SMALL)


# ── Title with logo ──
logo_path = os.path.join(PROJECT_DIR, "AEGIS_Logo.png")
if os.path.exists(logo_path):
    logo = Image.open(logo_path).convert("RGBA")
    logo_h = 90
    logo_w = int(logo_h * logo.width / logo.height)
    logo = logo.resize((logo_w, logo_h), Image.LANCZOS)
    logo_x = W // 2 - 220
    logo_y = 5
    img.paste(logo, (logo_x, logo_y), mask=logo.split()[3])
    draw = ImageDraw.Draw(img)
    text_x = logo_x + logo_w + 15
    draw.text((text_x, 15), "AEGIS Architecture", fill=TEXT_DARK, font=TITLE)
    draw.text((text_x, 55), "Anonymization & Exchange Gateway for Imaging Studies", fill=TEXT_LIGHT, font=HEADING)
else:
    draw.text((W // 2 - 200, 20), "AEGIS Architecture", fill=TEXT_DARK, font=TITLE)
    draw.text((W // 2 - 270, 60), "Anonymization & Exchange Gateway for Imaging Studies", fill=TEXT_LIGHT, font=HEADING)

# ══════════════════════════════════════════════════════
# SENDING SOURCES
# ══════════════════════════════════════════════════════
hosp_xy = (60, 110, 2430, 390)
rounded_rect(hosp_xy, fill=BG_HOSPITAL, outline=BORDER_HOSPITAL, width=3)
draw.text((80, 120), "SENDING SOURCES  (External Browser Upload + DICOM Network + Enterprise Ingest)", fill=BORDER_HOSPITAL, font=HEADING)

# Upload Portal
service_box(
    (90, 160, 680, 370),
    "React Upload Portal (PWA)",
    [
        "DICOM file/folder picker",
        "Client-side parsing (dcmjs)",
        "Tag de-identification (PS3.15)",
        "Anonymization preview (before/after)",
        "Multi-study detection & upload",
        "Auto-retry (3× exponential backoff)",
    ],
    ACCENT_REACT,
)

# De-id Engine
service_box(
    (710, 160, 1180, 370),
    "De-identification Engine",
    [
        "HIPAA Safe Harbor (18 identifiers)",
        "D/Z/X/U/C action codes per tag",
        "Deterministic UID hashing",
        "Date shifting",
        "Per-project retained-tag profiles",
    ],
    "#7B1FA2",
)

# DIMSE Receiver
service_box(
    (1220, 160, 1720, 370),
    "DIMSE Receiver (pynetdicom)",
    [
        "C-STORE SCP on port 11112",
        "Receives studies from PACS/scanners",
        "Writes raw DICOM to shared storage",
        "Calls POST /api/ingest on assoc close",
        "Exponential retry with dead-letter",
        "Durable state across restarts",
    ],
    ACCENT_PYTHON,
)

# Key callout
rounded_rect((1760, 160, 2400, 370), fill="#FFF3E0", outline=ACCENT_SECURITY, width=2)
draw.text((1780, 170), "Key Principle", fill=ACCENT_SECURITY, font=BOLD)
draw.text((1780, 200), "PHI is stripped in the browser", fill=TEXT_DARK, font=BODY)
draw.text((1780, 225), "BEFORE data leaves the", fill=TEXT_DARK, font=BODY)
draw.text((1780, 250), "hospital network.", fill=TEXT_DARK, font=BODY)
draw.text((1780, 285), "Only tag-de-identified DICOM", fill=TEXT_MED, font=SMALL)
draw.text((1780, 305), "is transmitted over HTTPS/TLS.", fill=TEXT_MED, font=SMALL)
draw.text((1780, 325), "No software install at sending site.", fill=TEXT_MED, font=SMALL)
draw.text((1780, 345), "DIMSE ingests raw (internal path).", fill=TEXT_MED, font=SMALL)

# ── Arrow: Sources → GCP ──
arrow_down(W // 2, 390, 460, "HTTPS (TLS 1.2+) · DICOM C-STORE (11112)")

# ══════════════════════════════════════════════════════
# GCP PROJECT
# ══════════════════════════════════════════════════════
gcp_xy = (60, 460, 2430, 1690)
rounded_rect(gcp_xy, fill=BG_GCP, outline=BORDER_GCP, width=3)
draw.text((80, 470), "GCP PROJECT  aegis-prod-488119 · us-central1  (Live, February 2026)", fill=BORDER_GCP, font=HEADING)

# Cloud Armor + LB
rounded_rect((90, 510, 2400, 570), fill="#E3F2FD", outline=ACCENT_GCP_SVC, width=2)
draw.text((110, 525), "Cloud Armor (DDoS / WAF)", fill=ACCENT_GCP_SVC, font=BOLD)
draw.text((510, 528), "+   Global HTTPS Load Balancer", fill=TEXT_MED, font=BODY)
draw.text((890, 528), "+   Identity-Aware Proxy (admin dashboard routes)", fill=TEXT_MED, font=BODY)
draw.text((1400, 528), "+   Cloud Run (all services)", fill=TEXT_MED, font=BODY)

# Arrow into services
arrow_down(W // 2, 570, 620)

# ── Go API Backend ──
service_box(
    (90, 620, 720, 930),
    "API Backend (Go / Cloud Run)",
    [
        "Upload orchestration (sessions, chunks)",
        "Study / project / institution management",
        "Routing rules engine (auto-pipeline)",
        "DICOMweb proxy (QIDO-RS + WADO-RS)",
        "Bulk actions, CSV export, study notes",
        "Export shares (token-auth, ZIP download)",
        "DIMSE retry control proxy",
        "Email digest scheduler",
        "distroless image (~10-20 MB)",
    ],
    ACCENT_GO,
)

# ── Admin Dashboard + OHIF ──
service_box(
    (760, 620, 1380, 930),
    "Admin Dashboard + OHIF (React / Cloud Run)",
    [
        "Behind Identity-Aware Proxy (IAP)",
        "Study browser — filter, search, paginate",
        "Bulk approve/reject + CSV export",
        "OHIF Viewer (before/after defacing review)",
        "Study diagnostics & routing log panel",
        "Routing rules / destinations config",
        "Institutions, users, projects management",
        "Protocol templates & audit log viewer",
        "Share management with countdown timers",
    ],
    ACCENT_VIEWER,
)

# ── Cloud SQL + Storage ──
service_box(
    (1420, 620, 1940, 780),
    "Cloud SQL (PostgreSQL 15)",
    [
        "Studies, projects, institutions",
        "Routing rules, audit trail",
        "Admin users, shares, digest subs",
        "Private IP, Secret Manager creds",
    ],
    "#F9A825",
)

service_box(
    (1980, 620, 2400, 780),
    "Cloud Storage (GCS)",
    [
        "dicom/raw/{studyUID}/ — tag-de-id'd",
        "dicom/clean/{studyUID}/ — defaced",
        "bids/{studyUID}/ — NIfTI/BIDS output",
        "Signed URL uploads from browser",
    ],
    ACCENT_GCP_SVC,
)

# Two DICOM stores note
rounded_rect((1420, 810, 2400, 930), fill="#E8EAF6", outline="#3F51B5", width=2)
draw.text((1440, 820), "Two Storage Paths (non-destructive pipeline)", fill="#3F51B5", font=BOLD)
draw.text((1440, 850), "raw   →  tag de-id'd only (upload portal / DIMSE input)", fill=TEXT_MED, font=SMALL)
draw.text((1440, 872), "clean →  fully processed + defaced (OHIF default view)", fill=TEXT_MED, font=SMALL)
draw.text((1440, 894), "Admin reviews raw vs clean side-by-side before approving", fill=TEXT_LIGHT, font=SMALL)

# Arrows between Go API and dependencies
arrow_right(720, 760, 760, "HTTP")
arrow_right(720, 1420, 700)
arrow_right(720, 1980, 680)

# ── Processing Sidecars row ──
sidecar_y = 970
sidecar_label_y = sidecar_y - 30
draw.text((90, sidecar_label_y), "Processing Sidecars (Python / Cloud Run)  — all triggered async by Go API via HTTP", fill=ACCENT_PYTHON, font=BOLD)

svc_w = 370
svc_h = 220
svc_gap = 15
svc_x = 90

# Defacing
service_box(
    (svc_x, sidecar_y, svc_x + svc_w, sidecar_y + svc_h),
    "Defacing (Python)",
    [
        "Head MRI/PET/CT — face removal",
        "mri_reface / DeepDefacer / mri_deface",
        "nibabel fallback (dev/test)",
        "dcm2niix (DICOM → NIfTI)",
        "Writes defaced DICOM to clean/",
        "~0.5-4 GB image, 1-10 min/vol",
    ],
    ACCENT_PYTHON,
)
svc_x += svc_w + svc_gap

# PHI Detection
service_box(
    (svc_x, sidecar_y, svc_x + svc_w, sidecar_y + svc_h),
    "PHI Detection (Python)",
    [
        "Burned-in text OCR on pixel data",
        "Tesseract (local) / Cloud Vision /",
        "  AWS Textract (cloud backends)",
        "Per-file findings with confidence",
        "Informational — non-blocking",
        "Sets phi_scan_status: clean|flagged",
    ],
    ACCENT_PYTHON,
)
svc_x += svc_w + svc_gap

# QC Service
service_box(
    (svc_x, sidecar_y, svc_x + svc_w, sidecar_y + svc_h),
    "QC Automation (Python)",
    [
        "File integrity check",
        "Slice consistency (dims/spacing)",
        "SNR estimation (signal/noise)",
        "Coverage completeness by body part",
        "Missing slice gap detection",
        "Sets qc_status: pass|warn|fail",
    ],
    ACCENT_PYTHON,
)
svc_x += svc_w + svc_gap

# Classification
service_box(
    (svc_x, sidecar_y, svc_x + svc_w, sidecar_y + svc_h),
    "Classification (Python)",
    [
        "Fills modality + body_part from DICOM",
        "Heuristic: tags → SOP UID → desc",
        "Cloud Vision / AWS Rekognition fallback",
        "Re-evaluates routing rules after classify",
        "Confidence-threshold gating (≥0.5)",
        "Sets classification_status: classified",
    ],
    ACCENT_PYTHON,
)
svc_x += svc_w + svc_gap

# BIDS Service
service_box(
    (svc_x, sidecar_y, svc_x + svc_w, sidecar_y + svc_h),
    "BIDS Conversion (Python)",
    [
        "DICOM → NIfTI (dcm2niix)",
        "BIDS directory structure",
        "sub-{hash8}/anat|func|dwi/",
        "JSON sidecar metadata",
        "Privacy: UID-hashed subject label",
        "ZIP download via Go API",
    ],
    ACCENT_PYTHON,
)
svc_x += svc_w + svc_gap

# Protocol Service
service_box(
    (svc_x, sidecar_y, svc_x + svc_w, sidecar_y + svc_h),
    "Protocol Check (Python)",
    [
        "Verifies TR/TE/flip/slice vs template",
        "Classic + Enhanced DICOM support",
        "Per-project protocol templates",
        "numeric/exact/range/contains match",
        "critical/warning/info severity",
        "Sets protocol_status: compliant|deviations",
    ],
    ACCENT_PYTHON,
)

# Arrow from Go API down to sidecars
arrow_down(405, 930, sidecar_y, "HTTP trigger")

# ── Security box ──
rounded_rect((90, 1210, 510, 1480), fill="#FCE4EC", outline=ACCENT_SECURITY, width=2)
draw.text((110, 1220), "Security Layers", fill=ACCENT_SECURITY, font=BOLD)
items_sec = [
    "VPC (private subnets + Cloud NAT)",
    "Cloud Armor DDoS/WAF",
    "TLS 1.2+ everywhere",
    "CMEK (Cloud KMS)",
    "IAM least privilege",
    "Cloud Audit Logs",
    "Secret Manager",
    "Artifact Registry scanning",
    "distroless containers",
    "IAP for admin routes",
    "No PHI in email/audit",
    "HIPAA-compliant pipeline",
]
for i, item in enumerate(items_sec):
    draw.text((110, 1252 + i * 20), f"• {item}", fill=TEXT_MED, font=SMALL)

# ── Export / Recipients box ──
rounded_rect((560, 1210, 1380, 1480), fill="#E8F5E9", outline=BORDER_GCP, width=2)
draw.text((580, 1220), "Export & Sharing", fill=BORDER_GCP, font=BOLD)
draw.text((580, 1252), "Export Portal (React / Vercel)", fill=BORDER_GCP, font=BODY)
draw.text((580, 1275), "• Token-authenticated share links", fill=TEXT_MED, font=SMALL)
draw.text((580, 1295), "• Modality/body-part badges, study info", fill=TEXT_MED, font=SMALL)
draw.text((580, 1315), "• Expiry countdown (server-anchored clock)", fill=TEXT_MED, font=SMALL)
draw.text((580, 1335), "• ZIP download of approved DICOM files", fill=TEXT_MED, font=SMALL)
draw.text((580, 1365), "Automated DICOM Forwarding", fill=BORDER_GCP, font=BODY)
draw.text((580, 1388), "• route_to routing action", fill=TEXT_MED, font=SMALL)
draw.text((580, 1408), "• DICOMweb (STOW-RS) destinations", fill=TEXT_MED, font=SMALL)
draw.text((580, 1428), "• DIMSE C-STORE to remote AE Title", fill=TEXT_MED, font=SMALL)
draw.text((580, 1448), "• Auto-forwards on approval", fill=TEXT_MED, font=SMALL)

# ── MCP Server box ──
rounded_rect((1420, 1210, 2400, 1480), fill="#EDE7F6", outline="#7B1FA2", width=2)
draw.text((1440, 1220), "MCP Server + Operator Tooling", fill="#7B1FA2", font=BOLD)
draw.text((1440, 1252), "Model Context Protocol (Claude integration)", fill="#7B1FA2", font=BODY)
draw.text((1440, 1275), "• Read tools: list_studies, get_study, list_audit, get_diagnostics", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1295), "• Write tools (MCP_ENABLE_WRITE_TOOLS=true): approve/reject", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1315), "• Readonly mode by default — safe for AI-assisted triage", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1335), "• DIMSE retry proxy: process, replay, clear dead-letter", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1365), "Batch Import CLI (aegis-import)", fill="#7B1FA2", font=BODY)
draw.text((1440, 1388), "• Bulk historical DICOM migration from local dir", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1408), "• POST /api/import/batch — institution-linked provenance", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1428), "• Dry-run mode, duplicate rejection, routing evaluation", fill=TEXT_MED, font=SMALL)
draw.text((1440, 1448), "• Internal ingest: /api/ingest with IP-based institution auto-match", fill=TEXT_MED, font=SMALL)

# ══════════════════════════════════════════════════════
# Bottom: Pipeline Flow + Tech Stack
# ══════════════════════════════════════════════════════
pipeline_y = 1720

# Pipeline visualization
rounded_rect((60, pipeline_y, 2430, pipeline_y + 130), fill="#E3F2FD", outline=ACCENT_GCP_SVC, width=2)
draw.text((80, pipeline_y + 10), "Automated Processing Pipeline  (triggered by routing rules, runs hands-free)", fill=ACCENT_GCP_SVC, font=BOLD)

stages = [
    ("Classification", "#9C27B0", "fills modality/body_part\nre-evaluates rules"),
    ("PHI Scan", "#E53935", "OCR on pixels\nflags burned-in text"),
    ("Protocol Check", "#F57C00", "validates TR/TE/flip\nvs project template"),
    ("Defacing", "#1565C0", "removes facial features\nhead imaging only"),
    ("QC Check", "#2E7D32", "SNR, coverage,\nslice consistency"),
    ("BIDS Convert", "#00695C", "NIfTI + sidecar JSON\nBIDS structure"),
    ("Export Forward", "#37474F", "STOW-RS or DIMSE\nroute_to destinations"),
]

stage_w = 330
stage_x = 80
arrow_x = stage_x + stage_w
for i, (name, color, desc) in enumerate(stages):
    rounded_rect((stage_x, pipeline_y + 40, stage_x + stage_w, pipeline_y + 120),
                 fill=color, outline=color, width=2, radius=8)
    tw = draw.textlength(name, font=BOLD)
    draw.text((stage_x + (stage_w - tw) // 2, pipeline_y + 48), name, fill="#FFFFFF", font=BOLD)
    for j, line in enumerate(desc.split("\n")):
        tw2 = draw.textlength(line, font=SMALL)
        draw.text((stage_x + (stage_w - tw2) // 2, pipeline_y + 75 + j * 18), line, fill="#FFFFFF", font=SMALL)
    if i < len(stages) - 1:
        ax = stage_x + stage_w + 2
        ay = pipeline_y + 80
        draw.polygon([(ax, ay - 8), (ax, ay + 8), (ax + 15, ay)], fill=ACCENT_GCP_SVC)
    stage_x += stage_w + 18

# Phases
phases_y = pipeline_y + 155

rounded_rect((60, phases_y, 2430, phases_y + 170), fill="#F3E5F5", outline="#7B1FA2", width=2)
draw.text((80, phases_y + 10), "Implementation Phases (all complete as of February 2026)", fill="#7B1FA2", font=BOLD)

phase_data = [
    ("Phase 1: Foundation  ✓", "Terraform + Go API + Upload Portal + Admin Dashboard + PostgreSQL + CI", "#4CAF50"),
    ("Phase 2: Processing  ✓", "Defacing + PHI Detection + QC + BIDS + Classification + Protocol + DIMSE", "#2196F3"),
    ("Phase 3: Operations  ✓", "Routing rules + Institutions + Audit + Export portal + Shares + Email digest", "#FF9800"),
    ("Phase 4: Production  ✓", "GCP Cloud Run deploy + Terraform infra + MCP server + Batch import + GCP live", "#9C27B0"),
]

phase_w = 560
for i, (title, desc, color) in enumerate(phase_data):
    x = 80 + i * (phase_w + 20)
    y = phases_y + 45
    rounded_rect((x, y, x + phase_w, y + 110), fill="#FFFFFF", outline=color, width=2)
    draw.text((x + 15, y + 10), title, fill=color, font=BOLD)
    words = desc.split(", ")
    line = ""
    ly = y + 38
    for w in words:
        test = f"{line}, {w}" if line else w
        if draw.textlength(test, font=SMALL) > phase_w - 30:
            draw.text((x + 15, ly), line, fill=TEXT_MED, font=SMALL)
            ly += 20
            line = w
        else:
            line = test
    if line:
        draw.text((x + 15, ly), line, fill=TEXT_MED, font=SMALL)

# Tech stack + repos
stack_y = phases_y + 180

rounded_rect((60, stack_y, 1240, stack_y + 200), fill="#F5F5F5", outline="#9E9E9E", width=2)
draw.text((80, stack_y + 10), "Key Open-Source Dependencies", fill=TEXT_DARK, font=BOLD)
deps = [
    ("Go:", "suyashkumar/dicom, pgx, testcontainers-go, testify", ACCENT_GO),
    ("Browser:", "dcmjs, dicomParser, OHIF Viewer (MIT), React 19, Vite", ACCENT_REACT),
    ("Defacing:", "mri_reface, DeepDefacer, mri_deface, dcm2niix, pydicom, nibabel", ACCENT_PYTHON),
    ("PHI / OCR:", "Tesseract, pytesseract, Google Cloud Vision, AWS Textract", ACCENT_PYTHON),
    ("QC / BIDS:", "pydicom, numpy, dcm2niix, pynetdicom (DIMSE C-STORE SCP)", ACCENT_PYTHON),
    ("Infra:", "Terraform Google + AWS providers, Docker distroless/slim", "#795548"),
]
for i, (cat, desc, color) in enumerate(deps):
    y = stack_y + 45 + i * 28
    draw.text((80, y), cat, fill=color, font=BOLD)
    draw.text((220, y + 2), desc, fill=TEXT_MED, font=SMALL)

rounded_rect((1280, stack_y, 2430, stack_y + 200), fill="#F5F5F5", outline="#9E9E9E", width=2)
draw.text((1300, stack_y + 10), "Multi-Cloud Support", fill=TEXT_DARK, font=BOLD)
multicloud = [
    ("GCP:", "Cloud Run · Cloud SQL · GCS · IAP · Cloud Armor · Cloud KMS", ACCENT_GCP_SVC),
    ("AWS:", "ECS Fargate · RDS · S3 · ALB + Cognito · ACM · Secrets Manager", "#FF9900"),
    ("Local Dev:", "Docker Compose · PostgreSQL 15 · Mailpit · local filesystem", "#546E7A"),
    ("Auth:", "GCP IAP · Azure AD Easy Auth · AWS ALB+Cognito · dev auto-auth", "#E65100"),
    ("Storage:", "STORAGE_MODE=gcs|s3|local — same Go API code, no changes", ACCENT_GO),
    ("CI:", "GitHub Actions — Go tests (120+), Python tests (206+), TS, Docker", "#2E7D32"),
]
for i, (cat, desc, color) in enumerate(multicloud):
    y = stack_y + 45 + i * 28
    draw.text((1300, y), cat, fill=color, font=BOLD)
    draw.text((1440, y + 2), desc, fill=TEXT_MED, font=SMALL)

# Save
out_path = os.path.join(PROJECT_DIR, "AEGIS_Architecture_Diagram.png")
img.save(out_path, "PNG", quality=95)
print(f"Saved to {out_path}")
