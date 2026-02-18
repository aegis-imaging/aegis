package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	StorageMode     string // "local" or "gcs"
	LocalStorageDir string
	APIBaseURL      string // for generating local upload URLs

	// GCP (only used when StorageMode = "gcs")
	GCPProject      string
	GCSBucket       string
	DicomDataset    string
	DicomStoreRaw   string
	DicomStoreClean string

	// Defacing service (Python Cloud Run sidecar)
	// Empty string disables the defacing service call (pipeline still records status).
	DefacingServiceURL string

	// CORS
	AllowedOrigins []string

	// Auth
	AuthEnabled bool

	// Email (SMTP)
	// EmailEnabled is derived: true when SMTPHost is non-empty.
	SMTPHost     string // SMTP_HOST — e.g. localhost; leave empty to disable email
	SMTPPort     string // SMTP_PORT — default 587; use 1025 with Mailpit
	SMTPFrom     string // SMTP_FROM — envelope sender address
	SMTPUsername string // SMTP_USERNAME — omit for unauthenticated relays
	SMTPPassword string // SMTP_PASSWORD
	EmailEnabled bool   // true when SMTPHost != ""
}

func Load() *Config {
	smtpHost := os.Getenv("SMTP_HOST")
	return &Config{
		Port:            envOr("PORT", "8080"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://aegis:aegis@localhost:5432/aegis?sslmode=disable"),
		StorageMode:     envOr("STORAGE_MODE", "local"),
		LocalStorageDir: envOr("LOCAL_STORAGE_DIR", "./data"),
		APIBaseURL:      envOr("API_BASE_URL", "http://localhost:8080"),

		GCPProject:      os.Getenv("GCP_PROJECT"),
		GCSBucket:       os.Getenv("GCS_BUCKET"),
		DicomDataset:    os.Getenv("DICOM_DATASET"),
		DicomStoreRaw:   envOr("DICOM_STORE_RAW", "raw"),
		DicomStoreClean: envOr("DICOM_STORE_CLEAN", "clean"),

		DefacingServiceURL: os.Getenv("DEFACING_SERVICE_URL"), // e.g. http://localhost:8081

		AllowedOrigins: strings.Split(envOr("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001"), ","),

		AuthEnabled: os.Getenv("AUTH_ENABLED") == "true",

		SMTPHost:     smtpHost,
		SMTPPort:     envOr("SMTP_PORT", "587"),
		SMTPFrom:     envOr("SMTP_FROM", "noreply@aegis.local"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		EmailEnabled: smtpHost != "",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
