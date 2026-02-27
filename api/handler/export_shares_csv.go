package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// ExportSharesCSV streams all export shares as a CSV download.
//
// GET /api/export-shares.csv?project_id=<uuid>&status=active|expired|revoked
// Columns: id, study_id, recipient_email, created_by, status, created_at,
//
//	expires_at, revoked_at, download_count, note
func (s *Server) ExportSharesCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := model.ShareStatusFilter(q.Get("status"))
	projectID := q.Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil {
		projectID = access.ProjectID
		if access.IsSiteScoped() {
			institutionID = *access.InstitutionID
		}
	}

	// Cap at 10 000 rows to protect against runaway exports.
	const maxRows = 10000
	var shares []model.ExportShare
	var err error
	if access != nil {
		shares, err = model.ListAllExportSharesForScope(r.Context(), s.db, status, maxRows, 0, projectID, institutionID)
	} else {
		shares, err = model.ListAllExportShares(r.Context(), s.db, status, maxRows, 0, projectID)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list shares")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="export-shares.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"id", "study_id", "recipient_email", "created_by",
		"status", "created_at", "expires_at", "revoked_at",
		"download_count", "max_downloads", "note",
	})

	now := time.Now().UTC()
	for _, share := range shares {
		shareStatus := "active"
		if share.RevokedAt != nil {
			shareStatus = "revoked"
		} else if now.After(share.ExpiresAt) {
			shareStatus = "expired"
		}

		revokedAt := ""
		if share.RevokedAt != nil {
			revokedAt = share.RevokedAt.Format(time.RFC3339)
		}
		maxDL := ""
		if share.MaxDownloads != nil {
			maxDL = strconv.Itoa(*share.MaxDownloads)
		}

		_ = cw.Write([]string{
			share.ID,
			share.StudyID,
			share.RecipientEmail,
			share.CreatedBy,
			shareStatus,
			share.CreatedAt.Format(time.RFC3339),
			share.ExpiresAt.Format(time.RFC3339),
			revokedAt,
			strconv.Itoa(share.DownloadCount),
			maxDL,
			share.Note,
		})
	}

	if len(shares) == maxRows {
		_ = cw.Write([]string{"# truncated", fmt.Sprintf("export capped at %d rows", maxRows)})
	}

	cw.Flush()
}
