package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	StorageMode     string // "local", "gcs", or "s3"
	LocalStorageDir string
	APIBaseURL      string // for generating local upload URLs
	AppTimezone     string // database session + server log timezone (default UTC)

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

	// CORS
	AllowedOrigins []string

	// Auth
	AuthEnabled  bool
	AuthProvider string // AUTH_PROVIDER — "auto" (default), "iap" (GCP), "azure" (Azure AD), or "aws" (ALB + Cognito)
	DevUserEmail string // DEV_USER_EMAIL — auto-authenticated email when AUTH_ENABLED=false

	// Pipeline
	PipelineAuto bool // PIPELINE_AUTO — auto-dispatch processing after routing (default true)

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
}

func Load() *Config {
	smtpHost := os.Getenv("SMTP_HOST")
	return &Config{
		Port:            envOr("PORT", "8080"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://aegis:aegis@localhost:5432/aegis?sslmode=disable"),
		StorageMode:     envOr("STORAGE_MODE", "local"),
		LocalStorageDir: envOr("LOCAL_STORAGE_DIR", "./data"),
		APIBaseURL:      envOr("API_BASE_URL", "http://localhost:8080"),
		AppTimezone:     envOr("APP_TIMEZONE", "UTC"),

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

		PipelineAuto: os.Getenv("PIPELINE_AUTO") != "false",

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
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
