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

type dimseRetrieveBatchRequest struct {
	AETitle             string   `json:"ae_title"`
	Host                string   `json:"host"`
	Port                int      `json:"port"`
	StudyInstanceUIDs   []string `json:"study_instance_uids"`
	MoveDestination     string   `json:"move_destination"` // optional; defaults to sidecar's DIMSE_AE_TITLE
}

type dimseRetrieveBatchResult struct {
	StudyInstanceUID string `json:"study_instance_uid"`
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
}

// DimseRetrieveBatch issues a C-MOVE for each study UID in the list, proxying
// each request to the DIMSE receiver sidecar's /retrieve endpoint sequentially.
// Results are collected and returned as a summary even when individual studies fail.
func (s *Server) DimseRetrieveBatch(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(s.cfg.DimseReceiverURL) == "" {
		s.writeError(w, http.StatusServiceUnavailable, "dimse receiver service not configured")
		return
	}

	var req dimseRetrieveBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.AETitle) == "" || strings.TrimSpace(req.Host) == "" || req.Port <= 0 {
		s.writeError(w, http.StatusBadRequest, "ae_title, host, and port are required")
		return
	}
	if len(req.StudyInstanceUIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_instance_uids must contain at least one UID")
		return
	}
	if len(req.StudyInstanceUIDs) > 50 {
		s.writeError(w, http.StatusBadRequest, "study_instance_uids must not exceed 50 entries per batch")
		return
	}

	// Deduplicate UIDs while preserving order.
	seen := make(map[string]bool, len(req.StudyInstanceUIDs))
	var uids []string
	for _, uid := range req.StudyInstanceUIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		uids = append(uids, uid)
	}
	if len(uids) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_instance_uids contains no valid entries")
		return
	}

	retrieveURL := strings.TrimRight(s.cfg.DimseReceiverURL, "/") + "/retrieve"

	var results []dimseRetrieveBatchResult
	succeeded := 0
	failed := 0

	for _, uid := range uids {
		payload, err := json.Marshal(map[string]any{
			"ae_title":           req.AETitle,
			"host":               req.Host,
			"port":               req.Port,
			"study_instance_uid": uid,
			"move_destination":   req.MoveDestination,
		})
		if err != nil {
			results = append(results, dimseRetrieveBatchResult{StudyInstanceUID: uid, Success: false, Error: "failed to build payload"})
			failed++
			continue
		}

		// Each C-MOVE can take several minutes for large studies.
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)

		proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, retrieveURL, bytes.NewReader(payload))
		if err != nil {
			cancel()
			results = append(results, dimseRetrieveBatchResult{StudyInstanceUID: uid, Success: false, Error: "failed to construct request"})
			failed++
			continue
		}
		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Accept", "application/json")

		resp, err := s.httpClient.Do(proxyReq)
		cancel()
		if err != nil {
			results = append(results, dimseRetrieveBatchResult{StudyInstanceUID: uid, Success: false, Error: "failed to reach dimse receiver"})
			failed++
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			results = append(results, dimseRetrieveBatchResult{StudyInstanceUID: uid, Success: true})
			succeeded++
		} else {
			errMsg := strings.TrimSpace(string(body))
			if errMsg == "" {
				errMsg = resp.Status
			}
			results = append(results, dimseRetrieveBatchResult{StudyInstanceUID: uid, Success: false, Error: errMsg})
			failed++
			log.Printf("dimse retrieve batch: uid=%s status=%d body=%s", uid, resp.StatusCode, errMsg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMultiStatus)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"total":     len(results),
		"succeeded": succeeded,
		"failed":    failed,
		"results":   results,
	}); err != nil {
		log.Printf("dimse retrieve batch: encode response: %v", err)
	}
}
