package handler

import (
	"encoding/json"
	"net/http"

	"github.com/msenjem/aegis/api/importer"
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

	// Auto-dispatch processing pipeline for each imported study.
	for _, studyID := range result.StudyIDs {
		s.AdvancePipeline(r.Context(), studyID)
	}

	s.writeJSON(w, http.StatusOK, result)
}
