// Package routing implements the AEGIS routing rules engine.
//
// On study ingest, EvaluateRules loads all enabled rules ordered by priority
// and evaluates each one against the study. Matching rules are executed and
// logged. Multiple rules may fire for a single study.
//
// Actions:
//   - require_defacing      — sets defacing_required=true (overrides tag-based detection)
//   - auto_approve          — immediately approves (skips manual QC)
//   - require_qa            — no-op; marks study for manual QC (default pipeline behaviour)
//   - reject                — auto-rejects the study
//   - route_to              — forwards DICOM files to an external DICOMweb destination via STOW-RS (async)
//   - require_export        — sets export_required=true (auto-forward on approval)
package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/storage"
)

// EvaluateRules loads all enabled routing rules and applies them to study.
// It mutates the study record in-place (e.g. setting DefacingRequired, Status)
// and persists each mutation immediately. Side effects (e.g. forwarding) run
// asynchronously so they don't block the ingest response.
func EvaluateRules(ctx context.Context, db *sql.DB, store storage.Storage, study *model.Study) {
	rules, err := model.ListEnabledRoutingRules(ctx, db)
	if err != nil {
		log.Printf("routing: load rules: %v", err)
		return
	}
	if len(rules) == 0 {
		return
	}

	for i := range rules {
		r := &rules[i]
		if !Matches(r, study) {
			continue
		}
		applyRule(ctx, db, store, r, study)
	}
}

// Matches returns true when all non-nil conditions of the rule apply to study.
func Matches(r *model.RoutingRule, s *model.Study) bool {
	if r.ProjectID != nil && *r.ProjectID != s.ProjectID {
		return false
	}
	if r.Modality != nil && !strings.EqualFold(*r.Modality, s.Modality) {
		return false
	}
	if r.BodyPart != nil && !strings.EqualFold(*r.BodyPart, s.BodyPart) {
		return false
	}
	if r.Source != nil && *r.Source != s.Source {
		return false
	}
	return true
}

// applyRule executes a single matching rule against the study.
func applyRule(ctx context.Context, db *sql.DB, store storage.Storage, r *model.RoutingRule, s *model.Study) {
	var outcome string

	switch r.Action {

	case "require_defacing":
		if !s.DefacingRequired {
			s.DefacingRequired = true
			if err := model.SetDefacingRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set defacing_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "defacing_required set to true"
			}
		} else {
			outcome = "defacing_required already true (no-op)"
		}

	case "auto_approve":
		if s.Status == "received" || s.Status == "clean" {
			newStatus := "approved"
			if err := model.UpdateStudyStatus(ctx, db, s.ID, newStatus); err != nil {
				log.Printf("routing: auto_approve (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				s.Status = newStatus
				outcome = "status set to approved"
				model.CreateAuditEntry(ctx, db, "study.auto_approved", "routing-engine", "study", s.ID, "", map[string]any{
					"rule_id":   r.ID,
					"rule_name": r.Name,
				})
			}
		} else {
			outcome = fmt.Sprintf("skipped auto_approve: status=%s", s.Status)
		}

	case "require_qa":
		// No-op — this is the default pipeline behaviour; the rule acts as documentation.
		outcome = "require_qa acknowledged (manual review required)"

	case "reject":
		if s.Status != "rejected" {
			if err := model.UpdateStudyStatus(ctx, db, s.ID, "rejected"); err != nil {
				log.Printf("routing: auto_reject (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				s.Status = "rejected"
				outcome = "status set to rejected"
				model.CreateAuditEntry(ctx, db, "study.auto_rejected", "routing-engine", "study", s.ID, "", map[string]any{
					"rule_id":   r.ID,
					"rule_name": r.Name,
				})
			}
		} else {
			outcome = "already rejected (no-op)"
		}

	case "require_phi_scan":
		if !s.PhiScanRequired {
			s.PhiScanRequired = true
			s.PhiScanStatus = "pending"
			if err := model.SetPhiScanRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set phi_scan_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "phi_scan_required set to true"
			}
		} else {
			outcome = "phi_scan_required already true (no-op)"
		}

	case "require_qc_check":
		if !s.QcRequired {
			s.QcRequired = true
			s.QcStatus = "pending"
			if err := model.SetQcRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set qc_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "qc_required set to true"
			}
		} else {
			outcome = "qc_required already true (no-op)"
		}

	case "require_bids_conversion":
		if !s.BidsRequired {
			s.BidsRequired = true
			s.BidsStatus = "pending"
			if err := model.SetBidsRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set bids_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "bids_required set to true"
			}
		} else {
			outcome = "bids_required already true (no-op)"
		}

	case "require_classification":
		if !s.ClassificationRequired {
			s.ClassificationRequired = true
			s.ClassificationStatus = "pending"
			if err := model.SetClassificationRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set classification_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "classification_required set to true"
			}
		} else {
			outcome = "classification_required already true (no-op)"
		}

	case "require_protocol_check":
		if !s.ProtocolRequired {
			s.ProtocolRequired = true
			s.ProtocolStatus = "pending"
			if err := model.SetProtocolRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set protocol_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "protocol_required set to true"
			}
		} else {
			outcome = "protocol_required already true (no-op)"
		}

	case "require_export":
		if !s.ExportRequired {
			s.ExportRequired = true
			s.ExportStatus = "pending"
			if err := model.SetExportRequired(ctx, db, s.ID, true); err != nil {
				log.Printf("routing: set export_required (study=%s rule=%s): %v", s.ID, r.ID, err)
				outcome = "error: " + err.Error()
			} else {
				outcome = "export_required set to true"
			}
		} else {
			outcome = "export_required already true (no-op)"
		}

	case "route_to":
		if r.DestinationID == nil {
			outcome = "error: route_to rule has no destination_id"
			break
		}
		dest, err := model.GetDestinationByID(ctx, db, *r.DestinationID)
		if err != nil {
			log.Printf("routing: load destination (study=%s rule=%s dest=%s): %v",
				s.ID, r.ID, *r.DestinationID, err)
			outcome = "error: destination not found"
			break
		}
		if !dest.Enabled {
			outcome = "destination disabled (skipped)"
			break
		}
		outcome = "forwarding dispatched"
		studyCopy := *s
		destCopy := *dest
		go func() {
			_ = ForwardStudy(context.Background(), db, store, &studyCopy, &destCopy)
		}()

	default:
		outcome = fmt.Sprintf("unknown action: %s", r.Action)
	}

	detail, _ := json.Marshal(map[string]any{
		"rule_name": r.Name,
		"priority":  r.Priority,
		"outcome":   outcome,
	})
	entry := &model.RoutingLogEntry{
		StudyID:       s.ID,
		RuleID:        r.ID,
		Action:        r.Action,
		DestinationID: r.DestinationID,
		Outcome:       outcome,
		Detail:        detail,
	}
	if err := model.CreateRoutingLogEntry(ctx, db, entry); err != nil {
		log.Printf("routing: write routing_log (study=%s rule=%s): %v", s.ID, r.ID, err)
	}
}

