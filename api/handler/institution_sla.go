package handler

import (
	"net/http"
	"strconv"
)

// InstitutionSLARow holds turnaround metrics for an institution.
type InstitutionSLARow struct {
	InstitutionID   string   `json:"institution_id"`
	InstitutionName string   `json:"institution_name"`
	TotalStudies    int      `json:"total_studies"`
	ApprovedStudies int      `json:"approved_studies"`
	AvgHoursToApproval *float64 `json:"avg_hours_to_approval,omitempty"`
}

// GetInstitutionSLA GET /api/institutions/{id}/sla
// Returns turnaround metrics for an institution.
func (s *Server) GetInstitutionSLA(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	var row InstitutionSLARow
	row.InstitutionID = instID

	// Get institution name.
	err := s.db.QueryRowContext(r.Context(), `SELECT name FROM institutions WHERE id = $1`, instID).Scan(&row.InstitutionName)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}

	// Get study counts and avg turnaround.
	err = s.db.QueryRowContext(r.Context(), `
		SELECT
			count(*) AS total,
			count(*) FILTER (WHERE status = 'approved') AS approved,
			avg(EXTRACT(EPOCH FROM (updated_at - created_at)) / 3600) FILTER (WHERE status = 'approved') AS avg_hours
		FROM studies
		WHERE institution_id = $1
		  AND created_at >= now() - ($2 * INTERVAL '1 day')`,
		instID, days).Scan(&row.TotalStudies, &row.ApprovedStudies, &row.AvgHoursToApproval)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"institution": row,
		"days":        days,
	})
}
