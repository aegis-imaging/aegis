package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/importer"
)

const tciaBaseURL = "https://services.cancerimagingarchive.net/nbia-api/services/v1"

// tciaSeriesRecord matches the NBIA getSeries JSON array element.
type tciaSeriesRecord struct {
	SeriesInstanceUID string `json:"SeriesInstanceUID"`
	Modality          string `json:"Modality"`
	BodyPartExamined  string `json:"BodyPartExamined"`
	SeriesDescription string `json:"SeriesDescription"`
	ImageCount        int    `json:"ImageCount"` // TCIA returns this as an integer
	Collection        string `json:"Collection"`
}

// TCIASeriesItem is the sanitised series descriptor returned to the frontend.
// No patient identifiers are forwarded.
type TCIASeriesItem struct {
	SeriesUID   string `json:"series_uid"`
	Modality    string `json:"modality"`
	BodyPart    string `json:"body_part"`
	Description string `json:"description"`
	SliceCount  int    `json:"slice_count"`
	Collection  string `json:"collection"`
}

// GetTCIASeries proxies the TCIA NBIA getSeries endpoint.
// Returns volumetric MR series (>= min_slices, default 20), with PII stripped.
//
// GET /api/tcia/series?collection=TCGA-GBM&min_slices=20
func (s *Server) GetTCIASeries(w http.ResponseWriter, r *http.Request) {
	collection := strings.TrimSpace(r.URL.Query().Get("collection"))
	if collection == "" {
		s.writeError(w, http.StatusBadRequest, "collection is required")
		return
	}

	minSlices := 20
	if v := r.URL.Query().Get("min_slices"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minSlices = n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tciaBaseURL+"/getSeries", nil)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "build request failed")
		return
	}
	q := req.URL.Query()
	q.Set("Collection", collection)
	q.Set("Modality", "MR")
	q.Set("format", "json")
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "TCIA request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("TCIA returned %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "read TCIA response: "+err.Error())
		return
	}
	if len(body) == 0 {
		// TCIA returns an empty body for restricted or unavailable collections.
		s.writeError(w, http.StatusBadGateway, "TCIA returned no data for collection "+collection+" — the collection may be restricted or temporarily unavailable")
		return
	}

	var raw []tciaSeriesRecord
	if err := json.Unmarshal(body, &raw); err != nil {
		s.writeError(w, http.StatusBadGateway, "decode TCIA response: "+err.Error())
		return
	}

	items := make([]TCIASeriesItem, 0, len(raw))
	for _, rec := range raw {
		if rec.ImageCount < minSlices {
			continue
		}
		items = append(items, TCIASeriesItem{
			SeriesUID:   rec.SeriesInstanceUID,
			Modality:    rec.Modality,
			BodyPart:    rec.BodyPartExamined,
			Description: rec.SeriesDescription,
			SliceCount:  rec.ImageCount,
			Collection:  rec.Collection,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"series":     items,
		"total":      len(items),
		"collection": collection,
	})
}

// tciaImportRequest is the body for POST /api/tcia/import.
type tciaImportRequest struct {
	SeriesUID   string `json:"series_uid"`
	Collection  string `json:"collection"`
	ProjectSlug string `json:"project_slug"`
}

// ImportTCIASeries downloads a DICOM series from TCIA and imports it into AEGIS.
//
// POST /api/tcia/import
func (s *Server) ImportTCIASeries(w http.ResponseWriter, r *http.Request) {
	var req tciaImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.SeriesUID = strings.TrimSpace(req.SeriesUID)
	req.Collection = strings.TrimSpace(req.Collection)
	req.ProjectSlug = strings.ToLower(strings.TrimSpace(req.ProjectSlug))

	if req.SeriesUID == "" {
		s.writeError(w, http.StatusBadRequest, "series_uid is required")
		return
	}
	if req.ProjectSlug == "" {
		req.ProjectSlug = "default"
	}

	log.Printf("tcia-import: downloading series %s (collection: %s)", req.SeriesUID, req.Collection)

	// Download the TCIA ZIP — large series can take several minutes.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	zipBytes, err := downloadTCIASeries(ctx, s.httpClient, req.SeriesUID)
	if err != nil {
		log.Printf("tcia-import: download failed: %v", err)
		s.writeError(w, http.StatusBadGateway, "TCIA download failed: "+err.Error())
		return
	}
	log.Printf("tcia-import: downloaded %.1f MB for series %s", float64(len(zipBytes))/1024/1024, req.SeriesUID)

	// Extract to a temp directory; clean up on exit.
	tmpDir, err := os.MkdirTemp("", "aegis-tcia-*")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create temp dir: "+err.Error())
		return
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			log.Printf("tcia-import: cleanup %s: %v", tmpDir, rmErr)
		}
	}()

	extracted, err := extractTCIAZip(zipBytes, tmpDir)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "extract zip: "+err.Error())
		return
	}
	if extracted == 0 {
		s.writeError(w, http.StatusBadGateway, "no DICOM files found in TCIA archive")
		return
	}
	log.Printf("tcia-import: extracted %d .dcm files to %s", extracted, tmpDir)

	// Import using the standard batch importer (same path as POST /api/import/batch).
	result, err := importer.Run(r.Context(), s.db, s.store, importer.Options{
		Dir:         tmpDir,
		ProjectSlug: req.ProjectSlug,
		Source:      "internal",
	})
	if err != nil {
		if importer.IsValidationError(err) {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeError(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}

	// Auto-advance processing pipeline for each imported study.
	for _, studyID := range result.StudyIDs {
		s.AdvancePipeline(r.Context(), studyID)
	}

	model := map[string]any{
		"files_scanned":   result.FilesScanned,
		"files_skipped":   result.FilesSkipped,
		"studies_created": result.StudiesCreated,
		"studies_failed":  result.StudiesFailed,
		"study_ids":       result.StudyIDs,
	}
	if len(result.Errors) > 0 {
		model["errors"] = result.Errors
	}
	s.writeJSON(w, http.StatusOK, model)
}

// downloadTCIASeries fetches a DICOM series ZIP from the TCIA NBIA getImage endpoint.
func downloadTCIASeries(ctx context.Context, client *http.Client, seriesUID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tciaBaseURL+"/getImage", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("SeriesInstanceUID", seriesUID)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TCIA returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// extractTCIAZip extracts .dcm files from a TCIA ZIP archive into destDir (flat layout).
// TCIA ZIPs sometimes nest files under a series subdirectory; they are flattened here.
// Returns the count of extracted DICOM files.
func extractTCIAZip(zipBytes []byte, destDir string) (int, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}

	count := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)
		if name == "" || name == "." {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".dcm") {
			continue
		}
		if err := extractZipEntry(f, filepath.Join(destDir, name)); err != nil {
			return count, fmt.Errorf("extract %s: %w", name, err)
		}
		count++
	}
	return count, nil
}

func extractZipEntry(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
