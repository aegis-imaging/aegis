package testutil

import (
	"database/sql"
	"os"
	"testing"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/handler"
	"github.com/msenjem/aegis/api/storage"
)

// TestServer creates a handler.Server wired to a real test DB and temp local storage.
// PipelineAuto is disabled to prevent async goroutines from calling sidecar URLs.
func TestServer(t *testing.T, db *sql.DB) *handler.Server {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "aegis-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	cfg := &config.Config{
		Port:            "0",
		StorageMode:     "local",
		LocalStorageDir: tmpDir,
		APIBaseURL:      "http://localhost:8080",
		PipelineAuto:    false,
		AuthEnabled:     false,
		DevUserEmail:    "test@aegis.local",
		AllowedOrigins:  []string{"http://localhost:3000"},
	}
	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg)
}
