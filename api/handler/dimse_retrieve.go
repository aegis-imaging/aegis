package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type dimseRetrieveRequest struct {
	AETitle          string `json:"ae_title"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	StudyInstanceUID string `json:"study_instance_uid"`
	MoveDestination  string `json:"move_destination"` // optional; defaults to sidecar's DIMSE_AE_TITLE
}

// DimseRetrieve proxies a C-MOVE SCU request to the DIMSE receiver sidecar.
// The remote PACS pushes DICOM files back to the AEGIS SCP (port 11112), which
// ingests them via the normal EVT_RELEASED → POST /api/ingest path.
func (s *Server) DimseRetrieve(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(s.cfg.DimseReceiverURL) == "" {
		s.writeError(w, http.StatusServiceUnavailable, "dimse receiver service not configured")
		return
	}

	var req dimseRetrieveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.AETitle) == "" || strings.TrimSpace(req.Host) == "" || req.Port <= 0 {
		s.writeError(w, http.StatusBadRequest, "ae_title, host, and port are required")
		return
	}
	if strings.TrimSpace(req.StudyInstanceUID) == "" {
		s.writeError(w, http.StatusBadRequest, "study_instance_uid is required")
		return
	}

	payload, err := json.Marshal(map[string]any{
		"ae_title":           req.AETitle,
		"host":               req.Host,
		"port":               req.Port,
		"study_instance_uid": req.StudyInstanceUID,
		"move_destination":   req.MoveDestination,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to build retrieve payload")
		return
	}

	retrieveURL := strings.TrimRight(s.cfg.DimseReceiverURL, "/") + "/retrieve"

	// C-MOVE can take a while — use a generous timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, retrieveURL, bytes.NewReader(payload))
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to construct dimse retrieve request")
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(proxyReq)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to reach dimse receiver")
		return
	}
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("dimse retrieve proxy: copy response body: %v", err)
	}
}
