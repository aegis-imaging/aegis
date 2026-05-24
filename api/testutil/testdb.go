package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/aegis-imaging/aegis/api/migrate"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Shared container state — one PostgreSQL container per test process (package).
// Ryuk (testcontainers' reaper) cleans up the container when the process exits.
var (
	sharedOnce    sync.Once
	sharedConnStr string
	sharedErr     error
)

func initSharedContainer() {
	ctx := context.Background()

	if _, err := exec.LookPath("docker"); err != nil {
		sharedErr = fmt.Errorf("docker not found in PATH")
		return
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		sharedErr = fmt.Errorf("docker daemon not running")
		return
	}

	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("aegis_test"),
		postgres.WithUsername("aegis"),
		postgres.WithPassword("aegis"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		sharedErr = fmt.Errorf("start postgres container: %w", err)
		return
	}
	// No t.Cleanup — Ryuk handles container termination when the process exits.
	_ = container

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		sharedErr = fmt.Errorf("connection string: %w", err)
		return
	}
	sharedConnStr = connStr

	// Run migrations once on the shared container.
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		sharedErr = fmt.Errorf("open db for migrations: %w", err)
		return
	}
	defer db.Close()

	if err := migrate.Run(db); err != nil {
		sharedErr = fmt.Errorf("run migrations: %w", err)
		return
	}
}

// TestDB returns a *sql.DB connected to a shared PostgreSQL container.
// The container is created once per test process (package) and reused across
// all tests in that package. Each call truncates all data tables and re-seeds
// the default project for test isolation.
//
// This reduces container startups from ~70 (one per test) to ~5 (one per package),
// cutting CI time from 10+ minutes to under 2 minutes.
func TestDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test (requires Docker)")
	}

	sharedOnce.Do(initSharedContainer)
	if sharedErr != nil {
		t.Skipf("skipping integration test: %v", sharedErr)
	}

	db, err := sql.Open("pgx", sharedConnStr)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Truncate all data tables for test isolation, then re-seed.
	cleanAndSeed(t, db)

	return db
}

// cleanAndSeed truncates all user data tables (preserving schema and goose
// migration state) and re-inserts the seed data from migration 001.
func cleanAndSeed(t *testing.T, db *sql.DB) {
	t.Helper()

	// Truncate in one statement with CASCADE to handle FK dependencies.
	// goose_db_version is excluded — it tracks applied migrations.
	_, err := db.ExecContext(context.Background(), `
		TRUNCATE
			routing_log,
			export_downloads,
			export_shares,
			digest_subscriptions,
			protocol_templates,
			anon_profiles,
			institution_projects,
			routing_rules,
			destinations,
			studies,
			upload_sessions,
			admin_users,
			institutions,
			audit_trail,
			desktop_installer_invites,
			desktop_installers,
			api_keys,
			invite_codes,
			projects,
			tenants
		CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	// Re-seed the default project (migration 001 seed data).
	_, err = db.ExecContext(context.Background(), `
		INSERT INTO projects (name, slug, description)
		VALUES ('Default', 'default', 'Default project for uploads')
	`)
	if err != nil {
		t.Fatalf("re-seed default project: %v", err)
	}
}
