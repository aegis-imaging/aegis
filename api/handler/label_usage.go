package handler

import (
	"net/http"
)

type labelUsageRow struct {
	Label      string `json:"label"`
	Count      int    `json:"count"`
	StudyCount int    `json:"study_count"` // distinct studies that carry this label
}

type labelUsageResponse struct {
	ProjectID   string          `json:"project_id,omitempty"`
	TotalLabels int             `json:"total_labels"`
	Labels      []labelUsageRow `json:"labels"`
}

// GetLabelUsage returns aggregated label usage stats across all studies.
//
// GET /api/stats/label-usage?project_id=<uuid>
func (s *Server) GetLabelUsage(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	query := `
		SELECT
			sl.label,
			COUNT(*) AS cnt,
			COUNT(DISTINCT sl.study_id) AS study_count
		FROM study_labels sl
		INNER JOIN studies s ON s.id = sl.study_id
		WHERE s.deleted_at IS NULL`
	args := []any{}
	if projectID != "" {
		query += " AND s.project_id = $1"
		args = append(args, projectID)
	}
	query += " GROUP BY sl.label ORDER BY cnt DESC"

	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query label usage")
		return
	}
	defer rows.Close()

	labels := make([]labelUsageRow, 0)
	for rows.Next() {
		var row labelUsageRow
		if err := rows.Scan(&row.Label, &row.Count, &row.StudyCount); err != nil {
			continue
		}
		labels = append(labels, row)
	}

	resp := labelUsageResponse{
		TotalLabels: len(labels),
		Labels:      labels,
	}
	if projectID != "" {
		resp.ProjectID = projectID
	}

	s.writeJSON(w, http.StatusOK, resp)
}
