package handler

import (
	"net/http"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// ListStudyWatchers GET /api/studies/{id}/watchers
func (s *Server) ListStudyWatchers(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	watchers, err := model.ListStudyWatchers(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if watchers == nil {
		watchers = []model.StudyWatcher{}
	}
	s.writeJSON(w, http.StatusOK, watchers)
}

// WatchStudy POST /api/studies/{id}/watch
func (s *Server) WatchStudy(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	watcher, err := model.AddStudyWatcher(r.Context(), s.db, studyID, user.ID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "watch failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.watcher_added", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"user_id": user.ID,
	})
	s.writeJSON(w, http.StatusCreated, watcher)
}

// UnwatchStudy DELETE /api/studies/{id}/watch
func (s *Server) UnwatchStudy(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := model.RemoveStudyWatcher(r.Context(), s.db, studyID, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "unwatch failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
