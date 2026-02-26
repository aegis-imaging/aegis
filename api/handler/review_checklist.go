package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type createChecklistItemRequest struct {
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	Required  bool   `json:"required"`
}

// ListReviewChecklist GET /api/projects/{projectID}/review-checklist
func (s *Server) ListReviewChecklist(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	items, err := model.ListReviewChecklistItems(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if items == nil {
		items = []model.ReviewChecklistItem{}
	}
	s.writeJSON(w, http.StatusOK, items)
}

// CreateReviewChecklistItem POST /api/projects/{projectID}/review-checklist
func (s *Server) CreateReviewChecklistItem(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")

	var req createChecklistItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		s.writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	if len(label) > 200 {
		s.writeError(w, http.StatusBadRequest, "label must be 200 characters or fewer")
		return
	}

	item := &model.ReviewChecklistItem{
		ProjectID: projectID,
		Label:     label,
		SortOrder: req.SortOrder,
		Required:  req.Required,
		Enabled:   true,
	}
	if err := model.CreateReviewChecklistItem(r.Context(), s.db, item); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "review_checklist.item_created", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"item_id": item.ID, "label": label,
	})
	s.writeJSON(w, http.StatusCreated, item)
}

type updateChecklistItemRequest struct {
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	Required  bool   `json:"required"`
	Enabled   bool   `json:"enabled"`
}

// UpdateReviewChecklistItem PUT /api/review-checklist/{id}
func (s *Server) UpdateReviewChecklistItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateChecklistItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		s.writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	if err := model.UpdateReviewChecklistItem(r.Context(), s.db, id, label, req.SortOrder, req.Required, req.Enabled); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteReviewChecklistItem DELETE /api/review-checklist/{id}
func (s *Server) DeleteReviewChecklistItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.DeleteReviewChecklistItem(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListStudyChecklistResponses GET /api/studies/{id}/checklist
func (s *Server) ListStudyChecklistResponses(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	responses, err := model.ListStudyChecklistResponses(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if responses == nil {
		responses = []model.StudyChecklistResponse{}
	}
	s.writeJSON(w, http.StatusOK, responses)
}

type checklistResponseRequest struct {
	ChecklistItemID string `json:"checklist_item_id"`
	Checked         bool   `json:"checked"`
}

// UpsertStudyChecklistResponse POST /api/studies/{id}/checklist
func (s *Server) UpsertStudyChecklistResponse(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	var req checklistResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ChecklistItemID == "" {
		s.writeError(w, http.StatusBadRequest, "checklist_item_id is required")
		return
	}

	if err := model.UpsertStudyChecklistResponse(r.Context(), s.db, studyID, req.ChecklistItemID, req.Checked, actorEmail(r)); err != nil {
		s.writeError(w, http.StatusInternalServerError, "upsert failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
