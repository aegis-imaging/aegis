package handler

import (
	"io"
	"net/http"
	"strings"
)

// ServeStorageFile streams a file from local storage. Used in dev mode so that
// GenerateDownloadURL can return real HTTP URLs. In production (GCS), the
// download URLs are signed GCS URLs and this endpoint is never needed.
func (s *Server) ServeStorageFile(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" || strings.Contains(key, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	rc, err := s.store.Retrieve(r.Context(), key)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/dicom")
	w.Header().Set("Content-Disposition", "attachment")
	io.Copy(w, rc)
}
