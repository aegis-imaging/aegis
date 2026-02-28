"""Tests for the CLI entry point."""

from pathlib import Path

from click.testing import CliRunner

from tests.conftest import make_dataset, make_dicom_file
from midi_b.cli import main


def test_dry_run(tmp_path: Path):
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study.dcm")

    output_dir = tmp_path / "output"
    runner = CliRunner()
    result = runner.invoke(main, [
        "--input-dir", str(input_dir),
        "--output-dir", str(output_dir),
        "--dry-run",
    ])
    assert result.exit_code == 0
    assert "dry run" in result.output.lower()


def test_full_run(tmp_path: Path):
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study.dcm")

    output_dir = tmp_path / "output"
    runner = CliRunner()
    result = runner.invoke(main, [
        "--input-dir", str(input_dir),
        "--output-dir", str(output_dir),
        "--salt", "test-salt",
        "--no-date-shift",
    ])
    assert result.exit_code == 0
    assert "Files processed" in result.output
    assert (output_dir / "uid_mappings.csv").exists()
    assert (output_dir / "patient_id_mappings.csv").exists()
