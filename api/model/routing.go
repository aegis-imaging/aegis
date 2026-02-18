package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Destination represents an external DICOM endpoint studies can be forwarded to.
type Destination struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Description        string    `json:"description"`
	Type               string    `json:"type"` // dicomweb | dimse
	DicomwebURL        string    `json:"dicomweb_url"`
	DicomwebAuthHeader string    `json:"dicomweb_auth_header,omitempty"`
	AETitle            string    `json:"ae_title"`
	Host               string    `json:"host"`
	Port               int       `json:"port"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

const destinationColumns = `
	id, name, slug, description, type, dicomweb_url, dicomweb_auth_header,
	ae_title, host, port, enabled, created_at, updated_at`

func scanDestination(row scannable, d *Destination) error {
	return row.Scan(
		&d.ID, &d.Name, &d.Slug, &d.Description, &d.Type,
		&d.DicomwebURL, &d.DicomwebAuthHeader,
		&d.AETitle, &d.Host, &d.Port, &d.Enabled,
		&d.CreatedAt, &d.UpdatedAt,
	)
}

func CreateDestination(ctx context.Context, db *sql.DB, d *Destination) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO destinations (name, slug, description, type, dicomweb_url, dicomweb_auth_header,
		                          ae_title, host, port, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`,
		d.Name, d.Slug, d.Description, d.Type, d.DicomwebURL, d.DicomwebAuthHeader,
		d.AETitle, d.Host, d.Port, d.Enabled,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func GetDestinationByID(ctx context.Context, db *sql.DB, id string) (*Destination, error) {
	var d Destination
	err := scanDestination(db.QueryRowContext(ctx,
		`SELECT`+destinationColumns+` FROM destinations WHERE id = $1`, id), &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func ListDestinations(ctx context.Context, db *sql.DB) ([]Destination, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+destinationColumns+` FROM destinations ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Destination
	for rows.Next() {
		var d Destination
		if err := scanDestination(rows, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func UpdateDestination(ctx context.Context, db *sql.DB, d *Destination) error {
	_, err := db.ExecContext(ctx, `
		UPDATE destinations SET
			name=$1, slug=$2, description=$3, type=$4,
			dicomweb_url=$5, dicomweb_auth_header=$6,
			ae_title=$7, host=$8, port=$9, enabled=$10, updated_at=now()
		WHERE id=$11`,
		d.Name, d.Slug, d.Description, d.Type,
		d.DicomwebURL, d.DicomwebAuthHeader,
		d.AETitle, d.Host, d.Port, d.Enabled, d.ID)
	return err
}

func DeleteDestination(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM destinations WHERE id = $1`, id)
	return err
}

// RoutingRule maps a set of conditions to an action taken when a study is ingested.
type RoutingRule struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Priority      int       `json:"priority"`
	Enabled       bool      `json:"enabled"`
	ProjectID     *string   `json:"project_id"`     // nil = any project
	Modality      *string   `json:"modality"`        // nil = any modality
	BodyPart      *string   `json:"body_part"`       // nil = any body part
	Source        *string   `json:"source"`          // nil = any source
	Action        string    `json:"action"`          // route_to | require_defacing | auto_approve | require_qa | reject
	DestinationID *string   `json:"destination_id"`  // only when action=route_to
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

const routingRuleColumns = `
	id, name, description, priority, enabled,
	project_id, modality, body_part, source,
	action, destination_id, created_at, updated_at`

func scanRoutingRule(row scannable, r *RoutingRule) error {
	return row.Scan(
		&r.ID, &r.Name, &r.Description, &r.Priority, &r.Enabled,
		&r.ProjectID, &r.Modality, &r.BodyPart, &r.Source,
		&r.Action, &r.DestinationID, &r.CreatedAt, &r.UpdatedAt,
	)
}

func CreateRoutingRule(ctx context.Context, db *sql.DB, r *RoutingRule) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO routing_rules (name, description, priority, enabled,
		                           project_id, modality, body_part, source,
		                           action, destination_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`,
		r.Name, r.Description, r.Priority, r.Enabled,
		r.ProjectID, r.Modality, r.BodyPart, r.Source,
		r.Action, r.DestinationID,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
}

func GetRoutingRuleByID(ctx context.Context, db *sql.DB, id string) (*RoutingRule, error) {
	var r RoutingRule
	err := scanRoutingRule(db.QueryRowContext(ctx,
		`SELECT`+routingRuleColumns+` FROM routing_rules WHERE id = $1`, id), &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func ListRoutingRules(ctx context.Context, db *sql.DB) ([]RoutingRule, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+routingRuleColumns+` FROM routing_rules ORDER BY priority, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RoutingRule
	for rows.Next() {
		var r RoutingRule
		if err := scanRoutingRule(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListEnabledRoutingRules returns rules ordered by priority (ascending) for evaluation.
func ListEnabledRoutingRules(ctx context.Context, db *sql.DB) ([]RoutingRule, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+routingRuleColumns+
			` FROM routing_rules WHERE enabled = TRUE ORDER BY priority ASC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RoutingRule
	for rows.Next() {
		var r RoutingRule
		if err := scanRoutingRule(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func UpdateRoutingRule(ctx context.Context, db *sql.DB, r *RoutingRule) error {
	_, err := db.ExecContext(ctx, `
		UPDATE routing_rules SET
			name=$1, description=$2, priority=$3, enabled=$4,
			project_id=$5, modality=$6, body_part=$7, source=$8,
			action=$9, destination_id=$10, updated_at=now()
		WHERE id=$11`,
		r.Name, r.Description, r.Priority, r.Enabled,
		r.ProjectID, r.Modality, r.BodyPart, r.Source,
		r.Action, r.DestinationID, r.ID)
	return err
}

func DeleteRoutingRule(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM routing_rules WHERE id = $1`, id)
	return err
}

// RoutingLogEntry records a routing rule that fired for a study.
type RoutingLogEntry struct {
	ID            string          `json:"id"`
	StudyID       string          `json:"study_id"`
	RuleID        string          `json:"rule_id"`
	Action        string          `json:"action"`
	DestinationID *string         `json:"destination_id,omitempty"`
	Outcome       string          `json:"outcome"`
	Detail        json.RawMessage `json:"detail,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

func CreateRoutingLogEntry(ctx context.Context, db *sql.DB, e *RoutingLogEntry) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO routing_log (study_id, rule_id, action, destination_id, outcome, detail)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		e.StudyID, e.RuleID, e.Action, e.DestinationID, e.Outcome, e.Detail,
	).Scan(&e.ID, &e.CreatedAt)
}

func ListRoutingLogForStudy(ctx context.Context, db *sql.DB, studyID string) ([]RoutingLogEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, rule_id, action, destination_id, outcome, detail, created_at
		FROM routing_log WHERE study_id = $1 ORDER BY created_at ASC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RoutingLogEntry
	for rows.Next() {
		var e RoutingLogEntry
		if err := rows.Scan(&e.ID, &e.StudyID, &e.RuleID, &e.Action, &e.DestinationID,
			&e.Outcome, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// matchesNullableString returns true if the rule field is nil (wildcard) or equals the value.
func matchesNullableString(ruleField *string, value string) bool {
	if ruleField == nil {
		return true
	}
	return fmt.Sprintf("%s", *ruleField) == value
}
