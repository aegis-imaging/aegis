package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/handler"
	"github.com/msenjem/aegis/api/middleware"
	"github.com/msenjem/aegis/api/migrate"
	"github.com/msenjem/aegis/api/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database ping: %v", err)
	}

	if err := migrate.Run(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("migrations complete")

	var store storage.Storage
	switch cfg.StorageMode {
	case "gcs":
		gcsStore, err := storage.NewGCS(context.Background(), cfg.GCSBucket)
		if err != nil {
			log.Fatalf("init gcs storage: %v", err)
		}
		store = gcsStore
	default:
		store = storage.NewLocal(cfg.LocalStorageDir, cfg.APIBaseURL)
	}

	srv := handler.NewServer(db, store, cfg)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", srv.Healthz)

	mux.HandleFunc("GET /api/projects", srv.ListProjects)
	mux.HandleFunc("POST /api/projects", srv.CreateProject)

	mux.HandleFunc("POST /api/upload/init", srv.UploadInit)
	mux.HandleFunc("PUT /api/upload/file/{sessionID}/{index}", srv.UploadFile)
	mux.HandleFunc("POST /api/upload/complete", srv.UploadComplete)

	mux.HandleFunc("GET /api/studies", srv.ListStudies)
	mux.HandleFunc("POST /api/studies/{id}/approve", srv.ApproveStudy)
	mux.HandleFunc("POST /api/studies/{id}/reject", srv.RejectStudy)
	mux.HandleFunc("POST /api/studies/{id}/share", srv.CreateShare)
	mux.HandleFunc("GET /api/studies/{id}/shares", srv.ListShares)

	mux.HandleFunc("DELETE /api/shares/{shareID}", srv.RevokeShare)

	// Public export endpoint — token-authenticated, no session required.
	mux.HandleFunc("GET /api/export/{token}", srv.RedeemExport)

	// Internal enterprise ingestion path.
	mux.HandleFunc("POST /api/ingest", srv.InternalIngest)

	// Local dev only: serve stored files over HTTP (in GCS mode, signed URLs are used instead).
	mux.HandleFunc("GET /api/storage/{key...}", srv.ServeStorageFile)

	mux.HandleFunc("GET /api/audit", srv.ListAudit)

	mux.HandleFunc("POST /api/deface/{studyUID}", srv.TriggerDeface)

	var h http.Handler = mux
	h = middleware.Recover(h)
	h = middleware.Logging(h)
	h = middleware.CORS(h, cfg.AllowedOrigins)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("aegis-api listening on :%s (storage=%s)", cfg.Port, cfg.StorageMode)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-done
	log.Println("shutting down...")

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("stopped")
}
