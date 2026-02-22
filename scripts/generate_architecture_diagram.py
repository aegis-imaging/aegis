#!/usr/bin/env python3
"""Generate AEGIS architecture diagram as PNG."""

from PIL import Image, ImageDraw, ImageFont
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)

# ── Canvas ─────────────────────────────────────────────────────────────
W, H = 3000, 2500
img = Image.new("RGB", (W, H), "#F8FAFC")
draw = ImageDraw.Draw(img)


def font(size):
    try:
        return ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", size)
    except Exception:
        return ImageFont.truetype("/System/Library/Fonts/SFNSMono.ttf", size)


TITLE  = font(38)
H1     = font(24)
H2     = font(20)
BODY   = font(17)
SMALL  = font(14)
BOLD   = font(20)
TINY   = font(12)

# ── Color palette ──────────────────────────────────────────────────────
C_BG          = "#F8FAFC"
C_HOSP        = "#EFF6FF"
C_GCP         = "#F0FDF4"
C_WHITE       = "#FFFFFF"
C_HOSP_BORDER = "#1D4ED8"
C_GCP_BORDER  = "#15803D"
C_GO          = "#00ADD8"
C_PYTHON      = "#3776AB"
C_REACT       = "#0EA5E9"
C_GCP_SVC     = "#4285F4"
C_ORANGE      = "#EA580C"
C_PURPLE      = "#7C3AED"
C_AMBER       = "#D97706"
C_CYAN        = "#0891B2"
C_RED         = "#DC2626"
C_GREEN       = "#15803D"
C_EMERALD     = "#059669"
C_ARROW       = "#64748B"
C_TEXT        = "#0F172A"
C_MED         = "#334155"
C_LIGHT       = "#64748B"
C_RULE        = "#CBD5E1"
M             = 75   # margin


# ── Drawing helpers ────────────────────────────────────────────────────
def rrect(xy, fill, outline, width=2, r=12):
    draw.rounded_rectangle(xy, radius=r, fill=fill, outline=outline, width=width)


