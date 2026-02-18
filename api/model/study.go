package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Study struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	UploadSessionID  *string   `json:"upload_session_id,omitempty"`
	StudyInstanceUID string    `json:"study_instance_uid"`
	Modality         string    `json:"modality"`
	BodyPart         string    `json:"body_part"`
	StudyDescription string    `json:"study_description"`
	SeriesCount      int       `json:"series_count"`
	InstanceCount    int       `json:"instance_count"`
	Status           string    `json:"status"`
	DefacingRequired bool      `json:"defacing_required"`
	DicomStore       string    `json:"dicom_store"`
	Source           string    `json:"source"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const studyColumns = `
	id, project_id, upload_session_id, study_instance_uid, modality, body_part,
	study_description, series_count, instance_count, status, defacing_required,
	dicom_store, source, created_at, updated_at`

type scannable interface {
	Scan(...any) error
}

func scanStudy(row scannable, s *Study) error {
	return row.Scan(
		&s.ID, &s.ProjectID, &s.UploadSessionID, &s.StudyInstanceUID,
		&s.Modality, &s.BodyPart, &s.StudyDescription, &s.SeriesCount, &s.InstanceCount,
		&s.Status, &s.DefacingRequired, &s.DicomStore, &s.Source, &s.CreatedAt, &s.UpdatedAt,
	)
}

func CreateStudy(ctx context.Context, db *sql.DB, s *Study) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO studies (project_id, upload_session_id, study_instance_uid, modality, body_part,
		                     study_description, series_count, instance_count, status, defacing_required,
		                     dicom_store, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`,
		s.ProjectID, s.UploadSessionID, s.StudyInstanceUID, s.Modality, s.BodyPart,
		s.StudyDescription, s.SeriesCount, s.InstanceCount, s.Status, s.DefacingRequired,
		s.DicomStore, s.Source).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func GetStudyByID(ctx context.Context, db *sql.DB, id string) (*Study, error) {
	var s Study
	err := scanStudy(db.QueryRowContext(ctx,
		`SELECT`+studyColumns+` FROM studies WHERE id = $1`, id), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetStudyByUID(ctx context.Context, db *sql.DB, uid string) (*Study, error) {
	var s Study
	err := scanStudy(db.QueryRowContext(ctx,
		`SELECT`+studyColumns+` FROM studies WHERE study_instance_uid = $1`, uid), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ListStudies(ctx context.Context, db *sql.DB, projectID string, limit, offset int) ([]Study, error) {
	query := `SELECT` + studyColumns + ` FROM studies`
	var args []any
	argN := 1

	if projectID != "" {
		query += fmt.Sprintf(` WHERE project_id = $%d`, argN)
		args = append(args, projectID)
		argN++
	}

	query += ` ORDER BY created_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argN)
		args = append(args, limit)
		argN++
	}
	if offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, argN)
		args = append(args, offset)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studies []Study
	for rows.Next() {
		var s Study
		if err := scanStudy(rows, &s); err != nil {
			return nil, err
		}
		studies = append(studies, s)
	}
	return studies, rows.Err()
}

func UpdateStudyStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}

// GetUploaderEmail returns the uploader_email from the upload session linked to a study.
// Returns an empty string (no error) if the study has no associated upload session.
func GetUploaderEmail(ctx context.Context, db *sql.DB, studyID string) (string, error) {
	var uploaderEmail string
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(us.uploader_email, '')
		FROM studies s
		LEFT JOIN upload_sessions us ON us.id = s.upload_session_id
		WHERE s.id = $1`, studyID).Scan(&uploaderEmail)
	return uploaderEmail, err
}

// SetDefacingRequired overrides the defacing_required flag on a study.
func SetDefacingRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET defacing_required = $1, updated_at = now() WHERE id = $2`, required, id)
	return err
}

// UpdateStudyDefaced marks a study as defacing-complete and moves it to the clean store.
func UpdateStudyDefaced(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET status = 'defaced', dicom_store = 'clean', updated_at = now()
		WHERE id = $1`, id)
	return err
}
