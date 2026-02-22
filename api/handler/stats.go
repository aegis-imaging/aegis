package handler

import (
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type statsResponse struct {
	StudyCounts  model.StudyStatusCounts `json:"study_counts"`
	ActiveShares int                     `json:"active_shares"`
	GeneratedAt  time.Time               `json:"generated_at"`
}

// GetStats returns a lightweight snapshot of study pipeline state and active
// share count. Used by the admin dashboard overview banner.
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	counts, err := model.GetStudyStatusCounts(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query study counts")
		return
	}

	activeShares, err := model.CountAllExportShares(r.Context(), s.db, model.ShareStatusActive)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query share counts")
		return
	}

	s.writeJSON(w, http.StatusOK, statsResponse{
		StudyCounts:  counts,
		ActiveShares: activeShares,
		GeneratedAt:  time.Now().UTC(),
	})
}

// GetBreakdownStats returns study counts grouped by modality and body part.
// GET /api/stats/breakdown
func (s *Server) GetBreakdownStats(w http.ResponseWriter, r *http.Request) {
	rows, err := model.GetStudyBreakdown(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query breakdown stats")
		return
	}
	if rows == nil {
		rows = []model.BreakdownRow{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"breakdown":    rows,
		"generated_at": time.Now().UTC(),
	})
}

// GetStorageStats returns aggregate DICOM file counts derived from the studies table.
// GET /api/storage/stats
func (s *Server) GetStorageStats(w http.ResponseWriter, r *http.Request) {
	stats, err := model.GetStorageStats(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query storage stats")
		return
	}
	stats.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	s.writeJSON(w, http.StatusOK, stats)
}
