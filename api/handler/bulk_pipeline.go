package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

type bulkPipelineTriggerRequest struct {
	StudyIDs []string `json:"study_ids"`
	Step     string   `json:"step"`
}

type bulkPipelineTriggerResponse struct {
	Triggered int              `json:"triggered"`
	Skipped   int              `json:"skipped"`
	Errors    []bulkStudyError `json:"errors"`
}

var validPipelineSteps = map[string]bool{
	"classify": true,
	"phi_scan": true,
	"protocol": true,
	"deface":   true,
	"qc":       true,
	"bids":     true,
	"export":   true,
}

// BulkPipelineTrigger resets and re-dispatches a pipeline step for multiple
// studies in one request.
//
// POST /api/studies/bulk-pipeline-trigger
// Body: {"study_ids":["uuid1","uuid2"],"step":"classify|phi_scan|protocol|deface|qc|bids|export"}
// Returns: {"triggered":N,"skipped":M,"errors":[]}
//
// Skipped = step not required for that study, or step is already in-flight.
// Triggered = step was reset to pending and the pipeline was advanced.
func (s *Server) BulkPipelineTrigger(w http.ResponseWriter, r *http.Request) {
	var req bulkPipelineTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validPipelineSteps[req.Step] {
		s.writeError(w, http.StatusBadRequest, "step must be one of: classify, phi_scan, protocol, deface, qc, bids, export")
		return
	}
	if len(req.StudyIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_ids must not be empty")
		return
	}
	if len(req.StudyIDs) > 200 {
		s.writeError(w, http.StatusBadRequest, "too many study_ids (max 200)")
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)
	res := bulkPipelineTriggerResponse{Errors: []bulkStudyError{}}

	for _, id := range req.StudyIDs {
		study, err := model.GetStudyByID(r.Context(), s.db, id)
		if err != nil {
			res.Errors = append(res.Errors, bulkStudyError{StudyID: id, Error: "not found"})
			continue
		}

		if err := resetStep(r.Context(), s.db, study, req.Step); err != nil {
			switch err {
			case errNotRequired, errInFlight:
				res.Skipped++
			default:
				res.Errors = append(res.Errors, bulkStudyError{StudyID: id, Error: "reset failed"})
			}
			continue
		}

		// Step successfully reset to pending — advance the pipeline.
		go s.AdvancePipeline(r.Context(), id)
		res.Triggered++
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.bulk_pipeline_trigger", actor, "study", "", ip, map[string]any{
		"step":      req.Step,
		"triggered": res.Triggered,
		"skipped":   res.Skipped,
		"errors":    len(res.Errors),
	})

	s.writeJSON(w, http.StatusOK, res)
}
