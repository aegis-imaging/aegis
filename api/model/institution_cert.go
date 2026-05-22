package model

import (
	"context"
	"database/sql"
	"strings"
)

// GetInstitutionByClientCertThumbprint returns the enabled institution with the
// given SHA-256 cert thumbprint, or sql.ErrNoRows.
func GetInstitutionByClientCertThumbprint(ctx context.Context, db *sql.DB, thumbprint string) (*Institution, error) {
	thumb := strings.ToLower(strings.TrimSpace(thumbprint))
	if thumb == "" {
		return nil, sql.ErrNoRows
	}
	var inst Institution
	err := scanInstitution(db.QueryRowContext(ctx,
		`SELECT`+institutionColumns+`
		 FROM institutions
		 WHERE lower(client_cert_thumbprint) = $1
		   AND enabled = TRUE`, thumb), &inst)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// SetInstitutionClientCert records the cert thumbprint + subject DN for an
// institution. Pass empty strings to revoke (clears both columns).
func SetInstitutionClientCert(ctx context.Context, db *sql.DB, institutionID, thumbprint, subjectDN string) error {
	if thumbprint == "" {
		_, err := db.ExecContext(ctx, `
			UPDATE institutions
			SET client_cert_thumbprint = NULL,
			    client_cert_subject_dn = NULL,
			    client_cert_enrolled_at = NULL
			WHERE id = $1`, institutionID)
		return err
	}
	thumb := strings.ToLower(strings.TrimSpace(thumbprint))
	_, err := db.ExecContext(ctx, `
		UPDATE institutions
		SET client_cert_thumbprint = $1,
		    client_cert_subject_dn = $2,
		    client_cert_enrolled_at = now()
		WHERE id = $3`, thumb, subjectDN, institutionID)
	return err
}
