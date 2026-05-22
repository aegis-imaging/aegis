package handler

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// FileManifestEntry represents one DICOM file in a study.
type FileManifestEntry struct {
	Index    int    `json:"index"`
	Filename string `json:"filename"`
	Store    string `json:"store"`
}

// GetStudyFileManifest GET /api/studies/{id}/files
// Returns a listing of DICOM files for a study.
func (s *Server) GetStudyFileManifest(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	limit := study.InstanceCount
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < limit {
			limit = n
		}
	}

	var files []FileManifestEntry
	for i := 0; i < limit; i++ {
		files = append(files, FileManifestEntry{
			Index:    i,
			Filename: filepath.Join("dicom", study.DicomStore, study.StudyInstanceUID, strconv.Itoa(i)+".dcm"),
			Store:    study.DicomStore,
		})
	}
	if files == nil {
		files = []FileManifestEntry{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_id":       studyID,
		"instance_count": study.InstanceCount,
		"dicom_store":    study.DicomStore,
		"files":          files,
		"total":          len(files),
	})
}
