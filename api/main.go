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

	"github.com/aegis-imaging/aegis/api/audit_retention"
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
	case "azure":
		azStore, err := storage.NewAzureBlob(context.Background(), cfg.AzureStorageAccount, cfg.AzureStorageContainer)
		if err != nil {
			log.Fatalf("init azure storage: %v", err)
		}
		store = azStore
	default:
		store = storage.NewLocal(cfg.LocalStorageDir, cfg.APIBaseURL)
	}

	srv := handler.NewServer(db, store, cfg)
	auth := middleware.RequireAuth(db, cfg)
	adminOnly := middleware.RequireRole("admin", db, cfg)
	optionalAuth := middleware.OptionalAuth(db, cfg)

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

	// Upload portal and authenticated users — optionalAuth so unauthenticated callers
	// still get the non-restricted project list while researchers get their scoped list.
	mux.HandleFunc("GET /api/projects", optionalAuth(srv.ListProjects))
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

	// DICOMweb proxy — QIDO-RS (metadata) + WADO-RS (retrieve), used by DWV viewer.
	mux.HandleFunc("GET /dicomweb/studies", srv.DicomwebStudies)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series", srv.DicomwebSeries)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances", srv.DicomwebInstances)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}", srv.DicomwebRetrieveInstance)
	mux.HandleFunc("GET /dicomweb/studies/{studyUID}/series/{seriesUID}/instances/{sopUID}/metadata", srv.DicomwebInstanceMetadata)

	// Raw DICOMweb proxy — identical to /dicomweb but WADO-RS always reads from the
	// pre-defacing "raw" store. Used by the DWV viewer (?store=raw) for
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
	mux.HandleFunc("GET /api/stats/processing-times", auth(srv.GetProcessingTimes))
	mux.HandleFunc("GET /api/stats/routing", auth(srv.GetRoutingStats))
	mux.HandleFunc("GET /api/stats/routing-rules", auth(srv.GetRoutingRuleStats))
	mux.HandleFunc("GET /api/stats/pipeline-funnel", auth(srv.GetPipelineFunnel))
	mux.HandleFunc("GET /api/stats/project-health", auth(srv.GetProjectHealth))
	mux.HandleFunc("GET /api/stats/daily-summary", auth(srv.GetDailySummary))
	mux.HandleFunc("GET /api/stats/protocol-trend", auth(srv.GetProtocolTrend))
	mux.HandleFunc("GET /api/stats/phi-trend", auth(srv.GetPhiTrend))
	mux.HandleFunc("GET /api/stats/modality-trend", auth(srv.GetModalityTrend))
	mux.HandleFunc("GET /api/stats/label-usage", auth(srv.GetLabelUsage))
	mux.HandleFunc("GET /api/stats/source-trend", auth(srv.GetSourceTrend))
	mux.HandleFunc("GET /api/stats/institution-attribution", auth(srv.GetInstitutionAttributionStats))
	mux.HandleFunc("GET /api/storage/stats", auth(srv.GetStorageStats))

	// System health summary — aggregated operational status panel.
	mux.HandleFunc("GET /api/system/health-summary", auth(srv.GetSystemHealthSummary))

	// Projects — create/update require admin; list is public (upload portal).
	mux.HandleFunc("POST /api/projects", adminOnly(srv.CreateProject))
	mux.HandleFunc("GET /api/projects/{id}", auth(srv.GetProject))
	mux.HandleFunc("PUT /api/projects/{id}", adminOnly(srv.UpdateProject))
	mux.HandleFunc("PUT /api/projects/{id}/retention", adminOnly(srv.SetProjectRetention))
	mux.HandleFunc("GET /api/projects/{id}/retention-preview", auth(srv.GetRetentionPreview))
	mux.HandleFunc("PUT /api/projects/{id}/sla-threshold", adminOnly(srv.SetProjectSLAThreshold))
	mux.HandleFunc("PUT /api/projects/{id}/storage-quota", adminOnly(srv.SetStorageQuota))
	mux.HandleFunc("GET /api/projects/{id}/storage-usage", auth(srv.GetStorageUsage))
	mux.HandleFunc("POST /api/projects/{id}/clone", adminOnly(srv.CloneProject))
	mux.HandleFunc("POST /api/projects/{id}/archive", adminOnly(srv.ArchiveProject))
	mux.HandleFunc("POST /api/projects/{id}/restore", adminOnly(srv.RestoreProject))
	mux.HandleFunc("GET /api/projects/{id}/phi-config", auth(srv.GetProjectPhiConfig))
	mux.HandleFunc("PUT /api/projects/{id}/phi-config", adminOnly(srv.UpdateProjectPhiConfig))
	mux.HandleFunc("GET /api/projects/{id}/compliance-report", auth(srv.GetProjectComplianceReport))
	mux.HandleFunc("GET /api/projects/{id}/compliance-report.csv", auth(srv.ExportComplianceReportCSV))
	mux.HandleFunc("GET /api/projects/{id}/cohort-report", auth(srv.GetCohortReport))
	mux.HandleFunc("GET /api/projects/{id}/institution-breakdown", auth(srv.GetInstitutionBreakdown))
	mux.HandleFunc("POST /api/projects/{id}/re-evaluate-routing", adminOnly(srv.BulkReEvaluateRouting))
	mux.HandleFunc("GET /api/projects/{id}/routing-rules/export", auth(srv.ExportRoutingRules))
	mux.HandleFunc("POST /api/projects/{id}/routing-rules/import", adminOnly(srv.ImportRoutingRules))

	// Project restricted flag — platform admin only.
	mux.HandleFunc("PUT /api/projects/{id}/restricted", adminOnly(srv.SetProjectRestricted))

	// Project members — list uses auth; write ops use auth + handler-level role check
	// (platform admin OR project owner are both allowed to manage members).
	mux.HandleFunc("GET /api/projects/{id}/members", auth(srv.ListProjectMembers))
	mux.HandleFunc("POST /api/projects/{id}/members", auth(srv.AddProjectMember))
	mux.HandleFunc("PUT /api/projects/{id}/members/{memberID}", auth(srv.UpdateProjectMember))
	mux.HandleFunc("DELETE /api/projects/{id}/members/{memberID}", auth(srv.RemoveProjectMember))

	// Project dashboard summary — compact KPI snapshot.
	mux.HandleFunc("GET /api/projects/{id}/summary", auth(srv.GetProjectDashboardSummary))

	// Project milestones — track project progress.
	mux.HandleFunc("GET /api/projects/{id}/milestones", auth(srv.ListProjectMilestones))
	mux.HandleFunc("POST /api/projects/{id}/milestones", adminOnly(srv.CreateProjectMilestone))
	mux.HandleFunc("PATCH /api/projects/{id}/milestones/{milestoneID}", adminOnly(srv.UpdateProjectMilestone))
	mux.HandleFunc("DELETE /api/projects/{id}/milestones/{milestoneID}", adminOnly(srv.DeleteProjectMilestone))

	// Custom field definitions — per-project custom metadata fields.
	mux.HandleFunc("GET /api/projects/{id}/custom-fields", auth(srv.ListCustomFieldDefinitions))
	mux.HandleFunc("POST /api/projects/{id}/custom-fields", adminOnly(srv.CreateCustomFieldDefinition))
	mux.HandleFunc("DELETE /api/projects/{id}/custom-fields/{fieldID}", adminOnly(srv.DeleteCustomFieldDefinition))

	// Auto-share rules — automatically create shares when studies are approved.
	mux.HandleFunc("GET /api/projects/{projectID}/auto-share-rules", auth(srv.ListAutoShareRules))
	mux.HandleFunc("POST /api/projects/{projectID}/auto-share-rules", adminOnly(srv.CreateAutoShareRule))
	mux.HandleFunc("PUT /api/auto-share-rules/{id}", adminOnly(srv.UpdateAutoShareRule))
	mux.HandleFunc("DELETE /api/auto-share-rules/{id}", adminOnly(srv.DeleteAutoShareRule))

	// Review checklist — per-project review checklist items.
	mux.HandleFunc("GET /api/projects/{projectID}/review-checklist", auth(srv.ListReviewChecklist))
	mux.HandleFunc("POST /api/projects/{projectID}/review-checklist", adminOnly(srv.CreateReviewChecklistItem))
	mux.HandleFunc("PUT /api/review-checklist/{id}", adminOnly(srv.UpdateReviewChecklistItem))
	mux.HandleFunc("DELETE /api/review-checklist/{id}", adminOnly(srv.DeleteReviewChecklistItem))

	// Project tags — categorize projects with tags.
	mux.HandleFunc("GET /api/projects/{id}/tags", auth(srv.ListProjectTags))
	mux.HandleFunc("POST /api/projects/{id}/tags", adminOnly(srv.AddProjectTag))
	mux.HandleFunc("DELETE /api/projects/{id}/tags/{tagID}", adminOnly(srv.DeleteProjectTag))

	// Anonymization profiles — per-project DICOM tag retention overrides.
	mux.HandleFunc("GET /api/projects/{projectID}/anon-profiles", auth(srv.ListAnonProfiles))
	mux.HandleFunc("POST /api/projects/{projectID}/anon-profiles", adminOnly(srv.CreateAnonProfile))
	mux.HandleFunc("PUT /api/projects/{projectID}/default-anon-profile", adminOnly(srv.SetDefaultAnonProfile))
	mux.HandleFunc("GET /api/anon-profiles/{id}", auth(srv.GetAnonProfile))
	mux.HandleFunc("PUT /api/anon-profiles/{id}", adminOnly(srv.UpdateAnonProfile))
	mux.HandleFunc("DELETE /api/anon-profiles/{id}", adminOnly(srv.DeleteAnonProfile))

	// Studies — list/detail are readable via auth; selected mutations enforce capability checks in handlers.
	mux.HandleFunc("GET /api/studies", auth(srv.ListStudies))
	mux.HandleFunc("GET /api/studies.csv", auth(srv.ExportStudiesCSV))
	mux.HandleFunc("GET /api/studies/stuck", auth(srv.GetStuckStudies))
	mux.HandleFunc("GET /api/studies/expiring", auth(srv.GetExpiringStudies))
	mux.HandleFunc("GET /api/studies/deleted", auth(srv.ListDeletedStudies))
	mux.HandleFunc("GET /api/studies/events", auth(srv.StudyEvents))
	mux.HandleFunc("GET /api/studies/priority-queue", auth(srv.GetStudyPriorityQueue))
	mux.HandleFunc("GET /api/studies/compare", auth(srv.CompareStudies))
	mux.HandleFunc("GET /api/studies/duplicates", auth(srv.GetStudyDuplicates))
	mux.HandleFunc("GET /api/studies/{id}", auth(srv.GetStudy))
	mux.HandleFunc("DELETE /api/studies/{id}", adminOnly(srv.DeleteStudy))
	mux.HandleFunc("GET /api/study-uid/{studyUID}", auth(srv.GetStudyByUID))
	mux.HandleFunc("GET /api/studies/{id}/audit", auth(srv.ListStudyAudit))
	mux.HandleFunc("GET /api/studies/{id}/audit.csv", auth(srv.ExportStudyAuditCSV))
	mux.HandleFunc("POST /api/studies/{id}/viewed", auth(srv.RecordStudyView))
	mux.HandleFunc("GET /api/studies/{id}/series", auth(srv.ListStudySeries))
	mux.HandleFunc("GET /api/studies/{id}/diagnostics", auth(srv.GetStudyDiagnostics))
	mux.HandleFunc("GET /api/studies/{id}/processing-summary", auth(srv.GetStudyProcessingSummary))
	mux.HandleFunc("GET /api/studies/{studyUID}/dicom-tags", auth(srv.InspectDicomTags))
	mux.HandleFunc("GET /api/studies/{studyUID}/anonymization-diff", auth(srv.GetAnonDiff))
	mux.HandleFunc("POST /api/studies/bulk", adminOnly(srv.BulkStudyAction))
	mux.HandleFunc("POST /api/studies/bulk-pipeline-trigger", adminOnly(srv.BulkPipelineTrigger))
	mux.HandleFunc("GET /api/studies/{id}/notes", auth(srv.ListStudyNotes))
	mux.HandleFunc("POST /api/studies/{id}/notes", auth(srv.AddStudyNote))
	mux.HandleFunc("PATCH /api/studies/{id}/flag", auth(srv.PatchStudyFlag))
	mux.HandleFunc("POST /api/studies/{id}/reset-pipeline-step", auth(srv.ResetPipelineStep))
	mux.HandleFunc("POST /api/studies/{id}/approve", auth(srv.ApproveStudy))
	mux.HandleFunc("POST /api/studies/{id}/reject", auth(srv.RejectStudy))
	mux.HandleFunc("POST /api/studies/{id}/reactivate", auth(srv.ReactivateStudy))
	mux.HandleFunc("POST /api/studies/{id}/soft-delete", adminOnly(srv.SoftDeleteStudy))
	mux.HandleFunc("POST /api/studies/{id}/restore", adminOnly(srv.RestoreStudy))
	mux.HandleFunc("POST /api/studies/{id}/share", adminOnly(srv.CreateShare))
	mux.HandleFunc("GET /api/studies/{id}/shares", auth(srv.ListShares))

	mux.HandleFunc("GET /api/shares", auth(srv.ListAllShares))
	mux.HandleFunc("GET /api/export-shares.csv", auth(srv.ExportSharesCSV))
	mux.HandleFunc("GET /api/shares/{shareID}/downloads", auth(srv.GetShareDownloads))
	mux.HandleFunc("GET /api/export-analytics", auth(srv.GetExportAnalytics))
	mux.HandleFunc("DELETE /api/shares/{shareID}", adminOnly(srv.RevokeShare))
	mux.HandleFunc("PATCH /api/shares/{shareID}/extend", adminOnly(srv.ExtendShare))

	// DICOMweb STOW-RS receiver — cross-cloud ingest endpoint, API-key authenticated.
	// /api/stow          — AEGIS-specific path (legacy / direct)
	// /api/stow/studies  — DICOMweb-standard path (routing engine appends /studies to dest URL)
	mux.Handle("POST /api/stow", rateLimit(srv.StowReceiver))
	mux.Handle("POST /api/stow/studies", rateLimit(srv.StowReceiver))

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

	// Institution activity — audit log per institution.
	mux.HandleFunc("GET /api/institutions/{id}/activity", auth(srv.GetInstitutionActivity))

	// Institution contacts — contact directory per institution.
	mux.HandleFunc("GET /api/institutions/{id}/contacts", auth(srv.ListInstitutionContacts))
	mux.HandleFunc("POST /api/institutions/{id}/contacts", adminOnly(srv.CreateInstitutionContact))
	mux.HandleFunc("DELETE /api/institutions/{id}/contacts/{contactID}", adminOnly(srv.DeleteInstitutionContact))

	// Destinations — external DICOM endpoints studies can be forwarded to.
	mux.HandleFunc("GET /api/destinations", auth(srv.ListDestinations))
	mux.HandleFunc("GET /api/destinations/health", auth(srv.GetAllDestinationsHealth))
	mux.HandleFunc("POST /api/destinations", adminOnly(srv.CreateDestination))
	mux.HandleFunc("PUT /api/destinations/{id}", adminOnly(srv.UpdateDestination))
	mux.HandleFunc("DELETE /api/destinations/{id}", adminOnly(srv.DeleteDestination))
	mux.HandleFunc("POST /api/destinations/{id}/test", adminOnly(srv.TestDestination))
	mux.HandleFunc("GET /api/destinations/{id}/stats", auth(srv.GetDestinationStats))
	mux.HandleFunc("GET /api/destinations/{id}/health", auth(srv.GetDestinationHealth))

	// Routing rules — condition → action mappings evaluated on study ingest.
	mux.HandleFunc("GET /api/routing-rules", auth(srv.ListRoutingRules))
	mux.HandleFunc("POST /api/routing-rules/simulate", auth(srv.SimulateRoutingRules))           // must be before {id} patterns
	mux.HandleFunc("POST /api/routing-rules/reorder", adminOnly(srv.ReorderRoutingRules))        // must be before {id} patterns
	mux.HandleFunc("POST /api/routing-rules/bulk-toggle", adminOnly(srv.BulkToggleRoutingRules)) // must be before {id} patterns
	mux.HandleFunc("POST /api/routing-rules", adminOnly(srv.CreateRoutingRule))
	mux.HandleFunc("PUT /api/routing-rules/{id}", adminOnly(srv.UpdateRoutingRule))
	mux.HandleFunc("DELETE /api/routing-rules/{id}", adminOnly(srv.DeleteRoutingRule))
	mux.HandleFunc("POST /api/routing-rules/evaluate/{studyID}", adminOnly(srv.EvaluateRoutingRules))
	mux.HandleFunc("GET /api/routing-rules/{id}/changelog", auth(srv.GetRoutingRuleChangelog))
	mux.HandleFunc("GET /api/studies/{studyID}/routing-log", auth(srv.GetStudyRoutingLog))

	// DIMSE retry control proxy — admin-only API façade over sidecar /ingest/retry* endpoints.
	mux.HandleFunc("GET /api/dimse/retry", adminOnly(srv.DimseRetryProxy))
	mux.HandleFunc("GET /api/dimse/retry/{path...}", adminOnly(srv.DimseRetryProxy))
	mux.HandleFunc("POST /api/dimse/retry", adminOnly(srv.DimseRetryProxy))
	mux.HandleFunc("POST /api/dimse/retry/{path...}", adminOnly(srv.DimseRetryProxy))

	// DIMSE C-FIND SCU — query a remote PACS/AE for matching studies/series/instances.
	mux.HandleFunc("POST /api/dimse/query", adminOnly(srv.DimseQuery))

	// DIMSE C-MOVE SCU — request a remote PACS to push a study to the AEGIS SCP.
	mux.HandleFunc("POST /api/dimse/retrieve", adminOnly(srv.DimseRetrieve))

	// DIMSE batch C-MOVE — retrieve multiple studies (up to 50) in one request.
	mux.HandleFunc("POST /api/dimse/retrieve-batch", adminOnly(srv.DimseRetrieveBatch))

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

	// Study assignment — assign studies to admin users for peer review.
	mux.HandleFunc("POST /api/studies/{id}/assign", adminOnly(srv.AssignStudy))
	mux.HandleFunc("DELETE /api/studies/{id}/assign", adminOnly(srv.UnassignStudy))

	// Study comments — threaded discussion on studies.
	mux.HandleFunc("GET /api/studies/{id}/comments", auth(srv.ListStudyComments))
	mux.HandleFunc("POST /api/studies/{id}/comments", adminOnly(srv.CreateStudyComment))
	mux.HandleFunc("DELETE /api/studies/{id}/comments/{commentID}", adminOnly(srv.DeleteStudyComment))

	// Study approval signatures — sign-off history.
	mux.HandleFunc("GET /api/studies/{id}/approvals", auth(srv.ListStudyApprovals))
	mux.HandleFunc("POST /api/studies/{id}/approvals", adminOnly(srv.RecordStudyApproval))

	// Study pinned notes — sticky notes visible at top of detail panel.
	mux.HandleFunc("GET /api/studies/{id}/pinned-notes", auth(srv.ListStudyPinnedNotes))
	mux.HandleFunc("POST /api/studies/{id}/pinned-notes", adminOnly(srv.CreateStudyPinnedNote))
	mux.HandleFunc("PATCH /api/studies/{id}/pinned-notes/{noteID}", adminOnly(srv.UpdateStudyPinnedNote))
	mux.HandleFunc("DELETE /api/studies/{id}/pinned-notes/{noteID}", adminOnly(srv.DeleteStudyPinnedNote))

	// Study activity timeline — unified feed of audit + comments.
	mux.HandleFunc("GET /api/studies/{id}/activity", auth(srv.GetStudyActivity))

	// Study transfer log — provenance tracking for project moves.
	mux.HandleFunc("GET /api/studies/{id}/transfers", auth(srv.ListStudyTransfers))
	mux.HandleFunc("POST /api/studies/{id}/transfer", adminOnly(srv.TransferStudy))

	// Study custom fields — per-study metadata from project-defined fields.
	mux.HandleFunc("GET /api/studies/{id}/custom-fields", auth(srv.ListStudyCustomFields))
	mux.HandleFunc("POST /api/studies/{id}/custom-fields", adminOnly(srv.SetStudyCustomField))

	// Study watchers — subscribe to study events.
	mux.HandleFunc("GET /api/studies/{id}/watchers", auth(srv.ListStudyWatchers))
	mux.HandleFunc("POST /api/studies/{id}/watch", auth(srv.WatchStudy))
	mux.HandleFunc("DELETE /api/studies/{id}/watch", auth(srv.UnwatchStudy))

	// Study review checklist — per-study checklist responses.
	mux.HandleFunc("GET /api/studies/{id}/checklist", auth(srv.ListStudyChecklistResponses))
	mux.HandleFunc("POST /api/studies/{id}/checklist", adminOnly(srv.UpsertStudyChecklistResponse))

	// Bulk study reassignment between projects.
	mux.HandleFunc("POST /api/studies/bulk-reassign", adminOnly(srv.BulkReassignStudies))

	// Study labels — free-text tags applied by admin users for structured triage.
	mux.HandleFunc("GET /api/studies/{id}/labels", auth(srv.ListStudyLabels))
	mux.HandleFunc("POST /api/studies/{id}/labels", adminOnly(srv.AddStudyLabel))
	mux.HandleFunc("DELETE /api/studies/{id}/labels/{labelID}", adminOnly(srv.DeleteStudyLabel))
	mux.HandleFunc("POST /api/studies/bulk-label", adminOnly(srv.BulkLabelStudies))
	mux.HandleFunc("POST /api/studies/bulk-share", adminOnly(srv.BulkCreateShares))
	mux.HandleFunc("POST /api/studies/bulk-custom-field", adminOnly(srv.BulkSetCustomField))

	// Study relationships — link studies as baseline/follow_up/comparison/replicate pairs.
	mux.HandleFunc("GET /api/studies/{id}/relationships", auth(srv.ListStudyRelationships))
	mux.HandleFunc("POST /api/studies/{id}/relationships", adminOnly(srv.CreateStudyRelationship))
	mux.HandleFunc("DELETE /api/studies/{id}/relationships/{relID}", adminOnly(srv.DeleteStudyRelationship))

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

	// Dashboard notifications — in-app notification bell.
	mux.HandleFunc("GET /api/notifications", auth(srv.ListNotifications))
	mux.HandleFunc("POST /api/notifications", adminOnly(srv.CreateNotification))
	mux.HandleFunc("POST /api/notifications/{id}/read", auth(srv.MarkNotificationRead))
	mux.HandleFunc("POST /api/notifications/read-all", auth(srv.MarkAllNotificationsRead))

	// Audit bookmarks — save important audit entries for compliance tracking.
	mux.HandleFunc("GET /api/audit-bookmarks", auth(srv.ListAuditBookmarks))
	mux.HandleFunc("POST /api/audit-bookmarks", auth(srv.CreateAuditBookmark))
	mux.HandleFunc("DELETE /api/audit-bookmarks/{id}", auth(srv.DeleteAuditBookmark))

	// Admin users — authorised dashboard users and their roles.
	mux.HandleFunc("GET /api/admin-users/activity", auth(srv.GetAdminUserActivity))
	mux.HandleFunc("GET /api/admin-users", auth(srv.ListAdminUsers))
	mux.HandleFunc("POST /api/admin-users", adminOnly(srv.CreateAdminUser))
	mux.HandleFunc("PUT /api/admin-users/{id}", adminOnly(srv.UpdateAdminUser))
	mux.HandleFunc("DELETE /api/admin-users/{id}", adminOnly(srv.DeleteAdminUser))
	mux.HandleFunc("POST /api/admin-users/{id}/send-invite", adminOnly(srv.SendAdminUserInvite))
	mux.HandleFunc("GET /api/admin-users/{id}/preferences", auth(srv.GetUserPreferences))
	mux.HandleFunc("PUT /api/admin-users/{id}/preferences", auth(srv.UpdateUserPreferences))
	mux.HandleFunc("GET /api/admin-users/{id}/sessions", auth(srv.ListSessions))

	// Auth session tracking.
	mux.HandleFunc("POST /api/auth/session", auth(srv.RecordSession))

	// Comment reactions — emoji reactions on study comments.
	mux.HandleFunc("GET /api/comments/{id}/reactions", auth(srv.ListCommentReactions))
	mux.HandleFunc("POST /api/comments/{id}/reactions", auth(srv.AddCommentReaction))
	mux.HandleFunc("DELETE /api/comments/{id}/reactions", auth(srv.DeleteCommentReaction))

	// Comment mentions — @user mentions in comments.
	mux.HandleFunc("GET /api/comments/{id}/mentions", auth(srv.ListCommentMentions))
	mux.HandleFunc("POST /api/comments/{id}/mentions", adminOnly(srv.AddCommentMention))
	mux.HandleFunc("GET /api/mentions", auth(srv.ListMyMentions))

	// Dashboard saved views — server-side filter presets.
	mux.HandleFunc("GET /api/saved-views", auth(srv.ListSavedViews))
	mux.HandleFunc("POST /api/saved-views", auth(srv.CreateSavedView))
	mux.HandleFunc("PUT /api/saved-views/{id}", auth(srv.UpdateSavedView))
	mux.HandleFunc("DELETE /api/saved-views/{id}", auth(srv.DeleteSavedView))

	// Export share templates — reusable share configurations.
	mux.HandleFunc("GET /api/export-share-templates", auth(srv.ListExportShareTemplates))
	mux.HandleFunc("POST /api/export-share-templates", adminOnly(srv.CreateExportShareTemplate))
	mux.HandleFunc("DELETE /api/export-share-templates/{id}", adminOnly(srv.DeleteExportShareTemplate))

	// Routing rule templates — shareable rule configurations.
	mux.HandleFunc("GET /api/routing-rule-templates", auth(srv.ListRoutingRuleTemplates))
	mux.HandleFunc("POST /api/routing-rule-templates", adminOnly(srv.CreateRoutingRuleTemplate))
	mux.HandleFunc("DELETE /api/routing-rule-templates/{id}", adminOnly(srv.DeleteRoutingRuleTemplate))

	// Project-level batch export — dispatch all eligible approved studies.
	mux.HandleFunc("POST /api/projects/{id}/export-batch", adminOnly(srv.ExportBatch))
	mux.HandleFunc("GET /api/projects/{id}/bids-export", auth(srv.ServeProjectBidsExport))
	mux.HandleFunc("GET /api/projects/{id}/bids-info", auth(srv.GetProjectBidsInfo))

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
	mux.HandleFunc("POST /api/studies/{studyUID}/pixel-redaction", adminOnly(srv.TriggerPixelRedaction))
	mux.HandleFunc("POST /api/studies/{studyUID}/qc-check", adminOnly(srv.TriggerQcCheck))
	mux.HandleFunc("POST /api/studies/{studyUID}/bids-convert", adminOnly(srv.TriggerBidsConversion))
	mux.HandleFunc("GET /api/studies/{studyUID}/bids-download", auth(srv.ServeBidsDownload))
	mux.HandleFunc("POST /api/studies/{studyUID}/classify", adminOnly(srv.TriggerClassification))
	mux.HandleFunc("POST /api/studies/{studyUID}/protocol-check", adminOnly(srv.TriggerProtocolCheck))
	mux.HandleFunc("GET /api/studies/{studyUID}/dicom-download", auth(srv.ServeDicomDownload))
	mux.HandleFunc("POST /api/studies/{studyUID}/analytics", adminOnly(srv.TriggerAnalytics))
	mux.HandleFunc("POST /api/studies/{studyUID}/longitudinal-analytics", adminOnly(srv.TriggerLongitudinalAnalytics))
	mux.HandleFunc("POST /api/studies/{studyUID}/run-sct", adminOnly(srv.TriggerSct))
	mux.HandleFunc("POST /api/studies/{studyUID}/trigger-export", adminOnly(srv.TriggerExport))

	// Biomarker database — ROI results, composite scores, longitudinal data.
	mux.HandleFunc("GET /api/studies/{id}/roi-results", auth(srv.ListROIResults))
	mux.HandleFunc("GET /api/studies/{id}/roi-results/summary", auth(srv.GetROIResultsSummary))
	mux.HandleFunc("GET /api/studies/{id}/composite-scores", auth(srv.ListCompositeScores))
	mux.HandleFunc("GET /api/studies/{id}/longitudinal-roi-results", auth(srv.ListLongitudinalROIResults))
	mux.HandleFunc("DELETE /api/studies/{id}/roi-results", adminOnly(srv.DeleteROIResults))
	mux.HandleFunc("GET /api/subjects/{subjectID}/roi-results", auth(srv.ListSubjectROIResults))
	mux.HandleFunc("GET /api/projects/{id}/roi-export", auth(srv.ExportProjectROIData))

	// Analyst QC ratings — human quality assessments for studies.
	mux.HandleFunc("GET /api/studies/{id}/qc-ratings", auth(srv.ListQCRatings))
	mux.HandleFunc("POST /api/studies/{id}/qc-ratings", adminOnly(srv.SubmitQCRating))
	mux.HandleFunc("PUT /api/qc-ratings/{id}", adminOnly(srv.UpdateQCRating))
	mux.HandleFunc("DELETE /api/qc-ratings/{id}", adminOnly(srv.DeleteQCRating))

	// Subject demographics — de-identified research metadata.
	mux.HandleFunc("GET /api/subjects/{subjectID}/demographics", auth(srv.GetSubjectDemographics))
	mux.HandleFunc("PUT /api/subjects/{subjectID}/demographics", adminOnly(srv.UpsertSubjectDemographics))
	mux.HandleFunc("GET /api/projects/{id}/demographics", auth(srv.ListProjectDemographics))
	mux.HandleFunc("GET /api/projects/{id}/demographics.csv", auth(srv.ExportProjectDemographicsCSV))

	// Analytics output files — browse and stream NIfTI/surface/stats files.
	mux.HandleFunc("GET /api/studies/{id}/analytics-files", auth(srv.ListAnalyticsFiles))
	mux.HandleFunc("GET /api/studies/{id}/analytics-files/{path...}", auth(srv.ServeAnalyticsFile))

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
	mux.HandleFunc("POST /api/invite-codes/{id}/send", adminOnly(srv.SendInviteCode))
	mux.HandleFunc("GET /api/invite-codes/{id}/activity", auth(srv.GetInviteCodeActivity))
	mux.HandleFunc("POST /api/invite-codes/{id}/send-admin-invite", adminOnly(srv.SendAdminInvite))
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

	// Start the destination health probe scheduler (no-op when DEST_HEALTH_INTERVAL=0).
	probeCtx, probeCancel := context.WithCancel(context.Background())
	defer probeCancel()
	srv.StartDestinationProbeScheduler(probeCtx, time.Duration(cfg.DestHealthInterval)*time.Second, cfg.DestHealthAlertEmail)

	// Start the study retention worker (daily sweep, no-op when no projects have retention_days set).
	retentionCtx, retentionCancel := context.WithCancel(context.Background())
	defer retentionCancel()
	retention.Start(retentionCtx, db)

	// Start the audit log purge worker (daily sweep, no-op when AUDIT_RETENTION_DAYS=0).
	auditRetentionCtx, auditRetentionCancel := context.WithCancel(context.Background())
	defer auditRetentionCancel()
	audit_retention.Start(auditRetentionCtx, db, cfg.AuditRetentionDays)

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
