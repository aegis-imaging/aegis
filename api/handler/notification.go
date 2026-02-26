package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// ListNotifications GET /api/notifications
func (s *Server) ListNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	unreadOnly := r.URL.Query().Get("unread") == "true"
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	notifs, err := model.ListAdminNotifications(r.Context(), s.db, user.ID, unreadOnly, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if notifs == nil {
		notifs = []model.AdminNotification{}
	}

	count, _ := model.CountUnreadNotifications(r.Context(), s.db, user.ID)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"notifications": notifs,
		"unread_count":  count,
	})
}

// MarkNotificationRead POST /api/notifications/{id}/read
func (s *Server) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	notifID := r.PathValue("id")
	if err := model.MarkNotificationRead(r.Context(), s.db, notifID, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// MarkAllNotificationsRead POST /api/notifications/read-all
func (s *Server) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	n, err := model.MarkAllNotificationsRead(r.Context(), s.db, user.ID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"marked": n})
}

type createNotificationRequest struct {
	UserID string `json:"user_id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Link   string `json:"link"`
}

// CreateNotification POST /api/notifications (admin-only, for system-generated notifications)
func (s *Server) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req createNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" || req.Title == "" {
		s.writeError(w, http.StatusBadRequest, "user_id and title are required")
		return
	}
	n := &model.AdminNotification{
		UserID: req.UserID,
		Title:  req.Title,
		Body:   req.Body,
		Link:   req.Link,
	}
	if err := model.CreateAdminNotification(r.Context(), s.db, n); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, n)
}
