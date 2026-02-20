package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/routing"
)

type ingestRequest struct {
	ProjectSlug        string        `json:"project_slug"`
	InstitutionID      string        `json:"institution_id,omitempty"`
	InstitutionSlug    string        `json:"institution_slug,omitempty"`
	InstitutionAETitle string        `json:"institution_ae_title,omitempty"`
	Metadata           studyMetadata `json:"study_metadata"`
}

type ipInstitutionMatch struct {
	Institution *model.Institution
	PrefixLen   int
}

// InternalIngest accepts studies originating inside the enterprise network.
// It tags the study as source="internal" and enters the same processing pipeline
// as externally-uploaded studies (de-id validation, defacing gate, QC, approval).
//
// NOTE: In Phase 1 the pipeline steps are stubs — the study record is created
// and routed, but server-side processing is not yet wired up.
func (s *Server) InternalIngest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	slug := req.ProjectSlug
	if slug == "" {
		slug = "default"
	}
	project, err := model.GetProjectBySlug(r.Context(), s.db, slug)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "project not found: "+slug)
		return
	}

	var institutionID *string
	sourceIP := ""
	sourceIPAttributionError := ""
	if strings.TrimSpace(req.InstitutionID) != "" || strings.TrimSpace(req.InstitutionSlug) != "" || strings.TrimSpace(req.InstitutionAETitle) != "" {
		inst, err := s.resolveIngestInstitution(r.Context(), project.ID, req.InstitutionID, req.InstitutionSlug, req.InstitutionAETitle)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		institutionID = &inst.ID
	} else {
		sourceIP = clientIP(r)
		inst, err := s.resolveIngestInstitutionFromSourceIP(r.Context(), project.ID, sourceIP)
		if err != nil {
			log.Printf("internal ingest: resolve institution by source IP (%s): %v", sourceIP, err)
			sourceIPAttributionError = err.Error()
		} else if inst != nil {
			institutionID = &inst.ID
		}
	}

	studyUID := req.Metadata.StudyInstanceUID
	if studyUID == "" {
		studyUID = fmt.Sprintf("2.25.%d", time.Now().UnixNano())
	}

	bodyPart := strings.ToUpper(req.Metadata.BodyPart)
	defacingRequired := bodyPart == "HEAD" || bodyPart == "BRAIN"

	study := &model.Study{
		ProjectID:        project.ID,
		StudyInstanceUID: studyUID,
		Modality:         req.Metadata.Modality,
		BodyPart:         req.Metadata.BodyPart,
		StudyDescription: req.Metadata.StudyDescription,
		SeriesCount:      req.Metadata.SeriesCount,
		InstanceCount:    req.Metadata.InstanceCount,
		Status:           "received",
		DefacingRequired: defacingRequired,
		DicomStore:       "raw",
		Source:           "internal",
		InstitutionID:    institutionID,
	}
	if err := model.CreateStudy(r.Context(), s.db, study); err != nil {
		log.Printf("create study (internal ingest): %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create study record")
		return
	}

	// Evaluate routing rules — may mutate study (e.g. auto_approve, require_defacing).
	routing.EvaluateRules(r.Context(), s.db, s.store, study)

	// Auto-dispatch processing pipeline.
	s.AdvancePipeline(r.Context(), study.ID)

	detail := map[string]any{
		"study_uid": studyUID,
		"modality":  study.Modality,
		"project":   slug,
	}
	if institutionID != nil {
		detail["institution_id"] = *institutionID
	}
	if strings.TrimSpace(req.InstitutionID) == "" && strings.TrimSpace(req.InstitutionSlug) == "" && strings.TrimSpace(req.InstitutionAETitle) == "" {
		detail["source_ip"] = sourceIP
		if sourceIPAttributionError != "" {
			detail["source_ip_attribution_error"] = sourceIPAttributionError
		}
	}
	if strings.TrimSpace(req.InstitutionSlug) != "" {
		detail["institution_slug"] = strings.TrimSpace(req.InstitutionSlug)
	}
	if strings.TrimSpace(req.InstitutionAETitle) != "" {
		detail["institution_ae_title"] = strings.TrimSpace(req.InstitutionAETitle)
	}
	model.CreateAuditEntry(r.Context(), s.db, "ingest.internal", actorEmail(r), "study", study.ID, clientIP(r), detail)

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "received",
		"study":   study,
		"message": "Study received — processing pipeline pending Phase 1 implementation",
	})
}

func normalizeAETitle(aeTitle string) string {
	return strings.ToUpper(strings.TrimSpace(aeTitle))
}

