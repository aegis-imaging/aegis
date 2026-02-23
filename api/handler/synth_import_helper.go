package handler

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aegis-imaging/aegis/api/importer"
	"github.com/aegis-imaging/aegis/api/storage"
)

// importSynthStudy imports synthetic DICOM files for the given studyUID into
// the database under projectSlug.
//
// Files are retrieved via the storage interface rather than read from the GCS
// FUSE mount. GCS FUSE can report stale file sizes (0) for objects written by
// another container in the same Cloud Run service group; reading through the
// GCS client API always returns correct data. Files are staged in a local
// temp dir before the normal batch importer runs.
func importSynthStudy(ctx context.Context, db *sql.DB, store storage.Storage, studyUID, projectSlug string) (*importer.Result, error) {
	// synth-service writes to "synth/{studyUID}/" in the shared storage bucket.
	prefix := "synth/" + studyUID

	keys, err := store.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("list synth files at %q: %w", prefix, err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no synth files found at prefix %q", prefix)
	}

	// Stage files in a local temp dir so the importer can read real local files.
	tmpDir, err := os.MkdirTemp("", "aegis-synth-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			log.Printf("importSynthStudy: cleanup %s: %v", tmpDir, removeErr)
		}
	}()

	for _, key := range keys {
		rc, err := store.Retrieve(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("retrieve %s: %w", key, err)
		}
		dest := filepath.Join(tmpDir, filepath.Base(key))
		f, createErr := os.Create(dest)
		if createErr != nil {
			rc.Close()
			return nil, fmt.Errorf("create temp file: %w", createErr)
		}
		_, copyErr := io.Copy(f, rc)
		f.Close()
		rc.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("write %s: %w", filepath.Base(key), copyErr)
		}
	}

	log.Printf("importSynthStudy: staged %d files for study %s in %s", len(keys), studyUID, tmpDir)

	return importer.Run(ctx, db, store, importer.Options{
		Dir:         tmpDir,
		ProjectSlug: projectSlug,
		Source:      "internal",
	})
}
