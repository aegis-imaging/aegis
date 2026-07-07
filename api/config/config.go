package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	StorageMode     string // "local", "gcs", "s3", or "azure"
	LocalStorageDir string
	APIBaseURL          string // for generating local upload URLs
	ExportPortalBaseURL string // base URL of the export portal UI — used in share email links
	UploadPortalBaseURL string // UPLOAD_PORTAL_BASE_URL — base URL of the upload portal UI; used in uploader-invite redeem links (empty = fall back to LandingBaseURL path)
	AppTimezone         string // database session + server log timezone (default UTC)

	// GCP (only used when StorageMode = "gcs")
	GCPProject      string
	GCSBucket       string
	DicomDataset    string
	DicomStoreRaw   string
	DicomStoreClean string

	// AWS (only used when StorageMode = "s3")
	S3Bucket   string
	S3Region   string
	S3Endpoint string // optional — set for S3-compatible stores (MinIO, LocalStack)

	// Azure (only used when StorageMode = "azure")
	AzureStorageAccount   string // AZURE_STORAGE_ACCOUNT — e.g. aegisproddicom
	AzureStorageContainer string // AZURE_STORAGE_CONTAINER — default "dicom"

	// Defacing service (Python Cloud Run sidecar)
	// Empty string disables the defacing service call (pipeline still records status).
	DefacingServiceURL string

	// Burned-in PHI detection service (Python Cloud Run sidecar)
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	PhiDetectionServiceURL string

	// QC automation service (Python Cloud Run sidecar)
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	QcServiceURL string

	// BIDS conversion service (Python Cloud Run sidecar)
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	BidsServiceURL string

	// Classification service (Python Cloud Run sidecar) — fills in missing modality/body_part.
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	ClassificationServiceURL string

	// Protocol compliance service (Python Cloud Run sidecar) — checks MRI acquisition parameters.
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	ProtocolServiceURL string

	// DIMSE receiver service (Python sidecar) — accepts DICOM C-STORE from PACS systems.
	DimseReceiverURL string
	// Optional operator key used by API when proxying DIMSE retry-control endpoints.
	DimseOperatorAPIKey string

	// Neuroimaging analytics service (Python Cloud Run sidecar) — FreeSurfer, FSL, ANTs, SPM.
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	AnalyticsServiceURL string

	// Spinal Cord Toolbox service (Python Cloud Run sidecar) — SCT CLI tools.
	// Empty string disables the service call (studies stay in "pending" until service is configured).
	SctServiceURL string

	// Synthetic MRI generation service (Python Cloud Run sidecar).
	// Empty string disables the endpoint (returns 503 until configured).
	SynthServiceURL string

	// CORS
	AllowedOrigins []string

	// Auth
	AuthEnabled  bool
	AuthProvider string // AUTH_PROVIDER — "auto" (default), "iap" (GCP), "azure" (Azure AD), or "aws" (ALB + Cognito)
	DevUserEmail string // DEV_USER_EMAIL — auto-authenticated email when AUTH_ENABLED=false
	AWSALBRegion string // AWS_REGION — region for ALB public key endpoint (default us-east-1)

	// Pipeline
	PipelineAuto bool // PIPELINE_AUTO — auto-dispatch processing after routing (default true)

	// Per-IP rate limiting on public upload/ingest endpoints.
	RateLimitEnabled bool    // RATE_LIMIT_ENABLED — default false
	RateLimitRPS     float64 // RATE_LIMIT_RPS — requests per second per IP (default 20)
	RateLimitBurst   int     // RATE_LIMIT_BURST — burst size per IP (default 50)

	// Email (SMTP)
	// EmailEnabled is derived: true when SMTPHost is non-empty.
	SMTPHost     string // SMTP_HOST — e.g. localhost; leave empty to disable email
	SMTPPort     string // SMTP_PORT — default 587; use 1025 with Mailpit
	SMTPFrom     string // SMTP_FROM — envelope sender address
	SMTPUsername string // SMTP_USERNAME — omit for unauthenticated relays
	SMTPPassword string // SMTP_PASSWORD
	EmailEnabled bool   // true when SMTPHost != ""

	// Contact form recipient
	ContactEmail string // CONTACT_EMAIL — where contact form submissions go (default: contact@aegisimaging.ai)

	// Invite request flow — "Request access" on the gate page.
	// InviteRequestSecret signs one-time approval tokens sent to the admin.
	// InviteRequestAdminEmail is who receives the notification (defaults to ContactEmail).
	// LandingBaseURL is used to build /?invite=CODE links in the code-issued email.
	InviteRequestSecret     string // INVITE_REQUEST_SECRET — HMAC key for approval tokens; feature disabled when empty
	InviteRequestAdminEmail string // INVITE_REQUEST_ADMIN_EMAIL — who gets notified (default: ContactEmail)
	LandingBaseURL          string // LANDING_BASE_URL — base URL for invite links (default: https://aegisimaging.ai)
	AdminDashboardURL       string // ADMIN_DASHBOARD_URL — used in approval email back-links (default: https://admin.aegisimaging.ai)

	// SLA stuck-study alerting — sends email when studies idle too long in pipeline.
	// Disabled when SLAPipelineMinutes == 0 or SLAAlertEmail is empty.
	SLAPipelineMinutes int    // SLA_PIPELINE_MINUTES — alert when study idle > N min (0 = disabled)
	SLACooldownHours   int    // SLA_COOLDOWN_HOURS — re-alert cooldown per study (default 24)
	SLAAlertEmail      string // SLA_ALERT_EMAIL — recipient for stuck-study alerts

	// Pipeline failure alerting — sends email whenever a pipeline service step fails.
	// Disabled when PipelineAlertEmail is empty or SMTP is not configured.
	PipelineAlertEmail  string // PIPELINE_ALERT_EMAIL — recipient for pipeline step failure alerts
	AutoShareExpiryDays int    // AUTO_SHARE_EXPIRY_DAYS — days until auto-generated share link expires (default 30)

	// Destination health probe scheduler — automatically probes all enabled destinations.
	// Disabled when DestHealthInterval is 0.
	DestHealthInterval  int    // DEST_HEALTH_INTERVAL — probe interval in seconds (0 = disabled)
	DestHealthAlertEmail string // DEST_HEALTH_ALERT_EMAIL — email for failure transition alerts

	// Audit log retention — purge audit_trail rows older than N days.
	// 0 (default) = keep forever.
	AuditRetentionDays int // AUDIT_RETENTION_DAYS

	// First-admin bootstrap — seeds the first admin user on startup when admin_users is empty.
	// Idempotent: has no effect once any admin user exists.
	FirstAdminEmail string // FIRST_ADMIN_EMAIL
	FirstAdminName  string // FIRST_ADMIN_NAME (optional; defaults to email address)

	// Satellite router enrollment (POST /api/satellites/enroll).
	// SatelliteCAEnabled gates the endpoint entirely. When the cert/key paths are
	// empty and SatelliteCAEphemeral=true the API mints a self-signed CA in
	// memory at startup — useful for dev/test, not durable across restarts.
	SatelliteCAEnabled       bool   // SATELLITE_CA_ENABLED — default true
	SatelliteCACertPath      string // SATELLITE_CA_CERT_PATH
	SatelliteCAKeyPath       string // SATELLITE_CA_KEY_PATH
	SatelliteCAEphemeral     bool   // SPOKE_CA_EPHEMERAL — when paths unset, generate in-memory CA (default true)
	SatelliteCertValidityDays int   // SPOKE_CERT_VALIDITY_DAYS — default 365
	SatelliteEnrollmentTokenTTLHours int // SPOKE_ENROLLMENT_TOKEN_TTL_HOURS — default 72
}

