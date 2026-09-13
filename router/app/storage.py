"""Local on-disk study layout.

    data_dir/
    └── studies/
        └── <study_uid>/
            ├── raw/            # bytes as they arrived from PACS / web upload
            │   ├── 0.dcm
            │   └── 1.dcm
            ├── deid/           # after tag de-id (and defacing if enabled)
            │   ├── 0.dcm
            │   └── 1.dcm
            └── manifest.json   # study metadata + pipeline state

A separate quarantine_dir is used by quarantine.py for studies that failed.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path


@dataclass
class StudyPaths:
    study_uid: str
    root: Path

    @property
    def raw(self) -> Path:
        return self.root / "raw"

    @property
    def deid(self) -> Path:
        return self.root / "deid"

    @property
    def manifest(self) -> Path:
        return self.root / "manifest.json"


def layout(data_dir: str, study_uid: str) -> StudyPaths:
    root = Path(data_dir) / "studies" / study_uid
    paths = StudyPaths(study_uid=study_uid, root=root)
    paths.raw.mkdir(parents=True, exist_ok=True)
    paths.deid.mkdir(parents=True, exist_ok=True)
    return paths


def next_raw_index(paths: StudyPaths) -> int:
    """Used by the SCP so re-sends append rather than overwrite."""
    if not paths.raw.exists():
        return 0
    return sum(1 for _ in paths.raw.glob("*.dcm"))


def write_raw_bytes(paths: StudyPaths, index: int, data: bytes) -> Path:
    target = paths.raw / f"{index}.dcm"
    tmp = target.with_suffix(".tmp")
    tmp.write_bytes(data)
    os.replace(tmp, target)
    return target


def list_raw_files(paths: StudyPaths) -> list[Path]:
    if not paths.raw.exists():
        return []
    return sorted(paths.raw.glob("*.dcm"), key=lambda p: int(p.stem) if p.stem.isdigit() else 0)


def list_deid_files(paths: StudyPaths) -> list[Path]:
    if not paths.deid.exists():
        return []
    return sorted(paths.deid.glob("*.dcm"), key=lambda p: int(p.stem) if p.stem.isdigit() else 0)


def read_manifest(paths: StudyPaths) -> dict:
    if not paths.manifest.exists():
        return {}
    try:
        return json.loads(paths.manifest.read_text())
    except (OSError, json.JSONDecodeError):
        return {}


def write_manifest(paths: StudyPaths, manifest: dict) -> None:
    tmp = paths.manifest.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(manifest, indent=2, sort_keys=True, default=str))
    os.replace(tmp, paths.manifest)
