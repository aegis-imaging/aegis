package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// SoftDeleteStudy marks a study as deleted without removing it from the database.
// The study is excluded from normal listings but can be restored with RestoreStudy.
func (s *Server) SoftDeleteStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		s.writeError(w, http.StatusBadRequest, "missing study id")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "study not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve study")
		return
	}

	if err := model.SoftDeleteStudy(r.Context(), s.db, id); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusConflict, "study is already deleted")
			return
		}
		log.Printf("soft delete study %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete study")
		return
	}

	actor := actorEmail(r)
	model.CreateAuditEntry(r.Context(), s.db, "study.soft_deleted", actor, "study", id, clientIP(r), map[string]any{
		"study_instance_uid": study.StudyInstanceUID,
		"status":             study.Status,
	})

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// RestoreStudy clears the deleted_at timestamp on a soft-deleted study,
// making it visible in normal listings again.
func (s *Server) RestoreStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		s.writeError(w, http.StatusBadRequest, "missing study id")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "study not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve study")
		return
	}

	if err := model.RestoreStudy(r.Context(), s.db, id); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusConflict, "study is not deleted")
			return
		}
		log.Printf("restore study %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to restore study")
		return
	}

	actor := actorEmail(r)
	model.CreateAuditEntry(r.Context(), s.db, "study.restored", actor, "study", id, clientIP(r), map[string]any{
		"study_instance_uid": study.StudyInstanceUID,
	})

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "restored", "id": id})
}

// ListDeletedStudies returns soft-deleted studies, newest deletion first.
// Query params: project_id (optional), limit (default 50, max 200), offset (default 0).
func (s *Server) ListDeletedStudies(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	studies, total, err := model.ListDeletedStudies(r.Context(), s.db, projectID, limit, offset)
	if err != nil {
		log.Printf("list deleted studies: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list deleted studies")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"studies": studies,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}
