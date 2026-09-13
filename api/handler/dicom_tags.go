package handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// DicomTagEntry represents one DICOM tag in the inspection response.
type DicomTagEntry struct {
	Tag     string `json:"tag"`     // e.g. "(0008,0060)"
	Keyword string `json:"keyword"` // e.g. "Modality"
	VR      string `json:"vr"`      // e.g. "CS"
	Value   string `json:"value"`   // string representation of the element value
}

// InspectDicomTags reads tags from the first DICOM file of a study (skipping pixel data)
// and returns them as a structured list.
// GET /api/studies/{studyUID}/dicom-tags
func (s *Server) InspectDicomTags(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")

	study, _, ok := s.requireStudyReadAccessByUID(w, r, studyUID)
	if !ok {
		return
	}

	// List files in the current DICOM store for this study.
	prefix := fmt.Sprintf("dicom/%s/%s", study.DicomStore, study.StudyInstanceUID)
	keys, err := s.store.List(r.Context(), prefix)
	if err != nil || len(keys) == 0 {
		s.writeError(w, http.StatusNotFound, "no DICOM files found for this study")
		return
	}

	// Use only the first file.
	rc, err := s.store.Retrieve(r.Context(), keys[0])
	if err != nil {
		log.Printf("dicom_tags: retrieve %s: %v", keys[0], err)
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve DICOM file")
		return
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read DICOM file")
		return
	}

	dataset, err := dicomlib.Parse(bytes.NewReader(data), int64(len(data)), nil, dicomlib.SkipPixelData())
	if err != nil {
		log.Printf("dicom_tags: parse %s: %v", keys[0], err)
		s.writeError(w, http.StatusUnprocessableEntity, "failed to parse DICOM file")
		return
	}

	var tags []DicomTagEntry
	for _, el := range dataset.Elements {
		if el.Tag == tag.PixelData {
			continue // always skip pixel data even if somehow present
		}
		kwStr := ""
		if info, err := tag.Find(el.Tag); err == nil {
			kwStr = info.Keyword
		}
		tags = append(tags, DicomTagEntry{
			Tag:     fmt.Sprintf("(%04X,%04X)", el.Tag.Group, el.Tag.Element),
			Keyword: kwStr,
			VR:      el.RawValueRepresentation,
			Value:   dicomValueString(el),
		})
	}
	if tags == nil {
		tags = []DicomTagEntry{}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"study_uid": study.StudyInstanceUID,
		"file":      keys[0],
		"tag_count": len(tags),
		"tags":      tags,
	})
}

// dicomValueString returns a concise string representation of a DICOM element's value.
func dicomValueString(el *dicomlib.Element) string {
	if el.Value == nil {
		return ""
	}
	v := el.Value.GetValue()
	switch val := v.(type) {
	case []string:
		return strings.Join(val, "\\")
	case []int:
		parts := make([]string, len(val))
		for i, n := range val {
			parts[i] = fmt.Sprintf("%d", n)
		}
		return strings.Join(parts, "\\")
	case []float64:
		parts := make([]string, len(val))
		for i, f := range val {
			parts[i] = fmt.Sprintf("%g", f)
		}
		return strings.Join(parts, "\\")
	default:
		return fmt.Sprintf("%v", val)
	}
}
