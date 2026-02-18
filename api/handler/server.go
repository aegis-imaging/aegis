package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/email"
	"github.com/msenjem/aegis/api/storage"
)

type Server struct {
	db     *sql.DB
	store  storage.Storage
	cfg    *config.Config
	mailer *email.Client
}

func NewServer(db *sql.DB, store storage.Storage, cfg *config.Config) *Server {
	return &Server{db: db, store: store, cfg: cfg, mailer: email.New(cfg)}
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
	if err := s.db.PingContext(r.Context()); err != nil {
		s.writeError(w, http.StatusServiceUnavailable, "database unhealthy")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
