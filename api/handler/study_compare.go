package handler

import (
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// CompareStudyField shows the value of a field for both studies.
type CompareStudyField struct {
	Field  string `json:"field"`
	ValueA string `json:"value_a"`
	ValueB string `json:"value_b"`
	Match  bool   `json:"match"`
}

// CompareStudies GET /api/studies/compare?a={id}&b={id}
// Returns a side-by-side metadata comparison of two studies.
func (s *Server) CompareStudies(w http.ResponseWriter, r *http.Request) {
	idA := r.URL.Query().Get("a")
	idB := r.URL.Query().Get("b")
	if idA == "" || idB == "" {
		s.writeError(w, http.StatusBadRequest, "both 'a' and 'b' query params are required")
		return
	}

	studyA, err := model.GetStudyByID(r.Context(), s.db, idA)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study A not found")
		return
	}
	studyB, err := model.GetStudyByID(r.Context(), s.db, idB)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study B not found")
		return
	}

	fields := buildComparison(studyA, studyB)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_a": studyA,
		"study_b": studyB,
		"diff":    fields,
	})
}

func buildComparison(a, b *model.Study) []CompareStudyField {
	cmp := func(field, va, vb string) CompareStudyField {
		return CompareStudyField{Field: field, ValueA: va, ValueB: vb, Match: va == vb}
	}
	return []CompareStudyField{
		cmp("modality", a.Modality, b.Modality),
		cmp("body_part", a.BodyPart, b.BodyPart),
		cmp("status", a.Status, b.Status),
		cmp("source", a.Source, b.Source),
		cmp("dicom_store", a.DicomStore, b.DicomStore),
		cmp("study_description", a.StudyDescription, b.StudyDescription),
		cmp("phi_scan_status", a.PhiScanStatus, b.PhiScanStatus),
		cmp("qc_status", a.QcStatus, b.QcStatus),
		cmp("bids_status", a.BidsStatus, b.BidsStatus),
		cmp("classification_status", a.ClassificationStatus, b.ClassificationStatus),
		cmp("protocol_status", a.ProtocolStatus, b.ProtocolStatus),
		cmp("export_status", a.ExportStatus, b.ExportStatus),
	}
}
