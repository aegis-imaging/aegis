package model

import (
	"context"
	"database/sql"
	"time"
)

type Project struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Slug                 string    `json:"slug"`
	Description          string    `json:"description"`
	DefaultAnonProfileID *string   `json:"default_anon_profile_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

const projectColumns = `id, name, slug, description, default_anon_profile_id, created_at, updated_at`

func scanProject(row scannable, p *Project) error {
	return row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.DefaultAnonProfileID, &p.CreatedAt, &p.UpdatedAt)
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
	err := db.QueryRowContext(ctx, `
		UPDATE projects SET name=$1, slug=$2, description=$3, updated_at=now()
		WHERE id=$4
		RETURNING `+projectColumns,
		name, slug, description, id).
		Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.DefaultAnonProfileID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreateProject(ctx context.Context, db *sql.DB, name, slug, description string) (*Project, error) {
	var p Project
	err := db.QueryRowContext(ctx, `
		INSERT INTO projects (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, description, created_at, updated_at`,
		name, slug, description).
		Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
