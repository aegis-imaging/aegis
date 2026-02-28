package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListROIResults returns filterable ROI results for a study.
// GET /api/studies/{id}/roi-results
func (s *Server) ListROIResults(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	f := model.ROIResultFilters{
		Tool:       r.URL.Query().Get("tool"),
		AtlasName:  r.URL.Query().Get("atlas_name"),
		MetricType: r.URL.Query().Get("metric_type"),
		ROIName:    r.URL.Query().Get("roi_name"),
		Hemisphere: r.URL.Query().Get("hemisphere"),
	}
	if lim := r.URL.Query().Get("limit"); lim != "" {
		if n, err := strconv.Atoi(lim); err == nil {
			f.Limit = n
		}
	}

	results, err := model.GetROIResultsByStudy(r.Context(), s.db, studyID, f)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query ROI results")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
		"total":   len(results),
	})
}

// GetROIResultsSummary returns an aggregate summary of ROI results for a study.
// GET /api/studies/{id}/roi-results/summary
func (s *Server) GetROIResultsSummary(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	summary, err := model.GetROIResultsSummary(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query ROI summary")
		return
	}
	s.writeJSON(w, http.StatusOK, summary)
}

// ListCompositeScores returns composite scores for a study.
// GET /api/studies/{id}/composite-scores
func (s *Server) ListCompositeScores(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	scores, err := model.GetCompositeScoresByStudy(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query composite scores")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"scores": scores,
		"total":  len(scores),
	})
}

// ListLongitudinalROIResults returns longitudinal ROI results for a study.
// GET /api/studies/{id}/longitudinal-roi-results
func (s *Server) ListLongitudinalROIResults(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	results, err := model.GetLongitudinalROIResultsByStudy(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query longitudinal ROI results")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
		"total":   len(results),
	})
}

// ListSubjectROIResults returns ROI results across all studies for a subject in a project.
// GET /api/subjects/{subjectID}/roi-results?project_id=
func (s *Server) ListSubjectROIResults(w http.ResponseWriter, r *http.Request) {
	subjectID := r.PathValue("subjectID")
	projectID := r.URL.Query().Get("project_id")
	if subjectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing subject ID")
		return
	}

	// Find all studies for this subject+project, then aggregate ROI results.
	studies, err := model.ListStudiesBySubject(r.Context(), s.db, subjectID, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query subject studies")
		return
	}

	var allResults []map[string]any
	for _, st := range studies {
		results, err := model.GetROIResultsByStudy(r.Context(), s.db, st.ID, model.ROIResultFilters{})
		if err != nil {
			continue
		}
		for _, roi := range results {
			allResults = append(allResults, map[string]any{
				"study_id":       st.ID,
				"study_uid":      st.StudyInstanceUID,
				"study_date":     st.StudyDate,
				"roi_id":         roi.ID,
				"tool":           roi.Tool,
				"atlas_name":     roi.AtlasName,
				"roi_name":       roi.ROIName,
				"metric_type":    roi.MetricType,
				"metric_value":   roi.MetricValue,
				"hemisphere":     roi.Hemisphere,
			})
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"subject_id": subjectID,
		"results":    allResults,
		"total":      len(allResults),
	})
}

// ExportProjectROIData exports ROI results for a project as CSV.
// GET /api/projects/{id}/roi-export
func (s *Server) ExportProjectROIData(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project ID")
		return
	}

	// Get all studies for the project with analytics complete.
	studies, err := model.ListStudiesByProject(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query studies")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="roi-results.csv"`)

	cw := csv.NewWriter(w)
	cw.Write([]string{
		"study_id", "study_uid", "subject_id", "tool", "atlas_name",
		"roi_name", "metric_type", "metric_value", "hemisphere", "scan_type",
	})

	var rowCount int
	const maxRows = 50000
	for _, st := range studies {
		if rowCount >= maxRows {
			break
		}
		results, err := model.GetROIResultsByStudy(r.Context(), s.db, st.ID, model.ROIResultFilters{Limit: 2000})
		if err != nil {
			continue
		}
		subjectID := ""
		if st.SubjectID != nil {
			subjectID = *st.SubjectID
		}
		for _, roi := range results {
			if rowCount >= maxRows {
				break
			}
			cw.Write([]string{
				st.ID,
				st.StudyInstanceUID,
				subjectID,
				roi.Tool,
				roi.AtlasName,
				roi.ROIName,
				roi.MetricType,
				fmt.Sprintf("%.6f", roi.MetricValue),
				roi.Hemisphere,
				roi.ScanType,
			})
			rowCount++
		}
	}
	cw.Flush()
}

// DeleteROIResults removes all ROI results for a study (for re-processing).
// DELETE /api/studies/{id}/roi-results
func (s *Server) DeleteROIResults(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	if err := model.DeleteROIResultsByStudy(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete ROI results")
		return
	}
	if err := model.DeleteCompositeScoresByStudy(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete composite scores")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "roi_results.deleted", actorEmail(r),
		"study", studyID, clientIP(r), nil)

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
