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
	InstitutionID    *string   `json:"institution_id,omitempty"`
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
	PhiScanRequired  bool      `json:"phi_scan_required"`
	PhiScanStatus    string    `json:"phi_scan_status"`
	QcRequired       bool      `json:"qc_required"`
	QcStatus         string    `json:"qc_status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const studyColumns = `
	id, project_id, upload_session_id, institution_id, study_instance_uid, modality, body_part,
	study_description, series_count, instance_count, status, defacing_required,
	dicom_store, source, phi_scan_required, phi_scan_status, qc_required, qc_status, created_at, updated_at`

type scannable interface {
	Scan(...any) error
}

func scanStudy(row scannable, s *Study) error {
	return row.Scan(
		&s.ID, &s.ProjectID, &s.UploadSessionID, &s.InstitutionID, &s.StudyInstanceUID,
		&s.Modality, &s.BodyPart, &s.StudyDescription, &s.SeriesCount, &s.InstanceCount,
		&s.Status, &s.DefacingRequired, &s.DicomStore, &s.Source,
		&s.PhiScanRequired, &s.PhiScanStatus, &s.QcRequired, &s.QcStatus, &s.CreatedAt, &s.UpdatedAt,
	)
}

func CreateStudy(ctx context.Context, db *sql.DB, s *Study) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO studies (project_id, upload_session_id, institution_id, study_instance_uid, modality, body_part,
		                     study_description, series_count, instance_count, status, defacing_required,
		                     dicom_store, source, phi_scan_required, phi_scan_status, qc_required, qc_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at`,
		s.ProjectID, s.UploadSessionID, s.InstitutionID, s.StudyInstanceUID, s.Modality, s.BodyPart,
		s.StudyDescription, s.SeriesCount, s.InstanceCount, s.Status, s.DefacingRequired,
		s.DicomStore, s.Source, s.PhiScanRequired, s.PhiScanStatus, s.QcRequired, s.QcStatus).
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

// StudyFilters holds optional filter values for ListStudies / CountStudies.
type StudyFilters struct {
	ProjectID string
	Status    string // received|defacing|clean|defaced|approved|rejected
	Modality  string // MRI|CT|PET|… (case-insensitive prefix match)
	Source    string // external|internal
	Search    string // substring match on study_instance_uid or study_description
}

func studyWhere(f StudyFilters) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	if f.ProjectID != "" {
		clauses = append(clauses, fmt.Sprintf(`project_id = $%d`, n))
		args = append(args, f.ProjectID)
		n++
	}
	if f.Status != "" {
		clauses = append(clauses, fmt.Sprintf(`status = $%d`, n))
		args = append(args, f.Status)
		n++
	}
	if f.Modality != "" {
		clauses = append(clauses, fmt.Sprintf(`upper(modality) = upper($%d)`, n))
		args = append(args, f.Modality)
		n++
	}
	if f.Source != "" {
		clauses = append(clauses, fmt.Sprintf(`source = $%d`, n))
		args = append(args, f.Source)
		n++
	}
	if f.Search != "" {
		clauses = append(clauses, fmt.Sprintf(
			`(study_instance_uid ILIKE $%d OR study_description ILIKE $%d)`, n, n))
		args = append(args, "%"+f.Search+"%")
		n++
	}
	_ = n

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + clauses[0]
		for _, c := range clauses[1:] {
			where += " AND " + c
		}
	}
	return where, args
}

func ListStudies(ctx context.Context, db *sql.DB, f StudyFilters, limit, offset int) ([]Study, error) {
	where, args := studyWhere(f)
	argN := len(args) + 1

	query := `SELECT` + studyColumns + ` FROM studies` + where + ` ORDER BY created_at DESC`

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

func CountStudies(ctx context.Context, db *sql.DB, f StudyFilters) (int, error) {
	where, args := studyWhere(f)
	var total int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM studies`+where, args...).Scan(&total)
	return total, err
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

// SetPhiScanRequired sets the phi_scan_required flag and initialises phi_scan_status to "pending".
func SetPhiScanRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET phi_scan_required = $1, phi_scan_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdatePhiScanStatus sets the phi_scan_status field on a study.
func UpdatePhiScanStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET phi_scan_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// SetQcRequired sets the qc_required flag and initialises qc_status to "pending".
func SetQcRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET qc_required = $1, qc_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateQcStatus sets the qc_status field on a study.
func UpdateQcStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET qc_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}
