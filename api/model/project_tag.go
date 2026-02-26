package model

import (
	"context"
	"database/sql"
	"time"
)

type ProjectTag struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Tag       string    `json:"tag"`
	CreatedAt time.Time `json:"created_at"`
}

func AddProjectTag(ctx context.Context, db *sql.DB, projectID, tag string) (*ProjectTag, error) {
	var t ProjectTag
	err := db.QueryRowContext(ctx, `
		INSERT INTO project_tags (project_id, tag)
		VALUES ($1, $2)
		ON CONFLICT (project_id, tag) DO UPDATE SET project_id = project_tags.project_id
		RETURNING id, project_id, tag, created_at`,
		projectID, tag).
		Scan(&t.ID, &t.ProjectID, &t.Tag, &t.CreatedAt)
	return &t, err
}

func ListProjectTags(ctx context.Context, db *sql.DB, projectID string) ([]ProjectTag, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, tag, created_at
		FROM project_tags
		WHERE project_id = $1
		ORDER BY tag ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectTag
	for rows.Next() {
		var t ProjectTag
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Tag, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func DeleteProjectTag(ctx context.Context, db *sql.DB, id, projectID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM project_tags WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}
