package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

var (
	errInFlight    = errors.New("in-flight")
	errUnknownStep = errors.New("unknown-step")
	errNotRequired = errors.New("not-required")
)

type resetPipelineStepRequest struct {
	Step string `json:"step"`
}

// ResetPipelineStep resets a single pipeline step back to "pending" so the
// auto-pipeline or an operator can re-dispatch it. Returns 409 when the step
// is currently in-flight.
//
// POST /api/studies/{id}/reset-pipeline-step
// Body: {"step": "deface"|"phi_scan"|"qc"|"bids"|"classify"|"protocol"|"export"}
func (s *Server) ResetPipelineStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req resetPipelineStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	study, _, ok := s.requireStudyWriteAccessByID(w, r, id, projectWriteIntentStudyMutation)
	if !ok {
		return
	}

	if err := resetStep(r.Context(), s.db, study, req.Step); err != nil {
		switch err {
		case errInFlight:
			s.writeError(w, http.StatusConflict, "step is currently in-flight; wait for it to complete before resetting")
		case errUnknownStep:
			s.writeError(w, http.StatusBadRequest, "unknown step; valid: deface, phi_scan, qc, bids, classify, protocol, export")
		case errNotRequired:
			s.writeError(w, http.StatusBadRequest, "step is not required for this study; enable it via a routing rule or pipeline trigger first")
		default:
			s.writeError(w, http.StatusInternalServerError, "reset failed")
		}
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.pipeline_reset", actorEmail(r), "study", id, clientIP(r), map[string]any{
		"step": req.Step,
	})

	// Re-advance the pipeline so auto-dispatch picks up the newly pending step.
	if s.cfg.PipelineAuto {
		go s.AdvancePipeline(r.Context(), id)
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "step": req.Step})
}

// resetStep performs the appropriate status transition for a given pipeline step.
func resetStep(ctx context.Context, db *sql.DB, study *model.Study, step string) error {
	switch step {
	case "deface":
		if !study.DefacingRequired {
			return errNotRequired
		}
		if study.Status == "defacing" {
			return errInFlight
		}
		// Reset study status to received and revert dicom_store to raw so
		// the defacing service reads from the original files.
		_, err := db.ExecContext(ctx, `
			UPDATE studies SET status = 'received', dicom_store = 'raw', updated_at = now()
			WHERE id = $1`, study.ID)
		return err

	case "phi_scan":
		if !study.PhiScanRequired {
			return errNotRequired
		}
		if study.PhiScanStatus == "scanning" {
			return errInFlight
		}
		return model.UpdatePhiScanStatus(ctx, db, study.ID, "pending")

	case "qc":
		if !study.QcRequired {
			return errNotRequired
		}
		if study.QcStatus == "checking" {
			return errInFlight
		}
		return model.UpdateQcStatus(ctx, db, study.ID, "pending")

	case "bids":
		if !study.BidsRequired {
			return errNotRequired
		}
		if study.BidsStatus == "converting" {
			return errInFlight
		}
		return model.UpdateBidsStatus(ctx, db, study.ID, "pending")

	case "classify":
		if !study.ClassificationRequired {
			return errNotRequired
		}
		if study.ClassificationStatus == "classifying" {
			return errInFlight
		}
		return model.UpdateClassificationStatus(ctx, db, study.ID, "pending")

	case "protocol":
		if !study.ProtocolRequired {
			return errNotRequired
		}
		if study.ProtocolStatus == "checking" {
			return errInFlight
		}
		return model.UpdateProtocolStatus(ctx, db, study.ID, "pending")

	case "export":
		if !study.ExportRequired {
			return errNotRequired
		}
		if study.ExportStatus == "exporting" {
			return errInFlight
		}
		return model.UpdateExportStatus(ctx, db, study.ID, "pending")

	default:
		return errUnknownStep
	}
}
