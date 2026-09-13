package handler

import (
	"net/http"
	"strconv"
	"time"
)

// dailySummaryTopProject is one project entry in the top-projects list.
type dailySummaryTopProject struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Received  int    `json:"received"`
}

// dailySummaryRecentEvent is a recent audit entry for the summary.
type dailySummaryRecentEvent struct {
	Action    string `json:"action"`
	Actor     string `json:"actor"`
	CreatedAt string `json:"created_at"`
}

// dailySummaryResponse is the payload for GetDailySummary.
type dailySummaryResponse struct {
	PeriodHours     int    `json:"period_hours"`
	GeneratedAt     string `json:"generated_at"`
	PeriodStartedAt string `json:"period_started_at"`

	// Study ingestion in the period.
	Ingestion struct {
		Received  int `json:"received"`
		Approved  int `json:"approved"`
		Rejected  int `json:"rejected"`
		Stuck     int `json:"stuck"`
	} `json:"ingestion"`

	// Pipeline state (current snapshot, not period-scoped).
	Pipeline struct {
		PendingReview  int `json:"pending_review"`
		InProcessing   int `json:"in_processing"`
		Failed         int `json:"failed"`
	} `json:"pipeline"`

	// Routing in the period.
	Routing struct {
		Attempts    int     `json:"attempts"`
		Successful  int     `json:"successful"`
		Failed      int     `json:"failed"`
		SuccessRate float64 `json:"success_rate"`
	} `json:"routing"`

	// Top 5 projects by received count in the period.
	TopProjects []dailySummaryTopProject `json:"top_projects"`

	// Last 10 significant audit events in the period.
	RecentEvents []dailySummaryRecentEvent `json:"recent_events"`
}

// GetDailySummary returns a system-wide ops briefing for the past N hours.
// Designed for AI agents that need a single call to understand platform state.
//
// GET /api/stats/daily-summary?hours=24
func (s *Server) GetDailySummary(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if n, err := strconv.Atoi(h); err == nil && n > 0 && n <= 168 {
			hours = n
		}
	}

	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	now := time.Now().UTC()

	var resp dailySummaryResponse
	resp.PeriodHours = hours
	resp.GeneratedAt = now.Format(time.RFC3339)
	resp.PeriodStartedAt = since.Format(time.RFC3339)

	// ── Ingestion counts (period-scoped) ──────────────────────────────────────
	ingRow := s.db.QueryRowContext(r.Context(), `
		SELECT
		  COUNT(*)                                           AS received,
		  COUNT(*) FILTER (WHERE status = 'approved')       AS approved,
		  COUNT(*) FILTER (WHERE status = 'rejected')       AS rejected
		FROM studies
		WHERE created_at >= $1`, since)
	if err := ingRow.Scan(&resp.Ingestion.Received, &resp.Ingestion.Approved, &resp.Ingestion.Rejected); err != nil {
		s.writeError(w, http.StatusInternalServerError, "ingestion query failed")
		return
	}

	// ── Stuck count (global, uses default 60-min threshold) ───────────────────
	stuck, err := s.db.QueryContext(r.Context(), `
		SELECT id FROM studies
		WHERE status NOT IN ('approved','rejected','expired')
		  AND updated_at < NOW() - INTERVAL '60 minutes'
		LIMIT 1000`)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "stuck count query failed")
		return
	}
	defer stuck.Close()
	for stuck.Next() {
		resp.Ingestion.Stuck++
	}

	// ── Pipeline state (current snapshot) ─────────────────────────────────────
	pipeRow := s.db.QueryRowContext(r.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'received'
		                     AND classification_status NOT IN ('classifying','classified')
		                     AND phi_scan_status NOT IN ('scanning','flagged')
		                     AND defacing_required = false
		                     AND qc_status NOT IN ('checking')
		                   )                                        AS pending_review,
		  COUNT(*) FILTER (WHERE status IN ('defacing')
		                      OR classification_status = 'classifying'
		                      OR phi_scan_status = 'scanning'
		                      OR qc_status = 'checking'
		                      OR bids_status = 'converting'
		                      OR protocol_status = 'checking'
		                   )                                        AS in_processing,
		  COUNT(*) FILTER (WHERE classification_status = 'failed'
		                      OR phi_scan_status = 'failed'
		                      OR qc_status = 'failed'
		                      OR bids_status = 'failed'
		                      OR defacing_required = true AND dicom_store != 'clean'
		                        AND status NOT IN ('approved','rejected','expired')
		                   )                                        AS failed
		FROM studies`)
	if err := pipeRow.Scan(&resp.Pipeline.PendingReview, &resp.Pipeline.InProcessing, &resp.Pipeline.Failed); err != nil {
		s.writeError(w, http.StatusInternalServerError, "pipeline state query failed")
		return
	}

	// ── Routing totals (period-scoped) ────────────────────────────────────────
	routRow := s.db.QueryRowContext(r.Context(), `
		SELECT
		  COUNT(*)                                                   AS attempts,
		  COUNT(*) FILTER (WHERE outcome = 'forwarding dispatched') AS successful
		FROM routing_log
		WHERE created_at >= $1
		  AND action = 'route_to'`, since)
	if err := routRow.Scan(&resp.Routing.Attempts, &resp.Routing.Successful); err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing query failed")
		return
	}
	resp.Routing.Failed = resp.Routing.Attempts - resp.Routing.Successful
	if resp.Routing.Attempts > 0 {
		resp.Routing.SuccessRate = float64(resp.Routing.Successful) / float64(resp.Routing.Attempts)
	}

	// ── Top 5 projects by received count in period ────────────────────────────
	projRows, err := s.db.QueryContext(r.Context(), `
		SELECT p.id, p.name, p.slug, COUNT(s.id) AS received
		FROM projects p
		JOIN studies s ON s.project_id = p.id
		WHERE s.created_at >= $1
		GROUP BY p.id, p.name, p.slug
		ORDER BY received DESC
		LIMIT 5`, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "top projects query failed")
		return
	}
	defer projRows.Close()
	resp.TopProjects = []dailySummaryTopProject{}
	for projRows.Next() {
		var tp dailySummaryTopProject
		if err := projRows.Scan(&tp.ProjectID, &tp.Name, &tp.Slug, &tp.Received); err != nil {
			continue
		}
		resp.TopProjects = append(resp.TopProjects, tp)
	}

	// ── Recent significant audit events (last 10 in period) ───────────────────
	auditRows, err := s.db.QueryContext(r.Context(), `
		SELECT action, actor, created_at
		FROM audit_trail
		WHERE created_at >= $1
		  AND action IN (
		    'study.approved','study.rejected','study.stuck',
		    'study.exported','pipeline.dispatch',
		    'routing.failed','destination.tested',
		    'project.created','project.cloned',
		    'admin_user.created','admin_user.deleted'
		  )
		ORDER BY created_at DESC
		LIMIT 10`, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "audit query failed")
		return
	}
	defer auditRows.Close()
	resp.RecentEvents = []dailySummaryRecentEvent{}
	for auditRows.Next() {
		var ev dailySummaryRecentEvent
		var ts time.Time
		if err := auditRows.Scan(&ev.Action, &ev.Actor, &ts); err != nil {
			continue
		}
		ev.CreatedAt = ts.UTC().Format(time.RFC3339)
		resp.RecentEvents = append(resp.RecentEvents, ev)
	}

	s.writeJSON(w, http.StatusOK, resp)
}
