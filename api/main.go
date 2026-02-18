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
	"github.com/msenjem/aegis/api/digest"
	"github.com/msenjem/aegis/api/email"
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
	auth := middleware.RequireAuth(db, cfg)

	mux := http.NewServeMux()

	// ── Public routes (no auth) ──────────────────────────────────────

	mux.HandleFunc("GET /healthz", srv.Healthz)

	// Upload portal needs project list and active anon profile.
	mux.HandleFunc("GET /api/projects", srv.ListProjects)
	mux.HandleFunc("GET /api/projects/{slug}/active-anon-profile", srv.GetDefaultAnonProfile)

	// Upload portal — public-facing, no auth.
	mux.HandleFunc("POST /api/upload/init", srv.UploadInit)
	mux.HandleFunc("PUT /api/upload/file/{sessionID}/{index}", srv.UploadFile)
	mux.HandleFunc("POST /api/upload/complete", srv.UploadComplete)

	// Public export endpoint — token-authenticated, no session required.
	mux.HandleFunc("GET /api/export/{token}", srv.RedeemExport)

	// Local dev only: serve stored files over HTTP (in GCS mode, signed URLs are used instead).
	mux.HandleFunc("GET /api/storage/{key...}", srv.ServeStorageFile)

	// DICOMweb proxy — QIDO-RS (metadata) + WADO-RS (retrieve), used by OHIF Viewer.
	mux.HandleFunc("GET /dicomweb/studies", srv.DicomwebStudies)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series", srv.DicomwebSeries)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances", srv.DicomwebInstances)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}", srv.DicomwebRetrieveInstance)

	// Raw DICOMweb proxy — identical to /dicomweb but WADO-RS always reads from the
	// pre-defacing "raw" store. Used by OHIF's "dicomweb-raw" data source for
	// side-by-side defacing review in the admin dashboard.
	mux.HandleFunc("GET /dicomweb-raw/studies", srv.DicomwebStudies)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series", srv.DicomwebSeries)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series/{seriesUID}/instances", srv.DicomwebInstances)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}", srv.DicomwebRawRetrieveInstance)

	// ── Admin-protected routes (require auth) ────────────────────────

	// Auth identity endpoint.
	mux.HandleFunc("GET /api/auth/me", auth(srv.AuthMe))

	// Projects — create/update require auth; list is public (upload portal).
	mux.HandleFunc("POST /api/projects", auth(srv.CreateProject))
	mux.HandleFunc("GET /api/projects/{id}", auth(srv.GetProject))
	mux.HandleFunc("PUT /api/projects/{id}", auth(srv.UpdateProject))

	// Anonymization profiles — per-project DICOM tag retention overrides.
	mux.HandleFunc("GET /api/projects/{projectID}/anon-profiles", auth(srv.ListAnonProfiles))
	mux.HandleFunc("POST /api/projects/{projectID}/anon-profiles", auth(srv.CreateAnonProfile))
	mux.HandleFunc("PUT /api/projects/{projectID}/default-anon-profile", auth(srv.SetDefaultAnonProfile))
	mux.HandleFunc("GET /api/anon-profiles/{id}", auth(srv.GetAnonProfile))
	mux.HandleFunc("PUT /api/anon-profiles/{id}", auth(srv.UpdateAnonProfile))
	mux.HandleFunc("DELETE /api/anon-profiles/{id}", auth(srv.DeleteAnonProfile))

	// Studies — all admin operations.
	mux.HandleFunc("GET /api/studies", auth(srv.ListStudies))
	mux.HandleFunc("POST /api/studies/{id}/approve", auth(srv.ApproveStudy))
	mux.HandleFunc("POST /api/studies/{id}/reject", auth(srv.RejectStudy))
	mux.HandleFunc("POST /api/studies/{id}/share", auth(srv.CreateShare))
	mux.HandleFunc("GET /api/studies/{id}/shares", auth(srv.ListShares))

	mux.HandleFunc("DELETE /api/shares/{shareID}", auth(srv.RevokeShare))

	// Internal enterprise ingestion path.
	mux.HandleFunc("POST /api/ingest", auth(srv.InternalIngest))

	// Batch import — import DICOM files from a server-local directory.
	mux.HandleFunc("POST /api/import/batch", auth(srv.BatchImport))

	mux.HandleFunc("GET /api/audit", auth(srv.ListAudit))

	// Institutions — organisations that send or receive studies.
	mux.HandleFunc("GET /api/institutions", auth(srv.ListInstitutions))
	mux.HandleFunc("POST /api/institutions", auth(srv.CreateInstitution))
	mux.HandleFunc("GET /api/institutions/{id}", auth(srv.GetInstitution))
	mux.HandleFunc("PUT /api/institutions/{id}", auth(srv.UpdateInstitution))
	mux.HandleFunc("DELETE /api/institutions/{id}", auth(srv.DeleteInstitution))
	mux.HandleFunc("GET /api/institutions/{id}/projects", auth(srv.ListInstitutionProjects))
	mux.HandleFunc("POST /api/institutions/{id}/projects", auth(srv.AddInstitutionProject))
	mux.HandleFunc("DELETE /api/institutions/{id}/projects/{projectID}", auth(srv.RemoveInstitutionProject))

	// Destinations — external DICOM endpoints studies can be forwarded to.
	mux.HandleFunc("GET /api/destinations", auth(srv.ListDestinations))
	mux.HandleFunc("POST /api/destinations", auth(srv.CreateDestination))
	mux.HandleFunc("PUT /api/destinations/{id}", auth(srv.UpdateDestination))
	mux.HandleFunc("DELETE /api/destinations/{id}", auth(srv.DeleteDestination))

	// Routing rules — condition → action mappings evaluated on study ingest.
	mux.HandleFunc("GET /api/routing-rules", auth(srv.ListRoutingRules))
	mux.HandleFunc("POST /api/routing-rules", auth(srv.CreateRoutingRule))
	mux.HandleFunc("PUT /api/routing-rules/{id}", auth(srv.UpdateRoutingRule))
	mux.HandleFunc("DELETE /api/routing-rules/{id}", auth(srv.DeleteRoutingRule))
	mux.HandleFunc("POST /api/routing-rules/evaluate/{studyID}", auth(srv.EvaluateRoutingRules))
	mux.HandleFunc("GET /api/studies/{studyID}/routing-log", auth(srv.GetStudyRoutingLog))

	// Email digest subscriptions — periodic summary emails per project.
	mux.HandleFunc("GET /api/digest-subscriptions", auth(srv.ListDigestSubscriptions))
	mux.HandleFunc("GET /api/projects/{projectID}/digest-subscriptions", auth(srv.ListDigestSubscriptions))
	mux.HandleFunc("POST /api/projects/{projectID}/digest-subscriptions", auth(srv.CreateDigestSubscription))
	mux.HandleFunc("DELETE /api/digest-subscriptions/{id}", auth(srv.DeleteDigestSubscription))

	// Admin users — authorised dashboard users and their roles.
	mux.HandleFunc("GET /api/admin-users", auth(srv.ListAdminUsers))
	mux.HandleFunc("POST /api/admin-users", auth(srv.CreateAdminUser))
	mux.HandleFunc("PUT /api/admin-users/{id}", auth(srv.UpdateAdminUser))
	mux.HandleFunc("DELETE /api/admin-users/{id}", auth(srv.DeleteAdminUser))

	// Async processing triggers — admin-initiated pipeline actions.
	mux.HandleFunc("POST /api/deface/{studyUID}", auth(srv.TriggerDeface))
	mux.HandleFunc("POST /api/studies/{studyUID}/phi-scan", auth(srv.TriggerPhiScan))
	mux.HandleFunc("POST /api/studies/{studyUID}/qc-check", auth(srv.TriggerQcCheck))
	mux.HandleFunc("POST /api/studies/{studyUID}/bids-convert", auth(srv.TriggerBidsConversion))
	mux.HandleFunc("GET /api/studies/{studyUID}/bids-download", auth(srv.ServeBidsDownload))
	mux.HandleFunc("POST /api/studies/{studyUID}/classify", auth(srv.TriggerClassification))

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

	// Start the email digest scheduler (hourly check, no-op when SMTP is disabled).
	digestCtx, digestCancel := context.WithCancel(context.Background())
	defer digestCancel()
	digest.Start(digestCtx, db, email.New(cfg))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("aegis-api listening on :%s (storage=%s, auth=%v, provider=%s)",
			cfg.Port, cfg.StorageMode, cfg.AuthEnabled, cfg.AuthProvider)
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
