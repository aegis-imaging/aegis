package handler

import (
	"net/http"
	"strconv"
	"time"
)

type sourceTrendDay struct {
	Day      string `json:"day"`
	External int    `json:"external"`
	Internal int    `json:"internal"`
	Total    int    `json:"total"`
}

type sourceTrendTotals struct {
	External int `json:"external"`
	Internal int `json:"internal"`
	Total    int `json:"total"`
}

type sourceTrendResponse struct {
	GeneratedAt string           `json:"generated_at"`
	PeriodDays  int              `json:"period_days"`
	ProjectID   string           `json:"project_id,omitempty"`
	Totals      sourceTrendTotals `json:"totals"`
	Days        []sourceTrendDay `json:"days"`
}

// GetSourceTrend returns daily study ingestion counts split by source (external vs internal).
//
// GET /api/stats/source-trend?days=30&project_id=<uuid>
func (s *Server) GetSourceTrend(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	projectID := r.URL.Query().Get("project_id")
	since := time.Now().UTC().AddDate(0, 0, -days)

	// ── Aggregate totals ──────────────────────────────────────────────────────
	totalsQuery := `
		SELECT
			COUNT(*) FILTER (WHERE source = 'external') AS external,
			COUNT(*) FILTER (WHERE source = 'internal') AS internal,
			COUNT(*) AS total
		FROM studies
		WHERE created_at >= $1
		  AND deleted_at IS NULL`
	args := []any{since}
	if projectID != "" {
		totalsQuery += " AND project_id = $2"
		args = append(args, projectID)
	}

	var totals sourceTrendTotals
	s.db.QueryRowContext(r.Context(), totalsQuery, args...).Scan( //nolint:errcheck
		&totals.External, &totals.Internal, &totals.Total,
	)

	// ── Daily breakdown ───────────────────────────────────────────────────────
	dayQuery := `
		SELECT
			DATE(created_at AT TIME ZONE 'UTC')                           AS day,
			COUNT(*) FILTER (WHERE source = 'external')                   AS external,
			COUNT(*) FILTER (WHERE source = 'internal')                   AS internal,
			COUNT(*)                                                       AS total
		FROM studies
		WHERE created_at >= $1
		  AND deleted_at IS NULL`
	dayArgs := []any{since}
	if projectID != "" {
		dayQuery += " AND project_id = $2"
		dayArgs = append(dayArgs, projectID)
	}
	dayQuery += " GROUP BY day ORDER BY day ASC"

	rows, err := s.db.QueryContext(r.Context(), dayQuery, dayArgs...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query source trend")
		return
	}
	defer rows.Close()

	dayRows := make([]sourceTrendDay, 0)
	for rows.Next() {
		var d sourceTrendDay
		if err := rows.Scan(&d.Day, &d.External, &d.Internal, &d.Total); err != nil {
			continue
		}
		// Normalize: PostgreSQL DATE scanned as string may include time suffix.
		if len(d.Day) > 10 {
			d.Day = d.Day[:10]
		}
		dayRows = append(dayRows, d)
	}

	resp := sourceTrendResponse{
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
