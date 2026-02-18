package handler

import (
	"net/http"
	"strconv"

	"github.com/msenjem/aegis/api/model"
)

type studiesResponse struct {
	Studies []model.Study `json:"studies"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
}

func (s *Server) ListStudies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	f := model.StudyFilters{
		ProjectID: q.Get("project_id"),
		Status:    q.Get("status"),
		Modality:  q.Get("modality"),
		Source:    q.Get("source"),
		Search:    q.Get("search"),
	}

	total, err := model.CountStudies(r.Context(), s.db, f)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to count studies")
		return
	}

	studies, err := model.ListStudies(r.Context(), s.db, f, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list studies")
		return
	}
	if studies == nil {
		studies = []model.Study{}
	}

	s.writeJSON(w, http.StatusOK, studiesResponse{
		Studies: studies,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}
