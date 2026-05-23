package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/events"
	"github.com/aegis-imaging/aegis/api/spoke_ca"
	"github.com/aegis-imaging/aegis/api/storage"
)


// sidecarHealthCache stores the last known health state of each sidecar service.
// It is populated by a background goroutine so that /healthz never blocks on
// cold-starting Cloud Run services.
type sidecarHealthCache struct {
	mu      sync.RWMutex
	results map[string]string
}

func (c *sidecarHealthCache) set(name, status string) {
	c.mu.Lock()
	c.results[name] = status
	c.mu.Unlock()
}

func (c *sidecarHealthCache) snapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.results))
	for k, v := range c.results {
		out[k] = v
	}
	return out
}

type Server struct {
	db            *sql.DB
	store         storage.Storage
	cfg           *config.Config
	mailer        *email.Client
	httpClient    *http.Client
	sidecarHealth *sidecarHealthCache
	bus           *events.Bus
	spokeCA       *spoke_ca.Signer // nil = enrollment disabled
}

// SetSpokeCA wires the spoke-router cert issuer into the server. Called by
// main.go at startup; nil means /api/spokes/enroll returns 503.
func (s *Server) SetSpokeCA(signer *spoke_ca.Signer) {
	s.spokeCA = signer
}

// SpokeCA exposes the configured signer (or nil) for handlers and tests.
func (s *Server) SpokeCA() *spoke_ca.Signer { return s.spokeCA }

func NewServer(db *sql.DB, store storage.Storage, cfg *config.Config) *Server {
	s := &Server{
		db:    db,
		store: store,
		cfg:   cfg,
		mailer: email.New(cfg),
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		sidecarHealth: &sidecarHealthCache{results: map[string]string{}},
		bus:           events.NewBus(),
	}
	go s.runSidecarHealthLoop()
	return s
}

// runSidecarHealthLoop probes sidecar /healthz endpoints in the background every
// 30 seconds. Results are cached and returned from Healthz without blocking.
// A 15-second probe timeout accommodates Cloud Run cold-start latency.
func (s *Server) runSidecarHealthLoop() {
	sidecars := func() map[string]string {
		return map[string]string{
			"defacing":       s.cfg.DefacingServiceURL,
			"phi_detection":  s.cfg.PhiDetectionServiceURL,
			"qc_service":     s.cfg.QcServiceURL,
			"bids_service":   s.cfg.BidsServiceURL,
			"classification": s.cfg.ClassificationServiceURL,
			"protocol":       s.cfg.ProtocolServiceURL,
			"dimse_receiver": s.cfg.DimseReceiverURL,
		}
	}
	probe := func() {
		m := sidecars()
		var wg sync.WaitGroup
		for name, url := range m {
			if url == "" {
				s.sidecarHealth.set(name, "disabled")
				continue
			}
			wg.Add(1)
			go func(name, url string) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health", nil)
				status := "unhealthy"
				if err == nil {
					if resp, err := s.httpClient.Do(req); err == nil {
						resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							status = "healthy"
						}
					}
				}
				s.sidecarHealth.set(name, status)
			}(name, url)
		}
		wg.Wait()
	}

	probe() // initial probe on startup
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		probe()
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{"status": "ok"}

	// Database
	if err := s.db.PingContext(r.Context()); err != nil {
		result["status"] = "degraded"
		result["database"] = "unhealthy"
	} else {
		result["database"] = "healthy"
	}

	// Storage backend
	if _, err := s.store.List(r.Context(), "dicom/"); err != nil {
		result["status"] = "degraded"
		result["storage"] = "unhealthy"
	} else {
		result["storage"] = "healthy"
	}

	// Sidecar services — returned from background cache (no inline blocking).
	// Cache is refreshed every 30s by runSidecarHealthLoop.
	result["services"] = s.sidecarHealth.snapshot()

	httpStatus := http.StatusOK
	if result["status"] == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}
	s.writeJSON(w, httpStatus, result)
}
