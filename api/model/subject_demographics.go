package model

import (
	"context"
	"database/sql"
	"time"
)

// SubjectDemographics stores de-identified research metadata for a subject.
type SubjectDemographics struct {
	ID             string    `json:"id"`
	SubjectID      string    `json:"subject_id"`
	ProjectID      string    `json:"project_id"`
	Sex            string    `json:"sex"`
	AgeAtScan      *int      `json:"age_at_scan,omitempty"`
	Diagnosis      string    `json:"diagnosis"`
	EducationYears *int16    `json:"education_years,omitempty"`
	MMSEScore      *int16    `json:"mmse_score,omitempty"`
	MoCAScore      *int16    `json:"moca_score,omitempty"`
	CDRGlobal      *float64  `json:"cdr_global,omitempty"`
	APOEGenotype   string    `json:"apoe_genotype"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UpsertSubjectDemographics creates or updates demographics for a subject+project.
func UpsertSubjectDemographics(ctx context.Context, db *sql.DB, d *SubjectDemographics) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO subject_demographics
			(subject_id, project_id, sex, age_at_scan, diagnosis, education_years,
			 mmse_score, moca_score, cdr_global, apoe_genotype, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (subject_id, project_id) DO UPDATE SET
			sex              = EXCLUDED.sex,
			age_at_scan      = EXCLUDED.age_at_scan,
			diagnosis        = EXCLUDED.diagnosis,
			education_years  = EXCLUDED.education_years,
			mmse_score       = EXCLUDED.mmse_score,
			moca_score       = EXCLUDED.moca_score,
			cdr_global       = EXCLUDED.cdr_global,
			apoe_genotype    = EXCLUDED.apoe_genotype,
			notes            = EXCLUDED.notes,
			updated_at       = now()
		RETURNING id, created_at, updated_at`,
		d.SubjectID, d.ProjectID, d.Sex, d.AgeAtScan, d.Diagnosis, d.EducationYears,
		d.MMSEScore, d.MoCAScore, d.CDRGlobal, d.APOEGenotype, d.Notes).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

// GetSubjectDemographics returns demographics for a subject in a project.
func GetSubjectDemographics(ctx context.Context, db *sql.DB, subjectID, projectID string) (*SubjectDemographics, error) {
	var d SubjectDemographics
	err := db.QueryRowContext(ctx, `
		SELECT id, subject_id, project_id, sex, age_at_scan, diagnosis, education_years,
			mmse_score, moca_score, cdr_global, apoe_genotype, notes, created_at, updated_at
		FROM subject_demographics
		WHERE subject_id = $1 AND project_id = $2`, subjectID, projectID).
		Scan(&d.ID, &d.SubjectID, &d.ProjectID, &d.Sex, &d.AgeAtScan, &d.Diagnosis,
			&d.EducationYears, &d.MMSEScore, &d.MoCAScore, &d.CDRGlobal,
			&d.APOEGenotype, &d.Notes, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListProjectDemographics returns all subject demographics for a project.
func ListProjectDemographics(ctx context.Context, db *sql.DB, projectID string) ([]SubjectDemographics, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, subject_id, project_id, sex, age_at_scan, diagnosis, education_years,
			mmse_score, moca_score, cdr_global, apoe_genotype, notes, created_at, updated_at
		FROM subject_demographics
		WHERE project_id = $1
		ORDER BY subject_id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SubjectDemographics
	for rows.Next() {
		var d SubjectDemographics
		if err := rows.Scan(&d.ID, &d.SubjectID, &d.ProjectID, &d.Sex, &d.AgeAtScan,
			&d.Diagnosis, &d.EducationYears, &d.MMSEScore, &d.MoCAScore,
			&d.CDRGlobal, &d.APOEGenotype, &d.Notes, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
