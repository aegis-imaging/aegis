package handler

import (
	"bytes"
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

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/model"
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

// modalitySOPClass maps DICOM modality codes to their primary SOP Class UID.
// OHIF requires SOPClassUID in the instance QIDO response to select the right
// image loader. Defaults to MR Image Storage when modality is unknown.
func modalitySOPClass(modality string) string {
	switch modality {
	case "CT":
		return "1.2.840.10008.5.1.4.1.1.2"   // CT Image Storage
	case "PT", "PET":
		return "1.2.840.10008.5.1.4.1.1.128"  // Positron Emission Tomography Image Storage
	case "US":
		return "1.2.840.10008.5.1.4.1.1.6.1"  // Ultrasound Image Storage
	case "CR", "DX":
		return "1.2.840.10008.5.1.4.1.1.1"    // Computed Radiography Image Storage
	default:
		return "1.2.840.10008.5.1.4.1.1.4"    // MR Image Storage
	}
}

// instanceQIDO returns instance-level metadata for a given 0-based file index.
// SOPInstanceUID = {studyUID}.1.{index}
func instanceQIDO(studyUID, modality string, index int) map[string]any {
	return map[string]any{
		"0020000D": dicomTag("UI", studyUID),                                // StudyInstanceUID
		"0020000E": dicomTag("UI", studyUID+".1"),                           // SeriesInstanceUID
		"00080016": dicomTag("UI", modalitySOPClass(modality)),              // SOPClassUID — required by OHIF
		"00080018": dicomTag("UI", fmt.Sprintf("%s.1.%d", studyUID, index)), // SOPInstanceUID
		"00200013": dicomTagInt("IS", index+1),                              // InstanceNumber
	}
}

// ── QIDO-RS: Studies ──────────────────────────────────────────────────────────

// DicomwebStudies handles GET /dicomweb/studies
func (s *Server) DicomwebStudies(w http.ResponseWriter, r *http.Request) {
	// Accept all standard QIDO-RS filter forms for StudyInstanceUID:
	//   "StudyInstanceUIDs" — plural form (legacy / our original)
	//   "StudyInstanceUID"  — singular, DICOM PS 3.18 keyword (what OHIF v3 sends)
	//   "0020000D"          — DICOM tag number form (some DICOMweb clients)
	studyUID := r.URL.Query().Get("StudyInstanceUIDs")
	if studyUID == "" {
		studyUID = r.URL.Query().Get("StudyInstanceUID")
	}
	if studyUID == "" {
		studyUID = r.URL.Query().Get("0020000D")
	}

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
		studies, err = model.ListStudies(r.Context(), s.db, model.StudyFilters{}, 100, 0)
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
		result[i] = instanceQIDO(studyUID, st.Modality, i)
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

	// If the client explicitly requests multipart/related (OHIF does), wrap in a
	// multipart envelope.  Otherwise (plain fetch, Accept: */*) return raw bytes
	// so browser-side DICOM parsers (dicom-parser, cornerstoneWADOImageLoader) can
	// consume the response directly without stripping the multipart wrapper.
	if strings.Contains(r.Header.Get("Accept"), "multipart/related") {
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
		return
	}

	// Raw DICOM bytes — consumed directly by browser-side parsers.
	w.Header().Set("Content-Type", "application/dicom")
	io.Copy(w, rc)
}

// ── WADO-RS: Instance Metadata ────────────────────────────────────────────────

// DicomwebInstanceMetadata handles GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}/metadata
// Returns DICOM tag metadata in DICOMweb JSON format without pixel data.
// OHIF v3 requires this endpoint to build the image manifest before loading pixels.
func (s *Server) DicomwebInstanceMetadata(w http.ResponseWriter, r *http.Request) {
	s.dicomwebMetadata(w, r, "")
}

// DicomwebRawInstanceMetadata is identical but forces the "raw" DICOM store.
func (s *Server) DicomwebRawInstanceMetadata(w http.ResponseWriter, r *http.Request) {
	s.dicomwebMetadata(w, r, "raw")
}

func (s *Server) dicomwebMetadata(w http.ResponseWriter, r *http.Request, storeOverride string) {
	studyUID := r.PathValue("studyUID")
	sopUID := r.PathValue("sopUID")

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
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		http.Error(w, "failed to read DICOM file", http.StatusInternalServerError)
		return
	}

	dataset, err := dicomlib.Parse(bytes.NewReader(data), int64(len(data)), nil, dicomlib.SkipPixelData())
	if err != nil {
		log.Printf("dicomweb metadata parse %s: %v", key, err)
		http.Error(w, "failed to parse DICOM file", http.StatusUnprocessableEntity)
		return
	}

	obj := datasetToDICOMwebJSON(dataset)
	w.Header().Set("Content-Type", "application/dicom+json")
	json.NewEncoder(w).Encode([]map[string]any{obj})
}

