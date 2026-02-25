package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

// probeResult holds the outcome of a single destination connectivity probe.
type probeResult struct {
	Success    bool
	LatencyMs  float64
	StatusCode *int
	Error      string
}

// probeDestination tests connectivity to a single DICOM destination and returns the result.
// It does NOT write to the audit trail — callers are responsible for that.
func (s *Server) probeDestination(ctx context.Context, dest model.Destination) probeResult {
	switch dest.Type {
	case "dicomweb":
		testURL := strings.TrimRight(dest.DicomwebURL, "/") + "/studies?limit=1"
		pCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(pCtx, "GET", testURL, nil)
		if err != nil {
			return probeResult{Error: fmt.Sprintf("failed to build request: %v", err)}
		}
		if dest.DicomwebAuthHeader != "" {
			req.Header.Set("Authorization", dest.DicomwebAuthHeader)
		}
		req.Header.Set("Accept", "application/dicom+json")

		t0 := time.Now()
		resp, err := s.httpClient.Do(req)
		latency := float64(time.Since(t0).Milliseconds())

		if err != nil {
			return probeResult{LatencyMs: latency, Error: fmt.Sprintf("request failed: %v", err)}
		}
		resp.Body.Close()
		code := resp.StatusCode
		success := resp.StatusCode < 400 || resp.StatusCode == 405
		res := probeResult{Success: success, LatencyMs: latency, StatusCode: &code}
		if !success {
			res.Error = fmt.Sprintf("unexpected status %d", resp.StatusCode)
		}
		return res

	case "dimse":
		if strings.TrimSpace(s.cfg.DimseReceiverURL) == "" {
			return probeResult{Error: "dimse receiver service not configured"}
		}
		echoURL := strings.TrimRight(s.cfg.DimseReceiverURL, "/") + "/echo"
		payload, _ := json.Marshal(map[string]any{
			"ae_title": dest.AETitle,
			"host":     dest.Host,
			"port":     dest.Port,
		})

		pCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(pCtx, "POST", echoURL, bytes.NewReader(payload))
		if err != nil {
			return probeResult{Error: fmt.Sprintf("failed to build echo request: %v", err)}
		}
		req.Header.Set("Content-Type", "application/json")

		t0 := time.Now()
		resp, err := s.httpClient.Do(req)
		latency := float64(time.Since(t0).Milliseconds())
		if err != nil {
			return probeResult{LatencyMs: latency, Error: fmt.Sprintf("dimse echo request failed: %v", err)}
		}
		defer resp.Body.Close()

		var echoResp struct {
			Success   bool    `json:"success"`
			LatencyMs float64 `json:"latency_ms"`
			Detail    string  `json:"detail"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&echoResp); err == nil && resp.StatusCode == 200 {
			res := probeResult{Success: echoResp.Success, LatencyMs: echoResp.LatencyMs}
			if !echoResp.Success {
				res.Error = echoResp.Detail
			}
			return res
		}
		return probeResult{LatencyMs: latency, Error: fmt.Sprintf("dimse echo returned status %d", resp.StatusCode)}

	default:
		return probeResult{Error: fmt.Sprintf("unknown destination type: %s", dest.Type)}
	}
}

// writeProbeAudit stores a destination.tested audit entry.
// auto=true marks entries written by the scheduler (vs manual "Test" button clicks).
func writeProbeAudit(ctx context.Context, db *sql.DB, destID, destType string, res probeResult, auto bool) {
	meta := map[string]any{"type": destType, "success": res.Success, "latency_ms": res.LatencyMs, "auto": auto}
	if res.Error != "" {
		meta["error"] = res.Error
	}
	if res.StatusCode != nil {
		meta["status_code"] = *res.StatusCode
	}
	model.CreateAuditEntry(ctx, db, "destination.tested", "scheduler", "destination", destID, "", meta)
}

// StartDestinationProbeScheduler starts a background goroutine that probes all enabled
// DICOM destinations every interval seconds.  Stops when ctx is cancelled.
// Sends an email alert to alertEmail when a destination transitions from passing to failing
// (requires SMTP_HOST to be configured).  No-op when interval ≤ 0.
func (s *Server) StartDestinationProbeScheduler(ctx context.Context, interval time.Duration, alertEmail string) {
	if interval <= 0 {
		return
	}
	log.Printf("destination probe scheduler: starting, interval=%s alert=%q", interval, alertEmail)
	go func() {
		// Run immediately on startup, then on each tick.
		s.runDestinationProbes(ctx, alertEmail)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runDestinationProbes(ctx, alertEmail)
			}
		}
	}()
}

// runDestinationProbes probes all enabled destinations once and optionally sends alerts.
func (s *Server) runDestinationProbes(ctx context.Context, alertEmail string) {
	dests, err := model.ListDestinations(ctx, s.db)
	if err != nil {
		log.Printf("destination probe: list destinations: %v", err)
		return
	}
	for _, dest := range dests {
		if !dest.Enabled {
			continue
		}
		// Capture previous state before probing.
		prevStatus := s.destPreviousStatus(ctx, dest.ID)

		res := s.probeDestination(ctx, dest)
		writeProbeAudit(ctx, s.db, dest.ID, dest.Type, res, true)

		// Alert on failure transition: was OK (or never tested) → now failing.
		if !res.Success && prevStatus != "failing" && alertEmail != "" {
			s.notifyDestinationFailing(ctx, dest, res.Error, alertEmail)
		}
	}
}

// destPreviousStatus returns the status derived from the MOST RECENT existing probe entry
// for this destination: "healthy", "failing", or "unknown" (never tested).
func (s *Server) destPreviousStatus(ctx context.Context, destID string) string {
	var successRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT detail->'success'
		FROM audit_trail
		WHERE action = 'destination.tested'
		  AND resource_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, destID).Scan(&successRaw)
	if err == sql.ErrNoRows {
		return "unknown"
	}
	if err != nil {
		return "unknown"
	}
	if string(successRaw) == "true" {
		return "healthy"
	}
	return "failing"
}

// notifyDestinationFailing sends a plain-text alert when a destination transitions to failing.
// No-op when SMTP is not configured (mailer is a no-op client).
func (s *Server) notifyDestinationFailing(ctx context.Context, dest model.Destination, errMsg, alertEmail string) {
	subj, body := email.DestinationFailing(dest.Name, dest.Type, errMsg)
	if err := s.mailer.Send(ctx, alertEmail, subj, body); err != nil {
		log.Printf("destination probe: send alert for %s: %v", dest.Name, err)
	}
}
