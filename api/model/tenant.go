package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

// Tenant is the top-level isolation boundary for a multi-tenant SaaS
// deployment. One tenant per customer organisation; projects belong to
// at most one tenant (NULL = legacy single-tenant deployment).
type Tenant struct {
	ID        string          `json:"id"`
	Slug      string          `json:"slug"`
	Name      string          `json:"name"`
	Settings  json.RawMessage `json:"settings"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ErrTenantNotFound is returned when a tenant lookup misses.
var ErrTenantNotFound = errors.New("tenant not found")

// tenantSlugRe matches the conservative subset of characters that work in
// both a DNS subdomain and a URL path segment.
var tenantSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

// ValidateTenantSlug returns nil if `slug` is acceptable as a tenant slug:
// 3-64 chars, lowercase alphanumeric + hyphen, must start and end with
// alphanumeric. This is the constraint we'll enforce at the subdomain
// boundary later, so reject bad slugs up front.
func ValidateTenantSlug(slug string) error {
	if !tenantSlugRe.MatchString(slug) {
		return errors.New("tenant slug must be 3-64 chars, lowercase alphanumeric with internal hyphens")
	}
	return nil
}

const tenantColumns = `id, slug, name, settings, enabled, created_at, updated_at`

func scanTenant(row scannable, t *Tenant) error {
	return row.Scan(&t.ID, &t.Slug, &t.Name, &t.Settings, &t.Enabled, &t.CreatedAt, &t.UpdatedAt)
}

// CreateTenant inserts a new tenant. `slug` is lower-cased + trimmed and
// validated; `settings` may be nil (defaults to `{}`).
func CreateTenant(ctx context.Context, db *sql.DB, t *Tenant) error {
	t.Slug = strings.ToLower(strings.TrimSpace(t.Slug))
	if err := ValidateTenantSlug(t.Slug); err != nil {
		return err
	}
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("tenant name is required")
	}
	if len(t.Settings) == 0 {
		t.Settings = json.RawMessage(`{}`)
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO tenants (slug, name, settings, enabled)
		VALUES ($1, $2, $3, COALESCE($4, TRUE))
		RETURNING id, created_at, updated_at`,
		t.Slug, t.Name, []byte(t.Settings), t.Enabled,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// GetTenant fetches one tenant by UUID. Returns ErrTenantNotFound when
// nothing matches.
func GetTenant(ctx context.Context, db *sql.DB, id string) (*Tenant, error) {
	var t Tenant
	err := scanTenant(db.QueryRowContext(ctx,
		`SELECT `+tenantColumns+` FROM tenants WHERE id = $1`, id), &t)
	if err == sql.ErrNoRows {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTenantBySlug fetches an *enabled* tenant by its slug. Disabled
// tenants are deliberately invisible here — the middleware uses this
// path during request handling, and a disabled tenant should look like
// "no such tenant" from the request's perspective.
func GetTenantBySlug(ctx context.Context, db *sql.DB, slug string) (*Tenant, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	var t Tenant
	err := scanTenant(db.QueryRowContext(ctx,
		`SELECT `+tenantColumns+` FROM tenants WHERE slug = $1 AND enabled = TRUE`, slug), &t)
	if err == sql.ErrNoRows {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTenants returns every tenant in creation order. Admin/platform use only.
func ListTenants(ctx context.Context, db *sql.DB) ([]Tenant, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+tenantColumns+` FROM tenants ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := scanTenant(rows, &t); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

// UpdateTenant updates the mutable fields. Slug is intentionally not
// updatable here — changing a slug breaks subdomain bookmarks and
// inbound webhook routing for the tenant's customers; do it via a
// separate dedicated migration if it ever becomes necessary.
func UpdateTenant(ctx context.Context, db *sql.DB, t *Tenant) error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("tenant name is required")
	}
	if len(t.Settings) == 0 {
		t.Settings = json.RawMessage(`{}`)
	}
	_, err := db.ExecContext(ctx, `
		UPDATE tenants
		SET name = $1, settings = $2, enabled = $3, updated_at = NOW()
		WHERE id = $4`,
		t.Name, []byte(t.Settings), t.Enabled, t.ID)
	return err
}

// DeleteTenant removes a tenant. Returns an error from the FK constraint
// if any project still references it — callers must re-home or delete
// those projects first.
func DeleteTenant(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	return err
}
