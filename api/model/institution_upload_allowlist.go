package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// UploadMethod describes one browser/desktop upload flow that can be
// individually allowed or denied per institution.
type UploadMethod struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// KnownUploadMethods is the canonical enum of methods the allowlist
// understands. Adding a new flow means appending here AND wiring the
// IsUploadMethodAllowed check into the matching handler — see
// docs/planning/browser-upload-allowlist-design.md for the rollout
// plan (chunk 2 wires enforcement; chunk 1 is the table + API only).
var KnownUploadMethods = []UploadMethod{
	{
		ID:          "browser.web-upload",
		Label:       "Standard web upload",
		Description: "Drag-and-drop upload via the public upload portal.",
	},
	{
		ID:          "browser.share-redeem",
		Label:       "Share-token redeem",
		Description: "Returning to upload via a share-token URL.",
	},
	{
		ID:          "browser.dimse-pull",
		Label:       "Pull from PACS (DIMSE C-MOVE)",
		Description: "Web wizard that triggers a DIMSE C-MOVE from the institution's PACS.",
	},
	{
		ID:          "browser.tcia-import",
		Label:       "TCIA collection import",
		Description: "Import series from the public Cancer Imaging Archive.",
	},
	{
		ID:          "desktop.installer-pair",
		Label:       "Desktop installer pairing",
		Description: "First-launch pairing of the desktop uploader/DIMSE bridge.",
	},
}

// IsKnownUploadMethod reports whether methodID matches a member of
// KnownUploadMethods. Used by the handler to reject bogus IDs at the
// edge so the table never accumulates orphan rows for methods that
// aren't wired anywhere.
func IsKnownUploadMethod(methodID string) bool {
	for _, m := range KnownUploadMethods {
		if m.ID == methodID {
			return true
		}
	}
	return false
}

// InstitutionUploadAllowlistRow is one explicit row in the table.
// Absence of a row for a (institution_id, method_id) pair means
// "allowed by default" — see the design doc.
type InstitutionUploadAllowlistRow struct {
	InstitutionID string    `json:"institution_id"`
	MethodID      string    `json:"method_id"`
	Enabled       bool      `json:"enabled"`
	Note          string    `json:"note"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdatedBy     string    `json:"updated_by"`
}

// EffectiveAllowlistEntry is one row of "what the API actually
// returns" — one entry per KnownUploadMethod with the resolved
// enabled state. IsDefault is true when no explicit row exists
// for this (institution_id, method_id) and the response is
// reflecting the default-on fallback.
type EffectiveAllowlistEntry struct {
	Method    UploadMethod `json:"method"`
	Enabled   bool         `json:"enabled"`
	IsDefault bool         `json:"is_default"`
	Note      string       `json:"note"`
	UpdatedAt *time.Time   `json:"updated_at,omitempty"`
	UpdatedBy string       `json:"updated_by,omitempty"`
}

// ListUploadAllowlistRows returns every explicit row for the given
// institution. Most callers should use ListEffectiveUploadAllowlist
// instead — this is the raw-row view, useful for diffing.
func ListUploadAllowlistRows(ctx context.Context, db *sql.DB, institutionID string) ([]InstitutionUploadAllowlistRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT institution_id, method_id, enabled, note, updated_at, updated_by
		FROM institution_upload_allowlist
		WHERE institution_id = $1`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstitutionUploadAllowlistRow
	for rows.Next() {
		var r InstitutionUploadAllowlistRow
		if err := rows.Scan(&r.InstitutionID, &r.MethodID, &r.Enabled, &r.Note, &r.UpdatedAt, &r.UpdatedBy); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListEffectiveUploadAllowlist returns one EffectiveAllowlistEntry per
// KnownUploadMethod, with explicit rows merged in. Stable ordering
// matches KnownUploadMethods so the UI can render a deterministic
// list of toggles.
func ListEffectiveUploadAllowlist(ctx context.Context, db *sql.DB, institutionID string) ([]EffectiveAllowlistEntry, error) {
	explicit, err := ListUploadAllowlistRows(ctx, db, institutionID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]InstitutionUploadAllowlistRow, len(explicit))
	for _, r := range explicit {
		byID[r.MethodID] = r
	}
	out := make([]EffectiveAllowlistEntry, 0, len(KnownUploadMethods))
	for _, m := range KnownUploadMethods {
		entry := EffectiveAllowlistEntry{Method: m, Enabled: true, IsDefault: true}
		if r, ok := byID[m.ID]; ok {
			entry.Enabled = r.Enabled
			entry.IsDefault = false
			entry.Note = r.Note
			t := r.UpdatedAt
			entry.UpdatedAt = &t
			entry.UpdatedBy = r.UpdatedBy
		}
		out = append(out, entry)
	}
	return out, nil
}

// IsUploadMethodAllowed reports whether the given upload method is
// permitted for the given institution. Unknown method IDs return an
// error so we never silently "allow" a method whose check site has
// drifted from KnownUploadMethods. Absence of a row → allowed (the
// default-on contract).
func IsUploadMethodAllowed(ctx context.Context, db *sql.DB, institutionID, methodID string) (bool, error) {
	if !IsKnownUploadMethod(methodID) {
		return false, fmt.Errorf("unknown upload method %q", methodID)
	}
	var enabled bool
	err := db.QueryRowContext(ctx, `
		SELECT enabled FROM institution_upload_allowlist
		WHERE institution_id = $1 AND method_id = $2`, institutionID, methodID).Scan(&enabled)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// UpsertUploadAllowlistRow inserts or updates a single (institution_id,
// method_id) row. Caller is responsible for validating methodID against
// IsKnownUploadMethod.
func UpsertUploadAllowlistRow(ctx context.Context, db *sql.DB, r *InstitutionUploadAllowlistRow) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO institution_upload_allowlist (
			institution_id, method_id, enabled, note, updated_at, updated_by
		)
		VALUES ($1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (institution_id, method_id) DO UPDATE
		SET enabled    = EXCLUDED.enabled,
		    note       = EXCLUDED.note,
		    updated_at = NOW(),
		    updated_by = EXCLUDED.updated_by
		RETURNING updated_at`,
		r.InstitutionID, r.MethodID, r.Enabled, r.Note, r.UpdatedBy,
	).Scan(&r.UpdatedAt)
}

// DeleteUploadAllowlistRow removes the explicit row, reverting the
// (institution, method) pair to default-on. Returns sql.ErrNoRows if
// nothing matched, so callers can distinguish "already at default"
// from a successful revert.
func DeleteUploadAllowlistRow(ctx context.Context, db *sql.DB, institutionID, methodID string) error {
	res, err := db.ExecContext(ctx, `
		DELETE FROM institution_upload_allowlist
		WHERE institution_id = $1 AND method_id = $2`, institutionID, methodID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
