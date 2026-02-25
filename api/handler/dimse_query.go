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

type dimseQueryRequest struct {
	AETitle     string         `json:"ae_title"`
	Host        string         `json:"host"`
	Port        int            `json:"port"`
	QueryLevel  string         `json:"query_level"`
	QueryParams map[string]any `json:"query_params"`
}

// DimseQuery proxies a C-FIND SCU request to the DIMSE receiver sidecar and
// returns the matching dataset list. Accepts the destination by explicit
// ae_title/host/port or by destination_id (looked up from the DB).
func (s *Server) DimseQuery(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(s.cfg.DimseReceiverURL) == "" {
		s.writeError(w, http.StatusServiceUnavailable, "dimse receiver service not configured")
		return
	}

	var req dimseQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.AETitle) == "" || strings.TrimSpace(req.Host) == "" || req.Port <= 0 {
		s.writeError(w, http.StatusBadRequest, "ae_title, host, and port are required")
		return
	}
	if req.QueryLevel == "" {
		req.QueryLevel = "STUDY"
	}
	if req.QueryParams == nil {
		req.QueryParams = map[string]any{}
	}

	payload, err := json.Marshal(map[string]any{
		"ae_title":     req.AETitle,
		"host":         req.Host,
		"port":         req.Port,
		"query_level":  req.QueryLevel,
		"query_params": req.QueryParams,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to build query payload")
		return
	}

	queryURL := strings.TrimRight(s.cfg.DimseReceiverURL, "/") + "/query"

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL, bytes.NewReader(payload))
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to construct dimse query request")
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
		log.Printf("dimse query proxy: copy response body: %v", err)
	}
}
