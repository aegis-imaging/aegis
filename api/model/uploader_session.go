package model

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"
)

// UploaderSession is a browser session for an uploader. The id IS the cookie
// value, so it must be unguessable - generated with crypto/rand at login time.
type UploaderSession struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// generateSessionID produces a 256-bit random token, URL-safe base64.
// 256 bits is the same entropy used elsewhere in the codebase for pairing
// tokens; safe to ship as an HttpOnly cookie value.
func generateSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// CreateUploaderSession inserts a fresh session row and returns it. Callers set
// the resulting ID as the cookie value.
func CreateUploaderSession(ctx context.Context, db *sql.DB, userID, ip, ua string, ttl time.Duration) (*UploaderSession, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}
	s := &UploaderSession{}
	err = db.QueryRowContext(ctx, `
		INSERT INTO uploader_sessions (id, user_id, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, ip_address, user_agent, expires_at, last_used_at, created_at`,
		id, userID, ip, ua, time.Now().UTC().Add(ttl),
	).Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.LastUsedAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetUploaderSession looks up an active session by id. Returns sql.ErrNoRows
// if missing or expired.
func GetUploaderSession(ctx context.Context, db *sql.DB, id string) (*UploaderSession, error) {
	s := &UploaderSession{}
	err := db.QueryRowContext(ctx, `
		SELECT id, user_id, ip_address, user_agent, expires_at, last_used_at, created_at
		FROM uploader_sessions
		WHERE id = $1 AND expires_at > now()`, id,
	).Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.LastUsedAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// TouchUploaderSession bumps last_used_at. Best-effort: callers fire-and-forget.
func TouchUploaderSession(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE uploader_sessions SET last_used_at = now() WHERE id = $1`, id)
	return err
}

// DeleteUploaderSession removes a single session row (logout).
func DeleteUploaderSession(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM uploader_sessions WHERE id = $1`, id)
	return err
}

// DeleteUploaderSessionsByUser removes every session for a user. Called when
// the user's project access is revoked or their password is reset, so an
// attacker holding an old cookie cannot continue to upload.
func DeleteUploaderSessionsByUser(ctx context.Context, db *sql.DB, userID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM uploader_sessions WHERE user_id = $1`, userID)
	return err
}
