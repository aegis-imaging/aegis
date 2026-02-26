package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// ListSavedViews GET /api/saved-views
func (s *Server) ListSavedViews(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.ID
	}

	views, err := model.ListSavedViews(r.Context(), s.db, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if views == nil {
		views = []model.SavedView{}
	}
	s.writeJSON(w, http.StatusOK, views)
}

// CreateSavedView POST /api/saved-views
func (s *Server) CreateSavedView(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.ID
	}
	if userID == "" {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Name    string          `json:"name"`
		Filters json.RawMessage `json:"filters"`
		Shared  bool            `json:"shared"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Name) > 100 {
		s.writeError(w, http.StatusBadRequest, "name must be 100 characters or fewer")
		return
	}
	if req.Filters == nil {
		req.Filters = json.RawMessage(`{}`)
	}

	v := &model.SavedView{
		UserID:  userID,
		Name:    req.Name,
		Filters: req.Filters,
		Shared:  req.Shared,
	}
	if err := model.CreateSavedView(r.Context(), s.db, v); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, v)
}

// DeleteSavedView DELETE /api/saved-views/{id}
func (s *Server) DeleteSavedView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.ID
	}

	if err := model.DeleteSavedView(r.Context(), s.db, id, userID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UpdateSavedView PUT /api/saved-views/{id}
func (s *Server) UpdateSavedView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.ID
	}

	var req struct {
		Name    string          `json:"name"`
		Filters json.RawMessage `json:"filters"`
		Shared  bool            `json:"shared"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Filters == nil {
		req.Filters = json.RawMessage(`{}`)
	}

	if err := model.UpdateSavedView(r.Context(), s.db, id, userID, req.Name, req.Filters, req.Shared); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
