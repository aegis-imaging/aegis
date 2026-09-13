package handler

import (
	"net/http"
	"strconv"
	"time"
)

// funnelStage is one step in the pipeline funnel.
type funnelStage struct {
	Stage       string  `json:"stage"`
	Count       int     `json:"count"`
	PctOfTotal  float64 `json:"pct_of_total"` // percentage of studies_received
	PctOfPrev   float64 `json:"pct_of_prev"`  // percentage of previous stage (conversion rate)
}

// pipelineFunnelResponse is the payload for GetPipelineFunnel.
type pipelineFunnelResponse struct {
	PeriodDays  int            `json:"period_days"`
	GeneratedAt string         `json:"generated_at"`
	ProjectID   string         `json:"project_id,omitempty"`
	Funnel      []funnelStage  `json:"funnel"`
}

// GetPipelineFunnel returns a conversion funnel showing how many studies pass
// through each pipeline stage from receipt to export.
//
// GET /api/stats/pipeline-funnel?days=30&project_id=<uuid>
func (s *Server) GetPipelineFunnel(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	projectID := r.URL.Query().Get("project_id")
	since := time.Now().UTC().AddDate(0, 0, -days)

	// Single-pass aggregation: all counts in one SQL query.
	var (
		received          int
		classified        int
		phiScanned        int
		pixelRedacted     int
		defaced           int
		qcPassed          int
		bidsConverted     int
		analyticsComplete int
		approved          int
		exported          int
	)
	err := s.db.QueryRowContext(r.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE true)                                                             AS received,
		  COUNT(*) FILTER (WHERE classification_status = 'classified')                             AS classified,
		  COUNT(*) FILTER (WHERE phi_scan_status IN ('clean', 'flagged'))                          AS phi_scanned,
		  COUNT(*) FILTER (WHERE pixel_redaction_required = false
		                      OR (pixel_redaction_required = true AND pixel_redaction_status = 'complete')) AS pixel_redacted,
		  COUNT(*) FILTER (WHERE defacing_required = false
		                      OR (defacing_required = true AND dicom_store = 'clean'))              AS defaced,
		  COUNT(*) FILTER (WHERE qc_status IN ('pass', 'warn'))                                    AS qc_passed,
		  COUNT(*) FILTER (WHERE bids_status = 'complete')                                         AS bids_converted,
		  COUNT(*) FILTER (WHERE analytics_required = false
		                      OR (analytics_required = true AND analytics_status IN ('complete', 'partial'))) AS analytics_complete,
		  COUNT(*) FILTER (WHERE status = 'approved')                                              AS approved,
		  COUNT(*) FILTER (WHERE export_required = false
		                      OR (export_required = true AND export_status = 'exported'))           AS exported
		FROM studies
		WHERE created_at >= $1
		  AND ($2 = '' OR project_id = $2::uuid)`,
		since, projectID,
	).Scan(&received, &classified, &phiScanned, &pixelRedacted, &defaced, &qcPassed,
		&bidsConverted, &analyticsComplete, &approved, &exported)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "pipeline funnel query failed")
		return
	}

	pct := func(n, base int) float64 {
		if base == 0 {
			return 0
		}
		return float64(n) / float64(base) * 100
	}

	stages := []struct {
		name  string
		count int
	}{
		{"received", received},
		{"classified", classified},
		{"phi_scanned", phiScanned},
		{"pixel_redacted", pixelRedacted},
		{"defaced", defaced},
		{"qc_passed", qcPassed},
		{"bids_converted", bidsConverted},
		{"analytics_complete", analyticsComplete},
		{"approved", approved},
		{"exported", exported},
	}

	funnel := make([]funnelStage, len(stages))
	for i, st := range stages {
		prev := received
		if i > 0 {
			prev = stages[i-1].count
		}
		funnel[i] = funnelStage{
			Stage:      st.name,
			Count:      st.count,
			PctOfTotal: pct(st.count, received),
			PctOfPrev:  pct(st.count, prev),
		}
	}

	s.writeJSON(w, http.StatusOK, pipelineFunnelResponse{
		PeriodDays:  days,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ProjectID:   projectID,
		Funnel:      funnel,
	})
}
