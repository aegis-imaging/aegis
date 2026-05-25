package model

import (
	"context"
	"database/sql"
	"time"
)

// AdminUser represents an authorised admin dashboard user.
type AdminUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"` // admin | viewer | researcher | uploader
	Enabled   bool      `json:"enabled"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// PasswordHash is set only for native-login users (role=uploader). IAP/Azure/
	// AWS users have NULL hash. Never serialised to JSON.
	PasswordHash *string `json:"-"`
}

const adminUserColumns = `id, email, name, role, enabled, notes, created_at, updated_at, password_hash`

func scanAdminUser(row scannable, u *AdminUser) error {
	return row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Enabled, &u.Notes, &u.CreatedAt, &u.UpdatedAt, &u.PasswordHash)
}

// SetAdminUserPasswordHash stores a pre-hashed (e.g. bcrypt) password for the
// given user. Callers must hash before calling — this function only persists.
func SetAdminUserPasswordHash(ctx context.Context, db *sql.DB, userID, hash string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE admin_users SET password_hash = $2, updated_at = now() WHERE id = $1`,
		userID, hash)
	return err
}

func ListAdminUsers(ctx context.Context, db *sql.DB) ([]AdminUser, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+adminUserColumns+` FROM admin_users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := scanAdminUser(rows, &u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func GetAdminUserByID(ctx context.Context, db *sql.DB, id string) (*AdminUser, error) {
	var u AdminUser
	err := scanAdminUser(db.QueryRowContext(ctx,
		`SELECT `+adminUserColumns+` FROM admin_users WHERE id = $1`, id), &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetAdminUserByEmail(ctx context.Context, db *sql.DB, email string) (*AdminUser, error) {
	var u AdminUser
	err := scanAdminUser(db.QueryRowContext(ctx,
		`SELECT `+adminUserColumns+` FROM admin_users WHERE LOWER(email) = LOWER($1)`, email), &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateAdminUser(ctx context.Context, db *sql.DB, u *AdminUser) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO admin_users (email, name, role, enabled, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		u.Email, u.Name, u.Role, u.Enabled, u.Notes,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func UpdateAdminUser(ctx context.Context, db *sql.DB, u *AdminUser) error {
	_, err := db.ExecContext(ctx, `
		UPDATE admin_users SET
			email=$1, name=$2, role=$3, enabled=$4, notes=$5, updated_at=now()
		WHERE id=$6`,
		u.Email, u.Name, u.Role, u.Enabled, u.Notes, u.ID)
	return err
}

func DeleteAdminUser(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM admin_users WHERE id = $1`, id)
	return err
}
