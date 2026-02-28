package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// AnonProfile defines a named set of DICOM tag retention overrides for a project.
// Tags listed in RetainedTags are kept as-is instead of being stripped by the
// PS3.15 Annex E Basic Profile de-identification applied client-side.
type AnonProfile struct {
	ID              string          `json:"id"`
	ProjectID       string          `json:"project_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	RetainedTags    json.RawMessage `json:"retained_tags"`     // JSON array of DICOM keyword strings
	KeepPrivateTags bool            `json:"keep_private_tags"` // Retain vendor private tags (odd group numbers)
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

const anonProfileColumns = `
	id, project_id, name, description, retained_tags, keep_private_tags, enabled, created_at, updated_at`

func scanAnonProfile(row scannable, p *AnonProfile) error {
	return row.Scan(
		&p.ID, &p.ProjectID, &p.Name, &p.Description,
		&p.RetainedTags, &p.KeepPrivateTags, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
}

func CreateAnonProfile(ctx context.Context, db *sql.DB, p *AnonProfile) error {
	if p.RetainedTags == nil {
		p.RetainedTags = json.RawMessage("[]")
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO anon_profiles (project_id, name, description, retained_tags, keep_private_tags, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		p.ProjectID, p.Name, p.Description, p.RetainedTags, p.KeepPrivateTags, p.Enabled,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func GetAnonProfileByID(ctx context.Context, db *sql.DB, id string) (*AnonProfile, error) {
	var p AnonProfile
	err := scanAnonProfile(db.QueryRowContext(ctx,
		`SELECT`+anonProfileColumns+` FROM anon_profiles WHERE id = $1`, id), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func ListAnonProfilesByProject(ctx context.Context, db *sql.DB, projectID string) ([]AnonProfile, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+anonProfileColumns+` FROM anon_profiles WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AnonProfile
	for rows.Next() {
		var p AnonProfile
		if err := scanAnonProfile(rows, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func UpdateAnonProfile(ctx context.Context, db *sql.DB, p *AnonProfile) error {
	if p.RetainedTags == nil {
		p.RetainedTags = json.RawMessage("[]")
	}
	_, err := db.ExecContext(ctx, `
		UPDATE anon_profiles SET
			name=$1, description=$2, retained_tags=$3, keep_private_tags=$4, enabled=$5, updated_at=now()
		WHERE id=$6`,
		p.Name, p.Description, p.RetainedTags, p.KeepPrivateTags, p.Enabled, p.ID)
	return err
}

func DeleteAnonProfile(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM anon_profiles WHERE id = $1`, id)
	return err
}

// SetProjectDefaultAnonProfile sets the default anonymization profile for a project.
// Pass an empty string to clear the default.
func SetProjectDefaultAnonProfile(ctx context.Context, db *sql.DB, projectID, profileID string) error {
	var err error
	if profileID == "" {
		_, err = db.ExecContext(ctx,
			`UPDATE projects SET default_anon_profile_id = NULL, updated_at = now() WHERE id = $1`,
			projectID)
	} else {
		_, err = db.ExecContext(ctx,
			`UPDATE projects SET default_anon_profile_id = $1, updated_at = now() WHERE id = $2`,
			profileID, projectID)
	}
	return err
}

// GetProjectDefaultAnonProfile returns the default anonymization profile for a project,
// or nil if none is set.
func GetProjectDefaultAnonProfile(ctx context.Context, db *sql.DB, projectSlug string) (*AnonProfile, error) {
	var p AnonProfile
	err := scanAnonProfile(db.QueryRowContext(ctx, `
		SELECT ap.id, ap.project_id, ap.name, ap.description, ap.retained_tags, ap.keep_private_tags, ap.enabled,
		       ap.created_at, ap.updated_at
		FROM anon_profiles ap
		JOIN projects pr ON pr.default_anon_profile_id = ap.id
		WHERE pr.slug = $1`, projectSlug), &p)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
