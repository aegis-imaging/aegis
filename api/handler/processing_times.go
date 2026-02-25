package handler

import (
	"net/http"
	"strconv"
	"time"
)

// StageTimingRow is the per-stage timing statistics returned by GetProcessingTimes.
type StageTimingRow struct {
	Stage      string  `json:"stage"`
	Count      int     `json:"count"`
	AvgSeconds float64 `json:"avg_seconds"`
	P95Seconds float64 `json:"p95_seconds"`
	MinSeconds float64 `json:"min_seconds"`
	MaxSeconds float64 `json:"max_seconds"`
}

// processingTimesResponse is the full response payload for GetProcessingTimes.
type processingTimesResponse struct {
	GeneratedAt string           `json:"generated_at"`
	PeriodDays  int              `json:"period_days"`
	Stages      []StageTimingRow `json:"stages"`
}

// GetProcessingTimes returns per-stage pipeline processing time statistics
// derived from paired .triggered / .complete audit trail events.
//
// GET /api/stats/processing-times?days=30&project_id=<uuid>
func (s *Server) GetProcessingTimes(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	projectID := r.URL.Query().Get("project_id")
	since := time.Now().UTC().AddDate(0, 0, -days)

	rows, err := queryProcessingTimes(r.Context(), s.db, since, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "processing times query failed")
		return
	}

	s.writeJSON(w, http.StatusOK, processingTimesResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		PeriodDays:  days,
		Stages:      rows,
	})
}
