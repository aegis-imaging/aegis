package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/routing"
)

// StowReceiver implements DICOMweb STOW-RS (PS3.18 §10.5).
// POST /api/stow
//
// Accepts multipart/related; type="application/dicom" requests, stores each
// DICOM part, creates study records, and kicks off the processing pipeline.
// Requires a valid API key: Authorization: Bearer <key>
//
// Optional query param: ?project=<slug> (defaults to "default")
func (s *Server) StowReceiver(w http.ResponseWriter, r *http.Request) {
	// ── API key auth ────────────────────────────────────────────────
	rawKey := ""
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		rawKey = strings.TrimPrefix(auth, "Bearer ")
	}
	if rawKey == "" {
		s.writeError(w, http.StatusUnauthorized, "missing API key")
		return
	}
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	apiKey, err := model.GetAPIKeyByHash(r.Context(), s.db, keyHash)
	if err != nil || !apiKey.Enabled {
		s.writeError(w, http.StatusUnauthorized, "invalid or disabled API key")
		return
	}

	// ── Project lookup ───────────────────────────────────────────────
	slug := r.URL.Query().Get("project")
	if slug == "" {
		slug = "default"
	}
	project, err := model.GetProjectBySlug(r.Context(), s.db, slug)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "project not found: "+slug)
		return
	}

	// ── Parse multipart/related ──────────────────────────────────────
	contentType := r.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid Content-Type: "+err.Error())
		return
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		s.writeError(w, http.StatusBadRequest, "Content-Type must be multipart/related")
		return
	}
	boundary := params["boundary"]
	if boundary == "" {
		s.writeError(w, http.StatusBadRequest, "missing multipart boundary")
		return
	}

	// ── Read and group DICOM parts by StudyInstanceUID ───────────────
	type dicomPart struct {
		data          []byte
		studyUID      string
		seriesUID     string
		modality      string
		bodyPart      string
		studyDesc     string
		anonPatientID string
		studyDate     string
	}

	// seriesAcc accumulates parts for one series within a study so we can
	// emit one study_series_metadata row per (study, series) below.
	type seriesAcc struct {
		firstDataset  dicomlib.Dataset
		hasFirst      bool
		instanceCount int
	}

	type studyGroup struct {
		parts         []dicomPart
		modality      string
		bodyPart      string
		studyDesc     string
		anonPatientID string
		studyDate     string
		// seriesOrder preserves the order series were first seen so we
		// produce deterministic upserts.
		seriesOrder []string
		series      map[string]*seriesAcc
	}

	groups := make(map[string]*studyGroup)
	var orderedUIDs []string // preserve insertion order

	mr := multipart.NewReader(r.Body, boundary)
	partIndex := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("stow_receiver: read part %d: %v", partIndex, err)
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to read part %d: %v", partIndex, err))
			return
		}

		partCT := part.Header.Get("Content-Type")
		if partCT != "" && !strings.Contains(partCT, "application/dicom") {
			// Skip non-DICOM parts (e.g. JSON metadata part in some STOW-RS clients)
			part.Close()
			partIndex++
			continue
		}

		data, err := io.ReadAll(part)
		part.Close()
		if err != nil {
			log.Printf("stow_receiver: read part %d body: %v", partIndex, err)
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to read part %d body: %v", partIndex, err))
			return
		}
		if len(data) == 0 {
			partIndex++
			continue
		}

		// Parse DICOM header (skip pixel data for speed).
		dataset, err := dicomlib.Parse(
			bytes.NewReader(data), int64(len(data)), nil,
			dicomlib.SkipPixelData(),
			dicomlib.AllowMissingMetaElementGroupLength(),
		)
		if err != nil {
			log.Printf("stow_receiver: parse DICOM part %d: %v", partIndex, err)
			s.writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("part %d is not a valid DICOM file", partIndex))
			return
		}

		p := dicomPart{
			data:          data,
			studyUID:      stowGetStringTag(dataset, tag.StudyInstanceUID),
			seriesUID:     stowGetStringTag(dataset, tag.SeriesInstanceUID),
			modality:      stowGetStringTag(dataset, tag.Modality),
			bodyPart:      stowGetStringTag(dataset, tag.BodyPartExamined),
			studyDesc:     stowGetStringTag(dataset, tag.StudyDescription),
			anonPatientID: stowGetStringTag(dataset, tag.PatientID),
			studyDate:     stowGetStringTag(dataset, tag.StudyDate),
		}
		if p.studyUID == "" {
			// Assign a generated UID based on the part index if missing.
			p.studyUID = fmt.Sprintf("2.25.stow.%s.%d", project.ID, partIndex)
		}

		g, exists := groups[p.studyUID]
		if !exists {
			g = &studyGroup{
				modality:      p.modality,
				bodyPart:      p.bodyPart,
				studyDesc:     p.studyDesc,
				anonPatientID: p.anonPatientID,
				studyDate:     p.studyDate,
				series:        make(map[string]*seriesAcc),
			}
			groups[p.studyUID] = g
			orderedUIDs = append(orderedUIDs, p.studyUID)
		} else {
			// First non-empty wins per study (in case later parts have the tag but the first didn't).
			if g.anonPatientID == "" && p.anonPatientID != "" {
				g.anonPatientID = p.anonPatientID
			}
			if g.studyDate == "" && p.studyDate != "" {
				g.studyDate = p.studyDate
			}
		}
		g.parts = append(g.parts, p)

		// Track per-series metadata if we have a SeriesInstanceUID. Files
		// without one still get stored (legacy / non-MR) but won't show up
		// in the series metadata table.
		if p.seriesUID != "" {
			sAcc, ok := g.series[p.seriesUID]
			if !ok {
				sAcc = &seriesAcc{firstDataset: dataset, hasFirst: true}
				g.series[p.seriesUID] = sAcc
				g.seriesOrder = append(g.seriesOrder, p.seriesUID)
			}
			sAcc.instanceCount++
		}
		partIndex++
	}

	if len(orderedUIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "no DICOM parts found in request")
		return
	}

	// ── Storage quota check (once per request, not per study) ────────
	if err := s.checkStorageQuota(r.Context(), project.ID); err != nil {
		s.writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	// ── Store files + create studies ─────────────────────────────────
	// DICOMweb STOW-RS success response: Referenced SOP Sequence
	type sopRef struct {
		StudyUID  string `json:"0020000D"`
		SeriesUID string `json:"0020000E"`
		SOPUID    string `json:"00080018"`
	}
	var referencedSOPs []sopRef

	for _, studyUID := range orderedUIDs {
		g := groups[studyUID]

		// Store each file under dicom/raw/{studyUID}/{index}.dcm
		for i, p := range g.parts {
			key := fmt.Sprintf("dicom/raw/%s/%d.dcm", studyUID, i)
			if err := s.store.Store(r.Context(), key, bytes.NewReader(p.data)); err != nil {
				log.Printf("stow_receiver: store %s: %v", key, err)
				s.writeError(w, http.StatusInternalServerError, "failed to store DICOM file")
				return
			}
			sopUID := stowGetStringTag(mustParseDicomHeader(p.data), tag.SOPInstanceUID)
			if sopUID == "" {
				sopUID = fmt.Sprintf("%s.1.%d", studyUID, i)
			}
			referencedSOPs = append(referencedSOPs, sopRef{
				StudyUID:  studyUID,
				SeriesUID: fmt.Sprintf("%s.1", studyUID),
				SOPUID:    sopUID,
			})
		}

		// Create study record.
		bodyPartUpper := strings.ToUpper(g.bodyPart)
		defacingRequired := bodyPartUpper == "HEAD" || bodyPartUpper == "BRAIN"
		var anonPatientIDPtr, studyDatePtr *string
		if g.anonPatientID != "" {
			v := g.anonPatientID
			anonPatientIDPtr = &v
		}
		if g.studyDate != "" {
			v := g.studyDate
			studyDatePtr = &v
		}
		study := &model.Study{
			ProjectID:        project.ID,
			StudyInstanceUID: studyUID,
			Modality:         g.modality,
			BodyPart:         g.bodyPart,
			StudyDescription: g.studyDesc,
			StudyDate:        studyDatePtr,
			AnonPatientID:    anonPatientIDPtr,
			InstanceCount:    len(g.parts),
			Status:           "received",
			DefacingRequired: defacingRequired,
			DicomStore:       "raw",
			Source:           "external",
		}
		if err := model.CreateStudy(r.Context(), s.db, study); err != nil {
			log.Printf("stow_receiver: create study %s: %v", studyUID, err)
			s.writeError(w, http.StatusInternalServerError, "failed to create study record")
			return
		}

		// Upsert per-series DICOM metadata so the dashboard, protocol checker,
		// and analytics can read TR/TE/protocol/etc. without re-parsing files.
		// Best-effort: a write failure here does not abort the receive.
		for _, seriesUID := range g.seriesOrder {
			sAcc := g.series[seriesUID]
			if sAcc == nil || !sAcc.hasFirst {
				continue
			}
			sm := extractSeriesMetadata(sAcc.firstDataset)
			if sm == nil {
				continue
			}
			sm.StudyID = study.ID
			if sm.SeriesInstanceUID == "" {
				sm.SeriesInstanceUID = seriesUID
			}
			sm.InstanceCount = sAcc.instanceCount
			if err := model.UpsertSeriesMetadata(r.Context(), s.db, sm); err != nil {
				log.Printf("stow_receiver: upsert series metadata %s/%s: %v", studyUID, seriesUID, err)
			}
		}

		// Evaluate routing rules (may mutate study — auto_approve, require_defacing, etc.)
		routing.EvaluateRules(r.Context(), s.db, s.store, study)

		// Auto-dispatch processing pipeline.
		s.AdvancePipeline(r.Context(), study.ID)

		model.CreateAuditEntry(r.Context(), s.db, "stow.received", apiKey.Name, "study", study.ID, clientIP(r), map[string]any{
			"study_uid":      studyUID,
			"instance_count": len(g.parts),
			"project":        slug,
			"api_key_name":   apiKey.Name,
		})
	}

	// ── DICOMweb STOW-RS response (PS3.18 §10.5.3) ──────────────────
	// Return referenced SOP sequence as DICOMweb JSON.
	resp := map[string]any{
		"00081199": map[string]any{
			"vr":    "SQ",
			"Value": referencedSOPs,
		},
	}
	w.Header().Set("Content-Type", "application/dicom+json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// stowGetStringTag extracts a string value from a parsed DICOM dataset.
func stowGetStringTag(dataset dicomlib.Dataset, t tag.Tag) string {
	el, err := dataset.FindElementByTag(t)
	if err != nil || el.Value == nil {
		return ""
	}
	strs, ok := el.Value.GetValue().([]string)
	if !ok || len(strs) == 0 {
		return ""
	}
	return strings.TrimSpace(strs[0])
}

// mustParseDicomHeader parses a DICOM file header, returning an empty dataset on error.
func mustParseDicomHeader(data []byte) dicomlib.Dataset {
	ds, err := dicomlib.Parse(
		bytes.NewReader(data), int64(len(data)), nil,
		dicomlib.SkipPixelData(),
		dicomlib.AllowMissingMetaElementGroupLength(),
	)
	if err != nil {
		return dicomlib.Dataset{}
	}
	return ds
}
