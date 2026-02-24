package handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/model"
)

type anonDiffTag struct {
	Tag        string `json:"tag"`
	Keyword    string `json:"keyword"`
	VR         string `json:"vr"`
	RawValue   string `json:"raw_value,omitempty"`
	CleanValue string `json:"clean_value,omitempty"`
}

type anonDiffResult struct {
	Removed  []anonDiffTag `json:"removed"`
	Modified []anonDiffTag `json:"modified"`
	Added    []anonDiffTag `json:"added"`
}

type anonDiffResponse struct {
	StudyID   string         `json:"study_id"`
	DicomStore string        `json:"dicom_store"`
	Diff      anonDiffResult `json:"diff"`
	RawFile   string         `json:"raw_file"`
	CleanFile string         `json:"clean_file"`
}

// GetAnonDiff computes the tag-level diff between the raw and clean DICOM
// stores for a study, showing exactly which tags were removed, modified, or
// added during de-identification.
//
// GET /api/studies/{studyUID}/anonymization-diff
//
// Returns 404 when the raw store has no files (raw store already cleaned up).
// Returns 204 when the clean store has no files yet (defacing not yet done).
func (s *Server) GetAnonDiff(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")

	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	rawPrefix := fmt.Sprintf("dicom/raw/%s", study.StudyInstanceUID)
	rawKeys, err := s.store.List(r.Context(), rawPrefix)
	if err != nil || len(rawKeys) == 0 {
		s.writeError(w, http.StatusNotFound, "raw DICOM store not available (may have been purged)")
		return
	}

	cleanPrefix := fmt.Sprintf("dicom/clean/%s", study.StudyInstanceUID)
	cleanKeys, err := s.store.List(r.Context(), cleanPrefix)
	if err != nil || len(cleanKeys) == 0 {
		// Clean store does not exist yet — defacing has not completed.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	rawTags, err := readDicomTags(s, r, rawKeys[0])
	if err != nil {
		log.Printf("anon_diff: read raw tags %s: %v", rawKeys[0], err)
		s.writeError(w, http.StatusInternalServerError, "failed to read raw DICOM file")
		return
	}

	cleanTags, err := readDicomTags(s, r, cleanKeys[0])
	if err != nil {
		log.Printf("anon_diff: read clean tags %s: %v", cleanKeys[0], err)
		s.writeError(w, http.StatusInternalServerError, "failed to read clean DICOM file")
		return
	}

	diff := diffTagMaps(rawTags, cleanTags)

	s.writeJSON(w, http.StatusOK, anonDiffResponse{
		StudyID:    study.ID,
		DicomStore: study.DicomStore,
		Diff:       diff,
		RawFile:    rawKeys[0],
		CleanFile:  cleanKeys[0],
	})
}

// readDicomTags retrieves and parses a DICOM file, returning a map from tag
// string to (keyword, vr, value) — pixel data excluded.
func readDicomTags(s *Server, r *http.Request, key string) (map[string]dicomTagInfo, error) {
	rc, err := s.store.Retrieve(r.Context(), key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	dataset, err := dicomlib.Parse(bytes.NewReader(data), int64(len(data)), nil, dicomlib.SkipPixelData())
	if err != nil {
		return nil, err
	}

	out := make(map[string]dicomTagInfo, len(dataset.Elements))
	for _, el := range dataset.Elements {
		if el.Tag == tag.PixelData {
			continue
		}
		tagStr := fmt.Sprintf("(%04X,%04X)", el.Tag.Group, el.Tag.Element)
		keyword := ""
		if info, err := tag.Find(el.Tag); err == nil {
			keyword = info.Keyword
		}
		out[tagStr] = dicomTagInfo{
			keyword: keyword,
			vr:      el.RawValueRepresentation,
			value:   dicomValueString(el),
		}
	}
	return out, nil
}

type dicomTagInfo struct {
	keyword string
	vr      string
	value   string
}

// diffTagMaps computes removed, modified, and added entries between raw and clean.
func diffTagMaps(raw, clean map[string]dicomTagInfo) anonDiffResult {
	res := anonDiffResult{
		Removed:  []anonDiffTag{},
		Modified: []anonDiffTag{},
		Added:    []anonDiffTag{},
	}

	for tagStr, rawInfo := range raw {
		cleanInfo, exists := clean[tagStr]
		if !exists {
			res.Removed = append(res.Removed, anonDiffTag{
				Tag:      tagStr,
				Keyword:  rawInfo.keyword,
				VR:       rawInfo.vr,
				RawValue: rawInfo.value,
			})
		} else if rawInfo.value != cleanInfo.value {
			res.Modified = append(res.Modified, anonDiffTag{
				Tag:        tagStr,
				Keyword:    rawInfo.keyword,
				VR:         rawInfo.vr,
				RawValue:   rawInfo.value,
				CleanValue: cleanInfo.value,
			})
		}
	}

	for tagStr, cleanInfo := range clean {
		if _, exists := raw[tagStr]; !exists {
			res.Added = append(res.Added, anonDiffTag{
				Tag:        tagStr,
				Keyword:    cleanInfo.keyword,
				VR:         cleanInfo.vr,
				CleanValue: cleanInfo.value,
			})
		}
	}

	return res
}