func (s *Server) resolveIngestInstitution(ctx context.Context, projectID, institutionID, institutionSlug, institutionAETitle string) (*model.Institution, error) {
	var (
		inst *model.Institution
		err  error
	)

	institutionID = strings.TrimSpace(institutionID)
	institutionSlug = strings.TrimSpace(institutionSlug)
	institutionAETitle = strings.TrimSpace(institutionAETitle)
	if institutionSlug != "" {
		institutionSlug = strings.ToLower(institutionSlug)
	}

	if institutionID != "" && institutionSlug != "" {
		return nil, fmt.Errorf("provide only one of institution_id or institution_slug")
	}

	if institutionID != "" {
		inst, err = model.GetInstitutionByID(ctx, s.db, institutionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("institution not found")
			}
			return nil, fmt.Errorf("lookup institution: %w", err)
		}
	} else if institutionSlug != "" {
		inst, err = model.GetInstitutionBySlug(ctx, s.db, institutionSlug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("institution not found for slug %q", institutionSlug)
			}
			return nil, fmt.Errorf("lookup institution by slug: %w", err)
		}
	}

	if institutionAETitle != "" {
		if inst == nil {
			inst, err = model.GetInstitutionByAETitle(ctx, s.db, institutionAETitle)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, fmt.Errorf("institution not found for ae_title %q", institutionAETitle)
				}
				return nil, fmt.Errorf("lookup institution by ae_title: %w", err)
			}
		} else if normalizeAETitle(inst.AETitle) != normalizeAETitle(institutionAETitle) {
			return nil, fmt.Errorf("institution_id and institution_ae_title do not match")
		}
	}

	if inst == nil {
		return nil, fmt.Errorf("institution_id, institution_slug, or institution_ae_title required")
	}
	if !inst.Enabled {
		return nil, fmt.Errorf("institution is disabled")
	}
	if inst.Type != "sender" && inst.Type != "both" {
		return nil, fmt.Errorf("institution type must be sender or both")
	}

	allowed, err := model.InstitutionCanSendToProject(ctx, s.db, inst.ID, projectID)
	if err != nil {
		return nil, fmt.Errorf("validate institution project link: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("institution is not linked to project as sender/admin")
	}
	return inst, nil
}

func (s *Server) resolveIngestInstitutionFromSourceIP(ctx context.Context, projectID, sourceIP string) (*model.Institution, error) {
	sourceIP = strings.TrimSpace(sourceIP)
	if sourceIP == "" {
		return nil, nil
	}
	parsedSourceIP := net.ParseIP(sourceIP)
	if parsedSourceIP == nil {
		return nil, fmt.Errorf("invalid source IP %q", sourceIP)
	}

	institutions, err := model.ListInstitutions(ctx, s.db)
	if err != nil {
		return nil, err
	}

	matches := make([]ipInstitutionMatch, 0)

	for i := range institutions {
		inst := &institutions[i]
		if !inst.Enabled {
			continue
		}
		if inst.Type != "sender" && inst.Type != "both" {
			continue
		}
		allowed, err := model.InstitutionCanSendToProject(ctx, s.db, inst.ID, projectID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}

		ranges := strings.Split(inst.IPRanges, ",")
		for _, r := range ranges {
			network := parseIPRange(strings.TrimSpace(r))
			if network == nil || !network.Contains(parsedSourceIP) {
				continue
			}
			prefixLen, _ := network.Mask.Size()
			matches = append(matches, ipInstitutionMatch{
				Institution: inst,
				PrefixLen:   prefixLen,
			})
		}
	}
	return selectBestInstitutionFromIPMatches(matches)
}

func parseIPRange(value string) *net.IPNet {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.Contains(value, "/") {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil
		}
		return network
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	if ip4 := ip.To4(); ip4 != nil {
		return &net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)}
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
}

func selectBestInstitutionFromIPMatches(matches []ipInstitutionMatch) (*model.Institution, error) {
	if len(matches) == 0 {
		return nil, nil
	}

	best := matches[0]
	ambiguous := false

	for i := 1; i < len(matches); i++ {
		candidate := matches[i]
		if candidate.PrefixLen > best.PrefixLen {
			best = candidate
			ambiguous = false
			continue
		}
		if candidate.PrefixLen == best.PrefixLen && candidate.Institution.ID != best.Institution.ID {
			ambiguous = true
		}
	}

	if ambiguous {
		return nil, fmt.Errorf("ambiguous source IP match for prefix length /%d", best.PrefixLen)
	}
	return best.Institution, nil
}