func Load() *Config {
	smtpHost := os.Getenv("SMTP_HOST")
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		dbHost := envOr("DB_HOST", "localhost")
		dbPort := envOr("DB_PORT", "5432")
		dbName := envOr("DB_NAME", "aegis")
		dbUser := envOr("DB_USER", "aegis")
		dbPassword := envOr("DB_PASSWORD", "aegis")
		dbSSLMode := envOr("DB_SSLMODE", "disable")
		databaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			url.QueryEscape(dbUser),
			url.QueryEscape(dbPassword),
			dbHost,
			dbPort,
			dbName,
			dbSSLMode,
		)
	}

	return &Config{
		Port:            envOr("PORT", "8080"),
		DatabaseURL:     databaseURL,
		StorageMode:     envOr("STORAGE_MODE", "local"),
		LocalStorageDir: envOr("LOCAL_STORAGE_DIR", "./data"),
		APIBaseURL:          envOr("API_BASE_URL", "http://localhost:8080"),
		ExportPortalBaseURL: os.Getenv("EXPORT_PORTAL_BASE_URL"), // e.g. https://export.aegisimaging.ai
		UploadPortalBaseURL: os.Getenv("UPLOAD_PORTAL_BASE_URL"), // e.g. https://upload.aegisimaging.ai
		AppTimezone:         envOr("APP_TIMEZONE", "UTC"),

		GCPProject:      os.Getenv("GCP_PROJECT"),
		GCSBucket:       os.Getenv("GCS_BUCKET"),
		DicomDataset:    os.Getenv("DICOM_DATASET"),
		DicomStoreRaw:   envOr("DICOM_STORE_RAW", "raw"),
		DicomStoreClean: envOr("DICOM_STORE_CLEAN", "clean"),

		S3Bucket:   os.Getenv("S3_BUCKET"),
		S3Region:   envOr("S3_REGION", "us-east-1"),
		S3Endpoint: os.Getenv("S3_ENDPOINT"), // e.g. http://localhost:4566 for LocalStack

		AzureStorageAccount:   os.Getenv("AZURE_STORAGE_ACCOUNT"),
		AzureStorageContainer: envOr("AZURE_STORAGE_CONTAINER", "dicom"),

		DefacingServiceURL:       os.Getenv("DEFACING_SERVICE_URL"),       // e.g. http://localhost:8081
		PhiDetectionServiceURL:   sidecarURL("PHI_DETECTION_SERVICE_URL",   "phi"),
		QcServiceURL:             sidecarURL("QC_SERVICE_URL",              "qc"),
		BidsServiceURL:           sidecarURL("BIDS_SERVICE_URL",            "bids"),
		ClassificationServiceURL: sidecarURL("CLASSIFICATION_SERVICE_URL",  "classify"),
		ProtocolServiceURL:       sidecarURL("PROTOCOL_SERVICE_URL",        "protocol"),
		SynthServiceURL:          sidecarURL("SYNTH_SERVICE_URL",           "synth"),
		DimseReceiverURL:         os.Getenv("DIMSE_RECEIVER_URL"),          // e.g. http://localhost:8087
		DimseOperatorAPIKey:      os.Getenv("DIMSE_OPERATOR_API_KEY"),
		AnalyticsServiceURL:      os.Getenv("ANALYTICS_SERVICE_URL"),       // e.g. http://localhost:8089
		SctServiceURL:            os.Getenv("SCT_SERVICE_URL"),             // e.g. http://localhost:8090

		PipelineAuto: os.Getenv("PIPELINE_AUTO") != "false",

		RateLimitEnabled: os.Getenv("RATE_LIMIT_ENABLED") == "true",
		RateLimitRPS:     floatEnvOr("RATE_LIMIT_RPS", 20),
		RateLimitBurst:   envInt("RATE_LIMIT_BURST", 50),

		AllowedOrigins: strings.Split(envOr("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:3002,http://localhost:3003,http://localhost:3004,http://localhost:3006,https://aegisimaging.ai,https://www.aegisimaging.ai"), ","),

		AuthEnabled:  os.Getenv("AUTH_ENABLED") == "true",
		AuthProvider: envOr("AUTH_PROVIDER", "auto"),
		DevUserEmail: envOr("DEV_USER_EMAIL", "ai@aegisimaging.ai"),
		AWSALBRegion: envOr("AWS_REGION", envOr("AWS_DEFAULT_REGION", "us-east-1")),

		SMTPHost:     smtpHost,
		SMTPPort:     envOr("SMTP_PORT", "587"),
		SMTPFrom:     envOr("SMTP_FROM", "noreply@aegisimaging.ai"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		EmailEnabled: smtpHost != "",

		ContactEmail: envOr("CONTACT_EMAIL", "contact@aegisimaging.ai"),

		InviteRequestSecret:     os.Getenv("INVITE_REQUEST_SECRET"),
		InviteRequestAdminEmail: os.Getenv("INVITE_REQUEST_ADMIN_EMAIL"),
		LandingBaseURL:          envOr("LANDING_BASE_URL", "https://aegisimaging.ai"),
		AdminDashboardURL:       envOr("ADMIN_DASHBOARD_URL", "https://admin.aegisimaging.ai"),

		SLAPipelineMinutes: envInt("SLA_PIPELINE_MINUTES", 0),
		SLACooldownHours:   envInt("SLA_COOLDOWN_HOURS", 24),
		SLAAlertEmail:      os.Getenv("SLA_ALERT_EMAIL"),
		PipelineAlertEmail:  os.Getenv("PIPELINE_ALERT_EMAIL"),
		AutoShareExpiryDays: envInt("AUTO_SHARE_EXPIRY_DAYS", 30),

		DestHealthInterval:   envInt("DEST_HEALTH_INTERVAL", 0),
		DestHealthAlertEmail: os.Getenv("DEST_HEALTH_ALERT_EMAIL"),

		AuditRetentionDays: envInt("AUDIT_RETENTION_DAYS", 0),

		FirstAdminEmail: os.Getenv("FIRST_ADMIN_EMAIL"),
		FirstAdminName:  envOr("FIRST_ADMIN_NAME", os.Getenv("FIRST_ADMIN_EMAIL")),

		SatelliteCAEnabled:               os.Getenv("SATELLITE_CA_ENABLED") != "false",
		SatelliteCACertPath:              os.Getenv("SATELLITE_CA_CERT_PATH"),
		SatelliteCAKeyPath:               os.Getenv("SATELLITE_CA_KEY_PATH"),
		SatelliteCAEphemeral:             os.Getenv("SPOKE_CA_EPHEMERAL") != "false",
		SatelliteCertValidityDays:        envInt("SPOKE_CERT_VALIDITY_DAYS", 365),
		SatelliteEnrollmentTokenTTLHours: envInt("SPOKE_ENROLLMENT_TOKEN_TTL_HOURS", 72),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func floatEnvOr(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// sidecarURL resolves the per-sidecar base URL from environment.
// Legacy override: if the per-sidecar env var (e.g. PHI_DETECTION_SERVICE_URL)
// is set, use it unchanged — preserves backward compatibility for any local
// dev setup that still runs the old standalone containers.
// Otherwise derive it from the unified DICOM_TOOLS_URL by appending the
// sub-module's prefix (e.g. DICOM_TOOLS_URL + "/phi"). This is the path that
// will be taken in production once the consolidated dicom-tools Cloud Run
// service is deployed.
// Returns empty string if neither is set, which keeps the existing
// "feature disabled" semantics on the consuming handlers.
func sidecarURL(legacyEnvKey, prefix string) string {
	if v := os.Getenv(legacyEnvKey); v != "" {
		return v
	}
	if base := os.Getenv("DICOM_TOOLS_URL"); base != "" {
		return base + "/" + prefix
	}
	return ""
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
