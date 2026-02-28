"""Mapping CSV export for MIDI-B submission artifacts."""

from __future__ import annotations

import csv
import io


class MappingCollector:
    """Accumulates UID and patient ID mappings across multiple files."""

    def __init__(self) -> None:
        self.uid_mappings: dict[str, str] = {}
        self.patient_id_mappings: dict[str, dict[str, str | int | None]] = {}

    def add_uid_mappings(self, mappings: dict[str, str]) -> None:
        """Merge UID mappings from a single file de-identification."""
        self.uid_mappings.update(mappings)

    def add_patient_id_mapping(
        self,
        original: str,
        replacement: str,
        date_shift_offset: int | None = None,
    ) -> None:
        """Record a patient ID → pseudonym mapping."""
        self.patient_id_mappings[original] = {
            "replacement": replacement,
            "date_shift_offset": date_shift_offset,
        }

    def to_uid_csv(self) -> str:
        """Export UID mappings as CSV text."""
        buf = io.StringIO()
        writer = csv.writer(buf)
        writer.writerow(["original_uid", "replacement_uid"])
        for original, replacement in sorted(self.uid_mappings.items()):
            writer.writerow([original, replacement])
        return buf.getvalue()

    def to_patient_id_csv(self) -> str:
        """Export patient ID mappings as CSV text."""
        buf = io.StringIO()
        writer = csv.writer(buf)
        writer.writerow(["original_patient_id", "replacement_patient_id", "date_shift_offset"])
        for original, info in sorted(self.patient_id_mappings.items()):
            writer.writerow([
                original,
                info["replacement"],
                info["date_shift_offset"] if info["date_shift_offset"] is not None else "",
            ])
        return buf.getvalue()
