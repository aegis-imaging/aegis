package model

import (
	"context"
	"database/sql"
	"time"
)

type Project struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Slug                  string    `json:"slug"`
	Description           string    `json:"description"`
	DefaultAnonProfileID  *string   `json:"default_anon_profile_id,omitempty"`
	RetentionDays         *int      `json:"retention_days,omitempty"`          // nil = keep indefinitely
	StuckThresholdMinutes *int      `json:"stuck_threshold_minutes,omitempty"` // nil = use request default (60)
	Archived              bool      `json:"archived"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

const projectColumns = `id, name, slug, description, default_anon_profile_id, retention_days, stuck_threshold_minutes, archived, created_at, updated_at`

func scanProject(row scannable, p *Project) error {
	return row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.DefaultAnonProfileID, &p.RetentionDays, &p.StuckThresholdMinutes, &p.Archived, &p.CreatedAt, &p.UpdatedAt)
}

func ListProjects(ctx context.Context, db *sql.DB) ([]Project, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+projectColumns+` FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := scanProject(rows, &p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func GetProjectBySlug(ctx context.Context, db *sql.DB, slug string) (*Project, error) {
	var p Project
	err := scanProject(db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE slug = $1`, slug), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetProjectByID(ctx context.Context, db *sql.DB, id string) (*Project, error) {
	var p Project
	err := scanProject(db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE id = $1`, id), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdateProject(ctx context.Context, db *sql.DB, id, name, slug, description string) (*Project, error) {
	var p Project
	err := scanProject(db.QueryRowContext(ctx, `
		UPDATE projects SET name=$1, slug=$2, description=$3, updated_at=now()
		WHERE id=$4
		RETURNING `+projectColumns,
		name, slug, description, id), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateProjectSLAThreshold sets or clears (nil) the per-project stuck threshold.
func UpdateProjectSLAThreshold(ctx context.Context, db *sql.DB, projectID string, minutes *int) error {
	_, err := db.ExecContext(ctx, `
		UPDATE projects SET stuck_threshold_minutes = $1, updated_at = now() WHERE id = $2`,
		minutes, projectID)
	return err
}

// UpdateProjectRetentionDays sets or clears (nil) the retention policy for a project.
func UpdateProjectRetentionDays(ctx context.Context, db *sql.DB, projectID string, days *int) error {
	_, err := db.ExecContext(ctx, `
		UPDATE projects SET retention_days = $1, updated_at = now() WHERE id = $2`,
		days, projectID)
	return err
}

func CreateProject(ctx context.Context, db *sql.DB, name, slug, description string) (*Project, error) {
	var p Project
	err := scanProject(db.QueryRowContext(ctx, `
		INSERT INTO projects (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING `+projectColumns,
		name, slug, description), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ProjectsWithRetentionPolicy returns all projects that have a non-null retention_days.
func ProjectsWithRetentionPolicy(ctx context.Context, db *sql.DB) ([]Project, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE retention_days IS NOT NULL ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := scanProject(rows, &p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// ArchiveProject sets archived=true for a project.
func ArchiveProject(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE projects SET archived = true, updated_at = now() WHERE id = $1`, id)
	return err
}

// RestoreProject sets archived=false for a project.
func RestoreProject(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE projects SET archived = false, updated_at = now() WHERE id = $1`, id)
	return err
}

// ExpireStudiesByRetention soft-expires approved studies in a project that were created
// more than retentionDays ago, marking them status="expired".
// Returns the number of studies updated.
func ExpireStudiesByRetention(ctx context.Context, db *sql.DB, projectID string, retentionDays int) (int64, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE studies
		SET status = 'expired', updated_at = now()
		WHERE project_id = $1
		  AND status = 'approved'
		  AND created_at < now() - ($2 * INTERVAL '1 day')`,
		projectID, retentionDays)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
