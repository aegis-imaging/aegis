package handler

import (
	"net/http"
	"strconv"
	"time"
)

type modalityTrendDay struct {
	Day      string         `json:"day"`
	Counts   map[string]int `json:"counts"` // modality → count
	Total    int            `json:"total"`
}

type modalityTrendResponse struct {
	GeneratedAt string                   `json:"generated_at"`
	PeriodDays  int                      `json:"period_days"`
	ProjectID   string                   `json:"project_id,omitempty"`
	Totals      map[string]int           `json:"totals"` // modality → total count
	Days        []modalityTrendDay       `json:"days"`
}

// GetModalityTrend returns daily study counts grouped by modality over the last N days.
//
// GET /api/stats/modality-trend?days=30&project_id=<uuid>
func (s *Server) GetModalityTrend(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	projectID := r.URL.Query().Get("project_id")
	since := time.Now().UTC().AddDate(0, 0, -days)

	// ── Daily breakdown by modality ────────────────────────────────────────────
	query := `
		SELECT
			DATE(created_at AT TIME ZONE 'UTC') AS day,
			COALESCE(NULLIF(modality, ''), 'unknown') AS modality,
			COUNT(*) AS cnt
		FROM studies
		WHERE created_at >= $1
		  AND deleted_at IS NULL`
	args := []any{since}
	if projectID != "" {
		query += " AND project_id = $2"
		args = append(args, projectID)
	}
	query += " GROUP BY day, modality ORDER BY day ASC, cnt DESC"

	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query modality trend")
		return
	}
	defer rows.Close()

	// Build day map
	dayMap := map[string]*modalityTrendDay{}
	totals := map[string]int{}

	for rows.Next() {
		var day, modality string
		var cnt int
		if err := rows.Scan(&day, &modality, &cnt); err != nil {
			continue
		}
		if _, ok := dayMap[day]; !ok {
			dayMap[day] = &modalityTrendDay{Day: day, Counts: map[string]int{}}
		}
		dayMap[day].Counts[modality] += cnt
		dayMap[day].Total += cnt
		totals[modality] += cnt
	}

	// Collect ordered days
	dayRows := make([]modalityTrendDay, 0, len(dayMap))
	for d := since; !d.After(time.Now().UTC()); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if entry, ok := dayMap[key]; ok {
			dayRows = append(dayRows, *entry)
		}
	}

	resp := modalityTrendResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		PeriodDays:  days,
		Totals:      totals,
		Days:        dayRows,
	}
	if projectID != "" {
		resp.ProjectID = projectID
	}

	s.writeJSON(w, http.StatusOK, resp)
}
