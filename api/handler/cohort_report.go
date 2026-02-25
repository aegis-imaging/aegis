package handler

import (
	"net/http"
	"time"
)

// cohortSubjectRow represents aggregate data for a single research subject.
type cohortSubjectRow struct {
	SubjectID              string    `json:"subject_id"`
	StudyCount             int       `json:"study_count"`
	ApprovedCount          int       `json:"approved_count"`
	RejectedCount          int       `json:"rejected_count"`
	PendingCount           int       `json:"pending_count"` // all non-terminal states
	Modalities             []string  `json:"modalities"`     // distinct, sorted
	EarliestStudyAt        time.Time `json:"earliest_study_at"`
	LatestStudyAt          time.Time `json:"latest_study_at"`
	AllApproved            bool      `json:"all_approved"`
	HasDefaced             bool      `json:"has_defaced"`
	HasExported            bool      `json:"has_exported"`
}

// cohortReportResponse is the full response for GET /api/projects/{id}/cohort-report.
type cohortReportResponse struct {
	ProjectID             string             `json:"project_id"`
	GeneratedAt           string             `json:"generated_at"`
	TotalSubjects         int                `json:"total_subjects"`
	SubjectsMultiStudy    int                `json:"subjects_multi_study"`   // subject with >= 2 studies
	TotalStudiesWithSubject int              `json:"total_studies_with_subject"`
	ModalityCoverage      map[string]int     `json:"modality_coverage"`      // modality → subject count
	Subjects              []cohortSubjectRow `json:"subjects"`
}

// GetCohortReport returns a per-subject cohort summary for a project,
// aggregating study count, modalities, approval status, and timeline per subject.
//
// GET /api/projects/{id}/cohort-report?limit=200
func (s *Server) GetCohortReport(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "project_id required")
		return
	}

	ctx := r.Context()

	// Verify the project exists.
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)`, projectID,
	).Scan(&exists); err != nil || !exists {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Per-subject aggregate query.
	// terminal = approved | rejected | exported | expired
	// pending  = all other statuses
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			subject_id,
			COUNT(*)                                       AS study_count,
			COUNT(*) FILTER (WHERE status = 'approved')   AS approved_count,
			COUNT(*) FILTER (WHERE status = 'rejected')   AS rejected_count,
			COUNT(*) FILTER (WHERE status NOT IN ('approved','rejected','exported','expired'))
			                                               AS pending_count,
			ARRAY_AGG(DISTINCT modality ORDER BY modality)
			                FILTER (WHERE modality <> '') AS modalities,
			MIN(created_at)                                AS earliest_study_at,
			MAX(created_at)                                AS latest_study_at,
			BOOL_AND(status = 'approved')                  AS all_approved,
			BOOL_OR(dicom_store = 'clean')                 AS has_defaced,
			BOOL_OR(export_status = 'exported')            AS has_exported
		FROM studies
		WHERE project_id = $1
		  AND subject_id IS NOT NULL
		  AND subject_id <> ''
		GROUP BY subject_id
		ORDER BY MAX(created_at) DESC
		LIMIT 500`,
		projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query cohort data: "+err.Error())
		return
	}
	defer rows.Close()

	var subjects []cohortSubjectRow
	modalityCoverage := map[string]int{}

	for rows.Next() {
		var r cohortSubjectRow
		var modalities []string // postgresql array → Go slice via pq
		if err := rows.Scan(
			&r.SubjectID, &r.StudyCount, &r.ApprovedCount, &r.RejectedCount, &r.PendingCount,
			nullableStringSlice(&modalities),
			&r.EarliestStudyAt, &r.LatestStudyAt,
			&r.AllApproved, &r.HasDefaced, &r.HasExported,
		); err != nil {
			s.writeError(w, http.StatusInternalServerError, "scan error: "+err.Error())
			return
		}
		r.Modalities = modalities
		for _, m := range modalities {
			modalityCoverage[m]++
		}
		subjects = append(subjects, r)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "rows error: "+err.Error())
		return
	}

	// Count subjects with multiple studies.
	multiStudy := 0
	totalWithSubject := 0
	for _, sub := range subjects {
		totalWithSubject += sub.StudyCount
		if sub.StudyCount >= 2 {
			multiStudy++
		}
	}

	if subjects == nil {
		subjects = []cohortSubjectRow{}
	}

	s.writeJSON(w, http.StatusOK, cohortReportResponse{
		ProjectID:              projectID,
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		TotalSubjects:          len(subjects),
		SubjectsMultiStudy:     multiStudy,
		TotalStudiesWithSubject: totalWithSubject,
		ModalityCoverage:       modalityCoverage,
		Subjects:               subjects,
	})
}

// nullableStringSlice returns a scanner that reads a PostgreSQL text[] or null into *[]string.
// Null → empty slice.
func nullableStringSlice(dest *[]string) interface{ Scan(any) error } {
	return &pgStringArray{dest}
}

type pgStringArray struct{ dest *[]string }

func (p *pgStringArray) Scan(v any) error {
	if v == nil {
		*p.dest = nil
		return nil
	}
	// PostgreSQL returns array as []byte in the form {"a","b","c"} or {a,b,c}.
	b, ok := v.([]byte)
	if !ok {
		// Could be string in some drivers.
		if s, ok2 := v.(string); ok2 {
			b = []byte(s)
		} else {
			*p.dest = nil
			return nil
		}
	}
	s := string(b)
	if s == "{}" || s == "" {
		*p.dest = []string{}
		return nil
	}
	// Strip braces and split on comma, unquoting PostgreSQL-style quoted strings.
	s = s[1 : len(s)-1] // remove { }
	*p.dest = splitPGArray(s)
	return nil
}

// splitPGArray splits a PostgreSQL array literal (without braces) on commas,
// handling double-quoted elements (e.g. `"MRI","CT"` or `MRI,CT`).
func splitPGArray(s string) []string {
	var result []string
	var current []byte
	inQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ',' && !inQuote:
			result = append(result, string(current))
			current = current[:0]
		default:
			current = append(current, ch)
		}
	}
	if len(current) > 0 || len(s) > 0 {
		result = append(result, string(current))
	}
	return result
}
