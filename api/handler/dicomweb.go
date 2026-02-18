package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/msenjem/aegis/api/model"
)

// ── DICOMweb JSON tag helpers ─────────────────────────────────────────────────

func dicomTag(vr, value string) map[string]any {
	if value == "" {
		return map[string]any{"vr": vr}
	}
	return map[string]any{"vr": vr, "Value": []any{value}}
}

func dicomTagInt(vr string, value int) map[string]any {
	return map[string]any{"vr": vr, "Value": []any{value}}
}

func dicomTagPN(name string) map[string]any {
	if name == "" {
		return map[string]any{"vr": "PN"}
	}
	return map[string]any{"vr": "PN", "Value": []any{map[string]any{"Alphabetic": name}}}
}

// studyQIDO converts a Study to a DICOMweb QIDO-RS JSON object.
// All patient-identifying fields are omitted — only anonymized UIDs and counts.
func studyQIDO(s model.Study) map[string]any {
	return map[string]any{
		"0020000D": dicomTag("UI", s.StudyInstanceUID), // StudyInstanceUID
		"00080020": dicomTag("DA", ""),                 // StudyDate — anonymized
		"00080030": dicomTag("TM", ""),                 // StudyTime — anonymized
		"00080050": dicomTag("SH", ""),                 // AccessionNumber — anonymized
		"00100010": dicomTagPN(""),                     // PatientName — anonymized
		"00100020": dicomTag("LO", ""),                 // PatientID — anonymized
		"00080060": dicomTag("CS", s.Modality),         // Modality
		"00200010": dicomTag("SH", ""),                 // StudyID
		"00201206": dicomTagInt("IS", 1),               // NumberOfStudyRelatedSeries
		"00201208": dicomTagInt("IS", s.InstanceCount), // NumberOfStudyRelatedInstances
	}
}

// seriesQIDO returns a single fake series for a study (one series per study).
func seriesQIDO(s model.Study) map[string]any {
	return map[string]any{
		"0020000E": dicomTag("UI", s.StudyInstanceUID+".1"), // SeriesInstanceUID
		"00200011": dicomTagInt("IS", 1),                    // SeriesNumber
		"00080060": dicomTag("CS", s.Modality),              // Modality
		"00201209": dicomTagInt("IS", s.InstanceCount),      // NumberOfSeriesRelatedInstances
	}
}

// instanceQIDO returns instance-level metadata for a given 0-based file index.
// SOPInstanceUID = {studyUID}.1.{index}
func instanceQIDO(studyUID string, index int) map[string]any {
	return map[string]any{
		"0020000D": dicomTag("UI", studyUID),                           // StudyInstanceUID
		"0020000E": dicomTag("UI", studyUID+".1"),                      // SeriesInstanceUID
		"00080018": dicomTag("UI", fmt.Sprintf("%s.1.%d", studyUID, index)), // SOPInstanceUID
		"00200013": dicomTagInt("IS", index+1),                        // InstanceNumber
	}
}

// ── QIDO-RS: Studies ──────────────────────────────────────────────────────────

// DicomwebStudies handles GET /dicomweb/studies
func (s *Server) DicomwebStudies(w http.ResponseWriter, r *http.Request) {
	studyUID := r.URL.Query().Get("StudyInstanceUIDs")

	var studies []model.Study
	if studyUID != "" {
		st, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
		if err != nil {
			if err == sql.ErrNoRows {
				w.Header().Set("Content-Type", "application/dicom+json")
				w.Write([]byte("[]"))
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		studies = []model.Study{*st}
	} else {
		var err error
		studies, err = model.ListStudies(r.Context(), s.db, "", 100, 0)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	result := make([]map[string]any, 0, len(studies))
	for _, st := range studies {
		result = append(result, studyQIDO(st))
	}
	w.Header().Set("Content-Type", "application/dicom+json")
	json.NewEncoder(w).Encode(result)
}

// ── QIDO-RS: Series ───────────────────────────────────────────────────────────

// DicomwebSeries handles GET /dicomweb/studies/{studyUID}/series
func (s *Server) DicomwebSeries(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	st, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		http.Error(w, "study not found", http.StatusNotFound)
		return
	}
	result := []map[string]any{seriesQIDO(*st)}
	w.Header().Set("Content-Type", "application/dicom+json")
	json.NewEncoder(w).Encode(result)
}

// ── QIDO-RS: Instances ────────────────────────────────────────────────────────

// DicomwebInstances handles GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances
func (s *Server) DicomwebInstances(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	st, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		http.Error(w, "study not found", http.StatusNotFound)
		return
	}
	result := make([]map[string]any, st.InstanceCount)
	for i := 0; i < st.InstanceCount; i++ {
		result[i] = instanceQIDO(studyUID, i)
	}
	w.Header().Set("Content-Type", "application/dicom+json")
	json.NewEncoder(w).Encode(result)
}

// ── WADO-RS: Retrieve Instance ────────────────────────────────────────────────

// DicomwebRetrieveInstance handles GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}
//
// SOPInstanceUID format: {studyUID}.1.{fileIndex}
// Files are stored as: dicom/{dicomStore}/{studyUID}/{fileIndex}.dcm
func (s *Server) DicomwebRetrieveInstance(w http.ResponseWriter, r *http.Request) {
	s.dicomwebRetrieve(w, r, "")
}

// DicomwebRawRetrieveInstance is identical to DicomwebRetrieveInstance but
// always reads from the "raw" store regardless of the study's current dicom_store.
// Used by the /dicomweb-raw/* routes to display pre-defacing images for review.
func (s *Server) DicomwebRawRetrieveInstance(w http.ResponseWriter, r *http.Request) {
	s.dicomwebRetrieve(w, r, "raw")
}

// dicomwebRetrieve is the shared WADO-RS implementation.
// storeOverride, when non-empty, reads from that store instead of st.DicomStore.
func (s *Server) dicomwebRetrieve(w http.ResponseWriter, r *http.Request, storeOverride string) {
	studyUID := r.PathValue("studyUID")
	sopUID := r.PathValue("sopUID")

	// Extract 0-based file index from the last segment of the SOP UID.
	parts := strings.Split(sopUID, ".")
	index, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || index < 0 {
		http.Error(w, "invalid SOP UID", http.StatusBadRequest)
		return
	}

	st, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		http.Error(w, "study not found", http.StatusNotFound)
		return
	}

	dicomStore := st.DicomStore
	if storeOverride != "" {
		dicomStore = storeOverride
	}

	key := fmt.Sprintf("dicom/%s/%s/%d.dcm", dicomStore, studyUID, index)
	rc, err := s.store.Retrieve(r.Context(), key)
	if err != nil {
		log.Printf("dicomweb wado-rs retrieve %s: %v", key, err)
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer rc.Close()

	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type",
		fmt.Sprintf(`multipart/related; type="application/dicom"; boundary=%s`, mw.Boundary()))

	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Type", "application/dicom")
	pw, err := mw.CreatePart(hdr)
	if err != nil {
		log.Printf("dicomweb multipart create part: %v", err)
		return
	}
	io.Copy(pw, rc)
	mw.Close()
}
