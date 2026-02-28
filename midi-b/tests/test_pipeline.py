"""Tests for the pipeline orchestrator."""

from pathlib import Path
from unittest.mock import patch

from tests.conftest import make_dataset, make_dicom_file
from midi_b.pipeline import PipelineOptions, process_study


def test_dry_run(tmp_path: Path):
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study1.dcm")

    output_dir = tmp_path / "output"
    opts = PipelineOptions(dry_run=True)
    stats, collector = process_study(input_dir, output_dir, opts)

    assert stats.files_processed >= 1
    assert len(stats.studies_found) == 1
    assert not output_dir.exists()  # dry run — no output


def test_end_to_end(tmp_path: Path):
    input_dir = tmp_path / "input"
    input_dir.mkdir()

    # Create 2 files in the same study
    ds1 = make_dataset()
    ds2 = make_dataset(SOPInstanceUID="1.2.840.113619.2.55.3.12345.1.2")
    make_dicom_file(input_dir, ds1, "file1.dcm")
    make_dicom_file(input_dir, ds2, "file2.dcm")

    output_dir = tmp_path / "output"
    opts = PipelineOptions(salt="test-salt", date_shift=False)
    stats, collector = process_study(input_dir, output_dir, opts)

    assert stats.files_processed == 2
    assert stats.files_skipped == 0
    assert len(stats.errors) == 0
    assert len(stats.studies_found) == 1

    # Check output files exist
    assert (output_dir / "file1.dcm").exists()
    assert (output_dir / "file2.dcm").exists()

    # Check mappings
    uid_csv = collector.to_uid_csv()
    assert "2.25." in uid_csv
    pid_csv = collector.to_patient_id_csv()
    assert "MRN-12345" in pid_csv
    assert "SUBJ-" in pid_csv


def test_skips_non_dicom(tmp_path: Path):
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    # Write a non-DICOM file
    (input_dir / "readme.txt").write_text("not a DICOM file")
    # Write an actual DICOM file
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study.dcm")

    output_dir = tmp_path / "output"
    stats, _ = process_study(input_dir, output_dir, PipelineOptions())

    assert stats.files_processed >= 1


def test_date_shift_applied(tmp_path: Path):
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study.dcm")

    output_dir = tmp_path / "output"
    opts = PipelineOptions(date_shift=True, date_shift_max_days=365)
    stats, collector = process_study(input_dir, output_dir, opts)

    assert stats.files_processed == 1
    # Patient ID mapping should include date shift offset
    pid_csv = collector.to_patient_id_csv()
    assert "SUBJ-" in pid_csv


def test_pixel_redact_local_skipped_when_tesseract_unavailable(tmp_path: Path):
    """Pixel redaction gracefully skips when Tesseract is not installed."""
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study.dcm")

    output_dir = tmp_path / "output"
    opts = PipelineOptions(
        salt="test-salt", date_shift=False, pixel_redact=True,
    )

    with patch("midi_b.deid.pixel_detect.tesseract_available", return_value=False):
        stats, _ = process_study(input_dir, output_dir, opts)

    assert stats.files_processed == 1
    assert stats.files_pixel_redacted == 0
    assert (output_dir / "study.dcm").exists()


def test_phi_service_url_passed_to_pipeline(tmp_path: Path):
    """When phi_service_url is set, the pipeline creates a remote text scrub backend."""
    input_dir = tmp_path / "input"
    input_dir.mkdir()
    ds = make_dataset()
    make_dicom_file(input_dir, ds, "study.dcm")

    output_dir = tmp_path / "output"
    opts = PipelineOptions(
        salt="test-salt",
        date_shift=False,
        phi_service_url="http://localhost:8082",
    )

    # Mock the RemoteTextScrubBackend to avoid actual HTTP calls
    with patch("midi_b.pipeline.RemoteTextScrubBackend") as mock_cls:
        mock_backend = mock_cls.return_value
        mock_backend.scrub.return_value = type("R", (), {"text": "clean", "phi_found": False})()
        stats, _ = process_study(input_dir, output_dir, opts)

    assert stats.files_processed == 1
    mock_cls.assert_called_once_with("http://localhost:8082")
