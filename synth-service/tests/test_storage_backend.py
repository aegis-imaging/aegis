"""Tests for synth-service storage backend (S3 upload path)."""

from __future__ import annotations

from unittest.mock import MagicMock, call, patch


class TestUploadSynthStudy:
    """Tests for upload_synth_study() — S3 and local paths."""

    def test_no_op_when_local_mode(self, tmp_path):
        """upload_synth_study is a no-op when STORAGE_MODE != 's3'."""
        dcm = tmp_path / "study" / "0001.dcm"
        dcm.parent.mkdir()
        dcm.write_bytes(b"DICOM")

        with patch("app.storage_backend.cfg") as mock_cfg:
            mock_cfg.storage_mode = "local"
            mock_cfg.s3_bucket = ""
            mock_cfg.s3_region = "us-east-1"

            from app.storage_backend import upload_synth_study
            upload_synth_study("1.2.3.4", [str(dcm)])

        # Local file must still exist (not cleaned up in local mode).
        assert dcm.exists()

    def test_uploads_to_s3_in_s3_mode(self, tmp_path):
        """In S3 mode each file is put to s3://{bucket}/synth/{uid}/{name}."""
        study_uid = "1.2.3.4.5"
        study_dir = tmp_path / "synth" / study_uid
        study_dir.mkdir(parents=True)

        files = []
        for i in range(3):
            f = study_dir / f"{i+1:04d}.dcm"
            f.write_bytes(b"DICOM" * (i + 1))
            files.append(str(f))

        mock_s3 = MagicMock()
        with patch("app.storage_backend.cfg") as mock_cfg, \
             patch("boto3.client", return_value=mock_s3):
            mock_cfg.storage_mode = "s3"
            mock_cfg.s3_bucket = "my-bucket"
            mock_cfg.s3_region = "us-east-1"

            from app.storage_backend import upload_synth_study
            upload_synth_study(study_uid, files)

        assert mock_s3.put_object.call_count == 3
        expected_keys = {f"synth/{study_uid}/{i+1:04d}.dcm" for i in range(3)}
        actual_keys = {c.kwargs["Key"] for c in mock_s3.put_object.call_args_list}
        assert actual_keys == expected_keys

        # Bucket must match config.
        for c in mock_s3.put_object.call_args_list:
            assert c.kwargs["Bucket"] == "my-bucket"

    def test_cleans_up_local_dir_after_s3_upload(self, tmp_path):
        """Local study dir is removed after successful S3 upload."""
        study_uid = "9.8.7.6"
        study_dir = tmp_path / "synth" / study_uid
        study_dir.mkdir(parents=True)
        dcm = study_dir / "0001.dcm"
        dcm.write_bytes(b"DICOM")

        mock_s3 = MagicMock()
        with patch("app.storage_backend.cfg") as mock_cfg, \
             patch("boto3.client", return_value=mock_s3):
            mock_cfg.storage_mode = "s3"
            mock_cfg.s3_bucket = "my-bucket"
            mock_cfg.s3_region = "us-east-1"

            from app.storage_backend import upload_synth_study
            upload_synth_study(study_uid, [str(dcm)])

        assert not study_dir.exists()

    def test_no_op_on_empty_path_list(self):
        """upload_synth_study with empty list does not crash."""
        mock_s3 = MagicMock()
        with patch("app.storage_backend.cfg") as mock_cfg, \
             patch("boto3.client", return_value=mock_s3):
            mock_cfg.storage_mode = "s3"
            mock_cfg.s3_bucket = "my-bucket"
            mock_cfg.s3_region = "us-east-1"

            from app.storage_backend import upload_synth_study
            upload_synth_study("1.2.3", [])

        mock_s3.put_object.assert_not_called()
