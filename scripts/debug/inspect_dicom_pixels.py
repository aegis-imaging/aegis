#!/usr/bin/env python3
"""
Inspect DICOM pixel data from a study directory.
Useful for debugging OHIF black-screen issues.

Usage:
    python3 scripts/debug/inspect_dicom_pixels.py /path/to/dicom/dir [--save-preview preview.png]

Example (local Docker data dir):
    python3 scripts/debug/inspect_dicom_pixels.py \
        data/dicom/clean/<study-uid> --save-preview /tmp/preview.png
"""

import argparse
import os
import sys
from pathlib import Path

import numpy as np
import pydicom


def inspect_dir(dicom_dir: str, save_path: str | None = None, max_files: int = 20) -> None:
    dcm_files = sorted(Path(dicom_dir).glob("*.dcm"), key=lambda p: int(p.stem) if p.stem.isdigit() else 0)
    if not dcm_files:
        print(f"No .dcm files found in {dicom_dir}")
        sys.exit(1)

    print(f"Found {len(dcm_files)} .dcm files in {dicom_dir}\n")

    # Print per-file summary
    print(f"{'File':<12} {'InstanceNum':>12} {'min':>6} {'max':>6} {'mean':>8} {'non-zero%':>10}  WW    WC    BitsStored")
    print("-" * 90)
    for dcm_path in dcm_files[:max_files]:
        try:
            ds = pydicom.dcmread(str(dcm_path))
            arr = ds.pixel_array
            ww = getattr(ds, "WindowWidth", "?")
            wc = getattr(ds, "WindowCenter", "?")
            bs = getattr(ds, "BitsStored", "?")
            nonzero_pct = 100.0 * np.count_nonzero(arr) / arr.size if arr.size > 0 else 0
            print(f"{dcm_path.name:<12} {getattr(ds,'InstanceNumber','?'):>12} {arr.min():>6} {arr.max():>6} {arr.mean():>8.1f} {nonzero_pct:>9.1f}%  {ww:<6}{wc:<6}{bs}")
        except Exception as e:
            print(f"{dcm_path.name:<12}  ERROR: {e}")

    # Simulate OHIF rendering for the first non-trivial file
    print("\n=== OHIF rendering simulation (WW/WC from DICOM tags) ===")
    for dcm_path in dcm_files:
        try:
            ds = pydicom.dcmread(str(dcm_path))
            arr = ds.pixel_array.astype(float)
            ww = float(getattr(ds, "WindowWidth", 0))
            wc = float(getattr(ds, "WindowCenter", 0))
            if ww > 0 and arr.max() > 0:
                lower = wc - ww / 2
                upper = wc + ww / 2
                rendered = np.clip((arr - lower) / ww, 0, 1)
                print(f"  {dcm_path.name}: rendered mean={rendered.mean()*100:.1f}%  max={rendered.max()*100:.1f}%")
                if rendered.mean() < 0.05:
                    print(f"    *** IMAGE WILL APPEAR NEARLY BLACK IN OHIF (mean brightness {rendered.mean()*100:.1f}%) ***")
                    print(f"    Suggested fix: WW={int(arr.max() - arr.min())}, WC={int(arr.mean())}")
                break
        except Exception:
            pass

    # Optionally save a preview PNG
    if save_path:
        try:
            from PIL import Image
            rows = 1
            cols = min(len(dcm_files), 6)
            W, H = 256, 256
            canvas = Image.new("L", (W * cols + 10 * (cols - 1), H + 30), 20)
            for j, dcm_path in enumerate(dcm_files[:cols]):
                ds = pydicom.dcmread(str(dcm_path))
                arr = ds.pixel_array.astype(np.float32)
                if arr.max() > 0:
                    arr_norm = (arr / arr.max() * 255).astype(np.uint8)
                else:
                    arr_norm = arr.astype(np.uint8)
                img = Image.fromarray(arr_norm, mode="L").resize((W, H))
                x = j * (W + 10)
                canvas.paste(img, (x, 30))
                from PIL import ImageDraw
                draw = ImageDraw.Draw(canvas)
                draw.text((x + 4, 4), f"max={ds.pixel_array.max()}", fill=200)
            canvas.save(save_path)
            print(f"\nPreview saved: {save_path}")
        except ImportError:
            print("\nPillow not installed — skipping preview generation (pip install pillow)")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Inspect DICOM pixel values and OHIF rendering")
    parser.add_argument("dicom_dir", help="Directory containing .dcm files")
    parser.add_argument("--save-preview", metavar="PATH", help="Save PNG preview to this path")
    parser.add_argument("--max-files", type=int, default=20, help="Max files to summarize (default: 20)")
    args = parser.parse_args()
    inspect_dir(args.dicom_dir, save_path=args.save_preview, max_files=args.max_files)
