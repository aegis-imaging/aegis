package handler

import (
	"net/http"
	"strconv"

	"github.com/msenjem/aegis/api/model"
)

func (s *Server) ListStudies(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	studies, err := model.ListStudies(r.Context(), s.db, projectID, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list studies")
		return
	}
	if studies == nil {
		studies = []model.Study{}
	}
	s.writeJSON(w, http.StatusOK, studies)
}
