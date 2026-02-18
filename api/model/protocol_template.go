package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

// ProtocolTemplate defines expected DICOM acquisition parameters for a specific
// scanner manufacturer/model/software-version and sequence type within a project.
type ProtocolTemplate struct {
	ID              string          `json:"id"`
	ProjectID       string          `json:"project_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Manufacturer    string          `json:"manufacturer"`       // e.g. "SIEMENS"
	Model           string          `json:"model"`              // e.g. "MAGNETOM Prisma"
	SoftwareVersion string          `json:"software_version"`   // e.g. "VE11C" (empty = any)
	SequenceType    string          `json:"sequence_type"`      // e.g. "T1w_MPRAGE", "FLAIR", "DWI"
	Rules           json.RawMessage `json:"rules"`              // JSON array of ParameterRule objects
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

const protocolTemplateColumns = `
	id, project_id, name, description, manufacturer, model, software_version,
	sequence_type, rules, enabled, created_at, updated_at`

func scanProtocolTemplate(row scannable, t *ProtocolTemplate) error {
	return row.Scan(
		&t.ID, &t.ProjectID, &t.Name, &t.Description,
		&t.Manufacturer, &t.Model, &t.SoftwareVersion,
		&t.SequenceType, &t.Rules, &t.Enabled,
		&t.CreatedAt, &t.UpdatedAt,
	)
}

func CreateProtocolTemplate(ctx context.Context, db *sql.DB, t *ProtocolTemplate) error {
	if t.Rules == nil {
		t.Rules = json.RawMessage("[]")
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO protocol_templates (project_id, name, description, manufacturer, model,
			software_version, sequence_type, rules, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		t.ProjectID, t.Name, t.Description, t.Manufacturer, t.Model,
		t.SoftwareVersion, t.SequenceType, t.Rules, t.Enabled,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func GetProtocolTemplateByID(ctx context.Context, db *sql.DB, id string) (*ProtocolTemplate, error) {
	var t ProtocolTemplate
	err := scanProtocolTemplate(db.QueryRowContext(ctx,
		`SELECT`+protocolTemplateColumns+` FROM protocol_templates WHERE id = $1`, id), &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func ListProtocolTemplatesByProject(ctx context.Context, db *sql.DB, projectID string) ([]ProtocolTemplate, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+protocolTemplateColumns+` FROM protocol_templates WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProtocolTemplate
	for rows.Next() {
		var t ProtocolTemplate
		if err := scanProtocolTemplate(rows, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func UpdateProtocolTemplate(ctx context.Context, db *sql.DB, t *ProtocolTemplate) error {
	if t.Rules == nil {
		t.Rules = json.RawMessage("[]")
	}
	_, err := db.ExecContext(ctx, `
		UPDATE protocol_templates SET
			name=$1, description=$2, manufacturer=$3, model=$4,
			software_version=$5, sequence_type=$6, rules=$7, enabled=$8, updated_at=now()
		WHERE id=$9`,
		t.Name, t.Description, t.Manufacturer, t.Model,
		t.SoftwareVersion, t.SequenceType, t.Rules, t.Enabled, t.ID)
	return err
}

func DeleteProtocolTemplate(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM protocol_templates WHERE id = $1`, id)
	return err
}

// FindMatchingTemplates returns all enabled templates for a project that match
// the given scanner device info. Manufacturer and model comparisons are case-
// insensitive. Empty template fields are treated as wildcards.
func FindMatchingTemplates(ctx context.Context, db *sql.DB, projectID, manufacturer, model, softwareVersion string) ([]ProtocolTemplate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT`+protocolTemplateColumns+`
		FROM protocol_templates
		WHERE project_id = $1
		  AND enabled = true
		  AND (manufacturer = '' OR upper(manufacturer) = upper($2))
		  AND (model = '' OR upper(model) = upper($3))
		  AND (software_version = '' OR upper(software_version) = upper($4))
		ORDER BY name`,
		projectID,
		strings.TrimSpace(manufacturer),
		strings.TrimSpace(model),
		strings.TrimSpace(softwareVersion))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProtocolTemplate
	for rows.Next() {
		var t ProtocolTemplate
		if err := scanProtocolTemplate(rows, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
