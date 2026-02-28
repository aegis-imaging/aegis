"""MIDI-B CLI entry point."""

from __future__ import annotations

import logging
import sys
from pathlib import Path

import click

from .pipeline import PipelineOptions, process_study


@click.command()
@click.option("--input-dir", required=True, type=click.Path(exists=True, path_type=Path), help="Directory with raw DICOM files")
@click.option("--output-dir", required=True, type=click.Path(path_type=Path), help="Output directory for de-identified files")
@click.option("--salt", default="aegis-midi-b", help="Pseudonymization salt")
@click.option("--date-shift/--no-date-shift", default=True, help="Enable date shifting")
@click.option("--date-shift-max-days", default=365, type=int, help="Max offset range in days")
@click.option("--keep-private-tags", is_flag=True, default=False, help="Keep private tags")
@click.option("--dry-run", is_flag=True, default=False, help="Scan only, don't write")
@click.option("--verbose", "-v", is_flag=True, default=False, help="Verbose logging")
def main(
    input_dir: Path,
    output_dir: Path,
    salt: str,
    date_shift: bool,
    date_shift_max_days: int,
    keep_private_tags: bool,
    dry_run: bool,
    verbose: bool,
) -> None:
    """MIDI-B: Medical Image De-Identification Benchmark CLI.

    Reads DICOM files from INPUT_DIR, applies full de-identification
    (PS3.15 Annex E Basic Profile), and writes results to OUTPUT_DIR.
    """
    logging.basicConfig(
        level=logging.DEBUG if verbose else logging.INFO,
        format="%(levelname)s %(message)s",
    )

    opts = PipelineOptions(
        salt=salt,
        date_shift=date_shift,
        date_shift_max_days=date_shift_max_days,
        keep_private_tags=keep_private_tags,
        dry_run=dry_run,
    )

    if not dry_run:
        output_dir.mkdir(parents=True, exist_ok=True)

    stats, collector = process_study(input_dir, output_dir, opts)

    click.echo(f"Studies found:        {len(stats.studies_found)}")
    click.echo(f"Files processed:      {stats.files_processed}")
    click.echo(f"Files skipped:        {stats.files_skipped}")
    click.echo(f"Private tags removed: {stats.private_tags_removed}")

    if stats.errors:
        click.echo(f"Errors:               {len(stats.errors)}")
        for err in stats.errors:
            click.secho(f"  {err}", fg="red", err=True)

    if dry_run:
        click.echo("(dry run — no files written)")
        return

    # Write mapping CSVs
    uid_csv_path = output_dir / "uid_mappings.csv"
    uid_csv_path.write_text(collector.to_uid_csv())
    click.echo(f"UID mappings:         {uid_csv_path}")

    patient_csv_path = output_dir / "patient_id_mappings.csv"
    patient_csv_path.write_text(collector.to_patient_id_csv())
    click.echo(f"Patient ID mappings:  {patient_csv_path}")

    if stats.errors:
        sys.exit(1)


if __name__ == "__main__":
    main()
