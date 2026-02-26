package model

import (
	"context"
	"database/sql"
	"time"
)

type InstitutionContact struct {
	ID            string    `json:"id"`
	InstitutionID string    `json:"institution_id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
}

func CreateInstitutionContact(ctx context.Context, db *sql.DB, c *InstitutionContact) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO institution_contacts (institution_id, name, email, phone, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		c.InstitutionID, c.Name, c.Email, c.Phone, c.Role).
		Scan(&c.ID, &c.CreatedAt)
}

func ListInstitutionContacts(ctx context.Context, db *sql.DB, institutionID string) ([]InstitutionContact, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, institution_id, name, email, phone, role, created_at
		FROM institution_contacts
		WHERE institution_id = $1
		ORDER BY name ASC`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstitutionContact
	for rows.Next() {
		var c InstitutionContact
		if err := rows.Scan(&c.ID, &c.InstitutionID, &c.Name, &c.Email, &c.Phone, &c.Role, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func DeleteInstitutionContact(ctx context.Context, db *sql.DB, id, institutionID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM institution_contacts WHERE id = $1 AND institution_id = $2`, id, institutionID)
	return err
}
