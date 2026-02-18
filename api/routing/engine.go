// Package routing implements the AEGIS routing rules engine.
//
// On study ingest, EvaluateRules loads all enabled rules ordered by priority
// and evaluates each one against the study. Matching rules are executed and
// logged. Multiple rules may fire for a single study.
//
// Actions:
//   - require_defacing  — sets defacing_required=true (overrides tag-based detection)
//   - auto_approve      — immediately approves (skips manual QC)
//   - require_qa        — no-op; marks study for manual QC (default pipeline behaviour)
//   - reject            — auto-rejects the study
//   - route_to          — dispatches the study to an external DICOMweb destination (async)
package routing

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/model"
)

// EvaluateRules loads all enabled routing rules and applies them to study.
// It mutates the study record in-place (e.g. setting DefacingRequired, Status)
// and persists each mutation immediately. Side effects (e.g. forwarding) run
// asynchronously so they don't block the ingest response.
func EvaluateRules(ctx context.Context, db *sql.DB, study *model.Study) {
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
		if !matches(r, study) {
			continue
		}
		applyRule(ctx, db, r, study)
	}
}

// matches returns true when all non-nil conditions of the rule apply to study.
func matches(r *model.RoutingRule, s *model.Study) bool {
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
func applyRule(ctx context.Context, db *sql.DB, r *model.RoutingRule, s *model.Study) {
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
		go forwardStudy(context.Background(), db, r.ID, &studyCopy, &destCopy)

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

// forwardStudy pushes study metadata to an external DICOMweb destination via a
// minimal STOW-RS metadata-only POST. In a production system this would stream
// the actual DICOM files; here it sends the study metadata payload so the
// destination can acknowledge receipt. Runs in a background goroutine.
func forwardStudy(ctx context.Context, db *sql.DB, ruleID string, s *model.Study, dest *model.Destination) {
	payload := map[string]any{
		"study_instance_uid": s.StudyInstanceUID,
		"modality":           s.Modality,
		"body_part":          s.BodyPart,
		"source":             s.Source,
		"sent_at":            time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(dest.DicomwebURL, "/")+"/studies", bytes.NewReader(body))
	if err != nil {
		log.Printf("routing: forward build request (study=%s dest=%s): %v", s.ID, dest.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if dest.DicomwebAuthHeader != "" {
		req.Header.Set("Authorization", dest.DicomwebAuthHeader)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("routing: forward request (study=%s dest=%s): %v", s.ID, dest.ID, err)
		model.CreateAuditEntry(ctx, db, "routing.forward_failed", "routing-engine", "study", s.ID, "", map[string]any{
			"destination": dest.Name,
			"error":       err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	outcome := fmt.Sprintf("HTTP %d", resp.StatusCode)
	model.CreateAuditEntry(ctx, db, "routing.forwarded", "routing-engine", "study", s.ID, "", map[string]any{
		"destination": dest.Name,
		"status_code": resp.StatusCode,
	})
	log.Printf("routing: forward complete (study=%s dest=%s): %s", s.ID, dest.ID, outcome)
}
