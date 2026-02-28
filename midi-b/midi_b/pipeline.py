"""Per-study de-identification pipeline orchestrator."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path

import pydicom

from .deid.dateshift import generate_date_offset
from .deid.engine import DeidOptions, DeidResult, deidentify
from .mapping import MappingCollector

log = logging.getLogger(__name__)


@dataclass
class PipelineOptions:
    """Options for the de-identification pipeline."""

    salt: str = "aegis-midi-b"
    date_shift: bool = True
    date_shift_max_days: int = 365
    keep_private_tags: bool = False
    retained_tags: list[str] | None = None
    dry_run: bool = False


@dataclass
class PipelineStats:
    """Summary statistics from a pipeline run."""

    files_processed: int = 0
    files_skipped: int = 0
    studies_found: set[str] = field(default_factory=set)
    private_tags_removed: int = 0
    errors: list[str] = field(default_factory=list)


def process_study(
    input_dir: Path,
    output_dir: Path,
    options: PipelineOptions | None = None,
) -> tuple[PipelineStats, MappingCollector]:
    """Scan *input_dir* for DICOM files, de-identify, and write to *output_dir*.

    Returns pipeline stats and collected mappings.
    """
    opts = options or PipelineOptions()
    stats = PipelineStats()
    collector = MappingCollector()

    # Discover DICOM files
    dcm_files = sorted(
        p for p in input_dir.rglob("*")
        if p.is_file() and p.suffix.lower() in (".dcm", ".dicom", "")
    )

    if not dcm_files:
        log.warning("No DICOM files found in %s", input_dir)
        return stats, collector

    # Pre-scan to determine per-patient date shift offsets
    patient_offsets: dict[str, int] = {}

    for dcm_path in dcm_files:
        try:
            ds = pydicom.dcmread(str(dcm_path), stop_before_pixels=True)
        except Exception as exc:
            log.warning("Skipping non-DICOM file %s: %s", dcm_path, exc)
            stats.files_skipped += 1
            continue

        study_uid = getattr(ds, "StudyInstanceUID", None)
        if study_uid:
            stats.studies_found.add(str(study_uid))

        # Compute date shift offset per patient
        if opts.date_shift:
            pid = str(getattr(ds, "PatientID", "")) or ""
            if pid and pid not in patient_offsets:
                patient_offsets[pid] = generate_date_offset(pid, opts.salt, opts.date_shift_max_days)

    if opts.dry_run:
        stats.files_processed = len(dcm_files) - stats.files_skipped
        return stats, collector

    # Reset skip count for actual processing pass
    stats.files_skipped = 0

    for dcm_path in dcm_files:
        try:
            ds = pydicom.dcmread(str(dcm_path))
        except Exception as exc:
            log.warning("Skipping non-DICOM file %s: %s", dcm_path, exc)
            stats.files_skipped += 1
            continue

        pid = str(getattr(ds, "PatientID", "")) or ""
        offset = patient_offsets.get(pid) if opts.date_shift else None

        deid_opts = DeidOptions(
            salt=opts.salt,
            keep_private_tags=opts.keep_private_tags,
            retained_tags=opts.retained_tags,
            date_shift_offset=offset,
        )

        try:
            result: DeidResult = deidentify(ds, deid_opts)
        except Exception as exc:
            msg = f"De-identification failed for {dcm_path}: {exc}"
            log.error(msg)
            stats.errors.append(msg)
            continue

        # Collect mappings
        collector.add_uid_mappings(result.uid_mappings)
        if result.patient_id_mapping:
            collector.add_patient_id_mapping(
                result.patient_id_mapping["original"],
                result.patient_id_mapping["replacement"],
                offset,
            )

        stats.private_tags_removed += result.private_tags_removed

        # Determine output path (preserve relative structure)
        rel_path = dcm_path.relative_to(input_dir)
        out_path = output_dir / rel_path
        out_path.parent.mkdir(parents=True, exist_ok=True)

        ds.save_as(str(out_path), write_like_original=False)
        stats.files_processed += 1

    return stats, collector
