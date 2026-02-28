package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// ExportStudiesCSV streams all studies matching the current filters as a CSV file.
// Accepts the same filter query params as GET /api/studies.
// Cap at 10 000 rows to prevent runaway exports.
func (s *Server) ExportStudiesCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var dateFrom, dateTo time.Time
	if v := q.Get("date_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dateFrom = t.UTC()
		}
	}
	if v := q.Get("date_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dateTo = t.UTC()
		}
	}

	f := model.StudyFilters{
		ProjectID:     q.Get("project_id"),
		Status:        q.Get("status"),
		Modality:      q.Get("modality"),
		BodyPart:      q.Get("body_part"),
		Source:        q.Get("source"),
		Search:        q.Get("search"),
		Label:         q.Get("label"),
		InstitutionID: q.Get("institution_id"),
		DateFrom:      dateFrom,
		DateTo:        dateTo,
	}
	access, ok := s.requireResearcherProjectScope(w, r, f.ProjectID)
	if !ok {
		return
	}
	if access != nil && access.IsSiteScoped() {
		f.InstitutionID = *access.InstitutionID
	}

	studies, err := model.ListStudies(r.Context(), s.db, f, 10000, 0)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list studies")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="studies.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"id", "study_instance_uid", "modality", "body_part", "study_description",
		"status", "source", "series_count", "instance_count", "dicom_store",
		"defacing_required", "phi_scan_status", "pixel_redaction_status", "qc_status", "bids_status",
		"classification_status", "protocol_status", "analytics_status", "export_status",
		"project_id", "institution_id", "created_at", "updated_at",
	})

	for _, st := range studies {
		instID := ""
		if st.InstitutionID != nil {
			instID = *st.InstitutionID
		}
		_ = cw.Write([]string{
			st.ID,
			st.StudyInstanceUID,
			st.Modality,
			st.BodyPart,
			st.StudyDescription,
			st.Status,
			st.Source,
			strconv.Itoa(st.SeriesCount),
			strconv.Itoa(st.InstanceCount),
			st.DicomStore,
			strconv.FormatBool(st.DefacingRequired),
			st.PhiScanStatus,
			st.PixelRedactionStatus,
			st.QcStatus,
			st.BidsStatus,
			st.ClassificationStatus,
			st.ProtocolStatus,
			st.AnalyticsStatus,
			st.ExportStatus,
			st.ProjectID,
			instID,
			st.CreatedAt.UTC().Format(time.RFC3339),
			st.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	cw.Flush()
}
