package model

import (
	"context"
	"database/sql"
	"time"
)

type UploadSession struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"project_id"`
	Status           string  `json:"status"`
	FileCount        int     `json:"file_count"`
	StoragePrefix    string  `json:"storage_prefix"`
	UploaderIP       string  `json:"uploader_ip,omitempty"`
	UploaderEmail    string  `json:"uploader_email,omitempty"`
	StudyInstanceUID *string `json:"study_instance_uid,omitempty"`
	Modality         *string `json:"modality,omitempty"`
	BodyPart         *string `json:"body_part,omitempty"`
	ErrorMessage     *string `json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func CreateUploadSession(ctx context.Context, db *sql.DB, projectID string, fileCount int, storagePrefix, uploaderIP, uploaderEmail string) (*UploadSession, error) {
	var s UploadSession
	err := db.QueryRowContext(ctx, `
		INSERT INTO upload_sessions (project_id, file_count, storage_prefix, uploader_ip, uploader_email)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, status, file_count, storage_prefix, uploader_ip, uploader_email,
		          study_instance_uid, modality, body_part, error_message, created_at, updated_at`,
		projectID, fileCount, storagePrefix, uploaderIP, uploaderEmail).
		Scan(&s.ID, &s.ProjectID, &s.Status, &s.FileCount, &s.StoragePrefix, &s.UploaderIP, &s.UploaderEmail,
			&s.StudyInstanceUID, &s.Modality, &s.BodyPart, &s.ErrorMessage, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetUploadSession(ctx context.Context, db *sql.DB, id string) (*UploadSession, error) {
	var s UploadSession
	err := db.QueryRowContext(ctx, `
		SELECT id, project_id, status, file_count, storage_prefix, uploader_ip, uploader_email,
		       study_instance_uid, modality, body_part, error_message, created_at, updated_at
		FROM upload_sessions WHERE id = $1`, id).
		Scan(&s.ID, &s.ProjectID, &s.Status, &s.FileCount, &s.StoragePrefix, &s.UploaderIP, &s.UploaderEmail,
			&s.StudyInstanceUID, &s.Modality, &s.BodyPart, &s.ErrorMessage, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func UpdateUploadSessionStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE upload_sessions SET status = $1, updated_at = now() WHERE id = $2`,
		status, id)
	return err
}

func UpdateUploadSessionComplete(ctx context.Context, db *sql.DB, id, studyUID, modality, bodyPart string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE upload_sessions
		SET status = 'completed', study_instance_uid = $1, modality = $2, body_part = $3, updated_at = now()
		WHERE id = $4`,
		studyUID, modality, bodyPart, id)
	return err
}

func UpdateUploadSessionFailed(ctx context.Context, db *sql.DB, id, errMsg string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE upload_sessions SET status = 'failed', error_message = $1, updated_at = now() WHERE id = $2`,
		errMsg, id)
	return err
}
