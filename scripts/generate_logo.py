#!/usr/bin/env python3
"""Generate AEGIS logo as PNG and PDF."""

from PIL import Image, ImageDraw, ImageFont
import math
import os

# Canvas (wider to fit text below shield)
W, H = 1200, 1500
img = Image.new("RGBA", (W, H), (255, 255, 255, 0))
draw = ImageDraw.Draw(img)

# Fonts
def font(size):
    try:
        return ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", size)
    except Exception:
        return ImageFont.truetype("/System/Library/Fonts/SFNSMono.ttf", size)

# Colors
NAVY = "#1A237E"
TEAL = "#00695C"
GOLD = "#F9A825"
WHITE = "#FFFFFF"
DARK_TEAL = "#004D40"

cx = W // 2


def draw_shield(draw, cx, cy, w, h, fill, outline=None, outline_width=0):
    """Draw a shield: rounded top rectangle tapering to a point at bottom."""
    points = []
    r = w * 0.15
    top = cy - h * 0.48
    bottom = cy + h * 0.48
    left = cx - w * 0.5
    right = cx + w * 0.5
    mid_y = cy + h * 0.08

    for angle in range(180, 271, 3):
        rad = math.radians(angle)
        points.append((left + r + r * math.cos(rad), top + r + r * math.sin(rad)))
    points.append((right - r, top))
    for angle in range(270, 361, 3):
        rad = math.radians(angle)
        points.append((right - r + r * math.cos(rad), top + r + r * math.sin(rad)))
    points.append((right, mid_y))
    points.append((cx, bottom))
    points.append((left, mid_y))
    points.append((left, top + r))

    if outline:
        draw.polygon(points, fill=fill, outline=outline, width=outline_width)
    else:
        draw.polygon(points, fill=fill)


# Shield center
shield_cy = 380

# Outer shield
draw_shield(draw, cx, shield_cy, 540, 640, fill=NAVY)

# Middle shield
draw_shield(draw, cx, shield_cy, 500, 600, fill=DARK_TEAL, outline=TEAL, outline_width=5)

# Inner gradient layers
draw_shield(draw, cx, shield_cy - 4, 460, 560, fill="#00796B")
draw_shield(draw, cx, shield_cy - 6, 420, 520, fill="#00897B")

# ── Medical cross (gold) ──
cross_cx, cross_cy = cx, shield_cy - 50
cw, ch = 32, 105
draw.rounded_rectangle(
    (cross_cx - cw, cross_cy - ch, cross_cx + cw, cross_cy + ch),
    radius=16, fill=GOLD
)
draw.rounded_rectangle(
    (cross_cx - ch, cross_cy - cw, cross_cx + ch, cross_cy + cw),
    radius=16, fill=GOLD
)

# Small shield in center of cross (privacy symbol)
draw_shield(draw, cross_cx, cross_cy, 45, 55, fill=WHITE, outline=GOLD, outline_width=3)

# ── Scan arcs above the cross ──
for i, r_val in enumerate([145, 170, 195]):
    bbox = (cross_cx - r_val, cross_cy - r_val, cross_cx + r_val, cross_cy + r_val)
    draw.arc(bbox, start=190 + i * 5, end=350 - i * 5, fill=GOLD, width=3)

# ── Data flow dots (representing encrypted transfer) ──
dot_y_start = shield_cy + 240
for i in range(5):
    dot_r = 6 - i
    alpha_val = 255 - i * 45
    dot_y = dot_y_start + i * 20
    color = (*tuple(int(GOLD.lstrip('#')[j:j+2], 16) for j in (0, 2, 4)), alpha_val)
    draw.ellipse((cx - dot_r, dot_y - dot_r, cx + dot_r, dot_y + dot_r), fill=color)

# ── "AEGIS" text ──
aegis_font = font(140)
text = "AEGIS"
tw = draw.textlength(text, font=aegis_font)
text_y = 770

