package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetProjectBidsInfo returns metadata about BIDS-converted studies in a project
// without streaming the actual ZIP. Used by the MCP server and admin dashboard
// to show availability before initiating a download.
//
// GET /api/projects/{id}/bids-info
//
// Returns: {project_id, bids_complete_count, status_filter, study_uids[], download_url, truncated}
func (s *Server) GetProjectBidsInfo(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project ID")
		return
	}
	if _, ok := s.requireProjectReadAccess(w, r, projectID); !ok {
		return
	}

	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "approved"
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT study_instance_uid
		FROM studies
		WHERE project_id = $1
		  AND bids_status = 'complete'
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3`,
		projectID, statusFilter, maxProjectBidsStudies)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var uids []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			s.writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		uids = append(uids, uid)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "query error")
		return
	}
	if uids == nil {
		uids = []string{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"project_id":          projectID,
		"bids_complete_count": len(uids),
		"status_filter":       statusFilter,
		"study_uids":          uids,
		"truncated":           len(uids) >= maxProjectBidsStudies,
		"download_url":        fmt.Sprintf("/api/projects/%s/bids-export?status=%s", projectID, statusFilter),
	})
}

// maxProjectBidsStudies caps how many BIDS-complete studies are included in one bulk export.
const maxProjectBidsStudies = 500

// ServeProjectBidsExport streams a merged BIDS archive for all studies in a project
// that have bids_status = 'complete'. Files from each study are nested under the
// study's existing BIDS directory structure and combined into one ZIP.
//
// GET /api/projects/{id}/bids-export
//
// Query params:
//   - status: optional study status filter (default: "approved")
func (s *Server) ServeProjectBidsExport(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project ID")
		return
	}
	if _, ok := s.requireProjectReadAccess(w, r, projectID); !ok {
		return
	}

	// Verify the project exists.
	proj, err := model.GetProjectByID(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Optional study status filter; defaults to "approved" so we only export finished studies.
	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "approved"
	}

	// Fetch all studies in the project with BIDS complete.
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, study_instance_uid
		FROM studies
		WHERE project_id = $1
		  AND bids_status = 'complete'
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3`,
		projectID, statusFilter, maxProjectBidsStudies)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query studies")
		return
	}
	defer rows.Close()

	type studyEntry struct {
		id  string
		uid string
	}
	var studies []studyEntry
	for rows.Next() {
		var e studyEntry
		if err := rows.Scan(&e.id, &e.uid); err != nil {
			s.writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		studies = append(studies, e)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "query error")
		return
	}

	if len(studies) == 0 {
		s.writeError(w, http.StatusNotFound, "no BIDS-complete studies found in this project")
		return
	}

	// Build a safe filename slug from the project name.
	nameSlug := strings.ReplaceAll(strings.ToLower(proj.Name), " ", "_")
	nameSlug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, nameSlug)
	if len(nameSlug) > 40 {
		nameSlug = nameSlug[:40]
	}
	if nameSlug == "" {
		nameSlug = "project"
	}

	filename := fmt.Sprintf("%s_bids_%d_studies.zip", nameSlug, len(studies))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	// Signal how many studies are included so callers can show a progress message.
	w.Header().Set("X-BIDS-Study-Count", fmt.Sprintf("%d", len(studies)))
	if len(studies) >= maxProjectBidsStudies {
		w.Header().Set("X-BIDS-Truncated", "true")
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, study := range studies {
		bidsDir := filepath.Join(s.cfg.LocalStorageDir, "bids", study.uid)
		if _, err := os.Stat(bidsDir); err != nil {
			// BIDS output missing from disk — skip silently (status is stale).
			continue
		}

		// Walk the study BIDS directory and add each file, prefixed by the study UID
		// so per-study directories remain intact in the merged archive.
		prefix := study.uid + "/"
		filepath.Walk(bidsDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info.IsDir() {
				return walkErr
			}
			relPath, _ := filepath.Rel(bidsDir, path)
			relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")
			entryPath := prefix + relPath

			fw, err := zw.Create(entryPath)
			if err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(fw, f)
			return err
		})
	}

	// Audit the export.
	model.CreateAuditEntry(r.Context(), s.db,
		"project.bids_export",
		actorEmail(r),
		"project", projectID,
		clientIP(r),
		map[string]any{
			"study_count":   len(studies),
			"status_filter": statusFilter,
			"truncated":     len(studies) >= maxProjectBidsStudies,
		})
}
