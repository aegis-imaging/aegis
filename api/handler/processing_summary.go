package handler

import (
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// pipelineStepSummary is the status of one pipeline processing step.
type pipelineStepSummary struct {
	Required   bool   `json:"required"`
	Status     string `json:"status"`               // current status value, or "not_required"
	Terminal   bool   `json:"terminal"`              // true when step has reached a final state
	InProgress bool   `json:"in_progress"`
	Failed     bool   `json:"failed"`
}

// processingSummaryResponse is the payload for GetStudyProcessingSummary.
type processingSummaryResponse struct {
	StudyID         string                         `json:"study_id"`
	StudyStatus     string                         `json:"study_status"`
	DicomStore      string                         `json:"dicom_store"`
	GeneratedAt     string                         `json:"generated_at"`
	PipelineComplete bool                          `json:"pipeline_complete"`
	Blockers        []string                       `json:"blockers"`
	Steps           map[string]pipelineStepSummary `json:"steps"`
}

// stepSummary builds a pipelineStepSummary for a named pipeline step.
// terminalStatuses is the set of status values that indicate completion (success or definitive result).
// inProgressStatuses is the set that mean work is actively running.
// failedStatuses is the set that mean an error occurred.
func stepSummary(required bool, status string, inProgressStatuses, failedStatuses, terminalStatuses []string) pipelineStepSummary {
	if !required {
		return pipelineStepSummary{Required: false, Status: "not_required", Terminal: true}
	}
	inProg := false
	failed := false
	terminal := false
	for _, s := range inProgressStatuses {
		if status == s {
			inProg = true
		}
	}
	for _, s := range failedStatuses {
		if status == s {
			failed = true
		}
	}
	for _, s := range terminalStatuses {
		if status == s {
			terminal = true
		}
	}
	return pipelineStepSummary{
		Required:   true,
		Status:     status,
		Terminal:   terminal,
		InProgress: inProg,
		Failed:     failed,
	}
}

// GetStudyProcessingSummary returns a structured summary of all pipeline steps
// for a single study — useful for AI agents and quick status checks.
//
// GET /api/studies/{id}/processing-summary
func (s *Server) GetStudyProcessingSummary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	steps := map[string]pipelineStepSummary{
		"classification": stepSummary(
			study.ClassificationRequired, study.ClassificationStatus,
			[]string{"classifying"},
			[]string{"failed"},
			[]string{"classified"},
		),
		"phi_scan": stepSummary(
			study.PhiScanRequired, study.PhiScanStatus,
			[]string{"scanning"},
			[]string{"failed"},
			[]string{"clean", "flagged"},
		),
		"protocol_check": stepSummary(
			study.ProtocolRequired, study.ProtocolStatus,
			[]string{"checking"},
			[]string{"failed"},
			[]string{"compliant", "minor_deviations", "non_compliant"},
		),
		"defacing": stepSummary(
			study.DefacingRequired, study.Status,
			[]string{"defacing"},
			[]string{},
			// defacing is terminal when the study is no longer in a defacing state
			// (it has advanced to clean/defaced/approved/rejected)
			[]string{"clean", "defaced", "approved", "rejected"},
		),
		"qc_check": stepSummary(
			study.QcRequired, study.QcStatus,
			[]string{"checking"},
			[]string{"failed"},
			[]string{"pass", "warn", "fail"},
		),
		"bids_conversion": stepSummary(
			study.BidsRequired, study.BidsStatus,
			[]string{"converting"},
			[]string{"failed"},
			[]string{"complete"},
		),
		"export": stepSummary(
			study.ExportRequired, study.ExportStatus,
			[]string{"exporting"},
			[]string{"failed"},
			[]string{"exported"},
		),
	}

	// Determine blockers: required steps that are not yet terminal and not in_progress.
	blockers := []string{}
	for name, st := range steps {
		if st.Required && !st.Terminal && !st.InProgress {
			blockers = append(blockers, name)
		}
	}

	// Pipeline is complete when: study is in a terminal state AND all required steps are terminal.
	studyTerminal := study.Status == "approved" || study.Status == "rejected" || study.Status == "expired"
	allStepsTerminal := true
	for _, st := range steps {
		if st.Required && !st.Terminal {
			allStepsTerminal = false
			break
		}
	}
	pipelineComplete := studyTerminal && allStepsTerminal

	s.writeJSON(w, http.StatusOK, processingSummaryResponse{
		StudyID:          study.ID,
		StudyStatus:      study.Status,
		DicomStore:       study.DicomStore,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		PipelineComplete: pipelineComplete,
		Blockers:         blockers,
		Steps:            steps,
	})
}
