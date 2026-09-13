package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/model"
)

// BackfillStudyMetadataRequest is the request body for POST /api/admin/backfill-study-metadata.
// Limit caps the number of studies scanned in one call (default 100, max 5000) so the
// admin can run repeated short batches against large historical backlogs.
// DryRun reports what would change without writing.
// SeedOrphanSubjectIDFromUID, when true, sets subject_id to the last 18 chars of
// study_instance_uid for studies whose DICOM files have been deleted from storage —
// so the orphan rows still show up in the XNAT-style subject listing rather than
// disappearing into the NULL bucket. Default false (don't synthesize).
type BackfillStudyMetadataRequest struct {
	Limit                       int  `json:"limit"`
	DryRun                      bool `json:"dry_run"`
	SeedOrphanSubjectIDFromUID  bool `json:"seed_orphan_subject_id_from_uid"`
}

// BackfillStudyMetadataResponse summarises the outcome of one backfill run.
type BackfillStudyMetadataResponse struct {
	Scanned       int      `json:"scanned"`
	Updated       int      `json:"updated"`
	Skipped       int      `json:"skipped"`
	OrphansSeeded int      `json:"orphans_seeded,omitempty"`
	DryRun        bool     `json:"dry_run"`
	Errors        []string `json:"errors,omitempty"`
}

// BackfillStudyMetadata reads the first .dcm of each study that is missing
// subject_id or study_date, extracts those tags from the DICOM header, and
// writes them back via model.UpdateStudyDicomMetadata (subject_id only
// overwrites NULL/empty values — researcher edits are never clobbered).
//
// File lookup is robust against historical naming drift: it lists the
// study's storage prefix (`dicom/{store}/{uid}/`) and takes the first
// .dcm found, rather than assuming `0.dcm`. If that comes up empty, it
// also tries the synth-service prefix (`synth/{uid}/`) for studies that
// originated from the synthetic generator and were never copied into
// dicom/raw/.
//
// Idempotent. Safe to call repeatedly.
//
// POST /api/admin/backfill-study-metadata
// Body: { "limit": 100, "dry_run": false, "seed_orphan_subject_id_from_uid": false }
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
		WHERE (subject_id IS NULL OR subject_id = '' OR study_date IS NULL)
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

		key, lookupErr := s.findStudyDicomKey(r.Context(), c.uid, store)
		if lookupErr != nil {
			// File doesn't exist at any known prefix. Optionally seed
			// subject_id from the study UID tail so the row shows up
			// somewhere in the Subjects view instead of being invisible.
			if req.SeedOrphanSubjectIDFromUID {
				synthesized := orphanSubjectIDFromUID(c.uid)
				if req.DryRun {
					resp.OrphansSeeded++
					log.Printf("backfill-study-metadata: would seed orphan study %s subject_id=%q from UID", c.id, synthesized)
					continue
				}
				sptr := synthesized
				if updErr := model.UpdateStudyDicomMetadata(r.Context(), s.db, c.id, &sptr, nil); updErr != nil {
					resp.Skipped++
					resp.Errors = append(resp.Errors, fmt.Sprintf("study %s: orphan seed update: %v", c.id, updErr))
					continue
				}
				resp.OrphansSeeded++
				continue
			}
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("study %s: %v", c.id, lookupErr))
			continue
		}

		subj, sd, err := s.readDicomMetadataHead(r.Context(), key)
		if err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("study %s (%s): %v", c.id, key, err))
			log.Printf("backfill-study-metadata: read %s: %v", key, err)
			continue
		}
		if subj == "" && sd == "" {
			resp.Skipped++
			continue
		}
		var subjPtr, sdPtr *string
		if subj != "" {
			s := subj
			subjPtr = &s
		}
		if sd != "" {
			d := sd
			sdPtr = &d
		}
		if req.DryRun {
			resp.Updated++
			log.Printf("backfill-study-metadata: would update study %s (subject=%q, study_date=%q)", c.id, subj, sd)
			continue
		}
		if err := model.UpdateStudyDicomMetadata(r.Context(), s.db, c.id, subjPtr, sdPtr); err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("study %s: update: %v", c.id, err))
			log.Printf("backfill-study-metadata: update %s: %v", c.id, err)
			continue
		}
		resp.Updated++
		log.Printf("backfill-study-metadata: updated study %s (subject=%q, study_date=%q)", c.id, subj, sd)
	}

	model.CreateAuditEntry(r.Context(), s.db, "admin.backfill_study_metadata",
		actorEmail(r), "studies", "", clientIP(r), map[string]any{
			"limit":           req.Limit,
			"dry_run":         req.DryRun,
			"seed_orphans":    req.SeedOrphanSubjectIDFromUID,
			"scanned":         resp.Scanned,
			"updated":         resp.Updated,
			"orphans_seeded":  resp.OrphansSeeded,
			"skipped":         resp.Skipped,
		})

	s.writeJSON(w, http.StatusOK, resp)
}

// findStudyDicomKey locates one DICOM file belonging to a study by listing
// known storage prefixes. Returns the first .dcm key found, preferring the
// declared dicom_store, then falling back to the other store, then the
// synth-service prefix.
//
// The historical assumption was `dicom/{store}/{uid}/0.dcm` but real-world
// naming has drifted (per-instance-UID filenames, synth-only studies that
// were never copied into dicom/raw/, etc.). Listing the prefix is more
// resilient and only marginally more expensive.
func (s *Server) findStudyDicomKey(ctx context.Context, studyUID, declaredStore string) (string, error) {
	prefixes := []string{
		"dicom/" + declaredStore + "/" + studyUID + "/",
	}
	if declaredStore != "raw" {
		prefixes = append(prefixes, "dicom/raw/"+studyUID+"/")
	}
	if declaredStore != "clean" {
		prefixes = append(prefixes, "dicom/clean/"+studyUID+"/")
	}
	prefixes = append(prefixes, "synth/"+studyUID+"/")

	var triedPrefixes []string
	for _, prefix := range prefixes {
		triedPrefixes = append(triedPrefixes, prefix)
		keys, err := s.store.List(ctx, prefix)
		if err != nil {
			// List failures (transient storage errors) try the next prefix.
			continue
		}
		for _, k := range keys {
			if strings.HasSuffix(strings.ToLower(k), ".dcm") {
				return k, nil
			}
		}
	}
	return "", fmt.Errorf("no DICOM file found at any prefix (tried %s)", strings.Join(triedPrefixes, ", "))
}

// orphanSubjectIDFromUID derives a synthetic subject_id from a study UID
// when the original DICOM file is gone. We use the last 18 characters of
// the UID so each orphan study lands as its own unique subject (rather
// than collapsing all orphans into one bucket).
func orphanSubjectIDFromUID(studyUID string) string {
	if len(studyUID) > 18 {
		return "orphan-" + studyUID[len(studyUID)-18:]
	}
	return "orphan-" + studyUID
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
