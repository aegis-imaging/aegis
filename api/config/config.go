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
	StorageMode     string // "local", "gcs", or "s3"
	LocalStorageDir string
	APIBaseURL          string // for generating local upload URLs
	ExportPortalBaseURL string // base URL of the export portal UI — used in share email links
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

	// Synthetic MRI generation service (Python Cloud Run sidecar).
	// Empty string disables the endpoint (returns 503 until configured).
	SynthServiceURL string

	// CORS
	AllowedOrigins []string

	// Auth
	AuthEnabled  bool
	AuthProvider string // AUTH_PROVIDER — "auto" (default), "iap" (GCP), "azure" (Azure AD), or "aws" (ALB + Cognito)
	DevUserEmail string // DEV_USER_EMAIL — auto-authenticated email when AUTH_ENABLED=false

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

	// SLA stuck-study alerting — sends email when studies idle too long in pipeline.
	// Disabled when SLAPipelineMinutes == 0 or SLAAlertEmail is empty.
	SLAPipelineMinutes int    // SLA_PIPELINE_MINUTES — alert when study idle > N min (0 = disabled)
	SLACooldownHours   int    // SLA_COOLDOWN_HOURS — re-alert cooldown per study (default 24)
	SLAAlertEmail      string // SLA_ALERT_EMAIL — recipient for stuck-study alerts

	// Pipeline failure alerting — sends email whenever a pipeline service step fails.
	// Disabled when PipelineAlertEmail is empty or SMTP is not configured.
	PipelineAlertEmail string // PIPELINE_ALERT_EMAIL — recipient for pipeline step failure alerts

	// First-admin bootstrap — seeds the first admin user on startup when admin_users is empty.
	// Idempotent: has no effect once any admin user exists.
	FirstAdminEmail string // FIRST_ADMIN_EMAIL
	FirstAdminName  string // FIRST_ADMIN_NAME (optional; defaults to email address)
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
		databaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			url.QueryEscape(dbUser),
			url.QueryEscape(dbPassword),
			dbHost,
			dbPort,
			dbName,
		)
	}

	return &Config{
		Port:            envOr("PORT", "8080"),
		DatabaseURL:     databaseURL,
		StorageMode:     envOr("STORAGE_MODE", "local"),
		LocalStorageDir: envOr("LOCAL_STORAGE_DIR", "./data"),
		APIBaseURL:          envOr("API_BASE_URL", "http://localhost:8080"),
		ExportPortalBaseURL: os.Getenv("EXPORT_PORTAL_BASE_URL"), // e.g. https://export.aegisimaging.ai
		AppTimezone:         envOr("APP_TIMEZONE", "UTC"),

		GCPProject:      os.Getenv("GCP_PROJECT"),
		GCSBucket:       os.Getenv("GCS_BUCKET"),
		DicomDataset:    os.Getenv("DICOM_DATASET"),
		DicomStoreRaw:   envOr("DICOM_STORE_RAW", "raw"),
		DicomStoreClean: envOr("DICOM_STORE_CLEAN", "clean"),

		S3Bucket:   os.Getenv("S3_BUCKET"),
		S3Region:   envOr("S3_REGION", "us-east-1"),
		S3Endpoint: os.Getenv("S3_ENDPOINT"), // e.g. http://localhost:4566 for LocalStack

		DefacingServiceURL:       os.Getenv("DEFACING_SERVICE_URL"),       // e.g. http://localhost:8081
		PhiDetectionServiceURL:   os.Getenv("PHI_DETECTION_SERVICE_URL"),  // e.g. http://localhost:8082
		QcServiceURL:             os.Getenv("QC_SERVICE_URL"),             // e.g. http://localhost:8083
		BidsServiceURL:           os.Getenv("BIDS_SERVICE_URL"),           // e.g. http://localhost:8084
		ClassificationServiceURL: os.Getenv("CLASSIFICATION_SERVICE_URL"), // e.g. http://localhost:8085
		ProtocolServiceURL:       os.Getenv("PROTOCOL_SERVICE_URL"),       // e.g. http://localhost:8086
		DimseReceiverURL:         os.Getenv("DIMSE_RECEIVER_URL"),         // e.g. http://localhost:8087
		DimseOperatorAPIKey:      os.Getenv("DIMSE_OPERATOR_API_KEY"),
		SynthServiceURL:          os.Getenv("SYNTH_SERVICE_URL"),          // e.g. http://localhost:8088

		PipelineAuto: os.Getenv("PIPELINE_AUTO") != "false",

		RateLimitEnabled: os.Getenv("RATE_LIMIT_ENABLED") == "true",
		RateLimitRPS:     floatEnvOr("RATE_LIMIT_RPS", 20),
		RateLimitBurst:   envInt("RATE_LIMIT_BURST", 50),

		AllowedOrigins: strings.Split(envOr("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:3002,http://localhost:3003,http://localhost:3004"), ","),

		AuthEnabled:  os.Getenv("AUTH_ENABLED") == "true",
		AuthProvider: envOr("AUTH_PROVIDER", "auto"),
		DevUserEmail: envOr("DEV_USER_EMAIL", "dev@aegis.local"),

		SMTPHost:     smtpHost,
		SMTPPort:     envOr("SMTP_PORT", "587"),
		SMTPFrom:     envOr("SMTP_FROM", "noreply@aegis.local"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		EmailEnabled: smtpHost != "",

		ContactEmail: envOr("CONTACT_EMAIL", "contact@aegisimaging.ai"),

		SLAPipelineMinutes: envInt("SLA_PIPELINE_MINUTES", 0),
		SLACooldownHours:   envInt("SLA_COOLDOWN_HOURS", 24),
		SLAAlertEmail:      os.Getenv("SLA_ALERT_EMAIL"),
		PipelineAlertEmail: os.Getenv("PIPELINE_ALERT_EMAIL"),

		FirstAdminEmail: os.Getenv("FIRST_ADMIN_EMAIL"),
		FirstAdminName:  envOr("FIRST_ADMIN_NAME", os.Getenv("FIRST_ADMIN_EMAIL")),
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

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
