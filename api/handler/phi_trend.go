package handler

import (
	"net/http"
	"strconv"
	"time"
)

// phiTrendDay is one day's PHI scan counts.
type phiTrendDay struct {
	Day         string  `json:"day"`
	Scanned     int     `json:"scanned"`
	Flagged     int     `json:"flagged"`
	FlagRatePct float64 `json:"flag_rate_pct"` // flagged / scanned * 100; -1 if no scans
}

// phiTrendResponse is the full response for GET /api/stats/phi-trend.
type phiTrendResponse struct {
	GeneratedAt string        `json:"generated_at"`
	PeriodDays  int           `json:"period_days"`
	ProjectID   string        `json:"project_id,omitempty"`
	Totals      phiTrendTotals `json:"totals"`
	Days        []phiTrendDay `json:"days"`
}

type phiTrendTotals struct {
	Scanned     int     `json:"scanned"`
	Flagged     int     `json:"flagged"`
	FlagRatePct float64 `json:"flag_rate_pct"` // -1 if no scans
}

// GetPhiTrend returns daily PHI scan counts over the last N days.
//
// GET /api/stats/phi-trend?days=30&project_id=<uuid>
func (s *Server) GetPhiTrend(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	projectID := r.URL.Query().Get("project_id")

	since := time.Now().UTC().AddDate(0, 0, -days)

	// ── Aggregate totals ──────────────────────────────────────────────────────
	query := `
		SELECT
			COUNT(*) FILTER (WHERE phi_scan_status NOT IN ('','pending')) AS scanned,
			COUNT(*) FILTER (WHERE phi_scan_status = 'flagged')           AS flagged
		FROM studies
		WHERE created_at >= $1`
	args := []any{since}
	if projectID != "" {
		query += " AND project_id = $2"
		args = append(args, projectID)
	}

	var totals phiTrendTotals
	s.db.QueryRowContext(r.Context(), query, args...).Scan( //nolint:errcheck
		&totals.Scanned, &totals.Flagged,
	)
	if totals.Scanned > 0 {
		totals.FlagRatePct = float64(totals.Flagged) / float64(totals.Scanned) * 100
	} else {
		totals.FlagRatePct = -1
	}

	// ── Daily breakdown ───────────────────────────────────────────────────────
	dayQuery := `
		SELECT
			DATE(created_at AT TIME ZONE 'UTC')                                                AS day,
			COUNT(*) FILTER (WHERE phi_scan_status NOT IN ('','pending')) AS scanned,
			COUNT(*) FILTER (WHERE phi_scan_status = 'flagged')           AS flagged
		FROM studies
		WHERE created_at >= $1`
	dayArgs := []any{since}
	if projectID != "" {
		dayQuery += " AND project_id = $2"
		dayArgs = append(dayArgs, projectID)
	}
	dayQuery += " GROUP BY day ORDER BY day ASC"

	rows, err := s.db.QueryContext(r.Context(), dayQuery, dayArgs...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query phi trend")
		return
	}
	defer rows.Close()

	dayRows := make([]phiTrendDay, 0)
	for rows.Next() {
		var d phiTrendDay
		if err := rows.Scan(&d.Day, &d.Scanned, &d.Flagged); err != nil {
			continue
		}
		if d.Scanned > 0 {
			d.FlagRatePct = float64(d.Flagged) / float64(d.Scanned) * 100
		} else {
			d.FlagRatePct = -1
		}
		dayRows = append(dayRows, d)
	}

	resp := phiTrendResponse{
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
