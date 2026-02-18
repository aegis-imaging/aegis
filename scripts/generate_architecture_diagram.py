#!/usr/bin/env python3
"""Generate AEGIS architecture diagram as PNG."""

from PIL import Image, ImageDraw, ImageFont
import math
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)

# Canvas
W, H = 2400, 1800
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
    x0, y0, x1, y1 = xy
    draw.rounded_rectangle(xy, radius=radius, fill=fill, outline=outline, width=width)


def arrow_down(x, y1, y2, label=None):
    draw.line([(x, y1), (x, y2 - 8)], fill=ARROW_COLOR, width=3)
    draw.polygon([(x - 8, y2 - 12), (x + 8, y2 - 12), (x, y2)], fill=ARROW_COLOR)
    if label:
        tw = draw.textlength(label, font=SMALL)
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
    # Title bar
    draw.rounded_rectangle((x0, y0, x1, y0 + 36), radius=12, fill=accent, outline=accent)
    # Fix bottom corners of title bar
    draw.rectangle((x0 + 1, y0 + 24, x1 - 1, y0 + 36), fill=accent)
    tw = draw.textlength(title, font=title_font)
    draw.text((x0 + (x1 - x0 - tw) / 2, y0 + 7), title, fill="#FFFFFF", font=title_font)
    # Items
    for i, item in enumerate(items):
        draw.text((x0 + 16, y0 + 46 + i * 22), f"• {item}", fill=TEXT_MED, font=SMALL)


# ── Title with logo ──
logo_path = os.path.join(PROJECT_DIR, "logo.png")
if os.path.exists(logo_path):
    logo = Image.open(logo_path).convert("RGBA")
    logo_h = 90
    logo_w = int(logo_h * logo.width / logo.height)
    logo = logo.resize((logo_w, logo_h), Image.LANCZOS)
    # Paste logo with transparency onto white background
    logo_x = W // 2 - 200
    logo_y = 5
    img.paste(logo, (logo_x, logo_y), mask=logo.split()[3])
    # Redraw draw object after paste
    draw = ImageDraw.Draw(img)
    text_x = logo_x + logo_w + 15
    draw.text((text_x, 15), "AEGIS Architecture", fill=TEXT_DARK, font=TITLE)
    draw.text((text_x, 55), "Anonymization & Exchange Gateway for Imaging Studies", fill=TEXT_LIGHT, font=HEADING)
