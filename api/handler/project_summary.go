package handler

import (
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetProjectDashboardSummary GET /api/projects/{id}/summary
// Returns a compact KPI summary for a project: study counts, modality breakdown, recent activity.
func (s *Server) GetProjectDashboardSummary(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, ok := s.requireProjectReadAccess(w, r, projectID); !ok {
		return
	}

	// Study status counts.
	counts, err := model.GetStudyStatusCounts(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "status counts failed")
		return
	}

	// Modality breakdown.
	breakdown, err := model.GetStudyBreakdown(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "breakdown failed")
		return
	}
	if breakdown == nil {
		breakdown = []model.BreakdownRow{}
	}

	// Storage stats.
	storage, err := model.GetStorageStats(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "storage stats failed")
		return
	}

	// Milestones.
	milestones, err := model.ListProjectMilestones(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "milestones failed")
		return
	}
	if milestones == nil {
		milestones = []model.ProjectMilestone{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"project_id": projectID,
		"studies":    counts,
		"breakdown":  breakdown,
		"storage":    storage,
		"milestones": milestones,
	})
}
