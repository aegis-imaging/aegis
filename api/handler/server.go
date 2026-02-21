package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/email"
	"github.com/msenjem/aegis/api/storage"
)

type Server struct {
	db         *sql.DB
	store      storage.Storage
	cfg        *config.Config
	mailer     *email.Client
	httpClient *http.Client
}

func NewServer(db *sql.DB, store storage.Storage, cfg *config.Config) *Server {
	return &Server{
		db:     db,
		store:  store,
		cfg:    cfg,
		mailer: email.New(cfg),
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
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

	// Sidecar services (informational — degraded sidecars don't fail the check)
	sidecars := map[string]string{
		"defacing":       s.cfg.DefacingServiceURL,
		"phi_detection":  s.cfg.PhiDetectionServiceURL,
		"qc_service":     s.cfg.QcServiceURL,
		"bids_service":   s.cfg.BidsServiceURL,
		"classification": s.cfg.ClassificationServiceURL,
		"protocol":       s.cfg.ProtocolServiceURL,
		"dimse_receiver": s.cfg.DimseReceiverURL,
	}
	services := map[string]string{}
	for name, url := range sidecars {
		if url == "" {
			services[name] = "disabled"
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/healthz", nil)
		if err != nil {
			cancel()
			services[name] = "unhealthy"
			continue
		}
		resp, err := s.httpClient.Do(req)
		cancel()
		if err != nil {
			services[name] = "unhealthy"
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			services[name] = "healthy"
		} else {
			services[name] = "unhealthy"
		}
	}
	result["services"] = services

	httpStatus := http.StatusOK
	if result["status"] == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}
	s.writeJSON(w, httpStatus, result)
}
