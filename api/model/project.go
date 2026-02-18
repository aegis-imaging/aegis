package model

import (
	"context"
	"database/sql"
	"time"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ListProjects(ctx context.Context, db *sql.DB) ([]Project, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func GetProjectBySlug(ctx context.Context, db *sql.DB, slug string) (*Project, error) {
	var p Project
	err := db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM projects WHERE slug = $1`, slug).
		Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetProjectByID(ctx context.Context, db *sql.DB, id string) (*Project, error) {
	var p Project
	err := db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM projects WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
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
