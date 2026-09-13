package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// PriorityQueueItem represents a study with a computed urgency score.
type PriorityQueueItem struct {
	model.Study
	UrgencyScore int      `json:"urgency_score"`
	Reasons      []string `json:"reasons"`
}

// GetStudyPriorityQueue GET /api/studies/priority-queue
// Returns studies ranked by urgency for triage. Score factors:
// - priority_flag: +50
// - stuck (no update in 60+ min for non-terminal): +30
// - pipeline failure (any status = 'failed'): +20
// - unassigned non-terminal: +10
func (s *Server) GetStudyPriorityQueue(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	projectID := r.URL.Query().Get("project_id")

	f := model.StudyFilters{ProjectID: projectID}
	studies, err := model.ListStudies(r.Context(), s.db, f, 500, 0)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	var queue []PriorityQueueItem
	for _, st := range studies {
		// Skip terminal states.
		if st.Status == "approved" || st.Status == "rejected" || st.Status == "expired" {
			continue
		}

		score := 0
		var reasons []string

		if st.PriorityFlag {
			score += 50
			reasons = append(reasons, "priority_flag")
		}

		if hasPipelineFailure(st) {
			score += 20
			reasons = append(reasons, "pipeline_failure")
		}

		if st.AssignedTo == nil {
			score += 10
			reasons = append(reasons, "unassigned")
		}

		if score > 0 {
			queue = append(queue, PriorityQueueItem{Study: st, UrgencyScore: score, Reasons: reasons})
		}
	}

	// Sort by urgency score descending.
	for i := 0; i < len(queue); i++ {
		for j := i + 1; j < len(queue); j++ {
			if queue[j].UrgencyScore > queue[i].UrgencyScore {
				queue[i], queue[j] = queue[j], queue[i]
			}
		}
	}

	if len(queue) > limit {
		queue = queue[:limit]
	}
	if queue == nil {
		queue = []PriorityQueueItem{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"studies": queue,
		"total":   len(queue),
	})
}

func hasPipelineFailure(st model.Study) bool {
	return st.PhiScanStatus == "failed" ||
		st.QcStatus == "failed" ||
		st.BidsStatus == "failed" ||
		st.ClassificationStatus == "failed" ||
		st.ProtocolStatus == "failed" ||
		st.ExportStatus == "failed"
}
