package handler

import (
	"net/http"
	"strconv"
	"time"
)

// AdminUserActivityRow holds per-user action counts.
type AdminUserActivityRow struct {
	Email       string     `json:"email"`
	ActionCount int        `json:"action_count"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

// GetAdminUserActivity GET /api/admin-users/activity
// Returns per-user action counts from the audit trail over the last N days.
func (s *Server) GetAdminUserActivity(w http.ResponseWriter, r *http.Request) {
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT actor, count(*) AS action_count, max(created_at) AS last_seen_at
		FROM audit_trail
		WHERE actor <> '' AND created_at >= now() - ($1 * INTERVAL '1 day')
		GROUP BY actor
		ORDER BY action_count DESC
		LIMIT $2`, days, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var result []AdminUserActivityRow
	for rows.Next() {
		var r AdminUserActivityRow
		if err := rows.Scan(&r.Email, &r.ActionCount, &r.LastSeenAt); err != nil {
			s.writeError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "rows error")
		return
	}
	if result == nil {
		result = []AdminUserActivityRow{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"users": result,
		"days":  days,
		"total": len(result),
	})
}
