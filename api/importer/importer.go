// Package importer provides bulk DICOM import into AEGIS.
//
// It scans a directory for DICOM files, groups them by StudyInstanceUID,
// and ingests each study into the database and storage layer. Used by
// both the CLI tool (api/cmd/import) and the HTTP endpoint (POST /api/import/batch).
package importer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	dicom "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/routing"
	"github.com/msenjem/aegis/api/storage"
)

// Options configures a batch import run.
type Options struct {
	Dir                string `json:"dir"`
	ProjectSlug        string `json:"project_slug"`
	InstitutionID      string `json:"institution_id"`
	InstitutionSlug    string `json:"institution_slug"`
	InstitutionAETitle string `json:"institution_ae_title"`
	Source             string `json:"source"`
	DryRun             bool   `json:"dry_run"`
}

// Result reports the outcome of a batch import run.
type Result struct {
	FilesScanned   int      `json:"files_scanned"`
	FilesSkipped   int      `json:"files_skipped"`
	StudiesCreated int      `json:"studies_created"`
	StudiesFailed  int      `json:"studies_failed"`
	Errors         []string `json:"errors,omitempty"`
	StudyIDs       []string `json:"study_ids,omitempty"` // IDs of successfully created studies
}

// ValidationError marks user input errors (bad dir/project/institution/options).
// Handlers should map these to HTTP 400.
type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string {
	return e.msg
}

func validationErrorf(format string, args ...any) error {
	return &ValidationError{msg: fmt.Sprintf(format, args...)}
}

func IsValidationError(err error) bool {
	var vErr *ValidationError
	return errors.As(err, &vErr)
}

// StudyGroup holds DICOM files grouped by StudyInstanceUID.
type StudyGroup struct {
	StudyInstanceUID string
	Modality         string
	BodyPart         string
	StudyDescription string
	SeriesUIDs       map[string]bool
	Files            []string // absolute file paths
}

