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
	RetentionDays         *int      `json:"retention_days,omitempty"`           // nil = keep indefinitely
	StuckThresholdMinutes *int      `json:"stuck_threshold_minutes,omitempty"`  // nil = use request default (60)
	StorageQuotaBytes     *int64    `json:"storage_quota_bytes,omitempty"`      // nil = unlimited
	Archived              bool      `json:"archived"`
	Restricted            bool      `json:"restricted"`                         // true = only project_members + platform admin can see
	TenantID              *string   `json:"tenant_id,omitempty"`                // nil = legacy single-tenant project
	MemberCount           int       `json:"member_count,omitempty"`             // populated by ListProjects when available
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

const projectColumns = `id, name, slug, description, default_anon_profile_id, retention_days, stuck_threshold_minutes, storage_quota_bytes, archived, restricted, tenant_id, created_at, updated_at`

func scanProject(row scannable, p *Project) error {
	return row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.DefaultAnonProfileID, &p.RetentionDays, &p.StuckThresholdMinutes, &p.StorageQuotaBytes, &p.Archived, &p.Restricted, &p.TenantID, &p.CreatedAt, &p.UpdatedAt)
}

// ListProjects returns all projects (used by platform admin/viewer and internal callers).
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

// ListProjectsForResearcher returns only the projects that a researcher-role user is a member of,
// with member_count populated. Used when admin_users.role = 'researcher'.
func ListProjectsForResearcher(ctx context.Context, db *sql.DB, adminUserID string) ([]Project, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			p.id, p.name, p.slug, p.description, p.default_anon_profile_id,
			p.retention_days, p.stuck_threshold_minutes, p.storage_quota_bytes,
			p.archived, p.restricted, p.tenant_id, p.created_at, p.updated_at
		FROM projects p
		JOIN project_members pm ON pm.project_id = p.id
		WHERE pm.admin_user_id = $1
		ORDER BY p.created_at`,
		adminUserID)
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

// ListProjectsPublic returns non-restricted projects.
// Used for unauthenticated callers (upload portal, public project list).
func ListProjectsPublic(ctx context.Context, db *sql.DB) ([]Project, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE restricted = false ORDER BY created_at`)
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

// SetProjectRestricted updates the restricted flag on a project.
func SetProjectRestricted(ctx context.Context, db *sql.DB, projectID string, restricted bool) error {
	_, err := db.ExecContext(ctx,
		`UPDATE projects SET restricted=$1, updated_at=now() WHERE id=$2`,
		restricted, projectID)
	return err
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

// UpdateProjectStorageQuota sets or clears (nil) the storage quota for a project.
func UpdateProjectStorageQuota(ctx context.Context, db *sql.DB, projectID string, bytes *int64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE projects SET storage_quota_bytes = $1, updated_at = now() WHERE id = $2`,
		bytes, projectID)
	return err
}

// GetProjectStorageUsage returns the total size in bytes of all studies for a project.
// It sums study_size_bytes from the studies table (populated by the DIMSE receiver and upload pipeline).
func GetProjectStorageUsage(ctx context.Context, db *sql.DB, projectID string) (int64, error) {
	var used sql.NullInt64
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(study_size_bytes), 0) FROM studies WHERE project_id = $1`,
		projectID).Scan(&used)
	if err != nil {
		return 0, err
	}
	return used.Int64, nil
}

// UpdateProjectRetentionDays sets or clears (nil) the retention policy for a project.
func UpdateProjectRetentionDays(ctx context.Context, db *sql.DB, projectID string, days *int) error {
	_, err := db.ExecContext(ctx, `
		UPDATE projects SET retention_days = $1, updated_at = now() WHERE id = $2`,
		days, projectID)
	return err
}

func CreateProject(ctx context.Context, db *sql.DB, name, slug, description string) (*Project, error) {
	return CreateProjectForTenant(ctx, db, name, slug, description, nil)
}

// CreateProjectForTenant inserts a project owned by the given tenant.
// Passing nil for tenantID creates a legacy untenanted project (same row
// shape CreateProject has always produced) — same on-disk default.
func CreateProjectForTenant(ctx context.Context, db *sql.DB, name, slug, description string, tenantID *string) (*Project, error) {
	var p Project
	err := scanProject(db.QueryRowContext(ctx, `
		INSERT INTO projects (name, slug, description, tenant_id)
		VALUES ($1, $2, $3, $4)
		RETURNING `+projectColumns,
		name, slug, description, tenantID), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListProjectsForTenant returns every project belonging to a tenant. Used
// by ListProjects when a tenant is in the request context.
func ListProjectsForTenant(ctx context.Context, db *sql.DB, tenantID string) ([]Project, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE tenant_id = $1 ORDER BY created_at`,
		tenantID)
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
