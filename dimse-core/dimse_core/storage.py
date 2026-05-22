"""Common on-disk study storage layout used by SCP + pipeline + shipper.

Layout:
    data_dir/
    └── dicom/
        ├── raw/                 # bytes as they arrived from PACS / web upload
        │   └── <study_uid>/
        │       ├── 0.dcm
        │       └── 1.dcm
        └── clean/               # post-pipeline (de-identified, etc.)
            └── <study_uid>/
                ├── 0.dcm
                └── 1.dcm

This is the same shape used by the cloud receiver and the spoke router so
de-id / shipper / cloud-API code can read either path without branching.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass
class StudyLayout:
    """Read/write helper for one study's directories."""

    data_dir: str
    study_instance_uid: str

    @property
    def raw_dir(self) -> Path:
        return Path(self.data_dir) / "dicom" / "raw" / self.study_instance_uid

    @property
    def clean_dir(self) -> Path:
        return Path(self.data_dir) / "dicom" / "clean" / self.study_instance_uid

    def ensure(self) -> "StudyLayout":
        self.raw_dir.mkdir(parents=True, exist_ok=True)
        self.clean_dir.mkdir(parents=True, exist_ok=True)
        return self

    def next_raw_index(self) -> int:
        if not self.raw_dir.exists():
            return 0
        return sum(1 for _ in self.raw_dir.glob("*.dcm"))

    def write_raw(self, index: int, data: bytes) -> Path:
        self.raw_dir.mkdir(parents=True, exist_ok=True)
        target = self.raw_dir / f"{index}.dcm"
        tmp = target.with_suffix(".tmp")
        tmp.write_bytes(data)
        os.replace(tmp, target)
        return target

    def list_raw(self) -> list[Path]:
        if not self.raw_dir.exists():
            return []
        return sorted(
            self.raw_dir.glob("*.dcm"),
            key=lambda p: int(p.stem) if p.stem.isdigit() else 0,
        )

    def list_clean(self) -> list[Path]:
        if not self.clean_dir.exists():
            return []
        return sorted(
            self.clean_dir.glob("*.dcm"),
            key=lambda p: int(p.stem) if p.stem.isdigit() else 0,
        )

    def raw_size_bytes(self) -> int:
        return sum(p.stat().st_size for p in self.list_raw())

    def clean_size_bytes(self) -> int:
        return sum(p.stat().st_size for p in self.list_clean())
