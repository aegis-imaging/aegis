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

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/digest"
	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/migrate"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	// Docker HEALTHCHECK support: the distroless container has no shell or curl,
	// so the binary itself can ping /healthz when invoked with --healthcheck.
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		port := "8080"
		if p := os.Getenv("PORT"); p != "" {
			port = p
		}
		resp, err := http.Get("http://localhost:" + port + "/healthz")
		if err != nil {
			os.Exit(1)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

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
	if err := applyDBSessionTimezone(ctx, db, cfg.AppTimezone); err != nil {
		log.Fatalf("database timezone setup (%s): %v", cfg.AppTimezone, err)
	}
	log.Printf("database session timezone set to %s", cfg.AppTimezone)

	// Production connection pool tuning.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := migrate.Run(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("migrations complete")

	bootstrapFirstAdmin(context.Background(), db, cfg)

	var store storage.Storage
	switch cfg.StorageMode {
	case "gcs":
		gcsStore, err := storage.NewGCS(context.Background(), cfg.GCSBucket)
		if err != nil {
			log.Fatalf("init gcs storage: %v", err)
		}
		store = gcsStore
	case "s3":
		s3Store, err := storage.NewS3(context.Background(), cfg.S3Bucket, cfg.S3Region, cfg.S3Endpoint)
		if err != nil {
			log.Fatalf("init s3 storage: %v", err)
		}
		store = s3Store
	default:
		store = storage.NewLocal(cfg.LocalStorageDir, cfg.APIBaseURL)
	}

	srv := handler.NewServer(db, store, cfg)
	auth := middleware.RequireAuth(db, cfg)
	adminOnly := middleware.RequireRole("admin", db, cfg)

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

	// Contact form — public, called from landing page (Vercel or local dev).
	mux.HandleFunc("POST /api/contact", srv.ContactForm)

	// Public export endpoints — token-authenticated, no session required.
	mux.HandleFunc("GET /api/export/{token}/download", srv.ServeDicomDownloadByToken)
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

	// Projects — create/update require admin; list is public (upload portal).
	mux.HandleFunc("POST /api/projects", adminOnly(srv.CreateProject))
	mux.HandleFunc("GET /api/projects/{id}", auth(srv.GetProject))
	mux.HandleFunc("PUT /api/projects/{id}", adminOnly(srv.UpdateProject))

	// Anonymization profiles — per-project DICOM tag retention overrides.
	mux.HandleFunc("GET /api/projects/{projectID}/anon-profiles", auth(srv.ListAnonProfiles))
	mux.HandleFunc("POST /api/projects/{projectID}/anon-profiles", adminOnly(srv.CreateAnonProfile))
	mux.HandleFunc("PUT /api/projects/{projectID}/default-anon-profile", adminOnly(srv.SetDefaultAnonProfile))
	mux.HandleFunc("GET /api/anon-profiles/{id}", auth(srv.GetAnonProfile))
	mux.HandleFunc("PUT /api/anon-profiles/{id}", adminOnly(srv.UpdateAnonProfile))
	mux.HandleFunc("DELETE /api/anon-profiles/{id}", adminOnly(srv.DeleteAnonProfile))

	// Studies — list, detail, and shares readable by all; mutations require admin.
	mux.HandleFunc("GET /api/studies", auth(srv.ListStudies))
	mux.HandleFunc("GET /api/studies/{id}", auth(srv.GetStudy))
	mux.HandleFunc("GET /api/studies/{id}/audit", auth(srv.ListStudyAudit))
	mux.HandleFunc("GET /api/studies/{id}/diagnostics", auth(srv.GetStudyDiagnostics))
	mux.HandleFunc("POST /api/studies/{id}/approve", adminOnly(srv.ApproveStudy))
	mux.HandleFunc("POST /api/studies/{id}/reject", adminOnly(srv.RejectStudy))
	mux.HandleFunc("POST /api/studies/{id}/share", adminOnly(srv.CreateShare))
	mux.HandleFunc("GET /api/studies/{id}/shares", auth(srv.ListShares))

	mux.HandleFunc("DELETE /api/shares/{shareID}", adminOnly(srv.RevokeShare))

	// Internal enterprise ingestion path.
	mux.HandleFunc("POST /api/ingest", adminOnly(srv.InternalIngest))

	// Batch import — import DICOM files from a server-local directory.
	mux.HandleFunc("POST /api/import/batch", adminOnly(srv.BatchImport))

	mux.HandleFunc("GET /api/audit", auth(srv.ListAudit))

	// Institutions — organisations that send or receive studies.
	mux.HandleFunc("GET /api/institutions", auth(srv.ListInstitutions))
	mux.HandleFunc("POST /api/institutions", adminOnly(srv.CreateInstitution))
	mux.HandleFunc("GET /api/institutions/{id}", auth(srv.GetInstitution))
	mux.HandleFunc("PUT /api/institutions/{id}", adminOnly(srv.UpdateInstitution))
	mux.HandleFunc("DELETE /api/institutions/{id}", adminOnly(srv.DeleteInstitution))
	mux.HandleFunc("GET /api/institutions/{id}/projects", auth(srv.ListInstitutionProjects))
	mux.HandleFunc("POST /api/institutions/{id}/projects", adminOnly(srv.AddInstitutionProject))
	mux.HandleFunc("DELETE /api/institutions/{id}/projects/{projectID}", adminOnly(srv.RemoveInstitutionProject))

	// Destinations — external DICOM endpoints studies can be forwarded to.
	mux.HandleFunc("GET /api/destinations", auth(srv.ListDestinations))
	mux.HandleFunc("POST /api/destinations", adminOnly(srv.CreateDestination))
	mux.HandleFunc("PUT /api/destinations/{id}", adminOnly(srv.UpdateDestination))
	mux.HandleFunc("DELETE /api/destinations/{id}", adminOnly(srv.DeleteDestination))

	// Routing rules — condition → action mappings evaluated on study ingest.
	mux.HandleFunc("GET /api/routing-rules", auth(srv.ListRoutingRules))
	mux.HandleFunc("POST /api/routing-rules", adminOnly(srv.CreateRoutingRule))
	mux.HandleFunc("PUT /api/routing-rules/{id}", adminOnly(srv.UpdateRoutingRule))
	mux.HandleFunc("DELETE /api/routing-rules/{id}", adminOnly(srv.DeleteRoutingRule))
	mux.HandleFunc("POST /api/routing-rules/evaluate/{studyID}", adminOnly(srv.EvaluateRoutingRules))
	mux.HandleFunc("GET /api/studies/{studyID}/routing-log", auth(srv.GetStudyRoutingLog))

	// DIMSE retry control proxy — admin-only API façade over sidecar /ingest/retry* endpoints.
	mux.HandleFunc("GET /api/dimse/retry", adminOnly(srv.DimseRetryProxy))
	mux.HandleFunc("GET /api/dimse/retry/{path...}", adminOnly(srv.DimseRetryProxy))
	mux.HandleFunc("POST /api/dimse/retry", adminOnly(srv.DimseRetryProxy))
	mux.HandleFunc("POST /api/dimse/retry/{path...}", adminOnly(srv.DimseRetryProxy))

	// Email digest subscriptions — periodic summary emails per project.
	mux.HandleFunc("GET /api/digest-subscriptions", auth(srv.ListDigestSubscriptions))
	mux.HandleFunc("GET /api/projects/{projectID}/digest-subscriptions", auth(srv.ListDigestSubscriptions))
	mux.HandleFunc("POST /api/projects/{projectID}/digest-subscriptions", adminOnly(srv.CreateDigestSubscription))
	mux.HandleFunc("DELETE /api/digest-subscriptions/{id}", adminOnly(srv.DeleteDigestSubscription))

	// Admin users — authorised dashboard users and their roles.
	mux.HandleFunc("GET /api/admin-users", auth(srv.ListAdminUsers))
	mux.HandleFunc("POST /api/admin-users", adminOnly(srv.CreateAdminUser))
	mux.HandleFunc("PUT /api/admin-users/{id}", adminOnly(srv.UpdateAdminUser))
	mux.HandleFunc("DELETE /api/admin-users/{id}", adminOnly(srv.DeleteAdminUser))

	// Protocol templates — per-project MRI acquisition parameter expectations.
	mux.HandleFunc("GET /api/projects/{projectID}/protocol-templates", auth(srv.ListProtocolTemplates))
	mux.HandleFunc("POST /api/projects/{projectID}/protocol-templates", adminOnly(srv.CreateProtocolTemplate))
	mux.HandleFunc("GET /api/protocol-templates/{id}", auth(srv.GetProtocolTemplate))
	mux.HandleFunc("PUT /api/protocol-templates/{id}", adminOnly(srv.UpdateProtocolTemplate))
	mux.HandleFunc("DELETE /api/protocol-templates/{id}", adminOnly(srv.DeleteProtocolTemplate))

	// Async processing triggers — admin-initiated pipeline actions.
	mux.HandleFunc("POST /api/deface/{studyUID}", adminOnly(srv.TriggerDeface))
	mux.HandleFunc("POST /api/studies/{studyUID}/phi-scan", adminOnly(srv.TriggerPhiScan))
	mux.HandleFunc("POST /api/studies/{studyUID}/qc-check", adminOnly(srv.TriggerQcCheck))
	mux.HandleFunc("POST /api/studies/{studyUID}/bids-convert", adminOnly(srv.TriggerBidsConversion))
	mux.HandleFunc("GET /api/studies/{studyUID}/bids-download", auth(srv.ServeBidsDownload))
	mux.HandleFunc("POST /api/studies/{studyUID}/classify", adminOnly(srv.TriggerClassification))
	mux.HandleFunc("POST /api/studies/{studyUID}/protocol-check", adminOnly(srv.TriggerProtocolCheck))
	mux.HandleFunc("GET /api/studies/{studyUID}/dicom-download", auth(srv.ServeDicomDownload))
	mux.HandleFunc("POST /api/studies/{studyUID}/trigger-export", adminOnly(srv.TriggerExport))

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

func applyDBSessionTimezone(ctx context.Context, db *sql.DB, timezone string) error {
	var applied string
	return db.QueryRowContext(ctx, `SELECT set_config('TimeZone', $1, false)`, timezone).Scan(&applied)
}

// bootstrapFirstAdmin seeds the first admin user when FIRST_ADMIN_EMAIL is set
// and the admin_users table is empty. Idempotent — does nothing once any admin
// user exists, so it is safe to leave set permanently in production.
func bootstrapFirstAdmin(ctx context.Context, db *sql.DB, cfg *config.Config) {
	if cfg.FirstAdminEmail == "" {
		return
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&count); err != nil {
		log.Printf("first-admin bootstrap: count query failed: %v", err)
		return
	}
	if count > 0 {
		return
	}
	u := &model.AdminUser{
		Email:   cfg.FirstAdminEmail,
		Name:    cfg.FirstAdminName,
		Role:    "admin",
		Enabled: true,
	}
	if err := model.CreateAdminUser(ctx, db, u); err != nil {
		log.Printf("first-admin bootstrap: failed to create %s: %v", cfg.FirstAdminEmail, err)
		return
	}
	log.Printf("first-admin bootstrap: created admin user %s (id=%s)", u.Email, u.ID)
}
