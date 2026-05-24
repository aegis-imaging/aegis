package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Study struct {
	ID                     string     `json:"id"`
	ProjectID              string     `json:"project_id"`
	UploadSessionID        *string    `json:"upload_session_id,omitempty"`
	InstitutionID          *string    `json:"institution_id,omitempty"`
	StudyInstanceUID       string     `json:"study_instance_uid"`
	Modality               string     `json:"modality"`
	BodyPart               string     `json:"body_part"`
	StudyDescription       string     `json:"study_description"`
	StudyDate              *string    `json:"study_date,omitempty"`
	AnonPatientID          *string    `json:"anon_patient_id,omitempty"`
	SeriesCount            int        `json:"series_count"`
	InstanceCount          int        `json:"instance_count"`
	Status                 string     `json:"status"`
	DefacingRequired       bool       `json:"defacing_required"`
	DicomStore             string     `json:"dicom_store"`
	Source                 string     `json:"source"`
	PhiScanRequired        bool       `json:"phi_scan_required"`
	PhiScanStatus          string     `json:"phi_scan_status"`
	QcRequired             bool       `json:"qc_required"`
	QcStatus               string     `json:"qc_status"`
	BidsRequired           bool       `json:"bids_required"`
	BidsStatus             string     `json:"bids_status"`
	ClassificationRequired bool       `json:"classification_required"`
	ClassificationStatus   string     `json:"classification_status"`
	ProtocolRequired       bool       `json:"protocol_required"`
	ProtocolStatus         string     `json:"protocol_status"`
	ExportRequired         bool       `json:"export_required"`
	ExportStatus           string     `json:"export_status"`
	PixelRedactionRequired bool       `json:"pixel_redaction_required"`
	PixelRedactionStatus   string     `json:"pixel_redaction_status"`
	AnalyticsRequired      bool       `json:"analytics_required"`
	AnalyticsStatus        string     `json:"analytics_status"`
	SctRequired            bool       `json:"sct_required"`
	SctStatus              string     `json:"sct_status"`
	DefaceQaScore          *float64   `json:"deface_qa_score,omitempty"`
	SubjectID              *string    `json:"subject_id,omitempty"`
	RejectionReason        *string    `json:"rejection_reason,omitempty"`
	StudySizeBytes         int64      `json:"study_size_bytes"`
	PriorityFlag           bool       `json:"priority_flag"`
	AssignedTo             *string    `json:"assigned_to,omitempty"`
	AssignedAt             *time.Time `json:"assigned_at,omitempty"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
	AutoShareURL           *string    `json:"auto_share_url,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

const studyColumns = `
	id, project_id, upload_session_id, institution_id, study_instance_uid, modality, body_part,
	study_description, study_date, anon_patient_id, series_count, instance_count, status, defacing_required,
	dicom_store, source, phi_scan_required, phi_scan_status, qc_required, qc_status,
	bids_required, bids_status, classification_required, classification_status,
	protocol_required, protocol_status,
	export_required, export_status,
	pixel_redaction_required, pixel_redaction_status,
	analytics_required, analytics_status,
	sct_required, sct_status,
	deface_qa_score,
	subject_id,
	rejection_reason,
	study_size_bytes,
	priority_flag,
	assigned_to,
	assigned_at,
	deleted_at,
	auto_share_url,
	created_at, updated_at`

type scannable interface {
	Scan(...any) error
}

func scanStudy(row scannable, s *Study) error {
	return row.Scan(
		&s.ID, &s.ProjectID, &s.UploadSessionID, &s.InstitutionID, &s.StudyInstanceUID,
		&s.Modality, &s.BodyPart, &s.StudyDescription, &s.StudyDate, &s.AnonPatientID, &s.SeriesCount, &s.InstanceCount,
		&s.Status, &s.DefacingRequired, &s.DicomStore, &s.Source,
		&s.PhiScanRequired, &s.PhiScanStatus, &s.QcRequired, &s.QcStatus,
		&s.BidsRequired, &s.BidsStatus,
		&s.ClassificationRequired, &s.ClassificationStatus,
		&s.ProtocolRequired, &s.ProtocolStatus,
		&s.ExportRequired, &s.ExportStatus,
		&s.PixelRedactionRequired, &s.PixelRedactionStatus,
		&s.AnalyticsRequired, &s.AnalyticsStatus,
		&s.SctRequired, &s.SctStatus,
		&s.DefaceQaScore,
		&s.SubjectID,
		&s.RejectionReason,
		&s.StudySizeBytes,
		&s.PriorityFlag,
		&s.AssignedTo,
		&s.AssignedAt,
		&s.DeletedAt,
		&s.AutoShareURL,
		&s.CreatedAt, &s.UpdatedAt,
	)
}

// UpdateStudyDicomMetadata writes DICOM-tag-derived metadata fields onto
// an existing study row. Used by both the ingest path (after the first
// .dcm is stored) and the admin backfill endpoint for legacy studies that
// pre-date the extraction step.
//
// Pass nil to leave a field unchanged (COALESCE semantics).
//
// Distinct from UpdateStudyMetadata, which the classification service uses
// to update modality/body_part after model inference.
func UpdateStudyDicomMetadata(ctx context.Context, db *sql.DB, studyID string, anonPatientID, studyDate *string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies
		SET anon_patient_id = COALESCE($1, anon_patient_id),
		    study_date      = COALESCE($2, study_date),
		    updated_at      = now()
		WHERE id = $3`,
		anonPatientID, studyDate, studyID)
	return err
}

func SetStudyAutoShareURL(ctx context.Context, db *sql.DB, studyID, url string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE studies SET auto_share_url = $1, updated_at = now() WHERE id = $2`,
		url, studyID)
	return err
}

