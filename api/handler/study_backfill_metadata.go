package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/model"
)

// BackfillStudyMetadataRequest is the request body for POST /api/admin/backfill-study-metadata.
// Limit caps the number of studies scanned in one call (default 100, max 5000) so the
// admin can run repeated short batches against large historical backlogs.
// DryRun reports what would change without writing.
type BackfillStudyMetadataRequest struct {
	Limit  int  `json:"limit"`
	DryRun bool `json:"dry_run"`
}

// BackfillStudyMetadataResponse summarises the outcome of one backfill run.
type BackfillStudyMetadataResponse struct {
	Scanned int      `json:"scanned"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	DryRun  bool     `json:"dry_run"`
	Errors  []string `json:"errors,omitempty"`
}

// BackfillStudyMetadata reads the first .dcm of each study that is missing
// anon_patient_id or study_date, extracts those tags from the header, and
// writes them back via model.UpdateStudyDicomMetadata (COALESCE semantics —
// already-populated fields are preserved).
//
// Idempotent. Safe to call repeatedly; subsequent calls only touch rows that
// still have NULLs after the previous run.
//
// POST /api/admin/backfill-study-metadata
// Body: { "limit": 100, "dry_run": false }
func (s *Server) BackfillStudyMetadata(w http.ResponseWriter, r *http.Request) {
	var req BackfillStudyMetadataRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Limit > 5000 {
		req.Limit = 5000
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, study_instance_uid, dicom_store
		FROM studies
		WHERE (anon_patient_id IS NULL OR study_date IS NULL)
		  AND deleted_at IS NULL
		ORDER BY created_at
		LIMIT $1`, req.Limit)
	if err != nil {
		log.Printf("backfill-study-metadata: select: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list candidate studies")
		return
	}
	defer rows.Close()

	type candidate struct {
		id, uid, store string
	}
	var todo []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.uid, &c.store); err != nil {
			log.Printf("backfill-study-metadata: scan: %v", err)
			s.writeError(w, http.StatusInternalServerError, "failed to read candidate row")
			return
		}
		todo = append(todo, c)
	}
	if err := rows.Err(); err != nil {
		log.Printf("backfill-study-metadata: rows.Err: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed reading candidates")
		return
	}

	resp := BackfillStudyMetadataResponse{DryRun: req.DryRun}
	for _, c := range todo {
		resp.Scanned++
		store := c.store
		if store == "" {
			store = "raw"
		}
		key := fmt.Sprintf("dicom/%s/%s/0.dcm", store, c.uid)
		anon, sd, err := s.readDicomMetadataHead(r.Context(), key)
		if err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("study %s (%s): %v", c.id, key, err))
			log.Printf("backfill-study-metadata: read %s: %v", key, err)
			continue
		}
		if anon == "" && sd == "" {
			resp.Skipped++
			continue
		}
		var anonPtr, sdPtr *string
		if anon != "" {
			a := anon
			anonPtr = &a
		}
		if sd != "" {
			d := sd
			sdPtr = &d
		}
		if req.DryRun {
			resp.Updated++
			log.Printf("backfill-study-metadata: would update study %s (anon=%q, study_date=%q)", c.id, anon, sd)
			continue
		}
		if err := model.UpdateStudyDicomMetadata(r.Context(), s.db, c.id, anonPtr, sdPtr); err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("study %s: update: %v", c.id, err))
			log.Printf("backfill-study-metadata: update %s: %v", c.id, err)
			continue
		}
		resp.Updated++
		log.Printf("backfill-study-metadata: updated study %s (anon=%q, study_date=%q)", c.id, anon, sd)
	}

	model.CreateAuditEntry(r.Context(), s.db, "admin.backfill_study_metadata",
		actorEmail(r), "studies", "", clientIP(r), map[string]any{
			"limit":   req.Limit,
			"dry_run": req.DryRun,
			"scanned": resp.Scanned,
			"updated": resp.Updated,
			"skipped": resp.Skipped,
		})

	s.writeJSON(w, http.StatusOK, resp)
}

// readDicomMetadataHead retrieves the first 1 MB of a stored DICOM file and
// parses the header to extract PatientID + StudyDate. The 1 MB window is more
// than sufficient — both tags sit in the top-level dataset before any pixel
// data — and keeps backfill cheap on large series.
func (s *Server) readDicomMetadataHead(ctx context.Context, key string) (string, string, error) {
	rc, err := s.store.Retrieve(ctx, key)
	if err != nil {
		return "", "", fmt.Errorf("retrieve: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	if err != nil {
		return "", "", fmt.Errorf("read: %w", err)
	}
	dataset, err := dicomlib.Parse(
		bytes.NewReader(data), int64(len(data)), nil,
		dicomlib.SkipPixelData(),
		dicomlib.AllowMissingMetaElementGroupLength(),
	)
	if err != nil {
		return "", "", fmt.Errorf("parse: %w", err)
	}
	return stowGetStringTag(dataset, tag.PatientID), stowGetStringTag(dataset, tag.StudyDate), nil
}
