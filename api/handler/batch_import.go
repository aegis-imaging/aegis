package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/importer"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/webhook"
)

// BatchImport imports DICOM files from a server-local directory.
// POST /api/import/batch
func (s *Server) BatchImport(w http.ResponseWriter, r *http.Request) {
	var req importer.Options
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Dir == "" {
		s.writeError(w, http.StatusBadRequest, "dir is required")
		return
	}

	result, err := importer.Run(r.Context(), s.db, s.store, req)
	if err != nil {
		if importer.IsValidationError(err) {
			s.writeError(w, http.StatusBadRequest, "import failed: "+err.Error())
			return
		}
		s.writeError(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}

	// Auto-dispatch processing pipeline and fire study.created webhook for each imported study.
	for _, studyID := range result.StudyIDs {
		s.AdvancePipeline(r.Context(), studyID)
		if study, err := model.GetStudyByID(r.Context(), s.db, studyID); err == nil {
			// Detach cancellation: the goroutine outlives the request (values kept).
			go webhook.Deliver(context.WithoutCancel(r.Context()), s.db, "study.created", study)
		}
	}

	s.writeJSON(w, http.StatusOK, result)
}
