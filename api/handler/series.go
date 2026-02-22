package handler

import (
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListStudySeries returns per-series DICOM metadata for a study.
// GET /api/studies/{id}/series
func (s *Server) ListStudySeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	series, err := model.ListStudySeries(r.Context(), s.db, id)
	if err != nil {
		log.Printf("list study series %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list series")
		return
	}
	if series == nil {
		series = []model.StudySeries{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_id": id,
		"series":   series,
		"total":    len(series),
	})
}
