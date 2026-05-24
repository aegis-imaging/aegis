package model

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

// SubjectAggregate enriches a subject row with the data the XNAT-style
// listing needs in a single round trip: study count, latest study date,
// distinct modalities, and optional research demographics.
//
// `SubjectID` is the canonical subject identifier — populated at ingest
// from the DICOM-derived pseudonymized PatientID, then optionally
// re-keyed by researchers to merge patients across studies.
type SubjectAggregate struct {
	SubjectID       string               `json:"subject_id"`
	ProjectID       string               `json:"project_id"`
	StudyCount      int                  `json:"study_count"`
	LatestStudyDate *string              `json:"latest_study_date,omitempty"`
	Modalities      []string             `json:"modalities,omitempty"`
	Demographics    *SubjectDemographics `json:"demographics,omitempty"`
}

// ListProjectSubjects returns one row per distinct subject_id in the given
// project, enriched with study count, most recent study_date, and the set
// of modalities seen across that subject's studies. Demographics are
// looked up separately by the handler so this stays cheap when only the
// listing is needed.
//
// If institutionID is non-empty (site-scoped researchers), the aggregate
// is computed only over studies in that institution.
func ListProjectSubjects(ctx context.Context, db *sql.DB, projectID, institutionID string) ([]SubjectAggregate, error) {
	q := `
		SELECT
			subject_id,
			project_id,
			count(*) AS study_count,
			max(study_date) AS latest_study_date,
			COALESCE(
				array_agg(DISTINCT modality) FILTER (WHERE modality IS NOT NULL AND modality <> ''),
				ARRAY[]::text[]
			) AS modalities
		FROM studies
		WHERE project_id = $1
		  AND subject_id IS NOT NULL
		  AND subject_id <> ''
		  AND deleted_at IS NULL`
	args := []any{projectID}
	if institutionID != "" {
		q += ` AND institution_id = $2`
		args = append(args, institutionID)
	}
	q += ` GROUP BY subject_id, project_id ORDER BY subject_id`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SubjectAggregate
	for rows.Next() {
		var a SubjectAggregate
		var mods pq.StringArray
		if err := rows.Scan(
			&a.SubjectID,
			&a.ProjectID,
			&a.StudyCount,
			&a.LatestStudyDate,
			&mods,
		); err != nil {
			return nil, err
		}
		a.Modalities = []string(mods)
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetProjectSubject returns a single subject aggregate (study count,
// latest study date, modalities) for a (project, subject_id) pair.
// Returns sql.ErrNoRows if no matching studies exist.
func GetProjectSubject(ctx context.Context, db *sql.DB, projectID, subjectID, institutionID string) (*SubjectAggregate, error) {
	q := `
		SELECT
			subject_id,
			project_id,
			count(*) AS study_count,
			max(study_date) AS latest_study_date,
			COALESCE(
				array_agg(DISTINCT modality) FILTER (WHERE modality IS NOT NULL AND modality <> ''),
				ARRAY[]::text[]
			) AS modalities
		FROM studies
		WHERE project_id = $1
		  AND subject_id = $2
		  AND deleted_at IS NULL`
	args := []any{projectID, subjectID}
	if institutionID != "" {
		q += ` AND institution_id = $3`
		args = append(args, institutionID)
	}
	q += ` GROUP BY subject_id, project_id`

	var a SubjectAggregate
	var mods pq.StringArray
	err := db.QueryRowContext(ctx, q, args...).
		Scan(
			&a.SubjectID,
			&a.ProjectID,
			&a.StudyCount,
			&a.LatestStudyDate,
			&mods,
		)
	if err != nil {
		return nil, err
	}
	a.Modalities = []string(mods)
	return &a, nil
}
