package model

import (
	"context"
	"database/sql"
	"time"
)

type ProjectACLEntry struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	UserEmail  string    `json:"user_email"`
	Permission string    `json:"permission"`
	GrantedBy  string    `json:"granted_by"`
	CreatedAt  time.Time `json:"created_at"`
}

func UpsertProjectACL(ctx context.Context, db *sql.DB, e *ProjectACLEntry) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO project_acl (project_id, user_email, permission, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (project_id, user_email) DO UPDATE SET permission = $3, granted_by = $4
		RETURNING id, created_at`,
		e.ProjectID, e.UserEmail, e.Permission, e.GrantedBy).
		Scan(&e.ID, &e.CreatedAt)
}

func ListProjectACL(ctx context.Context, db *sql.DB, projectID string) ([]ProjectACLEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, user_email, permission, granted_by, created_at
		FROM project_acl
		WHERE project_id = $1
		ORDER BY user_email ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectACLEntry
	for rows.Next() {
		var e ProjectACLEntry
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.UserEmail, &e.Permission, &e.GrantedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func DeleteProjectACL(ctx context.Context, db *sql.DB, id, projectID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM project_acl WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}
