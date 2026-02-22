package model

import (
	"context"
	"database/sql"
	"time"
)

// FederationPeer represents a trusted remote AEGIS instance.
// This is a stub for future cross-tenant federation.
type FederationPeer struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Slug       string     `json:"slug"`
	APIURL     string     `json:"api_url"`
	APIKeyHash *string    `json:"-"` // never serialised
	Enabled    bool       `json:"enabled"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

const federationPeerColumns = `id, name, slug, api_url, api_key_hash, enabled, notes, created_at, updated_at`

func scanFederationPeer(row interface{ Scan(...any) error }) (*FederationPeer, error) {
	p := &FederationPeer{}
	return p, row.Scan(&p.ID, &p.Name, &p.Slug, &p.APIURL, &p.APIKeyHash, &p.Enabled, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
}

func ListFederationPeers(ctx context.Context, db *sql.DB) ([]FederationPeer, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+federationPeerColumns+` FROM federation_peers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var peers []FederationPeer
	for rows.Next() {
		p, err := scanFederationPeer(rows)
		if err != nil {
			return nil, err
		}
		peers = append(peers, *p)
	}
	return peers, rows.Err()
}

func GetFederationPeer(ctx context.Context, db *sql.DB, id string) (*FederationPeer, error) {
	row := db.QueryRowContext(ctx, `SELECT `+federationPeerColumns+` FROM federation_peers WHERE id = $1`, id)
	return scanFederationPeer(row)
}

func CreateFederationPeer(ctx context.Context, db *sql.DB, name, slug, apiURL, notes string) (*FederationPeer, error) {
	row := db.QueryRowContext(ctx,
		`INSERT INTO federation_peers (name, slug, api_url, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+federationPeerColumns,
		name, slug, apiURL, notes)
	return scanFederationPeer(row)
}

func UpdateFederationPeer(ctx context.Context, db *sql.DB, id, name, slug, apiURL, notes string, enabled bool) (*FederationPeer, error) {
	row := db.QueryRowContext(ctx,
		`UPDATE federation_peers
		 SET name=$2, slug=$3, api_url=$4, notes=$5, enabled=$6, updated_at=now()
		 WHERE id=$1
		 RETURNING `+federationPeerColumns,
		id, name, slug, apiURL, notes, enabled)
	return scanFederationPeer(row)
}

func DeleteFederationPeer(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM federation_peers WHERE id = $1`, id)
	return err
}