else:
    draw.text((W // 2 - 200, 20), "AEGIS Architecture", fill=TEXT_DARK, font=TITLE)
    draw.text((W // 2 - 260, 60), "Anonymization & Exchange Gateway for Imaging Studies", fill=TEXT_LIGHT, font=HEADING)

# ══════════════════════════════════════════════════════
# SENDING SITE (Hospital)
# ══════════════════════════════════════════════════════
hosp_xy = (60, 110, 2340, 380)
rounded_rect(hosp_xy, fill=BG_HOSPITAL, outline=BORDER_HOSPITAL, width=3)
draw.text((80, 120), "SENDING SOURCES  (External Browser + Internal Enterprise)", fill=BORDER_HOSPITAL, font=HEADING)

# Upload Portal
service_box(
    (100, 160, 700, 360),
    "React Upload Portal (PWA)",
    [
        "DICOM file/folder picker",
        "Client-side parsing (dcmjs)",
        "Tag de-identification (PS3.15)",
        "Anonymization preview",
        "Chunked upload to GCS",
    ],
    ACCENT_REACT,
)

# Internal ingress lane callout
rounded_rect((720, 116, 1360, 152), fill="#E3F2FD", outline=BORDER_HOSPITAL, width=1)
draw.text((736, 126), "Internal-enterprise studies can ingest directly into the same GCP pipeline.", fill=BORDER_HOSPITAL, font=SMALL)

# De-id Engine
service_box(
    (740, 160, 1240, 360),
    "De-identification Engine",
    [
        "HIPAA Safe Harbor (18 identifiers)",
        "D/Z/X/U/C action codes per tag",
        "Deterministic UID hashing",
        "Date shifting",
        "Private tag removal",
    ],
    "#7B1FA2",
)

# Client Library
service_box(
    (1280, 160, 1700, 360),
    "Uploader Library (TypeScript)",
    [
        "Reusable npm package",
        "Web Workers for parsing",
        "Embeddable in other apps",
        "Progress tracking",
    ],
    "#00897B",
)

# Key callout
rounded_rect((1740, 160, 2320, 360), fill="#FFF3E0", outline=ACCENT_SECURITY, width=2)
draw.text((1760, 170), "Key Principle", fill=ACCENT_SECURITY, font=BOLD)
draw.text((1760, 200), "PHI is stripped in the browser", fill=TEXT_DARK, font=BODY)
draw.text((1760, 225), "BEFORE data leaves the", fill=TEXT_DARK, font=BODY)
draw.text((1760, 250), "hospital network.", fill=TEXT_DARK, font=BODY)
draw.text((1760, 285), "Only tag-de-identified DICOM", fill=TEXT_MED, font=SMALL)
draw.text((1760, 305), "is transmitted over HTTPS/TLS.", fill=TEXT_MED, font=SMALL)
draw.text((1760, 335), "No software installation required.", fill=TEXT_MED, font=SMALL)

# ── Arrow: Hospital → GCP ──
arrow_down(W // 2, 380, 450, "HTTPS (TLS 1.2+) — De-identified DICOM only")

# ══════════════════════════════════════════════════════
# GCP PROJECT
# ══════════════════════════════════════════════════════
gcp_xy = (60, 450, 2340, 1700)
rounded_rect(gcp_xy, fill=BG_GCP, outline=BORDER_GCP, width=3)
draw.text((80, 460), "GCP PROJECT  (Secured Enterprise Tenancy)", fill=BORDER_GCP, font=HEADING)

# Cloud Armor + LB
rounded_rect((100, 500, 2320, 560), fill="#E3F2FD", outline=ACCENT_GCP_SVC, width=2)
draw.text((120, 515), "Cloud Armor (DDoS / WAF)", fill=ACCENT_GCP_SVC, font=BOLD)
draw.text((520, 518), "+   Global HTTPS Load Balancer", fill=TEXT_MED, font=BODY)
draw.text((900, 518), "+   Identity-Aware Proxy (admin routes)", fill=TEXT_MED, font=BODY)

# Arrow into services
arrow_down(W // 2, 560, 610)

# ── Go API Backend ──
service_box(
    (100, 610, 750, 870),
    "API Backend (Go / Cloud Run)",
    [
        "Upload orchestration (signed URLs)",
        "Study / project management",
        "User auth (OAuth 2.0 / JWT)",
        "Routing rules engine",
        "DICOMweb proxy → Healthcare API",
        "Pub/Sub event handlers",
        "distroless image (~10-20 MB)",
    ],
    ACCENT_GO,
)

# ── Healthcare API ──
service_box(
    (800, 610, 1450, 870),
    "GCP Healthcare API",
    [
        "DICOM Store (DICOMweb)",
        "STOW-RS / WADO-RS / QIDO-RS",
        "Server-side de-id validation",
        "InfoType detection (DLP)",
        "Pub/Sub notifications",
        "BigQuery metadata export",
        "CMEK encryption at rest",
    ],
    ACCENT_GCP_SVC,
)

# ── Cloud Storage ──
service_box(
    (1500, 610, 1950, 800),
    "Cloud Storage",
    [
        "Staging bucket (uploads)",
        "Archive bucket (exports)",
        "Signed URL uploads",
        "Lifecycle policies",
    ],
    "#F9A825",
)

# ── Pub/Sub ──
service_box(
    (2000, 610, 2320, 770),
    "Pub/Sub",
    [
        "Ingest notifications",
        "Defacing triggers",
        "Routing events",
        "Audit events",
    ],
    "#AB47BC",
)

# Arrows between services
arrow_right(750, 800, 740, "STOW/WADO")
arrow_right(750, 1500, 680)
arrow_right(1450, 2000, 680)

# ── Defacing Service ──
service_box(
    (100, 920, 750, 1180),
    "Defacing Service (Python / Cloud Run)",
    [
        "Triggered for head MRI/PET/CT",
        "WADO-RS retrieval from DICOM store",
        "dcm2niix (DICOM → NIfTI)",
        "mri_deface or DeepDefacer",
        "pydicom pixel data injection",
        "STOW-RS back to 'clean' store",
        "~0.5-2 GB container image",
        "2-10 min per volume",
    ],
    ACCENT_PYTHON,
)

# Arrow from API to Defacing
arrow_down(425, 870, 920, "HTTP trigger")

# Arrow from Defacing back to Healthcare API
arrow_right(750, 800, 1050, "Defaced DICOM")

# ── OHIF Viewer / Admin Dashboard ──
service_box(
    (800, 920, 1450, 1180),
    "Admin Dashboard (React / Cloud Run)",
    [
        "Behind Identity-Aware Proxy (IAP)",
        "OHIF Viewer (DICOMweb)",
        "Study browser + QC review",
        "Defacing review (before/after)",
        "Routing rule configuration",
        "User/institution management",
        "Audit log viewer",
    ],
    ACCENT_VIEWER,
)

# ── Two DICOM Stores diagram ──
rounded_rect((1500, 920, 1950, 1100), fill="#E8EAF6", outline="#3F51B5", width=2)
draw.text((1520, 930), "Two DICOM Stores", fill="#3F51B5", font=BOLD)
draw.text((1520, 960), "raw", fill=TEXT_DARK, font=BODY)
draw.text((1620, 960), "Tag de-id'd, not defaced", fill=TEXT_LIGHT, font=SMALL)
draw.text((1520, 990), "clean", fill=TEXT_DARK, font=BODY)
draw.text((1620, 990), "Fully processed + defaced", fill=TEXT_LIGHT, font=SMALL)
draw.text((1520, 1025), "Non-destructive pipeline:", fill=TEXT_MED, font=SMALL)
draw.text((1520, 1045), "raw → deface → clean → approve", fill=TEXT_MED, font=SMALL)
draw.text((1520, 1075), "Admin reviews before release", fill=TEXT_MED, font=SMALL)

# ── Security box ──
rounded_rect((2000, 920, 2320, 1180), fill="#FCE4EC", outline=ACCENT_SECURITY, width=2)
draw.text((2020, 930), "Security Layers", fill=ACCENT_SECURITY, font=BOLD)
items_sec = [
    "VPC Service Controls",
    "Cloud Armor DDoS/WAF",
    "TLS 1.2+ everywhere",
    "CMEK (Cloud KMS)",
    "IAM least privilege",
    "Cloud Audit Logs",
    "Secret Manager",
    "Artifact Registry scanning",
    "distroless containers",
]
for i, item in enumerate(items_sec):
    draw.text((2020, 960 + i * 22), f"• {item}", fill=TEXT_MED, font=SMALL)

# ══════════════════════════════════════════════════════
# Bottom: Repository Structure + Tech Stack
# ══════════════════════════════════════════════════════
repos_y = 1230

rounded_rect((100, repos_y, 1150, repos_y + 220), fill="#F5F5F5", outline="#9E9E9E", width=2)
draw.text((120, repos_y + 10), "Repository Structure (5 Repos)", fill=TEXT_DARK, font=BOLD)

repos = [
    ("aegis-terraform-prj", "GCP project bootstrap, IAM, KMS", "#795548"),
    ("aegis-terraform-infra", "VPC, Cloud Run, Healthcare API, Cloud Armor", "#795548"),
    ("aegis-api", "Go backend — upload, routing, DICOMweb proxy", ACCENT_GO),
    ("aegis-frontend", "React — Upload Portal + Admin Dashboard", ACCENT_REACT),
    ("aegis-client", "TypeScript uploader library (npm package)", "#00897B"),
]
for i, (name, desc, color) in enumerate(repos):
    y = repos_y + 45 + i * 34
    draw.rounded_rectangle((120, y, 135, y + 20), radius=3, fill=color)
    draw.text((145, y), name, fill=TEXT_DARK, font=BODY)
    draw.text((530, y + 2), desc, fill=TEXT_LIGHT, font=SMALL)

# Tech stack
rounded_rect((1200, repos_y, 2320, repos_y + 220), fill="#F5F5F5", outline="#9E9E9E", width=2)
draw.text((1220, repos_y + 10), "Key Open-Source Dependencies", fill=TEXT_DARK, font=BOLD)

deps = [
    ("Go:", "suyashkumar/dicom, GCP Go SDK", ACCENT_GO),
    ("Browser:", "dcmjs, dicomParser, OHIF Viewer", ACCENT_REACT),
    ("Defacing:", "mri_deface, dcm2niix, pydicom, nibabel", ACCENT_PYTHON),
    ("Infra:", "Terraform Google Provider, Cloud Build", "#795548"),
    ("Viewing:", "OHIF Viewer (MIT) — DICOMweb native", ACCENT_VIEWER),
]
for i, (cat, desc, color) in enumerate(deps):
    y = repos_y + 45 + i * 34
    draw.text((1220, y), cat, fill=color, font=BOLD)
    draw.text((1350, y + 2), desc, fill=TEXT_MED, font=SMALL)

# ── Phases ──
phases_y = repos_y + 240
rounded_rect((100, phases_y, 2320, phases_y + 170), fill="#F3E5F5", outline="#7B1FA2", width=2)
draw.text((120, phases_y + 10), "Implementation Phases", fill="#7B1FA2", font=BOLD)

phase_data = [
    ("Phase 1: Foundation MVP", "Terraform + Go API + Upload Portal + Admin Dashboard"),
    ("Phase 2: Defacing Pipeline", "Python Cloud Run sidecar, mri_deface, automated trigger"),
    ("Phase 3: Operations", "Routing rules, institution management, audit logging"),
    ("Phase 4: Advanced", "Burned-in PHI OCR, QC automation, BIDS conversion"),
]

phase_w = 540
for i, (title, desc) in enumerate(phase_data):
    x = 120 + i * (phase_w + 20)
    y = phases_y + 45
    colors = ["#4CAF50", "#2196F3", "#FF9800", "#9C27B0"]
    rounded_rect((x, y, x + phase_w, y + 110), fill="#FFFFFF", outline=colors[i], width=2)
    draw.text((x + 15, y + 10), title, fill=colors[i], font=BOLD)
    # Word-wrap description
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

# Save
out_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "architecture.png")
img.save(out_path, "PNG", quality=95)
print(f"Saved to {out_path}")