// datasetToDICOMwebJSON converts a parsed DICOM dataset to a DICOMweb JSON object.
// Pixel data (7FE0,0010) is excluded. Binary VRs use BulkDataURI placeholders.
func datasetToDICOMwebJSON(dataset dicomlib.Dataset) map[string]any {
	obj := make(map[string]any, len(dataset.Elements))
	for _, el := range dataset.Elements {
		if el.Tag == tag.PixelData {
			continue
		}
		tagKey := fmt.Sprintf("%08X", uint32(el.Tag.Group)<<16|uint32(el.Tag.Element))
		vr := el.RawValueRepresentation

		switch vr {
		case "OB", "OD", "OF", "OL", "OV", "OW", "UN":
			// Binary VRs — omit inline; OHIF won't need them for metadata
			obj[tagKey] = map[string]any{"vr": vr}

		case "SQ":
			// Sequence — emit as empty array for simplicity
			obj[tagKey] = map[string]any{"vr": vr, "Value": []any{}}

		case "PN":
			values := []any{}
			if el.Value != nil {
				for _, v := range toStringSlice(el) {
					values = append(values, map[string]any{"Alphabetic": v})
				}
			}
			if len(values) == 0 {
				obj[tagKey] = map[string]any{"vr": vr}
			} else {
				obj[tagKey] = map[string]any{"vr": vr, "Value": values}
			}

		case "FL", "FD":
			values := toFloatSlice(el)
			if len(values) == 0 {
				obj[tagKey] = map[string]any{"vr": vr}
			} else {
				obj[tagKey] = map[string]any{"vr": vr, "Value": values}
			}

		case "SL", "SS", "UL", "US", "AT":
			values := toIntSlice(el)
			if len(values) == 0 {
				obj[tagKey] = map[string]any{"vr": vr}
			} else {
				obj[tagKey] = map[string]any{"vr": vr, "Value": values}
			}

		default:
			// All string VRs (AE, AS, CS, DA, DS, DT, IS, LO, LT, SH, ST, TM, UC, UI, UR, UT)
			values := toStringSlice(el)
			if len(values) == 0 {
				obj[tagKey] = map[string]any{"vr": vr}
			} else {
				anyValues := make([]any, len(values))
				for i, v := range values {
					anyValues[i] = v
				}
				obj[tagKey] = map[string]any{"vr": vr, "Value": anyValues}
			}
		}
	}
	return obj
}

func toStringSlice(el *dicomlib.Element) []string {
	if el.Value == nil {
		return nil
	}
	v := el.Value.GetValue()
	switch val := v.(type) {
	case []string:
		return val
	case []int:
		out := make([]string, len(val))
		for i, n := range val {
			out[i] = fmt.Sprintf("%d", n)
		}
		return out
	case []float64:
		out := make([]string, len(val))
		for i, f := range val {
			out[i] = fmt.Sprintf("%g", f)
		}
		return out
	default:
		s := fmt.Sprintf("%v", val)
		if s == "" || s == "[]" {
			return nil
		}
		return []string{s}
	}
}

func toIntSlice(el *dicomlib.Element) []any {
	if el.Value == nil {
		return nil
	}
	v := el.Value.GetValue()
	switch val := v.(type) {
	case []int:
		out := make([]any, len(val))
		for i, n := range val {
			out[i] = n
		}
		return out
	default:
		return nil
	}
}

func toFloatSlice(el *dicomlib.Element) []any {
	if el.Value == nil {
		return nil
	}
	v := el.Value.GetValue()
	switch val := v.(type) {
	case []float64:
		out := make([]any, len(val))
		for i, f := range val {
			out[i] = f
		}
		return out
	default:
		return nil
	}
}
