package handler

import (
	"net/http"
	"strconv"
	"time"
)

// protocolTrendDay is one day's protocol compliance counts.
type protocolTrendDay struct {
	Day            string `json:"day"`
	Checked        int    `json:"checked"`
	Compliant      int    `json:"compliant"`
	MinorDeviations int   `json:"minor_deviations"`
	NonCompliant   int    `json:"non_compliant"`
	CompliancePct  float64 `json:"compliance_pct"` // compliant / checked * 100; -1 if no checks
}

// protocolTrendResponse is the full response for GET /api/stats/protocol-trend.
type protocolTrendResponse struct {
	GeneratedAt string               `json:"generated_at"`
	PeriodDays  int                  `json:"period_days"`
	ProjectID   string               `json:"project_id,omitempty"`
	Totals      protocolTrendTotals  `json:"totals"`
	Days        []protocolTrendDay   `json:"days"`
}

type protocolTrendTotals struct {
	Checked         int     `json:"checked"`
	Compliant       int     `json:"compliant"`
	MinorDeviations int     `json:"minor_deviations"`
	NonCompliant    int     `json:"non_compliant"`
	CompliancePct   float64 `json:"compliance_pct"`
}

// GetProtocolTrend returns daily protocol compliance counts over the last N days.
//
// GET /api/stats/protocol-trend?days=30&project_id=<uuid>
func (s *Server) GetProtocolTrend(w http.ResponseWriter, r *http.Request) {
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
			COUNT(*) FILTER (WHERE protocol_required AND protocol_status NOT IN ('','pending','checking')) AS checked,
			COUNT(*) FILTER (WHERE protocol_status = 'compliant')                                         AS compliant,
			COUNT(*) FILTER (WHERE protocol_status = 'minor_deviations')                                  AS minor,
			COUNT(*) FILTER (WHERE protocol_status = 'non_compliant')                                     AS non_compliant
		FROM studies
		WHERE created_at >= $1`
	args := []any{since}
	if projectID != "" {
		query += " AND project_id = $2"
		args = append(args, projectID)
	}

	var totals protocolTrendTotals
	s.db.QueryRowContext(r.Context(), query, args...).Scan( //nolint:errcheck
		&totals.Checked, &totals.Compliant, &totals.MinorDeviations, &totals.NonCompliant,
	)
	if totals.Checked > 0 {
		totals.CompliancePct = float64(totals.Compliant) / float64(totals.Checked) * 100
	} else {
		totals.CompliancePct = -1
	}

	// ── Daily breakdown ───────────────────────────────────────────────────────
	dayQuery := `
		SELECT
			DATE(created_at AT TIME ZONE 'UTC')                                                            AS day,
			COUNT(*) FILTER (WHERE protocol_required AND protocol_status NOT IN ('','pending','checking')) AS checked,
			COUNT(*) FILTER (WHERE protocol_status = 'compliant')                                         AS compliant,
			COUNT(*) FILTER (WHERE protocol_status = 'minor_deviations')                                  AS minor,
			COUNT(*) FILTER (WHERE protocol_status = 'non_compliant')                                     AS non_compliant
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
		s.writeError(w, http.StatusInternalServerError, "failed to query protocol trend")
		return
	}
	defer rows.Close()

	dayRows := make([]protocolTrendDay, 0)
	for rows.Next() {
		var d protocolTrendDay
		if err := rows.Scan(&d.Day, &d.Checked, &d.Compliant, &d.MinorDeviations, &d.NonCompliant); err != nil {
			continue
		}
		if d.Checked > 0 {
			d.CompliancePct = float64(d.Compliant) / float64(d.Checked) * 100
		} else {
			d.CompliancePct = -1
		}
		dayRows = append(dayRows, d)
	}

	resp := protocolTrendResponse{
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
