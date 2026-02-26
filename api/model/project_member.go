package model

import (
	"context"
	"database/sql"
	"time"
)

// ProjectMember links an admin user to a project with a specific role.
// This is the foundation of project-level access control.
//
// Coordinating Center roles (InstitutionID == nil):
//   - owner       – Principal Investigator; full project control
//   - coordinator – Data manager; write access to studies
//   - reviewer    – QC reviewer; read all + approve/reject
//
// Site roles (InstitutionID != nil — scoped to one participating site):
//   - site_coordinator – Site research coordinator; upload + view own site's data
//   - site_viewer      – Site monitor; read-only for own site's data
//
// Study visibility rule:
//   InstitutionID == nil  → user sees ALL studies in the project
//   InstitutionID != nil  → user sees ONLY studies where study.institution_id matches
type ProjectMember struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	AdminUserID   string     `json:"admin_user_id"`
	Role          string     `json:"role"` // owner|coordinator|reviewer|site_coordinator|site_viewer
	InstitutionID *string    `json:"institution_id,omitempty"`
	Notes         string     `json:"notes"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	// Populated by joins for list endpoints
	UserEmail       string `json:"user_email,omitempty"`
	UserName        string `json:"user_name,omitempty"`
	InstitutionName string `json:"institution_name,omitempty"`
}

// IsSiteRole returns true for site-scoped roles.
func (m *ProjectMember) IsSiteRole() bool {
	return m.Role == "site_coordinator" || m.Role == "site_viewer"
}

// CanWrite returns true if the role permits study mutations (approve/reject/flag/notes).
func (m *ProjectMember) CanWrite() bool {
	return m.Role == "owner" || m.Role == "coordinator" || m.Role == "reviewer" || m.Role == "site_coordinator"
}

// CanApprove returns true if the role permits study approval/rejection.
func (m *ProjectMember) CanApprove() bool {
	return m.Role == "owner" || m.Role == "coordinator" || m.Role == "reviewer"
}

// CanManageProject returns true if the role permits editing project settings and members.
func (m *ProjectMember) CanManageProject() bool {
	return m.Role == "owner"
}

// UserProjectAccess is a compact summary of one user's access to one project,
// used by middleware to make scoping decisions without loading the full member record.
type UserProjectAccess struct {
	ProjectID     string
	Role          string
	InstitutionID *string // nil = see all studies; non-nil = scoped to this institution
}

// IsSiteScoped returns true when the user can only see one institution's studies.
func (a *UserProjectAccess) IsSiteScoped() bool {
	return a.InstitutionID != nil
}

// ValidProjectMemberRole returns true when the role string is a known valid value.
func ValidProjectMemberRole(role string) bool {
	switch role {
	case "owner", "coordinator", "reviewer", "site_coordinator", "site_viewer":
		return true
	}
	return false
}

// ListProjectMembers returns all members of a project, with joined user and institution names.
func ListProjectMembers(ctx context.Context, db *sql.DB, projectID string) ([]ProjectMember, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			pm.id, pm.project_id, pm.admin_user_id, pm.role, pm.institution_id, pm.notes,
			pm.created_at, pm.updated_at,
			au.email, au.name,
			COALESCE(i.name, '')
		FROM project_members pm
		JOIN admin_users au ON au.id = pm.admin_user_id
		LEFT JOIN institutions i ON i.id = pm.institution_id
		WHERE pm.project_id = $1
		ORDER BY
			CASE pm.role
				WHEN 'owner' THEN 0
				WHEN 'coordinator' THEN 1
				WHEN 'reviewer' THEN 2
				WHEN 'site_coordinator' THEN 3
				WHEN 'site_viewer' THEN 4
				ELSE 5
			END,
			au.email`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProjectMember
	for rows.Next() {
		var m ProjectMember
		if err := scanProjectMember(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetProjectMember returns a single member record by its UUID.
func GetProjectMember(ctx context.Context, db *sql.DB, id string) (*ProjectMember, error) {
	var m ProjectMember
	err := scanProjectMember(db.QueryRowContext(ctx, `
		SELECT
			pm.id, pm.project_id, pm.admin_user_id, pm.role, pm.institution_id, pm.notes,
			pm.created_at, pm.updated_at,
			au.email, au.name,
			COALESCE(i.name, '')
		FROM project_members pm
		JOIN admin_users au ON au.id = pm.admin_user_id
		LEFT JOIN institutions i ON i.id = pm.institution_id
		WHERE pm.id = $1`, id), &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetProjectMemberByUser returns the membership record for a specific user+project pair.
func GetProjectMemberByUser(ctx context.Context, db *sql.DB, projectID, adminUserID string) (*ProjectMember, error) {
	var m ProjectMember
	err := scanProjectMember(db.QueryRowContext(ctx, `
		SELECT
			pm.id, pm.project_id, pm.admin_user_id, pm.role, pm.institution_id, pm.notes,
			pm.created_at, pm.updated_at,
			au.email, au.name,
			COALESCE(i.name, '')
		FROM project_members pm
		JOIN admin_users au ON au.id = pm.admin_user_id
		LEFT JOIN institutions i ON i.id = pm.institution_id
		WHERE pm.project_id = $1 AND pm.admin_user_id = $2`, projectID, adminUserID), &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetUserProjectAccess returns all projects a user is a member of,
// as lightweight UserProjectAccess records for middleware use.
func GetUserProjectAccess(ctx context.Context, db *sql.DB, adminUserID string) ([]UserProjectAccess, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT project_id, role, institution_id
		FROM project_members
		WHERE admin_user_id = $1`, adminUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserProjectAccess
	for rows.Next() {
		var a UserProjectAccess
		if err := rows.Scan(&a.ProjectID, &a.Role, &a.InstitutionID); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetUserAccessForProject returns the UserProjectAccess for one specific project,
// or nil if the user has no membership.
func GetUserAccessForProject(ctx context.Context, db *sql.DB, adminUserID, projectID string) (*UserProjectAccess, error) {
	var a UserProjectAccess
	err := db.QueryRowContext(ctx, `
		SELECT project_id, role, institution_id
		FROM project_members
		WHERE admin_user_id = $1 AND project_id = $2`,
		adminUserID, projectID).Scan(&a.ProjectID, &a.Role, &a.InstitutionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateProjectMember inserts a new project member and populates ID + timestamps.
func CreateProjectMember(ctx context.Context, db *sql.DB, m *ProjectMember) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO project_members (project_id, admin_user_id, role, institution_id, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		m.ProjectID, m.AdminUserID, m.Role, m.InstitutionID, m.Notes,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// UpdateProjectMember updates role, institution_id, and notes for an existing member.
func UpdateProjectMember(ctx context.Context, db *sql.DB, m *ProjectMember) error {
	_, err := db.ExecContext(ctx, `
		UPDATE project_members
		SET role=$1, institution_id=$2, notes=$3, updated_at=now()
		WHERE id=$4`,
		m.Role, m.InstitutionID, m.Notes, m.ID)
	return err
}

// DeleteProjectMember removes a project membership record by its UUID.
func DeleteProjectMember(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM project_members WHERE id = $1`, id)
	return err
}

// CountProjectMembers returns the number of members in a project.
func CountProjectMembers(ctx context.Context, db *sql.DB, projectID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM project_members WHERE project_id = $1`, projectID).Scan(&n)
	return n, err
}

func scanProjectMember(row scannable, m *ProjectMember) error {
	return row.Scan(
		&m.ID, &m.ProjectID, &m.AdminUserID, &m.Role, &m.InstitutionID, &m.Notes,
		&m.CreatedAt, &m.UpdatedAt,
		&m.UserEmail, &m.UserName, &m.InstitutionName,
	)
}
