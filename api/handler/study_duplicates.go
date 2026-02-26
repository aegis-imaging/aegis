package handler

import (
	"net/http"
)

// DuplicateGroup represents studies sharing the same StudyInstanceUID.
type DuplicateGroup struct {
	StudyInstanceUID string `json:"study_instance_uid"`
	Count            int    `json:"count"`
}

// GetStudyDuplicates GET /api/studies/duplicates
// Returns StudyInstanceUIDs that appear more than once (potential duplicates).
func (s *Server) GetStudyDuplicates(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	where := "WHERE deleted_at IS NULL"
	args := []any{}
	if projectID != "" {
		where += " AND project_id = $1"
		args = append(args, projectID)
	}

	query := `SELECT study_instance_uid, count(*) AS cnt
	          FROM studies ` + where + `
	          GROUP BY study_instance_uid
	          HAVING count(*) > 1
	          ORDER BY cnt DESC
	          LIMIT 200`

	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var groups []DuplicateGroup
	for rows.Next() {
		var g DuplicateGroup
		if err := rows.Scan(&g.StudyInstanceUID, &g.Count); err != nil {
			s.writeError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "rows error")
		return
	}
	if groups == nil {
		groups = []DuplicateGroup{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"duplicates": groups,
		"total":      len(groups),
	})
}
