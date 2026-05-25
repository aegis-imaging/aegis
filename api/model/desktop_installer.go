package model

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// DesktopInstaller is one (product, platform, version) installer entry.
// Either StorageKey or ExternalURL is populated — the other is empty.
type DesktopInstaller struct {
	ID          string    `json:"id"`
	Product     string    `json:"product"`      // 'uploader' | 'dimse-bridge'
	Platform    string    `json:"platform"`     // 'macos-arm64','macos-x64','windows-x64','linux-deb','linux-appimage'
	Version     string    `json:"version"`      // semver
	StorageKey  string    `json:"storage_key,omitempty"`
	ExternalURL string    `json:"external_url,omitempty"`
	Filename    string    `json:"filename"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256      string    `json:"sha256"`
	Changelog   string    `json:"changelog,omitempty"`
	IsCurrent   bool      `json:"is_current"`
	ReleasedAt  time.Time `json:"released_at"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

const installerCols = `id, product, platform, version,
  COALESCE(storage_key, ''), COALESCE(external_url, ''),
  filename, size_bytes, sha256, COALESCE(changelog, ''),
  is_current, released_at, created_by, created_at`

func scanInstaller(rs interface{ Scan(...any) error }) (*DesktopInstaller, error) {
	var i DesktopInstaller
	err := rs.Scan(&i.ID, &i.Product, &i.Platform, &i.Version,
		&i.StorageKey, &i.ExternalURL,
		&i.Filename, &i.SizeBytes, &i.SHA256, &i.Changelog,
		&i.IsCurrent, &i.ReleasedAt, &i.CreatedBy, &i.CreatedAt)
	return &i, err
}

// ListDesktopInstallers returns all installers, newest first.
func ListDesktopInstallers(ctx context.Context, db *sql.DB) ([]DesktopInstaller, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+installerCols+` FROM desktop_installers
		 ORDER BY product, platform, released_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list desktop installers: %w", err)
	}
	defer rows.Close()
	var out []DesktopInstaller
	for rows.Next() {
		i, err := scanInstaller(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *i)
	}
	return out, rows.Err()
}

// GetDesktopInstaller fetches one row by ID.
func GetDesktopInstaller(ctx context.Context, db *sql.DB, id string) (*DesktopInstaller, error) {
	return scanInstaller(db.QueryRowContext(ctx,
		`SELECT `+installerCols+` FROM desktop_installers WHERE id = $1`, id))
}

// CreateDesktopInstallerInput collects all required fields for creation.
type CreateDesktopInstallerInput struct {
	Product     string
	Platform    string
	Version     string
	StorageKey  string // either this
	ExternalURL string // or this — not both
	Filename    string
	SizeBytes   int64
	SHA256      string
	Changelog   string
	CreatedBy   string
}

// CreateDesktopInstaller inserts a new installer row.
func CreateDesktopInstaller(ctx context.Context, db *sql.DB, in CreateDesktopInstallerInput) (*DesktopInstaller, error) {
	var storageKey, externalURL *string
	if in.StorageKey != "" {
		storageKey = &in.StorageKey
	}
	if in.ExternalURL != "" {
		externalURL = &in.ExternalURL
	}
	return scanInstaller(db.QueryRowContext(ctx,
		`INSERT INTO desktop_installers
		   (product, platform, version, storage_key, external_url,
		    filename, size_bytes, sha256, changelog, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING `+installerCols,
		in.Product, in.Platform, in.Version, storageKey, externalURL,
		in.Filename, in.SizeBytes, in.SHA256, in.Changelog, in.CreatedBy))
}

// UpdateDesktopInstallerChangelog edits the changelog text in place.
func UpdateDesktopInstallerChangelog(ctx context.Context, db *sql.DB, id, changelog string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE desktop_installers SET changelog = $1 WHERE id = $2`, changelog, id)
	return err
}

// SetDesktopInstallerCurrent marks one (product, platform) version as current and
// clears the flag on all other versions of the same (product, platform). The
// partial unique index on (product, platform) WHERE is_current ensures only
// one is_current per pair.
func SetDesktopInstallerCurrent(ctx context.Context, db *sql.DB, id string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var product, platform string
	if err := tx.QueryRowContext(ctx,
		`SELECT product, platform FROM desktop_installers WHERE id = $1`, id,
	).Scan(&product, &platform); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE desktop_installers SET is_current = FALSE
		 WHERE product = $1 AND platform = $2 AND is_current`,
		product, platform); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE desktop_installers SET is_current = TRUE WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteDesktopInstaller removes a row by ID. Storage cleanup is the caller's job.
func DeleteDesktopInstaller(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM desktop_installers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ----------------------------------------------------------------------------
// Invites

// DesktopInstallerInvite is one emailed install link.
type DesktopInstallerInvite struct {
	ID              string     `json:"id"`
	InstallerID     string     `json:"installer_id"`
	InstitutionID   *string    `json:"institution_id,omitempty"` // nil = institution-less invite (legacy / global)
	RecipientEmail  string     `json:"recipient_email"`
	RecipientName   string     `json:"recipient_name,omitempty"`
	Token           string     `json:"token"`
	ExpiresAt       time.Time  `json:"expires_at"`
	SentAt          time.Time  `json:"sent_at"`
	SentBy          string     `json:"sent_by"`
	FirstClickedAt  *time.Time `json:"first_clicked_at,omitempty"`
	LastClickedAt   *time.Time `json:"last_clicked_at,omitempty"`
	ClickCount      int        `json:"click_count"`
	PairedAt        *time.Time `json:"paired_at,omitempty"`
	PairedAPIKeyID  *string    `json:"paired_api_key_id,omitempty"`

	// Joined fields (populated by ListDesktopInstallerInvites only).
	Product         string  `json:"product,omitempty"`
	Platform        string  `json:"platform,omitempty"`
	Version         string  `json:"version,omitempty"`
	InstitutionName *string `json:"institution_name,omitempty"` // joined from institutions when InstitutionID is set

	// pairing carries the single-use pairing token only on the in-memory invite
	// returned from CreateDesktopInstallerInvite. Unexported so it never reaches
	// JSON or any read path — once the email is sent, the only copy lives in
	// the database until first exchange.
	pairing string
}

// generateInstallerToken returns a URL-safe 32-byte random token (43 chars).
func generateInstallerToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateDesktopInstallerInviteInput collects required fields for invite creation.
type CreateDesktopInstallerInviteInput struct {
	InstallerID    string
	InstitutionID  string // optional — when set, ties the invite to one institution for the per-institution panel
	RecipientEmail string
	RecipientName  string
	ExpiresAt      time.Time
	SentBy         string
}

// CreateDesktopInstallerInvite inserts a new invite row and returns it with
// freshly-generated token + pairing_token.
func CreateDesktopInstallerInvite(ctx context.Context, db *sql.DB, in CreateDesktopInstallerInviteInput) (*DesktopInstallerInvite, error) {
	token, err := generateInstallerToken()
	if err != nil {
		return nil, err
	}
	pairing, err := generateInstallerToken()
	if err != nil {
		return nil, err
	}

	inv := &DesktopInstallerInvite{
		InstallerID:    in.InstallerID,
		RecipientEmail: strings.TrimSpace(in.RecipientEmail),
		RecipientName:  strings.TrimSpace(in.RecipientName),
		Token:          token,
		ExpiresAt:      in.ExpiresAt,
		SentBy:         in.SentBy,
	}
	if instID := strings.TrimSpace(in.InstitutionID); instID != "" {
		inv.InstitutionID = &instID
	}
	var nameArg, instArg, pairedID interface{}
	if inv.RecipientName != "" {
		nameArg = inv.RecipientName
	}
	if inv.InstitutionID != nil {
		instArg = *inv.InstitutionID
	}

	err = db.QueryRowContext(ctx,
		`INSERT INTO desktop_installer_invites
		   (installer_id, institution_id, recipient_email, recipient_name, token, pairing_token, expires_at, sent_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, sent_at, paired_api_key_id`,
		inv.InstallerID, instArg, inv.RecipientEmail, nameArg,
		token, pairing, inv.ExpiresAt.UTC(), inv.SentBy,
	).Scan(&inv.ID, &inv.SentAt, &pairedID)
	if err != nil {
		return nil, err
	}
	// Re-attach the pairing token so the caller can build the install URL.
	// We never expose the pairing token in subsequent reads — it's single-use.
	inv.PairedAPIKeyID = nil
	return invWithPairing(inv, pairing), nil
}

// invWithPairing attaches a freshly-generated pairing token to the invite so
// the handler can read it once after creation. Callers retrieve it via
// PairingToken(). The field never appears in JSON or in reads from the DB.
func invWithPairing(inv *DesktopInstallerInvite, pairing string) *DesktopInstallerInvite {
	inv.pairing = pairing
	return inv
}

// PairingToken returns the one-time pairing token captured at creation time.
// Only populated on the in-memory invite returned from CreateDesktopInstallerInvite.
func (i *DesktopInstallerInvite) PairingToken() string { return i.pairing }

// ListDesktopInstallerInvites returns recent invites joined with installer
// product/platform/version for display. When institutionID is non-empty,
// restricts to invites tied to that institution; otherwise returns all.
func ListDesktopInstallerInvites(ctx context.Context, db *sql.DB, limit int) ([]DesktopInstallerInvite, error) {
	return listInstallerInvites(ctx, db, "", limit)
}

// ListDesktopInstallerInvitesByInstitution returns recent invites scoped to
// one institution. Used by the per-institution detail panel so the
// Institution view can list the invites it owns alongside its satellites.
func ListDesktopInstallerInvitesByInstitution(ctx context.Context, db *sql.DB, institutionID string, limit int) ([]DesktopInstallerInvite, error) {
	return listInstallerInvites(ctx, db, institutionID, limit)
}

func listInstallerInvites(ctx context.Context, db *sql.DB, institutionID string, limit int) ([]DesktopInstallerInvite, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `
		SELECT inv.id, inv.installer_id, inv.institution_id, inv.recipient_email,
		       COALESCE(inv.recipient_name, ''), inv.token,
		       inv.expires_at, inv.sent_at, inv.sent_by,
		       inv.first_clicked_at, inv.last_clicked_at, inv.click_count,
		       inv.paired_at, inv.paired_api_key_id,
		       di.product, di.platform, di.version,
		       inst.name
		  FROM desktop_installer_invites inv
		  JOIN desktop_installers di ON di.id = inv.installer_id
		  LEFT JOIN institutions inst ON inst.id = inv.institution_id`
	var rows *sql.Rows
	var err error
	if institutionID != "" {
		q += ` WHERE inv.institution_id = $1 ORDER BY inv.sent_at DESC LIMIT $2`
		rows, err = db.QueryContext(ctx, q, institutionID, limit)
	} else {
		q += ` ORDER BY inv.sent_at DESC LIMIT $1`
		rows, err = db.QueryContext(ctx, q, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DesktopInstallerInvite
	for rows.Next() {
		var i DesktopInstallerInvite
		if err := rows.Scan(&i.ID, &i.InstallerID, &i.InstitutionID, &i.RecipientEmail, &i.RecipientName,
			&i.Token, &i.ExpiresAt, &i.SentAt, &i.SentBy,
			&i.FirstClickedAt, &i.LastClickedAt, &i.ClickCount,
			&i.PairedAt, &i.PairedAPIKeyID,
			&i.Product, &i.Platform, &i.Version,
			&i.InstitutionName,
		); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetDesktopInstallerInviteByToken looks up an invite by its public URL token.
// Returns sql.ErrNoRows if not found.
func GetDesktopInstallerInviteByToken(ctx context.Context, db *sql.DB, token string) (*DesktopInstallerInvite, *DesktopInstaller, error) {
	var i DesktopInstallerInvite
	var inst DesktopInstaller
	err := db.QueryRowContext(ctx, `
		SELECT inv.id, inv.installer_id, inv.institution_id, inv.recipient_email,
		       COALESCE(inv.recipient_name, ''), inv.token,
		       inv.expires_at, inv.sent_at, inv.sent_by,
		       inv.first_clicked_at, inv.last_clicked_at, inv.click_count,
		       inv.paired_at, inv.paired_api_key_id,
		       di.id, di.product, di.platform, di.version,
		       COALESCE(di.storage_key, ''), COALESCE(di.external_url, ''),
		       di.filename, di.size_bytes, di.sha256, COALESCE(di.changelog, ''),
		       di.is_current, di.released_at, di.created_by, di.created_at
		  FROM desktop_installer_invites inv
		  JOIN desktop_installers di ON di.id = inv.installer_id
		 WHERE inv.token = $1`, token).Scan(
		&i.ID, &i.InstallerID, &i.InstitutionID, &i.RecipientEmail, &i.RecipientName,
		&i.Token, &i.ExpiresAt, &i.SentAt, &i.SentBy,
		&i.FirstClickedAt, &i.LastClickedAt, &i.ClickCount,
		&i.PairedAt, &i.PairedAPIKeyID,
		&inst.ID, &inst.Product, &inst.Platform, &inst.Version,
		&inst.StorageKey, &inst.ExternalURL,
		&inst.Filename, &inst.SizeBytes, &inst.SHA256, &inst.Changelog,
		&inst.IsCurrent, &inst.ReleasedAt, &inst.CreatedBy, &inst.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}
	return &i, &inst, nil
}

// RecordDesktopInstallerInviteClick increments click_count and updates
// first_clicked_at (set once) and last_clicked_at (every click).
func RecordDesktopInstallerInviteClick(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE desktop_installer_invites
		   SET click_count      = click_count + 1,
		       first_clicked_at = COALESCE(first_clicked_at, NOW()),
		       last_clicked_at  = NOW()
		 WHERE id = $1`, id)
	return err
}

// ClaimDesktopInstallerInvitePairing atomically marks the invite as paired
// (setting paired_at + paired_api_key_id) iff it isn't already paired.
// Returns (true, nil) on first call, (false, nil) on subsequent calls or
// expired/unknown pairing tokens.
//
// The pairing_token is single-use: this update only matches rows where
// paired_at IS NULL.
func ClaimDesktopInstallerInvitePairing(ctx context.Context, db *sql.DB, pairingToken, apiKeyID string) (*DesktopInstallerInvite, bool, error) {
	var inv DesktopInstallerInvite
	err := db.QueryRowContext(ctx, `
		UPDATE desktop_installer_invites
		   SET paired_at         = NOW(),
		       paired_api_key_id = $2
		 WHERE pairing_token = $1
		   AND paired_at IS NULL
		   AND expires_at > NOW()
		 RETURNING id, installer_id, institution_id, recipient_email, COALESCE(recipient_name, ''),
		           token, expires_at, sent_at, sent_by,
		           first_clicked_at, last_clicked_at, click_count,
		           paired_at, paired_api_key_id`,
		pairingToken, apiKeyID,
	).Scan(&inv.ID, &inv.InstallerID, &inv.InstitutionID, &inv.RecipientEmail, &inv.RecipientName,
		&inv.Token, &inv.ExpiresAt, &inv.SentAt, &inv.SentBy,
		&inv.FirstClickedAt, &inv.LastClickedAt, &inv.ClickCount,
		&inv.PairedAt, &inv.PairedAPIKeyID)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &inv, true, nil
}
