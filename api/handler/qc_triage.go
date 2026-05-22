package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// ListQCTriage GET /api/qc/triage
// Filters:
//
//	?assigned=me|all|<analyst_id>   ("me" resolves to the authenticated user)
//	?project_id=<uuid>
//	?status=received,clean,defaced  (comma-separated)
//	?open=true                       (only studies where qc_review_ended_at is NULL)
//	?limit=200                       (default 200, max 1000)
func (s *Server) ListQCTriage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	assigned := q.Get("assigned")
	filters := model.QCTriageFilters{
		ProjectID: q.Get("project_id"),
		OnlyOpen:  q.Get("open") == "true",
	}
	if statusParam := q.Get("status"); statusParam != "" {
		for _, p := range strings.Split(statusParam, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				filters.StatusIn = append(filters.StatusIn, p)
			}
		}
	}
	switch assigned {
	case "me":
		if u := middleware.UserFromContext(r.Context()); u != nil && u.ID != "" {
			filters.AssignedToID = u.ID
		}
	case "", "all":
		// no assignment filter
	default:
		filters.AssignedToID = assigned
	}
	limit := 200
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 1000 {
		limit = n
	}
	rows, err := model.ListTriageQueue(r.Context(), s.db, filters, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": rows, "total": len(rows)})
}

type assignReq struct {
	AnalystID string `json:"analyst_id"`
}

// AssignQC POST /api/studies/{id}/qc/assign
func (s *Server) AssignQC(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	var req assignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	analystID := strings.TrimSpace(req.AnalystID)
	if analystID == "" {
		// Self-assign by default
		if u := middleware.UserFromContext(r.Context()); u != nil && u.ID != "" {
			analystID = u.ID
		} else {
			s.writeError(w, http.StatusBadRequest, "analyst_id is required")
			return
		}
	}
	if err := model.AssignQCReviewer(r.Context(), s.db, studyID, analystID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "assign failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.assigned",
		actorEmail(r), "study", studyID, clientIP(r), map[string]any{"analyst_id": analystID})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "assigned", "analyst_id": analystID})
}

// UnassignQC DELETE /api/studies/{id}/qc/assign
func (s *Server) UnassignQC(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if err := model.UnassignQCReviewer(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "unassign failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.unassigned", actorEmail(r), "study", studyID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "unassigned"})
}

// StartQC POST /api/studies/{id}/qc/start — stamps qc_review_started_at = now()
// for throughput tracking.
func (s *Server) StartQC(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if err := model.StartQCReview(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "start failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.review_started", actorEmail(r), "study", studyID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

// CompleteQC POST /api/studies/{id}/qc/complete — stamps qc_review_ended_at = now()
// for throughput tracking.
func (s *Server) CompleteQC(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if err := model.CompleteQCReview(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "complete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.review_completed", actorEmail(r), "study", studyID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

// GetAnalystThroughput GET /api/qc/throughput?days=7
func (s *Server) GetAnalystThroughput(w http.ResponseWriter, r *http.Request) {
	days := 7
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	stats, err := model.GetAnalystThroughput(r.Context(), s.db, days)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"days": days, "analysts": stats, "total": len(stats)})
}
