"""Per-study de-identification pipeline orchestrator."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path

import pydicom

from .deid.dateshift import generate_date_offset
from .deid.engine import DeidOptions, DeidResult, deidentify
from .deid.text_scrub import RemoteTextScrubBackend
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
    pixel_redact: bool = False
    phi_service_url: str | None = None


@dataclass
class PipelineStats:
    """Summary statistics from a pipeline run."""

    files_processed: int = 0
    files_skipped: int = 0
    files_pixel_redacted: int = 0
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

    # Set up remote text scrub backend if phi-service URL is configured
    text_scrub_backend: RemoteTextScrubBackend | None = None
    if opts.phi_service_url:
        text_scrub_backend = RemoteTextScrubBackend(opts.phi_service_url)

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
            result: DeidResult = deidentify(ds, deid_opts, text_scrub_backend=text_scrub_backend)
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

        ds.save_as(str(out_path))
        stats.files_processed += 1

    # Pixel redaction pass (after tag de-identification)
    if opts.pixel_redact and not opts.dry_run:
        _run_pixel_redaction(output_dir, dcm_files, input_dir, opts, stats)

    return stats, collector


def _run_pixel_redaction(
    output_dir: Path,
    dcm_files: list[Path],
    input_dir: Path,
    opts: PipelineOptions,
    stats: PipelineStats,
) -> None:
    """Run pixel redaction on de-identified output files."""
    if opts.phi_service_url:
        _run_pixel_redaction_remote(output_dir, dcm_files, input_dir, opts, stats)
    else:
        _run_pixel_redaction_local(output_dir, dcm_files, input_dir, stats)


def _run_pixel_redaction_local(
    output_dir: Path,
    dcm_files: list[Path],
    input_dir: Path,
    stats: PipelineStats,
) -> None:
    """Run pixel redaction locally using Tesseract OCR."""
    try:
        from .deid.pixel_detect import detect_file, tesseract_available
        from .deid.pixel_redact import redact_dicom_pixels
    except ImportError as e:
        log.warning("Pixel redaction dependencies not available: %s", e)
        return

    if not tesseract_available():
        log.warning("Tesseract OCR not installed — skipping pixel redaction")
        return

    for dcm_path in dcm_files:
        rel_path = dcm_path.relative_to(input_dir)
        out_path = output_dir / rel_path
        if not out_path.exists():
            continue

        try:
            regions = detect_file(str(out_path))
            if regions:
                region_dicts = [{"bbox": r.bbox} for r in regions]
                redacted = redact_dicom_pixels(str(out_path), str(out_path), region_dicts)
                if redacted:
                    stats.files_pixel_redacted += 1
        except Exception as exc:
            msg = f"Pixel redaction failed for {out_path}: {exc}"
            log.error(msg)
            stats.errors.append(msg)


def _run_pixel_redaction_remote(
    output_dir: Path,
    dcm_files: list[Path],
    input_dir: Path,
    opts: PipelineOptions,
    stats: PipelineStats,
) -> None:
    """Run pixel redaction via the remote phi-detection /redact endpoint."""
    import json
    import urllib.request

    service_url = opts.phi_service_url.rstrip("/")

    # Collect output file paths
    output_paths: list[str] = []
    for dcm_path in dcm_files:
        rel_path = dcm_path.relative_to(input_dir)
        out_path = output_dir / rel_path
        if out_path.exists():
            output_paths.append(str(out_path))

    if not output_paths:
        return

    payload = json.dumps({
        "study_uid": "midi-b-cli",
        "input_paths": output_paths,
        "output_dir": str(output_dir),
    }).encode()

    req = urllib.request.Request(
        f"{service_url}/redact",
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            body = json.loads(resp.read())
        stats.files_pixel_redacted = body.get("files_redacted", 0)
    except Exception as exc:
        msg = f"Remote pixel redaction failed: {exc}"
        log.error(msg)
        stats.errors.append(msg)
