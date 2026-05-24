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
	if _, _, ok := s.requireStudyReadAccessByID(w, r, id); !ok {
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

// ListStudySeriesMetadata returns the rich per-series DICOM metadata
// (TR / TE / protocol / sequence / device etc.) captured at ingest time.
// GET /api/studies/{id}/series-metadata
func (s *Server) ListStudySeriesMetadata(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, _, ok := s.requireStudyReadAccessByID(w, r, id); !ok {
		return
	}

	series, err := model.ListSeriesMetadata(r.Context(), s.db, id)
	if err != nil {
		log.Printf("list study series-metadata %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list series metadata")
		return
	}
	if series == nil {
		series = []model.SeriesMetadata{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_id": id,
		"series":   series,
		"total":    len(series),
	})
}
