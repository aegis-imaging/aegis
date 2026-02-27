package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// checkStorageQuota returns an error if the project has a storage quota set and has met or exceeded it.
// Returns nil when no quota is configured or when usage is within quota.
func (s *Server) checkStorageQuota(ctx context.Context, projectID string) error {
	project, err := model.GetProjectByID(ctx, s.db, projectID)
	if err != nil {
		return nil // non-fatal: don't block upload on DB error
	}
	if project.StorageQuotaBytes == nil {
		return nil // no quota set
	}
	used, err := model.GetProjectStorageUsage(ctx, s.db, projectID)
	if err != nil {
		return nil // non-fatal
	}
	if used >= *project.StorageQuotaBytes {
		return fmt.Errorf("project storage quota exceeded (%d / %d bytes used)", used, *project.StorageQuotaBytes)
	}
	return nil
}

// SetStorageQuota handles PUT /api/projects/{id}/storage-quota (adminOnly).
// Body: {"storage_quota_bytes": 10737418240} or {"storage_quota_bytes": null} to clear.
func (s *Server) SetStorageQuota(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var body struct {
		StorageQuotaBytes *int64 `json:"storage_quota_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.StorageQuotaBytes != nil && *body.StorageQuotaBytes <= 0 {
		s.writeError(w, http.StatusBadRequest, "storage_quota_bytes must be a positive integer or null")
		return
	}

	project, err := model.GetProjectByID(r.Context(), s.db, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "project not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load project")
		return
	}

	if err := model.UpdateProjectStorageQuota(r.Context(), s.db, projectID, body.StorageQuotaBytes); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update storage quota")
		return
	}

	detail := map[string]any{"quota_bytes": body.StorageQuotaBytes}
	model.CreateAuditEntry(r.Context(), s.db, "project.storage_quota_updated", actorEmail(r), "project", projectID, clientIP(r), detail)

	// Return the updated project.
	project.StorageQuotaBytes = body.StorageQuotaBytes
	s.writeJSON(w, http.StatusOK, project)
}

// GetStorageUsage handles GET /api/projects/{id}/storage-usage (RequireAuth).
// Returns the current storage usage in bytes and the quota (if set).
func (s *Server) GetStorageUsage(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, ok := s.requireProjectReadAccess(w, r, projectID); !ok {
		return
	}

	project, err := model.GetProjectByID(r.Context(), s.db, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "project not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load project")
		return
	}

	usedBytes, err := model.GetProjectStorageUsage(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query storage usage")
		return
	}

	resp := map[string]any{
		"project_id":  projectID,
		"used_bytes":  usedBytes,
		"quota_bytes": project.StorageQuotaBytes,
	}
	if project.StorageQuotaBytes != nil && *project.StorageQuotaBytes > 0 {
		pct := float64(usedBytes) / float64(*project.StorageQuotaBytes) * 100
		resp["usage_pct"] = pct
	} else {
		resp["usage_pct"] = nil
	}

	s.writeJSON(w, http.StatusOK, resp)
}
