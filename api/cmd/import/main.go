// aegis-import is a CLI tool for importing DICOM files from a local directory
// into the AEGIS platform. It scans for .dcm files, groups them by study,
// and ingests them into the database and storage layer.
//
// Usage:
//
//	aegis-import --dir /path/to/dicom [--project slug] [--institution uuid] [--institution-slug slug] [--source internal|external] [--dry-run]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/importer"
	"github.com/msenjem/aegis/api/migrate"
	"github.com/msenjem/aegis/api/storage"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	dir := flag.String("dir", "", "Directory containing DICOM files to import (required)")
	project := flag.String("project", "default", "Project slug to import into")
	institution := flag.String("institution", "", "Institution UUID (optional)")
	institutionSlug := flag.String("institution-slug", "", "Institution slug (optional)")
	source := flag.String("source", "internal", "Study source: internal or external")
	dryRun := flag.Bool("dry-run", false, "Scan and report without importing")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "error: --dir is required")
		flag.Usage()
		os.Exit(1)
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

	if err := migrate.Run(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}

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

	opts := importer.Options{
		Dir:                *dir,
		ProjectSlug:        *project,
		InstitutionID:      *institution,
		InstitutionSlug:    *institutionSlug,
		Source:             *source,
		DryRun:             *dryRun,
	}

	result, err := importer.Run(context.Background(), db, store, opts)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	// Print summary.
	fmt.Printf("\n=== Import Summary ===\n")
	fmt.Printf("Files scanned:   %d\n", result.FilesScanned)
	fmt.Printf("Files skipped:   %d\n", result.FilesSkipped)
	fmt.Printf("Studies created: %d\n", result.StudiesCreated)
	fmt.Printf("Studies failed:  %d\n", result.StudiesFailed)
	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
}

func applyDBSessionTimezone(ctx context.Context, db *sql.DB, timezone string) error {
	var applied string
	return db.QueryRowContext(ctx, `SELECT set_config('TimeZone', $1, false)`, timezone).Scan(&applied)
}
