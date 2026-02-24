package model

import (
	"context"
	"database/sql"
	"time"
)

// StudyRelationship links two studies with a typed relationship.
type StudyRelationship struct {
	ID             string     `json:"id"`
	StudyID        string     `json:"study_id"`
	RelatedStudyID string     `json:"related_study_id"`
	Relationship   string     `json:"relationship"` // baseline | follow_up | comparison | replicate
	Notes          *string    `json:"notes"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
}

// RelationshipWithStudy embeds a brief related-study summary alongside the link record.
type RelationshipWithStudy struct {
	StudyRelationship
	RelatedStudy RelatedStudySummary `json:"related_study"`
}

// RelatedStudySummary is the minimal study info shown for a related study.
type RelatedStudySummary struct {
	ID                 string  `json:"id"`
	StudyInstanceUID   string  `json:"study_instance_uid"`
	StudyDescription   string  `json:"study_description"`
	Status             string  `json:"status"`
	Modality           string  `json:"modality"`
}

// ValidRelationshipTypes is the set of accepted relationship type values.
var ValidRelationshipTypes = map[string]bool{
	"baseline":   true,
	"follow_up":  true,
	"comparison": true,
	"replicate":  true,
}

// ListStudyRelationships returns all relationships where study_id OR related_study_id matches,
// so inverse links are also visible.
func ListStudyRelationships(ctx context.Context, db *sql.DB, studyID string) ([]RelationshipWithStudy, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			sr.id, sr.study_id, sr.related_study_id, sr.relationship, sr.notes, sr.created_by, sr.created_at,
			s.id, s.study_instance_uid, COALESCE(s.study_description,''), s.status, COALESCE(s.modality,'')
		FROM study_relationships sr
		JOIN studies s ON s.id = sr.related_study_id
		WHERE sr.study_id = $1
		UNION ALL
		SELECT
			sr.id, sr.study_id, sr.related_study_id, sr.relationship, sr.notes, sr.created_by, sr.created_at,
			s.id, s.study_instance_uid, COALESCE(s.study_description,''), s.status, COALESCE(s.modality,'')
		FROM study_relationships sr
		JOIN studies s ON s.id = sr.study_id
		WHERE sr.related_study_id = $1
		ORDER BY created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RelationshipWithStudy
	for rows.Next() {
		var r RelationshipWithStudy
		if err := rows.Scan(
			&r.ID, &r.StudyID, &r.RelatedStudyID, &r.Relationship, &r.Notes, &r.CreatedBy, &r.CreatedAt,
			&r.RelatedStudy.ID, &r.RelatedStudy.StudyInstanceUID, &r.RelatedStudy.StudyDescription,
			&r.RelatedStudy.Status, &r.RelatedStudy.Modality,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []RelationshipWithStudy{}
	}
	return out, rows.Err()
}

// CreateStudyRelationship inserts a new relationship link between two studies.
func CreateStudyRelationship(ctx context.Context, db *sql.DB, studyID, relatedStudyID, relationship, notes, createdBy string) (*StudyRelationship, error) {
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	r := &StudyRelationship{}
	err := db.QueryRowContext(ctx, `
		INSERT INTO study_relationships (study_id, related_study_id, relationship, notes, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, study_id, related_study_id, relationship, notes, created_by, created_at`,
		studyID, relatedStudyID, relationship, notesPtr, createdBy).
		Scan(&r.ID, &r.StudyID, &r.RelatedStudyID, &r.Relationship, &r.Notes, &r.CreatedBy, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// DeleteStudyRelationship removes a relationship by its ID.
func DeleteStudyRelationship(ctx context.Context, db *sql.DB, relID string) (bool, error) {
	res, err := db.ExecContext(ctx, `DELETE FROM study_relationships WHERE id = $1`, relID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
