package handler

import (
	"database/sql"
	"net/http"
	"time"
)

// SpokeSummary is one row in the admin-dashboard Spokes list. It joins
// institutions with the cert thumbprint metadata + study activity stats so
// the UI can render the full status with a single fetch.
type SpokeSummary struct {
	InstitutionID    string     `json:"institution_id"`
	InstitutionName  string     `json:"institution_name"`
	InstitutionSlug  string     `json:"institution_slug"`
	Enabled          bool       `json:"enabled"`
	CertThumbprint   string     `json:"cert_thumbprint"`
	CertSubjectDN    string     `json:"cert_subject_dn"`
	CertEnrolledAt   *time.Time `json:"cert_enrolled_at,omitempty"`
	StudyCount       int        `json:"study_count"`
	LastStudyAt      *time.Time `json:"last_study_at,omitempty"`
	ActiveTokenCount int        `json:"active_token_count"`
}

// ListSpokes GET /api/spokes
// Returns every institution that has a cert thumbprint enrolled (i.e. is
// reachable as a spoke), with summary activity stats. Lists tokens for
// in-progress enrollments too via active_token_count.
func (s *Server) ListSpokes(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `
		WITH activity AS (
		    SELECT institution_id,
		           count(*)         AS study_count,
		           max(created_at)  AS last_study_at
		    FROM studies
		    WHERE source = 'spoke'
		    GROUP BY institution_id
		),
		tokens AS (
		    SELECT institution_id,
		           count(*) FILTER (
		               WHERE used_at IS NULL
		                 AND revoked_at IS NULL
		                 AND expires_at > now()
		           ) AS active_token_count
		    FROM spoke_enrollment_tokens
		    GROUP BY institution_id
		)
		SELECT i.id, i.name, i.slug, i.enabled,
		       coalesce(i.client_cert_thumbprint, ''),
		       coalesce(i.client_cert_subject_dn, ''),
		       i.client_cert_enrolled_at,
		       coalesce(a.study_count, 0),
		       a.last_study_at,
		       coalesce(t.active_token_count, 0)
		FROM institutions i
		LEFT JOIN activity a ON a.institution_id = i.id
		LEFT JOIN tokens   t ON t.institution_id = i.id
		WHERE i.client_cert_thumbprint IS NOT NULL
		   OR t.active_token_count > 0
		ORDER BY i.name ASC`)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var out []SpokeSummary
	for rows.Next() {
		var row SpokeSummary
		var enrolledAt, lastStudyAt sql.NullTime
		if err := rows.Scan(
			&row.InstitutionID, &row.InstitutionName, &row.InstitutionSlug, &row.Enabled,
			&row.CertThumbprint, &row.CertSubjectDN, &enrolledAt,
			&row.StudyCount, &lastStudyAt, &row.ActiveTokenCount,
		); err != nil {
			s.writeError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		if enrolledAt.Valid {
			t := enrolledAt.Time
			row.CertEnrolledAt = &t
		}
		if lastStudyAt.Valid {
			t := lastStudyAt.Time
			row.LastStudyAt = &t
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "rows error")
		return
	}
	if out == nil {
		out = []SpokeSummary{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"spokes": out, "count": len(out)})
}
