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

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", srv.Healthz)

	mux.HandleFunc("GET /api/projects", srv.ListProjects)
	mux.HandleFunc("POST /api/projects", srv.CreateProject)
	mux.HandleFunc("GET /api/projects/{id}", srv.GetProject)
	mux.HandleFunc("PUT /api/projects/{id}", srv.UpdateProject)

	// Anonymization profiles — per-project DICOM tag retention overrides.
	mux.HandleFunc("GET /api/projects/{projectID}/anon-profiles", srv.ListAnonProfiles)
	mux.HandleFunc("POST /api/projects/{projectID}/anon-profiles", srv.CreateAnonProfile)
	mux.HandleFunc("PUT /api/projects/{projectID}/default-anon-profile", srv.SetDefaultAnonProfile)
	mux.HandleFunc("GET /api/projects/{slug}/active-anon-profile", srv.GetDefaultAnonProfile)
	mux.HandleFunc("GET /api/anon-profiles/{id}", srv.GetAnonProfile)
	mux.HandleFunc("PUT /api/anon-profiles/{id}", srv.UpdateAnonProfile)
	mux.HandleFunc("DELETE /api/anon-profiles/{id}", srv.DeleteAnonProfile)

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

	// Institutions — organisations that send or receive studies.
	mux.HandleFunc("GET /api/institutions", srv.ListInstitutions)
	mux.HandleFunc("POST /api/institutions", srv.CreateInstitution)
	mux.HandleFunc("GET /api/institutions/{id}", srv.GetInstitution)
	mux.HandleFunc("PUT /api/institutions/{id}", srv.UpdateInstitution)
	mux.HandleFunc("DELETE /api/institutions/{id}", srv.DeleteInstitution)
	mux.HandleFunc("GET /api/institutions/{id}/projects", srv.ListInstitutionProjects)
	mux.HandleFunc("POST /api/institutions/{id}/projects", srv.AddInstitutionProject)
	mux.HandleFunc("DELETE /api/institutions/{id}/projects/{projectID}", srv.RemoveInstitutionProject)

	// Destinations — external DICOM endpoints studies can be forwarded to.
	mux.HandleFunc("GET /api/destinations", srv.ListDestinations)
	mux.HandleFunc("POST /api/destinations", srv.CreateDestination)
	mux.HandleFunc("PUT /api/destinations/{id}", srv.UpdateDestination)
	mux.HandleFunc("DELETE /api/destinations/{id}", srv.DeleteDestination)

	// Routing rules — condition → action mappings evaluated on study ingest.
	mux.HandleFunc("GET /api/routing-rules", srv.ListRoutingRules)
	mux.HandleFunc("POST /api/routing-rules", srv.CreateRoutingRule)
	mux.HandleFunc("PUT /api/routing-rules/{id}", srv.UpdateRoutingRule)
	mux.HandleFunc("DELETE /api/routing-rules/{id}", srv.DeleteRoutingRule)
	mux.HandleFunc("POST /api/routing-rules/evaluate/{studyID}", srv.EvaluateRoutingRules)
	mux.HandleFunc("GET /api/studies/{studyID}/routing-log", srv.GetStudyRoutingLog)

	// Email digest subscriptions — periodic summary emails per project.
	mux.HandleFunc("GET /api/digest-subscriptions", srv.ListDigestSubscriptions)
	mux.HandleFunc("GET /api/projects/{projectID}/digest-subscriptions", srv.ListDigestSubscriptions)
	mux.HandleFunc("POST /api/projects/{projectID}/digest-subscriptions", srv.CreateDigestSubscription)
	mux.HandleFunc("DELETE /api/digest-subscriptions/{id}", srv.DeleteDigestSubscription)

	// Admin users — authorised dashboard users and their roles.
	mux.HandleFunc("GET /api/admin-users", srv.ListAdminUsers)
	mux.HandleFunc("POST /api/admin-users", srv.CreateAdminUser)
	mux.HandleFunc("PUT /api/admin-users/{id}", srv.UpdateAdminUser)
	mux.HandleFunc("DELETE /api/admin-users/{id}", srv.DeleteAdminUser)

	mux.HandleFunc("POST /api/deface/{studyUID}", srv.TriggerDeface)
	mux.HandleFunc("POST /api/studies/{studyUID}/phi-scan", srv.TriggerPhiScan)

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
