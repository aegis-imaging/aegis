#!/usr/bin/env python3
"""Download real brain MRI studies from TCIA (The Cancer Imaging Archive).

Fetches public DICOM series from the NBIA REST API — no account or token required
for public collections. Downloads ZIP archives and extracts them into a flat
per-series directory ready for the AEGIS batch importer:

    python3 scripts/tcia_download.py --count 3 --output /tmp/tcia-brain

Then import into AEGIS:

    cd api && go run ./cmd/import --dir /tmp/tcia-brain --project default

Collections used (all freely available, Creative Commons licensing):
  - TCGA-GBM: glioblastoma brain MRIs (multi-center, T1/T2/FLAIR)
  - TCGA-LGG: lower grade glioma brain MRIs

API docs: https://wiki.cancerimagingarchive.net/display/Public/NBIA+Search+REST+API+Guide
"""

import argparse
import io
import json
import os
import sys
import time
import zipfile
from pathlib import Path

try:
    import requests
except ImportError:
    sys.exit("requests not installed — run: pip install requests")

BASE_URL = "https://services.cancerimagingarchive.net/nbia-api/services/v1"

# Collections to draw from, in priority order.
# Both are public brain MRI collections with good 3D T1w volumes.
DEFAULT_COLLECTIONS = ["TCGA-GBM", "TCGA-LGG"]

# Only pull 3D volumetric series (skip 2D scout/localizer scans with few slices)
MIN_INSTANCE_COUNT = 20

# Polite delay between download requests so we don't hammer TCIA
DOWNLOAD_DELAY_SECONDS = 1.0


def get_series(collection: str, modality: str = "MR") -> list[dict]:
    """Return all series for a collection/modality from the NBIA search API."""
    url = f"{BASE_URL}/getSeries"
    params = {"Collection": collection, "Modality": modality, "format": "json"}
    resp = requests.get(url, params=params, timeout=30)
    resp.raise_for_status()
    return resp.json()


def download_series_zip(series_uid: str) -> bytes:
    """Download all DICOM files for a series as a ZIP archive (in memory)."""
    url = f"{BASE_URL}/getImage"
    params = {"SeriesInstanceUID": series_uid}
    resp = requests.get(url, params=params, timeout=300, stream=True)
    resp.raise_for_status()
    buf = io.BytesIO()
    for chunk in resp.iter_content(chunk_size=65536):
        buf.write(chunk)
    buf.seek(0)
    return buf.read()


def extract_zip_flat(zip_bytes: bytes, dest_dir: Path) -> int:
    """Extract a TCIA ZIP to dest_dir, flattening any subdirectories.

    TCIA ZIPs sometimes nest files under a series subdirectory.  We flatten
    everything into dest_dir so the AEGIS importer can find all .dcm files
    with a simple directory scan.

    Returns the number of DICOM files extracted.
    """
    dest_dir.mkdir(parents=True, exist_ok=True)
    count = 0
    with zipfile.ZipFile(io.BytesIO(zip_bytes)) as zf:
        for member in zf.infolist():
            if member.is_dir():
                continue
            name = Path(member.filename).name  # strip any directory prefix
            if not name:
                continue
            target = dest_dir / name
            target.write_bytes(zf.read(member.filename))
            count += 1
    return count


def pick_series(
    collections: list[str],
    count: int,
    min_slices: int,
) -> list[dict]:
    """Collect up to `count` 3D MR series across the given collections."""
    candidates: list[dict] = []
    for collection in collections:
        print(f"  Querying {collection}...", flush=True)
        try:
            series_list = get_series(collection)
        except Exception as exc:
            print(f"  WARNING: could not fetch {collection}: {exc}", file=sys.stderr)
            continue
        # Filter to 3D volumetric series
        volumetric = [
            s for s in series_list
            if int(s.get("ImageCount", 0)) >= min_slices
        ]
        print(
            f"  {collection}: {len(series_list)} total series, "
            f"{len(volumetric)} with >= {min_slices} slices"
        )
        candidates.extend(volumetric)
        if len(candidates) >= count:
            break

    # Stable sort by ImageCount descending so we pick richer studies first
    candidates.sort(key=lambda s: int(s.get("ImageCount", 0)), reverse=True)
    return candidates[:count]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument(
        "--count", type=int, default=3,
        help="Number of series to download (default: 3)",
    )
    parser.add_argument(
        "--output", default="test-data/tcia-brain",
        help="Output directory (default: test-data/tcia-brain)",
    )
    parser.add_argument(
        "--collection", action="append", dest="collections",
        metavar="NAME",
        help="TCIA collection to draw from (repeatable; default: TCGA-GBM then TCGA-LGG)",
    )
    parser.add_argument(
        "--min-slices", type=int, default=MIN_INSTANCE_COUNT,
        help=f"Minimum instance count to include a series (default: {MIN_INSTANCE_COUNT})",
    )
    parser.add_argument(
        "--list-only", action="store_true",
        help="Print available series without downloading",
    )
    args = parser.parse_args()

    collections = args.collections or DEFAULT_COLLECTIONS
    output_root = Path(args.output)

    print(f"TCIA Brain MRI downloader")
    print(f"  Collections : {', '.join(collections)}")
    print(f"  Target count: {args.count}")
    print(f"  Min slices  : {args.min_slices}")
    print(f"  Output dir  : {output_root}")
    print()

    print("Fetching series catalogue...")
    series_list = pick_series(collections, args.count, args.min_slices)

    if not series_list:
        sys.exit("No matching series found — try a different collection or lower --min-slices")

    if args.list_only:
        print(f"\nAvailable series ({len(series_list)}):")
        for s in series_list:
            print(
                f"  [{s.get('Modality','?')}] {s.get('SeriesInstanceUID','')} "
                f"  slices={s.get('ImageCount','?')}  "
                f"  desc={s.get('SeriesDescription','(none)')}  "
                f"  collection={s.get('Collection','?')}"
            )
        return

    print(f"\nDownloading {len(series_list)} series to {output_root}:")
    total_files = 0
    for i, series in enumerate(series_list, 1):
        uid = series.get("SeriesInstanceUID", "")
        slices = series.get("ImageCount", "?")
        desc = series.get("SeriesDescription") or series.get("Modality", "MR")
        collection = series.get("Collection", "")
        dest = output_root / f"{collection}_{uid}"

        print(f"  [{i}/{len(series_list)}] {uid[:40]}... ({slices} slices, {desc})")

        if dest.exists() and any(dest.iterdir()):
            print(f"           -> already exists, skipping")
            continue

        try:
            zip_bytes = download_series_zip(uid)
            n = extract_zip_flat(zip_bytes, dest)
            total_files += n
            size_mb = len(zip_bytes) / 1024 / 1024
            print(f"           -> {n} files  ({size_mb:.1f} MB)")
        except Exception as exc:
            print(f"           -> ERROR: {exc}", file=sys.stderr)

        if i < len(series_list):
            time.sleep(DOWNLOAD_DELAY_SECONDS)

    print(f"\nDone. {total_files} DICOM files in {output_root}")
    print()
    print("To import into AEGIS (local dev):")
    print(f"  cd api && go run ./cmd/import --dir {output_root} --project default")
    print()
    print("Or use the upload portal at http://localhost:3000")


if __name__ == "__main__":
    main()