// Run executes the batch import.
func Run(ctx context.Context, db *sql.DB, store storage.Storage, opts Options) (*Result, error) {
	if err := normalizeImportDir(&opts); err != nil {
		return nil, err
	}
	normalizeProjectSlug(&opts)
	if err := normalizeImportSource(&opts); err != nil {
		return nil, err
	}
	if err := normalizeInstitutionSelectors(&opts); err != nil {
		return nil, err
	}
	if err := validateSourceInstitutionPolicy(&opts); err != nil {
		return nil, err
	}

	// Validate directory exists.
	info, err := os.Stat(opts.Dir)
	if err != nil {
		return nil, validationErrorf("directory %q: %v", opts.Dir, err)
	}
	if !info.IsDir() {
		return nil, validationErrorf("%q is not a directory", opts.Dir)
	}

	// Resolve project.
	project, err := model.GetProjectBySlug(ctx, db, opts.ProjectSlug)
	if err != nil {
		return nil, validationErrorf("project %q not found", opts.ProjectSlug)
	}

	institution, err := resolveImportInstitution(ctx, db, opts)
	if err != nil {
		return nil, err
	}
	if institution != nil {
		if err := validateImportInstitution(institution); err != nil {
			return nil, err
		}
		allowed, err := model.InstitutionCanSendToProject(ctx, db, institution.ID, project.ID)
		if err != nil {
			return nil, fmt.Errorf("validate institution project link: %w", err)
		}
		if !allowed {
			return nil, validationErrorf("institution %q is not linked to project %q as sender/admin", institution.ID, opts.ProjectSlug)
		}
		opts.InstitutionID = institution.ID
	}

	// Scan and group DICOM files.
	log.Printf("aegis-import: scanning %s ...", opts.Dir)
	groups, result, err := scanDirectory(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("scan directory: %w", err)
	}
	log.Printf("aegis-import: found %d DICOM files across %d studies (%d skipped)",
		result.FilesScanned, len(groups), result.FilesSkipped)

	if opts.DryRun {
		for _, g := range sortedGroups(groups) {
			log.Printf("  study %s — %s, %s, %d files, %d series",
				g.StudyInstanceUID, g.Modality, g.BodyPart, len(g.Files), len(g.SeriesUIDs))
		}
		log.Printf("aegis-import: [DRY RUN] would import %d studies (%d files) into project %q",
			len(groups), result.FilesScanned-result.FilesSkipped, opts.ProjectSlug)
		return result, nil
	}

	// Import each study.
	sorted := sortedGroups(groups)
	for i, g := range sorted {
		log.Printf("aegis-import: [%d/%d] importing study %s (%d files)",
			i+1, len(sorted), g.StudyInstanceUID, len(g.Files))
		studyID, err := importStudy(ctx, db, store, project, g, opts)
		if err != nil {
			result.StudiesFailed++
			errMsg := fmt.Sprintf("study %s: %v", g.StudyInstanceUID, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("aegis-import: ERROR %s", errMsg)
			continue
		}
		result.StudiesCreated++
		result.StudyIDs = append(result.StudyIDs, studyID)
	}

	log.Printf("aegis-import: complete — %d studies created, %d failed, %d files imported",
		result.StudiesCreated, result.StudiesFailed, result.FilesScanned-result.FilesSkipped)
	return result, nil
}

func validateImportInstitution(inst *model.Institution) error {
	if inst == nil {
		return validationErrorf("institution is required")
	}
	if !inst.Enabled {
		return validationErrorf("institution %q is disabled", inst.ID)
	}
	if inst.Type != "sender" && inst.Type != "both" {
		return validationErrorf("institution %q type must be sender or both", inst.ID)
	}
	return nil
}

func normalizeInstitutionSelectors(opts *Options) error {
	opts.InstitutionID = strings.TrimSpace(opts.InstitutionID)
	opts.InstitutionSlug = strings.ToLower(strings.TrimSpace(opts.InstitutionSlug))
	opts.InstitutionAETitle = strings.TrimSpace(opts.InstitutionAETitle)
	if opts.InstitutionAETitle != "" {
		return validationErrorf("institution_ae_title is no longer supported for batch import; use institution_id or institution_slug")
	}
	if opts.InstitutionID != "" && opts.InstitutionSlug != "" {
		return validationErrorf("provide only one of institution_id or institution_slug")
	}
	return nil
}

func normalizeImportSource(opts *Options) error {
	opts.Source = strings.ToLower(strings.TrimSpace(opts.Source))
	if opts.Source == "" {
		opts.Source = "internal"
	}
	if opts.Source != "internal" && opts.Source != "external" {
		return validationErrorf("source must be internal or external")
	}
	return nil
}

func normalizeImportDir(opts *Options) error {
	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Dir == "" {
		return validationErrorf("dir is required")
	}
	opts.Dir = filepath.Clean(opts.Dir)
	return nil
}

func validateSourceInstitutionPolicy(opts *Options) error {
	if opts.Source != "external" {
		return nil
	}
	if opts.InstitutionID == "" && opts.InstitutionSlug == "" {
		return validationErrorf("institution_id or institution_slug required when source is external")
	}
	return nil
}

func normalizeProjectSlug(opts *Options) {
	opts.ProjectSlug = strings.ToLower(strings.TrimSpace(opts.ProjectSlug))
	if opts.ProjectSlug == "" {
		opts.ProjectSlug = "default"
	}
}

func resolveImportInstitution(ctx context.Context, db *sql.DB, opts Options) (*model.Institution, error) {
	if opts.InstitutionID == "" && opts.InstitutionSlug == "" {
		return nil, nil
	}
	var (
		institution *model.Institution
		err         error
	)
	if opts.InstitutionID != "" {
		institution, err = model.GetInstitutionByID(ctx, db, opts.InstitutionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, validationErrorf("institution %q not found", opts.InstitutionID)
			}
			return nil, fmt.Errorf("lookup institution: %w", err)
		}
	} else if opts.InstitutionSlug != "" {
		institution, err = model.GetInstitutionBySlug(ctx, db, opts.InstitutionSlug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, validationErrorf("institution slug %q not found", opts.InstitutionSlug)
			}
			return nil, fmt.Errorf("lookup institution by slug: %w", err)
		}
	}

	return institution, nil
}

// scanDirectory walks dir recursively, finds DICOM files, parses headers,
// and returns files grouped by StudyInstanceUID.
func scanDirectory(dir string) (map[string]*StudyGroup, *Result, error) {
	groups := make(map[string]*StudyGroup)
	result := &Result{}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}

		// Accept .dcm files (case-insensitive).
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".dcm" {
			return nil
		}

		result.FilesScanned++

		studyUID, modality, bodyPart, studyDesc, seriesUID, parseErr := parseDICOMHeaders(path)
		if parseErr != nil {
			result.FilesSkipped++
			result.Errors = append(result.Errors, fmt.Sprintf("parse %s: %v", filepath.Base(path), parseErr))
			return nil
		}

		g, ok := groups[studyUID]
		if !ok {
			g = &StudyGroup{
				StudyInstanceUID: studyUID,
				Modality:         modality,
				BodyPart:         strings.ToUpper(bodyPart),
				StudyDescription: studyDesc,
				SeriesUIDs:       make(map[string]bool),
			}
			groups[studyUID] = g
		}
		g.Files = append(g.Files, path)
		if seriesUID != "" {
			g.SeriesUIDs[seriesUID] = true
		}

		return nil
	})

	return groups, result, err
}

