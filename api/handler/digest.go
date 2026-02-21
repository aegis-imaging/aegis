package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListDigestSubscriptions returns all digest subscriptions, optionally filtered by project.
// GET /api/digest-subscriptions           — all subscriptions (admin overview)
// GET /api/projects/{projectID}/digest-subscriptions — filtered by project
func (s *Server) ListDigestSubscriptions(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	var (
		subs []model.DigestSubscription
		err  error
	)
	if projectID != "" {
		subs, err = model.ListDigestSubscriptionsByProject(r.Context(), s.db, projectID)
	} else {
		subs, err = model.ListAllDigestSubscriptions(r.Context(), s.db)
	}
	if err != nil {
		log.Printf("list digest subscriptions: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}
	if subs == nil {
		subs = []model.DigestSubscription{}
	}
	s.writeJSON(w, http.StatusOK, subs)
}

// CreateDigestSubscription creates a new digest subscription for a project.
// POST /api/projects/{projectID}/digest-subscriptions
func (s *Server) CreateDigestSubscription(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var sub model.DigestSubscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if sub.Email == "" {
		s.writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if sub.Frequency != "weekly" && sub.Frequency != "monthly" {
		s.writeError(w, http.StatusBadRequest, "frequency must be weekly or monthly")
		return
	}
	sub.ProjectID = projectID
	sub.Enabled = true

	if err := model.CreateDigestSubscription(r.Context(), s.db, &sub); err != nil {
		log.Printf("create digest subscription: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "digest_subscription.created", actorEmail(r),
		"digest_subscription", sub.ID, clientIP(r), map[string]any{
			"email":      sub.Email,
			"project_id": projectID,
			"frequency":  sub.Frequency,
		})
	s.writeJSON(w, http.StatusCreated, sub)
}

// DeleteDigestSubscription removes a digest subscription.
// DELETE /api/digest-subscriptions/{id}
func (s *Server) DeleteDigestSubscription(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetDigestSubscriptionByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	if err := model.DeleteDigestSubscription(r.Context(), s.db, id); err != nil {
		log.Printf("delete digest subscription %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete subscription")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "digest_subscription.deleted", actorEmail(r),
		"digest_subscription", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}
