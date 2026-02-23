package model

import (
	"context"
	"database/sql"
	"time"
)

// APIKey represents a long-lived machine-to-machine API key.
// The raw key value is never stored; only the SHA-256 hex hash and
// the first 8 characters (prefix) are retained for display.
type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	CreatedBy  string     `json:"created_by"`
	Enabled    bool       `json:"enabled"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func scanAPIKey(row interface {
	Scan(...any) error
}) (*APIKey, error) {
	var k APIKey
	err := row.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.CreatedBy,
		&k.Enabled, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt)
	return &k, err
}

const apiKeyCols = `id, name, key_prefix, created_by, enabled, last_used_at, expires_at, created_at, updated_at`

// ListAPIKeys returns all API keys ordered by creation time descending.
func ListAPIKeys(ctx context.Context, db *sql.DB) ([]APIKey, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+apiKeyCols+` FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *k)
	}
	return out, rows.Err()
}

// GetAPIKeyByHash looks up a key by its SHA-256 hash.
func GetAPIKeyByHash(ctx context.Context, db *sql.DB, keyHash string) (*APIKey, error) {
	return scanAPIKey(db.QueryRowContext(ctx,
		`SELECT `+apiKeyCols+` FROM api_keys WHERE key_hash = $1`, keyHash))
}

// CreateAPIKey inserts a new API key record and returns it.
func CreateAPIKey(ctx context.Context, db *sql.DB, name, keyHash, keyPrefix, createdBy string, expiresAt *time.Time) (*APIKey, error) {
	return scanAPIKey(db.QueryRowContext(ctx,
		`INSERT INTO api_keys (name, key_hash, key_prefix, created_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+apiKeyCols,
		name, keyHash, keyPrefix, createdBy, expiresAt))
}

// UpdateAPIKeyEnabled enables or disables an API key.
func UpdateAPIKeyEnabled(ctx context.Context, db *sql.DB, id string, enabled bool) error {
	_, err := db.ExecContext(ctx,
		`UPDATE api_keys SET enabled = $1, updated_at = now() WHERE id = $2`, enabled, id)
	return err
}

// RotateAPIKey replaces the key_hash and key_prefix for an existing API key and
// returns the updated record. The caller is responsible for generating the new
// values and returning the raw key to the user exactly once.
func RotateAPIKey(ctx context.Context, db *sql.DB, id, newHash, newPrefix string) (*APIKey, error) {
	return scanAPIKey(db.QueryRowContext(ctx,
		`UPDATE api_keys SET key_hash = $1, key_prefix = $2, updated_at = now()
		 WHERE id = $3
		 RETURNING `+apiKeyCols,
		newHash, newPrefix, id))
}

// DeleteAPIKey permanently removes an API key.
func DeleteAPIKey(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	return err
}

// TouchAPIKey updates last_used_at to now (best-effort, non-fatal).
func TouchAPIKey(ctx context.Context, db *sql.DB, id string) {
	db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id) //nolint:errcheck
}