// ForwardStudy pushes a study's DICOM files to an external DICOMweb destination
// via STOW-RS (multipart/related with application/dicom parts). Files are
// streamed from storage via io.Pipe to avoid buffering entire studies in memory.
// Returns nil on success (HTTP 2xx from destination) or an error.
func ForwardStudy(ctx context.Context, db *sql.DB, store storage.Storage, s *model.Study, dest *model.Destination) error {
	prefix := fmt.Sprintf("dicom/%s/%s", s.DicomStore, s.StudyInstanceUID)
	keys, err := store.List(ctx, prefix)
	if err != nil {
		log.Printf("routing: list files for forward (study=%s dest=%s): %v", s.ID, dest.ID, err)
		model.CreateAuditEntry(ctx, db, "routing.forward_failed", "routing-engine", "study", s.ID, "", map[string]any{
			"destination": dest.Name,
			"error":       err.Error(),
		})
		return fmt.Errorf("list files: %w", err)
	}
	if len(keys) == 0 {
		log.Printf("routing: no files to forward (study=%s dest=%s)", s.ID, dest.ID)
		return fmt.Errorf("no DICOM files found")
	}

	boundary := fmt.Sprintf("aegis-stow-%s-%d", s.StudyInstanceUID[:min(8, len(s.StudyInstanceUID))], time.Now().UnixNano())
	pr, pw := io.Pipe()

	// Write DICOM files as multipart parts in background.
	go func() {
		defer pw.Close()
		mw := multipart.NewWriter(pw)
		mw.SetBoundary(boundary)
		for _, key := range keys {
			partHeader := textproto.MIMEHeader{
				"Content-Type": {"application/dicom"},
			}
			part, err := mw.CreatePart(partHeader)
			if err != nil {
				log.Printf("routing: create multipart part %s: %v", key, err)
				continue
			}
			rc, err := store.Retrieve(ctx, key)
			if err != nil {
				log.Printf("routing: retrieve %s for forward: %v", key, err)
				continue
			}
			io.Copy(part, rc)
			rc.Close()
		}
		mw.Close()
	}()

	url := strings.TrimRight(dest.DicomwebURL, "/") + "/studies"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		pr.Close()
		log.Printf("routing: forward build request (study=%s dest=%s): %v", s.ID, dest.ID, err)
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type",
		fmt.Sprintf(`multipart/related; type="application/dicom"; boundary=%s`, boundary))
	if dest.DicomwebAuthHeader != "" {
		req.Header.Set("Authorization", dest.DicomwebAuthHeader)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("routing: forward request (study=%s dest=%s): %v", s.ID, dest.ID, err)
		model.CreateAuditEntry(ctx, db, "routing.forward_failed", "routing-engine", "study", s.ID, "", map[string]any{
			"destination": dest.Name,
			"file_count":  len(keys),
			"error":       err.Error(),
		})
		return fmt.Errorf("forward request: %w", err)
	}
	defer resp.Body.Close()

	model.CreateAuditEntry(ctx, db, "routing.forwarded", "routing-engine", "study", s.ID, "", map[string]any{
		"destination": dest.Name,
		"file_count":  len(keys),
		"status_code": resp.StatusCode,
	})
	log.Printf("routing: forward complete (study=%s dest=%s): HTTP %d (%d files)",
		s.ID, dest.ID, resp.StatusCode, len(keys))

	if resp.StatusCode >= 300 {
		return fmt.Errorf("destination returned HTTP %d", resp.StatusCode)
	}
	return nil
}
