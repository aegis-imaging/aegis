package model

import (
	"context"
	"database/sql"
	"time"
)

type ProjectMilestone struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Reached     bool       `json:"reached"`
	ReachedAt   *time.Time `json:"reached_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func CreateProjectMilestone(ctx context.Context, db *sql.DB, m *ProjectMilestone) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO project_milestones (project_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		m.ProjectID, m.Title, m.Description).
		Scan(&m.ID, &m.CreatedAt)
}

func ListProjectMilestones(ctx context.Context, db *sql.DB, projectID string) ([]ProjectMilestone, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, title, description, reached, reached_at, created_at
		FROM project_milestones
		WHERE project_id = $1
		ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectMilestone
	for rows.Next() {
		var m ProjectMilestone
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Title, &m.Description, &m.Reached, &m.ReachedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func UpdateProjectMilestoneReached(ctx context.Context, db *sql.DB, id string, reached bool) error {
	if reached {
		_, err := db.ExecContext(ctx, `
			UPDATE project_milestones SET reached = true, reached_at = now() WHERE id = $1`, id)
		return err
	}
	_, err := db.ExecContext(ctx, `
		UPDATE project_milestones SET reached = false, reached_at = NULL WHERE id = $1`, id)
	return err
}

func DeleteProjectMilestone(ctx context.Context, db *sql.DB, id, projectID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM project_milestones WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}
