package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// isUniqueViolation returns true if err is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}

// ListStudyRelationships handles GET /api/studies/{id}/relationships.
func (s *Server) ListStudyRelationships(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "study not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load study")
		return
	}

	rels, err := model.ListStudyRelationships(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list relationships")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"relationships": rels})
}

// CreateStudyRelationship handles POST /api/studies/{id}/relationships.
// Body: {"related_study_id":"...","relationship":"follow_up","notes":""}
func (s *Server) CreateStudyRelationship(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "study not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load study")
		return
	}

	var body struct {
		RelatedStudyID string `json:"related_study_id"`
		Relationship   string `json:"relationship"`
		Notes          string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.RelatedStudyID == "" {
		s.writeError(w, http.StatusBadRequest, "related_study_id is required")
		return
	}
	if body.RelatedStudyID == studyID {
		s.writeError(w, http.StatusBadRequest, "a study cannot be linked to itself")
		return
	}
	if !model.ValidRelationshipTypes[body.Relationship] {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid relationship %q: must be one of baseline, follow_up, comparison, replicate", body.Relationship))
		return
	}

	// Verify the related study exists.
	if _, err := model.GetStudyByID(r.Context(), s.db, body.RelatedStudyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "related study not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load related study")
		return
	}

	rel, err := model.CreateStudyRelationship(r.Context(), s.db, studyID, body.RelatedStudyID, body.Relationship, body.Notes, actorEmail(r))
	if err != nil {
		if isUniqueViolation(err) {
			s.writeError(w, http.StatusConflict, "relationship already exists between these studies")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to create relationship")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.relationship_added", actorEmail(r), "study", studyID, clientIP(r),
		map[string]any{"related_study_id": body.RelatedStudyID, "relationship": body.Relationship})

	s.writeJSON(w, http.StatusCreated, rel)
}

// DeleteStudyRelationship handles DELETE /api/studies/{id}/relationships/{relID}.
func (s *Server) DeleteStudyRelationship(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	relID := r.PathValue("relID")

	deleted, err := model.DeleteStudyRelationship(r.Context(), s.db, relID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete relationship")
		return
	}
	if !deleted {
		s.writeError(w, http.StatusNotFound, "relationship not found")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.relationship_removed", actorEmail(r), "study", studyID, clientIP(r),
		map[string]any{"relationship_id": relID})

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
