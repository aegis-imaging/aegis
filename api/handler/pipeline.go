package handler

import (
	"context"
	"log"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/webhook"
)

// AdvancePipeline inspects a study's processing state and dispatches the next
// eligible step(s). Called after routing rules evaluate and after each async
// service completes. Idempotent and race-safe via atomic SQL claims.
//
// No-op when cfg.PipelineAuto is false (manual mode).
//
// Dependency graph (3 phases):
//
//	Phase 0: Classification (blocks all other phases — fills modality/body_part)
//	Phase 1: PHI scan, protocol check, defacing (parallel, raw files)
//	Phase 2: QC check, BIDS conversion (after defacing completes, final files)
func (s *Server) AdvancePipeline(ctx context.Context, studyID string) {
	if !s.cfg.PipelineAuto {
		return
	}

	study, err := model.GetStudyByID(ctx, s.db, studyID)
	if err != nil {
		log.Printf("pipeline: fetch study %s: %v", studyID, err)
		return
	}

	// Terminal states — nothing to do.
	if study.Status == "approved" || study.Status == "rejected" {
		return
	}

	// Phase 0: Classification must complete before anything else.
	// It fills modality/body_part and re-evaluates routing rules, which may
	// add new requirements (e.g. require_defacing for HEAD studies).
	if study.ClassificationRequired {
		if study.ClassificationStatus == "pending" {
			if s.cfg.ClassificationServiceURL == "" {
				log.Printf("pipeline: classification required for %s but CLASSIFICATION_SERVICE_URL not set", study.StudyInstanceUID)
			} else {
				s.dispatchClassification(ctx, study)
			}
			return
		}
		if study.ClassificationStatus == "classifying" {
			return // wait for completion
		}
		// "classified" or "failed" — fall through to Phase 1
	}

	// Phase 1: Parallel services on raw files.
	s.dispatchPhiScan(ctx, study)
	s.dispatchProtocolCheck(ctx, study)
	s.dispatchDefacing(ctx, study)

	// Phase 2: Post-defacing services on final files.
	// Block if defacing is required but not yet complete.
	if study.DefacingRequired && study.Status != "defaced" {
		return
	}
	s.dispatchQcCheck(ctx, study)
	s.dispatchBidsConversion(ctx, study)

	// Fire study.processing_complete if all required steps are now terminal.
	s.maybeFireProcessingComplete(ctx, study.ID)
}

// maybeFireProcessingComplete fires the study.processing_complete webhook when all
// required pipeline steps are terminal and the study is awaiting human review.
// Called at the end of AdvancePipeline so it fires once per pipeline completion.
func (s *Server) maybeFireProcessingComplete(ctx context.Context, studyID string) {
	fresh, err := model.GetStudyByID(ctx, s.db, studyID)
	if err != nil {
		return
	}
	// Already terminal — approved/rejected/expired; nothing to signal.
	if fresh.Status == "approved" || fresh.Status == "rejected" || fresh.Status == "expired" {
		return
	}
	// Check each required step has reached a terminal status.
	classOK := !fresh.ClassificationRequired ||
		fresh.ClassificationStatus == "classified" || fresh.ClassificationStatus == "failed"
	phiOK := !fresh.PhiScanRequired ||
		fresh.PhiScanStatus == "clean" || fresh.PhiScanStatus == "flagged" || fresh.PhiScanStatus == "failed"
	protocolOK := !fresh.ProtocolRequired ||
		fresh.ProtocolStatus == "compliant" || fresh.ProtocolStatus == "minor_deviations" ||
		fresh.ProtocolStatus == "non_compliant" || fresh.ProtocolStatus == "failed"
	defacingOK := !fresh.DefacingRequired ||
		fresh.Status == "defaced" || fresh.Status == "clean"
	qcOK := !fresh.QcRequired ||
		fresh.QcStatus == "pass" || fresh.QcStatus == "warn" ||
		fresh.QcStatus == "fail" || fresh.QcStatus == "failed"
	bidsOK := !fresh.BidsRequired ||
		fresh.BidsStatus == "complete" || fresh.BidsStatus == "failed"
	exportOK := !fresh.ExportRequired ||
		fresh.ExportStatus == "exported" || fresh.ExportStatus == "failed"

	if classOK && phiOK && protocolOK && defacingOK && qcOK && bidsOK && exportOK {
		go webhook.Deliver(ctx, s.db, "study.processing_complete", fresh)
	}
}

