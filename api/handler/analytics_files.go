package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// analyticsFileEntry represents a file in the analytics output directory.
type analyticsFileEntry struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Tool     string `json:"tool"`
	FileType string `json:"file_type"`
}

// ListAnalyticsFiles returns the file tree for a study's analytics outputs.
// GET /api/studies/{id}/analytics-files
func (s *Server) ListAnalyticsFiles(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	analyticsDir := filepath.Join(s.cfg.LocalStorageDir, "analytics", study.StudyInstanceUID)
	if _, err := os.Stat(analyticsDir); os.IsNotExist(err) {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"files": []analyticsFileEntry{},
			"total": 0,
		})
		return
	}

	var files []analyticsFileEntry
	filepath.Walk(analyticsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(analyticsDir, path)

		// Prevent path traversal — reject any relative path that escapes the analytics dir.
		if strings.Contains(rel, "..") {
			return nil
		}

		tool := ""
		parts := strings.SplitN(rel, string(os.PathSeparator), 2)
		if len(parts) > 1 {
			tool = parts[0]
		}

		files = append(files, analyticsFileEntry{
			Path:     rel,
			Name:     info.Name(),
			Size:     info.Size(),
			Tool:     tool,
			FileType: classifyAnalyticsFile(info.Name()),
		})
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"files": files,
		"total": len(files),
	})
}

// ServeAnalyticsFile streams a single analytics output file.
// GET /api/studies/{id}/analytics-files/{path...}
func (s *Server) ServeAnalyticsFile(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	filePath := r.PathValue("path")
	if studyID == "" || filePath == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID or file path")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	// Path traversal protection: clean the path and reject any ".." components.
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		s.writeError(w, http.StatusBadRequest, "invalid file path")
		return
	}

	analyticsDir := filepath.Join(s.cfg.LocalStorageDir, "analytics", study.StudyInstanceUID)
	fullPath := filepath.Join(analyticsDir, filePath)

	// Ensure the resolved path is still within the analytics directory.
	if !strings.HasPrefix(fullPath, analyticsDir+string(os.PathSeparator)) {
		s.writeError(w, http.StatusBadRequest, "invalid file path")
		return
	}

	f, err := os.Open(fullPath)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "file not found")
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		s.writeError(w, http.StatusNotFound, "file not found")
		return
	}

	// Set content type based on extension.
	ext := strings.ToLower(filepath.Ext(fullPath))
	switch ext {
	case ".nii":
		w.Header().Set("Content-Type", "application/octet-stream")
	case ".gz":
		w.Header().Set("Content-Type", "application/gzip")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".csv":
		w.Header().Set("Content-Type", "text/csv")
	case ".mgz", ".mgh":
		w.Header().Set("Content-Type", "application/octet-stream")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.Header().Set("Content-Disposition", "inline; filename=\""+info.Name()+"\"")
	io.Copy(w, f)
}

// classifyAnalyticsFile returns a type label for an analytics output file.
func classifyAnalyticsFile(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".nii.gz") || strings.HasSuffix(lower, ".nii"):
		return "nifti"
	case strings.HasSuffix(lower, ".mgz") || strings.HasSuffix(lower, ".mgh"):
		return "freesurfer_volume"
	case strings.HasSuffix(lower, ".json"):
		return "json"
	case strings.HasSuffix(lower, ".csv"):
		return "csv"
	case strings.HasSuffix(lower, ".stats"):
		return "freesurfer_stats"
	case strings.HasSuffix(lower, ".annot"):
		return "freesurfer_annotation"
	case strings.HasSuffix(lower, ".curv") || strings.HasSuffix(lower, ".thickness"):
		return "freesurfer_overlay"
	case strings.HasSuffix(lower, ".pial") || strings.HasSuffix(lower, ".white") || strings.HasSuffix(lower, ".inflated"):
		return "freesurfer_surface"
	default:
		return "other"
	}
}
