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
	BidsRequired           bool      `json:"bids_required"`
	BidsStatus             string    `json:"bids_status"`
	ClassificationRequired bool      `json:"classification_required"`
	ClassificationStatus   string    `json:"classification_status"`
	ProtocolRequired       bool      `json:"protocol_required"`
	ProtocolStatus         string    `json:"protocol_status"`
	ExportRequired         bool      `json:"export_required"`
	ExportStatus           string    `json:"export_status"`
	DefaceQaScore          *float64  `json:"deface_qa_score,omitempty"`
	SubjectID              *string   `json:"subject_id,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

const studyColumns = `
	id, project_id, upload_session_id, institution_id, study_instance_uid, modality, body_part,
	study_description, series_count, instance_count, status, defacing_required,
	dicom_store, source, phi_scan_required, phi_scan_status, qc_required, qc_status,
	bids_required, bids_status, classification_required, classification_status,
	protocol_required, protocol_status,
	export_required, export_status,
	deface_qa_score,
	subject_id,
	created_at, updated_at`

type scannable interface {
	Scan(...any) error
}

func scanStudy(row scannable, s *Study) error {
	return row.Scan(
		&s.ID, &s.ProjectID, &s.UploadSessionID, &s.InstitutionID, &s.StudyInstanceUID,
		&s.Modality, &s.BodyPart, &s.StudyDescription, &s.SeriesCount, &s.InstanceCount,
		&s.Status, &s.DefacingRequired, &s.DicomStore, &s.Source,
		&s.PhiScanRequired, &s.PhiScanStatus, &s.QcRequired, &s.QcStatus,
		&s.BidsRequired, &s.BidsStatus,
		&s.ClassificationRequired, &s.ClassificationStatus,
		&s.ProtocolRequired, &s.ProtocolStatus,
		&s.ExportRequired, &s.ExportStatus,
		&s.DefaceQaScore,
		&s.SubjectID,
		&s.CreatedAt, &s.UpdatedAt,
	)
}

func CreateStudy(ctx context.Context, db *sql.DB, s *Study) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO studies (project_id, upload_session_id, institution_id, study_instance_uid, modality, body_part,
		                     study_description, series_count, instance_count, status, defacing_required,
		                     dicom_store, source, phi_scan_required, phi_scan_status, qc_required, qc_status,
		                     bids_required, bids_status,
		                     classification_required, classification_status,
		                     protocol_required, protocol_status,
		                     export_required, export_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING id, created_at, updated_at`,
		s.ProjectID, s.UploadSessionID, s.InstitutionID, s.StudyInstanceUID, s.Modality, s.BodyPart,
		s.StudyDescription, s.SeriesCount, s.InstanceCount, s.Status, s.DefacingRequired,
		s.DicomStore, s.Source, s.PhiScanRequired, s.PhiScanStatus, s.QcRequired, s.QcStatus,
		s.BidsRequired, s.BidsStatus,
		s.ClassificationRequired, s.ClassificationStatus,
		s.ProtocolRequired, s.ProtocolStatus,
		s.ExportRequired, s.ExportStatus).
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
	Status    string    // received|defacing|clean|defaced|approved|rejected
	Modality  string    // MRI|CT|PET|… (case-insensitive exact match)
	BodyPart  string    // HEAD|CHEST|… (case-insensitive exact match)
	Source    string    // external|internal
	Search    string    // substring match on study_instance_uid or study_description
	SubjectID string    // exact match on subject_id
	DateFrom  time.Time // created_at >= DateFrom (zero = no lower bound)
	DateTo    time.Time // created_at <= DateTo   (zero = no upper bound)
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
	if f.BodyPart != "" {
		clauses = append(clauses, fmt.Sprintf(`upper(body_part) = upper($%d)`, n))
		args = append(args, f.BodyPart)
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
	if f.SubjectID != "" {
		clauses = append(clauses, fmt.Sprintf(`subject_id = $%d`, n))
		args = append(args, f.SubjectID)
		n++
	}
	if !f.DateFrom.IsZero() {
		clauses = append(clauses, fmt.Sprintf(`created_at >= $%d`, n))
		args = append(args, f.DateFrom.UTC())
		n++
	}
	if !f.DateTo.IsZero() {
		clauses = append(clauses, fmt.Sprintf(`created_at <= $%d`, n))
		args = append(args, f.DateTo.UTC())
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

// UpdateDefaceQaScore stores the SSIM-based visual QA score from the defacing service.
func UpdateDefaceQaScore(ctx context.Context, db *sql.DB, id string, score float64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET deface_qa_score = $1, updated_at = now()
		WHERE id = $2`, score, id)
	return err
}

// UpdateStudySubjectID sets the subject_id on a study for cross-session linking.
func UpdateStudySubjectID(ctx context.Context, db *sql.DB, id string, subjectID *string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET subject_id = $1, updated_at = now()
		WHERE id = $2`, subjectID, id)
	return err
}

// SubjectSummary holds per-subject study counts for the subjects listing.
type SubjectSummary struct {
	SubjectID  string `json:"subject_id"`
	ProjectID  string `json:"project_id"`
	StudyCount int    `json:"study_count"`
}

// ListSubjects returns unique subject_ids with study counts, optionally filtered by project.
func ListSubjects(ctx context.Context, db *sql.DB, projectID string) ([]SubjectSummary, error) {
	query := `SELECT subject_id, project_id, count(*) AS study_count
	          FROM studies WHERE subject_id IS NOT NULL`
	args := []any{}
	if projectID != "" {
		query += ` AND project_id = $1`
		args = append(args, projectID)
	}
	query += ` GROUP BY subject_id, project_id ORDER BY subject_id`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubjectSummary
	for rows.Next() {
		var s SubjectSummary
		if err := rows.Scan(&s.SubjectID, &s.ProjectID, &s.StudyCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
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

// SetBidsRequired sets the bids_required flag and initialises bids_status to "pending".
func SetBidsRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET bids_required = $1, bids_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateBidsStatus sets the bids_status field on a study.
func UpdateBidsStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET bids_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// SetClassificationRequired sets the classification_required flag and initialises classification_status to "pending".
func SetClassificationRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET classification_required = $1, classification_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateClassificationStatus sets the classification_status field on a study.
func UpdateClassificationStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET classification_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// SetProtocolRequired sets the protocol_required flag and initialises protocol_status to "pending".
func SetProtocolRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET protocol_required = $1, protocol_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateProtocolStatus sets the protocol_status field on a study.
func UpdateProtocolStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET protocol_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// UpdateStudyMetadata updates the modality and body_part fields on a study.
// Used by the classification service to fill in missing metadata after DICOM header analysis.
func UpdateStudyMetadata(ctx context.Context, db *sql.DB, id, modality, bodyPart string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET modality = $1, body_part = $2, updated_at = now()
		WHERE id = $3`, modality, bodyPart, id)
	return err
}

// --- Atomic claim functions for pipeline orchestration ---
// Each function attempts to transition a study's processing status from "pending"
// to "in-progress" using a conditional UPDATE. Returns true if this caller won
// the race (RowsAffected == 1), false if another goroutine already claimed it.

// ClaimClassification atomically claims classification dispatch.
func ClaimClassification(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET classification_status = 'classifying', updated_at = now()
		WHERE id = $1 AND classification_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ClaimPhiScan atomically claims PHI scan dispatch.
func ClaimPhiScan(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET phi_scan_status = 'scanning', updated_at = now()
		WHERE id = $1 AND phi_scan_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ClaimDefacing atomically claims defacing dispatch.
func ClaimDefacing(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET status = 'defacing', updated_at = now()
		WHERE id = $1 AND status = 'received' AND defacing_required = true`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ClaimQcCheck atomically claims QC check dispatch.
func ClaimQcCheck(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET qc_status = 'checking', updated_at = now()
		WHERE id = $1 AND qc_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ClaimBidsConversion atomically claims BIDS conversion dispatch.
func ClaimBidsConversion(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET bids_status = 'converting', updated_at = now()
		WHERE id = $1 AND bids_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ClaimProtocolCheck atomically claims protocol check dispatch.
func ClaimProtocolCheck(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET protocol_status = 'checking', updated_at = now()
		WHERE id = $1 AND protocol_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetExportRequired sets the export_required flag and initialises export_status to "pending".
func SetExportRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET export_required = $1, export_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateExportStatus sets the export_status field on a study.
func UpdateExportStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET export_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// ClaimExport atomically claims export dispatch.
func ClaimExport(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET export_status = 'exporting', updated_at = now()
		WHERE id = $1 AND export_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// StudyStatusCounts holds per-status study counts for the dashboard overview.
type StudyStatusCounts struct {
	Received  int `json:"received"`
	Defacing  int `json:"defacing"`
	Clean     int `json:"clean"`
	Defaced   int `json:"defaced"`
	Approved  int `json:"approved"`
	Rejected  int `json:"rejected"`
	Total     int `json:"total"`
}

// GetStudyStatusCounts returns a snapshot count of studies by status.
func GetStudyStatusCounts(ctx context.Context, db *sql.DB) (StudyStatusCounts, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT status, count(*) FROM studies GROUP BY status`)
	if err != nil {
		return StudyStatusCounts{}, err
	}
	defer rows.Close()

	var c StudyStatusCounts
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return StudyStatusCounts{}, err
		}
		switch status {
		case "received":
			c.Received = n
		case "defacing":
			c.Defacing = n
		case "clean":
			c.Clean = n
		case "defaced":
			c.Defaced = n
		case "approved":
			c.Approved = n
		case "rejected":
			c.Rejected = n
		}
		c.Total += n
	}
	return c, rows.Err()
}

// BreakdownRow is one cell in the modality × body_part cross-tab.
type BreakdownRow struct {
	Modality  string `json:"modality"`
	BodyPart  string `json:"body_part"`
	Count     int    `json:"count"`
}

// GetStudyBreakdown returns study counts grouped by (modality, body_part).
// Empty modality/body_part values are normalised to the empty string.
func GetStudyBreakdown(ctx context.Context, db *sql.DB) ([]BreakdownRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT coalesce(modality, ''), coalesce(body_part, ''), count(*)
		FROM studies
		GROUP BY modality, body_part
		ORDER BY count(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []BreakdownRow
	for rows.Next() {
		var r BreakdownRow
		if err := rows.Scan(&r.Modality, &r.BodyPart, &r.Count); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
