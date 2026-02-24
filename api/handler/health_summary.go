package handler

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// healthSummaryCache holds the cached health summary response.
type healthSummaryCache struct {
	mu        sync.RWMutex
	data      map[string]any
	expiresAt time.Time
}

var globalHealthSummaryCache = &healthSummaryCache{}

// systemHealthSummary is the structured response payload.
type systemHealthSummary struct {
	GeneratedAt string            `json:"generated_at"`
	API         apiHealth         `json:"api"`
	Services    map[string]any    `json:"services"`
	Pipeline    pipelineHealth    `json:"pipeline"`
	DIMSE       dimseHealth       `json:"dimse"`
}

type apiHealth struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Storage  string `json:"storage"`
}

type pipelineHealth struct {
	StudiesReceived24h int     `json:"studies_received_24h"`
	StudiesApproved24h int     `json:"studies_approved_24h"`
	StudiesStuck       int     `json:"studies_stuck"`
	ErrorRate          float64 `json:"pipeline_error_rate"`
}

type dimseHealth struct {
	PendingRetries int `json:"pending_retries"`
	DeadLetter     int `json:"dead_letter"`
}

// GetSystemHealthSummary aggregates health info across all services.
// Results are cached for 30 seconds to prevent thundering herd.
//
// GET /api/system/health-summary
func (s *Server) GetSystemHealthSummary(w http.ResponseWriter, r *http.Request) {
	// Check cache.
	globalHealthSummaryCache.mu.RLock()
	if time.Now().Before(globalHealthSummaryCache.expiresAt) && globalHealthSummaryCache.data != nil {
		cached := globalHealthSummaryCache.data
		globalHealthSummaryCache.mu.RUnlock()
		s.writeJSON(w, http.StatusOK, cached)
		return
	}
	globalHealthSummaryCache.mu.RUnlock()

	// Build fresh summary.
	summary := s.buildHealthSummary(r.Context())

	// Serialize to map for caching.
	raw, _ := json.Marshal(summary)
	var m map[string]any
	json.Unmarshal(raw, &m) //nolint:errcheck

	globalHealthSummaryCache.mu.Lock()
	globalHealthSummaryCache.data = m
	globalHealthSummaryCache.expiresAt = time.Now().Add(30 * time.Second)
	globalHealthSummaryCache.mu.Unlock()

	s.writeJSON(w, http.StatusOK, m)
}

func (s *Server) buildHealthSummary(ctx context.Context) systemHealthSummary {
	summary := systemHealthSummary{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// ── API health ─────────────────────────────────────────────────────────────
	summary.API.Status = "ok"

	if err := s.db.PingContext(ctx); err != nil {
		summary.API.Status = "degraded"
		summary.API.Database = "unhealthy"
	} else {
		summary.API.Database = "healthy"
	}

	if _, err := s.store.List(ctx, "dicom/"); err != nil {
		summary.API.Status = "degraded"
		summary.API.Storage = "unhealthy"
	} else {
		summary.API.Storage = "healthy"
	}

	// ── Sidecar services (from background cache) ──────────────────────────────
	snap := s.sidecarHealth.snapshot()
	services := make(map[string]any, len(snap))
	for name, status := range snap {
		svcURL := ""
		switch name {
		case "defacing":
			svcURL = s.cfg.DefacingServiceURL
		case "phi_detection":
			svcURL = s.cfg.PhiDetectionServiceURL
		case "qc_service":
			svcURL = s.cfg.QcServiceURL
		case "bids_service":
			svcURL = s.cfg.BidsServiceURL
		case "classification":
			svcURL = s.cfg.ClassificationServiceURL
		case "protocol":
			svcURL = s.cfg.ProtocolServiceURL
		case "dimse_receiver":
			svcURL = s.cfg.DimseReceiverURL
		}
		entry := map[string]string{"status": status}
		if svcURL != "" {
			entry["url"] = svcURL
		}
		services[name] = entry
	}
	summary.Services = services

	// ── Pipeline: 24h stats ───────────────────────────────────────────────────
	since := time.Now().UTC().Add(-24 * time.Hour)
	row := s.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE created_at >= $1)                           AS received_24h,
			COUNT(*) FILTER (WHERE status='approved' AND updated_at >= $1)     AS approved_24h
		 FROM studies`, since)
	var received24h, approved24h int
	if err := row.Scan(&received24h, &approved24h); err != nil {
		log.Printf("health_summary: pipeline 24h query: %v", err)
	}
	summary.Pipeline.StudiesReceived24h = received24h
	summary.Pipeline.StudiesApproved24h = approved24h

	// Stuck studies (idle > 60 min, any project).
	stuck, err := s.db.QueryContext(ctx,
		`SELECT COUNT(*) FROM studies
		 WHERE status NOT IN ('approved','rejected','expired')
		   AND updated_at < NOW() - INTERVAL '60 minutes'`)
	if err == nil {
		defer stuck.Close()
		if stuck.Next() {
			stuck.Scan(&summary.Pipeline.StudiesStuck) //nolint:errcheck
		}
	}

	// Error rate: studies in any failed/rejected pipeline state in last 24h.
	if received24h > 0 {
		var failed24h int
		r2 := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM studies
			 WHERE status = 'rejected' AND updated_at >= $1`, since)
		if err := r2.Scan(&failed24h); err == nil && received24h > 0 {
			summary.Pipeline.ErrorRate = float64(failed24h) / float64(received24h)
		}
	}

	// ── DIMSE retry/dead-letter ───────────────────────────────────────────────
	if s.cfg.DimseReceiverURL != "" {
		dimseURL := s.cfg.DimseReceiverURL + "/ingest/retry"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, dimseURL, nil)
		if err == nil {
			if resp, err := s.httpClient.Do(req); err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				var dimseData map[string]any
				if json.Unmarshal(body, &dimseData) == nil {
					if v, ok := dimseData["pending"].(float64); ok {
						summary.DIMSE.PendingRetries = int(v)
					}
					if v, ok := dimseData["dead_letter"].(float64); ok {
						summary.DIMSE.DeadLetter = int(v)
					}
				}
			}
		}
	}

	return summary
}