// parseDICOMHeaders extracts key tags from a DICOM file without loading pixel data.
func parseDICOMHeaders(path string) (studyUID, modality, bodyPart, studyDesc, seriesUID string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("stat: %w", err)
	}

	dataset, err := dicom.Parse(f, info.Size(), nil, dicom.SkipPixelData())
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("parse DICOM: %w", err)
	}

	studyUID = getStringTag(dataset, tag.StudyInstanceUID)
	modality = getStringTag(dataset, tag.Modality)
	bodyPart = getStringTag(dataset, tag.BodyPartExamined)
	studyDesc = getStringTag(dataset, tag.StudyDescription)
	seriesUID = getStringTag(dataset, tag.SeriesInstanceUID)

	if studyUID == "" {
		return "", "", "", "", "", fmt.Errorf("missing StudyInstanceUID")
	}
	return studyUID, modality, bodyPart, studyDesc, seriesUID, nil
}

// getStringTag extracts a string value from a DICOM dataset element.
func getStringTag(dataset dicom.Dataset, t tag.Tag) string {
	el, err := dataset.FindElementByTag(t)
	if err != nil {
		return ""
	}
	if el.Value == nil {
		return ""
	}
	strs, ok := el.Value.GetValue().([]string)
	if !ok || len(strs) == 0 {
		return ""
	}
	return strings.TrimSpace(strs[0])
}

// importStudy processes a single study group: creates session, copies files,
// creates study record, evaluates routing rules, and creates audit entry.
func importStudy(ctx context.Context, db *sql.DB, store storage.Storage, project *model.Project, g *StudyGroup, opts Options) (string, error) {
	// Create upload session for traceability.
	session, err := model.CreateUploadSession(ctx, db, project.ID, len(g.Files), "batch-import", "batch-import", "")
	if err != nil {
		return "", fmt.Errorf("create upload session: %w", err)
	}

	// Sort files for deterministic ordering.
	sort.Strings(g.Files)

	// Copy files to raw DICOM store.
	for i, filePath := range g.Files {
		key := fmt.Sprintf("dicom/raw/%s/%d.dcm", g.StudyInstanceUID, i)
		f, err := os.Open(filePath)
		if err != nil {
			model.UpdateUploadSessionFailed(ctx, db, session.ID, err.Error())
			return "", fmt.Errorf("open source file %s: %w", filepath.Base(filePath), err)
		}
		if err := store.Store(ctx, key, f); err != nil {
			f.Close()
			model.UpdateUploadSessionFailed(ctx, db, session.ID, err.Error())
			return "", fmt.Errorf("store file %d: %w", i, err)
		}
		f.Close()
	}

	// Determine if defacing is needed (same logic as ingestFiles).
	bodyPart := strings.ToUpper(g.BodyPart)
	defacingRequired := bodyPart == "HEAD" || bodyPart == "BRAIN"

	// Build study record.
	study := &model.Study{
		ProjectID:        project.ID,
		UploadSessionID:  &session.ID,
		StudyInstanceUID: g.StudyInstanceUID,
		Modality:         g.Modality,
		BodyPart:         g.BodyPart,
		StudyDescription: g.StudyDescription,
		SeriesCount:      len(g.SeriesUIDs),
		InstanceCount:    len(g.Files),
		Status:           "received",
		DefacingRequired: defacingRequired,
		DicomStore:       "raw",
		Source:           opts.Source,
	}

	// Set institution if provided.
	if opts.InstitutionID != "" {
		study.InstitutionID = &opts.InstitutionID
	}

	if err := model.CreateStudy(ctx, db, study); err != nil {
		model.UpdateUploadSessionFailed(ctx, db, session.ID, err.Error())
		return "", fmt.Errorf("create study: %w", err)
	}

	// Evaluate routing rules — may set defacing_required, phi_scan, auto_approve, etc.
	routing.EvaluateRules(ctx, db, store, study)

	// Finalize upload session.
	model.UpdateUploadSessionComplete(ctx, db, session.ID, g.StudyInstanceUID, g.Modality, g.BodyPart)

	// Audit entry.
	model.CreateAuditEntry(ctx, db, "import.batch", "batch-import", "study", study.ID, "", map[string]any{
		"study_uid":      g.StudyInstanceUID,
		"modality":       g.Modality,
		"body_part":      g.BodyPart,
		"instance_count": len(g.Files),
		"series_count":   len(g.SeriesUIDs),
		"source_dir":     opts.Dir,
		"project":        opts.ProjectSlug,
	})

	return study.ID, nil
}

// sortedGroups returns study groups sorted by StudyInstanceUID for deterministic output.
func sortedGroups(groups map[string]*StudyGroup) []*StudyGroup {
	sorted := make([]*StudyGroup, 0, len(groups))
	for _, g := range groups {
		sorted = append(sorted, g)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StudyInstanceUID < sorted[j].StudyInstanceUID
	})
	return sorted
}
