package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func clearEnvVars(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT", "DATABASE_URL", "STORAGE_MODE", "LOCAL_STORAGE_DIR", "API_BASE_URL",
		"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD",
		"APP_TIMEZONE",
		"AUTH_ENABLED", "AUTH_PROVIDER", "DEV_USER_EMAIL", "AWS_REGION", "AWS_DEFAULT_REGION",
		"SMTP_HOST", "SMTP_PORT", "SMTP_FROM", "SMTP_USERNAME", "SMTP_PASSWORD",
		"PIPELINE_AUTO", "ALLOWED_ORIGINS",
		"DEFACING_SERVICE_URL", "PHI_DETECTION_SERVICE_URL", "QC_SERVICE_URL",
		"BIDS_SERVICE_URL", "CLASSIFICATION_SERVICE_URL", "PROTOCOL_SERVICE_URL", "DIMSE_RECEIVER_URL", "DIMSE_OPERATOR_API_KEY",
		"GCP_PROJECT", "GCS_BUCKET", "S3_BUCKET", "S3_REGION", "S3_ENDPOINT",
	} {
		t.Setenv(key, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnvVars(t)
	// Unset PORT so the default applies (t.Setenv("PORT", "") means PORT is set to "")
	// envOr returns fallback only when value is empty string... wait, let me check.
	// envOr: if v := os.Getenv(key); v != "" { return v }; return fallback
	// t.Setenv("PORT", "") sets it to "" so envOr returns fallback. Good.

	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "postgres://aegis:aegis@localhost:5432/aegis?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "local", cfg.StorageMode)
	assert.Equal(t, "UTC", cfg.AppTimezone)
	assert.False(t, cfg.AuthEnabled)
	assert.Equal(t, "auto", cfg.AuthProvider)
	assert.Equal(t, "ai@aegisimaging.ai", cfg.DevUserEmail)
	assert.Equal(t, "us-east-1", cfg.AWSALBRegion)
	assert.True(t, cfg.PipelineAuto)
}

func TestLoad_EmailDisabledByDefault(t *testing.T) {
	clearEnvVars(t)
	cfg := Load()

	assert.False(t, cfg.EmailEnabled)
	assert.Empty(t, cfg.SMTPHost)
}

func TestLoad_EmailEnabled(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("SMTP_HOST", "mail.example.com")
	t.Setenv("SMTP_PORT", "1025")
	cfg := Load()

	assert.True(t, cfg.EmailEnabled)
	assert.Equal(t, "mail.example.com", cfg.SMTPHost)
	assert.Equal(t, "1025", cfg.SMTPPort)
}

func TestLoad_AuthEnabled(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("AUTH_ENABLED", "true")
	cfg := Load()
	assert.True(t, cfg.AuthEnabled)
}

func TestLoad_AuthDisabledByDefault(t *testing.T) {
	clearEnvVars(t)
	cfg := Load()
	assert.False(t, cfg.AuthEnabled)
}

func TestLoad_PipelineAutoDefault(t *testing.T) {
	clearEnvVars(t)
	cfg := Load()
	assert.True(t, cfg.PipelineAuto, "pipeline auto should default to true")
}

func TestLoad_PipelineAutoDisabled(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("PIPELINE_AUTO", "false")
	cfg := Load()
	assert.False(t, cfg.PipelineAuto)
}

func TestLoad_SidecarURLs(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("DEFACING_SERVICE_URL", "http://localhost:8081")
	t.Setenv("QC_SERVICE_URL", "http://localhost:8083")
	t.Setenv("DIMSE_OPERATOR_API_KEY", "dimse-key")
	cfg := Load()

	assert.Equal(t, "http://localhost:8081", cfg.DefacingServiceURL)
	assert.Equal(t, "http://localhost:8083", cfg.QcServiceURL)
	assert.Empty(t, cfg.PhiDetectionServiceURL)
	assert.Empty(t, cfg.DimseReceiverURL)
	assert.Equal(t, "dimse-key", cfg.DimseOperatorAPIKey)
}

func TestLoad_AppTimezoneOverride(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("APP_TIMEZONE", "America/Chicago")
	cfg := Load()
	assert.Equal(t, "America/Chicago", cfg.AppTimezone)
}

func TestEnvOr(t *testing.T) {
	assert.Equal(t, "fallback", envOr("NONEXISTENT_VAR_12345", "fallback"))

	t.Setenv("TEST_ENVOR_KEY", "custom")
	assert.Equal(t, "custom", envOr("TEST_ENVOR_KEY", "fallback"))
}

func TestLoad_DatabaseURLExplicitOverride(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("DATABASE_URL", "postgres://override:pw@db:5432/custom?sslmode=disable")

	cfg := Load()
	assert.Equal(t, "postgres://override:pw@db:5432/custom?sslmode=disable", cfg.DatabaseURL)
}

func TestLoad_DatabaseURLFromDBEnvVars(t *testing.T) {
	clearEnvVars(t)
	t.Setenv("DB_HOST", "cloudsql.internal")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "aegis_prod")
	t.Setenv("DB_USER", "aegis-api")
	t.Setenv("DB_PASSWORD", "secret@value")

	cfg := Load()
	assert.Equal(t, "postgres://aegis-api:secret%40value@cloudsql.internal:5432/aegis_prod?sslmode=disable", cfg.DatabaseURL)
}
