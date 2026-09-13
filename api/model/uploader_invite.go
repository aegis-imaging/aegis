package model

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"
)

// UploaderInvite is a pending invitation from an admin/researcher to an outside
// contributor, scoped to one project. The recipient redeems by clicking the
// emailed link and setting a password.
type UploaderInvite struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	ProjectID      string     `json:"project_id"`
	InstitutionID  *string    `json:"institution_id,omitempty"`
	InviteToken    string     `json:"invite_token,omitempty"`
	InvitedBy      string     `json:"invited_by"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RedeemedAt     *time.Time `json:"redeemed_at,omitempty"`
	RedeemedUserID *string    `json:"redeemed_user_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	// Joined for display:
	ProjectName     string `json:"project_name,omitempty"`
	InstitutionName string `json:"institution_name,omitempty"`
}

// CreateUploaderInviteInput captures the fields a caller must supply when
// inviting a new uploader. The invite token is generated server-side.
type CreateUploaderInviteInput struct {
	Email         string
	Name          string
	ProjectID     string
	InstitutionID string // empty -> not site-scoped
	InvitedBy     string
	ExpiresAt     time.Time
}

// generateInviteToken returns a URL-safe random token (~43 chars).
func generateInviteToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func CreateUploaderInvite(ctx context.Context, db *sql.DB, in CreateUploaderInviteInput) (*UploaderInvite, error) {
	token, err := generateInviteToken()
	if err != nil {
		return nil, err
	}
	var instArg any
	if in.InstitutionID != "" {
		instArg = in.InstitutionID
	}
	inv := &UploaderInvite{}
	err = db.QueryRowContext(ctx, `
		INSERT INTO uploader_invites
		    (email, name, project_id, institution_id, invite_token, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, name, project_id, institution_id, invite_token,
		          invited_by, expires_at, redeemed_at, redeemed_user_id, created_at`,
		in.Email, in.Name, in.ProjectID, instArg, token, in.InvitedBy, in.ExpiresAt,
	).Scan(&inv.ID, &inv.Email, &inv.Name, &inv.ProjectID, &inv.InstitutionID,
		&inv.InviteToken, &inv.InvitedBy, &inv.ExpiresAt, &inv.RedeemedAt,
		&inv.RedeemedUserID, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GetUploaderInviteByToken returns the invite (plus joined project + institution
// names for display) when the supplied token matches.
func GetUploaderInviteByToken(ctx context.Context, db *sql.DB, token string) (*UploaderInvite, error) {
	inv := &UploaderInvite{}
	err := db.QueryRowContext(ctx, `
		SELECT inv.id, inv.email, inv.name, inv.project_id, inv.institution_id,
		       inv.invite_token, inv.invited_by, inv.expires_at,
		       inv.redeemed_at, inv.redeemed_user_id, inv.created_at,
		       p.name, COALESCE(i.name, '')
		FROM uploader_invites inv
		JOIN projects p ON p.id = inv.project_id
		LEFT JOIN institutions i ON i.id = inv.institution_id
		WHERE inv.invite_token = $1`, token,
	).Scan(&inv.ID, &inv.Email, &inv.Name, &inv.ProjectID, &inv.InstitutionID,
		&inv.InviteToken, &inv.InvitedBy, &inv.ExpiresAt,
		&inv.RedeemedAt, &inv.RedeemedUserID, &inv.CreatedAt,
		&inv.ProjectName, &inv.InstitutionName)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// ClaimUploaderInvite atomically marks the invite redeemed, but only if it has
// not been redeemed yet. Returns true when this caller won the claim. Two
// concurrent redeems of the same token therefore race on a single UPDATE and
// exactly one proceeds — the TOCTOU guard for the redeem flow.
func ClaimUploaderInvite(ctx context.Context, db *sql.DB, inviteID string) (bool, error) {
	res, err := db.ExecContext(ctx, `
		UPDATE uploader_invites
		   SET redeemed_at = now()
		 WHERE id = $1 AND redeemed_at IS NULL`, inviteID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// UnclaimUploaderInvite reverses ClaimUploaderInvite after provisioning failed,
// so the recipient can retry the link instead of losing the invitation.
func UnclaimUploaderInvite(ctx context.Context, db *sql.DB, inviteID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE uploader_invites
		   SET redeemed_at = NULL, redeemed_user_id = NULL
		 WHERE id = $1`, inviteID)
	return err
}

// SetUploaderInviteRedeemedUser records which admin_users row the redeemed
// invite produced. Called after provisioning succeeds; redeemed_at was already
// set by ClaimUploaderInvite.
func SetUploaderInviteRedeemedUser(ctx context.Context, db *sql.DB, inviteID, userID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE uploader_invites
		   SET redeemed_user_id = $2
		 WHERE id = $1`, inviteID, userID)
	return err
}

// ListUploaderInvitesByProject returns pending (un-redeemed, un-expired) invites
// for a project. Redeemed and expired rows are excluded; the admin UI only
// surfaces actionable invites.
func ListUploaderInvitesByProject(ctx context.Context, db *sql.DB, projectID string) ([]UploaderInvite, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT inv.id, inv.email, inv.name, inv.project_id, inv.institution_id,
		       '' AS invite_token,
		       inv.invited_by, inv.expires_at,
		       inv.redeemed_at, inv.redeemed_user_id, inv.created_at,
		       p.name, COALESCE(i.name, '')
		FROM uploader_invites inv
		JOIN projects p ON p.id = inv.project_id
		LEFT JOIN institutions i ON i.id = inv.institution_id
		WHERE inv.project_id = $1
		  AND inv.redeemed_at IS NULL
		  AND inv.expires_at > now()
		ORDER BY inv.created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UploaderInvite
	for rows.Next() {
		var inv UploaderInvite
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Name, &inv.ProjectID, &inv.InstitutionID,
			&inv.InviteToken, &inv.InvitedBy, &inv.ExpiresAt,
			&inv.RedeemedAt, &inv.RedeemedUserID, &inv.CreatedAt,
			&inv.ProjectName, &inv.InstitutionName); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// DeleteUploaderInvite removes an unredeemed invite. Used by admins/researchers
// who want to cancel a pending invitation.
func DeleteUploaderInvite(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM uploader_invites WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
