package handler

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// DimseRetryProxy proxies DIMSE retry-control requests from the API to the
// DIMSE receiver sidecar. Admin auth is enforced by API middleware.
func (s *Server) DimseRetryProxy(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(s.cfg.DimseReceiverURL) == "" {
		s.writeError(w, http.StatusServiceUnavailable, "dimse receiver service not configured")
		return
	}

	targetPath := "/ingest/retry"
	if suffix := strings.Trim(r.PathValue("path"), "/"); suffix != "" {
		targetPath += "/" + suffix
	}
	targetURL := strings.TrimRight(s.cfg.DimseReceiverURL, "/") + targetPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to construct dimse retry request")
		return
	}
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	if key := strings.TrimSpace(s.cfg.DimseOperatorAPIKey); key != "" {
		req.Header.Set("X-AEGIS-Operator-Key", key)
	}

	resp, err := s.httpClient.Do(req)
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
		log.Printf("dimse retry proxy: copy response body: %v", err)
	}
}
