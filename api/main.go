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
	"github.com/aegis-imaging/aegis/api/retention"
	"github.com/aegis-imaging/aegis/api/sla"
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

	// Per-IP rate limiter for public endpoints (upload, ingest, contact).
	// Enabled via RATE_LIMIT_ENABLED=true; defaults to 20 req/s, burst 50.
	var rl *middleware.RateLimiter
	if cfg.RateLimitEnabled {
		rl = middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
		log.Printf("rate limiting enabled: %.0f req/s per IP, burst %d", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
	// rateLimit wraps a HandlerFunc with the IP limiter when enabled.
	rateLimit := func(h http.HandlerFunc) http.Handler {
		if rl == nil {
			return h
		}
		return rl.Handler(h)
	}

	mux := http.NewServeMux()

	// ── Public routes (no auth) ──────────────────────────────────────

	mux.HandleFunc("GET /healthz", srv.Healthz)

	// Upload portal needs project list and active anon profile.
	mux.HandleFunc("GET /api/projects", srv.ListProjects)
	mux.HandleFunc("GET /api/projects/{slug}/active-anon-profile", srv.GetDefaultAnonProfile)

	// Upload portal — public-facing, rate-limited.
	mux.Handle("POST /api/upload/init", rateLimit(srv.UploadInit))
	mux.Handle("PUT /api/upload/file/{sessionID}/{index}", rateLimit(srv.UploadFile))
	mux.Handle("POST /api/upload/complete", rateLimit(srv.UploadComplete))

	// Contact form — public, rate-limited.
	mux.Handle("POST /api/contact", rateLimit(srv.ContactForm))

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
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}/metadata", srv.DicomwebInstanceMetadata)

	// Raw DICOMweb proxy — identical to /dicomweb but WADO-RS always reads from the
	// pre-defacing "raw" store. Used by OHIF's "dicomweb-raw" data source for
	// side-by-side defacing review in the admin dashboard.
	mux.HandleFunc("GET /dicomweb-raw/studies", srv.DicomwebStudies)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series", srv.DicomwebSeries)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series/{seriesUID}/instances", srv.DicomwebInstances)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}", srv.DicomwebRawRetrieveInstance)
	mux.HandleFunc("GET /dicomweb-raw/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}/metadata", srv.DicomwebRawInstanceMetadata)

	// ── Admin-protected routes (require auth) ────────────────────────

	// Auth identity endpoint.
	mux.HandleFunc("GET /api/auth/me", auth(srv.AuthMe))

	// Dashboard stats — lightweight study pipeline overview.
	mux.HandleFunc("GET /api/stats", auth(srv.GetStats))
	mux.HandleFunc("GET /api/stats/breakdown", auth(srv.GetBreakdownStats))
	mux.HandleFunc("GET /api/stats/timeline", auth(srv.GetTimeline))
	mux.HandleFunc("GET /api/storage/stats", auth(srv.GetStorageStats))

	// System health summary — aggregated operational status panel.
	mux.HandleFunc("GET /api/system/health-summary", auth(srv.GetSystemHealthSummary))

	// Projects — create/update require admin; list is public (upload portal).
	mux.HandleFunc("POST /api/projects", adminOnly(srv.CreateProject))
	mux.HandleFunc("GET /api/projects/{id}", auth(srv.GetProject))
	mux.HandleFunc("PUT /api/projects/{id}", adminOnly(srv.UpdateProject))
	mux.HandleFunc("PUT /api/projects/{id}/retention", adminOnly(srv.SetProjectRetention))
	mux.HandleFunc("PUT /api/projects/{id}/sla-threshold", adminOnly(srv.SetProjectSLAThreshold))
	mux.HandleFunc("POST /api/projects/{id}/clone", adminOnly(srv.CloneProject))
	mux.HandleFunc("POST /api/projects/{id}/archive", adminOnly(srv.ArchiveProject))
	mux.HandleFunc("POST /api/projects/{id}/restore", adminOnly(srv.RestoreProject))
	mux.HandleFunc("GET /api/projects/{id}/phi-config", auth(srv.GetProjectPhiConfig))
	mux.HandleFunc("PUT /api/projects/{id}/phi-config", adminOnly(srv.UpdateProjectPhiConfig))
	mux.HandleFunc("GET /api/projects/{id}/compliance-report", auth(srv.GetProjectComplianceReport))

	// Anonymization profiles — per-project DICOM tag retention overrides.
	mux.HandleFunc("GET /api/projects/{projectID}/anon-profiles", auth(srv.ListAnonProfiles))
	mux.HandleFunc("POST /api/projects/{projectID}/anon-profiles", adminOnly(srv.CreateAnonProfile))
	mux.HandleFunc("PUT /api/projects/{projectID}/default-anon-profile", adminOnly(srv.SetDefaultAnonProfile))
	mux.HandleFunc("GET /api/anon-profiles/{id}", auth(srv.GetAnonProfile))
	mux.HandleFunc("PUT /api/anon-profiles/{id}", adminOnly(srv.UpdateAnonProfile))
	mux.HandleFunc("DELETE /api/anon-profiles/{id}", adminOnly(srv.DeleteAnonProfile))

	// Studies — list, detail, and shares readable by all; mutations require admin.
	mux.HandleFunc("GET /api/studies", auth(srv.ListStudies))
	mux.HandleFunc("GET /api/studies.csv", auth(srv.ExportStudiesCSV))
	mux.HandleFunc("GET /api/studies/stuck", auth(srv.GetStuckStudies))
	mux.HandleFunc("GET /api/studies/events", auth(srv.StudyEvents))
	mux.HandleFunc("GET /api/studies/{id}", auth(srv.GetStudy))
	mux.HandleFunc("DELETE /api/studies/{id}", adminOnly(srv.DeleteStudy))
	mux.HandleFunc("GET /api/study-uid/{studyUID}", auth(srv.GetStudyByUID))
	mux.HandleFunc("GET /api/studies/{id}/audit", auth(srv.ListStudyAudit))
	mux.HandleFunc("POST /api/studies/{id}/viewed", auth(srv.RecordStudyView))
	mux.HandleFunc("GET /api/studies/{id}/series", auth(srv.ListStudySeries))
	mux.HandleFunc("GET /api/studies/{id}/diagnostics", auth(srv.GetStudyDiagnostics))
	mux.HandleFunc("GET /api/studies/{studyUID}/dicom-tags", auth(srv.InspectDicomTags))
	mux.HandleFunc("POST /api/studies/bulk", adminOnly(srv.BulkStudyAction))
	mux.HandleFunc("POST /api/studies/bulk-pipeline-trigger", adminOnly(srv.BulkPipelineTrigger))
	mux.HandleFunc("POST /api/studies/{id}/notes", adminOnly(srv.AddStudyNote))
	mux.HandleFunc("PATCH /api/studies/{id}/flag", adminOnly(srv.PatchStudyFlag))
	mux.HandleFunc("POST /api/studies/{id}/reset-pipeline-step", adminOnly(srv.ResetPipelineStep))
	mux.HandleFunc("POST /api/studies/{id}/approve", adminOnly(srv.ApproveStudy))
	mux.HandleFunc("POST /api/studies/{id}/reject", adminOnly(srv.RejectStudy))
	mux.HandleFunc("POST /api/studies/{id}/reactivate", adminOnly(srv.ReactivateStudy))
	mux.HandleFunc("POST /api/studies/{id}/share", adminOnly(srv.CreateShare))
	mux.HandleFunc("GET /api/studies/{id}/shares", auth(srv.ListShares))

	mux.HandleFunc("GET /api/shares", auth(srv.ListAllShares))
	mux.HandleFunc("GET /api/shares/{shareID}/downloads", auth(srv.GetShareDownloads))
	mux.HandleFunc("GET /api/export-analytics", auth(srv.GetExportAnalytics))
	mux.HandleFunc("DELETE /api/shares/{shareID}", adminOnly(srv.RevokeShare))
	mux.HandleFunc("PATCH /api/shares/{shareID}/extend", adminOnly(srv.ExtendShare))

	// Internal enterprise ingestion path.
	mux.HandleFunc("POST /api/ingest", adminOnly(srv.InternalIngest))

	// Batch import — import DICOM files from a server-local directory.
	mux.HandleFunc("POST /api/import/batch", adminOnly(srv.BatchImport))

	// TCIA (Cancer Imaging Archive) — browse public brain MRI collections and import series.
	mux.HandleFunc("GET /api/tcia/series", auth(srv.GetTCIASeries))
	mux.HandleFunc("POST /api/tcia/import", adminOnly(srv.ImportTCIASeries))

	mux.HandleFunc("GET /api/audit", auth(srv.ListAudit))
	mux.HandleFunc("GET /api/audit.csv", auth(srv.ExportAuditCSV))
	mux.HandleFunc("GET /api/audit/actors", auth(srv.GetAuditActors))

	// Institutions — organisations that send or receive studies.
	mux.HandleFunc("GET /api/institutions", auth(srv.ListInstitutions))
	mux.HandleFunc("POST /api/institutions", adminOnly(srv.CreateInstitution))
	mux.HandleFunc("GET /api/institutions/{id}", auth(srv.GetInstitution))
	mux.HandleFunc("GET /api/institutions/{id}/stats", auth(srv.GetInstitutionStats))
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
	mux.HandleFunc("POST /api/destinations/{id}/test", adminOnly(srv.TestDestination))

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

	// API keys — long-lived machine-to-machine credentials.
	mux.HandleFunc("GET /api/api-keys", auth(srv.ListAPIKeys))
	mux.HandleFunc("POST /api/api-keys", adminOnly(srv.CreateAPIKey))
	mux.HandleFunc("PATCH /api/api-keys/{id}/enable", adminOnly(srv.EnableAPIKey))
	mux.HandleFunc("PATCH /api/api-keys/{id}/disable", adminOnly(srv.DisableAPIKey))
	mux.HandleFunc("POST /api/api-keys/{id}/rotate", adminOnly(srv.RotateAPIKey))
	mux.HandleFunc("DELETE /api/api-keys/{id}", adminOnly(srv.DeleteAPIKey))

	// Study labels — free-text tags applied by admin users for structured triage.
	mux.HandleFunc("GET /api/studies/{id}/labels", auth(srv.ListStudyLabels))
	mux.HandleFunc("POST /api/studies/{id}/labels", adminOnly(srv.AddStudyLabel))
	mux.HandleFunc("DELETE /api/studies/{id}/labels/{labelID}", adminOnly(srv.DeleteStudyLabel))
	mux.HandleFunc("POST /api/studies/bulk-label", adminOnly(srv.BulkLabelStudies))

	// Webhook subscriptions — HTTP callbacks for study lifecycle events.
	mux.HandleFunc("GET /api/webhook-subscriptions", auth(srv.ListWebhooks))
	mux.HandleFunc("POST /api/webhook-subscriptions", adminOnly(srv.CreateWebhook))
	mux.HandleFunc("GET /api/webhook-subscriptions/{id}", auth(srv.GetWebhook))
	mux.HandleFunc("PUT /api/webhook-subscriptions/{id}", adminOnly(srv.UpdateWebhook))
	mux.HandleFunc("DELETE /api/webhook-subscriptions/{id}", adminOnly(srv.DeleteWebhook))
	mux.HandleFunc("GET /api/webhook-subscriptions/{id}/deliveries", auth(srv.GetWebhookDeliveries))
	mux.HandleFunc("GET /api/webhook-subscriptions/{id}/stats", auth(srv.GetWebhookStats))
	mux.HandleFunc("POST /api/webhook-subscriptions/{id}/test", adminOnly(srv.TestWebhookDelivery))
	mux.HandleFunc("GET /api/webhook-deliveries", auth(srv.ListAllDeliveries))
	mux.HandleFunc("POST /api/webhook-deliveries/{id}/retry", adminOnly(srv.RetryDelivery))

	// Subject-session linking — group studies by de-identified subject pseudonym.
	mux.HandleFunc("GET /api/subjects", auth(srv.ListSubjects))
	mux.HandleFunc("PUT /api/studies/{id}/subject", adminOnly(srv.SetStudySubject))
	mux.HandleFunc("PUT /api/studies/{id}/project", adminOnly(srv.ReassignStudy))

	// Federation peers — trusted remote AEGIS instances (stub for future cross-tenant federation).
	mux.HandleFunc("GET /api/federation-peers", auth(srv.ListFederationPeers))
	mux.HandleFunc("POST /api/federation-peers", adminOnly(srv.CreateFederationPeer))
	mux.HandleFunc("GET /api/federation-peers/{id}", auth(srv.GetFederationPeer))
	mux.HandleFunc("PUT /api/federation-peers/{id}", adminOnly(srv.UpdateFederationPeer))
	mux.HandleFunc("DELETE /api/federation-peers/{id}", adminOnly(srv.DeleteFederationPeer))

	// Admin users — authorised dashboard users and their roles.
	mux.HandleFunc("GET /api/admin-users", auth(srv.ListAdminUsers))
	mux.HandleFunc("POST /api/admin-users", adminOnly(srv.CreateAdminUser))
	mux.HandleFunc("PUT /api/admin-users/{id}", adminOnly(srv.UpdateAdminUser))
	mux.HandleFunc("DELETE /api/admin-users/{id}", adminOnly(srv.DeleteAdminUser))

	// Project-level batch export — dispatch all eligible approved studies.
	mux.HandleFunc("POST /api/projects/{id}/export-batch", adminOnly(srv.ExportBatch))

	// Protocol templates — per-project MRI acquisition parameter expectations.
	mux.HandleFunc("GET /api/projects/{projectID}/protocol-templates", auth(srv.ListProtocolTemplates))
	mux.HandleFunc("GET /api/projects/{projectID}/protocol-templates/export", auth(srv.ExportProtocolTemplates))
	mux.HandleFunc("POST /api/projects/{projectID}/protocol-templates/import", adminOnly(srv.ImportProtocolTemplates))
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

	// Synthetic MRI generation — creates a new synthetic brain MRI study.
	mux.HandleFunc("POST /api/studies/generate-synthetic", adminOnly(srv.GenerateSyntheticStudy))

	// Public landing page demo — generate a synthetic brain MRI and poll its pipeline status.
	// Rate-limited but no auth required; used by aegisimaging.ai to drive the interactive demo.
	mux.Handle("POST /api/demo/generate", rateLimit(srv.DemoGenerate))
	mux.Handle("GET /api/demo/study/{studyID}", rateLimit(srv.DemoStudyStatus))

	// Landing page invite codes — server-side access control for the private beta gate.
	// Validate is public/rate-limited; management endpoints are admin-only.
	mux.Handle("POST /api/invite/validate", rateLimit(srv.ValidateInviteCode))
	mux.Handle("POST /api/invite/request", rateLimit(srv.RequestInvite))
	mux.HandleFunc("GET /api/invite/request/approve", srv.ApproveInviteRequest)
	mux.HandleFunc("GET /api/invite-codes", adminOnly(srv.ListInviteCodes))
	mux.HandleFunc("POST /api/invite-codes", adminOnly(srv.CreateInviteCode))
	mux.HandleFunc("POST /api/invite-codes/{id}/revoke", adminOnly(srv.RevokeInviteCode))
	mux.HandleFunc("DELETE /api/invite-codes/{id}", adminOnly(srv.DeleteInviteCode))
	// Invite requests — stored in DB, manageable from admin dashboard.
	mux.HandleFunc("GET /api/invite/requests", auth(srv.ListInviteRequestsAdmin))
	mux.HandleFunc("POST /api/invite/requests/{id}/approve", adminOnly(srv.ApproveInviteRequestAdmin))
	mux.HandleFunc("POST /api/invite/requests/{id}/deny", adminOnly(srv.DenyInviteRequestAdmin))

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

	mailer := email.New(cfg)

	// Start the email digest scheduler (hourly check, no-op when SMTP is disabled).
	digestCtx, digestCancel := context.WithCancel(context.Background())
	defer digestCancel()
	digest.Start(digestCtx, db, mailer)

	// Start the SLA stuck-study alert scheduler (hourly, no-op when SLA_PIPELINE_MINUTES=0).
	slaCtx, slaCancel := context.WithCancel(context.Background())
	defer slaCancel()
	sla.Start(slaCtx, db, mailer, cfg.SLAPipelineMinutes, cfg.SLACooldownHours, cfg.SLAAlertEmail)

	// Start the study retention worker (daily sweep, no-op when no projects have retention_days set).
	retentionCtx, retentionCancel := context.WithCancel(context.Background())
	defer retentionCancel()
	retention.Start(retentionCtx, db)

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