func CreateStudy(ctx context.Context, db *sql.DB, s *Study) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO studies (project_id, upload_session_id, institution_id, study_instance_uid, modality, body_part,
		                     study_description, study_date, anon_patient_id, series_count, instance_count, status, defacing_required,
		                     dicom_store, source, phi_scan_required, phi_scan_status, qc_required, qc_status,
		                     bids_required, bids_status,
		                     classification_required, classification_status,
		                     protocol_required, protocol_status,
		                     export_required, export_status,
		                     pixel_redaction_required, pixel_redaction_status,
		                     analytics_required, analytics_status,
		                     sct_required, sct_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33)
		RETURNING id, created_at, updated_at`,
		s.ProjectID, s.UploadSessionID, s.InstitutionID, s.StudyInstanceUID, s.Modality, s.BodyPart,
		s.StudyDescription, s.StudyDate, s.AnonPatientID, s.SeriesCount, s.InstanceCount, s.Status, s.DefacingRequired,
		s.DicomStore, s.Source, s.PhiScanRequired, s.PhiScanStatus, s.QcRequired, s.QcStatus,
		s.BidsRequired, s.BidsStatus,
		s.ClassificationRequired, s.ClassificationStatus,
		s.ProtocolRequired, s.ProtocolStatus,
		s.ExportRequired, s.ExportStatus,
		s.PixelRedactionRequired, s.PixelRedactionStatus,
		s.AnalyticsRequired, s.AnalyticsStatus,
		s.SctRequired, s.SctStatus).
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
	ProjectID     string
	TenantID      string    // when non-empty, restrict to studies whose project belongs to this tenant (via JOIN through projects.tenant_id)
	Status        string    // received|defacing|clean|defaced|approved|rejected
	Modality      string    // MRI|CT|PET|… (case-insensitive exact match)
	BodyPart      string    // HEAD|CHEST|… (case-insensitive exact match)
	Source        string    // external|internal
	Search        string    // substring match on study_instance_uid or study_description
	SubjectID     string    // exact match on subject_id
	Label         string    // substring match on any study_labels.label value (case-insensitive)
	InstitutionID string    // exact match on institution_id (UUID)
	DateFrom      time.Time // created_at >= DateFrom (zero = no lower bound)
	DateTo        time.Time // created_at <= DateTo   (zero = no upper bound)
	StudyDateFrom string    // study_date >= StudyDateFrom (YYYYMMDD, empty = no lower bound)
	StudyDateTo   string    // study_date <= StudyDateTo   (YYYYMMDD, empty = no upper bound)
	Flagged       *bool     // if non-nil, filter by priority_flag value
	AssignedTo    string    // exact match on assigned_to UUID
	SortBy        string    // created_at|updated_at|status|modality|body_part|source|instance_count|study_date (default: created_at)
	SortDir       string    // asc|desc (default: desc)
}

func studyWhere(f StudyFilters) (string, []any) {
	// Always exclude soft-deleted studies from normal listings.
	clauses := []string{"deleted_at IS NULL"}
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
	if f.Label != "" {
		clauses = append(clauses, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM study_labels sl WHERE sl.study_id = studies.id AND sl.label ILIKE $%d)`, n))
		args = append(args, "%"+f.Label+"%")
		n++
	}
	if f.InstitutionID != "" {
		clauses = append(clauses, fmt.Sprintf(`institution_id = $%d`, n))
		args = append(args, f.InstitutionID)
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
	if f.StudyDateFrom != "" {
		clauses = append(clauses, fmt.Sprintf(`study_date >= $%d`, n))
		args = append(args, f.StudyDateFrom)
		n++
	}
	if f.StudyDateTo != "" {
		clauses = append(clauses, fmt.Sprintf(`study_date <= $%d`, n))
		args = append(args, f.StudyDateTo)
		n++
	}
	if f.Flagged != nil {
		clauses = append(clauses, fmt.Sprintf(`priority_flag = $%d`, n))
		args = append(args, *f.Flagged)
		n++
	}
	if f.AssignedTo != "" {
		clauses = append(clauses, fmt.Sprintf(`assigned_to = $%d`, n))
		args = append(args, f.AssignedTo)
		n++
	}
	if f.TenantID != "" {
		// Studies inherit their tenant from the parent project; we filter via
		// a sub-select rather than a JOIN so the rest of the query (sort, count,
		// pagination) keeps using the simple `FROM studies` form.
		clauses = append(clauses, fmt.Sprintf(
			`project_id IN (SELECT id FROM projects WHERE tenant_id = $%d)`, n))
		args = append(args, f.TenantID)
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

// allowedStudySortCols maps safe sort_by values to their SQL column names.
var allowedStudySortCols = map[string]string{
	"created_at":     "created_at",
	"updated_at":     "updated_at",
	"status":         "status",
	"modality":       "modality",
	"body_part":      "body_part",
	"source":         "source",
	"instance_count": "instance_count",
	"assigned_at":    "assigned_at",
	"study_date":     "study_date",
}

func ListStudies(ctx context.Context, db *sql.DB, f StudyFilters, limit, offset int) ([]Study, error) {
	where, args := studyWhere(f)
	argN := len(args) + 1

	sortCol := "created_at"
	if col, ok := allowedStudySortCols[f.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if f.SortDir == "asc" {
		sortDir = "ASC"
	}
	query := `SELECT` + studyColumns + ` FROM studies` + where + ` ORDER BY ` + sortCol + ` ` + sortDir

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

func UpdateStudySizeBytes(ctx context.Context, db *sql.DB, id string, sizeBytes int64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET study_size_bytes = $1, updated_at = now() WHERE id = $2`, sizeBytes, id)
	return err
}

func UpdateStudyStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}

// UpdateStudyRejected sets status to 'rejected' and stores an optional reason.
func UpdateStudyRejected(ctx context.Context, db *sql.DB, id, reason string) error {
	var reasonVal *string
	if reason != "" {
		reasonVal = &reason
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET status = 'rejected', rejection_reason = $1, updated_at = now()
		WHERE id = $2`, reasonVal, id)
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

// AssignStudy assigns a study to an admin user for review.
func AssignStudy(ctx context.Context, db *sql.DB, studyID, userID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET assigned_to = $1, assigned_at = now(), updated_at = now()
		WHERE id = $2`, userID, studyID)
	return err
}

// UnassignStudy removes the assignment from a study.
func UnassignStudy(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET assigned_to = NULL, assigned_at = NULL, updated_at = now()
		WHERE id = $1`, studyID)
	return err
}

// SetPriorityFlag sets or clears the priority_flag on a study for high-priority triage.
func SetPriorityFlag(ctx context.Context, db *sql.DB, id string, flagged bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET priority_flag = $1, updated_at = now() WHERE id = $2`, flagged, id)
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

// ReassignStudyProject moves a study to a different project.
func ReassignStudyProject(ctx context.Context, db *sql.DB, studyID, newProjectID string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE studies SET project_id = $1, updated_at = now() WHERE id = $2`,
		newProjectID, studyID)
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

// StudyStub is a lightweight study reference for cross-study queries.
type StudyStub struct {
	ID               string  `json:"id"`
	StudyInstanceUID string  `json:"study_instance_uid"`
	SubjectID        *string `json:"subject_id,omitempty"`
	StudyDate        *string `json:"study_date,omitempty"`
}

// ListStudiesBySubject returns study stubs for a subject, optionally filtered by project.
func ListStudiesBySubject(ctx context.Context, db *sql.DB, subjectID, projectID string) ([]StudyStub, error) {
	q := `SELECT id, study_instance_uid, subject_id, study_date FROM studies WHERE subject_id = $1`
	args := []any{subjectID}
	if projectID != "" {
		q += ` AND project_id = $2`
		args = append(args, projectID)
	}
	q += ` ORDER BY created_at`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyStub
	for rows.Next() {
		var s StudyStub
		if err := rows.Scan(&s.ID, &s.StudyInstanceUID, &s.SubjectID, &s.StudyDate); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListStudiesByProject returns study stubs for a project.
func ListStudiesByProject(ctx context.Context, db *sql.DB, projectID string) ([]StudyStub, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_instance_uid, subject_id, study_date
		FROM studies WHERE project_id = $1
		ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyStub
	for rows.Next() {
		var s StudyStub
		if err := rows.Scan(&s.ID, &s.StudyInstanceUID, &s.SubjectID, &s.StudyDate); err != nil {
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

// SetPixelRedactionRequired sets the pixel_redaction_required flag and initialises pixel_redaction_status to "pending".
func SetPixelRedactionRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET pixel_redaction_required = $1, pixel_redaction_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdatePixelRedactionStatus sets the pixel_redaction_status field on a study.
func UpdatePixelRedactionStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET pixel_redaction_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// ClaimPixelRedaction atomically claims pixel redaction dispatch.
func ClaimPixelRedaction(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET pixel_redaction_status = 'redacting', updated_at = now()
		WHERE id = $1 AND pixel_redaction_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetAnalyticsRequired sets the analytics_required flag and initialises analytics_status to "pending".
func SetAnalyticsRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET analytics_required = $1, analytics_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateAnalyticsStatus sets the analytics_status field on a study.
func UpdateAnalyticsStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET analytics_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// ClaimAnalytics atomically claims analytics dispatch.
func ClaimAnalytics(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET analytics_status = 'analyzing', updated_at = now()
		WHERE id = $1 AND analytics_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetSctRequired sets the sct_required flag and initialises sct_status to "pending".
func SetSctRequired(ctx context.Context, db *sql.DB, id string, required bool) error {
	status := ""
	if required {
		status = "pending"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET sct_required = $1, sct_status = $2, updated_at = now()
		WHERE id = $3`, required, status, id)
	return err
}

// UpdateSctStatus sets the sct_status field on a study.
func UpdateSctStatus(ctx context.Context, db *sql.DB, id, status string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies SET sct_status = $1, updated_at = now()
		WHERE id = $2`, status, id)
	return err
}

// ClaimSct atomically claims SCT dispatch.
func ClaimSct(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET sct_status = 'analyzing', updated_at = now()
		WHERE id = $1 AND sct_status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// StudyStatusCounts holds per-status study counts for the dashboard overview.
type StudyStatusCounts struct {
	Received int `json:"received"`
	Defacing int `json:"defacing"`
	Clean    int `json:"clean"`
	Defaced  int `json:"defaced"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
	Total    int `json:"total"`
}

// GetStudyStatusCounts returns a snapshot count of studies by status.
// An optional projectID filters to a single project.
func GetStudyStatusCounts(ctx context.Context, db *sql.DB, projectID ...string) (StudyStatusCounts, error) {
	where := ""
	var args []any
	if len(projectID) > 0 && projectID[0] != "" {
		where = " WHERE project_id = $1"
		args = append(args, projectID[0])
	}
	rows, err := db.QueryContext(ctx,
		`SELECT status, count(*) FROM studies`+where+` GROUP BY status`, args...)
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

func GetStudyStatusCountsForScope(ctx context.Context, db *sql.DB, projectID, institutionID string) (StudyStatusCounts, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT status, count(*) FROM studies WHERE project_id = $1::uuid AND ($2 = '' OR institution_id = NULLIF($2, '')::uuid) GROUP BY status`,
		projectID, institutionID)
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
	Modality string `json:"modality"`
	BodyPart string `json:"body_part"`
	Count    int    `json:"count"`
}

// GetStudyBreakdown returns study counts grouped by (modality, body_part).
// Empty modality/body_part values are normalised to the empty string.
// An optional projectID filters to a single project.
func GetStudyBreakdown(ctx context.Context, db *sql.DB, projectID ...string) ([]BreakdownRow, error) {
	where := ""
	var args []any
	if len(projectID) > 0 && projectID[0] != "" {
		where = " WHERE project_id = $1"
		args = append(args, projectID[0])
	}
	rows, err := db.QueryContext(ctx, `
		SELECT coalesce(modality, ''), coalesce(body_part, ''), count(*)
		FROM studies`+where+`
		GROUP BY modality, body_part
		ORDER BY count(*) DESC`, args...)
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

func GetStudyBreakdownForScope(ctx context.Context, db *sql.DB, projectID, institutionID string) ([]BreakdownRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT coalesce(modality, ''), coalesce(body_part, ''), count(*)
		FROM studies
		WHERE project_id = $1::uuid
		  AND ($2 = '' OR institution_id = NULLIF($2, '')::uuid)
		GROUP BY modality, body_part
		ORDER BY count(*) DESC`, projectID, institutionID)
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

// StorageStats summarises DICOM file counts across studies by store type.
type StorageStats struct {
	RawFileCount   int    `json:"raw_file_count"`   // files in dicom_store='raw'
	CleanFileCount int    `json:"clean_file_count"` // files in dicom_store='clean'
	TotalFileCount int    `json:"total_file_count"`
	TotalStudies   int    `json:"total_studies"`
	TotalSizeBytes int64  `json:"total_size_bytes"` // sum of study_size_bytes across all studies
	GeneratedAt    string `json:"generated_at"`
}

// GetStorageStats returns aggregate DICOM file counts derived from the studies table.
// An optional projectID filters to a single project.
func GetStorageStats(ctx context.Context, db *sql.DB, projectID ...string) (*StorageStats, error) {
	where := ""
	var args []any
	if len(projectID) > 0 && projectID[0] != "" {
		where = " WHERE project_id = $1"
		args = append(args, projectID[0])
	}
	row := db.QueryRowContext(ctx, `
		SELECT
		  coalesce(sum(instance_count) FILTER (WHERE dicom_store = 'raw'),   0)::int,
		  coalesce(sum(instance_count) FILTER (WHERE dicom_store = 'clean'), 0)::int,
		  coalesce(sum(instance_count), 0)::int,
		  count(*)::int,
		  coalesce(sum(study_size_bytes), 0)::bigint
		FROM studies`+where, args...)
	var s StorageStats
	if err := row.Scan(&s.RawFileCount, &s.CleanFileCount, &s.TotalFileCount, &s.TotalStudies, &s.TotalSizeBytes); err != nil {
		return nil, err
	}
	return &s, nil
}

func GetStorageStatsForScope(ctx context.Context, db *sql.DB, projectID, institutionID string) (*StorageStats, error) {
	row := db.QueryRowContext(ctx, `
		SELECT
		  coalesce(sum(instance_count) FILTER (WHERE dicom_store = 'raw'),   0)::int,
		  coalesce(sum(instance_count) FILTER (WHERE dicom_store = 'clean'), 0)::int,
		  coalesce(sum(instance_count), 0)::int,
		  count(*)::int,
		  coalesce(sum(study_size_bytes), 0)::bigint
		FROM studies
		WHERE project_id = $1::uuid
		  AND ($2 = '' OR institution_id = NULLIF($2, '')::uuid)`, projectID, institutionID)
	var s StorageStats
	if err := row.Scan(&s.RawFileCount, &s.CleanFileCount, &s.TotalFileCount, &s.TotalStudies, &s.TotalSizeBytes); err != nil {
		return nil, err
	}
	return &s, nil
}

// TimelineDay holds ingestion counts for a single UTC date.
type TimelineDay struct {
	Date     string `json:"date"`     // YYYY-MM-DD
	Received int    `json:"received"` // studies created that day
	Approved int    `json:"approved"` // studies approved that day
}

// InstitutionAttributionProjectRow is one project row in the attribution gap report.
type InstitutionAttributionProjectRow struct {
	ProjectID         string  `json:"project_id"`
	ProjectSlug       string  `json:"project_slug"`
	ProjectName       string  `json:"project_name"`
	TotalStudies      int     `json:"total_studies"`
	UnattributedCount int     `json:"unattributed_count"`
	UnattributedPct   float64 `json:"unattributed_pct"`
}

// InstitutionAttributionStats reports missing institution attribution in restricted projects.
type InstitutionAttributionStats struct {
	Days              int                                `json:"days"`
	TotalRestricted   int                                `json:"total_restricted_studies"`
	UnattributedTotal int                                `json:"unattributed_studies"`
	UnattributedPct   float64                            `json:"unattributed_pct"`
	Projects          []InstitutionAttributionProjectRow `json:"projects"`
}

// GetStudyTimeline returns daily ingestion counts for the last `days` calendar days.
// An optional projectID filters to a single project.
func GetStudyTimeline(ctx context.Context, db *sql.DB, days int, projectID ...string) ([]TimelineDay, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	where := ""
	var args []any
	args = append(args, days)
	if len(projectID) > 0 && projectID[0] != "" {
		where = " AND project_id = $2"
		args = append(args, projectID[0])
	}
	rows, err := db.QueryContext(ctx, `
		SELECT
		  to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
		  count(*) FILTER (WHERE status <> 'approved')  +
		    count(*) FILTER (WHERE status = 'approved')  AS received,
		  count(*) FILTER (WHERE status = 'approved')   AS approved
		FROM studies
		WHERE created_at >= now() - ($1 * INTERVAL '1 day')`+where+`
		GROUP BY day
		ORDER BY day`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TimelineDay
	for rows.Next() {
		var d TimelineDay
		if err := rows.Scan(&d.Date, &d.Received, &d.Approved); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if result == nil {
		result = []TimelineDay{}
	}
	return result, rows.Err()
}

func GetStudyTimelineForScope(ctx context.Context, db *sql.DB, days int, projectID, institutionID string) ([]TimelineDay, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	rows, err := db.QueryContext(ctx, `
		SELECT
		  to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
		  count(*) FILTER (WHERE status <> 'approved')  +
		    count(*) FILTER (WHERE status = 'approved')  AS received,
		  count(*) FILTER (WHERE status = 'approved')   AS approved
		FROM studies
		WHERE created_at >= now() - ($1 * INTERVAL '1 day')
		  AND project_id = $2::uuid
		  AND ($3 = '' OR institution_id = NULLIF($3, '')::uuid)
		GROUP BY day
		ORDER BY day`, days, projectID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TimelineDay
	for rows.Next() {
		var d TimelineDay
		if err := rows.Scan(&d.Date, &d.Received, &d.Approved); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if result == nil {
		result = []TimelineDay{}
	}
	return result, rows.Err()
}

// GetInstitutionAttributionStats returns unattributed study counts (institution_id IS NULL)
// across restricted projects for the last `days` days.
// Optional projectID narrows to one project; optional institutionID applies site-scope filter.
func GetInstitutionAttributionStats(ctx context.Context, db *sql.DB, days int, projectID, institutionID string) (*InstitutionAttributionStats, error) {
	if days <= 0 || days > 365 {
		days = 7
	}

	rows, err := db.QueryContext(ctx, `
		SELECT
			s.project_id,
			p.slug,
			p.name,
			count(*)::int AS total_studies,
			count(*) FILTER (WHERE s.institution_id IS NULL)::int AS unattributed_count
		FROM studies s
		JOIN projects p ON p.id = s.project_id
		WHERE p.restricted = true
		  AND s.created_at >= now() - ($1 * INTERVAL '1 day')
		  AND ($2 = '' OR s.project_id = $2::uuid)
		  AND ($3 = '' OR s.institution_id = NULLIF($3, '')::uuid)
		GROUP BY s.project_id, p.slug, p.name
		ORDER BY unattributed_count DESC, total_studies DESC, p.name ASC`, days, projectID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := &InstitutionAttributionStats{Days: days, Projects: []InstitutionAttributionProjectRow{}}
	for rows.Next() {
		var row InstitutionAttributionProjectRow
		if err := rows.Scan(&row.ProjectID, &row.ProjectSlug, &row.ProjectName, &row.TotalStudies, &row.UnattributedCount); err != nil {
			return nil, err
		}
		if row.TotalStudies > 0 {
			row.UnattributedPct = (float64(row.UnattributedCount) / float64(row.TotalStudies)) * 100
		}
		out.TotalRestricted += row.TotalStudies
		out.UnattributedTotal += row.UnattributedCount
		out.Projects = append(out.Projects, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out.TotalRestricted > 0 {
		out.UnattributedPct = (float64(out.UnattributedTotal) / float64(out.TotalRestricted)) * 100
	}
	return out, nil
}

// ExpiringStudyRow extends Study with retention metadata for the expiry endpoint.
type ExpiringStudyRow struct {
	Study
	RetentionDays   int    `json:"retention_days"`
	ExpiresAt       string `json:"expires_at"`
	DaysUntilExpiry int    `json:"days_until_expiry"`
}

// GetExpiringStudies returns approved studies that will be soft-expired within
// the next `days` days, based on their project's retention_days setting.
// Only projects with a non-null retention_days are included.
// Results are ordered by expiry date ascending (soonest first).
func GetExpiringStudies(ctx context.Context, db *sql.DB, projectID string, days, limit int) ([]ExpiringStudyRow, error) {
	query := `
		SELECT
		  s.id, s.project_id, s.upload_session_id, s.institution_id, s.study_instance_uid,
		  s.modality, s.body_part, s.study_description, s.series_count, s.instance_count,
		  s.status, s.defacing_required, s.dicom_store, s.source,
		  s.phi_scan_required, s.phi_scan_status, s.qc_required, s.qc_status,
		  s.bids_required, s.bids_status, s.classification_required, s.classification_status,
		  s.protocol_required, s.protocol_status, s.export_required, s.export_status,
		  s.pixel_redaction_required, s.pixel_redaction_status,
		  s.deface_qa_score, s.subject_id, s.rejection_reason, s.study_size_bytes,
		  s.priority_flag, s.assigned_to, s.assigned_at, s.created_at, s.updated_at,
		  p.retention_days,
		  (s.created_at + (p.retention_days * INTERVAL '1 day'))                              AS expires_at,
		  GREATEST(0, EXTRACT(DAY FROM (s.created_at + (p.retention_days * INTERVAL '1 day') - now()))::int) AS days_until_expiry
		FROM studies s
		JOIN projects p ON p.id = s.project_id
		WHERE s.status = 'approved'
		  AND p.retention_days IS NOT NULL
		  AND s.created_at + (p.retention_days * INTERVAL '1 day') > now()
		  AND s.created_at + (p.retention_days * INTERVAL '1 day') <= now() + ($1 * INTERVAL '1 day')
		  AND ($2 = '' OR s.project_id = $2::uuid)
		ORDER BY expires_at ASC
		LIMIT $3`

	rows, err := db.QueryContext(ctx, query, days, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExpiringStudyRow
	for rows.Next() {
		var row ExpiringStudyRow
		var expiresAt time.Time
		if err := func() error {
			return rows.Scan(
				&row.ID, &row.ProjectID, &row.UploadSessionID, &row.InstitutionID, &row.StudyInstanceUID,
				&row.Modality, &row.BodyPart, &row.StudyDescription, &row.SeriesCount, &row.InstanceCount,
				&row.Status, &row.DefacingRequired, &row.DicomStore, &row.Source,
				&row.PhiScanRequired, &row.PhiScanStatus, &row.QcRequired, &row.QcStatus,
				&row.BidsRequired, &row.BidsStatus,
				&row.ClassificationRequired, &row.ClassificationStatus,
				&row.ProtocolRequired, &row.ProtocolStatus,
				&row.ExportRequired, &row.ExportStatus,
				&row.PixelRedactionRequired, &row.PixelRedactionStatus,
				&row.DefaceQaScore,
				&row.SubjectID,
				&row.RejectionReason,
				&row.StudySizeBytes,
				&row.PriorityFlag,
				&row.AssignedTo,
				&row.AssignedAt,
				&row.CreatedAt, &row.UpdatedAt,
				&row.RetentionDays,
				&expiresAt,
				&row.DaysUntilExpiry,
			)
		}(); err != nil {
			return nil, err
		}
		row.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

// SoftDeleteStudy sets deleted_at on a study without removing it from the database.
// The study is excluded from normal listings but can be retrieved via ListDeletedStudies
// or restored with RestoreStudy.
func SoftDeleteStudy(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RestoreStudy clears deleted_at on a soft-deleted study, making it visible
// in normal listings again.
func RestoreStudy(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE studies SET deleted_at = NULL, updated_at = now()
		WHERE id = $1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListDeletedStudies returns soft-deleted studies, newest first.
// Optional projectID scopes the result to one project.
func ListDeletedStudies(ctx context.Context, db *sql.DB, projectID string, limit, offset int) ([]Study, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	where := "WHERE deleted_at IS NOT NULL"
	args := []any{}
	n := 1
	if projectID != "" {
		where += fmt.Sprintf(" AND project_id = $%d", n)
		args = append(args, projectID)
		n++
	}

	var total int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM studies `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offsetArg := n
	limitArg := n + 1
	args = append(args, offset, limit)
	rows, err := db.QueryContext(ctx,
		`SELECT `+studyColumns+` FROM studies `+where+
			fmt.Sprintf(` ORDER BY deleted_at DESC OFFSET $%d LIMIT $%d`, offsetArg, limitArg),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var studies []Study
	for rows.Next() {
		var s Study
		if err := scanStudy(rows, &s); err != nil {
			return nil, 0, err
		}
		studies = append(studies, s)
	}
	if studies == nil {
		studies = []Study{}
	}
	return studies, total, rows.Err()
}

// DeleteStudy permanently removes a study and all its dependent rows.
// export_shares lacks ON DELETE CASCADE, so it is cleared explicitly first.
// All other child tables (study_labels, study_sla_alerts, study_series,
// routing_rule_log, etc.) cascade automatically.
func DeleteStudy(ctx context.Context, db *sql.DB, id string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM export_shares WHERE study_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM studies WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}