func (s *Server) dispatchClassification(ctx context.Context, study *model.Study) {
	claimed, err := model.ClaimClassification(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: claim classification for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if !claimed {
		return
	}
	log.Printf("pipeline: dispatching classification for %s", study.StudyInstanceUID)
	model.CreateAuditEntry(ctx, s.db, "pipeline.dispatch", "pipeline", "study", study.ID, "", map[string]any{
		"service":   "classification",
		"study_uid": study.StudyInstanceUID,
	})
	go s.runClassification(study)
}

func (s *Server) dispatchPhiScan(ctx context.Context, study *model.Study) {
	if !study.PhiScanRequired || study.PhiScanStatus != "pending" {
		return
	}
	if s.cfg.PhiDetectionServiceURL == "" {
		return
	}
	claimed, err := model.ClaimPhiScan(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: claim phi_scan for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if !claimed {
		return
	}
	log.Printf("pipeline: dispatching phi_scan for %s", study.StudyInstanceUID)
	model.CreateAuditEntry(ctx, s.db, "pipeline.dispatch", "pipeline", "study", study.ID, "", map[string]any{
		"service":   "phi_scan",
		"study_uid": study.StudyInstanceUID,
	})
	go s.runPhiScan(study)
}

func (s *Server) dispatchProtocolCheck(ctx context.Context, study *model.Study) {
	if !study.ProtocolRequired || study.ProtocolStatus != "pending" {
		return
	}
	if s.cfg.ProtocolServiceURL == "" {
		return
	}
	// Load templates — if none exist, leave pending for admin to configure.
	templates, err := model.ListProtocolTemplatesByProject(ctx, s.db, study.ProjectID)
	if err != nil {
		log.Printf("pipeline: load protocol templates for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if len(templates) == 0 {
		return
	}
	claimed, err := model.ClaimProtocolCheck(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: claim protocol_check for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if !claimed {
		return
	}
	log.Printf("pipeline: dispatching protocol_check for %s", study.StudyInstanceUID)
	model.CreateAuditEntry(ctx, s.db, "pipeline.dispatch", "pipeline", "study", study.ID, "", map[string]any{
		"service":        "protocol_check",
		"study_uid":      study.StudyInstanceUID,
		"template_count": len(templates),
	})
	go s.runProtocolCheck(study, templates)
}

func (s *Server) dispatchDefacing(ctx context.Context, study *model.Study) {
	if !study.DefacingRequired || study.Status != "received" {
		return
	}
	if s.cfg.DefacingServiceURL == "" {
		return
	}
	claimed, err := model.ClaimDefacing(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: claim defacing for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if !claimed {
		return
	}
	log.Printf("pipeline: dispatching defacing for %s", study.StudyInstanceUID)
	model.CreateAuditEntry(ctx, s.db, "pipeline.dispatch", "pipeline", "study", study.ID, "", map[string]any{
		"service":   "defacing",
		"study_uid": study.StudyInstanceUID,
	})
	go s.runDefacing(study)
}

func (s *Server) dispatchQcCheck(ctx context.Context, study *model.Study) {
	if !study.QcRequired || study.QcStatus != "pending" {
		return
	}
	if s.cfg.QcServiceURL == "" {
		return
	}
	claimed, err := model.ClaimQcCheck(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: claim qc_check for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if !claimed {
		return
	}
	// Re-fetch study to get latest DicomStore (may have changed from "raw" to "clean").
	fresh, err := model.GetStudyByID(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: re-fetch study for qc_check %s: %v", study.StudyInstanceUID, err)
		return
	}
	log.Printf("pipeline: dispatching qc_check for %s (store=%s)", study.StudyInstanceUID, fresh.DicomStore)
	model.CreateAuditEntry(ctx, s.db, "pipeline.dispatch", "pipeline", "study", study.ID, "", map[string]any{
		"service":   "qc_check",
		"study_uid": study.StudyInstanceUID,
	})
	go s.runQcCheck(fresh)
}

func (s *Server) dispatchBidsConversion(ctx context.Context, study *model.Study) {
	if !study.BidsRequired || study.BidsStatus != "pending" {
		return
	}
	if s.cfg.BidsServiceURL == "" {
		return
	}
	claimed, err := model.ClaimBidsConversion(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: claim bids_conversion for %s: %v", study.StudyInstanceUID, err)
		return
	}
	if !claimed {
		return
	}
	// Re-fetch study to get latest DicomStore.
	fresh, err := model.GetStudyByID(ctx, s.db, study.ID)
	if err != nil {
		log.Printf("pipeline: re-fetch study for bids_conversion %s: %v", study.StudyInstanceUID, err)
		return
	}
	log.Printf("pipeline: dispatching bids_conversion for %s (store=%s)", study.StudyInstanceUID, fresh.DicomStore)
	model.CreateAuditEntry(ctx, s.db, "pipeline.dispatch", "pipeline", "study", study.ID, "", map[string]any{
		"service":   "bids_conversion",
		"study_uid": study.StudyInstanceUID,
	})
	go s.runBidsConversion(fresh)
}

// notifyPipelineFailure sends a plain-text alert email when a pipeline service step fails.
// No-op when PipelineAlertEmail is empty or SMTP is not configured.
func (s *Server) notifyPipelineFailure(ctx context.Context, studyUID, service, errMsg string) {
	if s.cfg.PipelineAlertEmail == "" {
		return
	}
	subj, body := email.PipelineFailure(studyUID, service, errMsg)
	if err := s.mailer.Send(ctx, s.cfg.PipelineAlertEmail, subj, body); err != nil {
		log.Printf("pipeline: send failure alert for %s/%s: %v", service, studyUID, err)
	}
}
