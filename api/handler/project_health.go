package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// projectHealthResponse is the payload for GetProjectHealth.
type projectHealthResponse struct {
	ProjectID   string                  `json:"project_id,omitempty"`
	PeriodDays  int                     `json:"period_days"`
	GeneratedAt string                  `json:"generated_at"`
	Studies     model.StudyStatusCounts `json:"studies"`
	Storage     *model.StorageStats     `json:"storage"`
	StuckCount  int                     `json:"stuck_count"`
	Routing     projectHealthRouting    `json:"routing"`
	Funnel      []funnelStage           `json:"funnel"`
}

type projectHealthRouting struct {
	Attempts    int     `json:"attempts"`
	Successful  int     `json:"successful"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// GetProjectHealth returns a consolidated health snapshot for a project (or all
// projects) — study counts, storage, routing totals, stuck count, and pipeline
// funnel — in a single call.
//
// GET /api/stats/project-health?days=30&project_id=<uuid>&stuck_minutes=60
func (s *Server) GetProjectHealth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	projectID := q.Get("project_id")

	days := 30
	if d := q.Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	stuckMinutes := 60
	if sm := q.Get("stuck_minutes"); sm != "" {
		if n, err := strconv.Atoi(sm); err == nil && n > 0 {
			stuckMinutes = n
		}
	}

	since := time.Now().UTC().AddDate(0, 0, -days)

	// ── Study counts ────────────────────────────────────────────────────────
	counts, err := model.GetStudyStatusCounts(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "study counts query failed")
		return
	}

	// ── Storage stats ────────────────────────────────────────────────────────
	storage, err := model.GetStorageStats(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "storage stats query failed")
		return
	}
	storage.GeneratedAt = ""

	// ── Stuck count ──────────────────────────────────────────────────────────
	stuck, err := model.GetStuckStudies(r.Context(), s.db, stuckMinutes, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "stuck studies query failed")
		return
	}

	// ── Routing totals (period-scoped) ───────────────────────────────────────
	var routAttempts, routSuccessful int
	routingQuery := `
		SELECT
		  COUNT(*)                                                        AS attempts,
		  COUNT(*) FILTER (WHERE outcome = 'forwarding dispatched')      AS successful
		FROM routing_log
		WHERE created_at >= $1
		  AND action = 'route_to'`
	routingArgs := []any{since}
	if projectID != "" {
		routingQuery += `
		  AND study_id IN (SELECT id FROM studies WHERE project_id = $2::uuid)`
		routingArgs = append(routingArgs, projectID)
	}
	if err := s.db.QueryRowContext(r.Context(), routingQuery, routingArgs...).
		Scan(&routAttempts, &routSuccessful); err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing totals query failed")
		return
	}
	routFailed := routAttempts - routSuccessful
	routRate := 0.0
	if routAttempts > 0 {
		routRate = float64(routSuccessful) / float64(routAttempts)
	}

	// ── Pipeline funnel (reuse same query as GetPipelineFunnel) ─────────────
	var (
		received, classified, phiScanned, defaced int
		qcPassed, bidsConverted, approved, exported int
	)
	funnelQuery := `
		SELECT
		  COUNT(*) FILTER (WHERE true)                                                         AS received,
		  COUNT(*) FILTER (WHERE classification_status = 'classified')                         AS classified,
		  COUNT(*) FILTER (WHERE phi_scan_status IN ('clean', 'flagged'))                      AS phi_scanned,
		  COUNT(*) FILTER (WHERE defacing_required = false
		                      OR (defacing_required = true AND dicom_store = 'clean'))          AS defaced,
		  COUNT(*) FILTER (WHERE qc_status IN ('pass', 'warn'))                                AS qc_passed,
		  COUNT(*) FILTER (WHERE bids_status = 'complete')                                     AS bids_converted,
		  COUNT(*) FILTER (WHERE status = 'approved')                                          AS approved,
		  COUNT(*) FILTER (WHERE export_required = false
		                      OR (export_required = true AND export_status = 'exported'))       AS exported
		FROM studies
		WHERE created_at >= $1
		  AND ($2 = '' OR project_id = $2::uuid)`
	if err := s.db.QueryRowContext(r.Context(), funnelQuery, since, projectID).
		Scan(&received, &classified, &phiScanned, &defaced,
			&qcPassed, &bidsConverted, &approved, &exported); err != nil {
		s.writeError(w, http.StatusInternalServerError, "funnel query failed")
		return
	}

	pct := func(n, base int) float64 {
		if base == 0 {
			return 0
		}
		return float64(n) / float64(base) * 100
	}
	rawStages := []struct {
		name  string
		count int
	}{
		{"received", received},
		{"classified", classified},
		{"phi_scanned", phiScanned},
		{"defaced", defaced},
		{"qc_passed", qcPassed},
		{"bids_converted", bidsConverted},
		{"approved", approved},
		{"exported", exported},
	}
	funnel := make([]funnelStage, len(rawStages))
	for i, st := range rawStages {
		prev := received
		if i > 0 {
			prev = rawStages[i-1].count
		}
		funnel[i] = funnelStage{
			Stage:      st.name,
			Count:      st.count,
			PctOfTotal: pct(st.count, received),
			PctOfPrev:  pct(st.count, prev),
		}
	}

	s.writeJSON(w, http.StatusOK, projectHealthResponse{
		ProjectID:  projectID,
		PeriodDays: days,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Studies:    counts,
		Storage:    storage,
		StuckCount: len(stuck),
		Routing: projectHealthRouting{
			Attempts:    routAttempts,
			Successful:  routSuccessful,
			Failed:      routFailed,
			SuccessRate: routRate,
		},
		Funnel: funnel,
	})
}
