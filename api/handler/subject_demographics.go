package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetSubjectDemographics returns demographics for a subject in a project.
// GET /api/subjects/{subjectID}/demographics?project_id=
func (s *Server) GetSubjectDemographics(w http.ResponseWriter, r *http.Request) {
	subjectID := r.PathValue("subjectID")
	projectID := r.URL.Query().Get("project_id")
	if subjectID == "" || projectID == "" {
		s.writeError(w, http.StatusBadRequest, "subject_id and project_id are required")
		return
	}

	d, err := model.GetSubjectDemographics(r.Context(), s.db, subjectID, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query demographics")
		return
	}
	if d == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"demographics": nil})
		return
	}
	s.writeJSON(w, http.StatusOK, d)
}

// UpsertSubjectDemographics creates or updates demographics for a subject.
// PUT /api/subjects/{subjectID}/demographics
func (s *Server) UpsertSubjectDemographics(w http.ResponseWriter, r *http.Request) {
	subjectID := r.PathValue("subjectID")
	if subjectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing subject ID")
		return
	}

	var d model.SubjectDemographics
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	d.SubjectID = subjectID

	if d.ProjectID == "" {
		s.writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	if err := model.UpsertSubjectDemographics(r.Context(), s.db, &d); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save demographics")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "demographics.upserted", actorEmail(r),
		"subject", subjectID, clientIP(r), map[string]any{
			"project_id": d.ProjectID,
		})

	s.writeJSON(w, http.StatusOK, &d)
}

// ListProjectDemographics returns all subject demographics for a project.
// GET /api/projects/{id}/demographics
func (s *Server) ListProjectDemographics(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project ID")
		return
	}

	demos, err := model.ListProjectDemographics(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query demographics")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"demographics": demos,
		"total":        len(demos),
	})
}

// ExportProjectDemographicsCSV exports demographics for a project as CSV.
// GET /api/projects/{id}/demographics.csv
func (s *Server) ExportProjectDemographicsCSV(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project ID")
		return
	}

	demos, err := model.ListProjectDemographics(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query demographics")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="demographics.csv"`)

	cw := csv.NewWriter(w)
	cw.Write([]string{
		"subject_id", "project_id", "sex", "age_at_scan", "diagnosis",
		"education_years", "mmse_score", "moca_score", "cdr_global",
		"apoe_genotype", "notes",
	})

	for _, d := range demos {
		cw.Write([]string{
			d.SubjectID,
			d.ProjectID,
			d.Sex,
			intPtrStr(d.AgeAtScan),
			d.Diagnosis,
			int16PtrStr(d.EducationYears),
			int16PtrStr(d.MMSEScore),
			int16PtrStr(d.MoCAScore),
			float64PtrStr(d.CDRGlobal),
			d.APOEGenotype,
			d.Notes,
		})
	}
	cw.Flush()
}

func intPtrStr(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

func int16PtrStr(v *int16) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

func float64PtrStr(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *v)
}
