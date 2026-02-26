package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

type createBookmarkRequest struct {
	AuditID string `json:"audit_id"`
	Note    string `json:"note"`
}

// ListAuditBookmarks GET /api/audit-bookmarks
func (s *Server) ListAuditBookmarks(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	bookmarks, err := model.ListAuditBookmarks(r.Context(), s.db, user.ID, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if bookmarks == nil {
		bookmarks = []model.AuditBookmark{}
	}
	s.writeJSON(w, http.StatusOK, bookmarks)
}

// CreateAuditBookmark POST /api/audit-bookmarks
func (s *Server) CreateAuditBookmark(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req createBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AuditID == "" {
		s.writeError(w, http.StatusBadRequest, "audit_id is required")
		return
	}

	b := &model.AuditBookmark{
		UserID:  user.ID,
		AuditID: req.AuditID,
		Note:    strings.TrimSpace(req.Note),
	}
	if err := model.CreateAuditBookmark(r.Context(), s.db, b); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, b)
}

// DeleteAuditBookmark DELETE /api/audit-bookmarks/{id}
func (s *Server) DeleteAuditBookmark(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	id := r.PathValue("id")
	if err := model.DeleteAuditBookmark(r.Context(), s.db, id, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
