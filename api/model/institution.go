package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Institution represents an organisation that sends or receives studies.
type Institution struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	Type         string    `json:"institution_type"` // sender | receiver | both
	ContactName  string    `json:"contact_name"`
	ContactEmail string    `json:"contact_email"`
	IPRanges     string    `json:"ip_ranges"` // comma-separated CIDR blocks
	AETitle      string    `json:"ae_title"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// InstitutionProject links an institution to a project with a role.
type InstitutionProject struct {
	InstitutionID string    `json:"institution_id"`
	ProjectID     string    `json:"project_id"`
	Role          string    `json:"role"` // sender | receiver | admin
	CreatedAt     time.Time `json:"created_at"`

	// Populated by joins for convenience
	InstitutionName string `json:"institution_name,omitempty"`
	ProjectName     string `json:"project_name,omitempty"`
}

const institutionColumns = `
	id, name, slug, description, institution_type,
	contact_name, contact_email, ip_ranges, ae_title,
	enabled, created_at, updated_at`

func scanInstitution(row scannable, inst *Institution) error {
	return row.Scan(
		&inst.ID, &inst.Name, &inst.Slug, &inst.Description, &inst.Type,
		&inst.ContactName, &inst.ContactEmail, &inst.IPRanges, &inst.AETitle,
		&inst.Enabled, &inst.CreatedAt, &inst.UpdatedAt,
	)
}

func CreateInstitution(ctx context.Context, db *sql.DB, inst *Institution) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO institutions (name, slug, description, institution_type,
		                          contact_name, contact_email, ip_ranges, ae_title, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		inst.Name, inst.Slug, inst.Description, inst.Type,
		inst.ContactName, inst.ContactEmail, inst.IPRanges, inst.AETitle, inst.Enabled,
	).Scan(&inst.ID, &inst.CreatedAt, &inst.UpdatedAt)
}

func GetInstitutionByID(ctx context.Context, db *sql.DB, id string) (*Institution, error) {
	var inst Institution
	err := scanInstitution(db.QueryRowContext(ctx,
		`SELECT`+institutionColumns+` FROM institutions WHERE id = $1`, id), &inst)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// GetInstitutionBySlug returns an institution by slug (case-insensitive).
func GetInstitutionBySlug(ctx context.Context, db *sql.DB, slug string) (*Institution, error) {
	var inst Institution
	err := scanInstitution(db.QueryRowContext(ctx,
		`SELECT`+institutionColumns+` FROM institutions WHERE lower(slug) = lower($1)`, slug), &inst)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

var ErrInstitutionAETitleAmbiguous = errors.New("institution ae_title matched multiple enabled institutions")

// GetInstitutionByAETitle returns the enabled institution matching ae_title
// (case-insensitive, trimmed). If more than one enabled institution matches,
// ErrInstitutionAETitleAmbiguous is returned so callers can require an explicit
// selector (institution_id or institution_slug).
func GetInstitutionByAETitle(ctx context.Context, db *sql.DB, aeTitle string) (*Institution, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT`+institutionColumns+`
		FROM institutions
		WHERE enabled = TRUE
		  AND lower(trim(ae_title)) = lower(trim($1))
		ORDER BY created_at ASC
		LIMIT 2`, aeTitle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []Institution
	for rows.Next() {
		var inst Institution
		if err := scanInstitution(rows, &inst); err != nil {
			return nil, err
		}
		matches = append(matches, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("%w: %q", ErrInstitutionAETitleAmbiguous, aeTitle)
	}
	return &matches[0], nil
}

func ListInstitutions(ctx context.Context, db *sql.DB) ([]Institution, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+institutionColumns+` FROM institutions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Institution
	for rows.Next() {
		var inst Institution
		if err := scanInstitution(rows, &inst); err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, rows.Err()
}

func UpdateInstitution(ctx context.Context, db *sql.DB, inst *Institution) error {
	_, err := db.ExecContext(ctx, `
		UPDATE institutions SET
			name=$1, slug=$2, description=$3, institution_type=$4,
			contact_name=$5, contact_email=$6, ip_ranges=$7, ae_title=$8,
			enabled=$9, updated_at=now()
		WHERE id=$10`,
		inst.Name, inst.Slug, inst.Description, inst.Type,
		inst.ContactName, inst.ContactEmail, inst.IPRanges, inst.AETitle,
		inst.Enabled, inst.ID)
	return err
}

func DeleteInstitution(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM institutions WHERE id = $1`, id)
	return err
}

// ── Institution-Project links ─────────────────────────────────────────────────

func AddInstitutionToProject(ctx context.Context, db *sql.DB, ip *InstitutionProject) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO institution_projects (institution_id, project_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (institution_id, project_id) DO UPDATE SET role = $3`,
		ip.InstitutionID, ip.ProjectID, ip.Role)
	return err
}

func RemoveInstitutionFromProject(ctx context.Context, db *sql.DB, institutionID, projectID string) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM institution_projects WHERE institution_id=$1 AND project_id=$2`,
		institutionID, projectID)
	return err
}

// ListProjectsForInstitution returns all projects linked to an institution.
func ListProjectsForInstitution(ctx context.Context, db *sql.DB, institutionID string) ([]InstitutionProject, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ip.institution_id, ip.project_id, ip.role, ip.created_at,
		       i.name AS institution_name, p.name AS project_name
		FROM institution_projects ip
		JOIN institutions i ON i.id = ip.institution_id
		JOIN projects     p ON p.id = ip.project_id
		WHERE ip.institution_id = $1
		ORDER BY p.name`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []InstitutionProject
	for rows.Next() {
		var ip InstitutionProject
		if err := rows.Scan(&ip.InstitutionID, &ip.ProjectID, &ip.Role, &ip.CreatedAt,
			&ip.InstitutionName, &ip.ProjectName); err != nil {
			return nil, err
		}
		out = append(out, ip)
	}
	return out, rows.Err()
}

// ListInstitutionsForProject returns all institutions linked to a project.
func ListInstitutionsForProject(ctx context.Context, db *sql.DB, projectID string) ([]InstitutionProject, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ip.institution_id, ip.project_id, ip.role, ip.created_at,
		       i.name AS institution_name, p.name AS project_name
		FROM institution_projects ip
		JOIN institutions i ON i.id = ip.institution_id
		JOIN projects     p ON p.id = ip.project_id
		WHERE ip.project_id = $1
		ORDER BY i.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []InstitutionProject
	for rows.Next() {
		var ip InstitutionProject
		if err := rows.Scan(&ip.InstitutionID, &ip.ProjectID, &ip.Role, &ip.CreatedAt,
			&ip.InstitutionName, &ip.ProjectName); err != nil {
			return nil, err
		}
		out = append(out, ip)
	}
	return out, rows.Err()
}

// InstitutionCanSendToProject returns true when the institution is enabled,
// has type sender/both, and is linked to the project with sender/admin role.
func InstitutionCanSendToProject(ctx context.Context, db *sql.DB, institutionID, projectID string) (bool, error) {
	var allowed bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM institutions i
			JOIN institution_projects ip ON ip.institution_id = i.id
			WHERE i.id = $1
			  AND i.enabled = TRUE
			  AND i.institution_type IN ('sender', 'both')
			  AND ip.project_id = $2
			  AND ip.role IN ('sender', 'admin')
		)`, institutionID, projectID).Scan(&allowed)
	if err != nil {
		return false, err
	}
	return allowed, nil
}
