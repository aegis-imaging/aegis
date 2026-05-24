package model

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// SatelliteEnrollmentToken is a one-time secret a satellite router presents during
// bootstrap to mint its mTLS client cert. Stored hashed; the raw value is
// returned exactly once at creation and never again.
type SatelliteEnrollmentToken struct {
	ID            string     `json:"id"`
	InstitutionID string     `json:"institution_id"`
	Label         string     `json:"label"`
	CreatedBy     string     `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

// IsActive reports whether the token can still be redeemed right now.
func (t *SatelliteEnrollmentToken) IsActive(now time.Time) bool {
	return t.UsedAt == nil && t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// ErrEnrollmentTokenInvalid is returned for any "the token is no good"
// situation (not found, expired, used, revoked, wrong institution).
// Deliberately broad — we don't want to leak which-specifically-failed to a
// public unauthenticated endpoint.
var ErrEnrollmentTokenInvalid = errors.New("enrollment token invalid or expired")

// GenerateEnrollmentToken creates a fresh random token and returns
// (raw_token, sha256_hash_hex). The raw token is what the cloud admin hands
// to the satellite; only the hash is stored.
func GenerateEnrollmentToken() (raw, hashHex string, err error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf[:])
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

// HashEnrollmentToken hashes a raw token for lookup.
func HashEnrollmentToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func CreateSatelliteEnrollmentToken(ctx context.Context, db *sql.DB, t *SatelliteEnrollmentToken, hashHex string) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO satellite_enrollment_tokens
			(institution_id, token_hash, label, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		t.InstitutionID, hashHex, t.Label, t.CreatedBy, t.ExpiresAt,
	).Scan(&t.ID, &t.CreatedAt)
}

func ListSatelliteEnrollmentTokens(ctx context.Context, db *sql.DB, institutionID string) ([]SatelliteEnrollmentToken, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, institution_id, label, created_by, created_at, expires_at, used_at, revoked_at
		FROM satellite_enrollment_tokens
		WHERE institution_id = $1
		ORDER BY created_at DESC`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SatelliteEnrollmentToken
	for rows.Next() {
		var t SatelliteEnrollmentToken
		var used, revoked sql.NullTime
		if err := rows.Scan(&t.ID, &t.InstitutionID, &t.Label, &t.CreatedBy,
			&t.CreatedAt, &t.ExpiresAt, &used, &revoked); err != nil {
			return nil, err
		}
		if used.Valid {
			u := used.Time
			t.UsedAt = &u
		}
		if revoked.Valid {
			r := revoked.Time
			t.RevokedAt = &r
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RedeemSatelliteEnrollmentToken looks up a raw token, validates it's still
// active, marks it used in the same transaction, and returns the token row.
// Errors with ErrEnrollmentTokenInvalid for any failure to avoid leaking
// detail to the public enrollment endpoint.
func RedeemSatelliteEnrollmentToken(ctx context.Context, db *sql.DB, rawToken string) (*SatelliteEnrollmentToken, error) {
	hash := HashEnrollmentToken(rawToken)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var t SatelliteEnrollmentToken
	var used, revoked sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT id, institution_id, label, created_by, created_at, expires_at, used_at, revoked_at
		FROM satellite_enrollment_tokens
		WHERE token_hash = $1
		FOR UPDATE`, hash,
	).Scan(&t.ID, &t.InstitutionID, &t.Label, &t.CreatedBy,
		&t.CreatedAt, &t.ExpiresAt, &used, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEnrollmentTokenInvalid
	}
	if err != nil {
		return nil, err
	}
	if used.Valid {
		u := used.Time
		t.UsedAt = &u
	}
	if revoked.Valid {
		r := revoked.Time
		t.RevokedAt = &r
	}
	if !t.IsActive(time.Now().UTC()) {
		return nil, ErrEnrollmentTokenInvalid
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE satellite_enrollment_tokens SET used_at = now() WHERE id = $1`, t.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	t.UsedAt = &now
	return &t, nil
}

func RevokeSatelliteEnrollmentToken(ctx context.Context, db *sql.DB, tokenID string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE satellite_enrollment_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL AND used_at IS NULL`,
		tokenID)
	return err
}
