package handler

import (
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type institutionBreakdownRow struct {
	InstitutionID   *string `json:"institution_id"`
	InstitutionName *string `json:"institution_name"`
	StudyCount      int     `json:"study_count"`
	Approved        int     `json:"approved"`
	Rejected        int     `json:"rejected"`
	Pending         int     `json:"pending"`
}

type institutionBreakdownResponse struct {
	ProjectID   string                    `json:"project_id"`
	GeneratedAt string                    `json:"generated_at"`
	Rows        []institutionBreakdownRow `json:"rows"`
}

// GetInstitutionBreakdown returns study counts grouped by institution for a project.
//
// GET /api/projects/{id}/institution-breakdown
func (s *Server) GetInstitutionBreakdown(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project id")
		return
	}
	if _, ok := s.requireProjectReadAccess(w, r, projectID); !ok {
		return
	}

	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT
			s.institution_id,
			i.name                                                           AS institution_name,
			COUNT(*)                                                         AS study_count,
			COUNT(*) FILTER (WHERE s.status = 'approved')                   AS approved,
			COUNT(*) FILTER (WHERE s.status = 'rejected')                   AS rejected,
			COUNT(*) FILTER (WHERE s.status NOT IN ('approved','rejected','expired')) AS pending
		FROM studies s
		LEFT JOIN institutions i ON i.id = s.institution_id
		WHERE s.project_id = $1
		  AND s.deleted_at IS NULL
		GROUP BY s.institution_id, i.name
		ORDER BY study_count DESC`, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query institution breakdown")
		return
	}
	defer rows.Close()

	result := make([]institutionBreakdownRow, 0)
	for rows.Next() {
		var row institutionBreakdownRow
		if err := rows.Scan(
			&row.InstitutionID,
			&row.InstitutionName,
			&row.StudyCount,
			&row.Approved,
			&row.Rejected,
			&row.Pending,
		); err != nil {
			continue
		}
		result = append(result, row)
	}

	s.writeJSON(w, http.StatusOK, institutionBreakdownResponse{
		ProjectID:   projectID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Rows:        result,
	})
}