# Draw each letter with slight letter-spacing
letters = list(text)
spacing = 18
total_w = sum(draw.textlength(l, font=aegis_font) for l in letters) + spacing * (len(letters) - 1)
lx = cx - total_w / 2
for letter in letters:
    lw = draw.textlength(letter, font=aegis_font)
    draw.text((lx, text_y), letter, fill=NAVY, font=aegis_font)
    lx += lw + spacing

# Gold accent bar under AEGIS
bar_w = total_w + 20
draw.rounded_rectangle(
    (cx - bar_w / 2, text_y + 155, cx + bar_w / 2, text_y + 163),
    radius=4, fill=GOLD
)

# ── Subtitle ──
sub_font = font(32)
sub = "Anonymization & Exchange Gateway"
sw = draw.textlength(sub, font=sub_font)
draw.text((cx - sw / 2, text_y + 176), sub, fill=TEAL, font=sub_font)

sub2 = "for Imaging Studies"
sw2 = draw.textlength(sub2, font=sub_font)
draw.text((cx - sw2 / 2, text_y + 218), sub2, fill=TEAL, font=sub_font)

# ── Modality badges ──
badge_y = text_y + 278
badge_font = font(22)

# Primary badge: "All DICOM Modalities"
primary_text = "All DICOM Modalities"
primary_color = "#00695C"
pw = draw.textlength(primary_text, font=badge_font) + 40
px = cx - pw / 2
draw.rounded_rectangle(
    (px, badge_y, px + pw, badge_y + 36),
    radius=18, fill=primary_color
)
ptw = draw.textlength(primary_text, font=badge_font)
draw.text((px + (pw - ptw) / 2, badge_y + 5), primary_text, fill=WHITE, font=badge_font)

# Secondary badges: specific modalities (smaller, muted)
secondary_y = badge_y + 48
secondary_font = font(16)
secondaries = ["MRI", "CT", "PET", "US", "XR", "NM", "MG", "RT"]
secondary_colors = ["#37474F"] * len(secondaries)  # uniform muted dark gray
sec_widths = []
for s in secondaries:
    sec_widths.append(draw.textlength(s, font=secondary_font) + 20)
sec_gap = 8
total_sec_w = sum(sec_widths) + sec_gap * (len(secondaries) - 1)

sx = cx - total_sec_w / 2
for i, (label, sw) in enumerate(zip(secondaries, sec_widths)):
    draw.rounded_rectangle(
        (sx, secondary_y, sx + sw, secondary_y + 28),
        radius=14, fill=secondary_colors[i]
    )
    stw = draw.textlength(label, font=secondary_font)
    draw.text((sx + (sw - stw) / 2, secondary_y + 4), label, fill=WHITE, font=secondary_font)
    sx += sw + sec_gap

# ── Save ──
out_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Trim transparent space
bbox = img.getbbox()
if bbox:
    padded = (max(0, bbox[0] - 40), max(0, bbox[1] - 40),
              min(W, bbox[2] + 40), min(H, bbox[3] + 40))
    img_trimmed = img.crop(padded)
else:
    img_trimmed = img

# PNG (transparent)
png_path = os.path.join(out_dir, "AEGIS_Logo.png")
img_trimmed.save(png_path, "PNG")
print(f"PNG saved to {png_path}")

# PDF (white background)
bg = Image.new("RGB", img_trimmed.size, (255, 255, 255))
bg.paste(img_trimmed, mask=img_trimmed.split()[3])
pdf_path = os.path.join(out_dir, "AEGIS_Logo.pdf")
bg.save(pdf_path, "PDF", resolution=300)
print(f"PDF saved to {pdf_path}")

# Small version for web/GitHub
small = img_trimmed.resize(
    (400, int(400 * img_trimmed.height / img_trimmed.width)),
    Image.LANCZOS
)
small_path = os.path.join(out_dir, "logo-small.png")
small.save(small_path, "PNG")
print(f"Small PNG saved to {small_path}")