def arrow_down(x, y1, y2, label=None, c=None):
    c = c or C_ARROW
    draw.line([(x, y1), (x, y2 - 9)], fill=c, width=3)
    draw.polygon([(x - 8, y2 - 14), (x + 8, y2 - 14), (x, y2)], fill=c)
    if label:
        tw = draw.textlength(label, font=TINY)
        draw.text((x - tw / 2, (y1 + y2) // 2 - 9), label, fill=c, font=TINY)


def arrow_right(x1, x2, y, label=None, c=None):
    c = c or C_ARROW
    draw.line([(x1, y), (x2 - 10, y)], fill=c, width=3)
    draw.polygon([(x2 - 14, y - 8), (x2 - 14, y + 8), (x2, y)], fill=c)
    if label:
        draw.text(((x1 + x2) // 2 - 12, y - 20), label, fill=c, font=TINY)


def sbox(xy, title, items, accent, row_h=24):
    """Service box with coloured header strip."""
    x0, y0, x1, y1 = xy
    rrect(xy, fill=C_WHITE, outline=accent, width=2)
    draw.rounded_rectangle((x0, y0, x1, y0 + 40), radius=12, fill=accent, outline=accent)
    draw.rectangle((x0 + 1, y0 + 28, x1 - 1, y0 + 40), fill=accent)
    tw = draw.textlength(title, font=BOLD)
    draw.text((x0 + (x1 - x0 - tw) / 2, y0 + 9), title, fill="#FFFFFF", font=BOLD)
    for i, item in enumerate(items):
        draw.text((x0 + 18, y0 + 52 + i * row_h), f"• {item}", fill=C_MED, font=SMALL)


def info_box(xy, heading, items, accent, bg):
    """Info panel with coloured heading (no filled header bar)."""
    x0, y0, x1, y1 = xy
    rrect(xy, fill=bg, outline=accent, width=2)
    draw.text((x0 + 16, y0 + 12), heading, fill=accent, font=BOLD)
    for i, item in enumerate(items):
        draw.text((x0 + 16, y0 + 48 + i * 24), f"• {item}", fill=C_MED, font=SMALL)


# ══════════════════════════════════════════════════════════════════════
# HEADER
# ══════════════════════════════════════════════════════════════════════
logo_path = os.path.join(PROJECT_DIR, "AEGIS_Logo.png")
if os.path.exists(logo_path):
    logo = Image.open(logo_path).convert("RGBA")
    lh = 90
    lw = int(lh * logo.width / logo.height)
    logo = logo.resize((lw, lh), Image.LANCZOS)
    lx, ly = W // 2 - 260, 8
    img.paste(logo, (lx, ly), mask=logo.split()[3])
    draw = ImageDraw.Draw(img)
    tx = lx + lw + 18
    draw.text((tx, 10), "AEGIS Architecture", fill=C_TEXT, font=TITLE)
    draw.text((tx, 58), "Anonymization & Exchange Gateway for Imaging Studies", fill=C_LIGHT, font=H1)
else:
    draw.text((W // 2 - 240, 16), "AEGIS Architecture", fill=C_TEXT, font=TITLE)
    draw.text((W // 2 - 310, 62), "Anonymization & Exchange Gateway for Imaging Studies", fill=C_LIGHT, font=H1)

draw.line([(M, 112), (W - M, 112)], fill=C_RULE, width=2)


# ══════════════════════════════════════════════════════════════════════
# SENDING SOURCES
# ══════════════════════════════════════════════════════════════════════
SRC_Y0, SRC_Y1 = 124, 395
rrect((M, SRC_Y0, W - M, SRC_Y1), fill=C_HOSP, outline=C_HOSP_BORDER, width=2)
draw.text((M + 22, SRC_Y0 + 13), "SENDING SOURCES", fill=C_HOSP_BORDER, font=BOLD)
draw.text((M + 240, SRC_Y0 + 15),
          "External Browser Upload  ·  DICOM Network (DIMSE C-STORE)  ·  Programmatic Batch Ingest",
          fill=C_LIGHT, font=BODY)

inner = W - 2 * M - 40
bw = (inner - 2 * 18) // 3
bx = M + 20
BY0, BY1 = SRC_Y0 + 52, SRC_Y1 - 16

sbox((bx, BY0, bx + bw, BY1), "React Upload Portal  (PWA)", [
    "Drag-and-drop: files, folders, DICOM dirs",
    "Client-side tag de-identification (PS3.15)",
    "Anonymization preview — before/after diff",
    "Multi-study detection with per-study cards",
    "Per-project retained-tag profiles",
    "Auto-retry: 3× exponential backoff",
], C_REACT)
bx += bw + 18

sbox((bx, BY0, bx + bw, BY1), "DIMSE Receiver  (pynetdicom)", [
    "C-STORE SCP on port 11112",
    "Receives studies from PACS / scanners",
    "Institution attribution (AE title / IP range)",
    "POST /api/ingest on DICOM association close",
    "Durable retry queue + dead-letter state",
    "Exponential backoff, operator control API",
], C_PYTHON)
bx += bw + 18

rrect((bx, BY0, bx + bw, BY1), fill="#FFF7ED", outline=C_ORANGE, width=2)
draw.text((bx + 18, BY0 + 12), "Key Privacy Principle", fill=C_ORANGE, font=BOLD)
draw.text((bx + 18, BY0 + 46), "PHI is stripped in the browser", fill=C_TEXT, font=BODY)
draw.text((bx + 18, BY0 + 70), "BEFORE data leaves the hospital.", fill=C_TEXT, font=BODY)
for i, t in enumerate([
    "• Only tag-de-identified DICOM is transmitted",
    "• HIPAA Safe Harbor: 18 identifier types removed",
    "• Deterministic UID hashing + date shifting",
    "• No software install required at sending site",
    "• DIMSE path ingests raw files (internal only)",
]):
    draw.text((bx + 18, BY0 + 112 + i * 28), t, fill=C_MED, font=SMALL)


# ══════════════════════════════════════════════════════════════════════
# GCP PROJECT
# ══════════════════════════════════════════════════════════════════════
arrow_down(W // 2, SRC_Y1, SRC_Y1 + 48, "HTTPS (TLS 1.2+)  ·  DICOM C-STORE (11112)")

GCP_Y0, GCP_Y1 = SRC_Y1 + 48, 1930
rrect((M, GCP_Y0, W - M, GCP_Y1), fill=C_GCP, outline=C_GCP_BORDER, width=2)
draw.text((M + 22, GCP_Y0 + 12), "GCP PROJECT", fill=C_GCP_BORDER, font=BOLD)
draw.text((M + 186, GCP_Y0 + 14),
          "aegis-prod-488120  ·  us-central1  ·  Live, February 2026",
          fill=C_LIGHT, font=BODY)

# ── Cloud Armor / LB bar ──────────────────────────────────────────────
LB_Y = GCP_Y0 + 52
rrect((M + 20, LB_Y, W - M - 20, LB_Y + 50), fill="#DBEAFE", outline=C_GCP_SVC, width=2)
draw.text((M + 40, LB_Y + 11), "Cloud Armor  (DDoS / WAF)", fill=C_GCP_SVC, font=BOLD)
for col, txt in [
    (340, "+ Global HTTPS Load Balancer"),
    (660, "+ Identity-Aware Proxy (admin routes)"),
    (1070, "+ Managed SSL cert v3  (api · admin · aegisimaging.ai · www)"),
    (1670, "+ Cloud Run  (API · Dashboard · Landing · 8 sidecars)"),
]:
    draw.text((M + col, LB_Y + 14), txt, fill=C_MED, font=BODY)

# ── Core services ─────────────────────────────────────────────────────
arrow_down(W // 2, LB_Y + 50, LB_Y + 92)
CORE_Y0 = LB_Y + 92
CORE_Y1 = CORE_Y0 + 330
IX0, IX1 = M + 20, W - M - 20       # inner left/right
IW = IX1 - IX0                       # 2830

API_W  = int(IW * 0.295)             # ~835
DASH_W = int(IW * 0.305)             # ~864
INFRA_W = IW - API_W - DASH_W - 36  # ~1095, split into SQL+GCS

sbox((IX0, CORE_Y0, IX0 + API_W, CORE_Y1), "API Backend  (Go / Cloud Run)", [
    "Upload orchestration (sessions, chunked PUT)",
    "Study / project / institution management",
    "Routing rules engine → auto-pipeline dispatch",
    "DICOMweb proxy (QIDO-RS + WADO-RS)",
    "Export shares: token-auth, ZIP download",
    "Email digest scheduler + SMTP relay",
    "DIMSE retry control proxy",
    "MCP tool backend, bulk import CLI",
    "distroless image — minimal CVE surface",
], C_GO)

sbox((IX0 + API_W + 18, CORE_Y0, IX0 + API_W + 18 + DASH_W, CORE_Y1),
     "Admin Dashboard + OHIF  (React / Cloud Run)", [
    "Behind Identity-Aware Proxy (IAP)",
    "Study browser: filter, search, paginate, bulk ops",
    "OHIF Viewer — before/after defacing review",
    "7-stage pipeline visualization per study",
    "RBAC: admin (write) + viewer (read-only)",
    "Routing rules, institutions, anon profiles",
    "Protocol templates, API keys, webhooks",
    "Audit log + CSV export, share management",
    "Federation peers, project lifecycle",
], C_CYAN)

# SQL + GCS stacked on the right
INFRA_X = IX0 + API_W + 18 + DASH_W + 18
SQL_H = 148
sbox((INFRA_X, CORE_Y0, INFRA_X + INFRA_W, CORE_Y0 + SQL_H), "Cloud SQL  (PostgreSQL 15)", [
    "Studies, projects, institutions, admin users",
    "Routing rules, audit trail, export shares",
    "Private IP · Secret Manager credentials",
    "Point-in-time recovery (7-day retention)",
], C_AMBER)

GCS_Y0 = CORE_Y0 + SQL_H + 14
sbox((INFRA_X, GCS_Y0, INFRA_X + INFRA_W, CORE_Y1), "Cloud Storage  (GCS)", [
    "dicom/raw/{uid}/   — tag-de-identified",
    "dicom/clean/{uid}/ — defaced + processed",
    "bids/{uid}/        — NIfTI / BIDS output",
    "Shared volume (Go API + all 8 sidecars)",
], C_GCP_SVC)

# Connector arrows
arrow_right(IX0 + API_W, IX0 + API_W + 18, CORE_Y0 + 165)
arrow_right(IX0 + API_W + 18 + DASH_W, INFRA_X, CORE_Y0 + 80, "SQL")
arrow_right(IX0 + API_W + 18 + DASH_W, INFRA_X, GCS_Y0 + 70, "GCS")

# ── Processing Sidecars ───────────────────────────────────────────────
SIDE_LABEL_Y = CORE_Y1 + 22
draw.text((IX0, SIDE_LABEL_Y),
          "Processing Sidecars  (Python / Cloud Run)  —  dispatched async by Go API, runs hands-free via routing rules",
          fill=C_PYTHON, font=BOLD)
arrow_down(IX0 + API_W // 2, CORE_Y1, SIDE_LABEL_Y + 2, c=C_PYTHON)

# Row 1: 4 sidecars
R1_Y0 = SIDE_LABEL_Y + 34
SVC_H  = 220
SVC_W  = (IW - 3 * 14) // 4   # ~697
SVC_G  = 14

row1 = [
    ("Defacing  (Python)", [
        "Head MRI/PET/CT — facial feature removal",
        "mri_reface / DeepDefacer / mri_deface",
        "nibabel fallback (dev / test only)",
        "dcm2niix — DICOM → NIfTI pre-process",
        "Writes defaced DICOM to clean/ store",
        "Defacing QA score (SSIM similarity)",
    ], C_PYTHON),
    ("PHI Detection  (Python)", [
        "Burned-in text OCR on DICOM pixel data",
        "Tesseract (local) / Cloud Vision / Textract",
        "Per-file findings with confidence score",
        "Per-project threshold configuration",
        "Informational — non-blocking to pipeline",
        "Sets phi_scan_status: clean | flagged",
    ], C_PYTHON),
    ("QC Automation  (Python)", [
        "File integrity + required DICOM tag check",
        "Slice consistency (rows/cols/pixel spacing)",
        "SNR estimation (signal mean / corner noise)",
        "Coverage completeness by body part",
        "Missing slice gap detection",
        "Sets qc_status: pass | warn | fail",
    ], C_PYTHON),
    ("Classification  (Python)", [
        "Fills modality + body_part from DICOM headers",
        "Heuristic: tags → SOP UID → series desc",
        "Cloud Vision / Rekognition image fallback",
        "Re-evaluates routing rules after classify",
        "Confidence threshold gating (≥ 0.5)",
        "Sets classification_status: classified",
    ], C_PYTHON),
]

sx = IX0
for title, items, accent in row1:
    sbox((sx, R1_Y0, sx + SVC_W, R1_Y0 + SVC_H), title, items, accent)
    sx += SVC_W + SVC_G

# Row 2: 3 sidecars — centred
R2_Y0 = R1_Y0 + SVC_H + 16
row2 = [
    ("BIDS Conversion  (Python)", [
        "DICOM → NIfTI via dcm2niix",
        "BIDS-compliant directory structure",
        "sub-{hash8}/anat | func | dwi | perf | pet/",
        "JSON sidecar metadata per series",
        "Privacy: UID-hashed subject label",
        "ZIP download via Go API",
    ], C_PYTHON),
    ("Protocol Check  (Python)", [
        "Verifies TR / TE / flip / thickness vs template",
        "Classic + Enhanced DICOM (multi-frame)",
        "Per-project protocol templates with CRUD",
        "numeric / exact / range / contains_all match",
        "critical / warning / info severity levels",
        "Sets protocol_status: compliant | deviations",
    ], C_PYTHON),
    ("Synth MRI  (Python)  ★ new", [
        "Synthetic brain MRI generation",
        "CPU: Shepp-Logan phantom (numpy + pydicom)",
        "GPU: MONAI BraTS LDM  (INCLUDE_MONAI=true)",
        "T1w contrast, Rician noise, face anatomy",
        "Seeded + reproducible, 20 – 200 slices",
        "Imports via normal pipeline (auto-dispatch)",
    ], C_PYTHON),
]

row2_w = len(row2) * SVC_W + (len(row2) - 1) * SVC_G
row2_x = IX0 + (IW - row2_w) // 2
sx = row2_x
for title, items, accent in row2:
    sbox((sx, R2_Y0, sx + SVC_W, R2_Y0 + SVC_H), title, items, accent)
    sx += SVC_W + SVC_G

# ── Secondary info boxes ──────────────────────────────────────────────
SEC_Y0 = R2_Y0 + SVC_H + 22
SEC_Y1 = GCP_Y1 - 22
SEC_W  = (IW - 3 * 18) // 4

sx = IX0

# Security
info_box((sx, SEC_Y0, sx + SEC_W, SEC_Y1), "Security Layers", [
    "VPC private subnets + Cloud NAT",
    "Cloud Armor DDoS / WAF",
    "TLS 1.2+ on all endpoints",
    "CMEK (Cloud KMS)",
    "IAM least privilege",
    "Secret Manager (DB creds, keys)",
    "Cloud Audit Logs",
    "Artifact Registry vuln scanning",
    "distroless containers (Go API)",
    "IAP on all admin routes",
    "No PHI in email / audit entries",
], C_RED, "#FFF1F2")
sx += SEC_W + 18

# Export & Sharing
rrect((sx, SEC_Y0, sx + SEC_W, SEC_Y1), fill="#F0FDF4", outline=C_GCP_BORDER, width=2)
draw.text((sx + 16, SEC_Y0 + 12), "Export & Sharing", fill=C_GCP_BORDER, font=BOLD)
sections = [
    ("Export Portal  (React / public SPA)", [
        "Token-authenticated share links",
        "Study info, expiry countdown",
        "ZIP download of approved DICOM",
    ]),
    ("DICOM Forwarding  (route_to)", [
        "DICOMweb STOW-RS destinations",
        "DIMSE C-STORE to remote AE Title",
        "Auto-forwards on study approval",
    ]),
    ("Share Lifecycle", [
        "Extend / revoke anytime",
        "Immutable download log per share",
        "Export analytics dashboard",
    ]),
]
for i, (h, ls) in enumerate(sections):
    draw.text((sx + 16, SEC_Y0 + 52 + i * 80), h, fill=C_GCP_BORDER, font=BODY)
    for j, l in enumerate(ls):
        draw.text((sx + 16, SEC_Y0 + 76 + i * 80 + j * 24), f"• {l}", fill=C_MED, font=SMALL)
sx += SEC_W + 18

# MCP Server
rrect((sx, SEC_Y0, sx + SEC_W, SEC_Y1), fill="#F5F3FF", outline=C_PURPLE, width=2)
draw.text((sx + 16, SEC_Y0 + 12), "MCP Server + Operator Tooling", fill=C_PURPLE, font=BOLD)
mcp_sections = [
    ("Model Context Protocol  (Claude tools)", [
        "17 read tools: studies, audit, stats, health",
        "13 write tools (confirm:true + reason)",
        "Zod-validated — read-safe by default",
    ]),
    ("Batch Import CLI  (aegis-import)", [
        "Bulk DICOM migration from local dir",
        "Institution provenance, dry-run mode",
        "POST /api/import/batch endpoint",
    ]),
    ("Webhooks + API Keys", [
        "5 events: HMAC-SHA256 signed payloads",
        "Machine-to-machine bearer tokens",
        "Delivery log per subscription",
    ]),
]
for i, (h, ls) in enumerate(mcp_sections):
    draw.text((sx + 16, SEC_Y0 + 52 + i * 80), h, fill=C_PURPLE, font=BODY)
    for j, l in enumerate(ls):
        draw.text((sx + 16, SEC_Y0 + 76 + i * 80 + j * 24), f"• {l}", fill=C_MED, font=SMALL)
sx += SEC_W + 18

# Landing Page
sbox((sx, SEC_Y0, sx + SEC_W, SEC_Y1), "Landing Page  (nginx / Cloud Run)", [
    "aegisimaging.ai — public marketing site",
    "React + Vite built, served by nginx",
    "Interactive synthetic DICOM demo widget",
    "Schedule Demo / Contact form",
    "Market opportunity: $45B imaging market",
    "LB default backend — no IAP",
    "www + apex covered by SSL cert v3",
], C_REACT)


# ══════════════════════════════════════════════════════════════════════
# PIPELINE FLOW
# ══════════════════════════════════════════════════════════════════════
PIPE_Y0 = GCP_Y1 + 28
PIPE_H  = 120
rrect((M, PIPE_Y0, W - M, PIPE_Y0 + PIPE_H), fill="#EFF6FF", outline=C_HOSP_BORDER, width=2)
draw.text((M + 22, PIPE_Y0 + 11),
          "Automated Processing Pipeline  (routing rules define requirements → pipeline runs hands-free)",
          fill=C_HOSP_BORDER, font=BOLD)

stages = [
    ("1. Classify",   "#7C3AED", "fills modality\nbody_part"),
    ("2. PHI Scan",   "#DC2626", "OCR on pixels\nflag burned-in text"),
    ("3. Protocol",   "#C2410C", "TR/TE/flip vs\nproject template"),
    ("4. Deface",     "#1D4ED8", "remove face\n(head imaging only)"),
    ("5. QC Check",   "#166534", "SNR · coverage\nslice consistency"),
    ("6. BIDS",       "#065F46", "NIfTI + sidecar\nBIDS structure"),
    ("7. Export Fwd", "#374151", "STOW-RS or\nDIMSE forward"),
]

aw = W - 2 * M - 40
sw = (aw - 6 * 12) // 7
px = M + 20
for i, (name, color, desc) in enumerate(stages):
    sy0, sy1 = PIPE_Y0 + 38, PIPE_Y0 + PIPE_H - 8
    rrect((px, sy0, px + sw, sy1), fill=color, outline=color, width=0, r=7)
    tw = draw.textlength(name, font=BOLD)
    draw.text((px + (sw - tw) // 2, sy0 + 6), name, fill="#FFFFFF", font=BOLD)
    for j, line in enumerate(desc.split("\n")):
        tw2 = draw.textlength(line, font=TINY)
        draw.text((px + (sw - tw2) // 2, sy0 + 31 + j * 16), line, fill="#E2E8F0", font=TINY)
    if i < len(stages) - 1:
        ax, ay = px + sw + 2, (sy0 + sy1) // 2
        draw.polygon([(ax, ay - 8), (ax, ay + 8), (ax + 10, ay)], fill=C_HOSP_BORDER)
    px += sw + 12


# ══════════════════════════════════════════════════════════════════════
# IMPLEMENTATION PHASES
# ══════════════════════════════════════════════════════════════════════
PH_Y0 = PIPE_Y0 + PIPE_H + 22
PH_H  = 150
rrect((M, PH_Y0, W - M, PH_Y0 + PH_H), fill="#FAF5FF", outline=C_PURPLE, width=2)
draw.text((M + 22, PH_Y0 + 11), "Implementation Phases", fill=C_PURPLE, font=BOLD)

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
     "GCP live (aegis-prod) · Landing Page · Batch import · Synth MRI sidecar (8th) · RBAC",
     "#7C3AED"),
    ("Phase 5: Enterprise  🚧",
     "AEGIS AI Agent · Observability dashboard · Multi-tenant federation · Repo split",
     "#DC2626"),
]

pw  = (W - 2 * M - 40 - 4 * 14) // 5
ppx = M + 20
for title, desc, color in phases:
    rrect((ppx, PH_Y0 + 42, ppx + pw, PH_Y0 + PH_H - 10),
          fill=C_WHITE, outline=color, width=2, r=8)
    draw.text((ppx + 14, PH_Y0 + 52), title, fill=color, font=BOLD)
    words, line, ly = desc.split(" · "), "", PH_Y0 + 80
    for w in words:
        test = f"{line} · {w}" if line else w
        if draw.textlength(test, font=TINY) > pw - 28:
            draw.text((ppx + 14, ly), line, fill=C_MED, font=TINY)
            ly += 17
            line = w
        else:
            line = test
    if line:
        draw.text((ppx + 14, ly), line, fill=C_MED, font=TINY)
    ppx += pw + 14


# ══════════════════════════════════════════════════════════════════════
# TECH STACK + MULTI-CLOUD
# ══════════════════════════════════════════════════════════════════════
ST_Y0 = PH_Y0 + PH_H + 22
ST_H  = 200
MID   = M + (W - 2 * M) // 2

rrect((M, ST_Y0, MID - 12, ST_Y0 + ST_H), fill="#F8FAFC", outline=C_RULE, width=2)
draw.text((M + 22, ST_Y0 + 12), "Key Open-Source Dependencies", fill=C_TEXT, font=BOLD)
deps = [
    ("Go:",       "suyashkumar/dicom · pgx · testcontainers-go · testify",          C_GO),
    ("Browser:",  "dcmjs · dicomParser · OHIF Viewer (MIT) · React 19 · Vite",      C_REACT),
    ("Defacing:", "mri_reface · DeepDefacer · mri_deface · dcm2niix · nibabel",     C_PYTHON),
    ("PHI/OCR:",  "pytesseract · Google Cloud Vision · AWS Textract · Pillow",      C_PYTHON),
    ("QC/BIDS:",  "pydicom · numpy · dcm2niix · pynetdicom (C-STORE SCP)",          C_PYTHON),
    ("Synth:",    "numpy · pydicom · MONAI Generative · torch · huggingface-hub",   C_PYTHON),
    ("Infra:",    "Terraform Google + AWS providers · Docker distroless/slim",       "#795548"),
]
for i, (cat, desc, color) in enumerate(deps):
    y = ST_Y0 + 48 + i * 22
    draw.text((M + 22, y), cat, fill=color, font=BOLD)
    draw.text((M + 138, y + 1), desc, fill=C_MED, font=SMALL)

rrect((MID + 12, ST_Y0, W - M, ST_Y0 + ST_H), fill="#F8FAFC", outline=C_RULE, width=2)
draw.text((MID + 34, ST_Y0 + 12), "Multi-Cloud + Test Coverage", fill=C_TEXT, font=BOLD)
mcols = [
    ("GCP:",      "Cloud Run · Cloud SQL · GCS · IAP · Cloud Armor · KMS · Pub/Sub",   C_GCP_SVC),
    ("AWS:",      "ECS Fargate · RDS · S3 · ALB + Cognito · ACM · Secrets Manager",    "#FF9900"),
    ("Local:",    "Docker Compose · PostgreSQL 15 · Mailpit · local filesystem",        "#546E7A"),
    ("Auth:",     "GCP IAP · Azure AD Easy Auth · AWS ALB+Cognito · dev auto-auth",     C_ORANGE),
    ("Storage:",  "STORAGE_MODE=gcs | s3 | local  —  same Go API, no changes",         C_GO),
    ("CI:",       "Go tests (273+) · Python tests (244+) · TS typecheck · Docker (9)", C_GCP_BORDER),
    ("Domains:",  "aegisimaging.ai · www · api · admin  —  SSL cert v3 provisioning",  C_PYTHON),
]
for i, (cat, desc, color) in enumerate(mcols):
    y = ST_Y0 + 48 + i * 22
    draw.text((MID + 34, y), cat, fill=color, font=BOLD)
    draw.text((MID + 150, y + 1), desc, fill=C_MED, font=SMALL)

# ── Save ──────────────────────────────────────────────────────────────
png_path = os.path.join(PROJECT_DIR, "AEGIS_Architecture_Diagram.png")
pdf_path = os.path.join(PROJECT_DIR, "AEGIS_Architecture_Diagram.pdf")

img.save(png_path, "PNG", quality=95)
print(f"Saved PNG → {png_path}")

# PDF: embed at 200 DPI → 15" × 12.5" landscape (readable in any PDF viewer)
img.save(pdf_path, "PDF", resolution=200.0)
print(f"Saved PDF → {pdf_path}")
